# Plan 0007 Update documentation for 0.2.6

| field | value |
|---|---|
| intent | [0007-update-documentation-for-0-2-6](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-04 |

The spec was accepted by the owner on 2026-11-04 (spec section 7). `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: each document after the intent 0006 task it documents, then one consistency sweep after the 0.2.6 bump (Steps). Reason: spec D2, documents describe merged behavior. |
| Q2 | **Proposed (auto)**: six tasks, one per document plus the sweep, each one small docs PR (Tasks). Reason: one reviewable PR per page; `plugins/` READMEs are human-merged, so they stay apart. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic. Cross-intent: T1 on `0006/T6` and `0006/T9`; T2 on `0006/T9`; T3 on `0006/T6`; T4 on `0006/T2`, `0006/T3` and `0006/T6`; T5 on `0006/T4` and `0006/T6`; T6 on `0006/T10` and `0006/T12` (gabo-ai's own engine files, so `gabo check` passes). Reason: each document waits for the behavior it describes. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; each task starts from a failing grep or Go check that proves the stale text, then fixes it. Reason: TRD test-first convention, applied to docs as a failing check. |
| Q6 | **Proposed (auto)**: yes, S1 to S7 each have at least one task. Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions. |

## Steps

1. Root README: Compatibility at 0.2.6, the four engines, install sections with `gabo init` engine integration and `@gabo-ai/pi` (T1). Delivers S1 and the README half of intent 0006 S13.
2. Setup page prose and the OpenCode row (T2). Delivers S2.
3. Cloud agent flow: four engines and the `gabo init` row (T3). Delivers S3.
4. Plugin READMEs for Claude Code and Codex (T4). Delivers S4.
5. Pi package README (T5). Delivers S5.
6. Consistency sweep after the 0.2.6 bump (T6). Delivers S6, S7.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | README: Compatibility for 0.2.6, four engines, `gabo init` engine integration and `@gabo-ai/pi` | S1, S7, D1, D3 | 0006/T6, 0006/T9 | T6 | Test first: `grep -n "testeos-z/gabo-pi\|Status for gabo 0.2.2" README.md` matches before the change and not after; the Compatibility section reads "Status for gabo 0.2.6 (tag `apps/cli/v0.2.6`)" with rows for Claude Code, Codex, Pi and OpenCode checked against the code on `dev`; the Pi row and the Pi package section name `@gabo-ai/pi` and `pi install npm:@gabo-ai/pi@<version>`; the Claude Code, Codex and Pi install sections hold one paragraph on what `gabo init` writes (`.claude/settings.json`, `.codex/config.toml`, `.pi/settings.json`; nothing for OpenCode; never global) and what the report asks the user to run, with the manual commands kept as the alternative and the Codex fallback commands while `0006/T3`'s flag is set; Quick start and Upgrading name the second `gabo init`; only `README.md` changes. Edit surfaces: `README.md`. | M |
| T2 | Setup page: OpenCode row in "Your engine" and the second `gabo init` | S2, S7 | 0006/T9 | T6 | Test first: a grep for `gabo-runtime-opencode` in `docs/setup-prompt.md` outside the marked block fails before the change; after it the "Your engine" table has rows for Claude Code, Codex, Pi and OpenCode and the prose says `gabo init` configures the declared engines and runs again once `gabo.yaml` exists; the marked prompt block is unchanged by this task and `go -C apps/cli test ./internal/setupprompt/...` passes. Edit surfaces: `docs/setup-prompt.md` (prose outside the marked prompt block only). | S |
| T3 | Cloud agent flow: four engines and the `gabo init` engine integration row | S3, S7 | 0006/T6 | T6 | Test first: `grep -n "\`claude\` or \`pi\`" docs/cloud-agent-flow.md` matches before the change and not after; the sessions line and the review-engine line name `claude`, `codex`, `pi` or `opencode`; the `gabo init` row of the "what the CLI must automate" table adds the project-level engine integration keys and that user-level engine config is never written; nothing else in the flow changes. Edit surfaces: `docs/cloud-agent-flow.md`. | S |
| T4 | Claude Code and Codex plugin READMEs: `gabo init` writes the project keys | S4, S7, D3 | 0006/T2, 0006/T3, 0006/T6 | T6 | Test first: `grep -n "gabo init" plugins/claude-code/README.md plugins/codex/README.md` finds no Install-section paragraph before the change; after it each Install section says which keys `gabo init` writes and in which project file, what the user still does (trust the folder; Codex: trust both hooks in `/hooks`), and keeps the manual commands; the release checklists name 0.2.6 and `@gabo-ai/pi` where they name a version or the Pi package; `grep -rn "testeos-z/gabo-pi" plugins/` has no match. Human-merged (`plugins/` is human-only). Edit surfaces: `plugins/claude-code/README.md`, `plugins/codex/README.md`. | S |
| T5 | Pi package README: install `@gabo-ai/pi` from npm and through `.pi/settings.json` | S5, S7, D3 | 0006/T4, 0006/T6 | T6 | Test first: `grep -n "testeos-z/gabo-pi" packages/pi/README.md` matches before the change and not after; the first install command is `pi install npm:@gabo-ai/pi@<version>`; one paragraph says `gabo init` writes the `packages` entry `npm:@gabo-ai/pi@<gabo version>` to the project `.pi/settings.json` when `gabo.yaml` declares `pi`, loaded after the project is trusted; the checkout install stays as the development path; Versioning says the package version follows the gabo release (0.2.6). Edit surfaces: `packages/pi/README.md`. | S |
| T6 | 0.2.6 documentation consistency sweep | S6, S7 | T1, T2, T3, T4, T5, 0006/T10, 0006/T12 | — | Test first: `grep -rn "testeos-z/gabo-pi\|\`claude\` or \`pi\`" README.md docs/setup-prompt.md docs/cloud-agent-flow.md plugins/*/README.md packages/pi/README.md` has no match and every version the docs name as current is 0.2.6, matching `packages/pi/package.json` and both plugin manifests after `0006/T10`; any leftover is fixed in the same PR within the files of S1 to S5; `gabo check` and `go -C apps/cli test ./...` pass; the diff touches documentation only. Human-merged only if it touches `plugins/`. Edit surfaces: `README.md`, `docs/setup-prompt.md` (prose only), `docs/cloud-agent-flow.md`, `plugins/claude-code/README.md`, `plugins/codex/README.md`, `packages/pi/README.md`. | S |

## Tracker

Intent tracker item: [#263](https://github.com/testeos-z/gabo-ai/issues/263) under product card #171.

| task | issue |
|---|---|
| T1 | [#275](https://github.com/testeos-z/gabo-ai/issues/275) |
| T2 | [#276](https://github.com/testeos-z/gabo-ai/issues/276) |
| T3 | [#277](https://github.com/testeos-z/gabo-ai/issues/277) |
| T4 | [#278](https://github.com/testeos-z/gabo-ai/issues/278) |
| T5 | [#279](https://github.com/testeos-z/gabo-ai/issues/279) |
| T6 | [#280](https://github.com/testeos-z/gabo-ai/issues/280) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| PR #262 is not merged when T1, T4 or T5 run, so the code still names `@testeos-z/gabo-pi`. | T1, T4, T5, T6 | The docs follow the owner's decision (spec D3); T6 checks the name against `packages/pi/package.json` after `0006/T10`, which waits for #262. |
| The intent 0006 Codex fallback flag changes after the owner smoke (`0006/T11`). | T1, T4 | The fallback paragraph is worded "while the fallback is printed"; a later flip is a one-line docs follow-up. |
| `plugins/` is human-only. | T4, T6 | The owner authorized the coordinator (2026-11-04) to merge PRs into `dev` including human-only paths. |

## Open questions

1. None blocks a task. The intent 0006 smoke results (`0006/T11`) may need a later docs follow-up (owner).
