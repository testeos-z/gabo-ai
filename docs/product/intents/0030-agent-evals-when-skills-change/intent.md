# Intent 0030 Agent evals when skills change

| field | value |
|---|---|
| intent id | 0030-agent-evals-when-skills-change |
| status | Draft |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#758](https://github.com/testeos-z/gabo-ai/issues/758) was) |
| intent issue | [#767 I0030: Agent evals when skills change](https://github.com/testeos-z/gabo-ai/issues/767) |
| date | 2026-11-13 |

## Problem

gabo changes how its agents behave by editing skills and the methodology adapter, and nothing measures whether the agents still behave as intended after an edit. The owner asked for a set of real tasks that measures agent behavior when a skill or the methodology adapter changes. Owner, relayed by the owner's local assistant on 2026-11-13, as cut D of the playbook review (last in the order: A, then B, then C, then D).

Evidence, checked 2026-11-13 at `origin/dev` `63b36df`:

- **No eval exists.** A search of the repository for `eval` and `evals` finds only shell `eval` lines in `wave-record.sh` scripts. There is no task set, no runner and no baseline.
- **The catalog is large and changes often.** `apps/cli/internal/skills/` holds 34 `gabo-*` skill folders. The coder skill and the methodology adapter (`.claude/skills/gabo-methodology-gentle-ai`, a link into the catalog) govern every coder run.
- **A behavior drift already went unseen.** Per the spec of intent 0027 (evidence base, the assistant's report of 13 coder runs in waves 0023, 0025 and 0026, not re-checked from transcripts), the methodology adapter text was preloaded in all 13 runs. One run started the RDD review and stopped at a consent envelope, one ran only the mode status, and eleven ran no review command. No check flagged them, and the wave reports did not show it.
- **The playbook's evals section (Anthropic, "The AI-native SDLC playbook", Louis Claxton, 2026-08-21, https://claude.com/resources/articles/the-ai-native-sdlc-playbook).** Quoted from a page extraction on 2026-11-13, not checked byte for byte:
  - "Evals are the AI-native equivalent of stage-gate QA."
  - "In practice that means a suite that runs whenever the agent's configuration changes."
  - "The platform engineer collects 20 to 50 real tasks from recent work with its expected/accepted outcome."
  - "The suite runs non-interactively in CI on a schedule and on any change to `CLAUDE.md`, skills or hooks,…" (truncated by the extraction).
  - "Gate configuration changes on the results." and "A skill change that drops the pass rate gets reviewed before it merges."
  - "The pass-rate threshold is enforced as a merge check, runs are logged so results can be compared over time,…" (truncated).
  - "Depending on the use case, some teams may prefer to run these evals offline on a set cadence rather than on every change."
  - "Each production incident gets an eval, written by the team that owned the incident,…" (truncated).
  - Correction to the earlier summary, which said "a suite of 20 to 50 real tasks runs in CI": the page says the platform engineer collects 20 to 50 tasks, and the CI sentence is about a schedule and the listed changes, not a fixed size.
- **The local assistant's proposal**, relayed by the owner's assistant: start with ten tasks. Size and frequency are the owner's questions (Open questions).

## Who is affected

- The owner, who changes the skills and accepts or rejects each change against the measure.
- The assistant and every coder, whose behavior is measured. Each skill change affects every wave that loads the skill.
- The owner's review of every skill change, which today is a read of the diff.

## Scope

In scope:

- A small set of real tasks, each taken from past gabo work (a finished task and its PR) with its accepted outcome written down.
- A run of the set when a skill or the methodology adapter changes, on the owner's machine, with the result compared against an accepted baseline.
- A pass rate per task and for the set, reported before the owner accepts the change.
- The first baseline: the set run on the current catalog before any change is judged.

Non-goals:

- CI and package publishing. Both are out of all four intents of this batch (owner, 2026-11-13). The playbook's scheduled CI run is not proposed.
- A new CLI command (PRD R31). The runner is a skill or a script under the catalog, not `gabo eval`.
- Measuring product QA. Product QA of the product is intent 0027's reviewer work; the evals measure agent behavior on fixed tasks.
- Hooks as a trigger (see open question 2), and the playbook's other stages, which the owner has not discussed: the Maintain stage, permissions and isolation, CLAUDE.md practices, the fresh-context verifier, metrics and traces, and Design-stage security skills.
- Any merge gate in this intent. Whether a result blocks a change is the owner's decision (open question 5).

## Desired outcome

- A change to a gabo skill or the methodology adapter shows, before the owner accepts it, how many tasks of the set pass on the new text and which ones changed from the baseline.
- A task that passed on the accepted baseline and fails after a change is visible in the change's review, with the observed difference.
- The set grows from real failures: a wave's wrong behavior, once fixed, becomes a task with its accepted outcome.
- Measure: pass rate per run, and the count of tasks that moved from pass to fail. Targets set by the owner at acceptance.

## Constraints

- PRD R31 (`docs/product/PRD.md:170`): no new CLI command; agents do everything else with skills.
- The methodology slot (TRD 9.1) and the coder's rules (`apps/cli/internal/skills/gabo-coder/SKILL.md`) define what a correct run is; the evals measure against them and do not change them.
- Intent 0020 (spend or turn cap for auto mode, accepted 2026-11-12) sets how a bounded run is capped; the eval runs reuse its rule instead of a new one.
- Secrets: a task fixture and its result never hold a credential or a token (the memory adapter's rule, R21, `.claude/skills/gabo-memory-engram/SKILL.md`). Runs use the owner's login on the owner's machine, which is not stored in the set.
- Test-first for behavior changes (`AGENTS.md`, conventions). The set itself is tested by its runner before it measures a skill.
- Intent 0027's done contract and RDD evidence define the observable outcome of a coder task. A task of this set should check against them where the outcome is observable.

## Open questions

Each question has a **Proposed (auto)** option with a one-line reason. None is accepted.

| # | question | status |
|---|---|---|
| 1 | How many tasks in the set? | open. **Proposed (auto): ten to start, one baseline, then grow to twenty when the first ten run stably.** Reason: the local assistant's proposal of ten keeps the first run cheap enough to repeat; the playbook says 20 to 50 tasks, which is a target after the set proves useful. Options: (b) twenty from the start, which doubles the cost of the first baseline; (c) twenty to fifty from the start, as the playbook's collection size. |
| 2 | What triggers a run? | open. **Proposed (auto): a change to any `apps/cli/internal/skills/gabo-*/SKILL.md` or to the methodology adapter.** Reason: the owner's trigger, and the one this intent measures. Options: (b) the same plus hooks (intent 0028) and `CLAUDE.md` or `AGENTS.md`, which is the playbook's list (extraction: "any change to `CLAUDE.md`, skills or hooks"); (c) every merge into `dev`, which costs a run per PR; (d) a set cadence with no trigger, which the playbook allows (extraction: "run these evals offline on a set cadence rather than on every change"). |
| 3 | Where does a run happen? | open. **Proposed (auto): on the owner's machine, started by the owner or by the assistant when a trigger is met; never in CI.** Reason: CI is out of all four intents (owner, 2026-11-13), and the runs use the owner's engine login. Options: (b) a cloud agent session (intent 0019) for each run, which needs the session's cost and credentials handled there; (c) on the reviewer host, which runs the owner's scripts already. |
| 4 | What is the cost cap per run? | open. **Proposed (auto): a turn or spend cap set by the owner, using the cap rule of intent 0020; this intent does not invent a number.** Reason: the playbook calls for "an API key with budget for eval runs" (extraction), and the owner has the budget decision. Options: (b) a fixed amount per run written here, which would be a made-up number today; (c) no cap, which the owner did not ask for. |
| 5 | What pass threshold, and does it gate a change? | open. **Proposed (auto): no task that passed on the accepted baseline may fail after a change, and the overall rate is reported as advisory; the owner decides whether a failure blocks the change.** Reason: an absolute threshold such as 90 percent needs a baseline first, and a threshold set now would be invented. Options: (b) an absolute threshold set by the owner after the first baseline; (c) advisory only, with no rule about the baseline. |
| 6 | Where do tasks come from, and who accepts each one? | open. **Proposed (auto): the assistant proposes each task from a finished gabo task (its issue, its PR and its wave record), and the owner accepts it and its expected outcome.** Reason: the playbook's source is real recent work, and the owner must agree what "accepted" means. Option (b): the owner writes every task, which is slower but needs no assistant proposal. |
| 7 | What counts as a pass: the outcome only, or the behavior too? | open. **Proposed (auto): the outcome and the behavior evidence of intent 0027: the expected result, plus the ODD evidence (RED and GREEN, or the stated exception), the covered `S#` and the RDD outcome line.** Reason: a run that reaches the right result without the method is the failure the owner described. Option (b): the outcome only, which misses a skipped method. Option (c): the commands run as well, which needs a tool-call record that no engine gives today. |
| 8 | How many runs per task, and how are they reported? | open. **Proposed (auto): three runs per task, reported as the count of passes out of three; a task counts as passed only when it passes in every run.** Reason: one run of an agent is noisy, and "every run" is the strict reading of "must not fail". Option (b): one run per task, which is cheap and noisy. Option (c): the count of runs set by the owner after the first baseline. |
| 9 | Which engine runs the set? | open. **Proposed (auto): the coder engine in `gabo.yaml` (`runtimes.coder`, `claude` today), one engine per run.** Reason: the set measures the skill on the engine that runs it in waves. Option (b): every full-tier engine (intent 0013 runs several engines in parallel), which multiplies the cost by the number of engines. |
| 10 | Is an eval the same as a product QA check of intent 0027? | open. **Proposed (auto): no. An eval measures the agent on a fixed task; product QA measures the product a task produced.** Reason: the two answer different questions and fail for different reasons. Option (b): share the done contract of intent 0027 as the expected outcome, which couples the two intents and needs 0027 accepted first. |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#767 I0030: Agent evals when skills change](https://github.com/testeos-z/gabo-ai/issues/767) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) (the done contract and RDD evidence this intent measures against) |
| sibling intent, B | [0028 Deterministic hooks as locks](../0028-deterministic-hooks-as-locks/intent.md), [#765](https://github.com/testeos-z/gabo-ai/issues/765) (a possible trigger, question 2) |
| sibling intent, C | [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md), [#766](https://github.com/testeos-z/gabo-ai/issues/766) |
| related intents | [0013](../0013-several-engines-in-parallel-in-one-project/intent.md) (engines in parallel), [0019](../0019-talk-to-cloud-agent-sessions/intent.md) (cloud sessions), [0020](../0020-spend-or-turn-cap-for-auto-mode/intent.md) (the cap rule), [0022](../0022-measure-roles/intent.md) (measuring roles) |
| PRD | [PRD](../../PRD.md) (R31) |
| TRD | [TRD](../../TRD.md) section 9.1 (methodology slot) |
| spec | none yet; written only after the owner reviews this intent |
| plan | none yet |
| sources | [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, 2026-08-21 (extraction, not byte-checked); `origin/dev` at `63b36df` for every repository path cited; the spec 0027 evidence base for the 13 coder runs (assistant report, not re-checked) |
