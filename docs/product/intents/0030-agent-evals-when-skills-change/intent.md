# Intent 0030 Agent evals, incident lessons and CLAUDE.md practices

| field | value |
|---|---|
| intent id | 0030-agent-evals-when-skills-change |
| status | Draft. Open questions closed 2026-11-13; the spec has not started (the owner ordered the intents first, then the specs). The owner accepts the spec, not this intent (owner decision D-6, 2026-11-13) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed; intent issue linked under it, as [#758](https://github.com/testeos-z/gabo-ai/issues/758) was) |
| intent issue | [#767 I0030: Agent evals when skills change](https://github.com/testeos-z/gabo-ai/issues/767) (its title predates the retitle of decision 15) |
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
- **The local assistant's proposal**, relayed by the owner's assistant: start with ten tasks. The owner now sets the bounds (decision 8).

## Owner decisions, 2026-11-13

Relayed by the owner's local assistant (laptop), on the owner's behalf. The owner answered by voice, question by question, on 2026-11-13. Owner words are quoted in the language the owner used, with an English gloss in brackets; everything else in this section and in the Decided table is the assistant's summary of the owner's answers, not an owner quote.

- Criterion (relayed summary, not a quote): no arbitrary numbers; everything is configurable.
- Questions the owner did not name, verbatim: "aprobado, siga. Tómala a las técnicas, toma las recomendadas" [approved, go ahead. Take it with the techniques, take the recommended ones]. Those questions take their Proposed (auto) option, adjusted where it clashed with a decision in the Decided table.
- Task count, verbatim: "por defecto va a venir el mínimo dos y máximo diez... son ampliables" [by default it will come with a minimum of two and a maximum of ten... they are extendable].
- Cost cap, verbatim: "que sea modificable por el usuario, manualmente quiero ponerlo" [it should be modifiable by the user, I want to set it manually].
- Runs per task (relayed summary, not a quote): configurable, one by default, which is what the playbook's example does.
- Threshold, verbatim: "Pongamos el umbral... en el mismo PR lo puede colocar, y que la decisión final sea del humano, porque puede ir más lento pero ser más seguro" [Let's set the threshold... in the same PR it can place it, and that the final decision be the human's, because it may go slower but be safer].
- Owner's CLAUDE.md choice: review findings reach CLAUDE.md through an agent proposal that the owner accepts (see Scope, b3). This narrows the playbook's "as part of that review".
- D-1 CI on PRs stays out. The owner's choice departs from the playbook's CI run, and the intent keeps the eval off CI.
- D-3 Only a small Maintain version, not the whole stage.
- D-6 The owner accepts the spec, not the intent nor the plan.
- D-1 and D-3 are the owner's numbered choices as relayed by the assistant; their wording is not in the record, so they are not quoted verbatim.

## Who is affected

- The owner, who changes the skills and accepts or rejects each change against the measure, who accepts each incident's lesson and each CLAUDE.md change, and who sets the task bounds, the cost cap and the threshold.
- The assistant and every coder, whose behavior is measured. Each skill change affects every wave that loads the skill.
- The owner's review of every skill change, which today is a read of the diff.

## Scope

In scope:

- A small set of real tasks, each taken from past gabo work (a finished task and its PR) with its accepted outcome written down. The set holds at least 2 and at most 10 tasks by default, configurable and extendable per project (decision 8).
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
- A merge gate beyond the warning. A pass-rate run below the owner's threshold leaves a warning on the PR, which holds that PR for a person (decision 14). The reviewer stays the merge gate (PRD R9), and no CI check is added.
- The rest of the playbook's Maintain stage: detection bands, scheduled scans, and the on-call channel. The fresh-context verifier, metrics and traces, and Design-stage security skills are not in this intent either.

## Desired outcome

- A change to a gabo skill, the methodology adapter, CLAUDE.md or a hook shows, before the owner accepts it, how many tasks of the set pass on the new text and which ones changed from the baseline.
- A task that passed on the accepted baseline and fails after a change is visible in the change's review, with the observed difference.
- A run below the owner's threshold leaves a warning with its run data on the same PR. That PR does not merge by itself; a person decides (decision 14).
- The set grows from real failures: an incident in a wave, once its fix ships, becomes a task with its accepted outcome; a mistake made twice becomes a CLAUDE.md entry the owner accepted.
- Measure: pass rate per run, and the count of tasks that moved from pass to fail. Targets and the threshold value are set by the owner by hand, and they can be changed (decisions 9 and 14).

## Constraints

- PRD R31 (`docs/product/PRD.md:170`): no new CLI command; agents do everything else with skills.
- The methodology slot (TRD 9.1) and the coder's rules (`apps/cli/internal/skills/gabo-coder/SKILL.md`) define what a correct run is; the evals measure against them and do not change them.
- Intent 0020 (spend or turn cap for auto mode, accepted 2026-11-12) sets how a bounded run is capped; the eval runs reuse its rule instead of a new one. The number is the owner's, set by hand (decision 9).
- Secrets: a task fixture and its result never hold a credential or a token (the memory adapter's rule, R21, `.claude/skills/gabo-memory-engram/SKILL.md`). Runs use the owner's login on the owner's machine, which is not stored in the set.
- Test-first for behavior changes (`AGENTS.md`, conventions). The set itself is tested by its runner before it measures a skill.
- Intent 0027's done contract and RDD evidence define the observable outcome of a coder task. A task of this set should check against them where the outcome is observable.
- CLAUDE.md and `gabo.yaml` are human-only paths (`gabo.yaml:53`, `review.human_only_paths`): an agent proposes a change, and the owner's acceptance lands it.
- **Interplay with the reviewer's auto-merge (decision 14).** `gabo.yaml:52` sets `review.auto_merge: true`, so the reviewer merges clean heads into `dev` today. A PR with a below-threshold eval warning must not merge by itself. The reviewer's merge script lives outside this repository (`~/.local/share/gabo/skills/gabo-reviewer/scripts/merge-head.sh`); it already stops a merge for human-only paths with a `NEEDS-HUMAN-MERGE` outcome (`merge-head.sh:77-90`). The spec decides how the eval warning reaches that rule. A change to the reviewer's script is outside this PR and needs its own intent or plan change.

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
| 8 | 1, how many tasks | A minimum of 2 and a maximum of 10 tasks by default, configurable and extendable per project. The playbook's "20 to 50" stays as the reference; the default maximum sits below it. This replaces the earlier "ten, then grow toward twenty". | Owner: "por defecto va a venir el mínimo dos y máximo diez... son ampliables" [by default it will come with a minimum of two and a maximum of ten... they are extendable]. Playbook: "20 to 50 real tasks" (collection size, reference only). |
| 9 | 2, cost cap per run | The cap rule of intent 0020, with no number in this intent. The owner sets the value by hand, and it can be changed. | Owner: "que sea modificable por el usuario, manualmente quiero ponerlo" [it should be modifiable by the user, I want to set it manually]. Intent 0020's rule (accepted 2026-11-12). Recommended option kept, the value now the owner's. |
| 10 | 3, what counts as a pass | The outcome, plus intent 0027's ODD evidence (RED and GREEN, or the stated exception), the covered `S#`, and the RDD outcome line. | Owner approved the recommended option. |
| 11 | 4, runs per task | One run per task by default, configurable. A task passes when it passes all configured runs. The earlier "three runs" and "passes all three" are dropped. | Owner (relayed summary): configurable, one by default, which is what the playbook's example does (it runs `evals/*.json` once). The three-run figure was the assistant's, not the playbook's. |
| 12 | 5, which engine runs the set | The coder engine in `gabo.yaml` (`runtimes.coder`, `claude` today), one engine per run. | Owner approved the recommended option. |
| 13 | 6, what an incident is | A coder run that reported done and was later found wrong by the owner, the assistant or the reviewer, recorded in the wave trace. | Owner approved the recommended option. |
| 14 | 7, is a pass-rate drop a merge check | The owner sets the threshold value by hand. A run below it: gabo posts a warning with the run's data on the same PR; that PR does not merge by itself; a person decides. The check runs on the owner's machine with the eval, not in CI (D-1). The reviewer's auto-merge must not take a PR with the warning (see Constraints); the spec decides the mechanism. The recommended "review only, no number" is replaced. | Owner: "Pongamos el umbral... en el mismo PR lo puede colocar, y que la decisión final sea del humano, porque puede ir más lento pero ser más seguro" [Let's set the threshold... in the same PR it can place it, and that the final decision be the human's, because it may go slower but be safer]. Playbook: "A skill change that drops the pass rate gets reviewed before it merges."; "The pass-rate threshold is enforced as a merge check". |
| 15 | 8, the title | Retitled "Agent evals, incident lessons and CLAUDE.md practices". The folder name is kept, and the README row changes in this PR. The intent issue #767 still carries the old title; renaming it is a tracker write outside this PR. | Owner approved the recommended option. |

## Open questions

None. Questions 1 to 8 of the previous draft are closed in Decided rows 8 to 15 (the mapping is in the `was question` column, which keeps the old numbers).

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) (closed) |
| intent issue | [#767 I0030: Agent evals when skills change](https://github.com/testeos-z/gabo-ai/issues/767) (old title; retitle is a tracker write outside this PR) |
| sibling intent, A | [0027 Coders and product QA share one definition of done](../0027-coders-follow-gentle-ai-odd-and-rdd/intent.md), [#758](https://github.com/testeos-z/gabo-ai/issues/758) (the done contract and RDD evidence this intent measures against) |
| sibling intent, B | [0028 Deterministic hooks as locks](../0028-deterministic-hooks-as-locks/intent.md), [#765](https://github.com/testeos-z/gabo-ai/issues/765) (a trigger: hook changes) |
| sibling intent, C | [0029 Branch protection on main and dev](../0029-branch-protection-on-main-and-dev/intent.md), [#766](https://github.com/testeos-z/gabo-ai/issues/766) |
| related intents | [0013](../0013-several-engines-in-parallel-in-one-project/intent.md) (engines in parallel), [0019](../0019-talk-to-cloud-agent-sessions/intent.md) (cloud sessions), [0020](../0020-spend-or-turn-cap-for-auto-mode/intent.md) (the cap rule), [0022](../0022-measure-roles/intent.md) (measuring roles) |
| PRD | [PRD](../../PRD.md) (R9, R31) |
| TRD | [TRD](../../TRD.md) section 9.1 (methodology slot) |
| spec | none yet; the owner ordered the specs after the intents (relayed 2026-11-13) |
| plan | none yet |
| sources | [Anthropic, The AI-native SDLC playbook](https://claude.com/resources/articles/the-ai-native-sdlc-playbook), Louis Claxton, 2026-08-21 (quotes byte-checked 2026-11-13); `origin/dev` at `63b36df` for every repository path cited; `origin/dev` at `23552d5` for `gabo.yaml:52` and `:53`; the reviewer's `merge-head.sh:77-90` (user-level, outside the repository); the spec 0027 evidence base for the 13 coder runs (assistant report, not re-checked) |
