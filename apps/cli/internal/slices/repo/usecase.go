// Package repo prepares the GitHub repository for the agent flow: mode and
// needs-user labels, and a ruleset protecting the integration branch.
package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// RulesetName identifies the ruleset gbai owns on the integration branch.
const RulesetName = "gbai-integration"

// Required status contexts posted by the reviewer.
var requiredChecks = []string{"reviewer/tests", "reviewer/review"}

// AdminWarning is printed on every run: rulesets do not bind repository or
// organization admins, who can edit or bypass them.
const AdminWarning = "warning: repository and organization admins can still edit or bypass rulesets; " +
	"never run the coder with an admin token"

// Label is a GitHub issue label.
type Label struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// Labels are the labels the flow relies on.
var Labels = []Label{
	{Name: "mode:cloud", Color: "0e8a16", Description: "Cloud agent PR: the reviewer may auto-merge"},
	{Name: "mode:local", Color: "1d76db", Description: "Local PR: a human merges"},
	{Name: "needs-user", Color: "d93f0b", Description: "Needs a human decision; agents skip it"},
}

// Settings locate the repository.
type Settings struct {
	Repo        string // owner/name
	Integration string // integration branch name
}

// Call is one planned API request.
type Call struct {
	Method string
	Path   string
	Body   []byte
	Note   string
}

// Output is the human-readable report.
type Output struct {
	Lines []string
}

// UseCase applies the repository setup idempotently.
type UseCase struct {
	api API
	s   Settings
}

// NewUseCase wires the use case.
func NewUseCase(api API, s Settings) *UseCase { return &UseCase{api: api, s: s} }

// Setup creates or updates the labels and the ruleset. With dryRun it only
// prints the calls it would make, assuming nothing exists yet.
func (u *UseCase) Setup(ctx context.Context, dryRun bool) (Output, error) {
	var out Output
	if dryRun {
		for _, c := range u.Plan(nil, nil) {
			out.Lines = append(out.Lines, fmt.Sprintf("%s %s  # %s", c.Method, c.Path, c.Note), string(c.Body))
		}
		out.Lines = append(out.Lines, "# existing labels are PATCHed and an existing "+RulesetName+" ruleset is PUT instead", AdminWarning)
		return out, nil
	}
	labels, err := u.listLabels(ctx)
	if err != nil {
		return out, err
	}
	rulesets, err := u.listRulesets(ctx)
	if err != nil {
		return out, err
	}
	for _, l := range unchangedLabels(labels) {
		out.Lines = append(out.Lines, "label "+l+": unchanged")
	}
	for _, c := range u.Plan(labels, rulesets) {
		if _, err := u.api.Call(ctx, c.Method, c.Path, c.Body); err != nil {
			return out, fmt.Errorf("%s: %w", c.Note, err)
		}
		out.Lines = append(out.Lines, c.Note+": done")
	}
	out.Lines = append(out.Lines, AdminWarning)
	return out, nil
}

// Ruleset is the summary of an existing repository ruleset.
type Ruleset struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Plan returns the write calls needed given the existing labels and rulesets.
func (u *UseCase) Plan(existing []Label, rulesets []Ruleset) []Call {
	base := "repos/" + u.s.Repo
	var calls []Call
	for _, want := range Labels {
		cur, found := findLabel(existing, want.Name)
		switch {
		case !found:
			calls = append(calls, Call{Method: "POST", Path: base + "/labels", Body: mustJSON(want), Note: "create label " + want.Name})
		case cur != want:
			body := mustJSON(map[string]string{"new_name": want.Name, "color": want.Color, "description": want.Description})
			calls = append(calls, Call{Method: "PATCH", Path: base + "/labels/" + url.PathEscape(cur.Name), Body: body, Note: "update label " + want.Name})
		}
	}
	body := rulesetBody(u.s.Integration)
	if id, ok := findRuleset(rulesets); ok {
		calls = append(calls, Call{Method: "PUT", Path: base + "/rulesets/" + strconv.FormatInt(id, 10), Body: body, Note: "update ruleset " + RulesetName})
	} else {
		calls = append(calls, Call{Method: "POST", Path: base + "/rulesets", Body: body, Note: "create ruleset " + RulesetName})
	}
	return calls
}

func (u *UseCase) listLabels(ctx context.Context) ([]Label, error) {
	var labels []Label
	if err := u.getJSON(ctx, "repos/"+u.s.Repo+"/labels?per_page=100", &labels); err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	return labels, nil
}

func (u *UseCase) listRulesets(ctx context.Context) ([]Ruleset, error) {
	var rs []Ruleset
	if err := u.getJSON(ctx, "repos/"+u.s.Repo+"/rulesets?per_page=100", &rs); err != nil {
		return nil, fmt.Errorf("list rulesets: %w", err)
	}
	return rs, nil
}

func (u *UseCase) getJSON(ctx context.Context, path string, v any) error {
	b, err := u.api.Call(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func findLabel(labels []Label, name string) (Label, bool) {
	for _, l := range labels {
		if strings.EqualFold(l.Name, name) {
			return l, true
		}
	}
	return Label{}, false
}

func unchangedLabels(existing []Label) []string {
	var out []string
	for _, want := range Labels {
		if cur, ok := findLabel(existing, want.Name); ok && cur == want {
			out = append(out, want.Name)
		}
	}
	return out
}

func findRuleset(rs []Ruleset) (int64, bool) {
	for _, r := range rs {
		if r.Name == RulesetName {
			return r.ID, true
		}
	}
	return 0, false
}

type rule struct {
	Type       string `json:"type"`
	Parameters any    `json:"parameters,omitempty"`
}

func rulesetBody(integration string) []byte {
	checks := make([]map[string]string, 0, len(requiredChecks))
	for _, c := range requiredChecks {
		checks = append(checks, map[string]string{"context": c})
	}
	return mustJSON(map[string]any{
		"name":          RulesetName,
		"target":        "branch",
		"enforcement":   "active",
		"bypass_actors": []any{},
		"conditions": map[string]any{
			"ref_name": map[string][]string{"include": {"refs/heads/" + integration}, "exclude": {}},
		},
		"rules": []rule{
			{Type: "deletion"},
			{Type: "non_fast_forward"},
			{Type: "pull_request", Parameters: map[string]any{
				"required_approving_review_count":   1,
				"dismiss_stale_reviews_on_push":     true,
				"require_code_owner_review":         false,
				"require_last_push_approval":        false,
				"required_review_thread_resolution": false,
			}},
			{Type: "required_status_checks", Parameters: map[string]any{
				"strict_required_status_checks_policy": false,
				"required_status_checks":               checks,
			}},
		},
	})
}

// mustJSON encodes values built from static types; failure is a programming error.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("encode json: %v", err))
	}
	return b
}
