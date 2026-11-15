package skills

import (
	"strings"
	"testing"
)

// Intent 0022 T7 (PD19): the wave report skill is read first by path, and the
// trace saves are mandatory and read back before the final answer.
func TestCycleReportStepReadsTheWaveReportSkillFirst(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle", doc, []string{
		"read `.gabo/skills/gabo-wave-report/SKILL.md` first and build the wave report from it",
	})

	step8 := doc[strings.Index(doc, "\n8. Report:"):]
	if !strings.Contains(step8, ".gabo/skills/gabo-wave-report/SKILL.md") {
		t.Error("gabo-cycle: step 8 does not name the wave report skill path")
	}
}

func TestCycleTraceSavesAreMandatoryBeforeTheFinalAnswer(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	i := strings.Index(doc, "- Trace (D17):")
	if i < 0 {
		t.Fatal("gabo-cycle: missing the Trace (D17) rule")
	}
	end := strings.Index(doc[i:], "\n\n")
	trace := doc[i : i+end]
	requireIn(t, "gabo-cycle Trace (D17)", trace, []string{
		"the trace saves are mandatory: the wave start entry, one entry per task and the wave report as the last entry",
		"each is read back before the final answer",
		"a save that does not show up after one retry is listed as unverified in the report",
	})
}

func TestAssistantWaveCloseReadsTheWaveReportSkillAndSavesFirst(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	i := strings.Index(doc, "## Wave Close")
	if i < 0 {
		t.Fatal("gabo-assistant: missing Wave Close")
	}
	close := doc[i:]
	requireIn(t, "gabo-assistant Wave Close", close, []string{
		"read `.gabo/skills/gabo-wave-report/SKILL.md` first",
		"the final answer is the wave report built from `gabo-wave-report` and is never composed before those saves are read back",
	})
}
