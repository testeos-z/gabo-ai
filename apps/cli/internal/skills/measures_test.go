package skills

import (
	"strings"
	"testing"
)

// Intent 0022 T1 (PD4, PD5, PD6, PD7): the wave report carries a Measures
// section 9 and the cycle trace holds the never-reset lines it reads.
func TestWaveReportHasMeasuresSection(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")

	order := []string{"**Plan-change proposals**", "**Measures**", "**Denied board moves**", "**Comfort answer**"}
	last := -1
	for _, s := range order {
		i := strings.Index(doc, s)
		if i < 0 {
			t.Errorf("gabo-wave-report: missing %q", s)
			continue
		}
		if i < last {
			t.Errorf("gabo-wave-report: %q is out of order", s)
		}
		last = i
	}
	requireIn(t, "gabo-wave-report", doc, []string{
		"9. **Measures**",
		"10. **Denied board moves**",
		"11. **Comfort answer**",
		"wall time from the wave start in the trace to the report time",
		"`cap <name> reset` lines",
		"fix rounds per task",
		"agent sessions started, by agent",
		"`gabo-assistant: 1 (the main session), gabo-coder: <n>`",
		"a coder resumed by its handle is not counted",
		"the reviewer script's runs are not counted",
		"PRs opened and merged",
		"plan-change proposals as a count and their scope",
		"`0` when there is none",
		"interactions with the owner",
		"tokens",
		"each task's outcome",
		"`unknown (<source that lacked it>)`",
		"never a guess",
		"`none` or `unknown`",
		"`/usage`",
		"OpenTelemetry",
		"`unknown (claude: not reported to the agent)`",
		"`unknown (<engine>: not verified)`",
		"Codex, OpenCode and Pi",
		"a subagent's own figure is never added into a total",
		"no numeric threshold",
	})
}

func TestCycleTraceHoldsNeverResetMeasureLines(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle", doc, []string{
		"`session started <agent> <task id> <UTC time>`",
		"every agent session it starts",
		"a resume by a coder's ledger handle starts none",
		"one line per PR it sees opened and one per PR it sees merged",
		"never reset by a \"continue\"",
		"built by `gabo-wave-report`",
	})
}
