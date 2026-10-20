package status

import (
	"context"
	"errors"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

type fakeGitHub struct {
	prs       []PullRequest
	prsErr    error
	statuses  map[string]string
	statusErr error
	comments  map[int]string
	issues    []Issue
	issuesErr error
}

func (f *fakeGitHub) OpenPullRequests(context.Context, string) ([]PullRequest, error) {
	return f.prs, f.prsErr
}

func (f *fakeGitHub) LatestStatus(_ context.Context, sha, _ string) (string, error) {
	return f.statuses[sha], f.statusErr
}

func (f *fakeGitHub) LastComment(_ context.Context, n int, _ string) (string, error) {
	return f.comments[n], nil
}

func (f *fakeGitHub) NeedsUserIssues(context.Context) ([]Issue, error) { return f.issues, f.issuesErr }

var settings = Settings{Integration: "dev", Bot: "vulcambot"}

func TestSnapshotHappyPath(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{
		prs:      []PullRequest{{Number: 7, Title: "feat: x", HeadSHA: "aaa", Labels: []string{"bug", "mode:cloud"}}, {Number: 8, HeadSHA: "bbb"}},
		statuses: map[string]string{"aaa": "success"},
		comments: map[int]string{7: "No blocking issues found.\n\nDetails..."},
		issues:   []Issue{{Number: 3, Title: "decide auth"}},
	}
	terms := &kerneltest.Terminals{Live: []kernel.Terminal{{Handle: "t1", Title: "gbai-coder"}, {Handle: "t2", Title: "cloud-pilot"}}}
	snap := NewUseCase(&kerneltest.RemoteShell{}, terms, gh, settings).Snapshot(context.Background())

	if snap.Server.State != StateOK || !snap.Server.Reachable {
		t.Fatalf("server = %+v", snap.Server)
	}
	if snap.Sessions.State != StateOK || len(snap.Sessions.Alive) != 1 || snap.Sessions.Alive[0].Title != "gbai-coder" {
		t.Fatalf("sessions = %+v", snap.Sessions)
	}
	if len(snap.PullRequests.Items) != 2 {
		t.Fatalf("prs = %+v", snap.PullRequests)
	}
	p7, p8 := snap.PullRequests.Items[0], snap.PullRequests.Items[1]
	if p7.Mode != "cloud" || p7.Review != "success" || p7.Verdict != "No blocking issues found." {
		t.Fatalf("pr 7 = %+v", p7)
	}
	if p8.Mode != "none" || p8.Review != "none" || p8.Verdict != "" {
		t.Fatalf("pr 8 = %+v", p8)
	}
	if snap.NeedsUser.State != StateOK || len(snap.NeedsUser.Items) != 1 {
		t.Fatalf("needs-user = %+v", snap.NeedsUser)
	}
}

func TestSnapshotProbesFailIndependently(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{
		prs:       []PullRequest{{Number: 7, HeadSHA: "aaa"}},
		statusErr: errors.New("rate limited"),
		issuesErr: errors.New("gh down"),
	}
	sh := &kerneltest.RemoteShell{Err: errors.New("railway unreachable")}
	terms := &kerneltest.Terminals{ListErr: errors.New("orca down")}
	snap := NewUseCase(sh, terms, gh, settings).Snapshot(context.Background())

	if snap.Server.State != StateUnknown || snap.Server.Error == "" {
		t.Fatalf("server = %+v", snap.Server)
	}
	if snap.Sessions.State != StateUnknown || snap.Sessions.Error != "orca down" {
		t.Fatalf("sessions = %+v", snap.Sessions)
	}
	if snap.PullRequests.State != StateOK || snap.PullRequests.Items[0].Review != "unknown: rate limited" {
		t.Fatalf("prs = %+v", snap.PullRequests)
	}
	if snap.NeedsUser.State != StateUnknown || snap.NeedsUser.Error != "gh down" {
		t.Fatalf("needs-user = %+v", snap.NeedsUser)
	}
}

func TestSnapshotServerNonZeroExitIsUnknown(t *testing.T) {
	t.Parallel()
	sh := &kerneltest.RemoteShell{Results: []kernel.Result{{ExitCode: 255, Stderr: []byte("ssh: no route")}}}
	snap := NewUseCase(sh, &kerneltest.Terminals{}, &fakeGitHub{}, settings).Snapshot(context.Background())
	if snap.Server.State != StateUnknown || snap.Server.Reachable {
		t.Fatalf("server = %+v", snap.Server)
	}
	if s := sh.Calls[0].Script; s.User != kernel.RootUser || s.Body != "true" {
		t.Fatalf("script = %+v", s)
	}
}
