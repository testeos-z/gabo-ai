package check

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// The route a coder reads for a push to main (S11): the same text in every
// test that pins the block message.
const (
	pushMainRule  = "git push to main is blocked"
	pushMainRoute = "push your task branch and open a pull request into dev; the reviewer merges it"
)

// judgeShell runs the shell judgement with cfg as gabo.yaml, the engine and
// agent id of the call, and out as the outcome sink.
func judgeShell(t *testing.T, cfg config.Reader, engine, agentID, command string, out *fakeOutcomes) (ShellVerdict, error) {
	t.Helper()
	s := Settings{Layers: layers(), Engine: engine, Task: "I0028/T4", OutcomePath: "/scratch/outcomes.jsonl", Outcomes: out}
	return NewUseCase(project(), cfg, goodUser(), s).JudgeShell(ShellCall{Command: command, AgentID: agentID})
}

// I0028 S2, S10, S11, S12: one violating call under each state of
// shell_guard: its exit, its printed message and its one outcome record.
func TestJudgeShellOneViolatingCallPerState(t *testing.T) {
	t.Parallel()
	block := "✗ shell_guard: " + pushMainRule + "\nnext: " + pushMainRoute + "\n"
	tests := []struct {
		name    string
		cfg     string
		exit    int
		out     string
		outcome string
		note    string
	}{
		{name: "no key blocks by default", cfg: validConfig, exit: ExitFailed, out: block, outcome: "blocked"},
		{name: "off runs with nothing recorded", cfg: validConfig + "locks:\n  shell_guard: off\n", exit: 0},
		{name: "warn runs and records warned", cfg: validConfig + "locks:\n  shell_guard: warn\n", exit: 0, outcome: "warned"},
		{name: "ask exits 3 and records asked", cfg: validConfig + "locks:\n  shell_guard: ask\n", exit: ExitAsk, out: block, outcome: "asked"},
		{name: "ask in auto blocks with the note", cfg: validConfig + "mode: auto\nlocks:\n  shell_guard: ask\n", exit: ExitFailed,
			out: "✗ shell_guard: " + pushMainRule + " (the state was ask; mode auto blocks instead of prompting)\nnext: " + pushMainRoute + "\n", outcome: "blocked"},
		{name: "all switch off", cfg: validConfig + "locks:\n  enabled: false\n", exit: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			v, err := judgeShell(t, configFiles{projectFile: tt.cfg}, "claude", "a1", "git push origin main", out)
			if err != nil {
				t.Fatalf("JudgeShell() error = %v", err)
			}
			if v.Exit != tt.exit || v.Out != tt.out {
				t.Fatalf("verdict = %+v, want exit %d and out %q", v, tt.exit, tt.out)
			}
			if tt.outcome == "" {
				if len(out.lines) != 0 {
					t.Fatalf("records = %q, want none", out.lines)
				}
				return
			}
			if len(out.lines) != 1 {
				t.Fatalf("records = %q, want one", out.lines)
			}
			for _, want := range []string{`"outcome":"` + tt.outcome + `"`, `"lock":"shell_guard"`, `"task":"I0028/T4"`, `"engine":"claude"`} {
				if !strings.Contains(out.lines[0], want) {
					t.Fatalf("record %q lacks %s", out.lines[0], want)
				}
			}
			if tt.outcome == "asked" && !strings.Contains(out.lines[0], `"answer":"pending"`) {
				t.Fatalf("asked record %q lacks answer pending", out.lines[0])
			}
		})
	}
}

// An allowed call exits 0, prints nothing and records nothing.
func TestJudgeShellAllowedCallRecordsNothing(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	v, err := judgeShell(t, configFiles{projectFile: validConfig}, "claude", "a1", "git status", out)
	if err != nil || v.Exit != 0 || v.Out != "" || len(out.lines) != 0 {
		t.Fatalf("verdict = %+v, err = %v, records = %q; want exit 0, no output, no record", v, err, out.lines)
	}
}

// Session kind (S4): a main session may force-push, a coder session may not,
// and an engine that gives no agent id gets the coder rules.
func TestJudgeShellSessionKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, engine, agentID string
		exit                  int
	}{
		{"main session force push allowed", "claude", "", 0},
		{"coder force push blocked", "claude", "a1", ExitFailed},
		{"engine without an agent id is coder everywhere", "pi", "", ExitFailed},
		{"unset engine is coder everywhere", "", "", ExitFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v, err := judgeShell(t, configFiles{projectFile: validConfig}, tt.engine, tt.agentID, "git push --force origin feat/x", &fakeOutcomes{})
			if err != nil || v.Exit != tt.exit {
				t.Fatalf("verdict = %+v, err = %v; want exit %d", v, err, tt.exit)
			}
		})
	}
}

// A config that cannot be read or is invalid leaves the lock state unknown:
// JudgeShell returns the error, and the command fails closed.
func TestJudgeShellUnknownLockStateIsAnError(t *testing.T) {
	t.Parallel()
	bad := validConfig + "locks:\n  shell_guard: maybe\n"
	if _, err := judgeShell(t, configFiles{projectFile: bad}, "claude", "a1", "git status", &fakeOutcomes{}); err == nil {
		t.Fatalf("JudgeShell with an invalid lock state: err = nil, want an error")
	}
}

