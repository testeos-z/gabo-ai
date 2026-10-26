package review

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRequestOpensDetachedTerminalRunningTheReviewAsTheCurrentUser(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	out, err := NewRequestUseCase(o).Request(context.Background(), RequestInput{PR: 12, SHA: sha, Notify: "run:abc"})
	if err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	if o.title != "gbai-review-12-0123456" {
		t.Errorf("title = %q", o.title)
	}
	want := "/usr/local/bin/gbai --config /etc/gbai/gbai.yaml review run head 12 " + sha +
		" | /usr/local/bin/gbai review notify --to 'run:abc' --pr 12 --sha " + sha
	if o.command != want {
		t.Errorf("command =\n%q\nwant\n%q", o.command, want)
	}
	if len(o.t.calls) != 1 || out.Title != o.title {
		t.Errorf("calls = %v, out = %+v", o.t.calls, out)
	}
	if strings.Contains(o.command, "sudo") || strings.Contains(o.command, "libexec") {
		t.Errorf("single-user mode must not go through sudo or a wrapper: %q", o.command)
	}
}

func TestRequestForwardsKeepWorktree(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	if _, err := NewRequestUseCase(o).Request(context.Background(), RequestInput{PR: 12, SHA: sha, Notify: "run:abc", KeepWorktree: true}); err != nil {
		t.Fatalf("Request() error = %v", err)
	}
	run, _, _ := strings.Cut(o.command, " | ")
	if want := "review run head 12 " + sha + " --keep-worktree"; !strings.HasSuffix(run, want) {
		t.Fatalf("run part = %q, want suffix %q", run, want)
	}
}

func TestRequestValidatesBeforeOrca(t *testing.T) {
	t.Parallel()
	for _, in := range []RequestInput{
		{PR: 0, SHA: sha, Notify: "run:a"},
		{PR: 1, SHA: "HEAD", Notify: "run:a"},
		{PR: 1, SHA: sha, Notify: "run:a; rm -rf /"},
		{PR: 1, SHA: sha, Notify: ""},
	} {
		o := &fakeOrca{t: &trace{}}
		if _, err := NewRequestUseCase(o).Request(context.Background(), in); err == nil {
			t.Errorf("Request(%+v) error = nil", in)
		}
		if len(o.t.calls) != 0 {
			t.Errorf("Request(%+v) reached orca", in)
		}
	}
}

func TestRequestReportsOrcaFailure(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}, err: errors.New("no orca")}
	if _, err := NewRequestUseCase(o).Request(context.Background(), RequestInput{PR: 1, SHA: sha, Notify: "dispatch:x"}); err == nil {
		t.Fatal("Request() error = nil")
	}
}

func TestNotifySendsTheOnlyMatchingSummary(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	var echo strings.Builder
	stdin := strings.Join([]string{
		"progress: secret-looking text",
		"| PR #12 head " + sha + " verdict=clean merge=merged",
		"PR #13 head " + sha + " verdict=clean merge=merged",
		"PR #12 head " + sha + " verdict=clean merge=needs-human-merge",
		"trailing noise",
	}, "\n")
	s, err := NewNotifyUseCase(o).Notify(context.Background(), NotifyInput{To: "run:abc", PR: 12, SHA: sha, Stdin: strings.NewReader(stdin), Echo: &echo})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	want := "PR #12 head " + sha + " verdict=clean merge=needs-human-merge"
	if s.String() != want || len(o.subjects) != 1 || o.subjects[0] != want || o.sentTo != "run:abc" {
		t.Fatalf("sent %q to %q, summary %s", o.subjects, o.sentTo, s)
	}
	if echo.String() != stdin+"\n" {
		t.Errorf("echo = %q", echo.String())
	}
}

func TestNotifyWithTwoMatchingSummariesSendsError(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	stdin := "PR #12 head " + sha + " verdict=blocking merge=none\nPR #12 head " + sha + " verdict=clean merge=merged\n"
	s, err := NewNotifyUseCase(o).Notify(context.Background(), NotifyInput{To: "run:abc", PR: 12, SHA: sha, Stdin: strings.NewReader(stdin), Echo: &strings.Builder{}})
	if err != nil || s.Verdict != VerdictError || o.subjects[0] != "PR #12 head "+sha+" verdict=error merge=none" {
		t.Fatalf("summary %s, sent %q, err %v", s, o.subjects, err)
	}
}

func TestNotifyWithoutSummarySendsError(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	_, err := NewNotifyUseCase(o).Notify(context.Background(), NotifyInput{To: "run:abc", PR: 12, SHA: sha, Stdin: strings.NewReader("panic: boom\n"), Echo: &strings.Builder{}})
	if err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if want := "PR #12 head " + sha + " verdict=error merge=none"; len(o.subjects) != 1 || o.subjects[0] != want {
		t.Fatalf("sent %q", o.subjects)
	}
}

func TestNotifyValidatesBeforeReading(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	if _, err := NewNotifyUseCase(o).Notify(context.Background(), NotifyInput{To: "bad", PR: 12, SHA: sha, Stdin: strings.NewReader(""), Echo: &strings.Builder{}}); err == nil {
		t.Fatal("Notify() error = nil")
	}
	if len(o.t.calls) != 0 {
		t.Fatalf("calls = %v", o.t.calls)
	}
}
