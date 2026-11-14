package config

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LockName names one of the five deterministic hook locks (intent 0028 D2).
type LockName string

// The five locks, in the order their keys are listed in the schema.
const (
	LockShellGuard     LockName = "shell_guard"
	LockReleaseGate    LockName = "release_gate"
	LockHumanOnlyPaths LockName = "human_only_paths"
	LockCredentials    LockName = "credentials"
	LockTestFiles      LockName = "test_files"
)

// LockNames lists every lock name.
var LockNames = []LockName{LockShellGuard, LockReleaseGate, LockHumanOnlyPaths, LockCredentials, LockTestFiles}

// LockState is what a lock does with a violating call (intent 0028 S10).
type LockState string

// The four lock states.
const (
	LockOff   LockState = "off"
	LockWarn  LockState = "warn"
	LockAsk   LockState = "ask"
	LockBlock LockState = "block"
)

// lockStates lists the states a lock key may hold.
var lockStates = []LockState{LockOff, LockWarn, LockAsk, LockBlock}

// Locks configures the deterministic hook locks (intent 0028 S13). Every key
// is optional; a pointer or an empty string tells unset from set, and no
// value is filled in on load. The per-lock keys are strings so that a bad
// state is one validation line, not a decode failure.
type Locks struct {
	// Enabled false is the all-off switch: every lock reads off (S13).
	Enabled        *bool     `yaml:"enabled,omitempty"`
	ShellGuard     LockState `yaml:"shell_guard,omitempty"`
	ReleaseGate    LockState `yaml:"release_gate,omitempty"`
	HumanOnlyPaths LockState `yaml:"human_only_paths,omitempty"`
	Credentials    LockState `yaml:"credentials,omitempty"`
	TestFiles      LockState `yaml:"test_files,omitempty"`

	// unknown holds the keys of the block that name no lock, sorted, so
	// validation reports each as one line (S13).
	unknown []string
}

// UnmarshalYAML decodes the block without the strict field check of the
// loader, so an unknown lock name reaches validateLockKeys as one line.
func (l *Locks) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		if n.ShortTag() == "!!null" {
			return nil
		}
		return fmt.Errorf("line %d: locks must be a mapping of lock keys", n.Line)
	}
	type plain Locks
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	var keys map[string]yaml.Node
	if err := n.Decode(&keys); err != nil {
		return err
	}
	*l = Locks(p)
	for k := range keys {
		if k != "enabled" && !slices.Contains(lockNameStrings(), k) {
			l.unknown = append(l.unknown, k)
		}
	}
	sort.Strings(l.unknown)
	return nil
}

// lockNameStrings lists the lock names as plain strings.
func lockNameStrings() []string {
	out := make([]string, 0, len(LockNames))
	for _, n := range LockNames {
		out = append(out, string(n))
	}
	return out
}

// state returns the key of one lock as written, "" when unset.
func (l Locks) state(name LockName) LockState {
	switch name {
	case LockShellGuard:
		return l.ShellGuard
	case LockReleaseGate:
		return l.ReleaseGate
	case LockHumanOnlyPaths:
		return l.HumanOnlyPaths
	case LockCredentials:
		return l.Credentials
	case LockTestFiles:
		return l.TestFiles
	}
	return ""
}

// Lock returns the state of one lock: the all-off switch first, then the
// lock's own key, then its default (intent 0028 question 2): block for four
// locks and warn for test_files, so an existing project gets the locks
// without editing its file.
func (c Config) Lock(name LockName) LockState {
	if c.Locks.Enabled != nil && !*c.Locks.Enabled {
		return LockOff
	}
	if s := c.Locks.state(name); s != "" {
		return s
	}
	if name == LockTestFiles {
		return LockWarn
	}
	return LockBlock
}

// validateLockKeys checks each set lock key: a state from the four, and
// test_files only off or warn (intent 0028 question 1; a test file is warned,
// never blocked, D-4).
func (c Config) validateLockKeys() []error {
	var errs []error
	for _, k := range c.Locks.unknown {
		errs = append(errs, fmt.Errorf("locks.%s: unknown lock; must be one of %s", k, strings.Join(lockNameStrings(), ", ")))
	}
	for _, name := range LockNames {
		s := c.Locks.state(name)
		switch {
		case s == "":
		case !slices.Contains(lockStates, s):
			errs = append(errs, fmt.Errorf("locks.%s: must be %q, %q, %q or %q, got %q",
				name, LockOff, LockWarn, LockAsk, LockBlock, s))
		case name == LockTestFiles && s != LockOff && s != LockWarn:
			errs = append(errs, fmt.Errorf("locks.%s: must be %q or %q, got %q",
				name, LockOff, LockWarn, s))
		}
	}
	return errs
}
