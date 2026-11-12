package skills

import (
	"strings"
	"testing"
)

// Intent 0024 (plan T4, S3, D1, D4): the cap lines and the paused answer come
// from `wave-record.sh caps`, pasted unchanged; the skills carry no label table.
const capsCall = "wave-record.sh caps --engine <engine>"

func TestWaveReportCapLinesAreTheScriptOutput(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report/SKILL.md", doc, []string{
		capsCall, "pasted unchanged", "`status: paused`", "continue or stop",
		"after the last section and after the `session id` line",
	})
}

func TestCyclePauseRunsCapsPaused(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		capsCall + " --paused", "paused report", "cycle returns its output in the paused report",
	})
}

func TestAssistantPausedAnswerCarriesTheScriptOutput(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		capsCall + " --paused", "unchanged", "its last line is `continue or stop`",
		"placed as the Wave section of `gabo-wave-report` says", "asks no comfort question", "when the script fails the answer still ends with `continue or stop`",
	})
}

func TestSkillsNameNoCapLabelOutsideTheScript(t *testing.T) {
	t.Parallel()
	for _, f := range []string{"gabo-wave-report/SKILL.md", "gabo-cycle/SKILL.md", "gabo-assistant/SKILL.md"} {
		doc := readCatalog(t, f)
		for _, bad := range []string{"`advisory`", "`enforced by the cycle`", "`enforced by the engine`", "max_coder_turns: advisory", "as `advisory` (coder-enforced)", "enforced by the engine", "enforced by the cycle"} {
			if strings.Contains(doc, bad) {
				t.Errorf("%s: names the label %s; only the script prints labels", f, bad)
			}
		}
	}
}

// The turn-cap label is the script's fixed table (S3); the skills never read
// the built agent file to compose it.
func TestWaveReportNoLongerComposesTheTurnCapLabel(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	for _, bad := range []string{"carries the engine key `maxTurns` or `steps`", "say `advisory`"} {
		if strings.Contains(doc, bad) {
			t.Errorf("gabo-wave-report/SKILL.md: still composes the turn-cap label (%q)", bad)
		}
	}
	requireIn(t, "gabo-wave-report/SKILL.md", doc, []string{"the labels belong to the script and are never composed here", "takes effect only after the agents are rebuilt", "`.opencode/agents/gabo-coder.md` for `steps:`", "a paused answer still ends with `continue or stop`"})
}
