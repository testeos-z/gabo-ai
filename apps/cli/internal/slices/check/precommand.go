package check

import (
	"errors"
	"strconv"
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

// JudgeShell judges one shell call under two locks (intent 0028 S2, S4, S6,
// S7, S10 to S13, S16): shell_guard for the shell rules (ShellViolations) and
// human_only_paths for a coder's write to a human-only path
// (humanOnlyShellWrites). It reads the lock states and the mode of gabo.yaml on
// every call, finds the violations in the call's session kind (CoderSession),
// takes each lock's state from Decide, prints one Explain block per violation
// of a lock that is not allowed, and records each lock's outcome through
// RecordDecision. The exit is the stronger of the locks' exits. A lock at off
// or warn, or a call that breaks no rule, exits 0 with no output. A config that
// cannot be read is an error: the mode fails closed. A record that cannot be
// written fails the call (exit 1).
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
	coder := CoderSession(call.AgentID, u.s.Engine)
	var groups []shellGroup
	if vs := ShellViolations(call.Command, coder); len(vs) > 0 {
		groups = append(groups, shellGroup{lock: config.LockShellGuard, violations: vs})
	}
	var writes []ShellViolation
	for _, p := range u.humanOnlyShellWrites(call.Command, coder, cfg.Review.HumanOnlyPaths) {
		writes = append(writes, ShellViolation{Rule: "a coder shell command writes the human-only path " + oneLine(p), Route: RouteHumanEdits})
	}
	if len(writes) > 0 {
		groups = append(groups, shellGroup{lock: config.LockHumanOnlyPaths, violations: writes})
	}
	var v ShellVerdict
	var b strings.Builder
	for _, g := range groups {
		// Each lock is decided and recorded on its own (intent 0028 S10, S13).
		d := Decide(cfg.Lock(g.lock), cfg.Mode)
		if err := u.RecordDecision(g.lock, g.violations[0].Rule, d); err != nil {
			return ShellVerdict{
				Exit: ExitFailed,
				Out:  Explain(g.lock, "the outcome record cannot be written ("+err.Error()+")", "fix the outcome file named by GABO_OUTCOME_FILE, then retry", Decision{}) + "\n",
			}, nil
		}
		if d.Exit == 0 {
			continue
		}
		for _, x := range g.violations {
			b.WriteString(Explain(g.lock, x.Rule, x.Route, d))
			b.WriteString("\n")
		}
		v.Exit = strictExit(v.Exit, d.Exit)
	}
	v.Out = b.String()
	return v, nil
}

// shellGroup is the violations one lock judges in one call.
type shellGroup struct {
	lock       config.LockName
	violations []ShellViolation
}

// strictExit returns the stronger of two exit codes of a call: block over ask
// over allow (intent 0028 S2).
func strictExit(a, b int) int {
	switch {
	case a == ExitFailed || b == ExitFailed:
		return ExitFailed
	case a == ExitAsk || b == ExitAsk:
		return ExitAsk
	}
	return 0
}

// oneLine quotes a path that holds a line break, so one message stays
// one line (intent 0028 S11).
func oneLine(p string) string {
	if strings.ContainsAny(p, "\r\n") {
		return strconv.Quote(p)
	}
	return p
}
