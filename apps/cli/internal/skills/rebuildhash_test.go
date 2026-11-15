package skills

import (
	"io/fs"
	"strings"
	"testing"
)

// rebuildSkills are the six catalog skills that build or rebuild agents
// (intent 0024 S5, PD8). A seventh skill added later fails this test until it
// is listed.
var rebuildSkills = []string{
	"gabo-runtime-builder",
	"gabo-runtime-claude",
	"gabo-runtime-codex",
	"gabo-runtime-opencode",
	"gabo-runtime-pi",
	"gabo-agent-templates",
}

// Each rebuild skill hashes gabo.yaml before the rebuild, compares it after,
// reports a change with one line, and never restores or edits the file.
func TestRebuildSkillsHashGaboYaml(t *testing.T) {
	t.Parallel()
	musts := []struct{ name, text string }{
		{"hash before", "`git hash-object gabo.yaml`"},
		{"absent", "`absent`"},
		{"compare after", "compare it after the rebuild"},
		{"failure line", "`✗ gabo.yaml changed during the rebuild; next: review it with git diff gabo.yaml`"},
		{"never restores or edits", "never restore or edit `gabo.yaml`"},
		{"writes no key", "rebuild writes no key of `gabo.yaml`"},
	}
	for _, name := range rebuildSkills {
		b, err := fs.ReadFile(Catalog(), name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		doc := string(b)
		for _, m := range musts {
			if !strings.Contains(doc, m.text) {
				t.Errorf("%s/SKILL.md: missing %s: %q", name, m.name, m.text)
			}
		}
	}
}
