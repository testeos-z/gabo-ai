// gabo extension for Pi (intent 0003, intent 0028). A thin connection layer
// (TRD 2, D9): it collects the session facts or one tool call, pipes it to
// gabo unchanged and applies what gabo returns. Every rule, comparison and
// message text lives in Go; this file decides nothing about tools, paths or
// commands. It writes no file (S8), and it reads no list of gentle-pi's: Pi's
// own handlers keep their guards, and a call is blocked when either blocks it.
//
//   session_start (T6): in a gabo project (gabo.yaml in the working
//   directory) run gabo once with the session facts and show each returned
//   notification unchanged; gabo missing or failing -> one warning.
//   tool_call write and edit (T7): run `gabo check --pre-write` once with the
//   session facts plus the call's input.path, unchanged, as the one target
//   path; gabo's block becomes { block: true, reason } with gabo's reason.
//   tool_call bash (T12, intent 0028 S3): run `gabo check --pre-command` once
//   with the tool-call JSON, unchanged; gabo's text becomes the block reason.
//
// Every run carries GABO_ENGINE=pi (S16). The exit mapping of a judged call
// is the one of the Claude Code relay (intent 0028 S10, S12, S15): 0 runs the
// call; 1 blocks it with gabo's text; 3 is an ask Pi cannot show (probe item
// (d) is no), so it blocks as 1 and says so; any other exit, a signal or a
// failed spawn blocks the call and names the failure. gabo not on PATH runs
// the call with one warning.
import { spawn } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext, ToolCallEvent, ToolCallEventResult } from "@earendil-works/pi-coding-agent";

export type GaboEvent = "session_start" | "tool_call";

// The two hook modes this relay calls: the file tool (session facts on stdin)
// and the shell tool (the tool-call JSON on stdin).
export type GaboMode = "pre-write" | "pre-command";

export interface GaboFacts {
  gabo_facts: 1;
  event: GaboEvent;
  engine: "pi";
  tools: string[];
  package_version: string;
  target_paths?: string[];
}

export interface GaboNotification {
  level: string;
  text: string;
}

export interface GaboResult {
  notify: GaboNotification[];
  block: { reason: string } | null;
}

// What one gabo run came to: absent (not on PATH), failed (an exit other than
// 0, 1 or 3, a signal, the timeout or a spawn error, each named in problem),
// or ran, with its exit code and stdout.
export type GaboRun =
  | { kind: "absent"; problem: string }
  | { kind: "failed"; problem: string }
  | { kind: "ran"; code: number; out: string };

const GABO_TIMEOUT_MS = 30_000;
const GABO_ENGINE = "pi";
const NO_PROMPT = "gabo extension: Pi shows no prompt for an ask, so the call is blocked (intent 0028 S12)";

let packageVersionCache: string | undefined;

// The version of this package, read from its own package.json.
export function packageVersion(): string {
  packageVersionCache ??= String(JSON.parse(readFileSync(new URL("../package.json", import.meta.url), "utf8")).version);
  return packageVersionCache;
}

// A gabo project has gabo.yaml in the working directory (S4).
export function isGaboProject(cwd: string): boolean {
  return existsSync(join(cwd, "gabo.yaml"));
}

// The session facts every invocation sends (D9): the names of every
// registered tool and the package version, unfiltered.
export function sessionFacts(pi: ExtensionAPI, event: GaboEvent): GaboFacts {
  return {
    gabo_facts: 1,
    event,
    engine: "pi",
    tools: pi.getAllTools().map((t) => t.name),
    package_version: packageVersion(),
  };
}

// Runs `gabo check --<mode>` in cwd with stdin, GABO_ENGINE=pi and the rest of
// the environment (GABO_TASK and GABO_OUTCOME_FILE pass through, S16). Exit 0,
// 1 or 3 is a run with its stdout; any other outcome is a failed run.
export function runGabo(cwd: string, mode: GaboMode, stdin: string): Promise<GaboRun> {
  return new Promise((resolve) => {
    let stdout = "";
    let settled = false;
    const done = (run: GaboRun) => {
      if (!settled) {
        settled = true;
        resolve(run);
      }
    };
    let child;
    try {
      child = spawn("gabo", ["check", `--${mode}`], {
        cwd,
        env: { ...process.env, GABO_ENGINE },
        stdio: ["pipe", "pipe", "ignore"],
        timeout: GABO_TIMEOUT_MS,
      });
    } catch (err) {
      done(spawnProblem(err));
      return;
    }
    child.on("error", (err) => done(spawnProblem(err)));
    child.stdout?.setEncoding("utf8");
    child.stdout?.on("data", (chunk: string) => {
      stdout += chunk;
    });
    child.stdin?.on("error", () => undefined);
    child.on("close", (code, signal) => {
      if (code === 0 || code === 1 || code === 3) {
        done({ kind: "ran", code, out: stdout });
      } else {
        done({ kind: "failed", problem: code === null ? `killed by ${signal}` : `exit ${code}` });
      }
    });
    child.stdin?.end(stdin);
  });
}

