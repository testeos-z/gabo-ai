// Package config loads and validates gabo.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Runtime is the agent runtime used for a role.
type Runtime string

// Supported runtimes.
const (
	RuntimeClaude Runtime = "claude"
	RuntimePi     Runtime = "pi"
)

// Config is the validated content of gabo.yaml.
type Config struct {
	Runtimes Runtimes `yaml:"runtimes"`
	Repo     Repo     `yaml:"repo"`
	Review   Review   `yaml:"review"`
	Coder    Coder    `yaml:"coder"`
	Worktree Worktree `yaml:"worktree"`
	// Parallelism is the most coder sessions the cycle runs at once; it
	// already leaves room for the reviewer's checks. Required, no default.
	Parallelism int `yaml:"parallelism"`

	// Version is the schema version of the file; a file without it is
	// version 0, the current shape (see keyclass.go).
	Version int `yaml:"version"`
	// Project keys of DB_SCHEMA migration step 3 (see projectkeys.go).
	Mode       Mode              `yaml:"mode,omitempty"`
	Slots      map[string]Slot   `yaml:"slots,omitempty"`
	Reviewer   Reviewer          `yaml:"reviewer,omitempty"`
	Developers map[string]string `yaml:"developers,omitempty"`
	PlanChange PlanChange        `yaml:"plan_change,omitempty"`
	// Personal keys (see PersonalKeys): read from the global layers only.
	Alias            string `yaml:"alias,omitempty"`
	InteractionLevel string `yaml:"interaction_level,omitempty"`
	Tone             string `yaml:"tone,omitempty"`
	Language         string `yaml:"language,omitempty"`
}

// Worktree configures the cycle's wave cleanup.
type Worktree struct {
	// Keep lists worktree names (or absolute paths) that are never closed.
	Keep []string `yaml:"keep,omitempty"`
}

// Coder is the GitHub identity the coder pushes with; it must never merge.
type Coder struct {
	Login string `yaml:"login"`
}

// Runtimes selects the runtime per role.
type Runtimes struct {
	Coder     Runtime `yaml:"coder"`
	Reviewer  Runtime `yaml:"reviewer"`
	Assistant Runtime `yaml:"assistant"`
}

// Repo is the target GitHub repository.
type Repo struct {
	Name              string `yaml:"name"` // owner/name
	IntegrationBranch string `yaml:"integration_branch"`
}

// Owner returns the owner part of Name.
func (r Repo) Owner() string { o, _, _ := strings.Cut(r.Name, "/"); return o }

// Repo returns the repository part of Name.
func (r Repo) Repo() string { _, n, _ := strings.Cut(r.Name, "/"); return n }

// Review configures the user-level reviewer scripts (gabo-reviewer skill,
// D20), which read these keys from the project config.
type Review struct {
	// Checks are run in order inside a copy of the merged tree, without
	// GitHub credentials. Each is split on whitespace into argv; no shell is
	// involved.
	Checks         []string `yaml:"checks"`
	AutoMerge      bool     `yaml:"auto_merge"`
	HumanOnlyPaths []string `yaml:"human_only_paths"`
	// PiCredentialEnv names the env var that receives the pi model
	// credential; required when runtimes.reviewer is pi.
	PiCredentialEnv string `yaml:"pi_credential_env"`
	// CheckEnv names environment variables the reviewer passes from its own
	// environment into the checks (never into the agent review). Names only,
	// never values (R21); a listed variable unset at review time fails the
	// checks.
	CheckEnv []string `yaml:"check_env,omitempty"`
	// StateDir holds the reviewer's own clone, worktrees and check
	// workspaces. Empty means the reviewer scripts' default state dir.
	StateDir string `yaml:"state_dir,omitempty"`
}

var (
	envPattern    = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	pathPattern   = regexp.MustCompile(`^/[A-Za-z0-9._/-]*$`)
	repoPattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	branchPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	loginPattern  = regexp.MustCompile(`^[A-Za-z0-9-]+(\[bot\])?$`)
)

