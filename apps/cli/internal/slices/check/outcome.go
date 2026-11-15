package check

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// The environment variables the cycle sets for a run (intent 0028 S16,
// decision 15). Only the composition root reads them, into Settings.
const (
	// EnvOutcomePath names the file the outcome records are appended to.
	EnvOutcomePath = "GABO_OUTCOME_FILE"
	// EnvTask names the task of the run, as its record shows it.
	EnvTask = "GABO_TASK"
	// EnvEngine names the engine of the run.
	EnvEngine = "GABO_ENGINE"
)

// answerPending is the answer of an asked record until the engine's relay
// appends the human's answer (intent 0028 PD11).
const answerPending = "pending"

// errNoOutcomeWriter is returned when a file is named but no appender is wired.
var errNoOutcomeWriter = errors.New("outcome file named but no outcome writer is wired")

// outcomeRecord is one JSON line of the outcome file (intent 0028 S16): the
// lock, task, engine and rule, the outcome and, when there is one, its note
// and the answer of an asked record.
type outcomeRecord struct {
	Outcome string `json:"outcome"`
	Lock    string `json:"lock"`
	Task    string `json:"task"`
	Engine  string `json:"engine"`
	Rule    string `json:"rule"`
	Note    string `json:"note,omitempty"`
	Answer  string `json:"answer,omitempty"`
}

// recordable reports whether an outcome is one that is written. An allowed
// call (no outcome) and an off lock are never recorded (S16).
func recordable(o Outcome) bool {
	switch o {
	case OutcomeWarned, OutcomeAsked, OutcomeBlocked, OutcomeNotRun:
		return true
	default:
		return false
	}
}

// RecordDecision appends the record of one violating call's decision to the
// outcome file (Settings.OutcomePath), as one JSON line with the lock, the
// rule, the run's task and engine, and the outcome (intent 0028 S10, S16). It
// writes nothing when the decision records nothing (off, an allowed call) or
// when no file is named. An asked record carries the answer pending. A hook
// mode calls it before it exits, and a returned error must fail the call.
func (u *UseCase) RecordDecision(lock config.LockName, rule string, d Decision) error {
	if !recordable(d.Outcome) || u.s.OutcomePath == "" {
		return nil
	}
	if u.s.Outcomes == nil {
		return errNoOutcomeWriter
	}
	rec := outcomeRecord{
		Outcome: string(d.Outcome),
		Lock:    string(lock),
		Task:    u.s.Task,
		Engine:  u.s.Engine,
		Rule:    rule,
		Note:    d.Note,
	}
	if d.Outcome == OutcomeAsked {
		rec.Answer = answerPending
	}
	var line bytes.Buffer
	enc := json.NewEncoder(&line)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(rec); err != nil {
		return err
	}
	return u.s.Outcomes.Append(u.s.OutcomePath, line.Bytes())
}
