package main

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/agentspec"
	"github.com/testeos-z/gbai-team/apps/cli/internal/skills"
)

const checkConfig = `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
parallelism: 3
review:
  checks: [go test ./...]
  human_only_paths: [.github/]
slots:
  tasks:
    adapter: .gbai/skills/gabo-contract-tasks/examples/tasks-gh.md
  repository_host:
    adapter: .gbai/skills/gabo-contract-repository-host/examples/host-github.md
  memory:
    adapter: .gbai/skills/gabo-contract-memory/examples/memory-engram.md
  methodology:
    adapter: .gbai/skills/gabo-contract-methodology/examples/methodology-tdd.md
  reviewer_placement:
    adapter: .gbai/skills/gabo-contract-reviewer-placement/examples/reviewer-local.md
`

// checkProject runs the real `gbai init` at version, then adds gbai.yaml, a
// Claude Code runtime skill and every agent of the embedded spec built with
// version, so `gbai check` passes on it; the slots point at the bundled
// example adapters in the catalog copy.
func checkProject(t *testing.T, version string) string {
	t.Helper()
	project := t.TempDir()
	if out, err := runInit(t, project, version); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	write := func(rel, data string) {
		t.Helper()
		p := filepath.Join(project, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("gbai.yaml", checkConfig)
	write(".gbai/runtimes/gabo-runtime-claude/SKILL.md", "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\n")
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range spec.IDs() {
		write(".claude/agents/"+id+".md", "---\nname: "+id+"\ngabo-version: "+version+"\n---\n")
	}
	return project
}

// runCheck runs `gbai check` in project with a temporary user config dir.
func runCheck(t *testing.T, project, version string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GABO_UPGRADE_PR", "")
	t.Chdir(project)
	var out bytes.Buffer
	root := newRootCommand(deps{version: version})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(append([]string{"check"}, args...))
	err := root.Execute()
	return out.String(), err
}

func exitOf(err error) int {
	if err == nil {
		return 0
	}
	return exitCode(&bytes.Buffer{}, err)
}

// modes maps every path under dir to its size and modification time.
func modes(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		m[p] = info.Mode().String() + " " + info.ModTime().Format(time.RFC3339Nano)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCheckPassesAndWritesNothing(t *testing.T) {
	project := checkProject(t, "1.2.3")
	before, beforeTimes := snapshot(t, project), modes(t, project)
	out, err := runCheck(t, project, "1.2.3")
	if exitOf(err) != 0 {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
	}
	if !strings.Contains(out, "check passed") {
		t.Fatalf("output:\n%s", out)
	}
	if !maps.Equal(before, snapshot(t, project)) || !maps.Equal(beforeTimes, modes(t, project)) {
		t.Fatal("check changed the project tree or modification times")
	}
}

func TestCheckFailureExitsOne(t *testing.T) {
	project := checkProject(t, "1.2.3")
	if err := os.Remove(filepath.Join(project, ".claude", "agents", "gabo-coder.md")); err != nil {
		t.Fatal(err)
	}
	out, err := runCheck(t, project, "1.2.3")
	if exitOf(err) != 1 || !strings.Contains(out, "✗ agent gabo-coder missing") {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
	}
}

func TestCheckMissingConfigExitsOne(t *testing.T) {
	out, err := runCheck(t, t.TempDir(), "1.2.3")
	if exitOf(err) != 1 || !strings.Contains(out, "✗ no gbai.yaml found\nnext: run gabo init, then ask your agent to configure gabo-ai") {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
	}
}

// An older stamp is a warning everywhere, also with CI set.
func TestCheckOlderStampWarnsLocallyAndInCI(t *testing.T) {
	for _, ci := range []string{"", "true"} {
		t.Run("CI="+ci, func(t *testing.T) {
			t.Setenv("CI", ci)
			project := checkProject(t, "1.2.3")
			out, err := runCheck(t, project, "1.4.0")
			if exitOf(err) != 0 || !strings.Contains(out, "! this project uses gabo 1.2.3; run gabo init to upgrade it") {
				t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
			}
		})
	}
}

func TestCheckUpgradePRValidatesOnlyTheConfig(t *testing.T) {
	project := checkProject(t, "1.2.3")
	if err := os.RemoveAll(filepath.Join(project, ".claude")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(project)
	t.Setenv("GABO_UPGRADE_PR", "1")
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.0.0"})
	root.SetOut(&out)
	root.SetArgs([]string{"check"})
	err := root.Execute()
	if exitOf(err) != 0 || out.String() != "upgrade PR: only config validated, human merge required\n" {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out.String())
	}
}

func TestCheckUsageErrorsExitTwo(t *testing.T) {
	project := checkProject(t, "1.2.3")
	for _, args := range [][]string{{"extra"}, {"--bogus"}, {"--config", "gbai.yaml"}} {
		_, err := runCheck(t, project, "1.2.3", args...)
		if exitOf(err) != 2 {
			t.Fatalf("check %v: exit %d (%v), want 2", args, exitOf(err), err)
		}
	}
}

func TestCheckReadErrorExitsTwo(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads unreadable files")
	}
	project := checkProject(t, "1.2.3")
	cfg := filepath.Join(project, "gbai.yaml")
	if err := os.Chmod(cfg, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(cfg, 0o644) })
	_, err := runCheck(t, project, "1.2.3")
	if exitOf(err) != 2 || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("exit %d: %v", exitOf(err), err)
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()
	var w bytes.Buffer
	if got := exitCode(&w, errors.New("boom")); got != 1 || w.String() != "gbai: boom\n" {
		t.Fatalf("plain error: %d %q", got, w.String())
	}
}

// The spec comes from the project's catalog copy, not the binary: a newer CLI
// passes on an older project whose spec lacks an agent the embedded spec has.
func TestCheckNewerCLIReadsTheProjectSpec(t *testing.T) {
	project := checkProject(t, "1.2.3")
	specPath := filepath.Join(project, ".gbai", "skills", filepath.FromSlash(agentspec.File))
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	older, _, ok := strings.Cut(string(b), "  - id: gabo-reviewer\n")
	if !ok {
		t.Fatal("embedded spec has no gabo-reviewer entry to drop")
	}
	if err := os.WriteFile(specPath, []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(project, ".claude", "agents", "gabo-reviewer.md")); err != nil {
		t.Fatal(err)
	}
	out, err := runCheck(t, project, "1.4.0")
	if exitOf(err) != 0 || strings.Contains(out, "gabo-reviewer") {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
	}
}
