# Plan 0004 OpenCode runtime

| field | value |
|---|---|
| intent | [0004-opencode-runtime](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-03 |

The spec was accepted by the owner on 2026-11-03 (spec section 7). `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: the `gabo-runtime-opencode` skill in three slices on the same file, then the `gabo check` OpenCode rules on the shared runtime enum of `0002/T1`, then the stage-gate plugin on the intent 0001 hook mode, then the reviewer engine on the intent 0001 marker and the intent 0002 preflight, then docs and rule alignment, then the owner-machine run (Steps). Reason: the skill slices share one file, and the check, plugin and reviewer engine consume the shared parts of intents 0001 and 0002. |
| Q2 | **Proposed (auto)**: nine live tasks (T2 to T6, T8 to T11; T1 and T7 removed 2026-11-03 as duplicates of `0002/T1` and `0002/T3`), each one PR of at most about 400 changed lines (Tasks). Reason: the skill's agent build, its links, its project config write, the `gabo check` rules, the plugin and the reviewer engine are separately reviewable behaviors. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements and decisions it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic. Cross-intent: T3 depends on `0001/T4` (the `gabo-reviewer-config` skill it links), T6 on `0001/T1` (the hook mode the plugin calls with path arguments), T8 on `0001/T5` (the isolation pass marker); T5 and T8 on `0002/T1` (the four-engine runtime enum), T8 on `0002/T2` (the reviewer scripts' pass-marker preflight for `codex` and `opencode`), T9 on `0002/T5` (TRD 2 and D27 wording), T11 on `0002/T3` (the wave-start refusal). Reason: reuse the shared parts planned once in intents 0001 and 0002, never duplicate them. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; every code task starts from a failing Go or shell test. Reason: TRD test-first convention. |
| Q6 | **Proposed (auto)**: yes, S1 to S15 each have at least one live task (S1 through T5's check of an all-`opencode` config on the enum of `0002/T1`; S13 and the manual halves of S5, S6, S7, S9, S14 and S15 through T11). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions; spec open questions 1, 2, 3, 4, 8, 9 and 10 stay with the owner; the plan follows each recommendation and none blocks a task except question 3 for unattended sibling-worktree waves (T11 records it). |

Plan items the spec left to `gabo-dev-lead`, answered here:

- **Runtime enum edit shared with intent 0002 S6**: **Proposed (auto)** made once by `0002/T1` (#204), which accepts `claude`, `codex`, `pi` and `opencode` in all three roles; this plan depends on it and its T1 is removed. Reason: S1 and 0002 S6 ask for one edit, and the intent 0002 plan owns it.
- **Pass-marker enforcement (S15, shared with intent 0002 S12)**: **Proposed (auto)** the intent 0002 mechanism for both engines: `reviewer.sh preflight` refuses `codex` or `opencode` without the `0001/T5` marker for the installed version (`0002/T2`), and `gabo-cycle` runs that preflight before claiming any task, auto mode included (`0002/T3`); this plan's T7 is removed. Reason: one mechanism for both engines, already planned in intent 0002.
- **Plugin test runner (S10 open choice)**: **Proposed (auto)** a Go test that runs the plugin with `node` (no import, plain JS) against a stubbed `$` and a fake `gabo` recording its argv, skipped with a named reason when `node` is not on `PATH`. Reason: Bun test is not in the stack; the gate stays under `go test`.
- **Plugin source location**: **Proposed (auto)** `apps/cli/internal/skills/gabo-runtime-opencode/plugins/gabo-stage-gate.js`, copied by the skill to `.opencode/plugins/gabo-stage-gate.js`. Reason: one catalog source next to the skill that installs it.
- **Builder hard-rule amendment (section 5)**: **Proposed (auto)** T10 narrows the rule in `gabo-runtime-builder`, `gabo-runtime-claude` and `gabo-runtime-pi` to "gabo writes only the engine keys it needs to operate, at project level only, merged, reported and verified by `gabo check`; never the user's global engine config". Reason: section 5 names it a plan item here.
- **TRD wording (S12, section 5 D27 and TRD 2 Stack rows)**: **Proposed (auto)** TRD 2 Supported runtimes and D27 for the four engines are `0002/T5`'s, the "JavaScript and TypeScript only as thin connection layers" Stack line is `0003/T3`'s; T9 adds only the OpenCode-specific wording (TRD 2 names OpenCode, D3, 8.1, 9.3, PRD) on top of `0002/T5`. Reason: one owner per sentence, no conflicting TRD PRs.
- **S7 worktree mechanism**: **Proposed (auto)** the skill's build appends to the built `gabo-coder` body an OpenCode-only first step: call `opencode.session_move` to the worktree path from the launch prompt, else use `shell` `workdir` and absolute paths (D6, spec open question 2 recommendation (a)). Reason: the rule is runtime-specific, so it lives in the runtime skill.

## Steps

1. Create `gabo-runtime-opencode`: frontmatter, sections, Sources, Tool Map, agent build with the version comment, deny-only permissions and reviewer rules (T2); then skill links, the `Skills:` line and the reviewer-configuration link (T3); then prerequisites, the project `experimental.subagent_depth` write, coexistence and the coder worktree step (T4). Delivers S2 to S9, S14.
2. Teach `gabo check` the OpenCode project: version comment, links and `experimental.subagent_depth`, on the four-engine enum of `0002/T1` (T5). Delivers S11 and the check of S1.
3. Ship the stage-gate plugin as a thin layer over `gabo check --pre-write` and install it in the build (T6). Delivers S10.
4. Add the `opencode run` reviewer engine on the marker of `0001/T5` and the preflight of `0002/T2`; the wave-start refusal is `0002/T3` (T8). Delivers S15.
5. Update the TRD and PRD (T9) and align the runtime skills' engine-config rule (T10). Delivers S12 and the section 5 plan items.
6. Owner-machine verification run, the ten steps of S13 plus the S14 and S15 manual checks (T11). Delivers S13.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | (removed 2026-11-03) Accept `claude`, `codex`, `pi` and `opencode` for every `runtimes.*` role | — | — | — | Removed: duplicate of `0002/T1` (#204), which makes the shared enum edit for all four engines; S1 is checked in T5 on top of it. | S |
| T2 | Catalog skill `gabo-runtime-opencode`: sections, Sources, Tool Map, agent build and reviewer rules | S2, S3, S4, S6, D1, D2, D3, D5, D8 | — | T3 | Test first: failing `internal/skills/runtimeopencode_test.go` (as `runtimepi_test.go`): the skill is in `skills.Names()`; frontmatter `name: gabo-runtime-opencode`, `metadata.agent_dir: .opencode/agents`, `metadata.skill_dir: .opencode/skills`, no `metadata.reviewer_link`; sections Activation Contract, Hard Rules, Sources (dated, OpenCode 2.0.32, with the recheck rule that stops the build when `opencode --version` differs and a row no longer holds), Tool Map, Fixed-Name Links, Build Steps, Output Contract; the agent rule writes `.opencode/agents/<id>.md` with only `description` (double-quoted role), `mode: subagent` and `permission`, body starting `<!-- gabo-version: ` then the `Skills:` line; the Tool Map rows of spec D3 with deny-only keys and no `allow`; the reviewer rules: no `gabo-reviewer` link, `permission.skill` `gabo-reviewer: deny` on every agent but `gabo-reviewer`, `{{reviewer_script}}` written literally with `${XDG_DATA_HOME:-$HOME/.local/share}`; existing skill-list tests updated. | M |
| T3 | Skill links, the `Skills:` line and the reviewer-configuration link in `gabo-runtime-opencode` | S5, S14, D4, D11, D12 | T2, 0001/T4 | T4 | Test first: failing assertions in `runtimeopencode_test.go`: the link rule writes `.opencode/skills/<name>` → `../../.gabo/skills/<name>` for each spec skill and `gabo-<slot>` (`_` written `-`) for each `SKILL.md` slot adapter, all relative symlinks; it links `gabo-reviewer-config` the same way and a rebuild keeps it; it removes any other `.opencode/skills/gabo-*` link and never creates `.opencode/skills/gabo-reviewer`; the `Skills:` line reads `Skills: before work read .opencode/skills/<name>/SKILL.md for each of <...>.`; a Go test over a sample tree built by those rules shows `gabo check` reports no `✗ skill link` line for `.opencode/skills`. | M |
| T4 | Prerequisites, project `experimental.subagent_depth` write, coexistence and the coder worktree step in `gabo-runtime-opencode` | S7, S8, S9, D6, D7, D8 | T3 | T6 | Test first: failing assertions in `runtimeopencode_test.go`: the build first checks `opencode --version`; the depth rule names the lookup order `opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `.opencode/opencode.jsonc`, else a new `.opencode/opencode.json`; writes only `experimental.subagent_depth: 3`, merging and keeping every other key and `.jsonc` comments, leaves 3 or more unchanged, never writes `~/.config/opencode/` or `.opencode/package.json`, reports file, key, old and new value; on missing `opencode` or an unmergeable file it builds and writes nothing and prints key, value 3 and file; the write list of S9 is the only one; the built `gabo-coder` body gains the OpenCode first step (`opencode.session_move` to the launch-prompt worktree, else `shell` `workdir` and absolute paths). | M |
| T5 | `gabo check` verifies an OpenCode project: version comment, links and `experimental.subagent_depth` | S11, S1, S3, D2, D7 | 0002/T1 | T11 | Test first: failing Go tests in `internal/slices/check` with a `testdata` OpenCode tree: a `gabo.yaml` with `runtimes.coder`, `runtimes.assistant` and `runtimes.reviewer` all `opencode` reports no runtime-value failure (enum of `0002/T1`); an agent with `<!-- gabo-version: <old> -->` yields `✗ agent <id> built with <old>, catalog is <new>`; a missing link yields `✗ skill link .opencode/skills/<name> missing`; no project config, or `experimental.subagent_depth` below 3 in the first existing file of the S8 order, yields one `✗` line naming `experimental.subagent_depth`, the value 3 and the file; 3 or more passes; projects without an `opencode` runtime keep their output (golden on existing fixtures). | M |
| T6 | Stage-gate plugin `gabo-stage-gate.js` over `gabo check --pre-write`, installed by every build | S10, D10 | T4, 0001/T1 | T11 | Test first: failing Go test that runs the plugin with `node` against a stubbed `$` and a fake `gabo` on `PATH` recording argv (skipped with a named reason when `node` is missing): an `edit`, `write` or `patch` call passes each target path to `gabo check --pre-write <path...>`; gabo exit 1 → `tool.execute.before` throws with gabo's output; exit 0 → no throw; exit 2 or no `gabo` on `PATH` → no throw and one warning; other tools never call `gabo`; a grep test proves the source has no `import`, no `require`, and no `spec.md`, `Acceptance` or `plan.md` rule; the skill copies it to `.opencode/plugins/gabo-stage-gate.js` and writes no `package.json`. | M |
| T7 | (removed 2026-11-03) Wave-start gate on the reviewer isolation pass marker | — | — | — | Removed: duplicate of `0002/T2` (#206, `reviewer.sh preflight` for `codex` and `opencode`) and `0002/T3` (#207, `gabo-cycle` refuses the wave), which enforce the marker for both engines; T8 and T11 depend on them. | M |
| T8 | `opencode` engine in the deterministic reviewer scripts | S15, D9, D5 | 0002/T1, 0002/T2, 0001/T5 | T11 | Test first: failing shell tests in `gabo-reviewer/scripts/tests/` with a fake `opencode` recording argv and environment: `REVIEWER_ENGINE=opencode` is accepted in `common.sh`; the agent-review step runs `opencode run --standalone --format json` with the diff by `--file`, outside the checkout, from an `env -i` environment with its own `HOME` and `XDG_*`, a reviewer-owned config holding one read-only agent (`edit` denied, `shell` limited to `git diff`, `git log`, `git show`, `git status`) and no GitHub token; the final assistant text is the review and its `VERDICT` line is read as for the other engines; a PR adding `.opencode/plugins/x.js`, `.opencode/agents/` or `opencode.json` is not passed to the run; `git status` of the checkout is unchanged; missing `opencode` fails naming `REVIEWER_OPENCODE`; the deterministic gates still run first; the flags match `0001/T5`, and the `opencode` pass-marker preflight of `0002/T2` runs before the engine. Human-merged (`skills/gabo-reviewer/`). | M |
| T9 | TRD and PRD name OpenCode as a supported runtime | S12, D9, D10 | 0002/T5 | — | Docs PR touching `docs/product/TRD.md` and `docs/product/PRD.md`, each change citing intent 0004, on top of `0002/T5` (four engines in TRD 2 and D27) and `0003/T3` (thin-layer Stack line), repeating neither: TRD 2 Supported runtimes lists Claude Code, Pi and OpenCode by name; D3 cites intent 0004; 8.1 names `gabo-runtime-opencode` and its body version comment; 9.3 lists `opencode` for `runtimes.*`; PRD runtime lists name OpenCode; `gabo check` passes. | S |
| T10 | Narrow the "never edit the engine's own config" rule in the runtime skills | S8, D7 | — | — | Test first: failing assertions in `runtimebuilder_test.go`, `runtimeclaude_test.go` and `runtimepi_test.go`: each skill's rule now reads that gabo writes only the engine keys it needs to operate, at project level only, merged with the existing project config, reported and verified by `gabo check`, never the user's global engine config; the rest of each rule (no install, no upgrade) is unchanged. | S |
| T11 | Owner-machine verification of `gabo-runtime-opencode` | S13, S5, S6, S7, S9, S14, S15 | T5, T6, T8, 0002/T3 | — | Owner runs spec S13 steps 1 to 10 on local OpenCode (2.0.32 or the installed version) in a sample repository, plus the reviewer-configuration link probe (S14) and one `opencode` engine review with a planted `.opencode/plugins/x.js` (S15), and records each result on the task as pass or not run with the reason; a fail blocks closing the intent; every **(binary)** fact is rechecked and any drift is a plan change. | S |

## Tracker

Intent tracker item: [#175](https://github.com/testeos-z/gabo-ai/issues/175) under product card #171.

| task | issue |
|---|---|
| T1 | [#193](https://github.com/testeos-z/gabo-ai/issues/193) (closed: removed 2026-11-03) |
| T2 | [#194](https://github.com/testeos-z/gabo-ai/issues/194) |
| T3 | [#195](https://github.com/testeos-z/gabo-ai/issues/195) |
| T4 | [#197](https://github.com/testeos-z/gabo-ai/issues/197) |
| T5 | [#198](https://github.com/testeos-z/gabo-ai/issues/198) |
| T6 | [#199](https://github.com/testeos-z/gabo-ai/issues/199) |
| T7 | [#200](https://github.com/testeos-z/gabo-ai/issues/200) (closed: removed 2026-11-03) |
| T8 | [#202](https://github.com/testeos-z/gabo-ai/issues/202) |
| T9 | [#203](https://github.com/testeos-z/gabo-ai/issues/203) |
| T10 | [#205](https://github.com/testeos-z/gabo-ai/issues/205) |
| T11 | [#208](https://github.com/testeos-z/gabo-ai/issues/208) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| `runtimes.*: opencode` is accepted by `0002/T1` before this intent ships the runtime skill and the `opencode` reviewer engine. | T5, T8 | Fails closed: the `0002/T2` preflight refuses an `opencode` reviewer without its marker, and `gabo check` reports missing agents for an `opencode` coder until T2 to T4 build them. |
| A **(binary)** fact (`session_move` from a subagent, `shell`/`subagent` permission aliases, `experimental.subagent_depth`) does not hold in the installed OpenCode. | T2, T4, T11 | The skill's recheck rule stops the build; T11 step 10 records the drift and the fallback (D6 `workdir`, extra deny keys) is a task-scope change. |
| `opencode run --standalone` still loads a PR-planted plugin. | T8, T11 | `0001/T5` refuses the engine at configuration time and `0002/T2`, `0002/T3` keep waves from starting (spec open question 12 (c)). |
| `node` is missing where the reviewer runs `go test`. | T6 | The plugin test skips with a named reason; the rule itself is tested in Go by `0001/T1`. |
| Unattended sibling-worktree waves meet the `external_directory` ask (spec open question 3 still open). | T4, T11 | Recorded in T11; once the owner decides, the key is written at project level under the D7 rule as a task-scope change to T4. |
| Three tasks edit the same `SKILL.md`. | T2, T3, T4, T6 | Chained by `depends_on` so they never run in parallel. |

## Open questions

1. T1 and T7 were removed on 2026-11-03, the day they were created, as duplicates of `0002/T1`, `0002/T2` and `0002/T3` (issues #193 and #200 closed with that reason). The intent 0002 plan's `unblocks` column should list `0004/T5`, `0004/T8` (from `0002/T1`), `0004/T8` (from `0002/T2`), `0004/T11` (from `0002/T3`) and `0004/T9` (from `0002/T5`); the tracker carries these edges (gabo-dev-lead for 0002).
2. Intent 0001's plan `unblocks` column does not list the cross-intent edges `0001/T1` → `0004/T6`, `0001/T4` → `0004/T3`, `0001/T5` → `0004/T8`; the tracker carries them, the 0001 table should mirror them (gabo-dev-lead for 0001).
3. T6 tests the plugin with `node` under `go test`; `0003/T3` adds `bun test` for thin connection layers to the TRD. If the owner prefers `bun test` for this plugin too, T6 changes its runner as a task-scope change (owner).
4. Spec open questions 1, 2, 4, 8, 9 and 10: the plan follows each recommendation; a different owner choice is a plan change (owner).
5. Spec open question 3 (`external_directory` for sibling worktrees) stays open; T11 records whether unattended waves work without it (owner).
