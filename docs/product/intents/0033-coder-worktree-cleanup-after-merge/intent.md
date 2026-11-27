# Intent 0033 Coder worktree cleanup after merge

| field | value |
|---|---|
| intent id | 0033-coder-worktree-cleanup-after-merge |
| status | Draft |
| owner | gabo-pm |
| product card | TBD (publication/traceability unknown; not invented here; owner to confirm which card holds [#916](https://github.com/testeos-z/gabo-ai/issues/916)) |
| intent issue | [#916](https://github.com/testeos-z/gabo-ai/issues/916) |
| date | 2026-11-26 |

## Problem

Merged coder worktrees are retained on the server instead of being removed. The issue reports worktrees surviving after their PRs merged because the cleanup gate sees leftover state that is not real unsaved work: an ignored verifier artifact (`.atl/`) created during verification, an untracked generated tree (`apps/storybook/storybook-static/`), orphan `*cwd-placeholder` files, and empty wave folders. Separately, external removals by Orca muddle the cleanup ledger, there is no cleanup owner for non-coder worktrees, and the cycle sweep only covers its own ledger entries in `DONE`/`BLOCKED` — while Task Close may never run when a coder is never resumed.

The owner expectation is simple: remove a merged worktree unless it holds real unsaved work.

## Who is affected

- The owner, who watches disk and hygiene on the cloud server (`vulcam cloud`) and sees merged worktrees pile up after every wave.
- The coder coordinator (`gabo-coder`) and each short coder session, whose finished worktrees stay behind even though their PRs merged.
- The assistant running the wave, whose ledger no longer matches what is actually on disk once Orca removes something externally.

## Desired outcome

- A merged coder worktree is removed unless it holds real unsaved work.
- Provenance-proven, regenerable leftovers do not keep a merged coder worktree alive; anything of unknown provenance keeps it.
- The ledger stays truthful when a worktree path goes missing, without inferring who removed it.
- Coder worktrees have a cleanup owner; non-coder worktrees are manual for now — reported only, never auto-removed (owner decision).
- A merged coder worktree owned by the same wave is still cleaned up when the coder is never resumed and Task Close never runs, without closing tasks or bypassing report controls.

## Constraints

- Draft only: this change writes `intent.md` and `spec.md` under this folder; no acceptance, no plan, no tracker writes, no remote calls, no commits.
- Clean up, never lose work (`docs/cloud-agent-flow.md` hard rule 6): unmerged branches, unpushed work, and real unsaved work are never deleted; merged-only removal.
- Ledger ownership safeguards: the flow never deletes outside what the ledger owns, and never rewrites ledger history to hide a missing worktree.
- Merged-only safeguard: removal applies to work whose PR merged, never to open or unmerged work.
- Unknown-content safeguard: unknown, untracked, or ignored content is preserved regardless of author; disposability is never inferred from a filename or glob alone.
- Non-coder safeguard (owner decision): non-coder cleanup is manual for now; the flow reports non-coder worktrees only and never auto-removes them.
- Missing-path safeguard: a missing path alone never proves an external actor; missing with unknown provenance is recorded as such.
- Fallback safeguard: the fallback sweep acts only on same-wave ledger entries with inactive ownership, and never closes a task or bypasses DONE/report controls.

## Open questions

| # | question | status |
|---|---|---|
| 1 | Which product card holds issue #916? | open (publication/traceability unknown; not invented here) |
| 2 | What counts as proof that ignored `.atl/` is rebuildable and safe to ignore at cleanup? | carried to spec |
| 3 | Which generated/placeholder/empty-folder leftovers are safe to treat as clean, and which must keep the worktree? | carried to spec |
| 4 | Who owns cleanup of non-coder worktrees? | answered (owner decision): manual for now — report only, never auto-remove |
| 5 | How does the ledger record a missing worktree without muddling ownership? | carried to spec (missing alone never proves an external actor; distinguish missing/unknown provenance) |
| 6 | Which sweep covers a merged worktree when the coder is never resumed and Task Close never runs? | carried to spec |

## Links

| kind | reference |
|---|---|
| intent issue | [#916](https://github.com/testeos-z/gabo-ai/issues/916) |
| product card | TBD (publication/traceability unknown; not invented here; owner to confirm which card holds #916) |
| PRD | [PRD](../../PRD.md) |
| TRD | [TRD](../../TRD.md) |
| flow | [docs/cloud-agent-flow.md](../../../../cloud-agent-flow.md) (hard rule 6: clean up, never lose work) |
| spec | [spec](spec.md) |
| plan | [plan](plan.md) (not written: draft only, no plan) |
| related intent | [0032 Intent landing branches](../0032-intent-landing-branches/intent.md) (branch lifecycle this cleanup follows) |
