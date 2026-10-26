// Package status takes a one-shot snapshot of the agent flow. Each probe is
// independent: a failing probe reports StateUnknown with its error.
package status

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// ReviewContext is the status context carrying the reviewer verdict.
const ReviewContext = "reviewer/review"

// Probe states.
const (
	StateOK      = "ok"
	StateUnknown = "unknown"
)

// Probe is the common outcome of a probe.
type Probe struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

// ServerProbe tells whether the server answers a remote shell.
type ServerProbe struct {
	Probe
	Reachable bool `json:"reachable"`
}

// SessionsProbe lists the live gbai session terminals.
type SessionsProbe struct {
	Probe
	Alive []kernel.Terminal `json:"alive"`
}

// PRStatus summarizes one open PR. Review and Verdict hold "unknown: <err>"
// when their lookup failed.
type PRStatus struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Mode    string `json:"mode"`    // cloud, local or none
	Review  string `json:"review"`  // latest reviewer/review state or none
	Verdict string `json:"verdict"` // first line of the bot's last comment
}

// PRProbe lists open PRs to the integration branch.
type PRProbe struct {
	Probe
	Items []PRStatus `json:"items"`
}

// IssuesProbe lists open needs-user issues.
type IssuesProbe struct {
	Probe
	Items []Issue `json:"items"`
}

// Snapshot is the full status report.
type Snapshot struct {
	Server       ServerProbe   `json:"server"`
	Sessions     SessionsProbe `json:"sessions"`
	PullRequests PRProbe       `json:"pull_requests"`
	NeedsUser    IssuesProbe   `json:"needs_user"`
}

// Settings name the integration branch and the reviewer's login (the
// owner's own login in single-user mode).
type Settings struct {
	Integration string
	Bot         string
}

// UseCase gathers the snapshot.
type UseCase struct {
	shell kernel.RemoteShell
	terms kernel.Terminals
	gh    GitHub
	s     Settings
}

// NewUseCase wires the use case.
func NewUseCase(shell kernel.RemoteShell, terms kernel.Terminals, gh GitHub, s Settings) *UseCase {
	return &UseCase{shell: shell, terms: terms, gh: gh, s: s}
}

// Snapshot runs every probe once; it never fails as a whole.
func (u *UseCase) Snapshot(ctx context.Context) Snapshot {
	return Snapshot{
		Server:       u.server(ctx),
		Sessions:     u.sessions(ctx),
		PullRequests: u.pullRequests(ctx),
		NeedsUser:    u.needsUser(ctx),
	}
}

func ok() Probe               { return Probe{State: StateOK} }
func unknown(err error) Probe { return Probe{State: StateUnknown, Error: err.Error()} }

func (u *UseCase) server(ctx context.Context) ServerProbe {
	res, err := u.shell.Exec(ctx, kernel.Script{User: kernel.RootUser, Body: "true"})
	if err != nil {
		return ServerProbe{Probe: unknown(err)}
	}
	if res.ExitCode != 0 {
		return ServerProbe{Probe: unknown(fmt.Errorf("remote shell exited %d: %s", res.ExitCode, kernel.StderrTail(res.Stderr)))}
	}
	return ServerProbe{Probe: ok(), Reachable: true}
}

func (u *UseCase) sessions(ctx context.Context) SessionsProbe {
	live, err := u.terms.List(ctx)
	if err != nil {
		return SessionsProbe{Probe: unknown(err)}
	}
	titles := kernel.SessionTitles()
	alive := []kernel.Terminal{}
	for _, t := range live {
		if slices.Contains(titles, t.Title) {
			alive = append(alive, t)
		}
	}
	return SessionsProbe{Probe: ok(), Alive: alive}
}

func (u *UseCase) pullRequests(ctx context.Context) PRProbe {
	prs, err := u.gh.OpenPullRequests(ctx, u.s.Integration)
	if err != nil {
		return PRProbe{Probe: unknown(err)}
	}
	items := make([]PRStatus, 0, len(prs))
	for _, pr := range prs {
		st := PRStatus{Number: pr.Number, Title: pr.Title, Mode: mode(pr.Labels), Review: "none"}
		if state, err := u.gh.LatestStatus(ctx, pr.HeadSHA, ReviewContext); err != nil {
			st.Review = "unknown: " + err.Error()
		} else if state != "" {
			st.Review = state
		}
		if body, err := u.gh.LastVerdict(ctx, pr.Number, u.s.Bot); err != nil {
			st.Verdict = "unknown: " + err.Error()
		} else {
			st.Verdict, _, _ = strings.Cut(strings.TrimSpace(body), "\n")
			st.Verdict = strings.TrimSpace(st.Verdict)
		}
		items = append(items, st)
	}
	return PRProbe{Probe: ok(), Items: items}
}

func (u *UseCase) needsUser(ctx context.Context) IssuesProbe {
	issues, err := u.gh.NeedsUserIssues(ctx)
	if err != nil {
		return IssuesProbe{Probe: unknown(err)}
	}
	if issues == nil {
		issues = []Issue{}
	}
	return IssuesProbe{Probe: ok(), Items: issues}
}

func mode(labels []string) string {
	for _, l := range labels {
		if m, found := strings.CutPrefix(l, "mode:"); found && (m == "cloud" || m == "local") {
			return m
		}
	}
	return "none"
}
