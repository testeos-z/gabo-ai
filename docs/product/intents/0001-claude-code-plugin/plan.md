# Plan 0001 Claude Code plugin

| field | value |
|---|---|
| intent | [0001-claude-code-plugin](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-03 |

The spec was accepted by the owner on 2026-11-03 (spec section 7). `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: shared Go first (hook mode, reviewer-configuration skill), then the plugin tree, then its two hooks, then docs and the owner-machine run (Steps). Reason: the hooks and the plugin skill set consume the shared Go and the new catalog skill. |
| Q2 | **Proposed (auto)**: ten tasks T1 to T10, each one PR of at most about 400 changed lines (Tasks). Reason: the hook mode, its stdin shapes, the skill and its isolation test, the plugin tree and each hook are separately reviewable behaviors. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements and decisions it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`; no cross-intent dependency in this plan, acyclic. Reason: intents 0002 to 0004 depend on this plan (`0001/T1`, `0001/T2`, `0001/T4`, `0001/T5`), never the reverse. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; every code task starts from a failing Go or shell test. Reason: TRD test-first convention. |
| Q6 | **Proposed (auto)**: yes, S1 to S12 each have at least one task (S2 and the manual half of S8 only through T10). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions; spec open questions 3, 4 and 5 stay with the owner and do not block any task. |

Plan items the spec left to `gabo-dev-lead`, answered here:

- **Hook-mode flag name**: **Proposed (auto)** `gabo check --pre-write [path...]`. Reason: the spec's proposed name, says what it judges.
- **Paths outside the project root in the hook mode**: **Proposed (auto)** treated as "not an intent's `plan.md`", exit 0, silent. Reason: a hook must never block writes outside the project (S7 spirit, S11 "not an intent's `plan.md`" case).
- **Third stdin shape (`gabo_facts`, intent 0003 D9)**: **Proposed (auto)** built in intent 0003's plan on top of `0001/T2`. Reason: S11 says its rules and text are specified in intent 0003.
- **Reviewer-configuration skill name**: **Proposed (auto)** `gabo-reviewer-config` in `apps/cli/internal/skills/`. Reason: one catalog source for all four engines (open question 6); exact name, not a prefix, is excluded by S3.
- **User-level reviewer scripts placement in that skill**: **Proposed (auto)** the skill tells the user to run `gabo init`, which already installs `skills.ReviewerFiles()` at user level (D20). Reason: no second installer.
- **Isolation pass marker**: **Proposed (auto)** one file per engine, `${XDG_CONFIG_HOME:-$HOME/.config}/gabo/reviewer-isolation/<engine>`, holding the engine's `--version` line. Reason: user level like the reviewer scripts, out of any PR's reach; the wave-side check is built in intents 0002 and 0004.
- **Shared parts for other intents**: the Go hook mode is **T1** (path arguments, rule) and **T2** (stdin shapes); the reviewer-configuration skill is **T4** and its isolation test **T5**. Other intents depend on them as `0001/T1`, `0001/T2`, `0001/T4`, `0001/T5`.

## Steps

1. Build the shared Go hook mode: `gabo check --pre-write` with path arguments on the existing stage-gate code (T1), then the Claude Code and Codex stdin shapes (T2). Delivers S11.
2. Name the flag in TRD 9.4 and PRD R31 (T3), the plan item of spec section 5. Supports S11.
3. Create the reviewer-configuration catalog skill once (T4), then its codex and opencode isolation test and pass marker (T5). Delivers S10 for all four engines.
4. Create the marketplace and the plugin tree with skill symlinks and no agents, commands, MCP or `bin/` (T6). Delivers S1, S3, S4, S5.
5. Add the post-write hook (T7), then the pre-write hook on the hook mode (T8). Delivers S6, S7, S8, S12.
6. Document install, the `checks` option, the release checklist and the coexistence with `gabo init` (T9). Delivers S9, the S5 release step.
7. Owner-machine verification run (T10), spec section 2 Verification steps 1 to 5. Delivers S2 and the manual checks of S3, S6, S8, S12.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Go hook mode: `gabo check --pre-write <path...>` judges pending writes with the stage gate (shared by intents 0001 to 0004) | S11, D8 | — | T2, T3 | Test first: failing Go tests in `apps/cli/internal/slices/check` and `apps/cli/cmd/gabo` for every S11 path-argument case (unaccepted spec → exit 1 with one `✗ <path>` line naming `spec.md` not accepted; filled Acceptance row → exit 0 silent; no `spec.md` → exit 1 naming it; unreadable spec → exit 1, fail closed; non-plan path or path outside the root → exit 0 silent; relative and absolute-inside-root paths equal); the rule reuses the existing `specAccepted` stage-gate code with no copy; no file is written; a golden test proves `gabo check` without the flag keeps its output and exit codes on the existing fixtures. | M |
| T2 | Go hook mode stdin: Claude Code `PreToolUse` and Codex `apply_patch` tool-call JSON (shared by intents 0001 to 0004) | S11 | T1 | T8 | Test first: failing Go tests for `--pre-write` with no path argument: a Claude Code JSON whose `tool_input.file_path` is the `plan.md` equals the path-argument result; a Codex `apply_patch` input with `*** Add File:` or `*** Update File:` headers in `tool_input.command` yields one result per target and exits 1 on the unaccepted `plan.md`; malformed JSON exits 2 with one usage line; empty stdin exits 2. | M |
| T3 | Name the hook-mode flag in TRD 9.4 and PRD R31 | S11, D8 | T1 | — | Docs-only PR touching `docs/product/TRD.md` and `docs/product/PRD.md`: TRD 9.4 and R31 name `gabo check --pre-write`, its exit codes and that the no-flag output is unchanged, citing intent 0001; `gabo check` passes. | S |
| T4 | Reviewer-configuration catalog skill `gabo-reviewer-config` (single source for all four engines) | S10, D4 | — | T5, T6 | Test first: failing Go test in `apps/cli/internal/skills` that the skill exists in `skills.Names()`, has valid frontmatter, names the four engines (`claude`, `codex`, `pi`, `opencode`) and `runtimes.reviewer`, and contains no `reviewer.sh` invocation; the body tells the agent to check the picked engine's CLI on `PATH` and, when missing, name the engine and the CLI and install or write nothing; it places the reviewer scripts at user level only through `gabo init`; `claude` and `pi` need no isolation test; existing skill-list tests updated. | M |
| T5 | Isolation test and pass marker for `codex` and `opencode` reviewers in `gabo-reviewer-config` (shared by intents 0002 and 0004) | S10 | T4 | — | Test first: failing shell tests (same style as `gabo-reviewer/scripts/tests/`) with stub `codex` and `opencode` on `PATH`: the script builds a scratch PR clone (`mktemp -d`) with planted `.opencode/` plugin, or `AGENTS.md` and `.codex/` config; it runs `codex` with `--ignore-user-config --ignore-rules -s read-only --ephemeral` and its own `CODEX_HOME`, and `opencode run --standalone` with its own `HOME` and `XDG_*`; on pass it writes the marker `${XDG_CONFIG_HOME:-$HOME/.config}/gabo/reviewer-isolation/<engine>` with the engine version; on fail it names the engine and the failed check, writes no marker and leaves `runtimes.reviewer` unchanged; never runs for `claude` or `pi`; the skill body calls the script once at configuration time. | M |
| T6 | Marketplace `gabo-ai` and plugin `gabo` with catalog skill symlinks | S1, S3, S4, S5, D1, D2, D3, D6, D7 | T4 | T7 | Test first: failing Go test that reads `.claude-plugin/marketplace.json` (name `gabo-ai`, exactly one entry `gabo`, source `./plugins/claude-code`, no `version`), lists `plugins/claude-code/skills/` and asserts it equals `skills.Names()` minus `gabo-reviewer` and `gabo-reviewer-placement-local`, each a relative symlink resolving to `apps/cli/internal/skills/<name>`, `gabo-reviewer-config` present; no `agents/`, `commands/`, `bin/` or `.mcp.json` and `plugin.json` declares none of them; `plugin.json` `version` equals the CLI release version; `claude plugin validate .` and `claude plugin validate plugins/claude-code` pass where Claude Code is installed. | M |
| T7 | Post-write hook `scripts/gabo-check.sh`, `hooks.json` and the `checks` option | S6, S7, S8, D8, D9, D10 | T6 | T8 | Test first: failing shell tests with a fake `gabo` on `PATH`: write to an intent `plan.md` with unaccepted spec → exit 2 with gabo's `✗` line on stderr; passing project → exit 0 silent; non-gabo path, no `.gabo/VERSION`, no `gabo` on `PATH`, `CLAUDE_PLUGIN_OPTION_CHECKS=false` → exit 0 silent each; `gabo` exit 2 → exit 0 with one warning line; `plugin.json` declares `userConfig.checks` boolean default `true`; `hooks.json` wires `PostToolUse` `Write\|Edit`; the script only reads and runs `gabo check` and writes nothing. | M |
| T8 | Pre-write hook `scripts/gabo-pre-write.sh` on the Go hook mode | S7, S12, D8 | T2, T7 | T9 | Test first: failing shell tests with a fake `gabo` that records its argv and stdin: the script pipes the tool-call JSON unchanged to `gabo check --pre-write` in `$CLAUDE_PROJECT_DIR`; gabo exit 1 → hook exit 2 with gabo's lines on stderr; exit 0 → exit 0 silent; exit 2 → exit 0 with one warning; the four S7 cases → exit 0 silent without calling `gabo`; `hooks.json` adds `PreToolUse` `Write\|Edit`; a grep test proves the script holds no acceptance or `plan.md` rule. | S |
| T9 | README install section and plugin release checklist | S9, S5, D1, D5, D9 | T8 | T10 | Docs PR: the README install section names `claude plugin marketplace add testeos-z/gabo-ai`, `claude plugin install gabo@gabo-ai`, the `checks` option, the private-repository credential step (`gh auth setup-git` or SSH key) and that `gabo init` stays required for per-project config, slots, adapters and agents; `plugins/claude-code/README.md` exists; the release checklist adds the `plugin.json` version bump and `claude plugin tag plugins/claude-code` matching `apps/cli/v<x.y.z>`. | S |
| T10 | Owner-machine verification of the installed plugin | S2, S3, S6, S8, S12 | T9 | — | Owner runs spec section 2 Verification steps 1 to 5 on local Claude Code and records the results on the task: both validates pass; `claude plugin list` shows `gabo@gabo-ai` enabled and `claude plugin details gabo` lists the S3 skills; cache `SKILL.md` files are byte-identical to the catalog (`cmp`); the pre-write block and post-write failure appear in a scratch gabo project and nothing outside it or with `checks` off; `gabo check` passes and `git status` is clean. | S |

## Tracker

Intent tracker item: [#172](https://github.com/testeos-z/gabo-ai/issues/172) under product card #171.

| task | issue |
|---|---|
| T1 | [#177](https://github.com/testeos-z/gabo-ai/issues/177) |
| T2 | [#178](https://github.com/testeos-z/gabo-ai/issues/178) |
| T3 | [#179](https://github.com/testeos-z/gabo-ai/issues/179) |
| T4 | [#180](https://github.com/testeos-z/gabo-ai/issues/180) |
| T5 | [#181](https://github.com/testeos-z/gabo-ai/issues/181) |
| T6 | [#182](https://github.com/testeos-z/gabo-ai/issues/182) |
| T7 | [#183](https://github.com/testeos-z/gabo-ai/issues/183) |
| T8 | [#184](https://github.com/testeos-z/gabo-ai/issues/184) |
| T9 | [#185](https://github.com/testeos-z/gabo-ai/issues/185) |
| T10 | [#186](https://github.com/testeos-z/gabo-ai/issues/186) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| Claude Code does not dereference the in-marketplace skill symlinks into the cache (D3). | T6, T10 | T10 step 3 proves it; the fallback (D3 option b, copied tree plus a byte-equality test) is one task-scope change to T6. |
| The isolation-test invocation of `codex` and `opencode` (T5) drifts from the reviewer-script engines that intents 0002 S12 and 0004 S15 build. | T5, 0002, 0004 | Those plans depend on `0001/T5` and reuse its flags; a mismatch is a plan change in the later intent. |
| Adding `gabo-reviewer-config` to the catalog makes `gabo check` report a new skill in existing projects until `gabo init` reruns. | T4 | Expected behavior (TRD 9.4); T4 updates the skill-list tests and this repository's copy is refreshed by `gabo init`. |
| Claude Code is not installed in the coder or reviewer environment, so `claude plugin validate` cannot run in CI. | T6, T10 | Go tests pin the manifest shape; the validate run is part of T10 on the owner's machine. |
| A hook-mode exit code change breaks the no-flag `gabo check` contract (R31). | T1, T2 | Golden test on the existing fixtures in T1. |

## Open questions

1. Spec open question 3 (marketplace host while the repository is private), 4 (install ref and who bumps the version) and 5 (duplicate plugin and project skills) stay with the owner; T9 documents the recommended options and changes when the owner decides (owner).
2. Spec open question 6 remainder: confirm `gabo-reviewer-placement-local` stays out of the plugin (S3); T6 excludes it as the accepted S3 says (owner).
3. T3 edits TRD 9.4 and PRD R31 wording as the accepted spec's section 5 asks; if the owner wants it as a separate plan-change proposal (global scope) instead of a docs task, T3 is replaced by that proposal (owner).
