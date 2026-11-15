package skills

import "testing"

// Intent 0026 S6 (plan T7): the end of a wave is the wave report entry in its
// trace, never an engine event; `agent_settled` with a coder return pending
// does not end the wave.
func TestCycleWaveEndIsTheReportEntry(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"the wave ends at the wave report entry in its trace, never at an engine event",
		"an `agent_settled` while a background coder return is pending does not end the wave",
		"keeps waiting for the coder's return or the claim's recorded state",
		"a driver that stops at `agent_settled` is outside the catalog and is reported as a finding",
	})
}
