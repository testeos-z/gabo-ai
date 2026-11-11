# Intent 0020 Spend or turn cap for auto mode

| field | value |
|---|---|
| intent id | 0020-spend-or-turn-cap-for-auto-mode |
| status | Draft (spec written when the owner picks this intent) |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-11 |

## Problem

In auto mode an unattended wave has no cap on turns or spend. It is bounded only by `parallelism` and by the fix-round limit: a coder that is still BLOCKED after 4 fix rounds leaves its task BLOCKED (`gabo-cycle` rule 3). The owner cannot say "stop this wave at this much".

Source of the idea: the assistant's analysis, approved as an intent by the owner on 2026-11-11 ("si" to creating an intent for each pending idea). Evidence: "Give it a cap on turns or spend, and have a person approve the result at the end" (https://code.claude.com/docs/en/best-practices).

## Who is affected

The owner, on every wave run in auto mode without the owner present.

## Desired outcome

- An auto-mode wave stops at a cap the owner sets, and the wave report says that the cap stopped it.
- A person approves the result at the end of a capped wave.

## Constraints

- The cap must be measurable on the engines gabo supports; an engine that reports neither turns nor spend needs a defined fallback.
- Today `review.auto_merge` lets the reviewer merge clean heads into `dev` in auto mode; "a person approves the result at the end" must be reconciled with it (open question 4).
- The fix-round limit and `parallelism` stay.
- No secrets (PRD R21): a spend figure that needs a billing credential is read through an env var name only.

## Assistant's analysis (input for the spec, not confirmed by the owner)

- Today an unattended wave is bounded only by fix rounds and parallelism; the cap adds a limit per wave that the owner sets in `gabo.yaml`.

## Open questions

| # | question | status |
|---|---|---|
| 1 | What is capped: turns, spend, wall-clock time, or several? | open (owner) |
| 2 | Which engines report turns or spend to a running session? | carried to spec (needs sources per engine) |
| 3 | At the cap, does the wave stop and report, or pause and ask the owner? | open (owner) |
| 4 | Does "a person approves the result at the end" change `review.auto_merge` for capped waves? | open (owner) |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#495 I0020: Spend or turn cap for auto mode](https://github.com/testeos-z/gabo-ai/issues/495) (sub-issue of #492) |
| PRD | [PRD](../../PRD.md), R21 |
| skill | `.gabo/skills/gabo-cycle/SKILL.md` (rule 3, `parallelism`) |
| spec | — (written when the owner picks this intent) |
| plan | — |
| related intent | 0022-measure-roles (records what a wave spends) |
