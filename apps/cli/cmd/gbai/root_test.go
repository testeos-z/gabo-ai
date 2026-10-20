package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
	for _, name := range []string{"version", "secrets", "provision", "review", "repo", "sessions", "status", "audit", "doctor", "mcp", "init"} {
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
	for _, want := range []string{"# step: packages", "orca ALL=(reviewer) NOPASSWD: /usr/local/libexec/gbai/review-run"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("plan missing %q:\n%s", want, out.String())
		}
	}
}

func TestResolveConfigPath(t *testing.T) {
	t.Parallel()
	if got := resolveConfigPath("custom.yaml", true); got != "custom.yaml" {
		t.Errorf("explicit = %q", got)
	}
	if got := resolveConfigPath(defaultConfigPath, true); got != defaultConfigPath {
		t.Errorf("explicit default = %q", got)
	}
	// The package directory has no gbai.yaml, so the server config is used.
	if got := resolveConfigPath(defaultConfigPath, false); got != "/etc/gbai/gbai.yaml" {
		t.Errorf("fallback = %q", got)
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
