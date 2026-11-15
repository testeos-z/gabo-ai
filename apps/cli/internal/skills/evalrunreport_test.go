package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0030 S6, S7 and D3 (plan T6): the run report names what the runner
// measured and compares it with the owner's baseline. The labels the report
// prints are pinned on both sides, the script and the skill text, so the two
// cannot drift apart. The fixture tests (eval-run.test.sh) check the output.

// The labels eval-run.sh prints for the run report, in the order it prints them.
var evalRunReportLabels = []string{
	"eval-run: head ",
	"run id: ",
	"memory topic: gabo/eval/",
	"changed configuration: ",
	"engine: ",
	"runs per task: ",
	"threshold: ",
	"baseline: none",
	"pass rate: ",
	"moved from pass to fail: ",
	"not compared (baseline: none)",
}

func TestEvalRunReportScriptPrintsEveryLabel(t *testing.T) {
	t.Parallel()
	script, err := os.ReadFile(filepath.Join(evalRunnerSkill, "scripts", "eval-run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range evalRunReportLabels {
		if !strings.Contains(string(script), label) {
			t.Errorf("eval-run.sh: the run report label %q is missing", label)
		}
	}
	// The runner never writes the baseline: no redirection or move targets it.
	for _, write := range []string{`>"$baseline"`, `> "$baseline"`, `>> "$baseline"`, `mv "$baseline"`, `cp "$baseline"`, `tee "$baseline"`} {
		if strings.Contains(string(script), write) {
			t.Errorf("eval-run.sh writes the baseline (%s); only the owner's acceptance replaces it", write)
		}
	}
}

func TestEvalRunnerSkillStatesTheRunReport(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(evalRunnerSkill, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	requireIn(t, "gabo-eval-runner/SKILL.md", string(b), []string{
		"## Run report",
		"`evals/baseline/baseline.json`",
		"`baseline: none`",
		"`threshold: not set`",
		"`gabo/eval/<head sha>/<run id>`",
		"`moved from pass to fail:`",
		"`pass rate:`",
		"never writes the baseline",
		"only the owner's acceptance",
	})
}

func TestEvalRunReportMemoryRecordRow(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "skills", "gabo-memory-engram", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	requireIn(t, "gabo-memory-engram/SKILL.md", string(b), []string{
		"| Eval run report | `gabo/eval/<head sha>/<run id>` | project |",
	})
}
