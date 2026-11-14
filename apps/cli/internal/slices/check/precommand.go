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

// JudgeShell judges one shell call under three locks (intent 0028 S2, S4, S6,
// S7, S10 to S13, S16): shell_guard and release_gate for the shell rules, and
// human_only_paths for a coder's write to a human-only path
// (humanOnlyShellWrites). Each lock's violations are judged in its state
// (judgeLock), and the verdicts are joined (joinVerdicts). A call that breaks no
// rule exits 0 with no output. A config that cannot be read is an error: the
// mode fails closed.
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
	shell := u.judgeLock(cfg, config.LockShellGuard, ShellViolations(call.Command, coder))
	release := u.judgeReleaseGate(cfg, call.Command, coder)
	human := u.judgeLock(cfg, config.LockHumanOnlyPaths, humanOnlyViolations(u.humanOnlyShellWrites(call.Command, coder, cfg.Review.HumanOnlyPaths)))
	return joinVerdicts(joinVerdicts(shell, release), human), nil
}

// humanOnlyViolations turns the human-only paths a call writes into one
// violation each, with the route a human edits the path (intent 0028 S7, S11).
func humanOnlyViolations(paths []string) []ShellViolation {
	var out []ShellViolation
	for _, p := range paths {
		out = append(out, ShellViolation{Rule: "a coder shell command writes the human-only path " + oneLine(p), Route: RouteHumanEdits})
	}
	return out
}

// oneLine quotes a path that holds a line break, so one message stays one
// line (intent 0028 S11).
func oneLine(p string) string {
	if strings.ContainsAny(p, "\r\n") {
		return strconv.Quote(p)
	}
	return p
}

// judgeLock judges the violations of one lock in its state: the first
// violation's decision is recorded, and when the call is blocked or asked each
// violation prints its Explain block (intent 0028 S10 to S12, S16). A lock at
// off or warn, or no violation, exits 0 with no output.
func (u *UseCase) judgeLock(cfg config.Config, lock config.LockName, violations []ShellViolation) ShellVerdict {
	if len(violations) == 0 {
		return ShellVerdict{}
	}
	d := Decide(cfg.Lock(lock), cfg.Mode)
	if err := u.RecordDecision(lock, violations[0].Rule, d); err != nil {
		return ShellVerdict{
			Exit: ExitFailed,
			Out:  Explain(lock, "the outcome record cannot be written ("+err.Error()+")", "fix the outcome file named by GABO_OUTCOME_FILE, then retry", Decision{}) + "\n",
		}
	}
	if d.Exit == 0 {
		return ShellVerdict{}
	}
	var b strings.Builder
	for _, v := range violations {
		b.WriteString(Explain(lock, v.Rule, v.Route, d))
		b.WriteString("\n")
	}
	return ShellVerdict{Exit: d.Exit, Out: b.String()}
}

// joinVerdicts joins the verdicts of two locks: a block wins over an ask and
// an ask over a run, and the messages keep their order.
func joinVerdicts(a, b ShellVerdict) ShellVerdict {
	exit := 0
	switch {
	case a.Exit == ExitFailed || b.Exit == ExitFailed:
		exit = ExitFailed
	case a.Exit == ExitAsk || b.Exit == ExitAsk:
		exit = ExitAsk
	}
	return ShellVerdict{Exit: exit, Out: a.Out + b.Out}
}
