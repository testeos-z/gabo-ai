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
		"each `review.checks` command of `gabo.yaml`",
		"`check: <command line as listed in review.checks>`",
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
		"`integration:`",
		"and two more required fields",
		"until then a missing `verifier:` line passes",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("gabo-cycle step 6: report gate lacks %q", want)
		}
	}
}

// The gates run on the outcome text, before the Task Close writes DONE: the
// cycle never writes DONE (D23), so a refused task is held or reopened.
func TestCycleStep6GatesRunBeforeTheClose(t *testing.T) {
	t.Parallel()
	s := step6(t, readCatalog(t, "gabo-cycle/SKILL.md"))
	for _, want := range []string{
		"text of each HANDED or DONE outcome",
		"the coder writes DONE in its Task Close, never you",
		"do not resume the coder with `PR #<n> merged`",
		"reopen the task issue",
		"count the task as not DONE",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("gabo-cycle step 6: gate ordering lacks %q", want)
		}
	}
	if strings.Contains(s, "before you write DONE") {
		t.Error("gabo-cycle step 6: still says the cycle writes DONE")
	}
}

// The gate and the coder contract name the same fields (S19, the S10
// pattern): each field the gate requires is in the coder's methodology report
// block. `verifier:` joins it with S28 (plan T3); the gate accepts its absence
// until then, so it is checked only once the coder names it.
func TestCycleGateFieldsAreInTheCoderReportBlock(t *testing.T) {
	t.Parallel()
	coder := readCatalog(t, "gabo-coder/SKILL.md")
	_, block, ok := strings.Cut(coder, "### Methodology report block")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no methodology report block")
	}
	// the block ends at the next heading.
	if i := strings.Index(block, "\n#"); i >= 0 {
		block = block[:i]
	}
	gate := step6(t, readCatalog(t, "gabo-cycle/SKILL.md"))
	for _, f := range []string{"`odd:", "`spec S<n>:", "`contract:", "`tests:", "`integration:", "`check:", "`commits:", "`RDD:"} {
		if !strings.Contains(gate, f) {
			t.Errorf("gabo-cycle step 6 does not require %s`", f)
		}
		if !strings.Contains(block, f) {
			t.Errorf("gabo-coder report block lacks %s` that the gate requires", f)
		}
	}
	if strings.Contains(block, "`verifier:") && !strings.Contains(gate, "`verifier:") {
		t.Error("the coder names `verifier:` but the gate does not require it")
	}
}

// Step 3 (Resume) keeps a task whose DONE was refused out of the DONE skip.
func TestCycleStep3KeepsARefusedDoneOpen(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	var step3 string
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "3. Resume") {
			step3 = line
		}
	}
	for _, want := range []string{"An open issue carrying a report gate refusal comment", "launch a coder only to complete the report and close again", "never to run the whole Task Close on a report the gates refused"} {
		if !strings.Contains(step3, want) {
			t.Errorf("gabo-cycle step 3: lacks %q", want)
		}
	}
}

// A refused DONE reaches a coder that may be gone: the rule says how.
func TestCycleStep6RefusedDoneSaysHowToReachTheCoder(t *testing.T) {
	t.Parallel()
	s := step6(t, readCatalog(t, "gabo-cycle/SKILL.md"))
	for _, want := range []string{"resumed by its handle", "a new `gabo-coder` launched from the main checkout", "never as live code work and never as DONE", "read the completed outcome through both gates again", "the coder closes the issue again with DONE via PR", "on a second refusal, mark the task BLOCKED with the refusal string"} {
		if !strings.Contains(s, want) {
			t.Errorf("gabo-cycle step 6: refused DONE lacks %q", want)
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
