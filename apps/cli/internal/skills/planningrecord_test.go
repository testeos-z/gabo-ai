package skills

import (
	"strings"
	"testing"
)

// Intent 0022 S2 (plan T2): the assistant saves the planning record from the
// directiva's own return, and the memory example names its topic (PD8, PD9).
func TestAssistantSavesPlanningRecord(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	i := strings.Index(doc, "## Planning Record")
	if i < 0 {
		t.Fatal("gabo-assistant/SKILL.md: missing the Planning Record section")
	}
	if dg := strings.Index(doc, "## Decision Gates"); dg < 0 || i < dg {
		t.Error("Planning Record must come after Decision Gates")
	}
	if q := strings.Index(doc, "## Questions by Mode"); q < 0 || i > q {
		t.Error("Planning Record must come before Questions by Mode")
	}
	sec := doc[i:]
	if j := strings.Index(sec[3:], "\n## "); j >= 0 {
		sec = sec[:j+3]
	}
	requireIn(t, "Planning Record", sec, []string{
		"`gabo-pm`", "`gabo-dev-lead`", "`save wave trace`",
		"`gabo/trace/<I####>/planning`", "read it back",
		"`agent`", "`roles: separate`", "`gaps: <n> (<ids>)`", "`gaps: none`",
		"`proposals: <n>`", "`proposals: none`", "UTC time",
		"traceability gaps (or none)", "never recomputes", "unverified",
		"exempt from the intent-claim check", "never touches `gabo/trace/<run id>`",
	})
	for _, p := range []string{"denied board moves", "merged tasks still open", "interaction count with reasons", "plan-change proposals"} {
		if strings.Contains(strings.ToLower(sec), p) {
			t.Errorf("Planning Record repeats wave-report phrase %q", p)
		}
	}
}

func TestMemoryExampleNamesPlanningTopic(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-contract-memory/examples/memory-engram.md")
	requireIn(t, "memory-engram.md", doc, []string{
		"| planning record | `gabo/trace/<I####>/planning` |",
		"topic key is exactly `gabo/trace/<wave>`",
		"ignoring keys with a segment after the wave id",
		"separate record saved with the same operation",
	})
	for _, h := range []string{"### save wave trace", "### read wave trace", "### save ledger"} {
		if !strings.Contains(doc, h) {
			t.Errorf("operation heading %q changed", h)
		}
	}
}
