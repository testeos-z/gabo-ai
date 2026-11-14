package check

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

var errSink = errors.New("sink failed")

// fakeOutcomes is a hand-written OutcomeAppender that keeps every call.
type fakeOutcomes struct {
	names []string
	lines []string
	err   error
}

func (f *fakeOutcomes) Append(name string, line []byte) error {
	f.names = append(f.names, name)
	f.lines = append(f.lines, string(line))
	return f.err
}

// outcomeUseCase wires the use case with the outcome file named and the fake
// sink; the project and config are not read by RecordDecision.
func outcomeUseCase(f *fakeOutcomes, path string) *UseCase {
	return NewUseCase(project(), configFiles{projectFile: validConfig}, goodUser(), Settings{
		OutcomePath: path,
		Outcomes:    f,
		Task:        "I0028/T3",
		Engine:      "claude",
	})
}

// decodeOutcomeLine checks that line is exactly one JSON object followed by a single
// newline and returns its fields.
func decodeOutcomeLine(t *testing.T, line string) map[string]string {
	t.Helper()
	if strings.Count(line, "\n") != 1 || !strings.HasSuffix(line, "\n") {
		t.Fatalf("record %q is not one line", line)
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(line), &m); err != nil {
		t.Fatalf("record %q is not a JSON object of strings: %v", line, err)
	}
	return m
}

// I0028 S16 (file side), S10: one violating call under warn, ask and block
// appends exactly one record with the lock, task, engine, rule and outcome.
// An ask in mode manual carries the answer field pending.
func TestRecordOutcomeOneLinePerOutcome(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		state   config.LockState
		outcome Outcome
		answer  string
	}{
		{"warn", config.LockWarn, OutcomeWarned, ""},
		{"ask", config.LockAsk, OutcomeAsked, "pending"},
		{"block", config.LockBlock, OutcomeBlocked, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := &fakeOutcomes{}
			u := outcomeUseCase(f, "/scratch/outcomes.jsonl")
			if err := u.RecordDecision(config.LockTestFiles, "edit to an existing test file", Decide(c.state, config.ModeManual)); err != nil {
				t.Fatalf("RecordDecision() error = %v", err)
			}
			if len(f.lines) != 1 || f.names[0] != "/scratch/outcomes.jsonl" {
				t.Fatalf("Append calls = %q to %q; want one call to the named file", f.lines, f.names)
			}
			got := decodeOutcomeLine(t, f.lines[0])
			want := map[string]string{
				"outcome": string(c.outcome),
				"lock":    string(config.LockTestFiles),
				"task":    "I0028/T3",
				"engine":  "claude",
				"rule":    "edit to an existing test file",
			}
			if c.answer != "" {
				want["answer"] = c.answer
			}
			if len(got) != len(want) {
				t.Fatalf("record = %v; want exactly %v", got, want)
			}
			for k, v := range want {
				if got[k] != v {
					t.Fatalf("record[%q] = %q; want %q (record %v)", k, got[k], v, got)
				}
			}
		})
	}
}

// I0028 S12, S10: in mode auto an ask is recorded as blocked with the note
// that the state was ask, and no answer field.
func TestRecordOutcomeAutoAskIsBlockedWithNote(t *testing.T) {
	t.Parallel()
	f := &fakeOutcomes{}
	u := outcomeUseCase(f, "/scratch/outcomes.jsonl")
	if err := u.RecordDecision(config.LockShellGuard, "git push to main is blocked", Decide(config.LockAsk, config.ModeAuto)); err != nil {
		t.Fatalf("RecordDecision() error = %v", err)
	}
	if len(f.lines) != 1 {
		t.Fatalf("Append calls = %d; want one", len(f.lines))
	}
	got := decodeOutcomeLine(t, f.lines[0])
	if got["outcome"] != string(OutcomeBlocked) {
		t.Fatalf("outcome = %q; want blocked", got["outcome"])
	}
	if !strings.Contains(got["note"], "the state was ask") {
		t.Fatalf("note = %q; want it to say the state was ask", got["note"])
	}
	if _, ok := got["answer"]; ok {
		t.Fatalf("record %v carries an answer field; a blocked record has none", got)
	}
}

// I0028 S16: off records nothing, and an allowed call is never recorded: the
// sink is not called and nothing is written.
func TestRecordOutcomeOffAndAllowedRecordNothing(t *testing.T) {
	t.Parallel()
	f := &fakeOutcomes{}
	u := outcomeUseCase(f, "/scratch/outcomes.jsonl")
	if err := u.RecordDecision(config.LockShellGuard, "git status", Decide(config.LockOff, config.ModeManual)); err != nil {
		t.Fatalf("RecordDecision(off) error = %v", err)
	}
	if err := u.RecordDecision(config.LockShellGuard, "git status", Decision{}); err != nil {
		t.Fatalf("RecordDecision(allowed) error = %v", err)
	}
	if len(f.lines) != 0 {
		t.Fatalf("Append calls = %q; want none", f.lines)
	}
}

// I0028 S16: with no file named the call writes nothing and calls no sink,
// even for a block; a sink with no name is never used.
func TestRecordOutcomeNoFileNamedWritesNothing(t *testing.T) {
	t.Parallel()
	f := &fakeOutcomes{}
	u := outcomeUseCase(f, "")
	if err := u.RecordDecision(config.LockShellGuard, "git push to main is blocked", Decide(config.LockBlock, config.ModeManual)); err != nil {
		t.Fatalf("RecordDecision() error = %v", err)
	}
	if len(f.lines) != 0 {
		t.Fatalf("Append calls = %q; want none when no file is named", f.lines)
	}
}

// I0028 S16: not run is a record kind; its note carries the reason.
func TestRecordOutcomeNotRunIsAKind(t *testing.T) {
	t.Parallel()
	f := &fakeOutcomes{}
	u := outcomeUseCase(f, "/scratch/outcomes.jsonl")
	d := Decision{Outcome: OutcomeNotRun, Note: "the engine shows no pre-command event"}
	if err := u.RecordDecision(config.LockShellGuard, "git push to main is blocked", d); err != nil {
		t.Fatalf("RecordDecision(not run) error = %v", err)
	}
	if len(f.lines) != 1 {
		t.Fatalf("Append calls = %d; want one", len(f.lines))
	}
	got := decodeOutcomeLine(t, f.lines[0])
	if got["outcome"] != "not run" || got["note"] != "the engine shows no pre-command event" {
		t.Fatalf("record = %v; want outcome \"not run\" with its reason", got)
	}
}

// I0028 S16: a sink error is returned, so the hook mode can fail closed.
func TestRecordOutcomeReturnsSinkError(t *testing.T) {
	t.Parallel()
	f := &fakeOutcomes{err: errSink}
	u := outcomeUseCase(f, "/scratch/outcomes.jsonl")
	if err := u.RecordDecision(config.LockShellGuard, "rule", Decide(config.LockBlock, config.ModeManual)); err == nil {
		t.Fatal("RecordDecision() = nil; want the sink error")
	}
}
