package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0030 S5 (plan T4): the runner judges a coder's method evidence with
// the field list of the 0027 report gate (S19), read from the gate text in
// gabo-cycle step 6. The fields, in the gate's order, are pinned on both sides
// so the runner and the gate cannot drift apart.
var reportGateFields = []string{
	"`odd:`",
	"`spec S<n>:`",
	"`RDD:`",
	"`contract:`",
	"`tests:`",
	"`integration:`",
	"`commits:`",
	"`verifier:`",
}

// The runner's script names each gate field with the same token.
var runnerReportFields = []string{
	"odd:", "spec S", "RDD:", "contract:", "tests:", "integration:", "commits:", "verifier:",
}

func TestEvalRunnerFieldListMatchesTheReportGate(t *testing.T) {
	t.Parallel()
	gate := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md (report gate, S19)", gate, reportGateFields)
	// The gate's wording: the missing-field refusal and the not-run rule.
	requireIn(t, "gabo-cycle/SKILL.md (report gate, S19)", gate, []string{
		"`refused: RDD line missing`",
		"(the same form `refused: <field> line missing` for any other field)",
		"a field present as `not run: <reason>` passes the gate",
	})

	script, err := os.ReadFile(filepath.Join(evalRunnerSkill, "scripts", "eval-run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	// Each field is named in the script, in the gate's order.
	text := string(script)
	last := -1
	for _, field := range runnerReportFields {
		i := strings.Index(text, field)
		if i < 0 {
			t.Fatalf("eval-run.sh: the report gate field %q is missing", field)
		}
		if i <= last {
			t.Fatalf("eval-run.sh: report gate field %q is out of the gate's order", field)
		}
		last = i
	}
}

// The skill documents the method-evidence rule and its refusal form (S5).
func TestEvalRunnerSkillStatesTheMethodEvidence(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(evalRunnerSkill, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	requireIn(t, "gabo-eval-runner/SKILL.md", string(b), []string{
		"## Method evidence",
		"`fail: <field> line missing`",
		"`spec S<n>:`",
		"`not run: <reason>`",
		"`fail: check <cmd> failed (exit <n>)`",
		"gabo-cycle",
	})
}
