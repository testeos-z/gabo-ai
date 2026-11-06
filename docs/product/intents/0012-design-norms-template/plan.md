# Plan 0012 Design norms template

| field | value |
|---|---|
| intent | [0012-design-norms-template](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-06 |

The spec was accepted by the owner on 2026-11-06 (spec section 7), with every **Proposed (auto)** answer; spec open questions 1 (D2, where the principles and the component inventory live) and 2 (D4, `gabo check` and `DESIGN.md`) are still with the owner. This plan is written for both proposed answers (D2 A, D4 A). Only T4 changes between D2 answers A and C, so only T4 is BLOCKED on the owner; T1 to T3 hold under A and C and can run. D2 answer B (a global `UX_UI.md` template again) would reopen spec S2 and the TRD D14 amendment the owner accepted with this spec, so it comes back as a plan change for the whole intent (see Open questions). D4 changes no task: answer B adds a new task through a plan change. `gabo.yaml` mode is `auto`: every fixed question below is answered by `gabo-dev-lead`, marked **Proposed (auto)** with a one-line reason, for one-pass approval.

## Fixed questions

| id | answer |
|---|---|
| Q1 | **Proposed (auto)**: one serial chain: the norms template (T1), then the spec template section 3 that takes over the messages table and links the norms (T2), then the `gabo-frontend` and `gabo-doc-guide` skill text (T3), then the component inventory placement once the owner answers D2 (T4) (Steps). Reason: every task edits `apps/cli/internal/skills/templates_test.go`, T2 links section numbers T1 creates, T3 names what T1 and T2 ship, and T4 edits the files of T1 and T3. |
| Q2 | **Proposed (auto)**: four tasks, each one PR (Tasks). Reason: the template with its byte copy is the largest diff (about 400 lines); the spec template, the skill text and the owner-dependent inventory are small and reviewable alone. |
| Q3 | **Proposed (auto)**: see the `spec refs` column. Reason: each task names the requirements and decisions it proves. |
| Q4 | **Proposed (auto)**: see `depends_on` and `unblocks`, acyclic: T1 → T2 → T3 → T4. No cross-intent dependency. Reason: serial edits of `templates_test.go` and of the same skill folders avoid merge conflicts. |
| Q5 | **Proposed (auto)**: see `acceptance` and `size`; each task starts from failing expectations in `apps/cli/internal/skills/templates_test.go` (spec S5, D5). Reason: TRD test-first convention and AGENTS.md. |
| Q6 | **Proposed (auto)**: yes: S1 (T1), S2 (T1), S3 (T1, T2, T4), S4 (T3, T4), S5 (T1, T2, T3, T4), S6 (T3), S7 (every task), S8 (every task). Reason: checked against the spec table. |
| Q7 | **Proposed (auto)**: see Risks to the plan. |
| Q8 | **Proposed (auto)**: see Open questions. |
| Q9 | **Proposed (auto)**, applying intent 0008 S10 (TRD 8.5) and spec S7 for the proposed D4 answer A: no `gabo check` update. `gabo check` reads no `DESIGN.md`, no skill template and no spec section 3, and `DESIGN.md` is written by an agent, not by gabo, so it is outside the 8.5 list; the template shape is pinned by the catalog tests. Every task states `gabo check impact: none` with that reason. A D4 answer B adds a `gabo check` task through a plan change. Reason: owner rule, verbatim: "1. pondremos una regla cada cambio debe revisar si es necesario actualizar el check". |

## Steps

1. The norms template `gabo-frontend/templates/DESIGN.md` is rebuilt from the PR #163 template (D1 A): 14 fixed questions, the 15 sections, the rules, exceptions and changelog columns, the rule-writing guidance, per-change header and no-GUI guidance; the old messages table leaves the template; the existing component inventory table is carried into section 12 unchanged until T4 (T1). Delivers S1, S2, S3 (no messages table in the norms), S5.
2. The spec template `gabo-pm/templates/spec.md` section 3 gets the messages table with the change's concrete cases and links `DESIGN.md` sections 2-3, 9 and 10 instead of tokens, generic accessibility or wording rules (T2). Delivers S3 (spec side), S5.
3. `gabo-frontend/SKILL.md` names the norms template, the one-rules-table-per-area shape, exceptions and changelog, the section 3 messages table and links, and the conversion rule for an existing `DESIGN.md` in another shape (D3 A), with its version bumped; `gabo-doc-guide` describes `DESIGN.md` as the design norms (T3). Delivers S4, S6, S5.
4. Once the owner answers D2: the shared component inventory gets its place (A: `DESIGN.md` section 12 with `stories` next to the `K` rules, and the skill step says so) (T4, BLOCKED). Delivers S3 (inventory), S4 (inventory), S5.
5. `gabo check` needs no update (Q9, spec S7): every task states `gabo check impact: none` with the reason. No task changes `docs/product/` (S8).

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

No PR of this plan touches `docs/product/` (spec S8, TRD D7), `.claude/`, `CLAUDE.md`, `AGENTS.md`, `gabo.yaml` or `apps/cli/internal/skills/skills_test.go`. Every task edits a catalog skill and refreshes its `plugins/codex/skills/<skill>/` byte copy in the same PR (`codexplugin_test.go` `TestCodexPluginSkillsCopyTheCatalog` names the first drifted path), so every PR touches `plugins/`, a `review.human_only_paths` entry: the coder opens it, and only the owner or the coordinator holding the owner's delegation merges it. `plugins/claude-code/skills/` and `.claude/skills/` are links and need no change. No version bump (`dev` is already at 0.2.10). Every PR body states `gabo check impact: none` with the spec S7 reason.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | `gabo-frontend` design norms template | S1, S2, S3, S5, S7, S8, D1, D5 | — | T2 | Test first: the `gabo-frontend/templates/DESIGN.md` entry of `apps/cli/internal/skills/templates_test.go` expects the title `# Design norms`, the 15 headings `## 1. Principles and sources` to `## 15. Open questions` of spec S1, the column rows `\| id \| rule \| level \| rationale \| verification \| examples \| source \|`, `\| rule id \| exception \| reason \| owner \| date \|` and `\| date \| rule ids \| change \| decision \|`, and keeps `\| component \| level \| states \| used by \|` (carried into section 12 until T4); new assertions in `templates_test.go` require the phrases that a rule without a verification is SHOULD at most, that ids are stable and never reused, that an exception or rule change is a row in sections 13 or 14, the spec section 3 sentence (UX and flow of each change live in that intent's spec section 3, which cites rule ids) and the no-GUI sentence (visual sections "not applicable" with the reason, conversation or CLI conventions under sections 1 and 10), and forbid `UX_UI`, `APP_FLOW` and `\| case \| what happened \|` in the template. The PR quotes the RED run (`missing heading` or `missing columns` on the old template), then the template is rebuilt from PR #163 (`b238d454d7a1548faf7a6b3a34dd7dee980706c0`, `apps/cli/internal/skills/gabo-frontend/templates/DESIGN.md`; its `SKILL.md` and PRD edits are not taken) with the header linking the PRD, the TRD and `docs/product/intents/`, and the GREEN run is quoted; `TestEveryTemplateHasParseableFixedQuestions` parses the 14 fixed questions; `go -C apps/cli test ./...`, `go -C apps/cli vet ./...` and gofmt pass. gabo check impact: none (spec S7: `gabo check` reads no `DESIGN.md` and no skill template; the shape is pinned by the catalog tests). Edit surfaces: `apps/cli/internal/skills/gabo-frontend/templates/DESIGN.md`, `apps/cli/internal/skills/templates_test.go`, `plugins/codex/skills/gabo-frontend/templates/DESIGN.md` (`review.human_only_paths`). | M |
| T2 | Spec template section 3 holds the change's messages and links the norms | S3, S5, S7, S8 | T1 | T3 | Test first: the `gabo-pm/templates/spec.md` entry of `templates_test.go` expects `\| case \| what happened \| why \| what to do next \|` in section `## 3. UX and app flow (optional)`, and new assertions require links to `DESIGN.md` sections 2-3 (tokens, color), 9 (accessibility) and 10 (wording), the sentence that product-specific keyboard paths stay in the journey, and no `\| token \|` table in the spec template; the PR quotes the RED run, then section 3 and the guidance line at the top of the template ("Global styles stay in the repository `DESIGN.md`" becomes the design norms wording) change, and the GREEN run is quoted; the existing spec columns and headings stay unchanged (`TestTemplatesKeepStableColumnsAndSections`); `go -C apps/cli test ./...`, `go -C apps/cli vet ./...` and gofmt pass. gabo check impact: none (spec S7: `gabo check` reads the spec acceptance table, never section 3). Edit surfaces: `apps/cli/internal/skills/gabo-pm/templates/spec.md`, `apps/cli/internal/skills/templates_test.go`, `plugins/codex/skills/gabo-pm/templates/spec.md` (`review.human_only_paths`). | S |
| T3 | `gabo-frontend` and `gabo-doc-guide` describe the design norms | S4, S5, S6, S7, S8, D3 | T2 | T4 | Test first: new assertions in `templates_test.go` (extending `TestDirectivaSkillsPointToTheirTemplates` and `TestPlanningSkillsOwnTheirIntentArtifacts`, not `skills_test.go`) require `gabo-frontend/SKILL.md` to name `templates/DESIGN.md` as the design norms, one rules table per area with an RFC 2119 level, rationale, verification, examples and source, exceptions and a changelog; the spec step to name the section 3 messages table and its `DESIGN.md` links; the conversion rule of S6 (an existing `DESIGN.md` in another shape is never rewritten silently: the next update proposes the conversion, to the owner in manual mode or as **Proposed (auto)** in auto mode, keeps every rule with a new id and records one changelog row); and `gabo-doc-guide/SKILL.md` to describe `DESIGN.md` as the design norms instead of "global styles". The PR quotes the RED run, then the two skills change: `gabo-frontend` metadata version `1.2` to `1.3`, `gabo-doc-guide` `2.0` to `2.1`, no `UX_UI` or `APP_FLOW` (`TestPlanningSkillsDropRetiredGlobalPlanning`), the existing `skills_test.go` doc-guide phrases still present; the GREEN run is quoted; `go -C apps/cli test ./...` (including `TestCodexPluginSkillsCopyTheCatalog`), `go -C apps/cli vet ./...` and gofmt pass. The skill does not yet say where the component inventory lives (T4). gabo check impact: none (spec S7: `gabo check` reads no catalog skill text). Edit surfaces: `apps/cli/internal/skills/gabo-frontend/SKILL.md`, `apps/cli/internal/skills/gabo-doc-guide/SKILL.md`, `apps/cli/internal/skills/templates_test.go`, `plugins/codex/skills/gabo-frontend/SKILL.md`, `plugins/codex/skills/gabo-doc-guide/SKILL.md` (`review.human_only_paths`). | S |
| T4 | Shared component inventory in `DESIGN.md` section 12 | S3, S4, S5, S7, S8, D2 | T3 | — | **BLOCKED on the owner's answer to spec D2 (open question 1).** Planned for the proposed answer A: test first, the `DESIGN.md` entry of `templates_test.go` expects `\| component \| level \| states \| stories \| used by \|` (replacing `\| component \| level \| states \| used by \|`) and new assertions require it under `## 12. Components` next to the `K` rules table, and require `gabo-frontend/SKILL.md` to say the `DESIGN.md` step writes the shared component inventory and the principles as its only non-rule content; the PR quotes the RED run, then the template section 12 and the skill change (`gabo-frontend` version `1.3` to `1.4`), the GREEN run is quoted; `go -C apps/cli test ./...`, `go -C apps/cli vet ./...` and gofmt pass. Under answer C this row is amended by a plan change: the inventory table leaves `DESIGN.md` and goes to the spec template section 3 (`gabo-pm/templates/spec.md`). gabo check impact: none (spec S7). Edit surfaces: `apps/cli/internal/skills/gabo-frontend/templates/DESIGN.md`, `apps/cli/internal/skills/gabo-frontend/SKILL.md`, `apps/cli/internal/skills/templates_test.go`, `plugins/codex/skills/gabo-frontend/templates/DESIGN.md`, `plugins/codex/skills/gabo-frontend/SKILL.md` (`review.human_only_paths`). | S |

