package skills

import (
	"strings"
	"testing"
)

// Intent 0022 S5 and S6 (plan T4): the assistant runs a merged pm and dev-lead
// trial only on the owner's approval and reports three measures against the
// baseline, as advice (PD12, PD14).
func TestAssistantRoleTrial(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	i := strings.Index(doc, "## Role Trial")
	if i < 0 {
		t.Fatal("gabo-assistant/SKILL.md: missing the Role Trial section")
	}
	if p := strings.Index(doc, "## Planning Record"); p < 0 || i < p {
		t.Error("Role Trial must come after Planning Record")
	}
	if q := strings.Index(doc, "## Questions by Mode"); q < 0 || i > q {
		t.Error("Role Trial must come before Questions by Mode")
	}
	sec := doc[i:]
	if j := strings.Index(sec[3:], "\n## "); j >= 0 {
		sec = sec[:j+3]
	}
	requireIn(t, "Role Trial", sec, []string{
		"only on the owner's approval naming an intent",
		"never chooses an intent",
		"one generic subagent that loads `gabo-pm` and `gabo-dev-lead`",
		"`agents.yaml`, the neutral templates and every runtime file stay unchanged",
		"`roles: merged (trial)`",
		"under the intent's own topics",
		"fix rounds, traceability gaps and plan-change count",
		"higher than every value of the baseline runs",
		"highest per-task count in the wave that implements",
		"read from the planning record",
		"earlier runs with separate roles",
		"`unknown (no baseline recorded)`",
		"neither worse nor not worse",
		"recommends keeping the roles separate",
		"leaves the decision to the owner",
		"`save decision`",
		"removes, merges or renames no role",
		"no numeric threshold",
		"3 to 5 waves (PRD section 11)",
	})
	for _, p := range []string{"denied board moves", "merged tasks still open", "interaction count with reasons", "plan-change proposals"} {
		if strings.Contains(strings.ToLower(sec), p) {
			t.Errorf("Role Trial repeats wave-report phrase %q", p)
		}
	}
}

func TestAgentSpecKeepsPlanningRolesSeparate(t *testing.T) {
	t.Parallel()
	spec := readCatalog(t, "gabo-agent-templates/agents.yaml")
	for _, s := range []string{"\n  - id: gabo-pm\n", "\n  - id: gabo-dev-lead\n"} {
		if strings.Count(spec, s) != 1 {
			t.Errorf("agents.yaml: want exactly one %q", strings.TrimSpace(s))
		}
	}
}
