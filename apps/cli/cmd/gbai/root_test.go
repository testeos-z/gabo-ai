package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
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
	if want := []string{"init", "plan", "review", "version"}; !slices.Equal(got, want) {
		t.Fatalf("commands = %v, want %v", got, want)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
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
	want := []string{"/etc/gbai/gbai.yaml", "/x/gbai/config.yaml", "/src/p/gbai.yaml"}
	for i, l := range got {
		if l.Path != want[i] {
			t.Fatalf("layer %d = %+v, want path %s", i, l, want[i])
		}
	}
	got = configLayers(context.Background(), r, func(string) string { return "" }, func() (string, error) { return "", errors.New("gone") })
	if got[2].Path != "" {
		t.Fatalf("project layer without a cwd = %+v", got[2])
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

func TestInitDryRunChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"init", "--dry-run", "--runtime", "both"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{"would install " + home + "/.claude/skills/gabo-setup/SKILL.md", "would install " + home + "/.pi/agent/skills/gabo-coder/SKILL.md"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q:\n%s", want, out.String())
		}
	}
	// The reviewer skill is installed only at user level by its own step
	// (T041), never by the catalog install, so PR code cannot change it.
	if strings.Contains(out.String(), "gabo-reviewer/") {
		t.Fatalf("init installs the reviewer skill with the catalog:\n%s", out.String())
	}
	if strings.Contains(strings.ToLower(out.String()), "mcp") {
		t.Fatalf("init still registers an MCP server:\n%s", out.String())
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("dry run wrote %v", entries)
	}
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
