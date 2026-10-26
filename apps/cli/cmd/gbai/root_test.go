package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestRootHelpListsVersion(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, name := range []string{"version", "secrets", "provision", "review", "repo", "sessions", "status", "audit", "doctor", "mcp", "init", "plan", "worktree", "config"} {
		if !strings.Contains(out.String(), name) {
			t.Fatalf("help output does not list %s command:\n%s", name, out.String())
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

func TestSecretsSetFailsWithoutConfigBeforeReadingValue(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetArgs([]string{"--config", t.TempDir() + "/missing.yaml", "secrets", "set", "reviewer-pi"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "config") {
		t.Fatalf("Execute() error = %v, want config error", err)
	}
}

func TestProvisionDryRunPrintsPlanFromExampleConfig(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"--config", "../../gbai.example.yaml", "provision", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{"# step: packages", "# step: legacy", "id -u 'orca'"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("plan missing %q:\n%s", want, out.String())
		}
	}
	for _, banned := range []string{"NOPASSWD", "useradd", "reviewer-ci"} {
		if strings.Contains(out.String(), banned) {
			t.Fatalf("single-user plan contains %q:\n%s", banned, out.String())
		}
	}
}

func TestConfigCommandWithFlagShowsOnlyThatFile(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"--config", "../../gbai.example.yaml", "config"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"flag", "../../gbai.example.yaml", "resolved config:", "name: cappy-lab/vulcam"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	for _, banned := range []string{"/etc/gbai/gbai.yaml", "config.yaml"} {
		if strings.Contains(got, banned) {
			t.Fatalf("--config must not layer, output has %q:\n%s", banned, got)
		}
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

func TestRepoSetupDryRunFromExampleConfig(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"--config", "../../gbai.example.yaml", "repo", "setup", "--dry-run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{"POST repos/cappy-lab/vulcam/rulesets", "refs/heads/dev", "admins"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("plan missing %q:\n%s", want, out.String())
		}
	}
}

func TestAuditHelpSaysOnlyGitHubIsAudited(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"audit", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "Only GitHub is audited") {
		t.Fatalf("help:\n%s", out.String())
	}
}

func TestSessionsRejectsBadModeWithoutConfig(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetArgs([]string{"--config", t.TempDir() + "/missing.yaml", "sessions", "start", "--mode", "hybrid"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "mode") {
		t.Fatalf("Execute() error = %v", err)
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
	for _, want := range []string{"would install " + home + "/.claude/skills/gbai-setup/SKILL.md", "would install " + home + "/.pi/agent/skills/gbai-coder/SKILL.md", "would run: claude mcp add --scope user gbai -- "} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q:\n%s", want, out.String())
		}
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Fatalf("dry run wrote %v", entries)
	}
}

func TestOwnerLoginDefaultsToTheGhLogin(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("Owner\n")}}
	got, err := ownerLogin(context.Background(), r, "")
	if err != nil || got != "Owner" {
		t.Fatalf("ownerLogin() = %q, %v", got, err)
	}
	if c := r.Calls[0].Command; c.Name != "gh" || strings.Join(c.Args, " ") != "api user --jq .login" {
		t.Fatalf("command = %s %q", c.Name, c.Args)
	}
	r = &kerneltest.Runner{}
	if got, err := ownerLogin(context.Background(), r, "configured"); err != nil || got != "configured" || len(r.Calls) != 0 {
		t.Fatalf("configured login = %q, %v, calls %d", got, err, len(r.Calls))
	}
	for name, res := range map[string]kernel.Result{
		"gh fails": {ExitCode: 1, Stderr: []byte("not logged in")},
		"empty":    {Stdout: []byte("\n")},
		"garbage":  {Stdout: []byte("a b")},
	} {
		if _, err := ownerLogin(context.Background(), &kerneltest.Runner{Result: res}, ""); err == nil {
			t.Errorf("%s: error = nil", name)
		}
	}
}

func TestWorktreeCloseNeedsConfigAndRejectsForce(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetArgs([]string{"--config", t.TempDir() + "/missing.yaml", "worktree", "close", "/src/repo-worktrees/x"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "config") {
		t.Fatalf("Execute() error = %v, want config error", err)
	}
	root = newRootCommand(deps{version: "1.2.3"})
	root.SetArgs([]string{"--config", "../../gbai.example.yaml", "worktree", "close", "--force", "/src/repo-worktrees/x"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "force") {
		t.Fatalf("Execute(--force) error = %v", err)
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
