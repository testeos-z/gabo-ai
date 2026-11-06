package initcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/osfs"
)

func TestCommandHasNoFlagsAndNoArgs(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--runtime", "pi"}, {"--force"}, {"extra"}} {
		called := false
		cmd := NewCommand(func(context.Context) (*Init, error) { called = true; return nil, nil })
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || called {
			t.Fatalf("%v: err %v, factory called %v", args, err, called)
		}
	}
}

// `gabo init --help` names the only engine files init writes and repeats
// that it never writes user-level engine config (intent 0006 S13).
func TestCommandHelpNamesTheEngineFiles(t *testing.T) {
	t.Parallel()
	called := false
	cmd := NewCommand(func(context.Context) (*Init, error) { called = true; return nil, nil })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil || called {
		t.Fatalf("err %v, factory called %v", err, called)
	}
	help := strings.Join(strings.Fields(out.String()), " ")
	for _, s := range []string{
		".claude/settings.json, .codex/config.toml, .pi/settings.json and the OpenCode configs named below are the only engine files it writes",
		"it never writes user-level engine config",
		`With runtimes.assistant: claude it also sets the top-level "agent": "gabo-assistant" key of .claude/settings.json`,
		`another value of "agent" is yours and kept`,
		`With another assistant engine it removes that exact "agent" value`,
		`With runtimes.assistant: opencode it sets "default_agent": "gabo-assistant" in the first existing of opencode.json, opencode.jsonc, .opencode/opencode.json and .opencode/opencode.jsonc, else in a new .opencode/opencode.json, comments kept`,
		`another value of "default_agent" is yours and kept`,
		`With another assistant engine it removes that exact "default_agent" value from each of those files, also when opencode is no longer in runtimes`,
		"With opencode in runtimes it also raises experimental.subagent_depth to 3 in the first existing of those files, else in a new .opencode/opencode.json, when it is unset or lower; a whole number of 3 or more is kept",
	} {
		if !strings.Contains(help, s) {
			t.Errorf("help is missing %q:\n%s", s, out.String())
		}
	}
}

// userEngineDirs points HOME and every user-level engine directory at an
// empty temporary directory and returns them, so a test can prove init
// writes nothing there (intent 0006 S2, S3, S7).
func userEngineDirs(t *testing.T) map[string]string {
	t.Helper()
	dirs := map[string]string{}
	for _, env := range []string{"HOME", "CLAUDE_CONFIG_DIR", "CODEX_HOME", "PI_CODING_AGENT_DIR", "GENTLE_PI_AGENT_HOME"} {
		dirs[env] = t.TempDir()
		t.Setenv(env, dirs[env])
	}
	return dirs
}

func assertEmpty(t *testing.T, dirs map[string]string) {
	t.Helper()
	for env, dir := range dirs {
		if entries, err := os.ReadDir(dir); err != nil || len(entries) != 0 {
			t.Errorf("$%s holds %v (%v), want nothing", env, entries, err)
		}
	}
}

// runCommand runs `init` on a real project directory with the reviewer
// installed into an outside temporary directory, whose path the output
// shows as <reviewer>.
func runCommand(t *testing.T, project string) (string, error) {
	t.Helper()
	reviewerDir := filepath.Join(t.TempDir(), "gabo-reviewer")
	factory := func(context.Context) (*Init, error) {
		return &Init{
			Reviewer: NewReviewer(osfs.FS{}, []File{{Path: "SKILL.md", Data: []byte("reviewer")}}, ReviewerSettings{Dir: reviewerDir, ProjectDir: project, Version: "0.2.6"}),
			Catalog:  NewUseCase(osfs.FS{}, catalog, Settings{ProjectDir: project, Version: "0.2.6"}),
		}, nil
	}
	var out bytes.Buffer
	cmd := NewCommand(factory)
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	return strings.ReplaceAll(out.String(), reviewerDir, "<reviewer>"), err
}

// S9, S10: init on an empty project with a gabo.yaml naming three engines
// (three runtime roles) writes their project files, never gabo.yaml nor a
// user-level engine directory, and ends with the golden report.
func TestCommandEngineReportGolden(t *testing.T) {
	user := userEngineDirs(t)
	project := t.TempDir()
	cfg := []byte("runtimes:\n  coder: codex\n  reviewer: claude\n  assistant: pi\n")
	if err := os.WriteFile(filepath.Join(project, ConfigFile), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, project)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	golden(t, "init-engines.golden", out)
	if got, _ := os.ReadFile(filepath.Join(project, ConfigFile)); !bytes.Equal(got, cfg) {
		t.Fatalf("gabo.yaml changed: %q", got)
	}
	for _, f := range []string{engineconfig.ClaudeSettings, engineconfig.CodexConfigPath, engineconfig.PiSettings} {
		if _, err := os.Stat(filepath.Join(project, f)); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	assertEmpty(t, user)
}

// S5: a .codex symlinked to a directory outside the project gets nothing
// written through it; the other engines and the stamp are written and the
// command fails after the report.
func TestCommandRefusesSymlinkedEngineDir(t *testing.T) {
	user := userEngineDirs(t)
	project, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(project, ".codex")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ConfigFile), []byte("runtimes:\n  coder: codex\n  reviewer: claude\n  assistant: claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := runCommand(t, project)
	if !errors.Is(err, ErrEngineFailures) {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("wrote through the symlink: %v", entries)
	}
	if !strings.Contains(out, "✗ .codex/config.toml: .codex is a symlink") || strings.Count(out, "✗") != 1 {
		t.Fatalf("output:\n%s", out)
	}
	if b, _ := os.ReadFile(filepath.Join(project, StampFile)); string(b) != "0.2.6\n" {
		t.Fatalf("stamp = %q", b)
	}
	if _, err := os.Stat(filepath.Join(project, engineconfig.ClaudeSettings)); err != nil {
		t.Fatalf("claude not written: %v", err)
	}
	assertEmpty(t, user)
}
