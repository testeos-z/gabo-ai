package audit

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var now = func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) }

func TestCommandViolationsExitNonZero(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{merged: []MergedPR{{Number: 2, MergedBy: "owner", HeadSHA: "b"}}}
	cmd := NewCommand(func(context.Context) (*UseCase, error) { return NewUseCase(gh, settings), nil }, now)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--since", "2026-10-01"})
	err := cmd.Execute()
	if !errors.Is(err, ErrViolations) {
		t.Fatalf("Execute() error = %v", err)
	}
	if !gh.gotSince.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("since = %s", gh.gotSince)
	}
	if !strings.Contains(out.String(), "#2: merged by owner") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestCommandDefaultsTo30DaysAndJSON(t *testing.T) {
	t.Parallel()
	gh := &fakeGitHub{}
	cmd := NewCommand(func(context.Context) (*UseCase, error) { return NewUseCase(gh, settings), nil }, now)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if want := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC); !gh.gotSince.Equal(want) {
		t.Fatalf("since = %s, want %s", gh.gotSince, want)
	}
	if !strings.Contains(out.String(), `"violations": []`) {
		t.Fatalf("json = %s", out.String())
	}
}

func TestCommandRejectsBadDate(t *testing.T) {
	t.Parallel()
	cmd := NewCommand(func(context.Context) (*UseCase, error) { return nil, errors.New("not reached") }, now)
	cmd.SetArgs([]string{"--since", "yesterday"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "since") {
		t.Fatalf("Execute() error = %v", err)
	}
}
