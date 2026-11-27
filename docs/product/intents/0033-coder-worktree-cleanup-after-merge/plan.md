# Plan 0033 Coder worktree cleanup after merge

| field | value |
|---|---|
| intent | [0033-coder-worktree-cleanup-after-merge](intent.md) |
| spec | [spec](spec.md) |
| status | Draft |
| owner | gabo-dev-lead |
| date | 2026-11-27 |

## Steps

1. Tighten the Task Close cleanup gate to merged-only removal with conservative provenance proof (S1–S4, D1–D3): untracked or ignored content of unproven provenance keeps the worktree regardless of author; filename or glob match alone never proves disposability; verifier scratch moves outside the worktree.
2. Make the ledger truthful about missing paths and ownership (S5–S6, D4–D5): record missing with provenance (proven external with evidence vs missing with unknown provenance, never inferring an actor from a missing path alone); coder worktrees resolve to exactly one owner while non-coder worktrees are manual — report only, never auto-removed.
3. Cover the never-resumed coder gap with the same-wave owning sweep fallback under strict guards (S7, D6): same-wave ledger plus verified inactive ownership, S1–S4 evaluation only, never closing the task and never bypassing DONE/report controls.
4. No `gabo check` update (TRD 8.5): this change touches skill text (`gabo-cycle`, `gabo-coder`, verifier integration, ledger record format) and adds no Go check command or check gate; no task covers a check update.

## Tasks

Task ids are local to the intent. A dependency in another intent is written `<NNNN>/T<n>`. `depends_on` and `unblocks` mirror each other; `—` means none. A removed task keeps its row, marked "(removed <YYYY-MM-DD>)", and the traceability check ignores it.

| id | title | spec refs | depends_on | unblocks | acceptance | size |
|---|---|---|---|---|---|---|
| T1 | Merged-only removal gate with conservative provenance default | S1 | — | T2, T3, T4 | Test first: skill text test asserting the merged-only rule plus a probe on fixtures — merged worktree clean removes; merged worktree with unpushed commits, staged changes, tracked modifications, or unknown-provenance untracked/ignored content is kept with the item named; filename/glob match alone never counts as disposable. | M |
| T2 | Verifier scratch outside the worktree with rebuild proof for `.atl/` | S2, D1, D2 | T1 | T3, T5 | Test first: skill text test of the ignored-scratch rule plus probes — merged worktree whose only leftover is rebuild-proven `.atl/` (regenerable by re-running the verifier, no user input inside, provenance proven not by path alone) does not block removal; non-rebuild-proven scratch keeps the worktree with reason naming it; verifier integration names the external scratch path outside the worktree. | S |
| T3 | Provenance-proven generated output, placeholders, and empty wave folders | S3, S4, D3 | T1, T2 | T5 | Test first: skill text test of the generated-output and placeholder/empty-folder rules plus probes — generation-proven untracked output (e.g. reproducible `apps/storybook/storybook-static/` with no hand input, proven not assumed from path) does not block removal while hand-authored output keeps the worktree; `*cwd-placeholder` with proven flow ownership and orphaned provenance plus actually empty same-wave-owned wave folders do not block removal and are removed with the worktree; filename-only match, non-empty folder, or folder not owned by the same wave keeps the worktree with reason naming the item. | M |
| T4 | Missing-path provenance ledger plus single-owner table with manual non-coder path | S5, S6, D4, D5 | T1 | T5 | Test first: skill text test of the missing-path record rule and ownership table read-back plus probes — ledger entry whose path is gone with no flow record and no proof of who removed it is marked missing with unknown provenance (kept as history, reported separately from flow cleanups and proven external removals); proven-external fixture records evidence cited; flow removal records owner and reason; coder worktree resolves to exactly one owner (coder flow or review cleanup); non-coder and unowned worktrees are reported as manual-cleanup/unowned and never auto-removed. | M |
| T5 | Same-wave owning sweep fallback with inactive checks and no-close/no-bypass guard | S7, D6, D4 | T2, T3, T4 | — | Test first: skill text test of the fallback trigger and its no-close/no-bypass guard plus probes — merged coder worktree with same-wave ledger entry, verified inactive coder ownership, and no Task Close record is evaluated under S1–S4 and removed unless real unsaved work exists; other-wave entry, active owner, or unmerged worktree does not act; fallback run never marks DONE/BLOCKED, never closes the task, and never bypasses report controls (task state and report controls untouched). | M |

## Risks to the plan

| risk | impact | mitigation |
|---|---|---|
| Provenance proof stated loosely lets filename/glob inference back in | T1, T2, T3 — real hand-authored work deleted | Require proof-by-rebuild/generation/ownership wording in each gate; fixtures pair proven vs filename-only cases and doubt keeps the worktree |
| Ledger rewrites a missing path as a flow cleanup or names an actor from the missing path alone | T4 — ledger history muddled | Keep missing entries as history with provenance values only (proven external with evidence vs missing with unknown provenance); report missing-unknown separately |
| Fallback sweep widens to other-wave entries, active owners, or touches task/report state | T5 — unmerged work removed or DONE/report bypassed | Gate fallback on same-wave ledger plus verified inactive ownership plus merged state only, with explicit never-close/never-bypass assertions in tests |
| Non-coder worktrees pulled into auto-removal | T4, T5 — flow deletes what it does not own | Ownership table plus manual-report-only path; unowned and non-coder fixtures assert report-only, never removed |

## Open questions

1. Which product card holds #916? (product card TBD — publication/traceability unknown; owner to confirm; not invented here)
2. Tracker traceability not verified: intent tracker item and task issues were not created or re-read in this local-plan-only step (no tracker writes, no remote calls, no commits authorized); traceability check against the tracker is pending owner action.
