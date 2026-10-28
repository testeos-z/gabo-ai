package config

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// without returns minimal with the dotted key removed.
func without(t *testing.T, key string) []byte {
	t.Helper()
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(minimal), &doc); err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(key, ".")
	m := doc
	for _, p := range parts[:len(parts)-1] {
		m = m[p].(map[string]any)
	}
	delete(m, parts[len(parts)-1])
	b, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// R12 / TD2: no code path fills a required key; a missing one fails with
// "✗ <key> missing" and "next: run setup".
func TestParseFailsOnEveryMissingRequiredKey(t *testing.T) {
	t.Parallel()
	for _, key := range []string{
		"runtimes.coder", "runtimes.reviewer", "runtimes.assistant",
		"repo.integration_branch", "review.checks", "review.human_only_paths", "parallelism",
	} {
		_, err := Parse(without(t, key))
		if err == nil {
			t.Errorf("%s absent: Parse() error = nil, want it missing", key)
			continue
		}
		for _, want := range []string{"✗ " + key + " missing", "next: run setup"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("%s absent: error = %q, want it to contain %q", key, err, want)
			}
		}
	}
}

func TestParseReportsAllMissingKeysAndOneNextStep(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte("repo:\n  name: o/r\n"))
	if err == nil {
		t.Fatal("Parse() error = nil, want missing keys")
	}
	msg := err.Error()
	for _, key := range []string{"runtimes.coder", "runtimes.reviewer", "runtimes.assistant", "repo.integration_branch", "review.checks", "review.human_only_paths", "parallelism"} {
		if !strings.Contains(msg, "✗ "+key+" missing") {
			t.Errorf("error = %q, want %s missing", msg, key)
		}
	}
	if n := strings.Count(msg, "next: run setup"); n != 1 {
		t.Errorf("error = %q, want exactly one next step, got %d", msg, n)
	}
	if !strings.HasSuffix(msg, "next: run setup") {
		t.Errorf("error = %q, want the next step last", msg)
	}
}

// The reviewer scripts run Claude or Pi (T073, D27): every runtime accepts
// pi, a pi reviewer with or without review.pi_credential_env.
func TestParseAcceptsPiReviewer(t *testing.T) {
	t.Parallel()
	pi := strings.Replace(minimal, "reviewer: claude", "reviewer: pi", 1)
	for name, tt := range map[string]struct{ doc, env string }{
		"without env": {pi, ""},
		"with env":    {pi + "  pi_credential_env: OPENAI_API_KEY\n", "OPENAI_API_KEY"},
	} {
		cfg, err := Parse([]byte(tt.doc))
		if err != nil || cfg.Runtimes.Reviewer != RuntimePi || cfg.Review.PiCredentialEnv != tt.env {
			t.Errorf("%s: Parse() = %+v, %q, %v; want a pi reviewer with env %q", name, cfg.Runtimes, cfg.Review.PiCredentialEnv, err, tt.env)
		}
	}
	cfg, err := Parse([]byte(strings.NewReplacer("coder: claude", "coder: pi", "assistant: claude", "assistant: pi").Replace(minimal)))
	if err != nil || cfg.Runtimes.Coder != RuntimePi || cfg.Runtimes.Assistant != RuntimePi {
		t.Fatalf("pi coder and assistant = %+v, %v; want accepted", cfg.Runtimes, err)
	}
	_, err = Parse([]byte(strings.Replace(minimal, "reviewer: claude", "reviewer: codex", 1)))
	if err == nil || !strings.Contains(err.Error(), `runtimes.reviewer: must be "claude" or "pi", got "codex"`) {
		t.Fatalf("codex reviewer: error = %v, want the runtime enum", err)
	}
}

// review.pi_credential_env is optional for every reviewer runtime.
func TestParsePiCredentialEnvOptional(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil || cfg.Review.PiCredentialEnv != "" {
		t.Fatalf("claude reviewer without pi_credential_env = %q, %v; want empty and no error", cfg.Review.PiCredentialEnv, err)
	}
	cfg, err = Parse([]byte(minimal + "  pi_credential_env: OPENAI_API_KEY\n"))
	if err != nil || cfg.Review.PiCredentialEnv != "OPENAI_API_KEY" {
		t.Fatalf("claude reviewer with pi_credential_env = %q, %v", cfg.Review.PiCredentialEnv, err)
	}
}

