package check

import (
	"errors"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// ShellVerdict is what the shell hook mode prints and exits with.
type ShellVerdict struct {
	// Exit is 0 (the call runs), ExitFailed (blocked) or ExitAsk (the engine
	// asks the human).
	Exit int
	// Out is the text to print, empty when nothing is printed.
	Out string
}

// JudgeShell judges one shell call under the shell_guard lock (intent 0028
// S2, S4, S6, S10 to S12, S16). It reads the lock state and the mode of
// gabo.yaml on every call, finds the violations (ShellViolations) in the call's
// session kind (CoderSession), takes the state from Decide, prints one
// Explain block per violation and records the call's outcome through
// RecordDecision. A lock at off or warn, or a call that breaks no rule, exits 0
// with no output. A config that cannot be read is an error: the mode fails
// closed. A record that cannot be written fails the call (exit 1).
func (u *UseCase) JudgeShell(call ShellCall) (ShellVerdict, error) {
	// A project with no config file reads every key as its default (PD2);
	// any other load error leaves the state unknown and fails the call.
	var cfg config.Config
	res, err := config.LoadLayers(u.config, u.s.Layers)
	switch {
	case err == nil:
		cfg = res.Config
	case errors.Is(err, config.ErrNoConfig):
	default:
		return ShellVerdict{}, err
	}
	violations := ShellViolations(call.Command, CoderSession(call.AgentID, u.s.Engine))
	if len(violations) == 0 {
		return ShellVerdict{}, nil
	}
	d := Decide(cfg.Lock(config.LockShellGuard), cfg.Mode)
	if err := u.RecordDecision(config.LockShellGuard, violations[0].Rule, d); err != nil {
		return ShellVerdict{
			Exit: ExitFailed,
			Out:  Explain(config.LockShellGuard, "the outcome record cannot be written ("+err.Error()+")", "fix the outcome file named by GABO_OUTCOME_FILE, then retry", Decision{}) + "\n",
		}, nil
	}
	if d.Exit == 0 {
		return ShellVerdict{}, nil
	}
	var b strings.Builder
	for _, v := range violations {
		b.WriteString(Explain(config.LockShellGuard, v.Rule, v.Route, d))
		b.WriteString("\n")
	}
	return ShellVerdict{Exit: d.Exit, Out: b.String()}, nil
}
