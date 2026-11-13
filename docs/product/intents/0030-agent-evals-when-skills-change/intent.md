# Intent 0030 Agent evals when skills change

| field | value |
|---|---|
| intent id | 0030-agent-evals-when-skills-change |
| status | Draft. The owner accepts the spec, not this intent (owner decision D-6, 2026-11-13) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#758](https://github.com/testeos-z/gabo-ai/issues/758) was) |
| intent issue | [#767 I0030: Agent evals when skills change](https://github.com/testeos-z/gabo-ai/issues/767) |
| date | 2026-11-13 (revised 2026-11-13 after the owner's decisions, see the Decided table) |

## Problem

gabo changes how its agents behave by editing skills and the methodology adapter, and nothing measures whether the agents still behave as intended after an edit. The owner asked for a set of real tasks that measures agent behavior when a skill or the methodology adapter changes. Owner, relayed by the owner's local assistant on 2026-11-13, as cut D of the playbook review (last in the order: A, then B, then C, then D).

The owner also asked for two more things in this intent, on 2026-11-13: the small Maintain version and the CLAUDE.md practices (see Scope). Owner, verbatim: "Esto sería de gran monitorización para obtener más posibles mejoras" [This would be great monitoring for getting more possible improvements] (Maintain). Owner, verbatim: "Listo, queda en el Intent D" [done, it stays in Intent D] (CLAUDE.md and the template).

Evidence, checked 2026-11-13 at `origin/dev` `63b36df`:

- **No eval exists.** A search of the repository for `eval` and `evals` finds only shell `eval` lines in `wave-record.sh` scripts. There is no task set, no runner and no baseline.
- **The catalog is large and changes often.** `apps/cli/internal/skills/` holds 34 `gabo-*` skill folders. The coder skill and the methodology adapter (`.claude/skills/gabo-methodology-gentle-ai`, a link into the catalog) govern every coder run.
- **A behavior drift already went unseen.** Per the spec of intent 0027 (evidence base, the assistant's report of 13 coder runs in waves 0023, 0025 and 0026, not re-checked from transcripts), the methodology adapter text was preloaded in all 13 runs. One run started the RDD review and stopped at a consent envelope, one ran only the mode status, and eleven ran no review command. No check flagged them, and the wave reports did not show it.
- **The playbook, Anthropic, "The AI-native SDLC playbook", Louis Claxton, 2026-08-21, https://claude.com/resources/articles/the-ai-native-sdlc-playbook.** Byte-checked on 2026-11-13 against the page text:
  - Evals: "Evals are the AI-native equivalent of stage-gate QA." "In practice that means a suite that runs whenever the agent's configuration changes." "The platform engineer collects 20 to 50 real tasks from recent work with its expected/accepted outcome." "The suite runs non-interactively in CI on a schedule and on any change to CLAUDE.md, skills or hooks," (the sentence continues on the page). "Gate configuration changes on the results." "A skill change that drops the pass rate gets reviewed before it merges." "Depending on the use case, some teams may prefer to run these evals offline on a set cadence rather than on every change." "Each production incident gets an eval, written by the team that owned the incident, and stays in the suite as a regression test." "The pass-rate threshold is enforced as a merge check, runs are logged so results can be compared over time, and the team that owns the configuration change approves it." "an API key with budget for eval runs" (prerequisite).
  - Maintain: "Any breached control band is diagnosed and written back into the loop as a new intent.md" (stage 6 summary). "When a fix ships, add an eval for the incident (the continuous evals play) to ensure that such issues are protected against going forwards." "writes the post-mortem to a version-controlled lessons file that future investigations can read" (the incident channel play).
  - CLAUDE.md: "list each command with an example of a healthy output." (the Commands section). "When Claude makes a mistake twice, the correction goes into CLAUDE.md." "Review findings feed back into CLAUDE.md." "When a review flags a mistake for the second time, the correction goes into CLAUDE.md as part of that review".
  - Correction to the earlier summary, which said "a suite of 20 to 50 real tasks runs in CI": the page says the platform engineer collects 20 to 50 tasks; the CI sentence is about a schedule and the listed changes, and it is not a fixed size.
- **The local assistant's proposal**, relayed by the owner's assistant: start with ten tasks. Size and frequency are the owner's questions (Open questions).

## Owner decisions, 2026-11-13

- D-1 CI on PRs stays out. The owner's choice departs from the playbook's CI run, and the intent keeps the eval off CI.
- D-3 Only a small Maintain version, not the whole stage.
- D-6 The owner accepts the spec, not the intent nor the plan.
- D-1 and D-3 are the owner's numbered choices as relayed by the assistant; their wording is not in the record, so they are not quoted verbatim.
- The owner's CLAUDE.md choice: review findings reach CLAUDE.md through an agent proposal that the owner accepts (see Scope, b3). This narrows the playbook's "as part of that review".

## Who is affected

- The owner, who changes the skills and accepts or rejects each change against the measure, and who accepts each incident's lesson and each CLAUDE.md change.
- The assistant and every coder, whose behavior is measured. Each skill change affects every wave that loads the skill.
- The owner's review of every skill change, which today is a read of the diff.

## Scope

In scope:

- A small set of real tasks, each taken from past gabo work (a finished task and its PR) with its accepted outcome written down.
- A run of the set when a skill, the methodology adapter, CLAUDE.md or a hook (intent 0028) changes, on the owner's machine, with the result compared against an accepted baseline.
- A pass rate per task and for the set, reported before the owner accepts the change.
- The first baseline: the set run on the current catalog before any change is judged.
- **(a) Maintain, small version.** An incident in a wave: (a1) it becomes a Draft intent in the Stage 1 format (`intent.md`); (a2) it is recorded in a version-controlled lessons file; (a3) when its fix ships, one task is added to the evals, with its accepted outcome.
- **(b) CLAUDE.md practices.** (b1) A commands block: each command with an example of its healthy output. (b2) A repeated-mistakes section, with the two-times rule: a mistake made twice goes into CLAUDE.md. (b3) Review findings for a repeated mistake become an agent proposal to CLAUDE.md, which the owner accepts before it lands; CLAUDE.md is a human-only path (`gabo.yaml:53`), so an agent cannot merge it. (b4) A template for projects that use gabo.

Non-goals:

- CI runs of any kind on PRs (D-1). No scheduled CI run is proposed either.
- Package publishing (out of all four intents of this batch).
- A new CLI command (PRD R31). The runner is a skill or a script under the catalog, not `gabo eval`.
- Measuring product QA. Product QA of the product is intent 0027's reviewer work; the evals measure agent behavior on fixed tasks.
- A merge gate in this intent. A pass-rate drop is reviewed before merge; whether a check blocks it is open question 7.
- The rest of the playbook's Maintain stage: detection bands, scheduled scans, and the on-call channel. The fresh-context verifier, metrics and traces, and Design-stage security skills are not in this intent either.

## Desired outcome

- A change to a gabo skill, the methodology adapter, CLAUDE.md or a hook shows, before the owner accepts it, how many tasks of the set pass on the new text and which ones changed from the baseline.
- A task that passed on the accepted baseline and fails after a change is visible in the change's review, with the observed difference.
- The set grows from real failures: an incident in a wave, once its fix ships, becomes a task with its accepted outcome; a mistake made twice becomes a CLAUDE.md entry the owner accepted.
- Measure: pass rate per run, and the count of tasks that moved from pass to fail. Targets set by the owner at acceptance.

## Constraints

- PRD R31 (`docs/product/PRD.md:170`): no new CLI command; agents do everything else with skills.
- The methodology slot (TRD 9.1) and the coder's rules (`apps/cli/internal/skills/gabo-coder/SKILL.md`) define what a correct run is; the evals measure against them and do not change them.
- Intent 0020 (spend or turn cap for auto mode, accepted 2026-11-12) sets how a bounded run is capped; the eval runs reuse its rule instead of a new one.
- Secrets: a task fixture and its result never hold a credential or a token (the memory adapter's rule, R21, `.claude/skills/gabo-memory-engram/SKILL.md`). Runs use the owner's login on the owner's machine, which is not stored in the set.
- Test-first for behavior changes (`AGENTS.md`, conventions). The set itself is tested by its runner before it measures a skill.
- Intent 0027's done contract and RDD evidence define the observable outcome of a coder task. A task of this set should check against them where the outcome is observable.
- CLAUDE.md and `gabo.yaml` are human-only paths (`gabo.yaml:53`, `review.human_only_paths`): an agent proposes a change, and the owner's acceptance lands it.

## Decided (owner, 2026-11-13)

| # | was question | decision | basis |
|---|---|---|---|
| 1 | 2, what triggers a run | A change to a `gabo-*` skill, the methodology adapter, CLAUDE.md or a hook (intent 0028). Not every merge. | Playbook: "The suite runs non-interactively in CI on a schedule and on any change to CLAUDE.md, skills or hooks," The CI part is out (D-1); the trigger list is kept. |
| 2 | 3, where a run happens | On the owner's machine, started by the owner or by the assistant when the trigger fires. Never on a PR. | D-1 (CI on PRs stays out). |
| 3 | 6, where tasks come from and who accepts them | The assistant proposes each task from finished gabo work (its issue, PR and wave record); the owner accepts the task and its expected outcome. | Playbook: "The platform engineer collects 20 to 50 real tasks from recent work with its expected/accepted outcome." |
| 4 | 10, is an eval the same as product QA | No. An eval measures the agent on a fixed task. | Playbook: the eval suite "says whether the agent still does the work to the same standard." |
| 5 | Maintain timing | An eval task is added when the incident's fix ships, not when the incident happens. | Playbook: "When a fix ships, add an eval for the incident (the continuous evals play) to ensure that such issues are protected against going forwards." |
| 6 | CLAUDE.md, the two-times rule and the commands block | Adopted as written in scope (b1, b2). | Playbook: "When Claude makes a mistake twice, the correction goes into CLAUDE.md."; "list each command with an example of a healthy output." |
| 7 | CLAUDE.md, review findings | Adopted with the owner's acceptance step (b3). The playbook says the correction goes in "as part of that review". | Owner decision (the CLAUDE.md choice above). Playbook: "When a review flags a mistake for the second time, the correction goes into CLAUDE.md as part of that review". |

## Open questions

Each question has a **Proposed (auto)** option with a one-line reason. None is accepted. Answers go into the spec, which the owner accepts (D-6).

| # | question | status |
|---|---|---|
| 1 | How many tasks at the start? | open. **Proposed (auto): ten, then one baseline, and grow toward twenty when the first ten run stably.** Reason: the assistant's ten keeps the first run cheap enough to repeat; the playbook's "20 to 50 real tasks" is a collection size, reached in phases (the playbook's own modular wording: "organizations may choose to prioritize transforming different stages at different times"). Option: twenty from the start, which doubles the first baseline's cost. |
| 2 | What is the cost cap per run? | open. **Proposed (auto): the cap rule of intent 0020, with the number set by the owner; this intent does not invent a number.** Reason: the playbook asks for "an API key with budget for eval runs", and the budget is the owner's decision. Option: a fixed number written here, which would be invented today. |
| 3 | What counts as a pass: the outcome only, or the behavior too? | open. **Proposed (auto): the outcome plus intent 0027's ODD evidence (RED and GREEN, or the stated exception), the covered `S#`, and the RDD outcome line.** Reason: a run that reaches the right result without the method is the failure the owner described. Option: the outcome only, which misses a skipped method; the commands run too, which needs a tool-call record no engine gives today. |
| 4 | How many runs per task, and how are they reported? | open. **Proposed (auto): three runs per task, reported as passes out of three; a task passes only when it passes all three.** Reason: one run of an agent is noisy, and "every run" is the strict reading of "must not fail". Option: one run per task, which is cheap and noisy. |
| 5 | Which engine runs the set? | open. **Proposed (auto): the coder engine in `gabo.yaml` (`runtimes.coder`, `claude` today), one engine per run.** Reason: the set measures the skill on the engine that runs it in waves. Option: every full-tier engine (intent 0013), which multiplies the cost by the number of engines. |
| 6 | What is an incident in a wave? | open. **Proposed (auto): a coder run that reported done and was later found wrong by the owner, the assistant or the reviewer, recorded in the wave trace.** Reason: the Maintain version needs a trigger the assistant can see in the trace. Option: only a reviewer blocker, which misses the failures the reviewer does not see. |
| 7 | Is a pass-rate drop an automatic merge check, or only a review? | open. **Proposed (auto): only a review. A drop in pass rate is reviewed before merge, and the owner decides; no absolute number now.** Reason: the playbook says "A skill change that drops the pass rate gets reviewed before it merges."; an absolute threshold needs a baseline that does not exist yet. Option: a threshold set by the owner after the first baseline, enforced as a merge check; the playbook's sentence "The pass-rate threshold is enforced as a merge check" does not name CI, so the owner also decides where that check runs. |
| 8 | Retitle intent 0030? | open. **Proposed (auto): keep the folder name; retitle to "Agent evals, incident lessons and CLAUDE.md practices" (the README row changes with it).** Reason: CLAUDE.md practices and the Maintain version are not evals, so the current title no longer fits. Option: keep the title and open the CLAUDE.md practices as a sibling intent (no folder is created by this document). |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#767 I0030: Agent evals when skills change](https://github.com/testeos-z/gabo-ai/issues/767) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) (the done contract and RDD evidence this intent measures against) |
| sibling intent, B | [0028 Deterministic hooks as locks](../0028-deterministic-hooks-as-locks/intent.md), [#765](https://github.com/testeos-z/gabo-ai/issues/765) (a trigger: hook changes) |
| sibling intent, C | [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md), [#766](https://github.com/testeos-z/gabo-ai/issues/766) |
| related intents | [0013](../0013-several-engines-in-parallel-in-one-project/intent.md) (engines in parallel), [0019](../0019-talk-to-cloud-agent-sessions/intent.md) (cloud sessions), [0020](../0020-spend-or-turn-cap-for-auto-mode/intent.md) (the cap rule), [0022](../0022-measure-roles/intent.md) (measuring roles) |
| PRD | [PRD](../../PRD.md) (R31) |
| TRD | [TRD](../../TRD.md) section 9.1 (methodology slot) |
| spec | none yet; written only after the owner reviews this intent |
| plan | none yet |
| sources | [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, 2026-08-21 (quotes byte-checked 2026-11-13); `origin/dev` at `63b36df` for every repository path cited; the spec 0027 evidence base for the 13 coder runs (assistant report, not re-checked) |
