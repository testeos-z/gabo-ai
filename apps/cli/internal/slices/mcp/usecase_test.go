package mcp

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

type calls struct{ got []string }

func (c *calls) add(s string) { c.got = append(c.got, s) }

func fakePorts(c *calls) Ports {
	ret := func(name string) func(context.Context) (any, error) {
		return func(context.Context) (any, error) { c.add(name); return name + "-data", nil }
	}
	return Ports{
		Status: func(_ context.Context, mode string) (any, error) { c.add("status " + mode); return "snap", nil },
		Doctor: ret("doctor"),
		Audit: func(_ context.Context, since time.Time) (any, error) {
			c.add("audit " + since.Format("2006-01-02"))
			return "rep", nil
		},
		ProvisionPlan:  ret("provision-plan"),
		ProvisionApply: ret("provision-apply"),
		RepoSetupPlan:  ret("repo-plan"),
		RepoSetupApply: ret("repo-apply"),
		SessionsStart: func(_ context.Context, role, mode string) (any, error) {
			c.add("start " + role + " " + mode)
			return "started", nil
		},
		SessionsStop: func(_ context.Context, role, mode string) (any, error) {
			c.add("stop " + role + " " + mode)
			return "stopped", nil
		},
		ReviewRequest: func(_ context.Context, pr int, s, notify string) (any, error) {
			c.add("review " + notify)
			return "opened", nil
		},
	}
}

func newTools(c *calls) *Tools {
	now := func() time.Time { return time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC) }
	return NewTools(fakePorts(c), []string{"reviewer-claude", "reviewer-github"}, now)
}

func TestApplyWithoutConfirmReturnsPlanOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		call func(*Tools, bool) (Result, error)
		plan string
		ok   string
	}{
		{"provision", func(tl *Tools, c bool) (Result, error) { return tl.ProvisionApply(ctx, ApplyInput{Confirm: c}) }, "provision-plan", "provision-apply"},
		{"repo", func(tl *Tools, c bool) (Result, error) { return tl.RepoSetupApply(ctx, ApplyInput{Confirm: c}) }, "repo-plan", "repo-apply"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &calls{}
			res, err := tt.call(newTools(c), false)
			if err != nil || res.Kind != KindPlan || res.Data != tt.plan+"-data" || !slices.Equal(c.got, []string{tt.plan}) {
				t.Fatalf("unconfirmed = %+v, %v, calls %v", res, err, c.got)
			}
			c.got = nil
			res, err = tt.call(newTools(c), true)
			if err != nil || res.Kind != KindApplied || !slices.Equal(c.got, []string{tt.ok}) {
				t.Fatalf("confirmed = %+v, %v, calls %v", res, err, c.got)
			}
		})
	}
}

func TestSessionsStopNeedsConfirm(t *testing.T) {
	t.Parallel()
	c := &calls{}
	tl := newTools(c)
	res, err := tl.SessionsStop(context.Background(), SessionsStopInput{Role: "coder"})
	if err != nil || res.Kind != KindPlan || len(c.got) != 0 || !strings.Contains(res.Note, "gbai-coder") {
		t.Fatalf("unconfirmed stop = %+v, %v, calls %v", res, err, c.got)
	}
	if _, err := tl.SessionsStop(context.Background(), SessionsStopInput{Role: "all", Mode: "local", Confirm: true}); err != nil || !slices.Equal(c.got, []string{"stop all local"}) {
		t.Fatalf("confirmed stop: %v, calls %v", err, c.got)
	}
}

func TestInputValidationRejectsBeforePorts(t *testing.T) {
	t.Parallel()
	c := &calls{}
	tl := newTools(c)
	ctx := context.Background()
	checks := map[string]error{}
	_, checks["status mode"] = tl.Status(ctx, StatusInput{Mode: "prod"})
	_, checks["audit since"] = tl.Audit(ctx, AuditInput{Since: "yesterday"})
	_, checks["start role"] = tl.SessionsStart(ctx, SessionsInput{Role: "boss"})
	_, checks["review pr"] = tl.ReviewRequest(ctx, ReviewRequestInput{PR: 0, SHA: sha, Notify: "run:abc"})
	_, checks["review sha"] = tl.ReviewRequest(ctx, ReviewRequestInput{PR: 3, SHA: "abc", Notify: "run:abc"})
	_, checks["review notify"] = tl.ReviewRequest(ctx, ReviewRequestInput{PR: 3, SHA: sha, Notify: "term_1"})
	for name, err := range checks {
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
	if len(c.got) != 0 {
		t.Fatalf("invalid input reached ports: %v", c.got)
	}
}

func TestDefaultsAndPassThrough(t *testing.T) {
	t.Parallel()
	c := &calls{}
	tl := newTools(c)
	ctx := context.Background()
	if res, err := tl.Status(ctx, StatusInput{}); err != nil || res.Data != "snap" || res.Kind != KindResult {
		t.Fatalf("status = %+v, %v", res, err)
	}
	if _, err := tl.Audit(ctx, AuditInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.Audit(ctx, AuditInput{Since: "2026-10-01"}); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.SessionsStart(ctx, SessionsInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tl.ReviewRequest(ctx, ReviewRequestInput{PR: 3, SHA: sha, Notify: "dispatch:x1"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"status cloud", "audit 2026-09-20", "audit 2026-10-01", "start all cloud", "review dispatch:x1"}
	if !slices.Equal(c.got, want) {
		t.Fatalf("calls = %v, want %v", c.got, want)
	}
}

func TestSecretInstructionsNeverCarryValues(t *testing.T) {
	t.Parallel()
	res, err := newTools(&calls{}).SecretInstructions(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	ins, ok := res.Data.(SecretInstructions)
	if !ok || len(ins.Commands) != 2 {
		t.Fatalf("data = %#v", res.Data)
	}
	if ins.Commands[0].Command != "gbai secrets set reviewer-claude" || !strings.Contains(ins.Rule, "never") {
		t.Fatalf("instructions = %+v", ins)
	}
}

func TestPortErrorsPropagate(t *testing.T) {
	t.Parallel()
	p := fakePorts(&calls{})
	p.Doctor = func(context.Context) (any, error) { return nil, errors.New("boom") }
	tl := NewTools(p, nil, time.Now)
	if _, err := tl.Doctor(context.Background(), struct{}{}); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v", err)
	}
}
