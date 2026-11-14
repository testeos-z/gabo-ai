package skills

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

const evalRunnerSkill = "gabo-eval-runner"

// Intent 0030 S1, S2, S11, S12 (plan T2): the eval runner is a catalog skill
// with its script, not a gabo command. The script ships with the catalog, and
// the project's .gabo/skills/ copy and the Codex plugin carry byte-identical
// copies of the skill and the script.
func TestEvalRunnerShipsItsScriptAndCopies(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"SKILL.md", "scripts/eval-set.sh", "scripts/tests/eval-set.test.sh"} {
		want := evalRunnerSkill + "/" + rel
		var data []byte
		for _, f := range files {
			if f.Path == want {
				data = f.Data
			}
		}
		if data == nil {
			t.Fatalf("%s is not in Files(): add it to the //go:embed lines of skills.go", want)
		}
		for _, dir := range []string{".gabo/skills", "plugins/codex/skills"} {
			cp, err := os.ReadFile(filepath.Join(repoRoot, dir, want))
			if err != nil {
				t.Fatalf("%s copy of %s: %v", dir, want, err)
			}
			if !bytes.Equal(cp, data) {
				t.Fatalf("%s/%s differs from the catalog; refresh it with cp", dir, want)
			}
		}
	}
}

// The skill text names the task file format, the acceptance rule, the
// refusals and the commands the runner's callers rely on (S1, S2, S11).
func TestEvalRunnerSkillStatesTheRules(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(evalRunnerSkill, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	fm, ok := frontmatter(doc)
	if !ok || fm["name"] != evalRunnerSkill {
		t.Fatalf("SKILL.md: frontmatter name %q, want %q", fm["name"], evalRunnerSkill)
	}
	for _, s := range []string{
		"`evals/*.json`", "`not accepted`", "`set below minimum:", "`maximum reached:",
		"`refused <file>: missing <field>`", "token or key pattern", "`eval.min_tasks`", "`eval.max_tasks`",
		"`source_issue`", "`source_pr`", "`starting_commit`", "`spec_refs`", "`outcome_checks`", "`acceptance`",
		"eval-set.sh check evals", "eval-set.sh propose evals",
		"Never write a token", "no engine",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("SKILL.md: missing %q", s)
		}
	}
}

// The shell fixture tests run from the Go suite and need no engine.
func TestEvalRunnerShellTests(t *testing.T) {
	if testing.Short() {
		t.Skip("shell tests skipped in -short mode")
	}
	for _, tool := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found: %v", tool, err)
		}
	}
	out, err := exec.Command("bash", evalRunnerSkill+"/scripts/tests/eval-set.test.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("eval-set shell tests: %v\n%s", err, out)
	}
}

// The script reads eval.min_tasks and eval.max_tasks itself, because a shell
// script cannot call the config accessors. This pins its reading to the
// config loader (T1) for the defaults and for raised values, so the two
// cannot drift: the script must report the counts config.Load gives.
func TestEvalRunnerReadsCountsLikeTheConfigLoader(t *testing.T) {
	if testing.Short() {
		t.Skip("shell script skipped in -short mode")
	}
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skipf("jq not found: %v", err)
	}
	base, err := os.ReadFile(filepath.Join(repoRoot, "gabo.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	script, err := filepath.Abs(filepath.Join(evalRunnerSkill, "scripts", "eval-set.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, eval string }{
		{"defaults", ""},
		{"raised", "eval:\n  min_tasks: 3\n  max_tasks: 12\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "gabo.yaml")
			if err := os.WriteFile(cfgPath, append(append([]byte{}, base...), c.eval...), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(cfgPath)
			if err != nil {
				t.Fatalf("config.Load: %v", err)
			}
			min, max := cfg.EvalMinTasks(), cfg.EvalMaxTasks()
			evals := filepath.Join(dir, "evals")
			if err := os.Mkdir(evals, 0o700); err != nil {
				t.Fatal(err)
			}
			for i := 1; i <= min; i++ {
				task := fmt.Sprintf(`{"id":"eval-%03d","source_issue":821,"source_pr":849,`+
					`"starting_commit":"1b5c1a57d667744802aec1690bedd7cfe94803c2","task_input":"fixture",`+
					`"spec_refs":["S1"],"outcome_checks":["true"],`+
					`"acceptance":{"accepted_by":"owner","accepted_on":"2026-11-14"}}`, i)
				if err := os.WriteFile(filepath.Join(evals, fmt.Sprintf("eval-%03d.json", i)), []byte(task), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			out, err := exec.Command("bash", script, "check", evals, "--config", cfgPath).CombinedOutput()
			if err != nil {
				t.Fatalf("eval-set.sh check: %v\n%s", err, out)
			}
			want := fmt.Sprintf("set: %d accepted of %d tasks, minimum %d, maximum %d", min, min, min, max)
			if !strings.Contains(string(out), want+"\n") {
				t.Fatalf("eval-set.sh check output %q lacks %q (config.Load gives min %d, max %d)", out, want, min, max)
			}
		})
	}
}
