package skills

import "testing"

// Intent 0020 (plan T2): the cycle pauses a capped wave, the assistant presents
// the paused report and asks "continue or stop", and the wave report carries
// the cap lines in its existing Wave section (PD3, PD8, PD9, PD10, PD11).
func TestCycleLaunchCapsPauseTheWave(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		// coder count cap (S3, PD9).
		"`wave.max_coders`", "no more than that many coders in the wave in total",
		"a resume of a coder by its ledger handle does not count", "a replacement `gabo-coder` counts",
		"pauses before a launch that would exceed it",
		// fix round cap (S3, D6): rule 3 stays.
		"`wave.max_fix_rounds`", "below 4", "without a clean review", "the task is not BLOCKED",
		"unset or 4, rule 3 is unchanged", "Never launch a fifth round",
		// turn limit in the launch input (S2, PD11).
		"`turn limit: <n>`", "`wave.max_coder_turns`",
	})
}

func TestCycleTurnStoppedCoderPausesTheWave(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"partial output", "no outcome status after a wrap-up summary", "`STOPPED at turn limit`",
		"stopped at the turn cap", "its task stays claimed with its ledger handle",
		"nothing is written BLOCKED or DONE", "the cap `max_coder_turns`",
		// resume by handle or replacement counted against max_coders (PD10).
		"resumes the coder by that handle", "that replacement counts against `wave.max_coders`",
	})
}

func TestCyclePauseAndContinue(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"A pause launches nothing new", "lets the running coders return",
		"saves the state in the tracker, the ledger and the trace", "paused report",
		"cap <name> reset <UTC time>", "counts that cap from zero",
		"the 4-round ceiling stays absolute",
	})
}

func TestAssistantWaveClosePresentsThePausedReport(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		"paused report", "final answer of the wave in auto mode", "continue or stop",
		"`resume <session id>`", "under its Resume rule",
		"closes the wave through the normal close", "unfinished tasks left as they are", "launches no new coder",
	})
}

func TestWaveReportCarriesTheCapLines(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report/SKILL.md", doc, []string{
		"`status: paused`", "the cap that fired", "the affected task ids", "continue or stop",
		"wave-record.sh caps --engine <engine>", "spend is not capped by gabo",
	})
}

func TestWaveReportEnforcementReadsTheBuiltAgent(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-cycle/SKILL.md", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"The one exception is a paused wave the owner stops", "unfinished tasks left as they are",
	})
}

func TestCycleMaxCodersCountRestartsOnContinue(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-cycle/SKILL.md", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"counted from the wave start or from the last `cap max_coders reset` line of the trace",
		"returns without an outcome status (`HANDED`, `DONE` or `BLOCKED`)",
	})
}

func TestWaveCapReviewFixes(t *testing.T) {
	t.Parallel()
	requireIn(t, "gabo-cycle/SKILL.md", readCatalog(t, "gabo-cycle/SKILL.md"), []string{
		"With `wave.max_coder_turns` set only", "unset keeps today's handling of a coder without a status",
		"stays claimed with no worktree and no PR; on resume the cycle launches it as a first launch",
	})
	requireIn(t, "gabo-wave-report/SKILL.md", readCatalog(t, "gabo-wave-report/SKILL.md"), []string{"`<cap>: <label>`", "the labels belong to the script"})
	requireIn(t, "gabo-assistant/SKILL.md", readCatalog(t, "gabo-assistant/SKILL.md"), []string{"In manual mode the paused report is shown the same way"})
}
