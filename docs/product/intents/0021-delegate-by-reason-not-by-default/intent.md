# Intent 0021 Delegate by reason, not by default

| field | value |
|---|---|
| intent id | 0021-delegate-by-reason-not-by-default |
| status | Specified |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-11 |

## Problem

The gabo assistant delegates by default: its skill describes it as "The owner's single point of contact; delegates all heavy work" (`.gabo/skills/gabo-assistant/SKILL.md`). Each delegation costs a handoff and a fresh context, yet handing the same task to another agent does not by itself give a better result.

Source of the idea: the assistant's analysis, approved as an intent by the owner on 2026-11-11 ("si" to creating an intent for each pending idea). Evidence: "Handing the same task to a subagent doesn't make the answer any better" (https://code.claude.com/docs/en/best-practices).

## Who is affected

The owner, in every session and wave: delegation adds time and tokens to each step the assistant hands off.

## Desired outcome

- The assistant hands work to another agent only when a named reason holds, and says which reason.
- Work without such a reason is done by the agent that already holds the context.

## Constraints

- Independence that the method needs stays: the plan reviewer runs in a fresh session (TRD D8), and each coder works in its own worktree within `parallelism`.
- Reviewer independence is out of scope here; it waits for the owner's decision on intent 0015 S3.
- The owner still talks to one point of contact.

## Assistant's analysis (input for the spec, not confirmed by the owner)

- The assistant skill says it "delegates all heavy work"; the change replaces that default with a list of reasons to delegate.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which reasons justify a delegation (for example parallel work, an independent context, a large exploration)? | answered: independent units with disjoint edit surfaces, a map that needs more than one bounded read batch, a long suite or build that would block the thread, and an independent verifier or reviewer; plus role ownership (spec D2, S1, S3). Recorded decision (resolver: agent, rule of the owner). |
| 2 | Does the change apply only to the assistant, or also to directiva and the cycle? | answered: the assistant, and the cycle when the assistant runs it (spec D1). Recorded decision (resolver: agent, rule of the owner). |
| 3 | In what order does it land relative to intent 0015 (one subagent layer on every engine)? | answered: after 0015 task T4 (#479), as a cross-intent blocker (spec D5, S7). Recorded decision (resolver: agent, rule of the owner). |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#496 I0021: Delegate by reason, not by default](https://github.com/testeos-z/gabo-ai/issues/496) (sub-issue of #492) |
| TRD | [TRD](../../TRD.md), D8 |
| skill | `.gabo/skills/gabo-assistant/SKILL.md` |
| spec | [spec](spec.md) |
| plan | — |
| related intent | 0015-one-subagent-layer-on-every-engine; 0022-measure-roles |

## Review findings

Results of the probe (spec S8, plan T5 and T6, plan decision PD13). Source: the probe comment on [#630](https://github.com/testeos-z/gabo-ai/issues/630#issuecomment-6836069180), Claude Code 2.1.302 (`claude-sonnet-5-5`), run on `testeos-z/gabo-repo`. The full tables are in the plan's Probe results.

### Handoffs per run

| run | text | handoffs (first line) |
|---|---|---|
| old | `a2e99a6` (before T1, PR #642) | 3 `gabo-coder` (`wave id: I0001`), 1 `gabo-pm` (spec of I0002): 4, no reason named |
| new | `d6e7e77` (merge of T3, PR #644) | 3 `gabo-coder` (`Reason: (a) parallel units`), 1 `gabo-pm` (`Reason: role ownership`): 4; the inline README request launched no agent |

Verdicts: S1 PASS on the observed clauses (the clauses (b), (c) and independent units outside the cycle are `not run`); S2 PASS; S3 PASS (the plan reviewer with the owner absent is `not run`); S4 PASS for a three-task wave (a wave of one task is `not run`); S8 PASS. Both waves ended 3 done, 0 blocked. One sample per text: the 13% lower token spend on the new text is not a measure.

### Deviations and decisions

- **S5, pinned string (T1, PR #642).** S5 says the pinned strings of the skill tests stay. T1 replaced the one pinned at `assistant_test.go:223` ("One exception: you run the `gabo-cycle` steps yourself; exploring, writing documents and writing code still go to subagents") with the new exception sentence, because that string is the old default and contradicts S1 (plan PD5). The spec is not edited. Decided by the owner's standing rule; if the owner prefers the spec amended, that is a plan-change proposal.
- **T2 role pins (#627, PR #643).** The plan asked the handoff sentence to name no roles, but `agenttemplates_test.go` pins the role list in both files and is outside T2's edit surfaces. The pins stay unchanged; the roles sit in a separate role-ownership line and the handoff sentence names none. Decided by the owner's standing rule.

### Probe findings

1. **Advisory, S2, owner report.** The wave's final report to the owner does not say which reason each coder had; the assistant names it only when asked "why". S2 asks it to name the reason when it tells the owner what it handed off, so the clause is met by the "why" answer, not by the wave report. Decided by the owner's standing rule (recommended option): recorded as an advisory for the follow-up intent, no new task here. If the owner wants the wave report to say "coders launched under (a)", it is a small text change in `gabo-wave-report` (a new task).
2. **S4 (a) for T003.** The dependent task T003 alone also carries `Reason: (a) parallel units`, as PD7 specifies ("a wave of one task included"). It is by the plan, not a deviation; (a) reads oddly for a single dependent unit.
3. **Probe artifacts, not defects.** `gabo init` writes plugin and marketplace keys into `.claude/settings.json` that load the published plugin text; they were removed on purpose so the catalog under test is the only text (as in #581). The run wrote `~/.config/gabo/config.yaml` both times; it was removed after each run.
