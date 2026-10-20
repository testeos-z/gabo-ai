// Package mcp exposes the gbai use cases as MCP tools. Tools are read-only or
// plan-first: apply tools only act with an explicit confirm, and no tool ever
// accepts or returns a secret value.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrInvalidInput is returned for tool input that fails validation.
var ErrInvalidInput = errors.New("invalid input")

// Result kinds.
const (
	KindResult  = "result"
	KindPlan    = "plan"
	KindApplied = "applied"
)

// Result is the structured output of every tool. Data is the use case's own
// structured result.
type Result struct {
	Kind string `json:"kind" jsonschema:"result, plan (nothing was changed) or applied"`
	Note string `json:"note,omitempty"`
	Data any    `json:"data"`
}

// Ports are the use cases the tools drive, wired by the composition root.
type Ports struct {
	Status         func(ctx context.Context, mode string) (any, error)
	Doctor         func(ctx context.Context) (any, error)
	Audit          func(ctx context.Context, since time.Time) (any, error)
	ProvisionPlan  func(ctx context.Context) (any, error)
	ProvisionApply func(ctx context.Context) (any, error)
	RepoSetupPlan  func(ctx context.Context) (any, error)
	RepoSetupApply func(ctx context.Context) (any, error)
	SessionsStart  func(ctx context.Context, role, mode string) (any, error)
	SessionsStop   func(ctx context.Context, role, mode string) (any, error)
	ReviewRequest  func(ctx context.Context, pr int, sha, notify string) (any, error)
}

// StatusInput selects where the sessions are listed.
type StatusInput struct {
	Mode string `json:"mode,omitempty" jsonschema:"cloud (default) or local"`
}

// AuditInput selects the audit window.
type AuditInput struct {
	Since string `json:"since,omitempty" jsonschema:"first merge date YYYY-MM-DD (default: 30 days ago)"`
}

// ApplyInput gates a mutating tool.
type ApplyInput struct {
	Confirm bool `json:"confirm,omitempty" jsonschema:"must be true to apply; otherwise only the plan is returned"`
}

// SessionsInput selects the sessions to start.
type SessionsInput struct {
	Role string `json:"role,omitempty" jsonschema:"coder, reviewer, assistant or all (default)"`
	Mode string `json:"mode,omitempty" jsonschema:"cloud (default, on the server) or local"`
}

// SessionsStopInput selects the sessions to stop; stopping needs confirm.
type SessionsStopInput struct {
	Role    string `json:"role,omitempty" jsonschema:"coder, reviewer, assistant or all (default)"`
	Mode    string `json:"mode,omitempty" jsonschema:"cloud (default, on the server) or local"`
	Confirm bool   `json:"confirm,omitempty" jsonschema:"must be true to close the terminals"`
}

// ReviewRequestInput identifies the pinned head to review.
type ReviewRequestInput struct {
	PR     int    `json:"pr" jsonschema:"pull request number"`
	SHA    string `json:"sha" jsonschema:"pinned 40-hex head SHA just pushed"`
	Notify string `json:"notify" jsonschema:"coder mailbox for the verdict: run:<id> or dispatch:<id>"`
}

// SecretCommand is a command the human runs in their own terminal.
type SecretCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// SecretInstructions tells the agent what to ask the human to run.
type SecretInstructions struct {
	Rule     string          `json:"rule"`
	Commands []SecretCommand `json:"commands"`
}

// Tools implements the tool handlers.
type Tools struct {
	p       Ports
	secrets []string
	now     func() time.Time
}

// NewTools wires the tools; secretNames are the names `gbai secrets set` accepts.
func NewTools(p Ports, secretNames []string, now func() time.Time) *Tools {
	return &Tools{p: p, secrets: secretNames, now: now}
}

var (
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	mailboxPattern = regexp.MustCompile(`^(run|dispatch):[A-Za-z0-9_.-]+$`)
	roles          = []string{"coder", "reviewer", "assistant"}
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, fmt.Sprintf(format, args...))
}

func result(data any, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}
	return Result{Kind: KindResult, Data: data}, nil
}

func mode(m string) (string, error) {
	switch m {
	case "":
		return "cloud", nil
	case "cloud", "local":
		return m, nil
	}
	return "", invalid("mode must be cloud or local, got %q", m)
}

func role(r string) (string, error) {
	if r == "" || r == "all" {
		return "all", nil
	}
	for _, known := range roles {
		if r == known {
			return r, nil
		}
	}
	return "", invalid("role must be coder, reviewer, assistant or all, got %q", r)
}

// Status takes the one-shot status snapshot.
func (t *Tools) Status(ctx context.Context, in StatusInput) (Result, error) {
	m, err := mode(in.Mode)
	if err != nil {
		return Result{}, err
	}
	return result(t.p.Status(ctx, m))
}