// A record that cannot be written fails the call (intent 0028 S16).
func TestJudgeShellRecordFailureFailsTheCall(t *testing.T) {
	t.Parallel()
	v, err := judgeShell(t, configFiles{projectFile: validConfig}, "claude", "a1", "git push origin main", &fakeOutcomes{err: errors.New("disk full")})
	if err != nil || v.Exit != ExitFailed || !strings.Contains(v.Out, "shell_guard") || !strings.Contains(v.Out, "disk full") {
		t.Fatalf("verdict = %+v, err = %v; want exit %d naming the write failure", v, err, ExitFailed)
	}
}

// runPreCommand runs the command with stdin as the tool-call JSON and cfg as
// the config reader; it reports whether the factory was called.
func runPreCommand(t *testing.T, cfg config.Reader, stdin string, args ...string) (out string, factoryCalled bool, err error) {
	t.Helper()
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		factoryCalled = true
		return NewUseCase(project(), cfg, goodUser(), Settings{Layers: layers(), Engine: "claude", Task: "I0028/T4"}), nil
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{}, args...))
	err = cmd.ExecuteContext(context.Background())
	return buf.String(), factoryCalled, err
}

const (
	pushMainCall = `{"agent_id":"a1","tool_name":"Bash","tool_input":{"command":"git push origin main"}}`
	statusCall   = `{"tool_name":"Bash","tool_input":{"command":"git status"}}`
)

// I0028 S2: a blocked call exits 1 with one ✗ line and its next: line; an
// allowed call exits 0 with no output.
func TestCommandPreCommandBlocksAndAllows(t *testing.T) {
	t.Parallel()
	out, _, err := runPreCommand(t, configFiles{projectFile: validConfig}, pushMainCall, "--pre-command")
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed || ee.Err != nil {
		t.Fatalf("err = %v, want a silent exit %d", err, ExitFailed)
	}
	if want := "✗ shell_guard: " + pushMainRule + "\nnext: " + pushMainRoute + "\n"; out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
	out, _, err = runPreCommand(t, configFiles{projectFile: validConfig}, statusCall, "--pre-command")
	if err != nil || out != "" {
		t.Fatalf("git status: err = %v, stdout = %q; want nil and no output", err, out)
	}
}

// I0028 S2: a call with an ask lock exits 3 and prints its message.
func TestCommandPreCommandAskExitsThree(t *testing.T) {
	t.Parallel()
	cfg := configFiles{projectFile: validConfig + "locks:\n  shell_guard: ask\n"}
	out, _, err := runPreCommand(t, cfg, pushMainCall, "--pre-command")
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitAsk || ee.Err != nil {
		t.Fatalf("err = %v, want a silent exit %d", err, ExitAsk)
	}
	if !strings.HasPrefix(out, "✗ shell_guard: ") {
		t.Fatalf("stdout = %q, want the message", out)
	}
}

// I0028 S2: no stdin, empty or malformed stdin, or any argument is a usage
// error: exit 2, one usage line, nothing printed, and the project is not read.
func TestCommandPreCommandUsageExitsTwo(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		stdin string
		args  []string
	}{
		{"no stdin", "", []string{"--pre-command"}},
		{"not json", "not json", []string{"--pre-command"}},
		{"no command", `{"tool_input":{}}`, []string{"--pre-command"}},
		{"path argument", statusCall, []string{"--pre-command", "x"}},
		{"with pre-write", statusCall, []string{"--pre-command", "--pre-write", prePlan}},
		{"with a value", statusCall, []string{"--pre-command=true"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			out, called, err := runPreCommand(t, configFiles{projectFile: validConfig}, c.stdin, c.args...)
			var ee *ExitError
			if !errors.As(err, &ee) || ee.Code != ExitUsage {
				t.Fatalf("err = %v, want exit %d", err, ExitUsage)
			}
			if msg := err.Error(); !strings.HasPrefix(msg, "usage: gabo check --pre-command") || strings.Contains(msg, "\n") {
				t.Fatalf("err = %q, want one usage line", msg)
			}
			if out != "" || called {
				t.Fatalf("stdout = %q, factory called = %v; want no output and no project read", out, called)
			}
		})
	}
}

// I0028 S2 (fail closed): a project that cannot be loaded blocks the call with
// exit 1 and a ✗ line; an unreadable config does the same.
func TestCommandPreCommandFailsClosed(t *testing.T) {
	t.Parallel()
	out, _, err := runPreCommand(t, failingReader{}, statusCall, "--pre-command")
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed {
		t.Fatalf("unreadable config: err = %v, want exit %d", err, ExitFailed)
	}
	if !strings.HasPrefix(out, "✗ shell_guard: ") || !strings.Contains(out, "next: ") {
		t.Fatalf("unreadable config: stdout = %q, want a ✗ line and next:", out)
	}
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return nil, fs.ErrPermission
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader(statusCall))
	cmd.SetArgs([]string{"--pre-command"})
	err = cmd.ExecuteContext(context.Background())
	if !errors.As(err, &ee) || ee.Code != ExitFailed || !strings.HasPrefix(buf.String(), "✗ shell_guard: ") {
		t.Fatalf("factory error: err = %v, stdout = %q; want exit %d and a ✗ line", err, buf.String(), ExitFailed)
	}
}

// I0028 S2: the help names the new mode and the flag (TRD 9.4 help text).
func TestCommandHelpNamesThePreCommandFlag(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{"--pre-command", "gabo check --pre-command"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}
