package skills

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0030 S3 (plan T9): the assistant's trigger rule names the path list
// of agent configuration, says where and when a run of the eval set happens,
// and names the gabo-eval-runner skill instead of re-implementing it.
func TestAssistantTriggerRuleNamesThePathListAndTheRun(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		"## Agent Eval Trigger (intent 0030 S3)",
		"a `gabo-*` skill",
		"an agent definition",
		"the methodology adapter",
		"the instruction file",
		"`gabo-eval-runner`",
		"on the owner's machine before the owner accepts the change",
		"started by the owner or by you",
		"`eval: not run: <reason>`",
		"a PR that touches only product code needs no run",
		"never in CI",
	})
}

// Intent 0030 S3: no workflow under .github/ starts a run of the eval set.
func TestNoWorkflowRunsTheEvalRunner(t *testing.T) {
	t.Parallel()
	root := filepath.Join(repoRoot, ".github")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, name := range []string{"gabo-eval-runner", "eval-set.sh", "eval-set"} {
			if strings.Contains(string(b), name) {
				t.Errorf("%s names the eval runner (%q): intent 0030 S3 runs it on the owner's machine, never in CI", path, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