// Doctor runs the read-only checks.
func (t *Tools) Doctor(ctx context.Context, _ struct{}) (Result, error) {
	return result(t.p.Doctor(ctx))
}

// Audit checks the merges since the given date.
func (t *Tools) Audit(ctx context.Context, in AuditInput) (Result, error) {
	since := t.now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	if in.Since != "" {
		d, err := time.Parse("2006-01-02", in.Since)
		if err != nil {
			return Result{}, invalid("since must be a date like 2026-10-01")
		}
		since = d
	}
	return result(t.p.Audit(ctx, since))
}

// ProvisionPlan returns the provisioning scripts without running them.
func (t *Tools) ProvisionPlan(ctx context.Context, _ struct{}) (Result, error) {
	return plan(t.p.ProvisionPlan(ctx))
}

// ProvisionApply provisions the server when confirmed, else returns the plan.
func (t *Tools) ProvisionApply(ctx context.Context, in ApplyInput) (Result, error) {
	return t.apply(ctx, in.Confirm, t.p.ProvisionPlan, t.p.ProvisionApply)
}

// RepoSetupPlan returns the planned GitHub API calls.
func (t *Tools) RepoSetupPlan(ctx context.Context, _ struct{}) (Result, error) {
	return plan(t.p.RepoSetupPlan(ctx))
}

// RepoSetupApply applies the repository setup when confirmed, else returns the plan.
func (t *Tools) RepoSetupApply(ctx context.Context, in ApplyInput) (Result, error) {
	return t.apply(ctx, in.Confirm, t.p.RepoSetupPlan, t.p.RepoSetupApply)
}

func plan(data any, err error) (Result, error) {
	if err != nil {
		return Result{}, err
	}
	return Result{Kind: KindPlan, Note: "nothing was changed; call the apply tool with confirm=true to apply", Data: data}, nil
}

func (t *Tools) apply(ctx context.Context, confirm bool, planFn, applyFn func(context.Context) (any, error)) (Result, error) {
	if !confirm {
		return plan(planFn(ctx))
	}
	data, err := applyFn(ctx)
	if err != nil {
		return Result{}, err
	}
	return Result{Kind: KindApplied, Data: data}, nil
}

// SessionsStart opens the session terminals (idempotent).
func (t *Tools) SessionsStart(ctx context.Context, in SessionsInput) (Result, error) {
	r, m, err := roleMode(in.Role, in.Mode)
	if err != nil {
		return Result{}, err
	}
	return result(t.p.SessionsStart(ctx, r, m))
}

// SessionsStop closes the session terminals when confirmed.
func (t *Tools) SessionsStop(ctx context.Context, in SessionsStopInput) (Result, error) {
	r, m, err := roleMode(in.Role, in.Mode)
	if err != nil {
		return Result{}, err
	}
	if !in.Confirm {
		selected := roles
		if r != "all" {
			selected = []string{r}
		}
		titles := make([]string, 0, len(selected))
		for _, ro := range selected {
			titles = append(titles, "gbai-"+ro)
		}
		return Result{Kind: KindPlan, Note: "would close the " + m + " Orca terminals titled " + strings.Join(titles, ", ") + "; call again with confirm=true"}, nil
	}
	data, err := t.p.SessionsStop(ctx, r, m)
	if err != nil {
		return Result{}, err
	}
	return Result{Kind: KindApplied, Data: data}, nil
}

func roleMode(r, m string) (string, string, error) {
	r, rerr := role(r)
	m, merr := mode(m)
	return r, m, errors.Join(rerr, merr)
}

// ReviewRequest opens the detached review terminal for a pushed head.
func (t *Tools) ReviewRequest(ctx context.Context, in ReviewRequestInput) (Result, error) {
	var errs []error
	if in.PR < 1 {
		errs = append(errs, invalid("pr must be a positive number"))
	}
	if !shaPattern.MatchString(in.SHA) {
		errs = append(errs, invalid("sha must be a 40-hex commit SHA"))
	}
	if !mailboxPattern.MatchString(in.Notify) {
		errs = append(errs, invalid("notify must be a durable mailbox run:<id> or dispatch:<id>"))
	}
	if err := errors.Join(errs...); err != nil {
		return Result{}, err
	}
	return result(t.p.ReviewRequest(ctx, in.PR, in.SHA, in.Notify))
}

// SecretInstructions returns the commands the human must run themselves.
func (t *Tools) SecretInstructions(context.Context, struct{}) (Result, error) {
	ins := SecretInstructions{
		Rule:     "never ask for or accept a secret in the chat; ask the human to run the command in their own terminal, where gbai reads the value without echo",
		Commands: make([]SecretCommand, 0, len(t.secrets)),
	}
	for _, n := range t.secrets {
		ins.Commands = append(ins.Commands, SecretCommand{Name: n, Command: "gbai secrets set " + n})
	}
	return Result{Kind: KindResult, Data: ins}, nil
}
