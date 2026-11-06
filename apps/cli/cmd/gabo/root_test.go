package main

import (
	"bytes"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/setupprompt"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// The work-time (T042) and the review and plan commands (T043) were retired
// (D18, D24): the CLI is an agent compiler with init, check, version and
// prompt (R31, R39).
func TestRootListsOnlyTheCompilerCommands(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	var got []string
	for _, c := range root.Commands() {
		got = append(got, c.Name())
	}
	if want := []string{"check", "init", "prompt", "version"}; !slices.Equal(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, name := range []string{"init", "check", "version", "prompt"} {
		if !strings.Contains(out.String(), "  "+name+" ") {
			t.Fatalf("help does not list %s:\n%s", name, out.String())
		}
	}
	for _, removed := range []string{"secrets", "provision", "repo", "sessions", "status", "audit", "doctor", "mcp", "worktree", "config ", "review", "plan"} {
		if strings.Contains(out.String(), "  "+removed) {
			t.Fatalf("help still lists %s:\n%s", removed, out.String())
		}
	}
}

func TestRootVersionSubcommand(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := out.String(); got != "gabo 1.2.3\n" {
		t.Fatalf("output = %q", got)
	}
}

// runPrompt runs `gabo prompt` through the CLI entry with separate stdout
// and stderr buffers.
func runPrompt(t *testing.T) (stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"prompt"})
	err = root.Execute()
	return out.String(), errOut.String(), err
}

// `gabo prompt` prints the embedded setup prompt byte for byte and nothing
// else, and creates nothing without a project (R39, D25).
func TestPromptPrintsTheEmbeddedPromptAndCreatesNothing(t *testing.T) {
	scratch := t.TempDir()
	dirs := map[string]string{}
	for _, name := range []string{"cwd", "HOME", "XDG_DATA_HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME"} {
		dirs[name] = filepath.Join(scratch, name)
		if err := os.Mkdir(dirs[name], 0o755); err != nil {
			t.Fatal(err)
		}
		if name != "cwd" {
			t.Setenv(name, dirs[name])
		}
	}
	t.Chdir(dirs["cwd"])
	stdout, stderr, err := runPrompt(t)
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}
	if stdout != setupprompt.Text {
		t.Fatalf("stdout differs from the embedded prompt:\n%q", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q", stderr)
	}
	for name, dir := range dirs {
		if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
			t.Fatalf("%s holds %v (%v), want empty", name, entries, err)
		}
	}
}

// Inside a project `gabo prompt` changes nothing: no catalog, stamp, config
// or user-level reviewer write, no self-update.
func TestPromptLeavesAProjectUntouched(t *testing.T) {
	project := t.TempDir()
	if _, err := runInit(t, project, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "gabo.yaml"), []byte("repo:\n  name: o/r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home, data := os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME")
	before := []map[string]string{snapshot(t, project), snapshot(t, home), snapshot(t, data)}
	stdout, stderr, err := runPrompt(t)
	if err != nil || stdout != setupprompt.Text || stderr != "" {
		t.Fatalf("prompt: %v, stderr %q, stdout:\n%s", err, stderr, stdout)
	}
	after := []map[string]string{snapshot(t, project), snapshot(t, home), snapshot(t, data)}
	for i := range before {
		if !maps.Equal(before[i], after[i]) {
			t.Fatalf("prompt changed %v", []string{project, home, data}[i])
		}
	}
}

func TestPromptRefusesArguments(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"prompt", "extra"})
	if err := root.Execute(); err == nil || strings.Contains(out.String(), setupprompt.Text) {
		t.Fatalf("prompt with an argument: %v\n%s", err, out.String())
	}
}

// runInit runs `gabo init` in a temporary project with a temporary HOME and
// XDG_DATA_HOME, so the user-level reviewer install never reaches real data.
func runInit(t *testing.T, project, version string) (string, error) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Chdir(project)
	var out bytes.Buffer
	root := newRootCommand(deps{version: version})
	root.SetOut(&out)
	root.SetArgs([]string{"init"})
	err := root.Execute()
	return out.String(), err
}

