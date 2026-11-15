package skills

import "testing"

// Intent 0020 (plan T7, PD14): the fix-round limit travels in the coder launch
// input like the turn limit, the coder stops at it, the cycle pauses on it and
// the paused report names the enforcement and spend lines.
func TestCycleFixRoundLimitInLaunchInput(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"`fix round limit: <n>`", "when `wave.max_fix_rounds` is set",
		"`STOPPED at fix round limit`", "the cap `max_fix_rounds`",
		"its task stays claimed with its ledger handle", "nothing is written BLOCKED or DONE",
		"`cap max_fix_rounds reset <UTC time>`", "the 4-round ceiling stays absolute",
		"no `fix round limit` is passed",
		"`fix rounds used: <k>`", "`fix round limit: <k + wave.max_fix_rounds, never above 4>`", "the limit counts total rounds",
	})
}

func TestCoderStopsAtFixRoundLimit(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"`fix round limit: <n>`", "that many fix rounds in total without a clean review",
		"commits nothing new", "`STOPPED at fix round limit`", "worktree, branch, containers",
		"counting from `fix rounds used: <k>`", "none means zero", "never raises the fix-round cap", "Max 4 fix rounds",
	})
}

func TestAssistantPausedReportCarriesEnforcementAndSpendLines(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		"cap lines of its Wave section", "wave-record.sh caps --engine <engine> --paused",
	})
}

func TestWaveReportFixRoundsAdvisory(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report/SKILL.md", doc, []string{
		"the labels belong to the script and are never composed here",
		"`spend is not capped by gabo`",
	})
}
