// Package doctor checks the known gotchas of the agent setup, locally and on
// the server. Every check is independent and read-only.
package doctor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// Status is the outcome of one check.
type Status string

// Check outcomes.
const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// ErrFailed is returned by the command when any check failed.
var ErrFailed = errors.New("doctor found failing checks")

// Check is one diagnosed item. Fix is set whenever Status is not pass.
type Check struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail,omitempty"`
	Fix    string `json:"fix,omitempty"`
}

// Report lists the checks in a stable order.
type Report struct {
	Checks []Check `json:"checks"`
}

// Failed reports whether any check failed.
func (r Report) Failed() bool {
	for _, c := range r.Checks {
		if c.Status == StatusFail {
			return true
		}
	}
	return false
}

// User is a server OS user; UID 0 means the uid is not managed.
type User struct {
	Name string
	UID  int
}

// Settings describe what to check.
type Settings struct {
	LocalOrca  string // orca CLI used in local mode
	Remote     bool   // run the server checks
	Users      []User // the orca user (single-user mode: it also runs the reviewer)
	OrcaUser   string
	SecretsDir string // model credential files written by `gbai secrets set`
}

// UseCase runs the checks.
type UseCase struct {
	local  Local
	runner kernel.Runner
	shell  kernel.RemoteShell
	s      Settings
}

// NewUseCase wires the use case.
func NewUseCase(local Local, runner kernel.Runner, shell kernel.RemoteShell, s Settings) *UseCase {
	return &UseCase{local: local, runner: runner, shell: shell, s: s}
}

type tool struct {
	name    string
	missing Status
	fix     string
}

// Run executes every check once.
func (u *UseCase) Run(ctx context.Context) Report {
	var rep Report
	tools := []tool{
		{"railway", StatusFail, "install the Railway CLI and run `railway login`"},
		{"gh", StatusFail, "install the GitHub CLI"},
		{u.s.LocalOrca, StatusWarn, "install Orca (only local mode needs it) or set sessions.local_orca"},
		{"go", StatusWarn, "install Go to build gbai from source"},
	}
	for _, t := range tools {
		rep.Checks = append(rep.Checks, u.lookPath(t))
	}
	rep.Checks = append(rep.Checks, u.ghAuth(ctx))
	if u.s.Remote {
		for _, rc := range u.remoteChecks() {
			rep.Checks = append(rep.Checks, u.remote(ctx, rc))
		}
	}
	return rep
}

func (u *UseCase) lookPath(t tool) Check {
	c := Check{Name: "local: " + t.name}
	p, err := u.local.LookPath(t.name)
	if err != nil {
		c.Status, c.Detail, c.Fix = t.missing, "not found in PATH", t.fix
		return c
	}
	c.Status, c.Detail = StatusPass, p
	return c
}

func (u *UseCase) ghAuth(ctx context.Context) Check {
	c := Check{Name: "local: gh auth status", Fix: "run `gh auth login` as the repository owner"}
	res, err := u.runner.Run(ctx, kernel.Command{Name: "gh", Args: []string{"auth", "status"}})
	switch {
	case err != nil:
		c.Status, c.Detail = StatusFail, err.Error()
	case res.ExitCode != 0:
		c.Status, c.Detail = StatusFail, "gh auth status exited "+strconv.Itoa(res.ExitCode)
	default:
		return Check{Name: c.Name, Status: StatusPass}
	}
	return c
}

type remoteCheck struct {
	name, script, fix string
}

// Each script runs as root, prints a short reason and exits 1 on failure.
// None of them write anything.
func (u *UseCase) remoteChecks() []remoteCheck {
	provision := "run `gbai provision`"
	var checks []remoteCheck
	for _, usr := range u.s.Users {
		s := fmt.Sprintf(`uid=$(id -u %s 2>/dev/null) || { echo "user %s does not exist"; exit 1; }`, q(usr.Name), usr.Name)
		if usr.UID > 0 {
			s += fmt.Sprintf("\n"+`[ "$uid" = "%d" ] || { echo "uid is $uid, expected %d"; exit 1; }`, usr.UID, usr.UID)
		}
		checks = append(checks, remoteCheck{"user " + usr.Name, s, provision})
	}
	var legacy strings.Builder
	for _, f := range kernel.LegacyReviewerFiles() {
		fmt.Fprintf(&legacy, `[ ! -e %s ] || { echo "%s is still present"; exit 1; }`+"\n", q(f), f)
	}
	checks = append(checks,
		remoteCheck{"legacy reviewer sudo removed", legacy.String(), provision},
		remoteCheck{"service env tokens", fmt.Sprintf(`names=$( { tr '\0' '\n' </proc/1/environ 2>/dev/null
for p in $(pgrep -u %s 2>/dev/null); do tr '\0' '\n' <"/proc/$p/environ" 2>/dev/null; done; } | cut -d= -f1 | grep -x -E 'GH_TOKEN|GITHUB_TOKEN|CLAUDE_CODE_OAUTH_TOKEN' | sort -u | tr '\n' ' ')
[ -z "$names" ] || { echo "visible to %s: $names"; exit 1; }`, q(u.s.OrcaUser), u.s.OrcaUser),
			"remove these variables from the Railway service and restart the sessions; the owner's gh login lives in gh's config and model credentials only in `gbai secrets set`"},
		remoteCheck{"secrets dir mode", fmt.Sprintf(`[ -e %s ] || exit 0
mode=$(stat -c %%a %s) || exit 1
[ "$mode" = 700 ] || { echo "mode is $mode, expected 700"; exit 1; }`, q(u.s.SecretsDir), q(u.s.SecretsDir)),
			"run `gbai secrets set` again or restrict " + u.s.SecretsDir + " to its owner (mode 700)"},
	)
	return checks
}

func (u *UseCase) remote(ctx context.Context, rc remoteCheck) Check {
	c := Check{Name: "remote: " + rc.name, Fix: rc.fix}
	res, err := u.shell.Exec(ctx, kernel.Script{User: kernel.RootUser, Body: rc.script})
	switch {
	case err != nil:
		c.Status, c.Detail, c.Fix = StatusFail, "server unreachable: "+err.Error(), "check `railway login` and the server ids in gbai.yaml"
	case res.ExitCode != 0:
		c.Status, c.Detail = StatusFail, strings.TrimSpace(string(res.Stdout))
		if c.Detail == "" {
			c.Detail = "exited " + strconv.Itoa(res.ExitCode) + ": " + kernel.StderrTail(res.Stderr)
		}
	default:
		return Check{Name: c.Name, Status: StatusPass}
	}
	return c
}

func q(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
