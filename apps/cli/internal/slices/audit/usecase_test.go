package audit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/verdict"
)

type fakeGitHub struct {
	merged    []MergedPR
	mergedErr error
	statuses  map[string][]CommitStatus
	statusErr map[string]error
	// comments per PR; a PR missing from the map gets a clean verdict comment
	// by the bot signing its head.
	comments   map[int][]verdict.Comment
	commentErr map[int]error
	gotBase    string
	gotSince   time.Time
}

func (f *fakeGitHub) MergedPullRequests(_ context.Context, base string, since time.Time) ([]MergedPR, error) {
	f.gotBase, f.gotSince = base, since
	return f.merged, f.mergedErr
}

func (f *fakeGitHub) Statuses(_ context.Context, sha string) ([]CommitStatus, error) {
	return f.statuses[sha], f.statusErr[sha]
}

func (f *fakeGitHub) Comments(_ context.Context, number int) ([]verdict.Comment, error) {
	if err := f.commentErr[number]; err != nil {
		return nil, err
	}
	if c, ok := f.comments[number]; ok {
		return c, nil
	}
	for _, pr := range f.merged {
		if pr.Number == number {
			return []verdict.Comment{signed("vulcambot", verdict.Clean, pr.HeadSHA)}, nil
		}
	}
	return nil, nil
}

var settings = Settings{Integration: "dev", Bot: "vulcambot", Coder: "vulcam-coder"}

// head returns a full sha made of one repeated hex digit.
func head(c byte) string { return strings.Repeat(string(c), 40) }

func signed(author, first, sha string) verdict.Comment {
	return verdict.Comment{Author: author, Body: first + "\n\nnotes\n\n" + verdict.Footer + "\n" + verdict.Marker(sha)}
}

func good() []CommitStatus {
	return []CommitStatus{{Context: "reviewer/review", State: "success", Creator: "vulcambot"}, {Context: "reviewer/review", State: "failure", Creator: "vulcambot"}}
}

func reasons(rep Report) map[int][]string {
	got := map[int][]string{}
	for _, v := range rep.Violations {
		got[v.PR] = append(got[v.PR], v.Reason)
	}
	return got
}

func TestAuditDetectsViolations(t *testing.T) {
	t.Parallel()
	since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	gh := &fakeGitHub{
		merged: []MergedPR{
			{Number: 1, MergedBy: "vulcambot", HeadSHA: head('1')},
			{Number: 2, MergedBy: "owner", HeadSHA: head('2')},
			{Number: 3, MergedBy: "vulcam-coder", HeadSHA: head('3')},
			{Number: 4, MergedBy: "vulcambot", HeadSHA: head('4')},
			{Number: 5, MergedBy: "vulcambot", HeadSHA: head('5')},
			{Number: 6, MergedBy: "vulcambot[bot]", HeadSHA: head('6')},
			{Number: 7, MergedBy: "vulcambot", HeadSHA: head('7')},
		},
		statuses: map[string][]CommitStatus{
			head('1'): good(), head('2'): good(), head('3'): good(),
			head('4'): {{Context: "reviewer/tests", State: "success", Creator: "vulcambot"}},
			head('5'): {{Context: "reviewer/review", State: "failure", Creator: "vulcambot"}, {Context: "reviewer/review", State: "success", Creator: "vulcambot"}},
			head('6'): {{Context: "reviewer/review", State: "success", Creator: "vulcam-coder"}},
		},
		statusErr: map[string]error{head('7'): errors.New("rate limited")},
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
		4: "no reviewer/review status on head " + head('4'),
		5: "latest reviewer/review on head " + head('5') + " is failure",
		6: "reviewer/review on head " + head('6') + " was set by vulcam-coder, not the bot",
		7: "could not read statuses of head " + head('7') + ": rate limited",
	}
	got := reasons(rep)
	if len(got) != len(want) {
		t.Fatalf("violations = %+v", rep.Violations)
	}
	for pr, reason := range want {
		if !strings.Contains(strings.Join(got[pr], "; "), reason) {
			t.Errorf("PR %d reasons = %v, want %q", pr, got[pr], reason)
		}
	}
}

func TestAuditChecksTheReviewerMarkerComment(t *testing.T) {
	t.Parallel()
	sha := head('a')
	tests := []struct {
		name     string
		comments []verdict.Comment
		err      error
		want     string // empty means no violation
	}{
		{name: "clean marker on head", comments: []verdict.Comment{signed("vulcambot", verdict.Clean, sha)}},
		{name: "login differs only in case and bot suffix", comments: []verdict.Comment{signed("VulcamBot[bot]", verdict.Clean, sha)}},
		{name: "no marker comment", comments: []verdict.Comment{{Author: "vulcambot", Body: "looks good"}},
			want: "no reviewer verdict comment by vulcambot on PR"},
		{name: "marker for an older sha", comments: []verdict.Comment{signed("vulcambot", verdict.Clean, head('b'))},
			want: "newest reviewer verdict comment signs " + head('b') + ", not head " + sha},
		{name: "marker by another author", comments: []verdict.Comment{signed("mallory", verdict.Clean, sha)},
			want: "no reviewer verdict comment by vulcambot on PR"},
		{name: "blocking verdict", comments: []verdict.Comment{signed("vulcambot", verdict.Clean, sha), signed("vulcambot", verdict.Blocking, sha)},
			want: "newest reviewer verdict comment on head " + sha + " does not start with \"No blocking issues found.\""},
		{name: "comments unreadable", err: errors.New("gh down"),
			want: "could not read comments of PR 9: gh down"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gh := &fakeGitHub{
				merged:     []MergedPR{{Number: 9, MergedBy: "vulcambot", HeadSHA: sha}},
				statuses:   map[string][]CommitStatus{sha: good()},
				comments:   map[int][]verdict.Comment{9: tt.comments},
				commentErr: map[int]error{9: tt.err},
			}
			rep, err := NewUseCase(gh, settings).Audit(context.Background(), time.Time{})
			if err != nil {
				t.Fatalf("Audit() error = %v", err)
			}
			got := reasons(rep)[9]
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("violations = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Fatalf("violations = %v, want one containing %q", got, tt.want)
			}
		})
	}
}

func TestAuditCleanAndListError(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{merged: []MergedPR{{Number: 1, MergedBy: "vulcambot", HeadSHA: head('1')}}, statuses: map[string][]CommitStatus{head('1'): good()}}
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
	gh := &fakeGitHub{merged: []MergedPR{{Number: 3, MergedBy: "someone", HeadSHA: head('1')}}, statuses: map[string][]CommitStatus{head('1'): good()}}
	rep, err := NewUseCase(gh, Settings{Integration: "dev", Bot: "vulcambot"}).Audit(context.Background(), time.Time{})
	if err != nil || len(rep.Violations) != 1 || !strings.Contains(rep.Violations[0].Reason, "not the bot") {
		t.Fatalf("Audit() = %+v, %v", rep, err)
	}
}
