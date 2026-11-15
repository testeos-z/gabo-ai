// gabo stage-gate plugin for OpenCode (intent 0004 S10, D10; ported to the
// OpenCode 2.2.6 plugin shape by intent 0015 T17; the lock relay of intent 0028
// S3, S10, S12, S15), installed by the gabo-runtime-opencode build as
// .opencode/plugins/gabo-stage-gate.js. A thin connection layer: a plain
// default definition (`id` and `setup`) that loads no module from disk and
// needs no package.json. For each shell call and each edit, write or patch
// call, its `execute.before` hook sends the call to gabo, unchanged, with
// GABO_ENGINE=opencode, and applies gabo's exit:
//   shell call          -> gabo check --pre-command, with the tool-call JSON on stdin;
//   edit, write, patch  -> gabo check --pre-write <path...>;
//   0                   -> the call runs;
//   1 and 3             -> the call is blocked with gabo's output (a hook blocks a
//                          call by throwing; the message reaches the model); exit 3
//                          adds that OpenCode shows no prompt for an ask (probe
//                          item (d) is no);
//   other, or a signal  -> the call is blocked and the message names the failure;
//   gabo not on PATH    -> the call runs with one warning (gabo could not decide).
// Which calls block is gabo's decision; this file holds no rule.

// The OpenCode tools that write files.
const WRITE_TOOLS = new Set(["edit", "write", "patch"]);

// The OpenCode tool that runs a shell command.
const SHELL_TOOL = "shell";

// The engine gabo judges every call for (GABO_ENGINE).
const ENGINE = "opencode";

const NO_PROMPT = "OpenCode shows no prompt for an ask, so the call is blocked (intent 0028 S12)";

// Patch headers that name a file the patch writes; the same headers the Go
// hook mode reads from a Codex apply_patch.
const PATCH_HEADERS = ["*** Add File: ", "*** Update File: ", "*** Move to: "];

// The target paths of one call: `path` (OpenCode v2) or `filePath` (v1)
// for edit and write, the file headers of `patchText` for patch.
function targetPaths(tool, input) {
  if (!input) return [];
  if (tool === "patch") {
    const paths = [];
    for (const line of String(input.patchText ?? "").split("\n")) {
      for (const header of PATCH_HEADERS) {
        if (line.startsWith(header)) {
          const p = line.slice(header.length).trim();
          if (p) paths.push(p);
        }
      }
    }
    return paths;
  }
  const p = input.path ?? input.filePath;
  return typeof p === "string" && p !== "" ? [p] : [];
}

// The gabo hook mode that judges one call, its arguments and its stdin, or
// null when the call is not judged. The shell call goes as the tool-call JSON
// with no agent_id: OpenCode gives the hook a session id for the parent's calls
// too, so gabo judges every OpenCode session with the coder rules.
function gaboCall(tool, input) {
  if (tool === SHELL_TOOL) {
    return { kind: "command", args: ["check", "--pre-command"], stdin: JSON.stringify({ tool_input: { command: input?.command } }) };
  }
  if (!WRITE_TOOLS.has(tool)) return null;
  const paths = targetPaths(tool, input);
  if (paths.length === 0) return null;
  return { kind: "write", args: ["check", "--pre-write", ...paths], stdin: "" };
}

function warn(kind, problem) {
  console.warn(`gabo stage-gate warning: gabo check --pre-${kind} could not run (${problem}); the call is not gated`);
}

// Runs gabo without blocking OpenCode's event loop; resolves with the exit
// status (null when a signal killed it), the signal, its output, or the spawn
// error. The call's stdin is closed after the input, so gabo sees its end.
function runGabo(spawn, args, stdin, cwd) {
  return new Promise((resolve) => {
    const child = spawn("gabo", args, { cwd, env: { ...process.env, GABO_ENGINE: ENGINE } });
    let out = "";
    child.stdout.on("data", (d) => { out += d; });
    child.stderr.on("data", (d) => { out += d; });
    // gabo may exit before it reads its stdin; its exit status still decides.
    child.stdin.on("error", () => {});
    child.on("error", (error) => resolve({ error }));
    child.on("close", (status, signal) => resolve({ status, signal, out }));
    child.stdin.end(stdin);
  });
}

// The error a blocked call throws: each non-empty line on its own line.
function blockError(lines) {
  return new Error(lines.filter((line) => line !== "").join("\n"));
}

// The error for gabo's exit when it is not 0: 1 and 3 block with gabo's
// output; any other exit or a signal blocks and names the failure (S15).
function refusal(kind, result) {
  const out = result.out.trim();
  if (result.status === 1 || result.status === 3) {
    const lines = [out || `gabo check --pre-${kind} blocked the call without a message`];
    if (result.status === 3) lines.push(NO_PROMPT);
    return blockError(lines);
  }
  if (result.status === null) {
    return blockError([`gabo check --pre-${kind} was killed by signal ${result.signal ?? "unknown"}; the call is blocked`, out]);
  }
  return blockError([`gabo check --pre-${kind} failed (exit ${result.status}); the call is blocked`, out]);
}

export default {
  id: "gabo-stage-gate",
  async setup(ctx) {
    // The runtime hands over its built-in child_process module; nothing is loaded from disk.
    const { spawn } = process.getBuiltinModule("node:child_process");
    // `ctx.location.directory` is the project directory OpenCode started in.
    const cwd = ctx.location?.directory ?? process.cwd();
    ctx.tool.hook("execute.before", async (event) => {
      const call = gaboCall(event?.tool, event?.input);
      if (!call) return;
      const result = await runGabo(spawn, call.args, call.stdin, cwd);
      if (result.error) {
        if (result.error.code === "ENOENT") {
          warn(call.kind, "gabo not found on PATH");
          return;
        }
        throw blockError([`gabo check --pre-${call.kind} could not run (${result.error.message ?? result.error}); the call is blocked`]);
      }
      if (result.status === 0) return;
      throw refusal(call.kind, result);
    });
  },
};
