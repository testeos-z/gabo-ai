// gabo stage-gate plugin for OpenCode (intent 0004 S10, D10), installed by
// the gabo-runtime-opencode build as .opencode/plugins/gabo-stage-gate.js.
// A thin connection layer: plain JavaScript that loads no module and needs
// no package.json. For each edit, write or patch call it passes the call's
// target paths to `gabo check --pre-write <path...>`, the shared Go hook
// mode, and applies its exit code:
//   0      -> the call runs;
//   1      -> the call fails before the write, with gabo's output;
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
function targetPaths(tool, args) {
  if (!args) return [];
  if (tool === "patch") {
    const paths = [];
    for (const line of String(args.patchText ?? "").split("\n")) {
      for (const header of PATCH_HEADERS) {
        if (line.startsWith(header)) {
          const p = line.slice(header.length).trim();
          if (p) paths.push(p);
        }
      }
    }
    return paths;
  }
  const p = args.path ?? args.filePath;
  return typeof p === "string" && p !== "" ? [p] : [];
}

function warn(problem) {
  console.warn(`gabo stage-gate warning: gabo check --pre-write could not run (${problem}); the write is not gated`);
}

export const GaboStageGate = async ({ $ }) => ({
  "tool.execute.before": async (input, output) => {
    if (!WRITE_TOOLS.has(input?.tool)) return;
    const paths = targetPaths(input.tool, output?.args);
    if (paths.length === 0) return;
    let result;
    try {
      // `which` is a Bun shell builtin; a missing gabo would otherwise look
      // like exit 1 ("command not found").
      const found = await $`which gabo`.nothrow().quiet();
      if (found.exitCode !== 0) {
        warn("gabo not found on PATH");
        return;
      }
      result = await $`gabo check --pre-write ${paths}`.nothrow().quiet();
    } catch (err) {
      warn(String(err?.message ?? err));
      return;
    }
    if (result.exitCode === 1) {
      const text = `${result.stdout ?? ""}${result.stderr ?? ""}`.trim();
      throw new Error(text || "gabo check --pre-write blocked this write");
    }
    if (result.exitCode !== 0) warn(`exit ${result.exitCode}`);
  },
});
