package skills

import (
	"os"
	"path/filepath"
	"testing"
)

// Intent 0025 S6 (plan T4): the planning record's `time` is a full UTC time
// from `date`, run when the record is saved, never a date alone.
func TestPlanningRecordTimeIsAFullUTCTime(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		"`date -u +%Y-%m-%dT%H:%M:%SZ`",
		"the `time` line is never a date alone",
	})
	b, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-assistant/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != doc {
		t.Error("plugins/codex/skills/gabo-assistant/SKILL.md differs from the catalog file")
	}
}
