package skills

import (
	"strings"
	"testing"
)

// Intent 0024 S7 (plan T7, D9): the directiva returns list the plan-change
// proposals, and the planning record reads `unknown` for a return without the
// field, never `none`.
func TestDirectiveReturnsListProposals(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"gabo-pm", "gabo-dev-lead"} {
		doc := readCatalog(t, name+"/SKILL.md")
		i := strings.Index(doc, "## Output Contract")
		if i < 0 {
			t.Fatalf("%s: missing the Output Contract section", name)
		}
		sec := doc[i:]
		if j := strings.Index(sec[3:], "\n## "); j >= 0 {
			sec = sec[:j+3]
		}
		requireIn(t, name+" Output Contract", sec, []string{
			"plan-change proposals opened", "the count", "the issue or PR and the scope of each", "`proposals: none`",
		})
	}
}

func TestPlanningRecordReadsMissingProposalsAsUnknown(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	i := strings.Index(doc, "## Planning Record")
	if i < 0 {
		t.Fatal("gabo-assistant/SKILL.md: missing the Planning Record section")
	}
	sec := doc[i:]
	if j := strings.Index(sec[3:], "\n## "); j >= 0 {
		sec = sec[:j+3]
	}
	requireIn(t, "Planning Record", sec, []string{
		"`proposals: <n>`", "a return that lists one proposal reads `proposals: 1`",
		"`proposals: unknown (no field in the return)`", "a return without the field", "never `none`",
	})
}
