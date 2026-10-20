// Package config loads and validates gbai.yaml.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
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

// Config is the validated content of gbai.yaml.
type Config struct {
	Server   Server   `yaml:"server"`
	Users    Users    `yaml:"users"`
	Runtimes Runtimes `yaml:"runtimes"`
	Repo     Repo     `yaml:"repo"`
	Bot      Bot      `yaml:"bot"`
	Review   Review   `yaml:"review"`
	Coder    Coder    `yaml:"coder"`
	Sessions Sessions `yaml:"sessions"`
}

// Coder is the GitHub identity the coder pushes with; it must never merge.
// An empty Login skips the coder-merge audit rule.
type Coder struct {
	Login string `yaml:"login"`
}

// Sessions configures the long-lived Orca agent sessions.
type Sessions struct {
	// LocalOrca is the Orca CLI used in local mode (orca-ide on Linux
	// desktops where orca is the screen reader).
	LocalOrca string  `yaml:"local_orca"`
	Coder     Session `yaml:"coder"`
	Reviewer  Session `yaml:"reviewer"`
	Assistant Session `yaml:"assistant"`
}

// Session configures one role's session.
type Session struct {
	// PromptFile, when set, is read on the session host and passed as the
	// startup prompt.
	PromptFile string `yaml:"prompt_file"`
}

// Server locates the cloud server.
type Server struct {
	Railway Railway `yaml:"railway"`
}

// Railway holds the explicit ids passed to `railway ssh`.
type Railway struct {
	ProjectID     string `yaml:"project_id"`
	EnvironmentID string `yaml:"environment_id"`
	ServiceID     string `yaml:"service_id"`
}

// Users lists the OS users on the server.
type Users struct {
	Orca       User `yaml:"orca"`
	Reviewer   User `yaml:"reviewer"`
	ReviewerCI User `yaml:"reviewer_ci"`
}

// All returns the users in a stable order.
func (u Users) All() []User { return []User{u.Orca, u.Reviewer, u.ReviewerCI} }

// User is one OS user on the server. UID 0 means the uid is not managed.
type User struct {
	Name      string   `yaml:"name"`
	Home      string   `yaml:"home"`
	UID       int      `yaml:"uid"`
	ExtraPath []string `yaml:"extra_path"`
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

// Bot is the GitHub identity the reviewer posts as.
type Bot struct {
	Login string `yaml:"login"`
}

// Review configures `gbai review run` and its reviewer-ci side.
type Review struct {
	// Checks are run in order inside the merged tree as reviewer-ci. Each is
	// split on whitespace into argv; no shell is involved.
	Checks         []string `yaml:"checks"`
	AutoMerge      bool     `yaml:"auto_merge"`
	HumanOnlyPaths []string `yaml:"human_only_paths"`
	// PiCredentialEnv names the env var that receives the pi model credential.
	PiCredentialEnv string `yaml:"pi_credential_env"`
}

// CheckArgv returns each check split into argv.
func (r Review) CheckArgv() [][]string {
	out := make([][]string, 0, len(r.Checks))
	for _, c := range r.Checks {
		out = append(out, strings.Fields(c))
	}
	return out
}

var (
	envPattern    = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	idPattern     = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	userPattern   = regexp.MustCompile(`^[a-z_][a-z0-9_-]{0,31}$`)
	pathPattern   = regexp.MustCompile(`^/[A-Za-z0-9._/-]*$`)
	repoPattern   = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	branchPattern = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	loginPattern  = regexp.MustCompile(`^[A-Za-z0-9-]+(\[bot\])?$`)
	filePattern   = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
)

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

// Parse decodes, defaults and validates a config document.
func Parse(b []byte) (Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode yaml: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Marshal encodes the validated config, defaults included, as YAML.
func (c Config) Marshal() ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode config: %w", err)
	}
	return b, nil
}

