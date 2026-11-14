package check

import (
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// I0028 S10, D3: one violating call under each state returns its exit code
// and its outcome. off runs with nothing recorded; warn runs and records a
// warned line; ask exits 3 and records asked; block exits 1 and records
// blocked. T2 returns the values only and writes no record (T3 writes them).
func TestDecideOneViolatingCallPerState(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		state   config.LockState
		exit    int
		outcome Outcome
	}{
		{"off", config.LockOff, 0, ""},
		{"warn", config.LockWarn, 0, OutcomeWarned},
		{"ask", config.LockAsk, ExitAsk, OutcomeAsked},
		{"block", config.LockBlock, ExitFailed, OutcomeBlocked},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := Decide(c.state, config.ModeManual)
			if got.Exit != c.exit || got.Outcome != c.outcome || got.Note != "" {
				t.Fatalf("Decide(%s, manual) = %+v; want exit %d, outcome %q, no note", c.state, got, c.exit, c.outcome)
			}
		})
	}
}

// I0028 S12: in mode auto a lock at ask behaves as block: exit 1, the outcome
// blocked, and a note that the state was ask. Other states keep their values.
func TestDecideAutoTurnsAskIntoBlock(t *testing.T) {
	t.Parallel()
	got := Decide(config.LockAsk, config.ModeAuto)
	if got.Exit != ExitFailed || got.Outcome != OutcomeBlocked {
		t.Fatalf("Decide(ask, auto) = %+v; want exit 1 and outcome blocked", got)
	}
	if !strings.Contains(got.Note, "the state was ask") {
		t.Fatalf("Decide(ask, auto).Note = %q; want it to say the state was ask", got.Note)
	}
	if warn := Decide(config.LockWarn, config.ModeAuto); warn.Exit != 0 || warn.Outcome != OutcomeWarned {
		t.Fatalf("Decide(warn, auto) = %+v; auto must not change warn", warn)
	}
}

// I0028 S11: the block message is one ✗ line with the lock, the rule that
// matched and the note, then one next: line with the route.
func TestExplainShape(t *testing.T) {
	t.Parallel()
	got := Explain(config.LockShellGuard, "git push to main is blocked",
		"the owner pushes outside agent sessions", Decide(config.LockBlock, config.ModeManual))
	want := "✗ shell_guard: git push to main is blocked\nnext: the owner pushes outside agent sessions"
	if got != want {
		t.Fatalf("Explain() =\n%s\nwant\n%s", got, want)
	}
}

// I0028 S11, S12: an auto-mode ask explains its own state in the ✗ line, and
// still has exactly one next: line.
func TestExplainAutoAskNamesTheState(t *testing.T) {
	t.Parallel()
	got := Explain(config.LockReleaseGate, "git tag v1.2.0 is a release",
		"the owner runs this outside agent sessions", Decide(config.LockAsk, config.ModeAuto))
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("Explain() has %d lines, want 2:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "✗ release_gate: git tag v1.2.0 is a release (the state was ask") {
		t.Fatalf("✗ line = %q; want the lock, the rule and the state note", lines[0])
	}
	if lines[1] != "next: the owner runs this outside agent sessions" {
		t.Fatalf("next line = %q", lines[1])
	}
}

// I0028 S13 end to end through the config: the state a lock reads from
// gabo.yaml decides the call, with the mode from the same file.
func TestDecideReadsStateFromGaboYAML(t *testing.T) {
	t.Parallel()
	cfg, err := config.Parse([]byte(validConfig + "mode: auto\nlocks:\n  shell_guard: ask\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	got := Decide(cfg.Lock(config.LockShellGuard), cfg.Mode)
	if got.Exit != ExitFailed || got.Outcome != OutcomeBlocked {
		t.Fatalf("shell_guard: ask with mode auto = %+v; want exit 1 and blocked", got)
	}
	if off := Decide(cfg.Lock(config.LockReleaseGate), cfg.Mode); off.Exit != ExitFailed {
		t.Fatalf("release_gate at its default = %+v; want block", off)
	}
}

// I0028 S13 golden: gabo check reports one unknown lock name as exactly one
// failing ✗ line, then the config's next: line.
func TestGaboCheckUnknownLockNameIsOneLine(t *testing.T) {
	t.Parallel()
	bad := validConfig + "locks:\n  shell_gard: block\n"
	r := run(t, project(), configFiles{projectFile: bad}, Settings{})
	wantOneFailureLine(t, r, projectFile, "✗ "+projectFile+": locks: additional properties 'shell_gard' not allowed")
}

// I0028 S13 golden, question 1: test_files set to ask is one exact failing line.
func TestGaboCheckTestFilesAskIsOneLine(t *testing.T) {
	t.Parallel()
	bad := validConfig + "locks:\n  test_files: ask\n"
	r := run(t, project(), configFiles{projectFile: bad}, Settings{})
	wantOneFailureLine(t, r, projectFile, "✗ "+projectFile+": locks.test_files: value must be one of 'off', 'warn', '', <nil>")
}

// I0028 S13 golden: an unknown lock state is one exact failing line.
func TestGaboCheckUnknownLockStateIsOneLine(t *testing.T) {
	t.Parallel()
	bad := validConfig + "locks:\n  shell_guard: deny\n"
	r := run(t, project(), configFiles{projectFile: bad}, Settings{})
	wantOneFailureLine(t, r, projectFile, "✗ "+projectFile+": locks.shell_guard: value must be one of 'off', 'warn', 'ask', 'block', '', <nil>")
}

// Intent 0028 decision 9: a user config that sets locks fails gabo check, and
// the failing line names the user file.
func TestGaboCheckRefusesLocksInUserConfig(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{userFile: "locks:\n  enabled: false\n", projectFile: validConfig}, Settings{})
	wantOneFailureLine(t, r, userFile, "✗ "+userFile+": locks: only gabo.yaml may set it; remove the block from the user config")
}

// I0028 S13: valid lock keys pass gabo check with its unchanged passed line.
func TestGaboCheckValidLocksPass(t *testing.T) {
	t.Parallel()
	good := validConfig + "locks:\n  enabled: true\n  shell_guard: ask\n  test_files: off\n"
	r := run(t, project(), configFiles{projectFile: good}, Settings{})
	wantPassed(t, r, PassedLine)
}

// wantOneFailureLine checks that the report failed with exactly one ✗ line for
// file, that the line is want, and that the report ends with NextFixConfig.
func wantOneFailureLine(t *testing.T, r Report, file, want string) {
	t.Helper()
	if !r.Failed {
		t.Fatalf("Failed = false, want true; output:\n%s", joined(r))
	}
	var own []string
	for _, l := range r.Lines {
		if strings.HasPrefix(l, "✗ "+file+": ") {
			own = append(own, l)
		}
	}
	if len(own) != 1 || own[0] != want {
		t.Fatalf("✗ lines for %s = %q, want exactly %q; output:\n%s", file, own, want, joined(r))
	}
	wantLines(t, r, NextFixConfig)
}
