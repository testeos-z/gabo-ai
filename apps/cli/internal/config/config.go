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
	RuntimeClaude   Runtime = "claude"
	RuntimeCodex    Runtime = "codex"
	RuntimePi       Runtime = "pi"
	RuntimeOpencode Runtime = "opencode"
)

// RuntimeSession in runtimes.coder or runtimes.reviewer runs that role in
// the main session, on the engines of runtimes.assistant (intent 0013 S1).
// It names no engine, so runtimes.assistant refuses it.
const RuntimeSession Runtime = "session"

// reviewerEngines are the engines the reviewer scripts run (TRD D27). The
// reviewer keeps this set whatever runtime skills exist (intent 0017 D3).
var reviewerEngines = []Runtime{RuntimeClaude, RuntimeCodex, RuntimePi, RuntimeOpencode}

// engineNamePattern is an engine name (intent 0017 D2, PD2): lower case
// letters, digits and "-", starting with a letter. Config accepts any name
// that matches it, and gabo check then needs a runtime skill for it (intent
// 0017 D1, D4). The schema's $defs/engine and $defs/runtime carry the same
// expression, and a test runs one list of names through both.
var engineNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// engineNameRule names the rule of engineNamePattern in the refusal lines.
const engineNameRule = `an engine name of lower case letters, digits and "-", starting with a letter`

// RuntimeList is runtimes.assistant: one engine or a list of engines. A
// scalar decodes to a one-item list; an absent, null or empty scalar to nil
// (missing); an explicit empty list to an empty, non-nil list (invalid).
type RuntimeList []Runtime

// UnmarshalYAML accepts a scalar or a sequence of scalars.
func (l *RuntimeList) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		var r Runtime
		if err := n.Decode(&r); err != nil {
			return err
		}
		*l = nil
		if r != "" {
			*l = RuntimeList{r}
		}
		return nil
	case yaml.SequenceNode:
		out := RuntimeList{}
		for _, item := range n.Content {
			if item.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: runtimes.assistant entries must be engine names", item.Line)
			}
			out = append(out, Runtime(item.Value))
		}
		*l = out
		return nil
	}
	return fmt.Errorf("line %d: runtimes.assistant must be an engine or a list of engines", n.Line)
}

// MarshalYAML writes one engine as a scalar, as 0.2.10 did, and nil as null.
func (l RuntimeList) MarshalYAML() (any, error) {
	switch {
	case l == nil:
		return nil, nil
	case len(l) == 1:
		return l[0], nil
	}
	return []Runtime(l), nil
}

// Single returns the engine of a one-item list and "" for any other list.
func (l RuntimeList) Single() Runtime {
	if len(l) == 1 {
		return l[0]
	}
	return ""
}

// Config is the validated content of gabo.yaml.
type Config struct {
	Runtimes Runtimes `yaml:"runtimes"`
	Repo     Repo     `yaml:"repo"`
	Review   Review   `yaml:"review"`
	Coder    Coder    `yaml:"coder"`
	Worktree Worktree `yaml:"worktree"`
	// Parallelism is the most coder sessions the cycle runs at once within
	// one wave (one intent's tasks); it already leaves room for the
	// reviewer's checks. Required, no default.
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
	Wave       Wave              `yaml:"wave,omitempty"`
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

// Runtimes selects the runtime per role. Coder and Reviewer hold one engine
// or RuntimeSession; Assistant holds one engine or a list (intent 0013 D1).
type Runtimes struct {
	Coder     Runtime     `yaml:"coder"`
	Reviewer  Runtime     `yaml:"reviewer"`
	Assistant RuntimeList `yaml:"assistant"`
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
	// PiCredentialEnv optionally names the env var whose value the pi
	// reviewer gets instead of reusing the owner's Pi login (T073).
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

// engineChoice is the message part naming the four reviewer engines.
var engineChoice = fmt.Sprintf("%q, %q, %q, %q", RuntimeClaude, RuntimeCodex, RuntimePi, RuntimeOpencode)

type fieldError struct{ field, msg string }

// runtimeErrors validates the runtimes block (intent 0013 S1, intent 0017):
// the name rules of runtimeNameErrors, then the rules that need the whole
// block: session needs runtimes.assistant, and an assistant list is non-empty
// and lists each engine once.
func (c Config) runtimeErrors() []fieldError {
	var errs []fieldError
	fail := func(field, msg string) { errs = append(errs, fieldError{field, msg}) }
	for _, role := range []struct {
		field string
		value Runtime
	}{{"runtimes.coder", c.Runtimes.Coder}, {"runtimes.reviewer", c.Runtimes.Reviewer}} {
		if role.value == RuntimeSession && c.Runtimes.Assistant == nil {
			fail(role.field, fmt.Sprintf("%q is valid only with runtimes.assistant set; next: set runtimes.assistant or name an engine for %s", RuntimeSession, role.field))
		}
	}
	errs = append(c.runtimeNameErrors(), errs...)
	const field = "runtimes.assistant"
	list := c.Runtimes.Assistant
	if list != nil && len(list) == 0 {
		fail(field, fmt.Sprintf("must list at least one engine; next: set one of %s or a list of them", engineChoice))
	}
	for i, r := range list {
		if r == RuntimeSession || !engineNamePattern.MatchString(string(r)) {
			continue
		}
		if slices.Index(list, r) < i && slices.Index(list[i+1:], r) < 0 {
			fail(field, fmt.Sprintf("%q is listed more than once; next: list each engine once", r))
		}
	}
	return errs
}

// runtimeNameErrors checks the engine names of runtimes.coder,
// runtimes.reviewer and runtimes.assistant, one line per bad value, with the
// rules of the role (intent 0017 S1, S3, S6). A missing (empty) value is not
// a name here: missingKeys reports it. The check needs no other key, so the
// loader runs it on each layer before the merge.
func (c Config) runtimeNameErrors() []fieldError {
	var errs []fieldError
	fail := func(field, msg string) { errs = append(errs, fieldError{field, msg}) }
	if r := c.Runtimes.Coder; r != "" && r != RuntimeSession && !engineNamePattern.MatchString(string(r)) {
		fail("runtimes.coder", fmt.Sprintf("must be %s, got %q", engineNameRule, r))
	}
	if r := c.Runtimes.Reviewer; r != "" && r != RuntimeSession && !slices.Contains(reviewerEngines, r) {
		fail("runtimes.reviewer", fmt.Sprintf("must be one of %s, got %q; next: set one of them (the reviewer scripts run only these engines) or ask your agent to configure the reviewer with gabo-reviewer-config", engineChoice, r))
	}
	const field = "runtimes.assistant"
	list := c.Runtimes.Assistant
	for _, r := range list {
		switch {
		case r == RuntimeSession:
			fail(field, fmt.Sprintf("%q is valid only in runtimes.coder and runtimes.reviewer; next: set the engine of the main session", RuntimeSession))
		case engineNamePattern.MatchString(string(r)):
		case len(list) == 1:
			fail(field, fmt.Sprintf("must be %s, got %q", engineNameRule, r))
		default:
			fail(field, fmt.Sprintf("entry %q must be %s; next: replace or remove it", r, engineNameRule))
		}
	}
	return errs
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
	for _, e := range c.runtimeErrors() {
		check(false, e.field, e.msg)
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
		case name == c.Review.PiCredentialEnv:
			check(false, "review.check_env", fmt.Sprintf("entry %q is review.pi_credential_env, the review engine's credential", name))
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
