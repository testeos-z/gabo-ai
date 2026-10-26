// Package config loads and validates gbai.yaml.
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
	Worktree Worktree `yaml:"worktree"`
	// Parallelism is the most coder sessions the cycle runs at once; it
	// already leaves room for the reviewer's checks. 0 means the default, 3.
	Parallelism int `yaml:"parallelism"`
}

// Worktree configures `gbai worktree close`.
type Worktree struct {
	// Keep lists worktree names (or absolute paths) that are never closed.
	Keep []string `yaml:"keep,omitempty"`
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

// Users lists the OS users on the server. Single-user mode: the reviewer
// runs as the orca user, the same OS user as the agent sessions.
type Users struct {
	Orca User `yaml:"orca"`
}

// All returns the users in a stable order.
func (u Users) All() []User { return []User{u.Orca} }

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

// Bot is the owner's GitHub login: in single-user mode the reviewer posts,
// sets statuses and merges with the owner's own gh login. Empty means the
// login gh is authenticated as; when set it must equal that login.
type Bot struct {
	Login string `yaml:"login"`
}

// Review configures `gbai review run` and its checks side.
type Review struct {
	// Checks are run in order inside a copy of the merged tree, without
	// GitHub credentials. Each is split on whitespace into argv; no shell is
	// involved.
	Checks         []string `yaml:"checks"`
	AutoMerge      bool     `yaml:"auto_merge"`
	HumanOnlyPaths []string `yaml:"human_only_paths"`
	// PiCredentialEnv names the env var that receives the pi model credential.
	PiCredentialEnv string `yaml:"pi_credential_env"`
	// StateDir holds the reviewer's own clone, worktrees and check
	// workspaces. Empty means $XDG_STATE_HOME/gbai/reviewer, else
	// ~/.local/state/gbai/reviewer, resolved on the host that runs the review.
	StateDir string `yaml:"state_dir,omitempty"`
}

// StateDirFrom resolves the reviewer state dir with getenv (os.Getenv).
func (r Review) StateDirFrom(getenv func(string) string) (string, error) {
	if r.StateDir != "" {
		return r.StateDir, nil
	}
	if xdg := getenv("XDG_STATE_HOME"); path.IsAbs(xdg) {
		return path.Join(xdg, "gbai", "reviewer"), nil
	}
	if home := getenv("HOME"); path.IsAbs(home) {
		return path.Join(home, ".local", "state", "gbai", "reviewer"), nil
	}
	return "", errors.New("reviewer state dir: set review.state_dir, XDG_STATE_HOME or HOME")
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
	if c.Parallelism == 0 {
		c.Parallelism = 3
	}
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

	u := c.Users.Orca
	check(userPattern.MatchString(u.Name), "users.orca.name", "must be a lowercase POSIX user name")
	check(pathPattern.MatchString(u.Home), "users.orca.home", "must be an absolute path of safe characters")
	check(u.UID >= 0, "users.orca.uid", "must not be negative")
	for _, p := range u.ExtraPath {
		check(pathPattern.MatchString(p), "users.orca.extra_path", "entries must be absolute paths of safe characters")
	}

	for field, rt := range map[string]Runtime{"runtimes.coder": c.Runtimes.Coder, "runtimes.reviewer": c.Runtimes.Reviewer, "runtimes.assistant": c.Runtimes.Assistant} {
		check(rt == RuntimeClaude || rt == RuntimePi, field, fmt.Sprintf("must be %q or %q, got %q", RuntimeClaude, RuntimePi, rt))
	}
	check(repoPattern.MatchString(c.Repo.Name), "repo.name", "must be owner/name")
	check(branchPattern.MatchString(c.Repo.IntegrationBranch), "repo.integration_branch", "invalid branch name")
	check(c.Bot.Login == "" || loginPattern.MatchString(c.Bot.Login), "bot.login", "must be the owner's GitHub login")

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
	check(c.Coder.Login == "" || !strings.EqualFold(c.Coder.Login, c.Bot.Login), "coder.login",
		"must differ from bot.login: the audit flags every merge by the coder login")
	check(c.Review.StateDir == "" || pathPattern.MatchString(c.Review.StateDir), "review.state_dir", "must be an absolute path of safe characters")
	check(filePattern.MatchString(c.Sessions.LocalOrca), "sessions.local_orca", "must be a command name or path")
	for key, s := range map[string]Session{"coder": c.Sessions.Coder, "reviewer": c.Sessions.Reviewer, "assistant": c.Sessions.Assistant} {
		check(s.PromptFile == "" || filePattern.MatchString(s.PromptFile), "sessions."+key+".prompt_file", "must be a path of safe characters")
	}
	check(c.Parallelism > 0, "parallelism", "must be a positive number of coder sessions")
	for _, k := range c.Worktree.Keep {
		ok := k != "" && !strings.ContainsAny(k, "\n\r") && (path.IsAbs(k) || !strings.Contains(k, "/")) && k != ".." && k != "."
		check(ok, "worktree.keep", fmt.Sprintf("entry %q must be a worktree name or an absolute path", k))
	}
	return errors.Join(errs...)
}
