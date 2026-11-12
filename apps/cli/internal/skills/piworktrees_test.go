package skills

import (
	"strings"
	"testing"
)

// Intent 0023 S3 (plan T4, finding #563, D6): on Pi the cycle registers each
// task worktree with `session_worktree_register` after creating it and before
// the coder launch, and writes `registered` on its ledger line. The step is
// text in `gabo-cycle`; the launch line itself stays as it is.
func TestCycleRegistersPiWorktreesBeforeTheCoder(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	step5 := lineWith(t, doc, "5. Per task:")
	requireIn(t, "gabo-cycle step 5", step5, []string{
		"your session engine is `pi`",
		"`session_worktree_register`",
		"add `registered` to that worktree's ledger line",
		"launch `gabo-coder` as a first-level in-process subagent of this session",
	})
	reg := strings.Index(step5, "`session_worktree_register`")
	launch := strings.Index(step5, "launch `gabo-coder`")
	gitAdd := strings.Index(step5, "`git worktree add")
	if reg < 0 || launch < 0 || gitAdd < 0 {
		t.Fatalf("gabo-cycle step 5: missing worktree add, registration or launch (%d, %d, %d)", gitAdd, reg, launch)
	}
	if !(gitAdd < reg && reg < launch) {
		t.Errorf("gabo-cycle step 5: registration must sit after `git worktree add` and before the coder launch (add %d, register %d, launch %d)", gitAdd, reg, launch)
	}
}

// Intent 0023 S3 (D6): a failed or missing registration never stops the wave:
// the coder is launched anyway and the reason is written on the ledger line
// and in the wave report.
func TestCycleLaunchesTheCoderWhenRegistrationFails(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-cycle/SKILL.md", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"When the tool is missing or the call fails",
		"launch the coder anyway",
		"`worktree not registered (<reason>)`",
		"to the wave report",
		"never stop the wave for it",
	})
}

// Intent 0023 S3: a Claude Code, Codex, OpenCode or other-engine wave registers
// nothing and its ledger and report carry no registration line.
func TestCycleRegistersNothingOffPi(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-cycle/SKILL.md", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"On another engine register nothing",
	})
}

// Intent 0023 S3 and PD6: the wave report gets that one reason line in its
// Leftovers item (item 7), nothing else.
func TestWaveReportLeftoversNamesAnUnregisteredWorktree(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	leftovers := lineWith(t, doc, "7. **Leftovers**")
	requireIn(t, "gabo-wave-report Leftovers item", leftovers, []string{
		"`worktree not registered (<reason>)`",
		"coder task closes",
	})
}
