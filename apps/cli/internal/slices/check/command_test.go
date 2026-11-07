package check

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func runCommand(t *testing.T, proj fs.FS, args ...string) (string, error) {
	t.Helper()
	return runCommandConfig(t, proj, validConfig, args...)
}

// runCommandConfig runs gabo check with cfg as the project gabo.yaml.
func runCommandConfig(t *testing.T, proj fs.FS, cfg string, args ...string) (string, error) {
	t.Helper()
	s := Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return NewUseCase(proj, configFiles{projectFile: cfg}, goodUser(), s), nil
	})
	// The root command silences cobra's own error and usage output.
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	return out.String(), err
}

// Without docs/product/intents/ the command prints exactly what it printed
// before the intent check and exits 0.
func TestCommandWithoutIntentsIsUnchanged(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if out != PassedLine+"\n" {
		t.Fatalf("stdout = %q, want %q", out, PassedLine+"\n")
	}
}

func TestCommandIntentFailureExitsOne(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, withTree(t, project(), "plan-without-spec"))
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed {
		t.Fatalf("err = %v, want exit %d", err, ExitFailed)
	}
	want := "✗ " + intentsP + "0001-alpha/plan.md: no spec.md; a plan starts only after a human accepts the spec\n" + NextFixIntents + "\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestCommandHelpNamesTheIntentCheck(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	line := "  - each " + IntentsDir + "/<NNNN-slug>/ folder: its name, intent.md, an accepted"
	if !strings.Contains(out, line) {
		t.Fatalf("help lacks %q:\n%s", line, out)
	}
}

