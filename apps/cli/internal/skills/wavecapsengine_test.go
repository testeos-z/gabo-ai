package skills

import (
	"strings"
	"testing"
)

// Intent 0025 S4 (plan T7, D4): `caps --engine <engine>` only when the built
// coder agent of that engine holds the engine's turn key, `--engine none`
// otherwise; the skills compose no label (only the script prints them).
func TestCapsEngineIsTheEngineOnlyWithTheTurnKey(t *testing.T) {
	t.Parallel()
	const rule = "`<engine>` is the session engine only when the built `gabo-coder` agent of that engine holds the turn key its runtime skill names (`maxTurns:` for Claude Code, `steps:` for OpenCode), and `--engine none` otherwise"
	for _, f := range []string{"gabo-cycle/SKILL.md", "gabo-assistant/SKILL.md", "gabo-wave-report/SKILL.md"} {
		doc := readCatalog(t, f)
		requireIn(t, f, doc, []string{rule})
		if strings.Contains(doc, capsCall+" --paused` (your session engine)") || strings.Contains(doc, capsCall+"` (the session engine)") {
			t.Errorf("%s: still passes the session engine whatever the built agent holds", f)
		}
	}
}
