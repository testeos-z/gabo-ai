package skills

import "testing"

// Intent 0021 (plan T3): while the main session runs the cycle it launches
// gabo-coder subagents and nothing else, each under reason (a), with the
// reason line first in the launch input.
func TestCycleLaunchesCodersOnlyUnderReasonA(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		// S4: coders only, at most parallelism at once, one per task worktree.
		"while the main session runs the cycle it launches `gabo-coder` subagents and no other agent",
		"no explorer, verifier or reviewer: the coder runs the reviewer script itself",
		// S2: each coder launch is reason (a); a wave of one task launches one coder.
		"each coder launch is reason (a)",
		"the launch input starts with `Reason: (a) parallel units`",
		"a wave of one task launches one coder, because the task is a separate unit in its own worktree",
		// PD7: resume and replacement continue the same unit.
		"a resume of a coder by its ledger handle and a replacement coder for the same task continue the unit launched under (a)",
		"repeat the line `Reason: (a) parallel units`",
		"neither is a new handoff",
		// the existing launch sentences and wave caps stay.
		"first-level in-process subagent",
		"never a named teammate",
		"`wave.max_coders`",
	})
}
