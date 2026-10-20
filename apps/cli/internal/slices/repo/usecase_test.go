package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type apiCall struct {
	Method, Path string
	Body         string
}

type fakeAPI struct {
	responses map[string]string // "GET path" -> body
	calls     []apiCall
	err       error
}

func (f *fakeAPI) Call(_ context.Context, method, path string, body []byte) ([]byte, error) {
	f.calls = append(f.calls, apiCall{Method: method, Path: path, Body: string(body)})
	if f.err != nil {
		return nil, f.err
	}
	return []byte(f.responses[method+" "+path]), nil
}

func (f *fakeAPI) writes() []string {
	var out []string
	for _, c := range f.calls {
		if c.Method != "GET" {
			out = append(out, c.Method+" "+c.Path)
		}
	}
	return out
}

var settings = Settings{Repo: "cappy-lab/vulcam", Integration: "dev"}

const (
	labelsPath   = "repos/cappy-lab/vulcam/labels?per_page=100"
	rulesetsPath = "repos/cappy-lab/vulcam/rulesets?per_page=100"
)

func TestSetupCreatesMissingLabelsAndRuleset(t *testing.T) {
	t.Parallel()
	api := &fakeAPI{responses: map[string]string{"GET " + labelsPath: `[]`, "GET " + rulesetsPath: `[]`}}
	out, err := NewUseCase(api, settings).Setup(context.Background(), false)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	want := []string{
		"POST repos/cappy-lab/vulcam/labels",
		"POST repos/cappy-lab/vulcam/labels",
		"POST repos/cappy-lab/vulcam/labels",
		"POST repos/cappy-lab/vulcam/rulesets",
	}
	if got := api.writes(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("writes = %v, want %v", got, want)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "admin") {
		t.Fatalf("output lacks the admin bypass warning: %v", out.Lines)
	}
}

func TestSetupUpdatesExistingAndSkipsUnchanged(t *testing.T) {
	t.Parallel()
	labels := `[{"name":"mode:cloud","color":"0e8a16","description":"Cloud agent PR: the reviewer may auto-merge"},
		{"name":"Mode:Local","color":"ffffff","description":"old"},
		{"name":"needs-user","color":"d93f0b","description":"Needs a human decision; agents skip it"}]`
	rulesets := `[{"id":42,"name":"gbai-integration"},{"id":7,"name":"other"}]`
	api := &fakeAPI{responses: map[string]string{"GET " + labelsPath: labels, "GET " + rulesetsPath: rulesets}}
	if _, err := NewUseCase(api, settings).Setup(context.Background(), false); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	want := []string{"PATCH repos/cappy-lab/vulcam/labels/Mode:Local", "PUT repos/cappy-lab/vulcam/rulesets/42"}
	if got := api.writes(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("writes = %v, want %v", got, want)
	}
}

func TestRulesetBody(t *testing.T) {
	t.Parallel()
	var doc struct {
		Name         string `json:"name"`
		Target       string `json:"target"`
		Enforcement  string `json:"enforcement"`
		BypassActors []any  `json:"bypass_actors"`
		Conditions   struct {
			RefName struct {
				Include []string `json:"include"`
			} `json:"ref_name"`
		} `json:"conditions"`
		Rules []struct {
			Type       string         `json:"type"`
			Parameters map[string]any `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(rulesetBody("dev"), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Name != RulesetName || doc.Target != "branch" || doc.Enforcement != "active" {
		t.Fatalf("header = %+v", doc)
	}
	if doc.BypassActors == nil || len(doc.BypassActors) != 0 {
		t.Fatalf("bypass_actors must be an explicit empty list, got %v", doc.BypassActors)
	}
	if len(doc.Conditions.RefName.Include) != 1 || doc.Conditions.RefName.Include[0] != "refs/heads/dev" {
		t.Fatalf("include = %v", doc.Conditions.RefName.Include)
	}
	types := map[string]map[string]any{}
	for _, r := range doc.Rules {
		types[r.Type] = r.Parameters
	}
	for _, ty := range []string{"deletion", "non_fast_forward", "pull_request", "required_status_checks"} {
		if _, ok := types[ty]; !ok {
			t.Fatalf("missing rule %s in %v", ty, types)
		}
	}
	pr := types["pull_request"]
	if pr["required_approving_review_count"] != float64(1) || pr["dismiss_stale_reviews_on_push"] != true {
		t.Fatalf("pull_request = %v", pr)
	}
	checks, _ := json.Marshal(types["required_status_checks"]["required_status_checks"])
	if string(checks) != `[{"context":"reviewer/tests"},{"context":"reviewer/review"}]` {
		t.Fatalf("checks = %s", checks)
	}
}

func TestSetupDryRunNeverCallsAPI(t *testing.T) {
	t.Parallel()
	api := &fakeAPI{}
	out, err := NewUseCase(api, settings).Setup(context.Background(), true)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if len(api.calls) != 0 {
		t.Fatalf("dry run called the API: %v", api.calls)
	}
	text := strings.Join(out.Lines, "\n")
	for _, want := range []string{"POST repos/cappy-lab/vulcam/labels", `"name":"mode:cloud"`, "POST repos/cappy-lab/vulcam/rulesets", "refs/heads/dev", "admin"} {
		if !strings.Contains(text, want) {
			t.Fatalf("dry run output missing %q:\n%s", want, text)
		}
	}
}

func TestSetupStopsOnListError(t *testing.T) {
	t.Parallel()
	api := &fakeAPI{err: errors.New("boom")}
	if _, err := NewUseCase(api, settings).Setup(context.Background(), false); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("Setup() error = %v", err)
	}
}