func TestInitCopiesEmbeddedCatalogWithoutReviewer(t *testing.T) {
	project := t.TempDir()
	out, err := runInit(t, project, "1.2.3")
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	embedded, err := skills.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range embedded {
		got, err := os.ReadFile(filepath.Join(project, ".gabo", "skills", filepath.FromSlash(f.Path)))
		if err != nil || !bytes.Equal(got, f.Data) {
			t.Fatalf("%s not copied byte for byte: %v", f.Path, err)
		}
	}
	// The reviewer skill and its scripts never enter the repository (D20).
	if _, err := os.Stat(filepath.Join(project, ".gabo", "skills", "gabo-reviewer")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("gabo-reviewer copied into the project: %v", err)
	}
	if stamp, _ := os.ReadFile(filepath.Join(project, ".gabo", "VERSION")); string(stamp) != "1.2.3\n" {
		t.Fatalf("stamp = %q", stamp)
	}
	entries, _ := os.ReadDir(project)
	if len(entries) != 1 || entries[0].Name() != ".gabo" {
		t.Fatalf("project holds %v, want only .gabo", entries)
	}
	// The setup prompt follows the next step verbatim and is written nowhere (D25).
	if !strings.HasSuffix(out, "next: open your agent in this project and paste this prompt\n"+setupprompt.Text) {
		t.Fatalf("output:\n%s", out)
	}
	if out, err := runInit(t, project, "1.2.3"); err != nil || strings.Contains(out, setupprompt.Text) {
		t.Fatalf("second init printed the prompt or failed: %v\n%s", err, out)
	}
}

