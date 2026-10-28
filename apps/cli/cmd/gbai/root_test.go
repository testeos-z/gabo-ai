package main

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
	"github.com/testeos-z/gbai-team/apps/cli/internal/setupprompt"
	"github.com/testeos-z/gbai-team/apps/cli/internal/skills"
)

// The work-time commands were retired (T042, D18): the CLI is an agent
// compiler, and `review` and `plan` remain only until T043 retires them.
func TestRootListsOnlyTheCompilerCommands(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	var got []string
	for _, c := range root.Commands() {
		got = append(got, c.Name())
	}
	if want := []string{"check", "init", "plan", "review", "version"}; !slices.Equal(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "  check ") {
		t.Fatalf("help does not list check:\n%s", out.String())
	}
	for _, removed := range []string{"secrets", "provision", "repo", "sessions", "status", "audit", "doctor", "mcp", "worktree", "config "} {
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
	if got := out.String(); got != "gbai 1.2.3\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestConfigLayersUseTheGitTopLevel(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("/src/p\n")}}
	env := map[string]string{"XDG_CONFIG_HOME": "/x"}
	got := configLayers(context.Background(), r, func(k string) string { return env[k] }, func() (string, error) { return "/src/p/sub", nil })
	want := []string{"/x/gbai/config.yaml", "/src/p/gbai.yaml"}
	if len(got) != len(want) {
		t.Fatalf("layers = %+v, want only the user and project layers", got)
	}
	for i, l := range got {
		if l.Path != want[i] {
			t.Fatalf("layer %d = %+v, want path %s", i, l, want[i])
		}
	}
	got = configLayers(context.Background(), r, func(string) string { return "" }, func() (string, error) { return "", errors.New("gone") })
	if got[1].Path != "" {
		t.Fatalf("project layer without a cwd = %+v", got[1])
	}
}

func TestReviewNotifyNeedsNoConfig(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetArgs([]string{"review", "notify", "--to", "bad", "--pr", "1", "--sha", "x"})
	err := root.Execute()
	if err == nil || strings.Contains(err.Error(), "config") {
		t.Fatalf("Execute() error = %v, want validation error without config", err)
	}
}

// runInit runs `gbai init` in a temporary project with a temporary HOME and
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
		got, err := os.ReadFile(filepath.Join(project, ".gbai", "skills", filepath.FromSlash(f.Path)))
		if err != nil || !bytes.Equal(got, f.Data) {
			t.Fatalf("%s not copied byte for byte: %v", f.Path, err)
		}
	}
	// The reviewer skill and its scripts never enter the repository (D20).
	if _, err := os.Stat(filepath.Join(project, ".gbai", "skills", "gabo-reviewer")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("gabo-reviewer copied into the project: %v", err)
	}
	if stamp, _ := os.ReadFile(filepath.Join(project, ".gbai", "VERSION")); string(stamp) != "1.2.3\n" {
		t.Fatalf("stamp = %q", stamp)
	}
	entries, _ := os.ReadDir(project)
	if len(entries) != 1 || entries[0].Name() != ".gbai" {
		t.Fatalf("project holds %v, want only .gbai", entries)
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
	cfg := []byte("repo:\n  name: o/r\n")
	if err := os.WriteFile(filepath.Join(project, "gbai.yaml"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runInit(t, project, "1.2.3"); err != nil {
		t.Fatalf("restore on missing stamp: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(project, ".gbai", "VERSION"), []byte("9.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, project)
	_, err := runInit(t, project, "1.2.3")
	if err == nil || !strings.Contains(err.Error(), "go install github.com/testeos-z/gbai-team/apps/cli/cmd/gbai@v9.0.0") {
		t.Fatalf("err = %v", err)
	}
	if after := snapshot(t, project); !maps.Equal(before, after) {
		t.Fatal("newer stamp changed the project")
	}
	if got, _ := os.ReadFile(filepath.Join(project, "gbai.yaml")); !bytes.Equal(got, cfg) {
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
		b, err := os.ReadFile(p)
		m[p] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestConfigWarningsGoToStderr(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	printWarnings(&errOut, []string{"a: ignored", "b: ignored"})
	if got, want := errOut.String(), "gbai: warning: a: ignored\ngbai: warning: b: ignored\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
	printWarnings(nil, []string{"no writer must not panic"})
}

func TestProjectPersonalKeyWarnsOnceOnStderr(t *testing.T) {
	example, err := os.ReadFile("../../gbai.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/gbai.yaml", append(example, []byte("\nalias: Leo\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(dir)
	var out, errOut bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3", stderr: &errOut})
	root.SetOut(&out)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	// review ci cleanup loads the config before it validates the workspace
	// name; the command's own outcome does not matter here.
	root.SetArgs([]string{"review", "ci", "cleanup", "--dir", "../outside"})
	_ = root.Execute()
	if got := strings.Count(errOut.String(), `personal key "alias" is ignored`); got != 1 {
		t.Fatalf("warning printed %d times, want 1:\n%s", got, errOut.String())
	}
}

func TestConfigLoaderPrintsWarningsOncePerProcess(t *testing.T) {
	t.Parallel()
	var errOut bytes.Buffer
	calls := 0
	resolve := newWarnOnceResolver(&errOut, func(context.Context) (config.Resolution, error) {
		calls++
		return config.Resolution{Warnings: []string{"p: ignored"}}, nil
	})
	for range 3 {
		if _, err := resolve(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Fatalf("resolver ran %d times, want 3 (config is re-read on each load)", calls)
	}
	if got, want := errOut.String(), "gbai: warning: p: ignored\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

// A committed .gbai symlink must not let init delete or write outside the
// project.
func TestInitRefusesSymlinkedGbaiDir(t *testing.T) {
	project, outside := t.TempDir(), t.TempDir()
	keep := filepath.Join(outside, "skills", "mine", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(project, ".gbai")); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, outside)
	if _, err := runInit(t, project, "1.2.3"); err == nil {
		t.Fatal("init accepted a symlinked .gbai")
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
	dir := filepath.Join(data, "gbai", "skills", "gabo-reviewer")
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
		cfg := []byte("repo:\n  name: o/r\n")
		if err := os.WriteFile(filepath.Join(project, "gbai.yaml"), cfg, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(project, ".gbai"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, ".gbai", "VERSION"), []byte("1.2.3\n"), 0o644); err != nil {
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
		assertUserReviewer(t, filepath.Join(home, ".local", "share", "gbai", "skills", "gabo-reviewer"), "1.2.3")
		if after := snapshot(t, project); !maps.Equal(before, after) {
			t.Fatalf("XDG_DATA_HOME=%q: matching stamp changed the project", xdg)
		}
	}
}

// An older user copy is replaced, dropping files the embedded skill no longer
// has; a second run leaves the current copy untouched.
func TestInitReplacesOlderUserReviewerThenLeavesItAlone(t *testing.T) {
	project, data := t.TempDir(), t.TempDir()
	dir := filepath.Join(data, "gbai", "skills", "gabo-reviewer")
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
