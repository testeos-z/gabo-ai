# Intent 0026 Pi probe follow-ups

| field | value |
|---|---|
| intent id | 0026-pi-probe-follow-ups |
| status | Draft |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-13 |

## Problem

Intent 0023 (unattended Pi waves and flow hygiene) closed with its tasks done. Its probes on Pi 1.5.2 with gentle-pi 4.7.0 left eight open questions, which the owner moved here from [#700](https://github.com/testeos-z/gabo-ai/issues/700) so that the accepted spec of 0023 is not reopened (owner's standing rule, [#700 comment](https://github.com/testeos-z/gabo-ai/issues/700#issuecomment-6853940948), 2026-11-13).

The probes found that an unattended Pi wave does not do what the 0023 spec says, in five places:

- **Session mode.** A Pi session cannot read its own mode under `--print`, `--mode rpc` or interactive, so the warning of 0023 S1 prints in every mode, where 0023 S1 says rpc and interactive print no line ([#585 comment](https://github.com/testeos-z/gabo-ai/issues/585#issuecomment-6840556586), Pi 1.5.2, gentle-pi 4.5.x).
- **Coder model.** The built Pi `gabo-coder` has no model, so gentle-pi used a default model that the account rejects. The coder launch failed until the probe pinned another model in its scratch agent file.
- **Approval gate.** gentle-pi's `branch-pr` skill requires a `status:approved` label before it opens a PR. One task stopped before its push on that label, and the other task's close was blocked by it.
- **Remote delete.** With `autonomousMode` true, gentle-pi deleted a merged remote branch with no confirmation, where 0023 S9 says gentle-pi asks even in autonomous mode.
- **Grants and waits.** gentle-pi 4.7.0 has no project-level grant key, so the outside paths of a wave could only be granted in the session prompt, and 0023 S4 does not name the wave's stamps folder. A Pi session also emitted `agent_settled` while a background coder return was still pending.

The wave probe ran with gabo built from dev `d8dcaea` (stamped 0.2.10) on testeos-z/gabo-repo, not on a disposable repository, by the owner's decision of 2026-11-12 on [#585](https://github.com/testeos-z/gabo-ai/issues/585#issuecomment-6831839550). Its results, recorded on 2026-11-13, are in the [#599 comment](https://github.com/testeos-z/gabo-ai/issues/599#issuecomment-6841235718). Several 0023 checks were never observed: the grant dialog (no dialog reached the driver), the status labels of closed tasks (no task reached Task Close) and a denied remote delete (none was denied).

The intent grew by one finding that is not about Pi. The reviewer state of I0025 T2 (PR [#720](https://github.com/testeos-z/gabo-ai/pull/720), findings in [#705](https://github.com/testeos-z/gabo-ai/issues/705)) is still keyed by PR number in its mirror, its PR refs and its base checkout, so two repositories that share one reviewer state dir are not isolated, and two reviewer comments still name the old `pr-<n>` layout. The owner's standing rule on [#695](https://github.com/testeos-z/gabo-ai/issues/695#issuecomment-6855499990) (2026-11-13) sends these findings here before the owner accepts this spec, not into accepted spec 0025.

## Who is affected

The owner, who starts Pi waves and expects them to reach the merge without a question that nobody can answer; the team in `mode: auto`, which has no person at each gate; every Pi wave, because the mode, model, gate, delete and grant behaviors apply to every task of every Pi wave; and the owner when two repositories share one reviewer on one machine.

## Desired outcome

- Each of the eight questions of [#700](https://github.com/testeos-z/gabo-ai/issues/700) has either an answer in the spec or an open decision for the owner with a recommended option.
- A Pi wave stops at a gate only when gabo owns that gate. gentle-pi's default model, its approval label and its remote delete never stop a wave silently.
- Two repositories that share one reviewer state dir never share a mirror, a PR ref or a base checkout, and no reviewer script comment or test description names the old `pr-<n>` layout (S11, S12).
- Each 0023 check that did not pass, or was not observed, is either observed on a probe or recorded as a known limit with its owner decision. A check not run reads `not run: <reason>`, never a pass.
- The owner can tell from a wave report which Pi facts were observed, with the Pi and gentle-pi versions and the gabo build under test.

## Constraints

- Accepted specs 0023 and 0025 are not reopened. Each change to their behavior is written in the spec of this intent and takes effect only when the owner accepts it (owner's standing rule, #700 comment).
- gabo writes no permission and no user engine config. A Pi wave's grants are the owner's (intent 0004 S8 and D7; PRD R32; 0023 S4).
- Delete only what the workflow created (PRD R23). No secret goes into a trace, a probe comment or a report (PRD R21).
- gabo never creates a label to pass a gate. A `status:approved` label added to fake an approval is not a fix.
- The CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt` (PRD R31; TRD D24). The owner's preference (2026-11-12) is to keep custom scripts to a minimum and to prefer engine-native features.
- The probes run on testeos-z/gabo-repo, by the owner's decision on #585. Its `dev` is reset to seed `73ec699` before and after each run, and runs are coordinated so that only one uses the repository at a time.
- The reviewer changes of S11 and S12 widen S10 of the spec for the reviewer's scripts they name only. No other S10 limit changes.

## Open questions

| # | question | status |
|---|---|---|
| 1 | How does a Pi session read its own mode, or does the warning stay in every mode? | carried to spec (S1) |
| 2 | Should `gabo-runtime-pi` pin a coder model? | carried to spec (S2) |
| 3 | Does gentle-pi's `status:approved` gate apply to gabo coders? | carried to spec (S3) |
| 4 | Which remote branch delete is right in autonomous mode? | carried to spec (S4) |
| 5 | Where do the grants of an unattended Pi wave go, and which paths do they name? | carried to spec (S5) |
| 6 | What ends a Pi wave when `agent_settled` arrives while a background coder is pending? | carried to spec (S6) |
| 7 | Is the user-level reviewer copy that a Pi wave runs current? | carried to spec (S7) |
| 8 | Is the reviewer folder grant asked at most once per path? | carried to spec (S8, S9) |
| 9 | What happens to an existing unqualified reviewer mirror, its PR refs and its base checkout after the change? | carried to spec (S11, spec question 9). **Proposed (auto)**: leave them in place and create the qualified ones beside them. Owner decides. |
| 10 | Does the title still fit the intent? | **Proposed (auto)**: keep the title for this PR and retitle at acceptance to "Pi probe follow-ups and reviewer state isolation", because S11 and S12 are not Pi facts. The folder keeps its name; no rename here. Owner decides. |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#703 I0026: Pi probe follow-ups](https://github.com/testeos-z/gabo-ai/issues/703) (sub-issue of #492) |
| follow-up issue | [#700 Follow-up: open questions from the I0023 Pi probes](https://github.com/testeos-z/gabo-ai/issues/700) (sub-issue of #703) |
| sources | [#585](https://github.com/testeos-z/gabo-ai/issues/585) (probe T1, comment [6840556586](https://github.com/testeos-z/gabo-ai/issues/585#issuecomment-6840556586)); [#599](https://github.com/testeos-z/gabo-ai/issues/599) (probe T15, comment [6841235718](https://github.com/testeos-z/gabo-ai/issues/599#issuecomment-6841235718)); `docs/product/intents/0023-unattended-pi-waves-and-flow-hygiene/intent.md` (`## Review findings`) and `plan.md` (`## Probe results`) |
| reviewer findings | [#705](https://github.com/testeos-z/gabo-ai/issues/705) (I0025 T2 findings, PR [#720](https://github.com/testeos-z/gabo-ai/pull/720)); owner's standing rule, [#695 comment](https://github.com/testeos-z/gabo-ai/issues/695#issuecomment-6855499990); open reviewer task [#706](https://github.com/testeos-z/gabo-ai/issues/706) (I0025 T3) |
| related intent | 0023-unattended-pi-waves-and-flow-hygiene (closed; its spec is accepted and not reopened); 0016-gabo-init-pi-subagent-package-check-and-reviewer-content-check (reviewer content hash, S7); 0025-wave-record-and-reviewer-follow-ups (same skill files, accepted) |
| PRD | [PRD](../../PRD.md), R21, R23, R31, R32 |
| TRD | [TRD](../../TRD.md), D7, D24 |
| spec | [spec](spec.md) |
| plan | — (written only after the spec is accepted) |
