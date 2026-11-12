# Intent 0024 Mechanical wave steps and follow-up fixes

| field | value |
|---|---|
| intent id | 0024-mechanical-wave-steps-and-follow-up-fixes |
| status | Specified |
| owner | gabo-pm |
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| date | 2026-11-12 |

## Problem

The probes of intents 0020, 0021 and 0022 and the reviewer left twelve findings that no intent covers. Owner request, 2026-11-12: "si", to writing one intent that groups them. Each finding is lettered (a) to (l) and quotes its evidence.

**From intent 0020 (spend or turn cap), [#495](https://github.com/testeos-z/gabo-ai/issues/495), probe [#581](https://github.com/testeos-z/gabo-ai/issues/581), `docs/product/intents/0020-spend-or-turn-cap-for-auto-mode/plan.md` ("Review findings of the probe", items 3 to 6):**

- (a) **Claude Code printed `enforced by the cycle` for `max_fix_rounds`**, while `gabo-wave-report` says `advisory` (the coder enforces it). The assistant composed its own wording; the cap itself was honored.
- (b) **The OpenCode rebuild rewrote the working-tree `gabo.yaml`** (it removed `wave.max_coder_turns: 3` while writing `steps: 3`). A runtime skill must not touch `gabo.yaml`.
- (c) **`STOPPED at fix round limit` with a clean head is finished, not paused.** The cycle re-reads the review, sees clean and finishes the task. The plan calls it reasonable behavior and raised it for a follow-up intent.
- (d) **Codex paused answers do not literally ask "continue or stop"**, although `gabo-assistant` Wave Close step 4 says to ask it; "continue" still resumed the wave.

**From intent 0022 (measure roles), [#497](https://github.com/testeos-z/gabo-ai/issues/497), probe [#637](https://github.com/testeos-z/gabo-ai/issues/637), `docs/product/intents/0022-measure-roles/plan.md` (PD19, PD20, "Review findings") and the wave report on #497 ([comment](https://github.com/testeos-z/gabo-ai/issues/497#issuecomment-6837225872), follow-up items 1 to 6):**

- (e) **No `session started` trace lines from the launch step**, in the third run ([#637 comment 6837003143](https://github.com/testeos-z/gabo-ai/issues/637#issuecomment-6837003143)) although the T8 text asks for them.
- (f) **`Measures` wall time is estimated** (`~3 min`), not computed from trace stamps.
- (g) **Only the report save is read back**; the other saves are listed as unverified.
- (h) **The `gabo-pm` and `gabo-dev-lead` return contracts list no proposals**, while the planning record takes `proposals: <n>` from the directiva's return.
- (i) **No test covers `.claude/skills/gabo-memory-engram`** (the live memory adapter, [#635](https://github.com/testeos-z/gabo-ai/issues/635)); its Records table is checked only by readback.
- (j) **Two wording nits in TRD 9.2**: a stray `.;` after "each task's outcome" and "section 9" in the `Measures` clause, which does not say which document's section 9.

**From intent 0021 (delegate by reason), [#630](https://github.com/testeos-z/gabo-ai/issues/630), `docs/product/intents/0021-delegate-by-reason-not-by-default/intent.md` ("Review findings", Probe findings 1):**

- (k) **The final wave report does not name each coder's reason**; the assistant names it only when asked "why".

**From the reviewer (user-level `gabo-reviewer` skill), measured 2026-11-12:**

- (l) **The reviewer's state grows without bound.** `~/.local/state/gabo/reviewer/worktrees` holds one `pr-<n>` worktree per reviewed PR and nothing removes them (349 of them, 4.0 G on 2026-11-12, `review-head.sh` creates `pr-<n>` and reuses it, no script deletes it); the reviewer's Go build cache and `check-home` also only grow (reported 4.7 G for the build cache, measured 280 M for `cache/go-build` and 1.3 G for `check-home` when this intent was written, so the size moves). On 2026-11-12 `/data` reached 38 G of 46 G.

**Common cause behind (a), (e), (f) and (g).** In each of these a headless assistant paraphrased skill text instead of following it: it wrote its own enforcement label (a), skipped the trace line (e), wrote `~3 min` instead of computing it (f), and read back one save instead of all (g). Intent 0022 already tried more skill text twice, and both adjustments (T7, PD19; T8, PD20) left (e), (f) and (g) failing in the third run.

## Who is affected

The owner, who reads the wave report and cannot trust a label, a time or a "none unverified" line that the assistant composed; and whoever runs the reviewer on a machine, because the disk fills slowly until a run fails. (a) to (k) affect every wave on every engine; (l) affects every reviewed PR.

## Desired outcome

- The wave report's labels, times and trace lines come from a fixed mechanical step, not from the assistant's paraphrase, and a paused report ends with the words "continue or stop".
- A rebuild never changes `gabo.yaml` without the owner seeing it; a clean `STOPPED at fix round limit` is stated as finished; the directiva returns list proposals; the live memory adapter has a test; TRD 9.2 reads cleanly; the final wave report names each coder's reason.
- The reviewer's state stays bounded: no `pr-<n>` worktree outlives its PR, and its caches stay under a stated size.

## Constraints

- The CLI keeps `gabo init`, `gabo check`, `gabo version` and `gabo prompt` (PRD R31; TRD D24); a new command is the owner's change, not an agent's.
- `gabo.yaml` and `.claude/` are human-only paths for agents (the plan decisions of 0022 PD17); no requirement edits them.
- The reviewer deletes only what it created (PRD R23) and its state is disposable (TRD 9.4, reviewer state row).
- No secrets in any script or trace line (PRD R21).
- Same files as intents 0020, 0021, 0022 and 0023 (the cycle, the assistant skill, the wave report, the coder skill): order the plan tasks after theirs where they edit the same text.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Would a mechanical step (a script or a CLI command) fix the paraphrase cause better than more skill text? | decided by the owner's standing rule (2026-11-12): a script in the catalog skill that the skill tells the agent to run and paste, not a CLI command and not more text (spec D1). |
| 2 | Does intent 0023 (#575) already cover any of the twelve findings? | answered: no. Checked against its spec on `origin/dev`: 0023 covers Pi sessions, the sandbox, `git push` confirmation, status labels, the `Closes`/`Refs` keyword and remote branches; none of (a) to (l). It edits the same skill files, so the plan orders the tasks (spec section 5). |
| 3 | One intent for twelve findings? | decided by the owner's standing rule (2026-11-12): one intent, as the owner asked; the spec keeps each finding in its own requirement so a plan can split them (spec D2). |
| 4 | The limits for (l): how many PR worktrees to keep and how large a cache. | decided by the owner's standing rule (2026-11-12): carried to the spec as Assumption values the owner sets (spec D6). |

## Links

| kind | reference |
|---|---|
| product card | [#492 Flow improvements](https://github.com/testeos-z/gabo-ai/issues/492) |
| intent issue | [#664 I0024: Mechanical wave steps and follow-up fixes](https://github.com/testeos-z/gabo-ai/issues/664) (sub-issue of #492) |
| sources | I0020 [#495](https://github.com/testeos-z/gabo-ai/issues/495), [#581](https://github.com/testeos-z/gabo-ai/issues/581); I0022 [#497](https://github.com/testeos-z/gabo-ai/issues/497), [#637](https://github.com/testeos-z/gabo-ai/issues/637), [#635](https://github.com/testeos-z/gabo-ai/issues/635); I0021 [#630](https://github.com/testeos-z/gabo-ai/issues/630) |
| PRD | [PRD](../../PRD.md), R13, R21, R23, R29, R31 |
| TRD | [TRD](../../TRD.md), 9.2, 9.4, D17, D23, D24 |
| skill | `.gabo/skills/gabo-wave-report/SKILL.md`, `.gabo/skills/gabo-cycle/SKILL.md`, `.gabo/skills/gabo-assistant/SKILL.md`, `.gabo/skills/gabo-coder/SKILL.md`, `.gabo/skills/gabo-pm/SKILL.md`, `.gabo/skills/gabo-dev-lead/SKILL.md`, `.gabo/skills/gabo-reviewer/SKILL.md` |
| code | `apps/cli/internal/skills/gabo-reviewer/scripts/` |
| spec | [spec](spec.md) |
| plan | — |
| related intent | 0020-spend-or-turn-cap-for-auto-mode; 0021-delegate-by-reason-not-by-default; 0022-measure-roles; 0023-unattended-pi-waves-and-flow-hygiene (open, same skill files, no overlap in findings) |
