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
