package skills

import (
	"strings"
	"testing"
)

// step6 returns step 6 of the gabo-cycle execution steps (one line).
func step6(t *testing.T, doc string) string {
	t.Helper()
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "6. On each outcome") {
			return line
		}
	}
	t.Fatal("gabo-cycle/SKILL.md: no step 6 (`6. On each outcome`)")
	return ""
}

// Intent 0027 S18 (plan T4): the done gate is a rule of step 6, read before DONE.
func TestCycleStep6DoneGate(t *testing.T) {
	t.Parallel()
	s := step6(t, readCatalog(t, "gabo-cycle/SKILL.md"))
	for _, want := range []string{
		"done gate",
		"before you write DONE",
		"each `review.checks` command of `gabo.yaml`",
		"command line as listed",
		"`exit: <n>`",
		"`refused: <command>`",
		"`refused: exit line missing for <command>`",
		"`passed`",
		"A refused task does not close",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("gabo-cycle step 6: done gate lacks %q", want)
		}
	}
}

// Intent 0027 S19 (plan T4): the report gate refuses an incomplete coder report.
func TestCycleStep6ReportGate(t *testing.T) {
	t.Parallel()
	s := step6(t, readCatalog(t, "gabo-cycle/SKILL.md"))
	for _, want := range []string{
		"report gate",
		"`refused: RDD line missing`",
		"`refused: verifier line missing`",
		"`verifier: not applicable: <reason>`",
		"`not run: <reason>` passes the gate",
		"`odd:`", "`spec S<n>:`", "`contract:`", "`tests:`", "`commits:`", "`RDD:`",
		"`integration: not applicable: <reason>`",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("gabo-cycle step 6: report gate lacks %q", want)
		}
	}
}

// The three copies of gabo-cycle stay byte-identical (plan T4).
func TestCycleCopiesStayByteIdentical(t *testing.T) {
	t.Parallel()
	base := readRepoDoc(t, "apps/cli/internal/skills/gabo-cycle/SKILL.md")
	for _, p := range []string{".gabo/skills/gabo-cycle/SKILL.md", "plugins/codex/skills/gabo-cycle/SKILL.md"} {
		if readRepoDoc(t, p) != base {
			t.Errorf("%s differs from the catalog copy", p)
		}
	}
}
