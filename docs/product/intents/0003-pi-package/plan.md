# Plan 0003 Pi package

| field | value |
|---|---|
| intent | [0003-pi-package](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-03 |

The spec was accepted by the owner on 2026-11-03 (spec section 7). `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: Go first (the `gabo_facts` stdin shape on the shared hook mode of `0001/T1` and `0001/T2`), then the TRD wording, then the package tree and its skills, then the TypeScript extension, then docs and the owner-machine run (Steps). Reason: the extension only relays what Go returns (D9), so the Go contract must exist before the thin layer that calls it. |
| Q2 | **Proposed (auto)**: nine tasks T1 to T9, each one PR of at most about 400 changed lines (Tasks). Reason: the Go facts rules, the version and stage-gate rules, the TRD wording, the package tree, the shared skill copy, each extension handler and the docs are separately reviewable behaviors. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements and decisions it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`; cross-intent dependencies `0001/T2` (T1), `0001/T3` (T3), `0001/T4` and `0001/T5` (T5); acyclic. Reason: the hook mode, its TRD naming and the reviewer-configuration skill are built once in intent 0001 and reused here, never duplicated. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; every code task starts from a failing Go or `bun test`. Reason: TRD test-first convention. |
| Q6 | **Proposed (auto)**: yes, S1 to S10 each have at least one task (S2 and S9 manual halves only through T9). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions; spec open questions 1, 2, 4 and 5 stay with the owner and do not block any task. |

Plan items the spec left to `gabo-dev-lead`, answered here:

- **Session-facts flag and keys (D9)**: **Proposed (auto)** the same `gabo check --pre-write` with no path argument; stdin is one flat JSON object `{"gabo_facts": 1, "event": "session_start" | "tool_call", "engine": "pi", "tools": [...], "package_version": "x.y.z", "target_paths": [...]}`, where `gabo_facts` holds the schema version and `target_paths` is required only for `tool_call`. Reason: one key is both the shape discriminator against the Claude Code and Codex shapes of `0001/T2` and the schema version, and the spec names every field.
- **Session-facts output**: **Proposed (auto)** stdout is exactly one JSON object `{"notify": [{"level": "warning" | "info", "text": "…"}], "block": null | {"reason": "…"}}`; exit 0 without a block, 1 with a block, 2 on a usage or input error (unknown schema version, `engine` other than `pi`, unknown `event`, `tool_call` without `target_paths`, malformed JSON) with one usage line on stderr. When several rules block one call, `block.reason` holds each text on its own line, the subagent refusal first. Reason: the spec's D9 shape, with a deterministic order the extension never reorders.
- **Subagent tool list**: **Proposed (auto)** one Go constant list `["subagent_run"]` beside the refusal-text constant in `apps/cli/internal/slices/check`. Reason: D2 default; adding a package is a CLI release (spec owner doubt 4).
- **Pi tool-call target path**: **Proposed (auto)** the extension reads `event.input.path` of Pi's `write` and `edit` tools and passes it unchanged as the one `target_paths` entry. Reason: Pi 1.4.0 `write`/`edit` take `path`; Go normalizes relative and absolute paths as `0001/T1` does.
- **npm package name**: **Proposed (auto)** `@testeos-z/gabo-pi` in `packages/pi/package.json` (spec open question 5, recommendation B). Reason: the name is one field; the owner can still change it before the first publish (Open questions 1).
- **Reviewer-configuration skill in the package (S10)**: **Proposed (auto)** a committed copy at `packages/pi/skills/gabo-reviewer-config/`, kept byte-identical to `apps/cli/internal/skills/gabo-reviewer-config/` by a `bun test`. Reason: npm tarballs do not reliably carry symlinks, and Pi installs from the npm tarball (D5); drift fails the test, not the user.
- **TRD amendment (spec section 5 row 1, open question 6)**: **Proposed (auto)** task T3, one docs PR. Reason: the owner approved it as a plan item; intent 0004 S10 relies on the same sentence.

## Steps

1. Add the `gabo_facts` stdin shape to the shared Go hook mode with the subagent rule and the refusal-text constant (T1), then the version rule and the stage gate for `tool_call` (T2). Delivers the Go half of S4, S5, S6, S7; D2, D7, D9.
2. Amend the TRD: TypeScript and JavaScript only as thin connection layers, `bun test` for them, and the `gabo_facts` shape in 9.4 (T3). Spec section 5 row 1.
3. Create `packages/pi/` with its manifest and the `gabo-pi` bootstrap skill (T4), then add the shared reviewer-configuration skill copy (T5). Delivers S1, S3, S10; D1, D5, D6.
4. Add the extension's `session_start` handler (T6), then its `tool_call` handler (T7). Delivers the TypeScript half of S4 to S7 and S8.
5. Document install, coexistence, the release checklist and the S9 verification procedure (T8). Delivers S9 (written), S7 release check, S2 and S8 documentation.
6. Owner-machine verification run on Pi 1.4.0 (T9). Delivers S2, S9 and the manual checks of S4 to S8 and S10.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Go hook mode: `gabo_facts` session-facts stdin shape, subagent rule and refusal-text constant | S4, S5, D2, D7, D9 | 0001/T2 | T2, T6 | Test first: failing Go tests in `apps/cli/internal/slices/check` and `apps/cli/cmd/gabo`: a `session_start` facts JSON whose `tools` lack `subagent_run` → exit 0 and stdout `{"notify":[{"level":"warning","text":<refusal>}],"block":null}`; with `subagent_run` → no notification; a `tool_call` on `.pi/agents/gabo-coder.md` without `subagent_run` → exit 1 and `block.reason` equals the refusal text; with `subagent_run` → exit 0, no block; a `tool_call` on any other path → no block from this rule; malformed JSON, unknown schema version, `engine` other than `pi`, unknown `event` or `tool_call` without `target_paths` → exit 2 with one usage line; a test reads the embedded catalog `gabo-runtime-pi/SKILL.md` Prerequisite step 4 quoted block and asserts it equals the refusal constant byte for byte; the Claude Code and Codex shapes of `0001/T2` and the no-flag `gabo check` golden output are unchanged; no file is written. | M |
| T2 | Go hook mode: version rule and stage gate for `gabo_facts` | S6, S7, D4, D9 | T1 | T3, T7 | Test first: failing Go tests: `session_start` with `package_version` `0.5.0` and `.gabo/VERSION` `0.4.0` → one `info` notification naming `0.5.0`, `0.4.0` and the next step (`gabo init` or a package update); equal versions or no `.gabo/VERSION` → none; a `tool_call` whose target is an intent `plan.md` with an unaccepted spec → exit 1 and `block.reason` equals the `✗` line of the path-argument mode of `0001/T1` for that path; accepted spec → exit 0; unreadable spec → blocked (fail closed); a `tool_call` blocked by both rules → one reason, subagent refusal line first; the stage gate reuses the `0001/T1` rule code with no copy. | M |
| T3 | TRD: TypeScript and JavaScript only as thin connection layers, and the `gabo_facts` shape in 9.4 | S4, S9, D9 | T2, 0001/T3 | — | Docs-only PR touching `docs/product/TRD.md`: TRD 2 Stack says TypeScript and JavaScript are allowed only as thin connection layers with no rule logic (owner 2026-11-03, intent 0003 open question 6); TRD 11 adds `bun test` for those layers; TRD 9.4 names the `gabo_facts` stdin shape, its output object and exit codes next to the `--pre-write` flag named by `0001/T3`, citing intent 0003; `gabo check` passes. | S |
| T4 | Package `packages/pi/`: manifest and `gabo-pi` bootstrap skill | S1, S3, D1, D5, D6 | — | T5, T6 | Test first: failing `bun test` in `packages/pi/`: `package.json` has `name` `@testeos-z/gabo-pi`, `version` a release `x.y.z` (the CLI has no in-repo version source, it is injected by `-ldflags`, so lockstep with the tag is the T8 release check), `keywords` containing `pi-package`, `pi.skills` and `pi.extensions` listing exactly the shipped directories and files, the Pi host packages (`@earendil-works/*`, `typebox`) only in `peerDependencies` with `"*"` and none in `dependencies`; `npm pack --dry-run --json` lists `package.json`, the listed skill and extension files and nothing named `gabo-reviewer`; the `gabo-pi` name equals no catalog skill name in `apps/cli/internal/skills/`, no slot link name in `gabo-agent-templates/agents.yaml` and no `gabo-<slot>` name; `skills/gabo-pi/SKILL.md` contains `gabo prompt`, `gabo init` and the CLI install step, and no line of `docs/setup-prompt.md`; `extensions/gabo.ts` is a no-op stub registered in `pi.extensions`; the workspace `test` script runs `bun test`. | M |
| T5 | Ship the reviewer-configuration skill `gabo-reviewer-config` in the Pi package | S10, S1, S3 | T4, 0001/T4, 0001/T5 | T8 | Test first: failing `bun test`: `packages/pi/skills/gabo-reviewer-config/` is byte-identical, file by file including its isolation-test script, to `apps/cli/internal/skills/gabo-reviewer-config/`; `pi.skills` lists it; `npm pack --dry-run --json` holds it and still nothing named `gabo-reviewer`; its name passes the T4 name-collision check; no skill in the package runs reviews (no `reviewer.sh` invocation). | S |
| T6 | Extension `session_start` handler: collect facts, invoke the gabo hook mode, show gabo's notifications | S4, S7, S8, D2, D9 | T1, T4 | T7 | Test first: failing `bun test` with a fake `ExtensionAPI` and a fake `gabo` on `PATH` that records its argv and stdin: with `gabo.yaml` in the working directory the handler runs `gabo check --pre-write` once with stdin `{"gabo_facts":1,"event":"session_start","engine":"pi","tools":<every name from pi.getAllTools()>,"package_version":<package.json version>}` and calls `ctx.ui.notify` once per returned notification with gabo's text and level unchanged; an empty `notify` shows nothing; without `gabo.yaml` `gabo` is not run; `gabo` missing from `PATH` or exiting 2 → one warning, nothing else; the handler writes no file; a source test proves `extensions/` and `src/` hold no `subagent_run`, no line of the refusal text, no `.gabo/VERSION` read and no version comparison. | M |
| T7 | Extension `tool_call` handler: relay `write`/`edit` targets and apply gabo's block | S5, S6, S8, D3, D4, D9 | T2, T6 | T8 | Test first: failing `bun test` with the fake `ExtensionAPI` and fake `gabo`: a `write` or `edit` call in a gabo project runs `gabo` once with the session facts of T6 plus `"event":"tool_call"` and `target_paths` `[event.input.path]`; gabo exit 1 → the handler returns `{ block: true, reason }` with `reason` equal to gabo's `block.reason`; exit 0 → no block; `gabo` missing or exit 2 → no block and one warning; other tools and calls outside a gabo project → `gabo` not run; a source test proves the extension tests no `.pi/agents/` path and parses no `spec.md` or Acceptance table. | M |
| T8 | Package README, release checklist and the S9 verification procedure | S2, S7, S8, S9, D5, D6, D8 | T5, T7 | T9 | Docs PR: `packages/pi/README.md` names `pi install npm:@testeos-z/gabo-pi@<version>` and the local-path install, `pi list`, that `gabo init` and `gabo-runtime-pi` stay required and that the package writes no project file and is not a subagent package (S8); the release checklist adds the version bump in lockstep with `apps/cli/v<x.y.z>`, the check `test "$(jq -r .version packages/pi/package.json)" = "<x.y.z>"` and `npm publish` by the owner; `packages/pi/VERIFY.md` lists the S9 steps (local install, `pi list`, start with gentle-pi, start with `--no-extensions -e <package>`, `.pi/agents/` write blocked, one stage-gate block, version notice, `pi remove`) each with its expected output; the root README links it. | S |
| T9 | Owner-machine verification of the Pi package on Pi 1.4.0 | S2, S4, S5, S6, S8, S9, S10 | T8 | — | Owner runs `packages/pi/VERIFY.md` on local Pi 1.4.0 and records each step's observed output on the task and in the intent's review findings: the package appears under "User packages"; no warning with gentle-pi; one warning with `--no-extensions -e <package>`; a `.pi/agents/` write and an unaccepted `plan.md` write are blocked with gabo's text; `git status` shows no change made by the package; `gabo check` passes; `gabo-runtime-pi` still refuses with only the gabo package installed; `pi.skills` shows `gabo-pi` and `gabo-reviewer-config`; `pi remove` succeeds. | S |

## Tracker

Intent tracker item: [#174](https://github.com/testeos-z/gabo-ai/issues/174) under product card #171.

| task | issue |
|---|---|
| T1 | [#187](https://github.com/testeos-z/gabo-ai/issues/187) |
| T2 | [#188](https://github.com/testeos-z/gabo-ai/issues/188) |
| T3 | [#189](https://github.com/testeos-z/gabo-ai/issues/189) |
| T4 | [#190](https://github.com/testeos-z/gabo-ai/issues/190) |
| T5 | [#191](https://github.com/testeos-z/gabo-ai/issues/191) |
| T6 | [#192](https://github.com/testeos-z/gabo-ai/issues/192) |
| T7 | [#196](https://github.com/testeos-z/gabo-ai/issues/196) |
| T8 | [#201](https://github.com/testeos-z/gabo-ai/issues/201) |
| T9 | [#209](https://github.com/testeos-z/gabo-ai/issues/209) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| The merge gate runs only the Go checks of `gabo.yaml` `review.checks`, so the `bun test` of T4 to T7 is not run by the reviewer, and there is no CI for `npm pack --dry-run`. | T4, T5, T6, T7 | Open questions 2: the owner adds `bun test --cwd packages/pi` to `review.checks` (`gabo.yaml` is human-only); until then the coder reports the `bun test` output in the PR and the reviewer reads it. |
| `0001/T2` lands a stdin dispatcher that cannot be extended additively with a third shape. | T1 | T1 depends on `0001/T2`; a needed refactor of the dispatcher stays inside T1 and keeps the `0001/T2` tests green. |
| `gabo-reviewer-config` changes in the catalog after T5, and the package copy drifts. | T5 | The T5 byte-equality test fails on any drift; the fix is a re-copy in the same PR as the catalog change. |
| Pi 1.4.0 `write`/`edit` input or `getAllTools()` differ from the spec's evidence. | T6, T7, T9 | The fake `ExtensionAPI` mirrors the 1.4.0 `extensions/types.d.ts`; T9 on the real Pi proves it; a mismatch is a task-scope change to T6 or T7. |
| T3 and `0001/T3` both edit TRD 9.4. | T3 | T3 depends on `0001/T3` and adds to its wording only. |
| No npm token or account exists for the first publish (D5). | T8, T9 | T9 verifies from the local path, as S2 and S9 allow; the publish is an owner release step in T8's checklist. |

## Open questions

1. Spec open questions 1 (package scope), 2 (npm only), 4 (subagent tool list) and 5 (npm name and owner account) stay with the owner; the plan builds the recommended answers (D1 B, D5 A, D2 A, `@testeos-z/gabo-pi`), and a different owner answer is a task-scope change to T4, T1 or T8 (owner).
2. Add `bun test --cwd packages/pi` to `gabo.yaml` `review.checks` so the merge gate runs the package tests; `gabo.yaml` is a human-only path (owner).
3. Intent 0004 S10 says "Bun test is not in the stack"; T3 adds `bun test` for thin connection layers to the TRD, which also answers that spec's open choice for its plugin test. The intent 0004 plan should cite T3 if it uses it (`gabo-dev-lead`, intent 0004 plan).
