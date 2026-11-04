// gabo extension for Pi (intent 0003). A thin connection layer (TRD 2, D9):
// it collects the session facts, pipes them to `gabo check --pre-write` and
// shows or applies what gabo returns. Every rule, comparison and message
// text lives in Go; this file decides nothing about tools, paths or
// versions. It writes no file (S8).
//
//   session_start (T6): in a gabo project (gabo.yaml in the working
//   directory) run gabo once with the session facts and show each returned
//   notification unchanged; gabo missing or failing -> one warning.
//   tool_call (T7): for Pi's write and edit tools in a gabo project, run
//   gabo once with the session facts plus the call's input.path, unchanged,
//   as the one target path; gabo's block becomes { block: true, reason }
//   with gabo's reason; gabo missing or failing -> no block, one warning.
import { spawn } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { join } from "node:path";
import type { ExtensionAPI, ExtensionContext, ToolCallEvent, ToolCallEventResult } from "@earendil-works/pi-coding-agent";

export type GaboEvent = "session_start" | "tool_call";

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

export type GaboRun = { ok: true; code: number; result: GaboResult } | { ok: false; problem: string };

const GABO_TIMEOUT_MS = 30_000;

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

// Runs `gabo check --pre-write` in cwd with the facts on stdin. Exit 0 or 1
// with gabo's JSON is a result; anything else (not on PATH, exit 2, a
// signal, unreadable output) is a problem the caller reports as a warning.
export function runGabo(cwd: string, facts: GaboFacts): Promise<GaboRun> {
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
      child = spawn("gabo", ["check", "--pre-write"], {
        cwd,
        stdio: ["pipe", "pipe", "ignore"],
        timeout: GABO_TIMEOUT_MS,
      });
    } catch (err) {
      done({ ok: false, problem: describeSpawnError(err) });
      return;
    }
    child.on("error", (err) => done({ ok: false, problem: describeSpawnError(err) }));
    child.stdout?.setEncoding("utf8");
    child.stdout?.on("data", (chunk: string) => {
      stdout += chunk;
    });
    child.stdin?.on("error", () => undefined);
    child.on("close", (code, signal) => {
      if (code !== 0 && code !== 1) {
        done({ ok: false, problem: code === null ? `killed by ${signal}` : `exit ${code}` });
        return;
      }
      const result = parseResult(stdout);
      done(result ? { ok: true, code, result } : { ok: false, problem: "unreadable output" });
    });
    child.stdin?.end(JSON.stringify(facts));
  });
}

function describeSpawnError(err: unknown): string {
  const code = (err as { code?: string })?.code;
  return code === "ENOENT" ? "gabo not found on PATH" : String((err as Error)?.message ?? err);
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

// The one warning shown when gabo could not decide (intent 0001 S7).
export function warnGaboUnavailable(ctx: ExtensionContext, problem: string): void {
  ctx.ui.notify(`gabo extension warning: gabo check --pre-write could not run (${problem}); gabo's checks skipped`, "warning");
}

// Shows each notification gabo returned, text and level unchanged.
export function showNotifications(ctx: ExtensionContext, result: GaboResult): void {
  for (const n of result.notify) ctx.ui.notify(n.text, n.level as "info" | "warning" | "error");
}

// Pi's file-writing tools whose target gabo judges (S5, S6).
const WRITE_TOOLS = new Set(["write", "edit"]);

// Relays one write or edit call to gabo and applies gabo's answer: exit 1
// with a block reason blocks the call with that reason unchanged; any other
// outcome lets it run. Which paths block is gabo's decision, not this one.
export async function onToolCall(
  pi: ExtensionAPI,
  event: ToolCallEvent,
  ctx: ExtensionContext,
): Promise<ToolCallEventResult | undefined> {
  if (!WRITE_TOOLS.has(event.toolName) || !isGaboProject(ctx.cwd)) return undefined;
  const facts = { ...sessionFacts(pi, "tool_call"), target_paths: [event.input.path as string] };
  const run = await runGabo(ctx.cwd, facts);
  if (!run.ok) {
    warnGaboUnavailable(ctx, run.problem);
    return undefined;
  }
  showNotifications(ctx, run.result);
  if (run.code !== 1) return undefined;
  const reason = run.result.block?.reason;
  if (typeof reason !== "string") {
    warnGaboUnavailable(ctx, "exit 1 without a block reason");
    return undefined;
  }
  return { block: true, reason };
}

export default function gabo(pi: ExtensionAPI): void {
  pi.on("session_start", async (_event, ctx) => {
    if (!isGaboProject(ctx.cwd)) return;
    const run = await runGabo(ctx.cwd, sessionFacts(pi, "session_start"));
    if (run.ok) showNotifications(ctx, run.result);
    else warnGaboUnavailable(ctx, run.problem);
  });
  pi.on("tool_call", (event, ctx) => onToolCall(pi, event, ctx));
}
