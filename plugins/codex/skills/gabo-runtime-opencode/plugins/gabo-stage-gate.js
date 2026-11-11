// gabo stage-gate plugin for OpenCode (intent 0004 S10, D10; ported to the
// OpenCode 2.2.6 plugin shape by intent 0015 T17), installed by the
// gabo-runtime-opencode build as .opencode/plugins/gabo-stage-gate.js.
// A thin connection layer: a plain default definition (`id` and `setup`)
// that loads no module from disk and needs no package.json. For each edit, write or
// patch call its `execute.before` hook passes the call's target paths to
// `gabo check --pre-write <path...>`, the shared Go hook mode, and applies
// its exit code:
//   0      -> the call runs;
//   1      -> the call fails before the write, with gabo's output (a hook
//             blocks a call by throwing; the message reaches the model);
//   other  -> the call runs and one warning is logged, as when gabo is not
//             on PATH (gabo could not decide).
// Which paths block is gabo's decision; this file holds no rule.
// A shell call that writes files is not gated (documented limit).

// The OpenCode tools that write files.
const WRITE_TOOLS = new Set(["edit", "write", "patch"]);

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

function warn(problem) {
  console.warn(`gabo stage-gate warning: gabo check --pre-write could not run (${problem}); the write is not gated`);
}

export default {
  id: "gabo-stage-gate",
  async setup(ctx) {
    // The runtime hands over its built-in child_process module; nothing is loaded from disk.
    const { spawnSync } = process.getBuiltinModule("node:child_process");
    ctx.tool.hook("execute.before", async (event) => {
      if (!WRITE_TOOLS.has(event?.tool)) return;
      const paths = targetPaths(event.tool, event.input);
      if (paths.length === 0) return;
      const result = spawnSync("gabo", ["check", "--pre-write", ...paths], {
        cwd: ctx.location?.directory ?? process.cwd(),
        encoding: "utf8",
      });
      if (result.error) {
        warn(result.error.code === "ENOENT" ? "gabo not found on PATH" : String(result.error.message ?? result.error));
        return;
      }
      if (result.status === 1) {
        const text = `${result.stdout ?? ""}${result.stderr ?? ""}`.trim();
        throw new Error(text || "gabo check --pre-write blocked this write");
      }
      if (result.status !== 0) warn(`exit ${result.status}`);
    });
  },
};
