# Spec 0033 Coder worktree cleanup after merge

| field | value |
|---|---|
| intent | [0033-coder-worktree-cleanup-after-merge](intent.md) |
| status | Accepted |
| owner | gabo-pm |
| contributors | — (no screen and no stored data: sections 3 and 4 are not applicable) |
| date | 2026-11-26 |

Mode is `auto`. Every decision below carries a **Proposed (auto)** mark with a one-line reason unless it restates an owner statement or an existing hard rule. No agent accepts a spec: section 7 is the owner's. The owner accepted this spec for local planning only; no implementation or publication was authorized.

## 0. Terms and evidence

Terms used below:

- **Merged worktree:** a coder worktree whose PR merged into the integration branch (`dev` today) or its intent branch.
- **Real unsaved work:** tracked modifications, staged changes, unpushed commits, or any untracked or ignored content whose provenance/disposability is not proven under S2–S4 — regardless of author. A filename or glob match alone never proves disposability.
- **Ledger:** the wave/cycle record of which worktrees exist and who owns them; the cycle sweep today only covers its own `DONE`/`BLOCKED` entries.
- **Task Close:** the step that should close a finished coder worktree; it may never run when the coder is never resumed.

Issue evidence (from the task brief for #916, paths as reported):

- Retained merged worktrees blocked by: ignored verifier artifact `.atl/`, untracked `apps/storybook/storybook-static/`, orphan `*cwd-placeholder` files, empty wave folders.
- Orca external removals muddle the ledger.
- No cleanup owner for non-coder worktrees.
- Cycle sweep covers only its ledger `DONE`/`BLOCKED`; Task Close may not run when the coder is never resumed.

## 1. Requirements

| id | requirement | acceptance criteria | verification |
|---|---|---|---|
| S1 | When a coder worktree's PR has merged, the flow shall remove that worktree unless it holds real unsaved work. Any untracked or ignored content of unproven provenance counts as unsaved work regardless of author, and disposability is never inferred from a filename or glob alone. | Given a coder worktree whose PR merged and whose tree holds no real unsaved work and no content of unproven provenance, When the cleanup step runs, Then the worktree is removed. Given unpushed commits, staged changes, tracked modifications, or any untracked or ignored content whose provenance/disposability is not proven under S2–S4, Then the worktree is kept regardless of author and the reason names the item. Given a filename or glob match alone with no provenance proof, Then it never counts as disposable. | skill text test of the merged-only removal rule; probe on one merged worktree with and without unsaved work, including unknown-provenance fixtures |
| S2 | The cleanup gate shall not treat ignored verifier scratch as unsaved work when it is proven rebuildable; external verifier scratch shall be passed outside the worktree so it never blocks the gate. Proof is by rebuild provenance, never by path or glob alone. | Given a merged worktree containing only ignored `.atl/` proven rebuildable (regenerable by re-running the verifier, holding no user input, provenance proven — not by path alone), When the cleanup step runs, Then `.atl/` does not block removal. Given verifier scratch that cannot be proven rebuildable, Then the worktree is kept and the reason names it. | skill text test of the ignored-scratch rule; probe with rebuildable and non-rebuildable `.atl/` fixtures |
| S3 | The cleanup gate shall treat untracked generated output as safe-to-ignore only when it is provably generated with provenance proven (never by path or glob alone); otherwise it keeps the worktree. | Given a merged worktree whose only untracked tree is provably generated output with provenance proven (e.g. `apps/storybook/storybook-static/` reproducible by its build, holding no hand input — proven, not assumed from the path), When the cleanup step runs, Then it does not block removal. Given untracked output that is not provably generated, Then the worktree is kept regardless of path. | skill text test of the generated-output rule; probe with generated and hand-authored untracked fixtures |
| S4 | The cleanup gate shall treat a `*cwd-placeholder` file as non-blocking only when its flow ownership and orphaned provenance are proven (never by filename or glob alone), and a wave folder as removable only when it is actually empty and owned by the same wave; otherwise the worktree is kept. | Given a merged worktree whose only leftovers are `*cwd-placeholder` files with proven flow ownership and orphaned provenance plus wave folders proven actually empty and owned by the same wave, When the cleanup step runs, Then they do not block removal and are removed with the worktree. Given a filename/glob match alone with no ownership/provenance proof, a non-empty wave folder, or a folder not owned by the same wave, Then the worktree is kept and the reason names the item. | skill text test of the placeholder/empty-folder rule; probe with proven-orphan vs filename-only placeholders and empty-owned vs non-empty/unowned folders |
| S5 | When a ledger worktree path is missing, the sweep shall record it as missing with its provenance — proven external removal vs missing with unknown provenance — and never report a missing path of unknown provenance as a flow-owned cleanup or as a proven external removal. A missing path alone never proves which external actor acted. | Given a ledger entry whose worktree path is gone with no flow cleanup record and no proof of who removed it, When the sweep reconciles, Then the entry is marked missing with unknown provenance (kept as history, not rewritten) and reported separately from flow cleanups and from proven external removals. Given a removal proven external with evidence, Then the entry records it as proven external with the evidence cited. Given a flow-owned removal, Then the entry records the flow owner and reason. | skill text test of the missing-path record rule; probe with a missing path of unknown provenance and a proven-external fixture |
| S6 | Coder worktrees shall resolve to exactly one cleanup owner. Non-coder worktrees are manual for now (owner decision): the flow shall report them only and never auto-remove them. | Given a coder worktree under the flow's worktree roots, When ownership is resolved, Then exactly one owner is named (coder flow or the review cleanup, per D4). Given any non-coder worktree, Then it is reported as manual-cleanup and never auto-removed. Given a worktree with no resolvable owner, Then it is reported as unowned and never auto-removed. | ownership table read-back; skill text test that non-coder and unowned worktrees are reported, not removed |
| S7 | When Task Close never runs because the coder was never resumed, the same-wave owning sweep may evaluate that merged coder worktree under S1–S4 only after same-wave ledger and inactive-ownership checks pass; it shall never close the task nor bypass DONE/report controls. | Given a merged coder worktree with a same-wave ledger entry, proven inactive coder ownership, and no Task Close record, When the owning sweep runs, Then the worktree is evaluated under S1–S4 and removed unless real unsaved work exists, with the task state and report controls untouched. Given no same-wave ledger entry, an active owner, or an unmerged worktree, Then the fallback does not act. Given the fallback runs, Then it never marks the task DONE/BLOCKED, never closes the task, and never bypasses report controls. | skill text test of the sweep fallback trigger and its no-close/no-bypass guard; probe with same-wave/inactive vs other-wave/active fixtures and no Task Close record |

## 2. Design

The change tightens the existing cleanup gate without weakening its safeguards: the merged-only and real-unsaved-work rules stay absolute, and any content of unproven provenance is preserved regardless of author; the gate additionally recognizes only provenance-proven cases as non-blocking (rebuild-proven ignored scratch, provenance-proven generated output, ownership- and orphan-proven placeholders, actually empty same-wave-owned folders — never by filename or glob alone); verifier scratch moves outside the worktree; the ledger distinguishes flow cleanups from proven external removals and from missing paths of unknown provenance; coder worktrees get one owner while non-coder worktrees stay manual (report only, owner decision); and the same-wave owning sweep covers merged coder worktrees with inactive ownership even when Task Close never runs, without closing tasks or bypassing report controls.

Affected components (final paths are the plan's): `gabo-cycle` (same-wave sweep trigger, ownership table, no-close/no-bypass guard), `gabo-coder` (Task Close gate: S1–S4 checks with provenance proof), the verifier integration (external scratch location, `.atl/` rebuild proof), the ledger record (missing-path provenance marking, S5), and the reviewer placement only insofar as D4 keeps its worktrees out of auto-removal.

Every decision below is **Proposed (auto)**; the owner's acceptance confirms or changes each.

| id | decision | options considered | choice | consequences |
|---|---|---|---|---|
| D1 | Where verifier scratch lives | (a) pass external scratch to the verifier, outside the worktree; (b) keep scratch in the worktree and allowlist it at cleanup | (a). Reason: scratch never enters the gate, so `.atl/` cannot block removal again | The verifier integration names the external scratch path; worktree reads stay scratch-free |
| D2 | When ignored `.atl/` is recognized as rebuildable | (a) proven rebuildable only (regenerable by re-running the verifier, no user input inside); (b) any ignored path | (a). Reason: narrow rule keeps the never-lose-work guarantee | Non-proven ignored content still keeps the worktree (S2) |
| D3 | How generated output, placeholders, and empty folders are handled | (a) provenance-proven classes only (rebuild-/generation-proven output; ownership- and orphan-proven `*cwd-placeholder`; actually empty same-wave-owned folders), everything else keeps the worktree; (b) blanket untracked-is-clean | (a). Reason: blanket rules and filename/glob inference would delete hand-authored or unowned files | S3–S4 stay conservative: doubt keeps the worktree; filename match alone never suffices |
| D4 | Cleanup ownership, including non-coder worktrees | (a) coder worktrees owned by the cycle sweep fallback (S7), reviewer worktrees by the review cleanup, all non-coder worktrees manual for now — report only, never auto-remove (owner decision); unowned never auto-removed; (b) one global sweeper owns everything | (a), with the non-coder half as an owner decision. Reason: per-kind owners preserve the ledger-ownership safeguard; a global sweeper could delete what it does not own | S6 needs the ownership table plus the manual-report path for non-coder worktrees |
| D5 | How missing worktree paths are recorded | (a) ledger marks missing with provenance (proven external with evidence vs missing with unknown provenance), kept as history and reported separately, never rewrites the entry as a flow cleanup; a missing path alone never names an actor; (b) ledger deletes the entry silently | (a). Reason: silent deletion and actor inference are what muddle the ledger today | S5 keeps an auditable history without inventing an actor |
| D6 | Which sweep covers a merged coder worktree when Task Close never runs | (a) the same-wave owning sweep (cycle sweep for coder worktrees) evaluates S1–S4 on merged state after same-wave ledger and inactive-ownership checks, regardless of Task Close, and never closes the task or bypasses DONE/report controls; (b) only Task Close may remove | (a). Reason: Task Close is not guaranteed to run when the coder never resumes, and the fallback must not touch task state | S7 closes the gap without touching unmerged work, other-wave entries, active owners, or task/report controls |

## 3. UX and app flow (optional)

Not applicable: the change has no user-facing surface. The owner sees fewer stale coder worktrees on the server and ledger entries that distinguish flow cleanups from proven external removals and from missing paths of unknown provenance.

## 4. DB migration steps (optional)

Not applicable: the change alters no stored data schema. The ledger gains new status values (e.g. missing with unknown provenance, proven external) through the owning skill's record format, not through a migration; the plan defines the exact values.

## 5. Policy conflicts

| policy | conflict | resolution |
|---|---|---|
| `docs/cloud-agent-flow.md` hard rule 6 (clean up, never lose work; unmerged never deleted) | A broader cleanup gate could delete real work | No conflict: S1 keeps merged-only removal and unproven-provenance content always keeps the worktree; S2–S4 only narrow what counts as blocking on provenance proof, doubt keeps the worktree |
| Test-first for behavior changes (`AGENTS.md`) | Cleanup-gate behavior changes | S1–S7 each name their test layer; fixtures cover rebuildable vs non-rebuildable, generated vs hand-authored, filename-only vs provenance-proven, missing-unknown vs proven-external, and missing Task Close with same-wave/inactive vs other-wave/active |
| Ledger ownership safeguards | Auto-removal of non-coder worktrees could delete what the flow does not own | Resolved by S6/D4 (owner decision): non-coder worktrees are report-only and never auto-removed; unowned worktrees are reported, never auto-removed |
| Task and report controls | A fallback sweep could close tasks or bypass DONE/report gates | No conflict: S7/D6 forbid it — the fallback never marks DONE/BLOCKED, never closes the task, never bypasses report controls |

## 6. Open questions

| # | question | status |
|---|---|---|
| 1 | Intent question 1 (product card for #916) | carried to owner: confirm the card holding #916 before acceptance (publication/traceability unknown; not invented here) |
| 2 | Intent question 2 (proof that `.atl/` is rebuildable) | answered by S2/D2, to confirm at acceptance: rebuildable means regenerable by re-running the verifier with no user input inside, proven by rebuild provenance — never by path or glob alone |
| 3 | Intent question 3 (which generated/placeholder/empty leftovers are safe) | answered by S3–S4/D3, to confirm at acceptance: only provenance-proven cases — generation-proven output, ownership- and orphan-proven `*cwd-placeholder`, actually empty same-wave-owned folders; filename/glob match alone never suffices |
| 4 | Intent question 4 (non-coder worktree owner) | answered (owner decision): non-coder cleanup is manual for now — report only, never auto-remove (S6/D4) |
| 5 | Intent question 5 (missing-worktree record) | answered by S5/D5, to confirm at acceptance: mark missing with provenance (proven external with evidence vs missing with unknown provenance), report separately; a missing path alone never names an actor |
| 6 | Intent question 6 (sweep when Task Close never runs) | answered by S7/D6, to confirm at acceptance: the same-wave owning sweep evaluates merged state after same-wave ledger and inactive-ownership checks, and never closes the task or bypasses DONE/report controls |

## 7. Acceptance

Filled by a human only.

| accepted by | date | notes |
|---|---|---|
| Project owner (interactive selection) | 2026-11-26 | “Aceptar la spec y delegar el plan”; local plan only, without implementation or publication. The product card remains TBD. |
