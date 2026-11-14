package skills

import (
	"strings"
	"testing"
)

// Intent 0027 S9 (plan T5): step 6 of gabo-cycle writes one task line per
// outcome to the wave trace: the RDD outcome, the ODD line and both gate
// outcomes. A missing value reads `not run: <reason>`.
func TestCycleStep6WritesTheTaskTraceLines(t *testing.T) {
	t.Parallel()
	s := step6(t, readCatalog(t, "gabo-cycle/SKILL.md"))
	requireIn(t, "gabo-cycle step 6", s, []string{
		"the trace gets one task line per outcome",
		"`task <id>: RDD: <outcome>; ODD: <RED/GREEN or exception>; gates: done <passed or refused>, report <passed or refused>`",
		"each value copied from that outcome's own `RDD:` and `odd:` lines and from the two gate results",
		"a missing value reads `not run: <reason>`",
		"a resumed wave keeps these lines (PRD R25)",
		"a BLOCKED outcome runs no gate, so its gates value reads `not run: blocked`",
	})
}

// Intent 0027 S9 (plan T5): the wave report's Tasks line shows the trace
// values for every task, and a resumed wave keeps them (PRD R25).
func TestWaveReportTasksLineShowsTheTraceFields(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	_, rest, ok := strings.Cut(doc, "2. **Tasks**")
	if !ok {
		t.Fatal("gabo-wave-report/SKILL.md: no section 2 (Tasks)")
	}
	tasks, _, _ := strings.Cut(rest, "\n3. **Questions**")
	requireIn(t, "gabo-wave-report Tasks line", tasks, []string{
		"`RDD: <outcome>`",
		"`ODD: <RED/GREEN or exception>`",
		"`gates: done <passed or refused>, report <passed or refused>`",
		"for every task",
		"`not run: <reason>`",
		"a resumed wave keeps the lines the trace holds",
	})
}
