package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
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
    adapter: .gabo/skills/gabo-contract-tasks/examples/tasks-gh.md
  repository_host:
    adapter: .gabo/skills/gabo-contract-repository-host/examples/host-github.md
  memory:
    adapter: .gabo/skills/gabo-contract-memory/examples/memory-engram.md
  methodology:
    adapter: .gabo/skills/gabo-contract-methodology/examples/methodology-tdd.md
  reviewer_placement:
    adapter: .gabo/skills/gabo-contract-reviewer-placement/examples/reviewer-local.md
`

// checkProject runs the real `gabo init` at version, then adds gabo.yaml, a
// Claude Code runtime skill and every agent of the embedded spec built with
// version, so `gabo check` passes on it; the slots point at the bundled
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
	write("gabo.yaml", checkConfig)
	// With gabo.yaml in place, gabo init again writes the engine keys gabo
	// check verifies (intent 0006 S11).
	if out, err := runInit(t, project, version); err != nil {
		t.Fatalf("init after gabo.yaml: %v\n%s", err, out)
	}
	write(".gabo/runtimes/gabo-runtime-claude/SKILL.md", "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\n")
	// The catalog copy's gabo-runtime-claude wins over .gabo/runtimes: its
	// skill links and reviewer link are made below.
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		// The skills list of gabo-runtime-claude: spec skills, then the
		// fixed-name links the template names (intent 0008 S1).
		tmpl, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, a.Template))
		if err != nil {
			t.Fatal(err)
		}
		list := slices.Clone(a.Skills)
		for _, l := range [][2]string{{"{{methodology}}", "gabo-methodology"}, {"{{adapter:tasks}}", "gabo-tasks"}, {"{{adapter:repository_host}}", "gabo-repository-host"}, {"{{adapter:memory}}", "gabo-memory"}} {
			if strings.Contains(string(tmpl), l[0]) {
				list = append(list, l[1])
			}
		}
		write(".claude/agents/"+a.ID+".md", "---\nname: "+a.ID+"\nskills: ["+strings.Join(list, ", ")+"]\ngabo-version: "+version+"\n---\n")
		for _, name := range a.Skills {
			link(t, "../../.gabo/skills/"+name, filepath.Join(project, ".claude", "skills", name))
		}
	}
	// The main-session entry gabo-runtime-claude builds for runtimes.assistant
	// (spec 0009 S1, S2): link, agent file and the one agent key.
	m := spec.MainSession
	for _, name := range m.Skills {
		link(t, "../../.gabo/skills/"+name, filepath.Join(project, ".claude", "skills", name))
	}
	write(".claude/agents/"+m.ID+".md", "---\nname: "+m.ID+"\ngabo-version: "+version+"\n---\n")
	b, err := os.ReadFile(filepath.Join(project, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]any{}
	if err := json.Unmarshal(b, &settings); err != nil {
		t.Fatal(err)
	}
	settings["agent"] = m.ID
	if b, err = json.MarshalIndent(settings, "", "  "); err != nil {
		t.Fatal(err)
	}
	write(".claude/settings.json", string(b)+"\n")
	// The user-level reviewer link into the copy runInit installed under the
	// temporary HOME and XDG_DATA_HOME.
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	dir, err := reviewerSkillDir(os.Getenv)
	if err != nil {
		t.Fatal(err)
	}
	link(t, dir, filepath.Join(os.Getenv("HOME"), ".claude", "skills", "gabo-reviewer"))
	return project
}

// link makes the symlink name -> target, replacing an existing link.
func link(t *testing.T, target, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(name); err == nil {
		return
	}
	if err := os.Symlink(target, name); err != nil {
		t.Fatal(err)
	}
}

// runCheck runs `gabo check` in project with a temporary user config dir.
func runCheck(t *testing.T, project, version string, args ...string) (string, error) {
	t.Helper()
	return runCheckStdin(t, project, version, "", args...)
}

// runCheckStdin runs `gabo check args...` like runCheck with stdin as its
// standard input.
func runCheckStdin(t *testing.T, project, version, stdin string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GABO_UPGRADE_PR", "")
	t.Chdir(project)
	var out bytes.Buffer
	root := newRootCommand(deps{version: version})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(stdin))
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

// #135: a missing skill link or reviewer link fails the check instead of
// passing it.
func TestCheckMissingLinksExitOne(t *testing.T) {
	t.Setenv("CI", "")
	project := checkProject(t, "1.2.3")
	if err := os.Remove(filepath.Join(project, ".claude", "skills", "gabo-reviewer-notes")); err != nil {
		t.Fatal(err)
	}
	reviewer := filepath.Join(os.Getenv("HOME"), ".claude", "skills", "gabo-reviewer")
	if err := os.Remove(reviewer); err != nil {
		t.Fatal(err)
	}
	out, err := runCheck(t, project, "1.2.3")
	for _, want := range []string{
		"✗ skill link .claude/skills/gabo-reviewer-notes missing",
		"✗ reviewer link " + reviewer + " missing",
	} {
		if exitOf(err) != 1 || !strings.Contains(out, want) {
			t.Fatalf("exit %d: %v, want %q in:\n%s", exitOf(err), err, want, out)
		}
	}
}

func TestCheckMissingConfigExitsOne(t *testing.T) {
	out, err := runCheck(t, t.TempDir(), "1.2.3")
	if exitOf(err) != 1 || !strings.Contains(out, "✗ no gabo.yaml found\nnext: run gabo init, then ask your agent to configure gabo-ai") {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
	}
}

// A project config or global dir still named gabo fails with the rename
// instead of being read silently (TRD D26).
func TestCheckFormerNamesFail(t *testing.T) {
	project := checkProject(t, "1.2.3")
	if err := os.WriteFile(filepath.Join(project, "gbai.yaml"), []byte(checkConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	xdg := t.TempDir()
	if err := os.MkdirAll(filepath.Join(xdg, "gbai"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(xdg, "gbai", "config.yaml"), []byte("tone: dry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GABO_UPGRADE_PR", "")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Chdir(project)
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"check"})
	err := root.Execute()
	for _, want := range []string{
		"✗ " + filepath.Join(project, "gbai.yaml") + " uses a name from before the gabo-ai rename and is not read\n" +
			"next: rename " + filepath.Join(project, "gbai.yaml") + " to " + filepath.Join(project, "gabo.yaml") + ", then rerun gabo check",
		"✗ " + filepath.Join(xdg, "gbai", "config.yaml") + " uses a name from before the gabo-ai rename and is not read\n" +
			"next: rename " + filepath.Join(xdg, "gbai") + " to " + filepath.Join(xdg, "gabo") + ", then rerun gabo check",
	} {
		if exitOf(err) != 1 || !strings.Contains(out.String(), want) {
			t.Fatalf("exit %d: %v, want %q in:\n%s", exitOf(err), err, want, out.String())
		}
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
	for _, args := range [][]string{{"extra"}, {"--bogus"}, {"--config", "gabo.yaml"}} {
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
	cfg := filepath.Join(project, "gabo.yaml")
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
	if got := exitCode(&w, errors.New("boom")); got != 1 || w.String() != "gabo: boom\n" {
		t.Fatalf("plain error: %d %q", got, w.String())
	}
}

// The spec comes from the project's catalog copy, not the binary: a newer CLI
// passes on an older project whose spec lacks an agent the embedded spec has.
func TestCheckNewerCLIReadsTheProjectSpec(t *testing.T) {
	project := checkProject(t, "1.2.3")
	specPath := filepath.Join(project, ".gabo", "skills", filepath.FromSlash(agentspec.File))
	b, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatal(err)
	}
	head, rest, ok := strings.Cut(string(b), "  - id: gabo-coder\n")
	if !ok {
		t.Fatal("embedded spec has no gabo-coder entry to drop")
	}
	_, tail, ok := strings.Cut(rest, "main_session:")
	if !ok {
		t.Fatal("embedded spec has no main_session entry")
	}
	if err := os.WriteFile(specPath, []byte(head+"main_session:"+tail), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(project, ".claude", "agents", "gabo-coder.md")); err != nil {
		t.Fatal(err)
	}
	out, err := runCheck(t, project, "1.4.0")
	if exitOf(err) != 0 || strings.Contains(out, "gabo-coder") {
		t.Fatalf("exit %d: %v\n%s", exitOf(err), err, out)
	}
}