// reservedCheckEnv reports the names the reviewer sets for the checks itself
// or keeps from them (credentials): review.check_env may not list them.
func reservedCheckEnv(name string) bool {
	switch name {
	case "PATH", "HOME", "CI", "GOCACHE", "GOMODCACHE", "GOPATH", "GABO_UPGRADE_PR":
		return true
	}
	for _, prefix := range []string{"GH_", "GITHUB_", "CLAUDE_", "REVIEWER_"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// Load reads and validates the config file at path.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	cfg, err := Parse(b)
	if err != nil {
		return Config{}, fmt.Errorf("config %s: %w", path, err)
	}
	return cfg, nil
}

// Parse decodes and validates a config document. It fills no value: a
// missing required key fails with "✗ <key> missing" and NextSetup (R12);
// the former defaults are only offered through Suggestions.
func Parse(b []byte) (Config, error) {
	if err := formerCloudKeyErrors(b); err != nil {
		return Config{}, err
	}
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode yaml: %w", err)
	}
	// An int cannot tell an absent parallelism from an explicit 0.
	var set struct {
		Parallelism *int `yaml:"parallelism"`
	}
	if err := yaml.Unmarshal(b, &set); err != nil {
		return Config{}, fmt.Errorf("decode yaml: %w", err)
	}
	if err := cfg.validate(set.Parallelism != nil); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Marshal encodes the validated config as YAML.
func (c Config) Marshal() ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return b, nil
}

// validate reports every missing required key first, then every invalid
// value of a key that is set, and ends with NextSetup when a key was missing.
func (c Config) validate(parallelismSet bool) error {
	missing := c.missingKeys(parallelismSet)
	errs := missingErrors(missing)
	check := func(ok bool, field, msg string) {
		if !ok && !slices.Contains(missing, field) {
			errs = append(errs, fmt.Errorf("%s: %s", field, msg))
		}
	}
	for _, rt := range []struct {
		field string
		value Runtime
	}{{"runtimes.coder", c.Runtimes.Coder}, {"runtimes.reviewer", c.Runtimes.Reviewer}, {"runtimes.assistant", c.Runtimes.Assistant}} {
		check(rt.value == RuntimeClaude || rt.value == RuntimePi, rt.field, fmt.Sprintf("must be %q or %q, got %q", RuntimeClaude, RuntimePi, rt.value))
	}
	check(repoPattern.MatchString(c.Repo.Name), "repo.name", "must be owner/name")
	check(branchPattern.MatchString(c.Repo.IntegrationBranch), "repo.integration_branch", "invalid branch name")

	check(len(c.Review.Checks) > 0, "review.checks", "at least one check is required")
	for _, ch := range c.Review.Checks {
		check(strings.TrimSpace(ch) != "", "review.checks", "entries must not be blank")
	}
	check(len(c.Review.HumanOnlyPaths) > 0, "review.human_only_paths", "must not be empty")
	for _, p := range c.Review.HumanOnlyPaths {
		ok := p != "" && !strings.HasPrefix(p, "/") && !slices.Contains(strings.Split(strings.TrimSuffix(p, "/"), "/"), "..")
		check(ok, "review.human_only_paths", fmt.Sprintf("entry %q must be a relative repository path", p))
	}
	check(c.Review.PiCredentialEnv == "" || envPattern.MatchString(c.Review.PiCredentialEnv), "review.pi_credential_env", "must be an environment variable name")
	for _, name := range c.Review.CheckEnv {
		switch {
		case !envPattern.MatchString(name):
			check(false, "review.check_env", fmt.Sprintf("entry %q must be an environment variable name", name))
		case reservedCheckEnv(name):
			check(false, "review.check_env", fmt.Sprintf("entry %q is reserved for the reviewer", name))
		}
	}
	check(c.Coder.Login == "" || loginPattern.MatchString(c.Coder.Login), "coder.login", "must be a GitHub login")
	check(c.Review.StateDir == "" || pathPattern.MatchString(c.Review.StateDir), "review.state_dir", "must be an absolute path of safe characters")
	check(c.Parallelism > 0, "parallelism", "must be a positive number of coder sessions")
	for _, k := range c.Worktree.Keep {
		ok := k != "" && !strings.ContainsAny(k, "\n\r") && (path.IsAbs(k) || !strings.Contains(k, "/")) && k != ".." && k != "."
		check(ok, "worktree.keep", fmt.Sprintf("entry %q must be a worktree name or an absolute path", k))
	}
	errs = append(errs, c.validateKeyClasses()...)
	errs = append(errs, c.validateProjectKeys()...)
	return withNextSetup(errors.Join(errs...), missing)
}
