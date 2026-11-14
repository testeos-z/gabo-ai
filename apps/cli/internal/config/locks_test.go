package config

import (
	"strings"
	"testing"
)

// I0028 S13, question 2: with no locks block every lock reads its default
// state, block for four locks and warn for test_files, and Marshal adds no
// block.
func TestLocksDefaultWhenBlockAbsent(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := map[LockName]LockState{
		LockShellGuard:     LockBlock,
		LockReleaseGate:    LockBlock,
		LockHumanOnlyPaths: LockBlock,
		LockCredentials:    LockBlock,
		LockTestFiles:      LockWarn,
	}
	for _, name := range LockNames {
		if got := cfg.Lock(name); got != want[name] {
			t.Errorf("Lock(%s) = %q, want the default %q", name, got, want[name])
		}
	}
	out, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "locks") {
		t.Errorf("Marshal() added a locks block:\n%s", out)
	}
}

// I0028 S13: locks.enabled false turns every lock off, even a lock whose own
// key says otherwise.
func TestLocksEnabledFalseTurnsEveryLockOff(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal + "locks:\n  enabled: false\n  shell_guard: block\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	for _, name := range LockNames {
		if got := cfg.Lock(name); got != LockOff {
			t.Errorf("Lock(%s) = %q, want %q under the all-off switch", name, got, LockOff)
		}
	}
}

// I0028 S13: a per-lock key turns only that lock; the others keep their
// defaults.
func TestPerLockKeyChangesOnlyThatLock(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal + "locks:\n  shell_guard: off\n  test_files: off\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := cfg.Lock(LockShellGuard); got != LockOff {
		t.Errorf("Lock(shell_guard) = %q, want off", got)
	}
	if got := cfg.Lock(LockTestFiles); got != LockOff {
		t.Errorf("Lock(test_files) = %q, want off", got)
	}
	if got := cfg.Lock(LockReleaseGate); got != LockBlock {
		t.Errorf("Lock(release_gate) = %q, want its default block", got)
	}
	if got := cfg.Lock(LockCredentials); got != LockBlock {
		t.Errorf("Lock(credentials) = %q, want its default block", got)
	}
}

// I0028 S10, D2: the four states load for the four locks that take them.
func TestLockStatesLoad(t *testing.T) {
	t.Parallel()
	for _, state := range []LockState{LockOff, LockWarn, LockAsk, LockBlock} {
		doc := minimal + "locks:\n  human_only_paths: " + string(state) + "\n"
		cfg, err := Parse([]byte(doc))
		if err != nil {
			t.Fatalf("state %s: Parse() error = %v", state, err)
		}
		if got := cfg.Lock(LockHumanOnlyPaths); got != state {
			t.Errorf("Lock(human_only_paths) = %q, want %q", got, state)
		}
	}
}

// Question 1 (spec section 6): test_files accepts off and warn only; ask and
// block fail the config check, and the failure names the key.
func TestTestFilesRefusesAskAndBlock(t *testing.T) {
	t.Parallel()
	for _, state := range []string{"ask", "block"} {
		_, err := Parse([]byte(minimal + "locks:\n  test_files: " + state + "\n"))
		if err == nil {
			t.Fatalf("test_files: %s parsed; want a failure", state)
		}
		if !strings.Contains(err.Error(), "locks.test_files") {
			t.Errorf("test_files: %s error = %v; want it to name locks.test_files", state, err)
		}
	}
}

// S13: an unknown lock name is one validation line that names the key, not a
// YAML decode failure of several lines.
func TestUnknownLockNameIsOneLine(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(minimal + "locks:\n  shell_gard: block\n"))
	if err == nil {
		t.Fatal("Parse() accepted an unknown lock name")
	}
	msg := err.Error()
	if strings.Contains(msg, "decode yaml") || strings.Contains(msg, "unmarshal") {
		t.Errorf("unknown lock name is a decode failure, want one validation line:\n%s", msg)
	}
	if !strings.Contains(msg, "locks.shell_gard") || strings.Contains(msg, "\n") {
		t.Errorf("error = %q; want one line naming locks.shell_gard", msg)
	}
}

// S13: a state outside the four is a failure that names its key.
func TestUnknownLockStateFails(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(minimal + "locks:\n  shell_guard: deny\n"))
	if err == nil || !strings.Contains(err.Error(), "locks.shell_guard") {
		t.Fatalf("Parse() error = %v; want a failure naming locks.shell_guard", err)
	}
}
