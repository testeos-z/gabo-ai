# Intent 0023 Unattended Pi waves and flow hygiene

| field | value |
|---|---|
| intent id | 0023-unattended-pi-waves-and-flow-hygiene |
| status | Specified |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-12 |

## Problem

Four findings of the probe of an unattended Pi wave (P2a, [#414](https://github.com/testeos-z/gabo-ai/issues/414)) have no intent. They were split from [#463](https://github.com/testeos-z/gabo-ai/issues/463) on 2026-11-12 because the accepted specs of intents 0015 and 0016 do not cover them.

1. **A wave under `pi --print` dies with the claim held ([#562](https://github.com/testeos-z/gabo-ai/issues/562)).** `pi --print` ends after one turn and kills the background coder subagents, so the wave stops and leaves its intent claimed by a session that no longer exists. Nothing tells the owner to use a long-lived session such as `pi --mode rpc`, and `gabo-agent-host-local` starts a Pi `assistant` with `--print`.
2. **Pi waves stop at the sandbox ([#563](https://github.com/testeos-z/gabo-ai/issues/563)).** gentle-pi keeps the main session inside the project root and the worktrees registered with it. The task worktrees and the user-level reviewer folder are outside it, and the cycle registers no worktree, so the owner has to know which grants an unattended wave needs.
3. **Pi coders stop at `git push` ([#564](https://github.com/testeos-z/gabo-ai/issues/564)).** gentle-pi asks before `git push` unless its autonomous mode is on, and an unattended run cannot answer. `gabo init` writes no setting for it and `gabo check` does not look.
4. **The flow leaves litter behind ([#565](https://github.com/testeos-z/gabo-ai/issues/565)).** Closed tasks keep `status:in-progress` (92 closed issues of this repository carry it on 2026-11-12). Coder PR bodies say `Closes #<n>` against `dev`, where the keyword closes nothing, and the repository-host examples say `Refs #<n>`. On Pi, merged remote branches are left because gentle-pi asks before deleting them, and nobody can answer.

Each one is already written down as a known limit for the owner in `docs/compatibility.md` (Pi section, "Unattended Pi waves need three Pi settings") or in the probe comment; none is a gabo behavior yet.

## Who is affected

The owner, and the team in auto mode, who start a wave on Pi and expect it to run to the end without a person. Every wave on Pi is affected; the label and PR-body findings affect every wave on every engine.

## Desired outcome

- A wave on Pi never starts where it is certain to die, and when the session mode cannot be known the owner is told what to use.
- An unattended Pi wave reaches the push, the review and the merge without a question nobody can answer, and what the owner must still grant is named, not discovered.
- A closed task carries no status label, a PR body never claims to close a task the Task Close closes, and a branch gabo cannot delete is reported as a leftover with the command to run, never left silently.

## Constraints

- gabo writes only the project-level engine settings it needs to operate and never the user's global engine config (intent 0004 S8, D7; PRD R32). Permissions a person must give an engine stay the person's.
- gabo never works around an engine's guard (PRD R23: delete only what the workflow created; the guard is the engine's, not gabo's).
- The CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt` (PRD R31); agents do the rest with skills.
- Output of other engines stays as it is; the change adds no `gabo.yaml` key.
- Same files as intents 0015, 0020 and 0021 (the cycle, the assistant skill and the coder skill): order the plan tasks after theirs where they edit the same text.

## Open questions

| # | question | status |
|---|---|---|
| 1 | One intent for the four findings, or one per finding? The issues name three destinations: "Pi long-lived sessions and permissions" (#562, #563), a `gabo init` intent (#564) and a flow-hygiene intent (#565). | decided by the owner's standing rule (2026-11-12): one intent, as the owner asked; the spec keeps each finding in its own requirements so a plan can split them (spec D1). |
| 2 | Finding 4 of the #463 comment (the "no `subagent_*`" acceptance wording of 0015 T15) and the T15 text: in this intent? | decided by the owner's standing rule (2026-11-12): no. It belongs to intent 0015, which is closed; #565 records it for its owner. |
| 3 | How does a Pi session learn it runs under `--print`? | open, carried to the spec (S1, probe S11); the spec does not assume a mechanism. |
| 4 | Is turning off gentle-pi's `git push` confirmation for a project a decision the owner takes? (intent 0016 recorded it as a safety decision.) | decided by the owner's standing rule (2026-11-12): yes, taken by accepting this spec, and only for projects in `mode: auto` (spec D4). |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#575 I0023: Unattended Pi waves and flow hygiene](https://github.com/testeos-z/gabo-ai/issues/575) (sub-issue of #492) |
| findings (sub-issues of #575) | [#562](https://github.com/testeos-z/gabo-ai/issues/562), [#563](https://github.com/testeos-z/gabo-ai/issues/563), [#564](https://github.com/testeos-z/gabo-ai/issues/564), [#565](https://github.com/testeos-z/gabo-ai/issues/565) |
| source | [#463](https://github.com/testeos-z/gabo-ai/issues/463) (closed), probe P2a [#414](https://github.com/testeos-z/gabo-ai/issues/414#issuecomment-6703380366) |
| PRD | [PRD](../../PRD.md), R10, R13, R15, R23, R32 |
| skill | `.gabo/skills/gabo-assistant/SKILL.md`, `.gabo/skills/gabo-cycle/SKILL.md`, `.gabo/skills/gabo-coder/SKILL.md`, `.gabo/skills/gabo-runtime-pi/SKILL.md`, `.gabo/skills/gabo-agent-host-local/SKILL.md`, `.gabo/skills/gabo-contract-tasks/SKILL.md` |
| code | `apps/cli/internal/engineconfig/pi.go`, `apps/cli/internal/slices/check/` |
| docs | `docs/compatibility.md` (Pi) |
| spec | [spec](spec.md) |
| plan | — |
| related intent | 0013-several-engines-in-parallel-in-one-project (claim, worktree namespace); 0015-one-subagent-layer-on-every-engine (closed; Pi coders and the cycle on Pi); 0016-gabo-init-pi-subagent-package-check-and-reviewer-content-check (closed; it left these findings out); 0020-spend-or-turn-cap-for-auto-mode, 0021-delegate-by-reason-not-by-default (same skill files) |
