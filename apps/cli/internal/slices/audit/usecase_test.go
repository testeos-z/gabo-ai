package audit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeGitHub struct {
	merged    []MergedPR
	mergedErr error
	statuses  map[string][]CommitStatus
	statusErr map[string]error
	gotBase   string
	gotSince  time.Time
}

func (f *fakeGitHub) MergedPullRequests(_ context.Context, base string, since time.Time) ([]MergedPR, error) {
	f.gotBase, f.gotSince = base, since
	return f.merged, f.mergedErr
}

func (f *fakeGitHub) Statuses(_ context.Context, sha string) ([]CommitStatus, error) {
	return f.statuses[sha], f.statusErr[sha]
}

var settings = Settings{Integration: "dev", Bot: "vulcambot", Coder: "vulcam-coder"}

func good(sha string) []CommitStatus {
	return []CommitStatus{{Context: "reviewer/review", State: "success", Creator: "vulcambot"}, {Context: "reviewer/review", State: "failure", Creator: "vulcambot"}}
}

func TestAuditDetectsViolations(t *testing.T) {
	t.Parallel()
	since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	gh := &fakeGitHub{
		merged: []MergedPR{
			{Number: 1, MergedBy: "vulcambot", HeadSHA: "a"},
			{Number: 2, MergedBy: "owner", HeadSHA: "b"},
			{Number: 3, MergedBy: "vulcam-coder", HeadSHA: "c"},
			{Number: 4, MergedBy: "vulcambot", HeadSHA: "d"},
			{Number: 5, MergedBy: "vulcambot", HeadSHA: "e"},
			{Number: 6, MergedBy: "vulcambot[bot]", HeadSHA: "f"},
			{Number: 7, MergedBy: "vulcambot", HeadSHA: "g"},
		},
		statuses: map[string][]CommitStatus{
			"a": good("a"), "b": good("b"), "c": good("c"),
			"d": {{Context: "reviewer/tests", State: "success", Creator: "vulcambot"}},
			"e": {{Context: "reviewer/review", State: "failure", Creator: "vulcambot"}, {Context: "reviewer/review", State: "success", Creator: "vulcambot"}},
			"f": {{Context: "reviewer/review", State: "success", Creator: "vulcam-coder"}},
		},
		statusErr: map[string]error{"g": errors.New("rate limited")},
	}
	rep, err := NewUseCase(gh, settings).Audit(context.Background(), since)
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	if gh.gotBase != "dev" || !gh.gotSince.Equal(since) || rep.Checked != 7 {
		t.Fatalf("base = %s since = %s checked = %d", gh.gotBase, gh.gotSince, rep.Checked)
	}
	want := map[int]string{
		2: "merged by owner, not the bot vulcambot",
		3: "merged by the coder identity vulcam-coder",
		4: "no reviewer/review status on head d",
		5: "latest reviewer/review on head e is failure",
		6: "reviewer/review on head f was set by vulcam-coder, not the bot",
		7: "could not read statuses of head g: rate limited",
	}
	got := map[int][]string{}
	for _, v := range rep.Violations {
		got[v.PR] = append(got[v.PR], v.Reason)
	}
	if len(got) != len(want) {
		t.Fatalf("violations = %+v", rep.Violations)
	}
	for pr, reason := range want {
		if !strings.Contains(strings.Join(got[pr], "; "), reason) {
			t.Errorf("PR %d reasons = %v, want %q", pr, got[pr], reason)
		}
	}
}

func TestAuditCleanAndListError(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{merged: []MergedPR{{Number: 1, MergedBy: "vulcambot", HeadSHA: "a"}}, statuses: map[string][]CommitStatus{"a": good("a")}}
	rep, err := NewUseCase(gh, settings).Audit(context.Background(), time.Time{})
	if err != nil || len(rep.Violations) != 0 || rep.Violations == nil {
		t.Fatalf("Audit() = %+v, %v", rep, err)
	}
	gh = &fakeGitHub{mergedErr: errors.New("gh down")}
	if _, err := NewUseCase(gh, settings).Audit(context.Background(), time.Time{}); err == nil {
		t.Fatal("Audit() error = nil")
	}
}

func TestAuditWithoutCoderLoginSkipsCoderRule(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{merged: []MergedPR{{Number: 3, MergedBy: "someone", HeadSHA: "a"}}, statuses: map[string][]CommitStatus{"a": good("a")}}
	rep, err := NewUseCase(gh, Settings{Integration: "dev", Bot: "vulcambot"}).Audit(context.Background(), time.Time{})
	if err != nil || len(rep.Violations) != 1 || !strings.Contains(rep.Violations[0].Reason, "not the bot") {
		t.Fatalf("Audit() = %+v, %v", rep, err)
	}
}
