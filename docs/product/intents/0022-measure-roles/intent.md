# Intent 0022 Measure roles

| field | value |
|---|---|
| intent id | 0022-measure-roles |
| status | Draft (spec written when the owner picks this intent) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-11 |

## Problem

gabo splits the work into many roles (assistant, pm, dev-lead, frontend, db-master, plan reviewer, cycle, coder, reviewer), but nothing measures what each role costs or adds. The wave trace records each task's status, PR, last verdict, fix rounds and open questions (`gabo-cycle`, D17); it records no token count and no time per role. So the owner cannot tell whether a role still earns its place.

Source of the idea: the assistant's analysis, approved as an intent by the owner on 2026-11-11 ("si" to creating an intent for each pending idea). Evidence: "every component in a harness encodes an assumption about what the model can't do on its own" (https://www.anthropic.com/engineering/harness-design-long-running-apps).

## Who is affected

The owner, when deciding how gabo should be shaped; it affects the cost and time of every wave.

## Desired outcome

- Each wave records, per role, the tokens spent, the fix rounds and the time taken.
- The owner decides from that data whether each role stays.

## Constraints

- Recording must not change how a wave runs, and stores no secrets (PRD R21).
- The data goes through the `memory` slot like the trace (D17).
- Changing or removing a role is a convention change, so the owner decides it.

## Assistant's analysis (input for the spec, not confirmed by the owner)

- Record per wave tokens, fix rounds and time; trial one wave with pm and dev-lead merged; decide from the data whether roles stay.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which engines report token counts to the session, and what is recorded when one does not? | carried to spec (needs sources per engine) |
| 2 | Does the owner approve a trial wave with pm and dev-lead merged, and on which intent? | open (owner) |
| 3 | What result keeps a role, and what result removes it? | open (owner) |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#497 I0022: Measure roles](https://github.com/testeos-z/gabo-ai/issues/497) (sub-issue of #492) |
| PRD | [PRD](../../PRD.md), R21 |
| skill | `.gabo/skills/gabo-cycle/SKILL.md` (trace, D17), `.gabo/skills/gabo-agent-templates/agents.yaml` (roles) |
| spec | — (written when the owner picks this intent) |
| plan | — |
| related intent | 0020-spend-or-turn-cap-for-auto-mode; 0021-delegate-by-reason-not-by-default |