// TestCommandHelpNamesTheStricterRules pins the help lines of the intent
// 0008 rules (S1, S6, S7, S8, S4; S9 names them in the docs).
func TestCommandHelpNamesTheStricterRules(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{
		"  - each agent's gabo-version line, a version x.y.z, against " + StampFile + "; a",
		"    missing line fails, and without a valid stamp the versions are reported",
		"    as not compared, never skipped;",
		"  - for a runtime skill declaring metadata.agent_skills, each agent file lists",
		"    exactly the skills of its spec agent plus the fixed-name links its template",
		"    a full-tier runtime skill without it fails, a flexible-tier one warns;",
		"  - for a runtime skill declaring metadata.agent_name, each agent file's",
		"    declared name equals its spec id (a file that declares none passes);",
		"  - no gabo-reviewer in the project: no <skill_dir>/gabo-reviewer entry and no",
		"    agent file listing it among its skills;",
		"  - for a runtime skill declaring metadata.former_skill_dir, no gabo-* link left",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

// TestCommandHelpNamesTheGaboInitHints pins the help lines of the intent
// 0010 S6 hints: each line whose fix is gabo init names it.
func TestCommandHelpNamesTheGaboInitHints(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{
		"  - when a role runs on opencode, experimental.subagent_depth at 3 or more in",
		"    a missing or lower whole value names \"run gabo init (it sets",
		"    experimental.subagent_depth to 3 in <file>)\", any other value names setting",
		"    (a full-tier runtime skill without it fails, a flexible-tier one warns).",
		"    For a full-tier engine the key is gabo init's: a missing key names \"run gabo",
		"    init (it merges \"<key>\": \"gabo-assistant\" into <file>)\", a key holding",
		"    another value fails as \"<file>: <key> is \"<value>\", not \"gabo-assistant\"\"",
		"    and names \"change <key> to \"gabo-assistant\" in <file>, or remove it and run",
		"    gabo init\", and another engine's key names \"run gabo init (it removes",
		"    \"<key>\": \"gabo-assistant\" from <file>)\"; every other piece, and a",
		"    flexible-tier engine's key, names asking your agent to rebuild the agents;",
		"    it by hand; a file gabo init refuses names \"fix <file> (<its reason>)\";",
		"    a file gabo init refuses names \"fix <file> (<its reason>)\", and a key gabo",
		"    init would leave as it is names setting or removing it by hand;",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

// Intent 0013 S5: the help says the main-session entry is checked for every
// engine listed in runtimes.assistant, and names the stale line with a list.
func TestCommandHelpNamesEveryListedAssistant(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{
		"  - the main-session entry of every engine listed in runtimes.assistant, from",
		"    each runtime skill's metadata.main_session: each listed engine's",
		"    block; no entry of a full-tier engine that is not listed, named as \"<file>:",
		"    gabo assistant entry for <engine>, but runtimes.assistant is <the listed",
		"    engines, joined with \", \">\"",
	} {
		if !strings.Contains(out, want+"\n") {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

func runPreWrite(t *testing.T, proj fs.FS, args ...string) (string, error) {
	t.Helper()
	return runPreWriteStdin(t, proj, "", args...)
}

// runPreWriteStdin runs the command like runPreWrite with stdin as its
// standard input.
func runPreWriteStdin(t *testing.T, proj fs.FS, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return NewUseCase(proj, failingReader{}, goodUser(), Settings{Root: preRoot}), nil
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{}, args...))
	err := cmd.ExecuteContext(context.Background())
	return out.String(), err
}

// The hook mode judges only the given paths: it never reads the config
// (failingReader would fail the run) and prints only the blocked paths.
func TestCommandPreWrite(t *testing.T) {
	t.Parallel()
	out, err := runPreWrite(t, intentWith(unfilledMD), "--pre-write", "README.md", prePlan)
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed || ee.Err != nil {
		t.Fatalf("err = %v, want a silent exit %d", err, ExitFailed)
	}
	if out != notAccepted+"\n" {
		t.Fatalf("stdout = %q, want %q", out, notAccepted+"\n")
	}
	out, err = runPreWrite(t, intentWith(acceptedMD), "--pre-write", prePlan, "README.md")
	if err != nil || out != "" {
		t.Fatalf("accepted: err = %v, stdout = %q, want nil and no output", err, out)
	}
}

func TestCommandPreWriteUsageExitsTwo(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--pre-write"}, {"--pre-write", ""}, {"--bogus", "--pre-write", prePlan}} {
		out, err := runPreWrite(t, intentWith(""), args...)
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitUsage {
			t.Fatalf("%q: err = %v, want exit %d", args, err, ExitUsage)
		}
		if !strings.Contains(err.Error(), "usage: gabo check --pre-write <path>...") || out != "" {
			t.Fatalf("%q: err = %v, stdout = %q", args, err, out)
		}
	}
}

// Spec 0012 S7: the help names the design norms rule.
func TestCommandHelpNamesTheDesignRule(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{
		"  - docs/product/DESIGN.md when present: its 15 sections, the rule-table columns,",
		"    the RFC 2119 level, MUST verification, the exceptions and the changelog.",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

func TestCommandHelpNamesThePreWriteFlag(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{"--pre-write", "gabo check --pre-write <path>..."} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

// Usage errors of the hook mode never depend on the project: the factory
// is not called and the flag may come anywhere.
func TestCommandPreWriteUsageBeforeTheFactory(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--pre-write"}, {"--pre-write", ""}, {"--bogus", "--pre-write", prePlan}, {"--pre-write=x", prePlan}} {
		cmd := NewCommand(func(context.Context) (*UseCase, error) {
			t.Fatalf("%q: factory called", args)
			return nil, nil
		})
		cmd.SilenceUsage, cmd.SilenceErrors = true, true
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitUsage || !strings.Contains(err.Error(), "usage: gabo check --pre-write <path>...") {
			t.Fatalf("%q: err = %v, want the pre-write usage and exit %d", args, err, ExitUsage)
		}
	}
}

// --pre-write=false is not the plain check: it is a usage error, as any
// flag of the plain mode.
func TestCommandPreWriteFalseIsAUsageError(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--pre-write=false")
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitUsage || out != "" || !strings.Contains(err.Error(), "usage: gabo check") {
		t.Fatalf("err = %v, stdout = %q, want exit %d", err, out, ExitUsage)
	}
}

// Every argument after --pre-write is a path, also one that starts with
// "-": a hook passing paths in argv cannot be turned into a usage error
// (which hooks let through) by the path's name.
func TestCommandPreWriteArgumentsAfterTheFlagArePaths(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"--pre-write", "-x/../" + prePlan},
		{"--pre-write", "--help", prePlan},
		{"--pre-write", "--", prePlan},
		{"--pre-write=true", "-x/../" + prePlan},
		{"--pre-write=1", "--help", prePlan},
		{"--pre-write=true", "--", prePlan},
	} {
		out, err := runPreWrite(t, intentWith(unfilledMD), args...)
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitFailed || out != notAccepted+"\n" {
			t.Fatalf("%q: err = %v, stdout = %q, want exit %d and %q", args, err, out, ExitFailed, notAccepted)
		}
	}
}

// A project that cannot be loaded (here the working directory cannot be
// read) blocks the writes: the hook fails closed with exit 1 and a ✗ line,
// never exit 2, which hooks let through.
func TestCommandPreWriteFactoryErrorFailsClosed(t *testing.T) {
	t.Parallel()
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return nil, errors.New("getwd: no such file or directory")
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--pre-write", prePlan})
	err := cmd.ExecuteContext(context.Background())
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed {
		t.Fatalf("err = %v, want exit %d", err, ExitFailed)
	}
	want := "✗ " + prePlan + ": the project cannot be read (getwd: no such file or directory); it counts as not accepted\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
}

const (
	claudeUnaccepted = `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"/proj/docs/product/intents/0009-x/plan.md","content":"# plan"}}`
	codexPatch       = "*** Begin Patch\n*** Add File: README.md\n+hi\n*** Update File: docs/product/intents/0009-x/plan.md\n@@\n-a\n+b\n*** End Patch\n"
)

// With --pre-write and no path the targets come from the hook's tool-call
// JSON on stdin: Claude Code's tool_input.file_path, or the Add File and
// Update File headers of a Codex apply_patch in tool_input.command. The
// result equals the path-argument result.
func TestCommandPreWriteFromStdin(t *testing.T) {
	t.Parallel()
	codexJSON, err := json.Marshal(map[string]any{"tool_name": "apply_patch", "tool_input": map[string]any{"command": codexPatch}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, specMD, stdin string
		want                string
		code                int
	}{
		{name: "Claude Code, unaccepted spec", specMD: unfilledMD, stdin: claudeUnaccepted, want: notAccepted + "\n", code: ExitFailed},
		{name: "Claude Code, accepted spec", specMD: acceptedMD, stdin: claudeUnaccepted},
		{name: "Claude Code, not a plan", specMD: unfilledMD, stdin: `{"tool_input":{"file_path":"/proj/README.md"}}`},
		{name: "Codex apply_patch, unaccepted spec", specMD: unfilledMD, stdin: string(codexJSON), want: notAccepted + "\n", code: ExitFailed},
		{name: "Codex apply_patch, accepted spec", specMD: acceptedMD, stdin: string(codexJSON)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := runPreWriteStdin(t, intentWith(tt.specMD), tt.stdin, "--pre-write")
			code := 0
			var ee *ExitError
			if errors.As(err, &ee) {
				code = ee.Code
				if ee.Err != nil {
					t.Fatalf("err = %v, want a silent exit", err)
				}
			} else if err != nil {
				t.Fatalf("err = %v", err)
			}
			if code != tt.code || out != tt.want {
				t.Fatalf("exit %d, stdout %q; want exit %d, stdout %q", code, out, tt.code, tt.want)
			}
		})
	}
}

// Stdin is read only when no path is given: with paths it is ignored.
func TestCommandPreWritePathsIgnoreStdin(t *testing.T) {
	t.Parallel()
	out, err := runPreWriteStdin(t, intentWith(unfilledMD), claudeUnaccepted, "--pre-write", "README.md")
	if err != nil || out != "" {
		t.Fatalf("err = %v, stdout = %q, want nil and no output", err, out)
	}
}

// Empty, malformed or targetless stdin is a usage error: exit 2 with one
// usage line, nothing on stdout, and the project is never loaded.
func TestCommandPreWriteBadStdinExitsTwo(t *testing.T) {
	t.Parallel()
	for _, stdin := range []string{"", " \n", "{", "not json", "[]", `"x"`, `{}`, `{"tool_input":{}}`, `{"tool_input":{"file_path":""}}`, `{"tool_input":{"file_path":7}}`, `{"tool_input":{"command":7}}`, `{"tool_input":{"file_path":"a"}} {}`} {
		cmd := NewCommand(func(context.Context) (*UseCase, error) {
			t.Errorf("%q: the factory was called", stdin)
			return nil, errors.New("unexpected")
		})
		cmd.SilenceUsage, cmd.SilenceErrors = true, true
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&out)
		cmd.SetIn(strings.NewReader(stdin))
		cmd.SetArgs([]string{"--pre-write"})
		err := cmd.ExecuteContext(context.Background())
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitUsage {
			t.Fatalf("%q: err = %v, want exit %d", stdin, err, ExitUsage)
		}
		if msg := err.Error(); !strings.HasPrefix(msg, "usage: gabo check --pre-write <path>...") || strings.Contains(msg, "\n") || out.String() != "" {
			t.Fatalf("%q: err = %q, stdout = %q, want one usage line", stdin, msg, out.String())
		}
	}
}
