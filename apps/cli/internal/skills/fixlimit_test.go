package skills

import "testing"

// Intent 0024 S6 and S10 (plan T5, D8): a clean head at the fix round limit
// finishes the task, and the report names each coder's launch reason.
func TestCleanHeadAtFixRoundLimitFinishesTheTask(t *testing.T) {
	requireIn(t, "gabo-cycle", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"re-read the review of the head SHA it reports",
		"finishes through Task Close (the normal DONE path) and the wave does not pause",
		"`finished at the fix round limit (clean head)`",
		"with a blocking review the wave pauses with the cap `max_fix_rounds` as before",
	})
	requireIn(t, "gabo-coder", readCatalog(t, "gabo-coder/SKILL.md"), []string{
		"a clean head at the limit is reported as finished at the limit, not as paused",
	})
}

func TestReportNamesEachCodersLaunchReason(t *testing.T) {
	requireIn(t, "gabo-cycle", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"keeps the first handoff line `Reason: <label>` of each coder in the trace and the report carries it",
	})
	requireIn(t, "gabo-wave-report", readCatalog(t, "gabo-wave-report/SKILL.md"), []string{
		"`launched under <reason>`",
		"`launched under: no reason stated`",
	})
}
