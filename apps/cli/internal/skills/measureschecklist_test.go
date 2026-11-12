package skills

import (
	"strings"
	"testing"
)

// Intent 0022 T8 (PD20): Measures is an ordered checklist, the launch and merge
// steps append their own trace lines, the saved report is read back before the
// final answer, and the Decision Gates rows carry the planning record save.

func TestWaveReportMeasuresIsAnOrderedChecklist(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report Measures", doc, []string{
		"Measures is an ordered checklist of eight lines, built line by line from the trace and the tracker: wall time; fix rounds per task; agent sessions; PRs opened and merged; plan-change proposals; interactions; tokens; each task's outcome",
		"each line is printed even when it reads `unknown (<source that lacked it>)`",
	})
}

func TestCycleReportStepBuildsTheMeasuresChecklist(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	step8 := doc[strings.Index(doc, "\n8. Report:"):]
	requireIn(t, "gabo-cycle step 8", step8, []string{
		"Measures is an ordered checklist of eight lines, built line by line from the trace and the tracker: wall time; fix rounds per task; agent sessions; PRs opened and merged; the proposal count; interactions; tokens; each task's outcome",
		"each line is printed even when it reads `unknown (<source that lacked it>)`",
	})
}

func TestCycleReportStepReadsTheSavedReportBackBeforeTheFinalAnswer(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	step8 := doc[strings.Index(doc, "\n8. Report:"):]
	requireIn(t, "gabo-cycle step 8", step8, []string{
		"read the saved wave report back with `read wave trace` and compare it before the final answer, never after it",
	})
}

func TestAssistantWaveCloseReadsTheSavedReportBackBeforeTheFinalAnswer(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	i := strings.Index(doc, "## Wave Close")
	if i < 0 {
		t.Fatal("gabo-assistant: missing Wave Close")
	}
	requireIn(t, "gabo-assistant Wave Close", doc[i:], []string{
		"read the saved wave report back with `read wave trace` and compare it before the final answer, never after it",
	})
}

func TestCycleLaunchStepAppendsTheSessionStartedLine(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	i := strings.Index(doc, "\n5. Per task:")
	j := strings.Index(doc, "\n6. On each outcome")
	if i < 0 || j < i {
		t.Fatal("gabo-cycle: steps 5 and 6 not found")
	}
	requireIn(t, "gabo-cycle step 5", doc[i:j], []string{
		"before the launch call, append `session started <agent> <task id> <UTC time>` to `gabo/trace/<wave>` yourself (a coder resumed by its handle appends none)",
	})
}

func TestCycleOutcomeStepAppendsThePRLines(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	i := strings.Index(doc, "\n6. On each outcome")
	j := strings.Index(doc, "\n7. The wave closes")
	if i < 0 || j < i {
		t.Fatal("gabo-cycle: steps 6 and 7 not found")
	}
	requireIn(t, "gabo-cycle step 6", doc[i:j], []string{
		"in the same step, append the PR line yourself for each PR you see opened and for each PR you see merged",
	})
}

func TestAssistantDecisionGatesRowsSaveThePlanningRecord(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	i := strings.Index(doc, "## Decision Gates")
	j := strings.Index(doc, "## Planning Record")
	if i < 0 || j < i {
		t.Fatal("gabo-assistant: Decision Gates or Planning Record not found")
	}
	var pm, lead string
	for _, l := range strings.Split(doc[i:j], "\n") {
		switch {
		case strings.HasPrefix(l, "| PRD, intent, spec"):
			pm = l
		case strings.HasPrefix(l, "| TRD, an intent's plan"):
			lead = l
		}
	}
	want := "after the return and before the answer to the owner, save the planning record described in Planning Record (`gabo/trace/<I####>/planning`)"
	requireIn(t, "gabo-assistant gabo-pm row", pm, []string{want})
	requireIn(t, "gabo-assistant gabo-dev-lead row", lead, []string{want})
	requireIn(t, "gabo-assistant Planning Record", doc[j:], []string{
		"The Decision Gates rows for `gabo-pm` and `gabo-dev-lead` say when to save it",
	})
}
