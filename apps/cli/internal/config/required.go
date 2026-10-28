package config

import (
	"errors"
	"fmt"
)

// NextSetup is the last line of a load error that names missing keys: the
// owner's agent writes them during setup, gabo never fills them (R12).
const NextSetup = "next: run setup"

// Suggestion is a value setup may offer for a required key, among others.
// gabo never applies it; the owner chooses (R12, TRD TD2).
type Suggestion struct {
	Key string
	// Values holds the suggested value; a list key has one entry per item.
	Values []string
}

// formerDefaults are the values the loader used to fill silently.
var formerDefaults = []Suggestion{
	{Key: "runtimes.coder", Values: []string{string(RuntimeClaude)}},
	{Key: "runtimes.reviewer", Values: []string{string(RuntimePi)}},
	{Key: "runtimes.assistant", Values: []string{string(RuntimeClaude)}},
	{Key: "repo.integration_branch", Values: []string{"dev"}},
	{Key: "review.checks", Values: []string{"bun install --frozen-lockfile", "bun run build", "bun run check-types", "bun run lint", "bun run test"}},
	{Key: "review.human_only_paths", Values: []string{".claude/", ".github/", ".opencode/", "AGENTS.md", "CLAUDE.md", "docs/policies/", "tools/hook-tests/", "agents/"}},
	{Key: "review.pi_credential_env", Values: []string{"ANTHROPIC_API_KEY"}},
	{Key: "parallelism", Values: []string{"3"}},
}

// Suggestions returns a fresh copy of the former defaults, in key order, for
// setup to offer; changing it changes nothing in gabo.
func Suggestions() []Suggestion {
	out := make([]Suggestion, len(formerDefaults))
	for i, s := range formerDefaults {
		out[i] = Suggestion{Key: s.Key, Values: append([]string(nil), s.Values...)}
	}
	return out
}

// missingKeys lists the required keys absent from c, in a fixed order.
// parallelismSet tells an absent parallelism from an explicit 0. An explicit
// empty list is set, not missing, and fails its own validation.
// review.pi_credential_env is required only for the pi reviewer runtime,
// the only one that reads it.
func (c Config) missingKeys(parallelismSet bool) []string {
	var keys []string
	add := func(absent bool, key string) {
		if absent {
			keys = append(keys, key)
		}
	}
	add(c.Runtimes.Coder == "", "runtimes.coder")
	add(c.Runtimes.Reviewer == "", "runtimes.reviewer")
	add(c.Runtimes.Assistant == "", "runtimes.assistant")
	add(c.Repo.IntegrationBranch == "", "repo.integration_branch")
	add(c.Review.Checks == nil, "review.checks")
	add(c.Review.HumanOnlyPaths == nil, "review.human_only_paths")
	add(c.Runtimes.Reviewer == RuntimePi && c.Review.PiCredentialEnv == "", "review.pi_credential_env")
	add(!parallelismSet, "parallelism")
	return keys
}

// missingErrors renders missing keys as "✗ <key> missing" lines.
func missingErrors(keys []string) []error {
	errs := make([]error, 0, len(keys))
	for _, k := range keys {
		errs = append(errs, fmt.Errorf("✗ %s missing", k))
	}
	return errs
}

// withNextSetup ends err with NextSetup when any key was missing.
func withNextSetup(err error, missing []string) error {
	if len(missing) == 0 {
		return err
	}
	return errors.Join(err, errors.New(NextSetup))
}