func (c *Config) applyDefaults() {
	defaultUser(&c.Users.Orca, User{Name: "orca", Home: "/data/home", ExtraPath: []string{"/data/home/.local/bin"}})
	defaultUser(&c.Users.Reviewer, User{Name: "reviewer", Home: "/data/reviewer", UID: 2000})
	defaultUser(&c.Users.ReviewerCI, User{Name: "reviewer-ci", Home: "/data/reviewer-ci", UID: 2001})
	defaultString(&c.Runtimes.Coder, RuntimeClaude)
	defaultString(&c.Runtimes.Reviewer, RuntimePi)
	defaultString(&c.Runtimes.Assistant, RuntimeClaude)
	defaultString(&c.Repo.IntegrationBranch, "dev")
	if c.Review.Checks == nil {
		c.Review.Checks = []string{"bun install --frozen-lockfile", "bun run build", "bun run check-types", "bun run lint", "bun run test"}
	}
	if c.Review.HumanOnlyPaths == nil {
		c.Review.HumanOnlyPaths = []string{".claude/", ".github/", ".opencode/", "AGENTS.md", "CLAUDE.md", "docs/policies/", "tools/hook-tests/", "agents/"}
	}
	defaultString(&c.Review.PiCredentialEnv, "ANTHROPIC_API_KEY")
	defaultString(&c.Sessions.LocalOrca, "orca")
}

func defaultUser(u *User, d User) {
	defaultString(&u.Name, d.Name)
	defaultString(&u.Home, d.Home)
	if u.UID == 0 {
		u.UID = d.UID
	}
	if u.ExtraPath == nil {
		u.ExtraPath = d.ExtraPath
	}
}

func defaultString[T ~string](v *T, d T) {
	if *v == "" {
		*v = d
	}
}

func (c Config) validate() error {
	var errs []error
	check := func(ok bool, field, msg string) {
		if !ok {
			errs = append(errs, fmt.Errorf("%s: %s", field, msg))
		}
	}
	r := c.Server.Railway
	check(idPattern.MatchString(r.ProjectID), "server.railway.project_id", "required id")
	check(idPattern.MatchString(r.EnvironmentID), "server.railway.environment_id", "required id")
	check(idPattern.MatchString(r.ServiceID), "server.railway.service_id", "required id")

	seen := map[string]bool{}
	for key, u := range map[string]User{"orca": c.Users.Orca, "reviewer": c.Users.Reviewer, "reviewer_ci": c.Users.ReviewerCI} {
		field := "users." + key
		check(userPattern.MatchString(u.Name), field+".name", "must be a lowercase POSIX user name")
		check(pathPattern.MatchString(u.Home), field+".home", "must be an absolute path of safe characters")
		check(u.UID >= 0, field+".uid", "must not be negative")
		for _, p := range u.ExtraPath {
			check(pathPattern.MatchString(p), field+".extra_path", "entries must be absolute paths of safe characters")
		}
		check(!seen[u.Name], field+".name", fmt.Sprintf("duplicate user name %q", u.Name))
		seen[u.Name] = true
	}

	for field, rt := range map[string]Runtime{"runtimes.coder": c.Runtimes.Coder, "runtimes.reviewer": c.Runtimes.Reviewer, "runtimes.assistant": c.Runtimes.Assistant} {
		check(rt == RuntimeClaude || rt == RuntimePi, field, fmt.Sprintf("must be %q or %q, got %q", RuntimeClaude, RuntimePi, rt))
	}
	check(repoPattern.MatchString(c.Repo.Name), "repo.name", "must be owner/name")
	check(branchPattern.MatchString(c.Repo.IntegrationBranch), "repo.integration_branch", "invalid branch name")
	check(loginPattern.MatchString(c.Bot.Login), "bot.login", "required GitHub login")

	check(len(c.Review.Checks) > 0, "review.checks", "at least one check is required")
	for _, ch := range c.Review.Checks {
		check(strings.TrimSpace(ch) != "", "review.checks", "entries must not be blank")
	}
	check(len(c.Review.HumanOnlyPaths) > 0, "review.human_only_paths", "must not be empty")
	for _, p := range c.Review.HumanOnlyPaths {
		ok := p != "" && !strings.HasPrefix(p, "/") && !slices.Contains(strings.Split(strings.TrimSuffix(p, "/"), "/"), "..")
		check(ok, "review.human_only_paths", fmt.Sprintf("entry %q must be a relative repository path", p))
	}
	check(envPattern.MatchString(c.Review.PiCredentialEnv), "review.pi_credential_env", "must be an environment variable name")
	check(c.Coder.Login == "" || loginPattern.MatchString(c.Coder.Login), "coder.login", "must be a GitHub login")
	check(filePattern.MatchString(c.Sessions.LocalOrca), "sessions.local_orca", "must be a command name or path")
	for key, s := range map[string]Session{"coder": c.Sessions.Coder, "reviewer": c.Sessions.Reviewer, "assistant": c.Sessions.Assistant} {
		check(s.PromptFile == "" || filePattern.MatchString(s.PromptFile), "sessions."+key+".prompt_file", "must be a path of safe characters")
	}
	return errors.Join(errs...)
}
