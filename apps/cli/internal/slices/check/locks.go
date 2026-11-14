package check

import "github.com/testeos-z/gabo-ai/apps/cli/internal/config"

// ExitAsk is the exit code of a hook mode for a violating call under a lock
// at ask: the engine shows its yes/no prompt (intent 0028 D3). It joins the
// TRD 9.4 codes 0, 1 and 2.
const ExitAsk = 3

// Outcome is what a lock records for one violating call (intent 0028 S16).
// An off lock records nothing, and an allowed call is never recorded.
type Outcome string

// The recorded outcomes of a violating call.
const (
	OutcomeWarned  Outcome = "warned"
	OutcomeAsked   Outcome = "asked"
	OutcomeBlocked Outcome = "blocked"
)

// Decision is the result of one violating call under one lock.
type Decision struct {
	// Exit is 0 (the call runs), ExitFailed (blocked) or ExitAsk (the engine
	// asks the human).
	Exit int
	// Outcome is empty when nothing is recorded (the lock is off).
	Outcome Outcome
	// Note says why the state was changed by the mode; empty otherwise.
	Note string
}

// Decide judges one violating call under a lock in state, in the project's
// mode (intent 0028 S10, S12). It is pure: it reads no file and writes no
// record. An unknown state fails closed as block.
func Decide(state config.LockState, mode config.Mode) Decision {
	switch state {
	case config.LockOff:
		return Decision{}
	case config.LockWarn:
		return Decision{Outcome: OutcomeWarned}
	case config.LockAsk:
		if mode == config.ModeAuto {
			return Decision{
				Exit:    ExitFailed,
				Outcome: OutcomeBlocked,
				Note:    "the state was ask; mode auto blocks instead of prompting",
			}
		}
		return Decision{Exit: ExitAsk, Outcome: OutcomeAsked}
	default:
		return Decision{Exit: ExitFailed, Outcome: OutcomeBlocked}
	}
}

// nextPrefix starts the route line of a block or an ask (TRD 9.4).
const nextPrefix = "next: "

// Explain renders the message of a blocked or asked call (intent 0028 S11):
// one "✗ <lock>: <rule>" line, with the decision's note in parentheses when
// it has one, then one "next: <route>" line.
func Explain(lock config.LockName, rule, route string, d Decision) string {
	line := "✗ " + string(lock) + ": " + rule
	if d.Note != "" {
		line += " (" + d.Note + ")"
	}
	return line + "\n" + nextPrefix + route
}