## Tracker

Intent tracker item: [#374](https://github.com/testeos-z/gabo-ai/issues/374) under product card #171.

| task | issue |
|---|---|
| T1 | [#395](https://github.com/testeos-z/gabo-ai/issues/395) |
| T2 | [#396](https://github.com/testeos-z/gabo-ai/issues/396) |
| T3 | [#397](https://github.com/testeos-z/gabo-ai/issues/397) |
| T4 | [#398](https://github.com/testeos-z/gabo-ai/issues/398) |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| Between T1 and T2 merging, neither the norms template nor the spec template carries the messages table. | T1, T2 | T2 depends only on T1 and is small; both ship in 0.2.10 and no release is cut between them. |
| The owner answers D2 with B (a global `UX_UI.md` template again). | T1 to T4 | B reopens spec S2 and the TRD D14 amendment accepted with this spec; it comes back as a plan change for the whole intent before T1 starts if the answer arrives first, or as follow-up tasks after it. |
| The owner answers D4 with B (`gabo check` validates `DESIGN.md`). | none of T1 to T4 | A plan change adds a `gabo check` task with a TRD 9.4 and PRD R33 amendment; no existing task changes. |
| `plugins/codex/skills/` copies drift from the catalog. | T1 to T4 | Each PR refreshes its copy; `codexplugin_test.go` names the first drifted path and the refresh command. |
| `plugins/` is a human-only path. | T1 to T4 | The coder opens the PR; the owner or the coordinator with the owner's delegation merges. |
| Intent 0013 edits the catalog skill tests in parallel. | T1 to T4 | 0012 edits only `templates_test.go` among the test files and only the `gabo-frontend`, `gabo-pm` (template) and `gabo-doc-guide` skill folders; intent 0013's spec names none of them. A conflict in `templates_test.go` is resolved by rebasing the later PR. |
| A cleanup edit reaches `docs/product/` in a code PR. | all | Each PR is checked against S8 before review. |

## Open questions

1. Spec D2, open question 1 (owner): where the principles and the shared component inventory live. Proposed answer A; T4 is BLOCKED until the owner answers. Answer C amends T4 only; answer B reopens the intent (see Risks).
2. Spec D4, open question 2 (owner): whether `gabo check` validates `DESIGN.md`. Proposed answer A; no task waits on it. Answer B adds a task through a plan change.
3. The open issues #169 (T078) and #170 (T079) of the closed plan PR #168 describe the same work under the retired model and are labeled `status:blocked`; this plan supersedes them. Closing them is the coordinator's or the owner's call (coordinator).
