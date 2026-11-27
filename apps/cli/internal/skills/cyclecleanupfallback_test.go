package skills

import (
	"testing"
)

// Intent 0033 S5 (plan T4, cycle side): the ledger marks a gone path with its
// provenance — missing with unknown provenance, proven external with evidence,
// or flow-owned with owner and reason — and never rewrites a missing entry as
// a flow cleanup or names an actor from the missing path alone.
func TestCycleMissingPathProvenanceLedger(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"Missing-path provenance (intent 0033 S5, D5)",
		"is marked `missing-unknown` (missing with unknown provenance), kept as history and reported separately from flow cleanups and from proven external removals",
		"a missing path alone never names an actor",
		"A removal proven external is marked `removed-external` with the evidence cited",
		"a flow-owned removal records the owner and the reason",
		"A missing entry is never rewritten as a flow cleanup",
	})
}

// Intent 0033 S6 (plan T4, cycle side): every coder worktree resolves to
// exactly one owner; non-coder and unowned worktrees are reported only and
// never auto-removed.
func TestCycleCleanupOwnershipTable(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"Cleanup ownership (intent 0033 S6, D4)",
		"every coder worktree under the flow's worktree roots resolves to exactly one owner",
		"the coder flow or the review cleanup",
		"A non-coder worktree is manual cleanup for now (owner decision): report it only, never auto-remove it",
		"A worktree with no resolvable owner is reported as unowned and never auto-removed",
	})
}

// Intent 0033 S7 (plan T5): the same-wave owning sweep covers a merged coder
// worktree with a same-wave ledger entry, verified inactive ownership and no
// Task Close record, evaluates it under the S1–S4 gate, and never closes the
// task nor bypasses DONE or report controls.
func TestCycleSameWaveSweepFallbackGuards(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"Never-resumed coder fallback (intent 0033 S7, D6)",
		"as the only exception to the DONE-or-BLOCKED rule above",
		"a same-wave ledger entry, verified inactive coder ownership and no Task Close record",
		"the owning sweep evaluates it under the S1–S4 gate (intent 0033) and removes it unless real unsaved work exists",
		"It never marks the task DONE or BLOCKED, never closes the task, and never bypasses DONE or report controls",
		"the task state and the report controls stay untouched",
		"With no same-wave ledger entry, an active owner, or an unmerged worktree, the fallback does not act",
	})
	step7 := lineWith(t, doc, "7. The wave closes when")
	requireIn(t, "gabo-cycle step 7", step7, []string{
		"except a merged coder worktree with a same-wave ledger entry, verified inactive coder ownership and no Task Close record, which the never-resumed coder fallback evaluates under the S1–S4 gate without touching task state",
	})
}