// An explicit zero or empty value is set, not missing: it gets the key's own
// validation message instead of "run setup".
func TestParseExplicitEmptyValueIsInvalidNotMissing(t *testing.T) {
	t.Parallel()
	for key, doc := range map[string]string{
		"parallelism":             strings.Replace(minimal, "parallelism: 3", "parallelism: 0", 1),
		"review.checks":           strings.Replace(minimal, "checks: [go test ./...]", "checks: []", 1),
		"review.human_only_paths": strings.Replace(minimal, "human_only_paths: [.github/]", "human_only_paths: []", 1),
	} {
		_, err := Parse([]byte(doc))
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("%s empty: error = %v, want it named", key, err)
			continue
		}
		if strings.Contains(err.Error(), "missing") || strings.Contains(err.Error(), "run setup") {
			t.Errorf("%s empty: error = %q, want a validation error, not missing", key, err)
		}
	}
}

// Parse keeps exactly the values written: nothing is filled in.
func TestParseFillsNothing(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{
		Runtimes:    Runtimes{Coder: RuntimeClaude, Reviewer: RuntimeClaude, Assistant: RuntimeClaude},
		Repo:        Repo{Name: "cappy-lab/vulcam", IntegrationBranch: "dev"},
		Review:      Review{Checks: []string{"go test ./..."}, HumanOnlyPaths: []string{".github/"}},
		Parallelism: 3,
	}
	if !reflect.DeepEqual(cfg, want) {
		t.Fatalf("Parse() = %+v\nwant      %+v", cfg, want)
	}
}

func TestSuggestionsListFormerDefaults(t *testing.T) {
	t.Parallel()
	want := []Suggestion{
		{Key: "runtimes.coder", Values: []string{"claude"}},
		{Key: "runtimes.reviewer", Values: []string{"claude"}},
		{Key: "runtimes.assistant", Values: []string{"claude"}},
		{Key: "repo.integration_branch", Values: []string{"dev"}},
		{Key: "review.checks", Values: []string{"bun install --frozen-lockfile", "bun run build", "bun run check-types", "bun run lint", "bun run test"}},
		{Key: "review.human_only_paths", Values: []string{".claude/", ".github/", ".opencode/", "AGENTS.md", "CLAUDE.md", "docs/policies/", "tools/hook-tests/", "agents/"}},
		{Key: "parallelism", Values: []string{"3"}},
	}
	if got := Suggestions(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Suggestions() = %+v\nwant %+v", got, want)
	}
}

func TestSuggestionsAreReadOnly(t *testing.T) {
	t.Parallel()
	got := Suggestions()
	got[0].Key = "changed"
	got[4].Values[0] = "changed"
	again := Suggestions()
	if again[0].Key != "runtimes.coder" || again[4].Values[0] != "bun install --frozen-lockfile" {
		t.Fatalf("Suggestions() leaked a mutation: %+v", again[:5])
	}
}

// A required key unset in every layer fails after the merge, and `key: null`
// in a higher layer unsets it instead of restoring a default.
func TestLoadLayersFailsOnMissingRequiredKey(t *testing.T) {
	t.Parallel()
	r := files{
		"/home/o/.config/gabo/config.yaml": string(without(t, "parallelism")) + "parallelism: 4\n",
		"/src/p/gabo.yaml":                 "parallelism: null\n",
	}
	_, err := LoadLayers(r, twoLayers())
	if err == nil || !strings.Contains(err.Error(), "✗ parallelism missing") || !strings.HasSuffix(err.Error(), NextSetup) {
		t.Fatalf("LoadLayers() error = %v, want parallelism missing and %q", err, NextSetup)
	}
	_, err = LoadFile(files{"/x/c.yaml": string(without(t, "repo.integration_branch"))}, "/x/c.yaml")
	if err == nil || !strings.Contains(err.Error(), "✗ repo.integration_branch missing") || !strings.HasSuffix(err.Error(), NextSetup) {
		t.Fatalf("LoadFile() error = %v", err)
	}
}