func TestInitNewerStampExitsWithUpgradeCommand(t *testing.T) {
	project := t.TempDir()
	cfg := []byte("repo:\n  name: o/r\nruntimes:\n  coder: opencode\n  reviewer: opencode\n  assistant: opencode\n")
	if err := os.WriteFile(filepath.Join(project, "gabo.yaml"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runInit(t, project, "1.2.3"); err != nil {
		t.Fatalf("restore on missing stamp: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(project, ".gabo", "VERSION"), []byte("9.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, project)
	_, err := runInit(t, project, "1.2.3")
	if err == nil || !strings.Contains(err.Error(), "go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v9.0.0") {
		t.Fatalf("err = %v", err)
	}
	if after := snapshot(t, project); !maps.Equal(before, after) {
		t.Fatal("newer stamp changed the project")
	}
	if got, _ := os.ReadFile(filepath.Join(project, "gabo.yaml")); !bytes.Equal(got, cfg) {
		t.Fatalf("config changed: %q", got)
	}
}

// snapshot maps every file under dir to its content.
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			target, err := os.Readlink(p)
			m[p] = "-> " + target
			return err
		}
		b, err := os.ReadFile(p)
		m[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// A committed .gabo symlink must not let init delete or write outside the
// project.
func TestInitRefusesSymlinkedGaboDir(t *testing.T) {
	project, outside := t.TempDir(), t.TempDir()
	keep := filepath.Join(outside, "skills", "mine", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, ".gabo")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, outside)
	if _, err := runInit(t, project, "1.2.3"); err == nil {
		t.Fatal("init accepted a symlinked .gabo")
	}
	if after := snapshot(t, outside); !maps.Equal(before, after) {
		t.Fatalf("outside changed:\n%v\n%v", before, after)
	}
}

// assertUserReviewer checks the user copy holds the embedded gabo-reviewer
// skill byte for byte plus its own VERSION file.
func assertUserReviewer(t *testing.T, dir, version string) {
	t.Helper()
	files, err := skills.ReviewerFiles()
	if err != nil || len(files) == 0 {
		t.Fatalf("ReviewerFiles: %d files, %v", len(files), err)
	}
	for _, f := range files {
		rel := strings.TrimPrefix(f.Path, "gabo-reviewer/")
		got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil || !bytes.Equal(got, f.Data) {
			t.Fatalf("%s not installed byte for byte: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "scripts", "reviewer.sh")); err != nil {
		t.Fatalf("scripts/ not installed: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "VERSION")); string(got) != version+"\n" {
		t.Fatalf("user copy VERSION = %q", got)
	}
	// The reviewer preflight refuses a copy writable by group or others.
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err == nil && info.Mode().Perm()&0o022 != 0 {
			t.Errorf("%s is writable by group or others: %v", p, info.Mode())
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInitInstallsReviewerUnderXDGDataHome(t *testing.T) {
	project, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", data)
	t.Chdir(project)
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"init"})
	if err := root.Execute(); err != nil {
		t.Fatalf("init: %v\n%s", err, out.String())
	}
	dir := filepath.Join(data, "gabo", "skills", "gabo-reviewer")
	assertUserReviewer(t, dir, "1.2.3")
	if !strings.HasPrefix(out.String(), "reviewer skill installed to "+dir+" (1.2.3)\n") {
		t.Fatalf("output:\n%s", out.String())
	}
}

// Without XDG_DATA_HOME (or with a relative one) the copy goes to
// ~/.local/share; a configured project with a matching stamp still gets it.
func TestInitInstallsReviewerUnderHomeWhenXDGUnsetOrRelative(t *testing.T) {
	for _, xdg := range []string{"", "relative/data"} {
		project, home := t.TempDir(), t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("XDG_DATA_HOME", xdg)
		t.Chdir(project)
		cfg := []byte("repo:\n  name: o/r\nruntimes:\n  coder: opencode\n  reviewer: opencode\n  assistant: opencode\n")
		if err := os.WriteFile(filepath.Join(project, "gabo.yaml"), cfg, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(project, ".gabo"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, ".gabo", "VERSION"), []byte("1.2.3\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		// The assistant is OpenCode, so a config already holding gabo's
		// default_agent keeps init from writing to the project.
		if err := os.MkdirAll(filepath.Join(project, ".opencode"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, ".opencode", "opencode.json"), []byte("{\"default_agent\": \"gabo-assistant\"}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		before := snapshot(t, project)
		var out bytes.Buffer
		root := newRootCommand(deps{version: "1.2.3"})
		root.SetOut(&out)
		root.SetArgs([]string{"init"})
		if err := root.Execute(); err != nil {
			t.Fatalf("XDG_DATA_HOME=%q: %v\n%s", xdg, err, out.String())
		}
		assertUserReviewer(t, filepath.Join(home, ".local", "share", "gabo", "skills", "gabo-reviewer"), "1.2.3")
		if after := snapshot(t, project); !maps.Equal(before, after) {
			t.Fatalf("XDG_DATA_HOME=%q: matching stamp changed the project", xdg)
		}
	}
}

// An older user copy is replaced, dropping files the embedded skill no longer
// has; a second run leaves the current copy untouched.
func TestInitReplacesOlderUserReviewerThenLeavesItAlone(t *testing.T) {
	project, data := t.TempDir(), t.TempDir()
	dir := filepath.Join(data, "gabo", "skills", "gabo-reviewer")
	if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"VERSION": "1.0.0\n", "scripts/gone.sh": "old"} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", data)
	t.Chdir(project)
	for i, want := range []string{"reviewer skill replaced 1.0.0 → 1.2.3 in " + dir, "reviewer skill is current (1.2.3) in " + dir} {
		var out bytes.Buffer
		root := newRootCommand(deps{version: "1.2.3"})
		root.SetOut(&out)
		root.SetArgs([]string{"init"})
		before := snapshot(t, dir)
		if err := root.Execute(); err != nil {
			t.Fatalf("run %d: %v\n%s", i, err, out.String())
		}
		if !strings.HasPrefix(out.String(), want+"\n") {
			t.Fatalf("run %d output:\n%s", i, out.String())
		}
		assertUserReviewer(t, dir, "1.2.3")
		if _, err := os.Stat(filepath.Join(dir, "scripts", "gone.sh")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("run %d: stale script kept: %v", i, err)
		}
		if i == 1 && !maps.Equal(before, snapshot(t, dir)) {
			t.Fatal("current user copy changed")
		}
	}
}