function spawnProblem(err: unknown): GaboRun {
  const code = (err as { code?: string })?.code;
  return code === "ENOENT"
    ? { kind: "absent", problem: "gabo not found on PATH" }
    : { kind: "failed", problem: String((err as Error)?.message ?? err) };
}

function parseResult(stdout: string): GaboResult | undefined {
  try {
    const parsed = JSON.parse(stdout);
    if (!parsed || !Array.isArray(parsed.notify)) return undefined;
    return { notify: parsed.notify, block: parsed.block ?? null };
  } catch {
    return undefined;
  }
}

// The one warning shown when gabo could not decide a call that runs anyway
// (intent 0001 S7).
export function warnGaboUnavailable(ctx: ExtensionContext, mode: GaboMode, problem: string): void {
  ctx.ui.notify(`gabo extension warning: gabo check --${mode} could not run (${problem}); gabo's checks skipped`, "warning");
}

// Shows each notification gabo returned, text and level unchanged.
export function showNotifications(ctx: ExtensionContext, result: GaboResult): void {
  for (const n of result.notify) ctx.ui.notify(n.text, n.level as "info" | "warning" | "error");
}

// The reason of a blocked call when gabo exits 1 or 3 (S3, S12): gabo's text
// unchanged, or one generic line when it gave none; exit 3 adds the no-prompt
// line.
function blockReason(mode: GaboMode, code: number, text: string): string {
  const lines = [text || `gabo extension: gabo check --${mode} blocked the call without a message`];
  if (code === 3) lines.push(NO_PROMPT);
  return lines.join("\n");
}

// The reason of a blocked call when gabo fails (S15): the failure, named.
function failureReason(mode: GaboMode, problem: string): string {
  return `gabo extension: gabo check --${mode} failed (${problem}); the call is blocked`;
}

// Pi's file-writing tools whose target gabo judges (S5, S6).
const WRITE_TOOLS = new Set(["write", "edit"]);

// Pi's shell tool, whose command gabo judges (intent 0028 S3).
const SHELL_TOOL = "bash";

// The verdict on one write or edit: exit 0 runs it, a block is gabo's reason,
// a failure blocks it and names the failure, a missing gabo runs it with one
// warning.
function judgeWrite(ctx: ExtensionContext, run: GaboRun): ToolCallEventResult | undefined {
  if (run.kind === "absent") {
    warnGaboUnavailable(ctx, "pre-write", run.problem);
    return undefined;
  }
  if (run.kind === "failed") return { block: true, reason: failureReason("pre-write", run.problem) };
  const result = parseResult(run.out);
  if (run.code === 0) {
    if (result) showNotifications(ctx, result);
    else warnGaboUnavailable(ctx, "pre-write", "unreadable output");
    return undefined;
  }
  if (result) showNotifications(ctx, result);
  const reason = typeof result?.block?.reason === "string" ? result.block.reason : "";
  return { block: true, reason: blockReason("pre-write", run.code, reason) };
}

// The verdict on one shell call: exit 0 runs it, gabo's text blocks it, a
// failure blocks it and names the failure, a missing gabo runs it with one
// warning.
function judgeShell(ctx: ExtensionContext, run: GaboRun): ToolCallEventResult | undefined {
  if (run.kind === "absent") {
    warnGaboUnavailable(ctx, "pre-command", run.problem);
    return undefined;
  }
  if (run.kind === "failed") return { block: true, reason: failureReason("pre-command", run.problem) };
  if (run.code === 0) return undefined;
  return { block: true, reason: blockReason("pre-command", run.code, run.out.replace(/\n+$/, "")) };
}

// Relays one write, edit or bash call to gabo and applies gabo's answer. Which
// calls block is gabo's decision, not this one; a call this relay allows
// returns no result, so Pi's other handlers, gentle-pi's among them, still
// decide it.
export async function onToolCall(
  pi: ExtensionAPI,
  event: ToolCallEvent,
  ctx: ExtensionContext,
): Promise<ToolCallEventResult | undefined> {
  if (!isGaboProject(ctx.cwd)) return undefined;
  if (event.toolName === SHELL_TOOL) {
    const call = JSON.stringify({ tool_name: event.toolName, tool_input: event.input });
    return judgeShell(ctx, await runGabo(ctx.cwd, "pre-command", call));
  }
  if (!WRITE_TOOLS.has(event.toolName)) return undefined;
  const facts = { ...sessionFacts(pi, "tool_call"), target_paths: [event.input.path as string] };
  return judgeWrite(ctx, await runGabo(ctx.cwd, "pre-write", JSON.stringify(facts)));
}

export default function gabo(pi: ExtensionAPI): void {
  pi.on("session_start", async (_event, ctx) => {
    if (!isGaboProject(ctx.cwd)) return;
    const run = await runGabo(ctx.cwd, "pre-write", JSON.stringify(sessionFacts(pi, "session_start")));
    const result = run.kind === "ran" ? parseResult(run.out) : undefined;
    if (result) showNotifications(ctx, result);
    else warnGaboUnavailable(ctx, "pre-write", run.kind === "ran" ? "unreadable output" : run.problem);
  });
  pi.on("tool_call", (event, ctx) => onToolCall(pi, event, ctx));
}
