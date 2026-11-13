# Intent 0027 Coders follow gentle-ai ODD and RDD

| field | value |
|---|---|
| intent id | 0027-coders-follow-gentle-ai-odd-and-rdd |
| status | Draft |
| owner | gabo-pm |
| product card | none open: [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) is closed, so no card was created; the intent issue is linked under #492 as #703 was |
| date | 2026-11-13 |

## Problem

The owner asked that the coders use gentle-ai as closely as possible to its ODD and its RDD. Owner, verbatim, relayed 2026-11-13 by the owner's local assistant: "quiero que los coder usen gentle-ai lo mas apegado a ODD y RDD".

Today a coder loads the methodology adapter, but the adapter is not followed. The evidence is in the repository and in the coder runs reported by the assistant:

- The adapter is wired in: `gabo.yaml:27-28` sets `slots.methodology.adapter`, and `.claude/skills/gabo-methodology` is a symlink to the gentle-ai adapter. The Claude coder agent preloads `gabo-methodology` (`.claude/agents/gabo-coder.md:8`) and is told to follow the methodology slot adapter (`.claude/agents/gabo-coder.md:14`). The coder skill tells the coder to load that adapter before reading code and says the skill carries no methodology of its own (`apps/cli/internal/skills/gabo-coder/SKILL.md:11`).
- The adapter already states the RDD rule: it runs only when `gentle-ai review mode status` is on, and the coder then runs assess on the commit and follows its transitions (`.claude/skills/gabo-methodology-gentle-ai/SKILL.md`, lines 17 and 24, and the coder-side review operation at lines 38-52).
- Reported by the assistant from session claude-b2e28fa5, not re-checked against the transcripts from this run: in 13 coder runs of waves I0023, I0025 and I0026 the adapter text was preloaded in all 13. Only one run (I0025 T3, PR #706) ran `gentle-ai review start` and `status`, and it stopped on the per-candidate consent envelope (`gentle-ai.review-integration.consent/v3`), which a coder cannot answer. The owner's assistant relayed it, the owner granted it, and the review ran after merge. One run (I0026 T8, PR #739) ran only `review mode status` and reported skipping assess as a process miss. The other eleven ran no gentle-ai review command and reported no RDD outcome. No coder created an ODD feature document (`odd/tasks/`).
- The adapter's assess command does not work on committed work as written. Checked on 2026-11-13 with gentle-ai 4.4.0 in a clean worktree at `origin/dev` (nothing pending): `gentle-ai review assess --cwd <wt> --json` returned `risk: high`, `reason: unassessable`, `review_due: true`, and an error that names `--base-ref` and `--committed-only` as the way to assess committed work.

The result: the owner's RDD and ODD are not visible in any wave. A coder's report does not say whether RDD ran, the wave report cannot show it, and a consent envelope has no route to the owner.

## Who is affected

- The owner, who wants the coders to build the way gentle-ai builds, and who needs to see that they did in each wave report.
- The assistant that runs each wave. It relays consent envelopes and reads the RDD outcome of every task.
- Every coder task of every wave. The gap appears in each one of the 13 runs reported above, so it is the normal case, not an exception.
- The gabo reviewer, which stays the merge gate and must not be replaced by a coder-side review.

## Desired outcome

- Every coder task reports its ODD evidence (RED and GREEN, or the stated exception, with the spec ids it covered) and its RDD outcome. The outcome is one of `granted`, `declined`, `passive`, `off`, `unavailable`, or the gentle-ai reason when no review was due. A missing item reads `not run: <reason>`, never as a pass.
- Every wave report shows that outcome for each task, so the owner can count how many tasks ran RDD, and with what result, without reading the coder transcripts.
- A coder never answers a consent envelope, never toggles RDD, and never lowers a risk tier. A consent envelope reaches the owner through a route the owner decides.
- The gabo reviewer stays the only merge gate (PRD R9). RDD adds an outside view and never replaces the reviewer.
- Measure: the share of tasks whose wave report line carries an RDD outcome, per wave. The target is to be set by the owner at acceptance.

## Constraints

- PRD R8: the coder uses the methodology declared in the methodology slot. R9: two layers, an optional coder-side review per methodology and a mandatory reviewer-side merge gate, which never merges into main. R5: every question a coder raises reaches the assistant and the wave report. R15: in auto mode the human receives no prompts, only the final answer. R25: an interrupted wave resumes from tracker state and traces. R29: every wave produces a report. R31: the CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt`, so this intent adds no CLI command.
- gentle-ai rules, as the adapter and the orchestration rules of gentle-ai state them: global RDD on never grants consent for a candidate, consent is candidate-scoped; a coder never runs `gentle-ai review mode enable|disable`; an unavailable or failed assessment never lowers the risk tier.
- `gabo.yaml:53` lists `.claude/` in `review.human_only_paths`, and the user-level reviewer scripts merge a change to such a path only by hand (`common.sh:256-257`). The adapter and the coder agent live under `.claude/`, so their change merges by a human, not by the reviewer.
- The methodology contract (`apps/cli/internal/skills/gabo-contract-methodology/SKILL.md`) requires an adapter to declare its coder-side review explicitly and never to weaken the coder's hard rules.
- Accepted specs 0023, 0025 and 0026 are not reopened. Their behavior changes only through this intent's spec.
- Owner preference (intent 0026, #700 comment): few custom scripts; prefer the engine's own features.
- Assumption, not decided: this intent binds the gentle-ai adapter that gabo-ai ships and the coders that load it. Other methodology adapters keep their own contract; the methodology contract changes only if the owner asks for it.

## Open questions

| # | question | status |
|---|---|---|
| 1 | How does a per-candidate RDD consent reach the owner when a coder cannot give it, and what happens to the task meanwhile? | open; **Proposed (auto)** in the spec, section 6 |
| 2 | Does the ODD feature document (`odd/tasks/<feature>.md` and its Engram mirror) get created per task, or do the gabo plan row and the task issue act as it? | open; **Proposed (auto)** in the spec, section 6 |
| 3 | When does RDD run next to the gabo reviewer, and which one gates the merge? | open; **Proposed (auto)** in the spec, section 6 |
| 4 | Is the RDD candidate each work-unit commit before the push, or the PR head? | open; **Proposed (auto)** in the spec, section 6 |
| 5 | What happens when RDD is unavailable or its assessment fails? | open; **Proposed (auto)** in the spec, section 6 |
| 6 | Beyond visibility, does a gate enforce the coder's ODD and RDD? | open; **Proposed (auto)** in the spec, section 6 |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it) |
| intent issue | [#758 I0027: Coders follow gentle-ai ODD and RDD](https://github.com/testeos-z/gabo-ai/issues/758) |
| PRD | [PRD](../../PRD.md) |
| TRD | [TRD](../../TRD.md) sections 9.1 (methodology slot), 9.2 (coder outcome, wave report), D6 |
| spec | [spec](spec.md) |
| plan | none yet: the plan is written only after the owner accepts the spec |
| related intent | [0026](../0026-pi-probe-follow-ups/intent.md) (same probe evidence, accepted spec untouched) |
| sources | coder runs of I0023, I0025 (PR #706) and I0026 (PR #739), reported by the assistant, session claude-b2e28fa5; gentle-ai 4.4.0 `review assess --help` and the 2026-11-13 assess check above |
