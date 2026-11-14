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

// I0028 S13: gabo check reports one unknown lock name as one failing ✗ line,
// and names the key.
func TestGaboCheckUnknownLockNameIsOneLine(t *testing.T) {
	t.Parallel()
	bad := validConfig + "locks:\n  shell_gard: block\n"
	r := run(t, project(), configFiles{projectFile: bad}, Settings{})
	wantFailedOneProjectLine(t, r, "locks")
}

// I0028 S13, question 1: gabo check fails test_files set to ask as one
// failing line that names the key.
func TestGaboCheckTestFilesAskIsOneLine(t *testing.T) {
	t.Parallel()
	bad := validConfig + "locks:\n  test_files: ask\n"
	r := run(t, project(), configFiles{projectFile: bad}, Settings{})
	wantFailedOneProjectLine(t, r, "test_files")
}

// I0028 S13: valid lock keys pass gabo check with its unchanged passed line.
func TestGaboCheckValidLocksPass(t *testing.T) {
	t.Parallel()
	good := validConfig + "locks:\n  enabled: true\n  shell_guard: ask\n  test_files: off\n"
	r := run(t, project(), configFiles{projectFile: good}, Settings{})
	wantPassed(t, r, PassedLine)
}

// wantFailedOneProjectLine checks that the report failed with exactly one ✗
// line for the project file and that line names key, plus NextFixConfig.
func wantFailedOneProjectLine(t *testing.T, r Report, key string) {
	t.Helper()
	if !r.Failed {
		t.Fatalf("Failed = false, want true; output:\n%s", joined(r))
	}
	var own []string
	for _, l := range r.Lines {
		if strings.HasPrefix(l, "✗ "+projectFile+": ") {
			own = append(own, l)
		}
	}
	if len(own) != 1 {
		t.Fatalf("got %d ✗ lines for %s, want 1:\n%s", len(own), projectFile, joined(r))
	}
	if !strings.Contains(own[0], key) {
		t.Fatalf("✗ line %q does not name %q", own[0], key)
	}
	wantLines(t, r, NextFixConfig)
}
