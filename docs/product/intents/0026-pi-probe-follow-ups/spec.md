# Spec 0026 Pi probe follow-ups

| field | value |
|---|---|
| intent | [0026-pi-probe-follow-ups](intent.md) |
| status | Draft |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-13 |

Mode is `auto`. Every requirement that needs an owner product decision is marked **Proposed (auto)** with its recommended option and a one-line reason, and its question in section 6 stays open until the owner decides. This spec decides nothing as accepted. Section 7 stays empty until the owner fills it: an agent never accepts a spec.

Evidence base (checked 2026-11-13 against `origin/dev` at `6f5b6c6`):

- **The eight questions.** [#700](https://github.com/testeos-z/gabo-ai/issues/700) (body, questions 1 to 8) and its comment [6853940948](https://github.com/testeos-z/gabo-ai/issues/700#issuecomment-6853940948) (owner's standing rule).
- **Pi facts, session mode.** [#585 comment 6840556586](https://github.com/testeos-z/gabo-ai/issues/585#issuecomment-6840556586): Pi 1.5.2, gentle-pi 4.5.x, `--model openai/gpt-6.1-terra`; the mode was unreadable under `--print`, `--mode rpc` and interactive. The Pi extension API exposes `ctx.mode` only to extension code; no gentle-pi tool the session can call returns it.
- **Pi facts, the wave.** [#599 comment 6841235718](https://github.com/testeos-z/gabo-ai/issues/599#issuecomment-6841235718): Pi 1.5.2, gentle-pi 4.7.0, gabo built from dev `d8dcaea` (stamped 0.2.10), on testeos-z/gabo-repo with dev reset to seed `73ec699`. Results: one merge of two tasks (PR #204); the other task stopped before its push on `status:approved`; the built `gabo-coder` has no model field; a merged remote branch was deleted with no confirmation while `autonomousMode` was true; no grant key in gentle-pi 4.7.0 project settings; `agent_settled` while a subagent wake was pending.
- **Earlier 0023 requirements these facts touch**: `docs/product/intents/0023-unattended-pi-waves-and-flow-hygiene/spec.md` S1 ("`--mode rpc` and interactive print no line"), S4 (three outside paths), S9 ("gentle-pi asks before deleting a remote branch even in autonomous mode"), and decision D4 (turning off gentle-pi's push confirmation for `mode: auto`).
- **The reviewer copy (question 7).** Commit `13991b9` (2026-11-11, "refresh the user-level reviewer when its content differs at the same version") is contained in `d8dcaea`. It implements intent 0016 S4, accepted 2026-11-11. The probe's cleanup note says it left the user-level reviewer unchanged, so the probe's copy most likely did not get that refresh. This is inferred from the note, not measured. The Pi reviewer link resolves to gabo's user-level copy.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | Question 1, session mode. While a Pi session cannot read its own mode, the wave start of `gabo-assistant` shall print the warning of 0023 S1 once, before its claim, in every Pi mode (`--print`, `--mode rpc`, interactive), and shall not refuse the wave for a mode it cannot read. The refusal of 0023 S1 stays for a session that can read its mode. **Proposed (auto)**, open question 1: gabo adds no model-callable mode tool in this intent. | Given a Pi session in `--mode rpc` whose mode cannot be read, When the owner asks for a wave, Then the 0023 S1 warning prints once before the claim and the wave continues. Given the same in interactive mode, Then the same. Given a session that can read its mode, Then the 0023 S1 refusal line, unchanged. | skill review of the wave-start text in `gabo-assistant`; probe S9 (a) |
| S2 | Question 2, coder model. gabo shall not pin a model in the built Pi `gabo-coder` or in `gabo-runtime-pi`: a Pi coder takes the model of the owner's Pi configuration. When a coder launch fails because the account refuses its model, the task shall stop with the engine's error on its ledger line, and no other model shall be tried. **Proposed (auto)**, open question 2: no pin. | Given the built Pi `gabo-coder`, Then it has no `model` key. Given a coder launch refused for its model, Then the task's ledger line names the error, the wave records the task as stopped, and no launch uses another model. | pinning test in `apps/cli/internal/skills` for the built agent and the runtime text; probe S9 (c) |
| S3 | Question 3, approval gate. The `status:approved` gate of gentle-pi's `branch-pr` skill shall not stop a gabo Pi coder: the coder opens its PR without that label, and gabo shall never create, add or require it. The review gate stays the reviewer's (PRD and TRD review rules). **Proposed (auto)**, open question 3: the gate does not apply to gabo coders; the plan finds how gentle-pi's gate is switched off (its own setting or skill text), never by a label. | Given a Pi coder whose task passed its checks, Then it opens its PR with no `status:approved` label present and stops on no approval gate. Given the diff, Then no gabo file creates, adds or requires `status:approved`. | grep test over the coder and runtime skills and their copies; probe S9 (d) |
| S4 | Question 4, remote delete. In `mode: auto`, the Task Close step of gabo deletes the remote branch of a merged task that the workflow created, after the merge (the repository-host `delete branch` operation), and a Pi session shall not delete any other remote branch. A delete that the engine refuses or asks for, with nobody to answer, is a leftover under 0023 S9 with its command. **Proposed (auto)**, open question 4: a merged task branch created by the workflow may be deleted under autonomous mode; the 0023 S9 sentence that gentle-pi asks even in autonomous mode is replaced by the observed behavior once probe S9 (e) confirms it. | Given a merged task branch the workflow created, When Task Close runs in `mode: auto`, Then the delete is that step and the report shows it done, or a leftover with the command. Given a delete of a branch the workflow did not create, Then the coder text forbids it. Given a refused delete, Then the 0023 S9 leftover and the issue still closes. | skill review; pinning test of the Task Close text; probe S9 (e) |
| S5 | Question 5, grants. The grant notes of `gabo-runtime-pi` and the Pi section of `docs/compatibility.md` (0023 S4) shall name four outside paths: the worktree root, the user file `${XDG_CONFIG_HOME:-~/.config}/gabo/config.yaml` (which must exist before a wave; the probe wrote a placeholder in scratch), the user-level reviewer folder with its state, and the wave's stamps folder `${XDG_STATE_HOME:-~/.local/state}/gabo/waves`. The grants stay in the session prompt, and gabo writes no grant key until gentle-pi has a project-level one. **Proposed (auto)**, open question 5: the list grows by the stamps folder, and the grant key is an external item for gentle-pi, not a gabo change. | Given the two documents, Then each lists the four paths, says the grant is the owner's and that gabo writes no permission to the user's engine config. Given the diff, Then no gabo code writes a file under the user's home for a grant. | document review; grep test for the four paths; probe S9 (b) |
| S6 | Question 6, end of a Pi wave. The end of a wave is the wave report entry in its trace, never an engine event: a Pi session that emits `agent_settled` while a background coder return is pending has not ended the wave. `gabo-cycle` shall say so, and the cycle shall keep waiting for the coder's return or the claim's recorded state. A driver that stops at `agent_settled` is outside gabo's catalog and is reported as a finding; this spec does not fix it. **Proposed (auto)**, open question 6. | Given the `gabo-cycle` text, Then it names the report entry as the end of a wave and not an engine event. Given an `agent_settled` with a coder return pending, Then the cycle does not report the wave finished. | skill review; pinning test of the cycle text; probe S9 (a) |
| S7 | Question 7, reviewer copy. A Pi wave runs the user-level reviewer through its link, which resolves to gabo's user-level copy. The probe left the user-level copy unchanged, so it most likely predates the content refresh of `gabo init` (commit `13991b9`, intent 0016 S4) and differs from the embedded reviewer while its `VERSION` is equal. This is inferred from the probe's cleanup note and is not measured; the probe checks it (S9 (f)). No new check in gabo: the refresh is the existing `gabo init` behavior. **Proposed (auto)**, open question 7: the probe records the stamped `REVIEWER_HASH` and the embedded hash before its first review, runs `gabo init` once with the build under test, and records both again. | Given a probe host, Then the probe records both hashes before its first review. Given they differ, Then one `gabo init` with the build under test runs, its output names `reviewer skill refreshed`, and the hashes are recorded equal afterwards. Given the owner declines that run, Then the probe reports the stale copy as a known limit. | probe S9 (f); 0016 S4 tests stay as they are |
| S8 | Question 8, grant dialog. Whether the reviewer folder grant is asked at most once per path is a fact to observe, not a product rule. The probe records it only from a session where a grant dialog reaches the driver. | Given the probe, Then the result is either observed (the dialog and its count per path) or `not observed: <reason>`. A run without a dialog is never recorded as a pass. | probe S9 (b) |
| S9 | Probe. The changes of S1 to S8 that a probe can check shall be probed with the build under test first on `PATH`: the probe records `command -v gabo`, `gabo version`, the build's commit, Pi and gentle-pi versions, and the model used, and reports `not run: wrong build on PATH` when `gabo` is not that build. Probes: (a) a Pi wave under `--mode rpc` with two tasks: S1 warning once, S6 end of wave; (b) S5 grant paths and S8 dialog; (c) S2 model, the built agent has no `model` key; (d) S3 PR without the label, and the two closed tasks carry no `status:in-progress` label (0023 status-label check); (e) S4 remote delete of a merged task branch and the leftover path of 0023 S9; (f) S7 hashes. The probe runs on testeos-z/gabo-repo, dev reset to seed `73ec699` before and after, runs coordinated. Each result is a comment on the intent's issue (#703), read back. A `not run: <reason>` is reported, never invented; a failed probe is one finding and one adjustment task, not a loop. | Given each probe, Then its result is a comment on #703, read back, with the versions, the model, `gabo version` and the commit. Given a check not run, Then the comment says `not run: <reason>`. | probe |
| S10 | Scope. This intent adds no script, no CLI command, no `gabo.yaml` key and no Go production code. It changes skill text and its copies (`apps/cli/internal/skills/`, `plugins/codex/skills/`), the pinning tests of that text, and the Pi section of `docs/compatibility.md`. | Given the diff, Then no new file under any `scripts/` folder, no `.go` file outside `_test.go` is changed, no CLI flag or `gabo.yaml` key is added, and `gabo check` exits 0. | review of the diff; `go -C apps/cli build ./...`, `vet`, `test ./...` |

## 2. Design

The change is text and probes; it adds no mechanism. Each requirement names the skill or document that carries its text. The plan lists the files.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | One intent or one per question | (a) one intent for the eight questions; (b) one intent per question | (a), by the owner's standing rule on [#700](https://github.com/testeos-z/gabo-ai/issues/700#issuecomment-6853940948) (the questions go to one follow-up intent 0026 with its own spec). | The requirements stay separable by question (S1 to S8), so the plan may split them. |
| D2 | Accepted specs 0023 and 0025 | (a) reopen 0023 by editing its spec; (b) keep both specs as accepted and record each change to their behavior here | (b), by the owner's standing rule that accepted specs are not reopened. | The changes to 0023 S1, S4 and S9 take effect only when the owner accepts this spec's section 6. |
| D3 | Host for the probes | (a) a new disposable repository; (b) testeos-z/gabo-repo, with dev reset before and after each run | (b), by the owner's decision on [#585](https://github.com/testeos-z/gabo-ai/issues/585) (2026-11-12). | Runs are coordinated; no two runs share the repository; no probe deletes the repository. |
| D4 | What counts as evidence for a Pi fact | (a) the probe's own comment; (b) a claim from a session's reading of a skill | (a): each fact cites the probe comment or the commit that shows it. | A fact not in a probe comment is an assumption and is marked as one. |

## 3. UX and app flow (optional)

Not applicable: no screen and no component. The owner sees three text outputs change, and their wording is in S1, S5 and S6: the warning line of the wave start, the grant notes and the report's wave-end entry.

## 4. DB migration steps (optional)

Not applicable: the change stores no product data. The probes write to the disposable gabo-repo's issues and branches under the owner's decision on #585, and the wave writes its usual trace and ledger entries.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| 0023 S1 ("`--mode rpc` and interactive print no line") | The mode is unreadable in every probed mode ([#585](https://github.com/testeos-z/gabo-ai/issues/585#issuecomment-6840556586)). | resolved by S1 (Proposed (auto)); the accepted 0023 text is not edited by this PR. |
| 0023 S4 (three outside paths) | The wave also writes the stamps folder. | resolved by S5 (Proposed (auto)). |
| 0023 S9 ("gentle-pi asks before deleting a remote branch even in autonomous mode") | A merged remote branch was deleted with no confirmation ([#599](https://github.com/testeos-z/gabo-ai/issues/599#issuecomment-6841235718)). | resolved by S4 (Proposed (auto)); 0023 D4 already accepts autonomy for `mode: auto`. |
| PRD R23 (delete only what the workflow created) | S4 names the delete of a merged task branch in `mode: auto`. | none: the branch is created by the workflow and merged; any other branch is forbidden by S4. |
| PRD R32 and intent 0004 S8 and D7 (gabo writes no user engine config) | S2 and S5 touch the model and the grants. | none: S2 pins no model in any user file; S5 writes no grant key. |
| Intent 0016 S4 (user-level reviewer refresh by content hash, accepted 2026-11-11, in `13991b9`) | Question 7. | none: S7 relies on it and adds no check. |
| Plan-change scope (TRD D7, the plan-change flow) | Changes to 0023's accepted behavior come after the owner accepts this spec. | the plan-change proposals, if any, are opened after acceptance through the plan-change flow; this PR changes only `docs/product`. |

## 6. Open questions

Each **Proposed (auto)** option is a recommendation. The owner decides it; the agent does not.

| # | question | status |
|---|---|---|
| 1 | How does a Pi session read its own mode, or does the S1 warning stay in every mode? | **Proposed (auto)**: the warning in every mode and no model-callable mode tool in this intent (S1). Reason: no probed means reads the mode, and a tool that returns `ctx.mode` needs a gentle-pi change outside gabo. Owner decides. |
| 2 | Should `gabo-runtime-pi` pin a coder model? | **Proposed (auto)**: no pin; the owner's Pi configuration chooses the model (S2). Reason: model access is per account, and a pin would make one account's choice the product's. Owner decides. |
| 3 | Does gentle-pi's `status:approved` gate apply to gabo coders? | **Proposed (auto)**: no; a gabo coder opens its PR without the label, and gabo never creates it (S3). Reason: the gabo coder contract has no such gate, and a label would fake an approval. Owner decides. |
| 4 | Which remote branch delete is right in autonomous mode? | **Proposed (auto)**: a merged task branch the workflow created may be deleted in `mode: auto`, and any refused delete is a leftover (S4). Reason: R23 limits deletes to branches the workflow created, and 0023 D4 already accepts autonomy. Owner decides. |
| 5 | Where do the grants of an unattended Pi wave go, and which paths do they name? | **Proposed (auto)**: four paths in the notes, grants kept in the session prompt (S5). Reason: gentle-pi 4.7.0 has no project grant key, and gabo writes no grant. Owner decides. |
| 6 | What ends a Pi wave when `agent_settled` arrives while a coder is pending? | **Proposed (auto)**: the wave report entry, never the engine event (S6). Reason: the event fired with a coder still pending in the probe ([#599](https://github.com/testeos-z/gabo-ai/issues/599#issuecomment-6841235718), question 6). Owner decides. |
| 7 | Is the user-level reviewer copy that a Pi wave runs current? | **Proposed (auto)**: the probe refreshes it once with the build under test and records both hashes (S7). Reason: the refresh already exists in `gabo init` (0016 S4), and a new check would duplicate it. Owner decides whether the run may happen. |
| 8 | Is the reviewer folder grant asked at most once per path? | answered by probe only: observed in S9 (b) or `not observed`. Not a product decision (S8). |

What needs the owner later, not now: accepting this spec (section 7), and the decisions of questions 1 to 7 (each a **Proposed (auto)** option). Nothing else needs the owner before the plan.

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
