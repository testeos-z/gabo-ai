package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

const minimal = `
server:
  railway:
    project_id: p1
    environment_id: e1
    service_id: s1
repo:
  name: cappy-lab/vulcam
bot:
  login: vulcambot
`

func TestParseAppliesDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Users.Orca.Name != "orca" || cfg.Users.Orca.Home != "/data/home" {
		t.Errorf("orca = %+v", cfg.Users.Orca)
	}
	if len(cfg.Users.Orca.ExtraPath) != 1 || cfg.Users.Orca.ExtraPath[0] != "/data/home/.local/bin" {
		t.Errorf("orca extra path = %v", cfg.Users.Orca.ExtraPath)
	}
	if all := cfg.Users.All(); len(all) != 1 || all[0].Name != "orca" {
		t.Errorf("single-user mode: users = %+v, want only orca", all)
	}
	if r := cfg.Runtimes; r.Coder != RuntimeClaude || r.Reviewer != RuntimePi || r.Assistant != RuntimeClaude {
		t.Errorf("runtimes = %+v", r)
	}
	if cfg.Repo.IntegrationBranch != "dev" {
		t.Errorf("integration branch = %q, want dev", cfg.Repo.IntegrationBranch)
	}
	if cfg.Repo.Owner() != "cappy-lab" || cfg.Repo.Repo() != "vulcam" {
		t.Errorf("repo split = %q/%q", cfg.Repo.Owner(), cfg.Repo.Repo())
	}
}

func TestParseKeepsExplicitValues(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal + `
runtimes:
  coder: pi
  reviewer: claude
users:
  orca:
    home: /srv/home
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Runtimes.Coder != RuntimePi || cfg.Runtimes.Reviewer != RuntimeClaude {
		t.Errorf("runtimes = %+v", cfg.Runtimes)
	}
	if cfg.Users.Orca.Home != "/srv/home" || cfg.Users.Orca.Name != "orca" {
		t.Errorf("orca = %+v", cfg.Users.Orca)
	}
}

func TestParseValidationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"missing project", strings.Replace(minimal, "project_id: p1", "project_id: \"\"", 1), "server.railway.project_id"},
		{"missing service", strings.Replace(minimal, "service_id: s1", "service_id: \"\"", 1), "server.railway.service_id"},
		{"bad repo", strings.Replace(minimal, "cappy-lab/vulcam", "vulcam", 1), "repo.name"},
		{"bad bot", strings.Replace(minimal, "login: vulcambot", "login: \"bad login\"", 1), "bot.login"},
		{"bad runtime", minimal + "runtimes:\n  coder: opencode\n", "runtimes.coder"},
		{"relative home", minimal + "users:\n  orca:\n    home: data/home\n", "users.orca.home"},
		{"unsafe user name", minimal + "users:\n  orca:\n    name: \"rev;id\"\n", "users.orca.name"},
		{"removed reviewer user", minimal + "users:\n  reviewer:\n    name: reviewer\n", "reviewer"},
		{"relative state dir", minimal + "review:\n  state_dir: state\n", "review.state_dir"},
		{"coder is the owner", minimal + "coder:\n  login: VulcamBot\n", "coder.login"},
		{"unknown field", minimal + "extra: 1\n", "extra"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tt.yaml))
			if err == nil {
				t.Fatalf("Parse() error = nil, want error mentioning %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Parse() error = %v, want it to mention %q", err, tt.want)
			}
		})
	}
}

func TestLoadExampleFile(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join("..", "..", "gbai.example.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Railway.ServiceID != "fb7c97ca-82e3-4554-8463-1d00e95fa57f" || cfg.Repo.Name != "cappy-lab/vulcam" {
		t.Fatalf("example config = %+v", cfg)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err == nil || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load() error = %v, want not-exist", err)
	}
}

func TestMarshalRoundTrips(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	b, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	again, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse(Marshal()) error = %v\n%s", err, b)
	}
	if again.Repo != cfg.Repo || again.Users.Orca.Home != cfg.Users.Orca.Home {
		t.Fatalf("round trip = %+v, want %+v", again, cfg)
	}
}

func TestParseReviewDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	r := cfg.Review
	if len(r.Checks) != 5 || r.Checks[0] != "bun install --frozen-lockfile" || r.Checks[4] != "bun run test" {
		t.Errorf("checks = %q", r.Checks)
	}
	if r.AutoMerge {
		t.Error("auto_merge must default to false")
	}
	if strings.Join(r.HumanOnlyPaths, ",") != ".claude/,.github/,.opencode/,AGENTS.md,CLAUDE.md,docs/policies/,tools/hook-tests/,agents/" {
		t.Errorf("human_only_paths = %q", r.HumanOnlyPaths)
	}
	if r.PiCredentialEnv != "ANTHROPIC_API_KEY" {
		t.Errorf("pi_credential_env = %q", r.PiCredentialEnv)
	}
}

func TestParseReviewExplicit(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal + "review:\n  checks: [\"go test ./...\"]\n  auto_merge: true\n  human_only_paths: [secret/]\n  pi_credential_env: OPENAI_API_KEY\n"))
	if err != nil {
		t.Fatal(err)
	}
	r := cfg.Review
	if len(r.Checks) != 1 || !r.AutoMerge || r.HumanOnlyPaths[0] != "secret/" || r.PiCredentialEnv != "OPENAI_API_KEY" {
		t.Fatalf("review = %+v", r)
	}
	if argv := r.CheckArgv(); len(argv) != 1 || strings.Join(argv[0], "|") != "go|test|./..." {
		t.Fatalf("CheckArgv() = %q", argv)
	}
}

func TestParseReviewValidation(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct{ yaml, want string }{
		"empty checks":    {"review:\n  checks: []\n", "review.checks"},
		"blank check":     {"review:\n  checks: [\"  \"]\n", "review.checks"},
		"absolute path":   {"review:\n  human_only_paths: [/etc/]\n", "review.human_only_paths"},
		"traversal path":  {"review:\n  human_only_paths: [../x]\n", "review.human_only_paths"},
		"empty path list": {"review:\n  human_only_paths: []\n", "review.human_only_paths"},
		"bad env":         {"review:\n  pi_credential_env: \"A=B\"\n", "review.pi_credential_env"},
	} {
		if _, err := Parse([]byte(minimal + tt.yaml)); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error = %v, want %q", name, err, tt.want)
		}
	}
}

func TestParseSessionsAndCoder(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Sessions.LocalOrca != "orca" || cfg.Coder.Login != "" {
		t.Fatalf("defaults: sessions = %+v coder = %+v", cfg.Sessions, cfg.Coder)
	}
	cfg, err = Parse([]byte(minimal + "coder:\n  login: vulcam-coder\nsessions:\n  local_orca: orca-ide\n  coder:\n    prompt_file: /data/home/prompts/coder.md\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Coder.Login != "vulcam-coder" || cfg.Sessions.LocalOrca != "orca-ide" || cfg.Sessions.Coder.PromptFile != "/data/home/prompts/coder.md" {
		t.Fatalf("explicit: sessions = %+v coder = %+v", cfg.Sessions, cfg.Coder)
	}
	for name, doc := range map[string]string{
		"sessions.coder.prompt_file": minimal + "sessions:\n  coder:\n    prompt_file: \"a;b\"\n",
		"sessions.local_orca":        minimal + "sessions:\n  local_orca: \"orca ide\"\n",
		"coder.login":                minimal + "coder:\n  login: \"bad login\"\n",
	} {
		if _, err := Parse([]byte(doc)); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func TestParseBotLoginIsOptional(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(strings.Replace(minimal, "login: vulcambot", "login: \"\"", 1)))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Bot.Login != "" {
		t.Fatalf("bot.login = %q, want empty (defaults to the gh login at run time)", cfg.Bot.Login)
	}
}

func TestReviewStateDir(t *testing.T) {
	t.Parallel()
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	for name, tt := range map[string]struct {
		review Review
		env    map[string]string
		want   string
		err    bool
	}{
		"configured":    {Review{StateDir: "/srv/state"}, map[string]string{"XDG_STATE_HOME": "/x", "HOME": "/h"}, "/srv/state", false},
		"xdg":           {Review{}, map[string]string{"XDG_STATE_HOME": "/x", "HOME": "/h"}, "/x/gbai/reviewer", false},
		"home":          {Review{}, map[string]string{"HOME": "/data/home"}, "/data/home/.local/state/gbai/reviewer", false},
		"relative xdg":  {Review{}, map[string]string{"XDG_STATE_HOME": "x", "HOME": "/h"}, "/h/.local/state/gbai/reviewer", false},
		"no home":       {Review{}, map[string]string{}, "", true},
		"relative home": {Review{}, map[string]string{"HOME": "h"}, "", true},
	} {
		got, err := tt.review.StateDirFrom(env(tt.env))
		if got != tt.want || (err != nil) != tt.err {
			t.Errorf("%s: StateDirFrom() = %q, %v; want %q (error %v)", name, got, err, tt.want, tt.err)
		}
	}
}

func TestParseWorktreeKeep(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil || len(cfg.Worktree.Keep) != 0 {
		t.Fatalf("default keep = %q, %v", cfg.Worktree.Keep, err)
	}
	cfg, err = Parse([]byte(minimal + "worktree:\n  keep: [pr-399, /src/repo-worktrees/demo]\n"))
	if err != nil || strings.Join(cfg.Worktree.Keep, ",") != "pr-399,/src/repo-worktrees/demo" {
		t.Fatalf("keep = %q, %v", cfg.Worktree.Keep, err)
	}
	for _, bad := range []string{`""`, "a/b", "../x", `"a\nb"`} {
		if _, err := Parse([]byte(minimal + "worktree:\n  keep: [" + bad + "]\n")); err == nil || !strings.Contains(err.Error(), "worktree.keep") {
			t.Errorf("keep %s: error = %v", bad, err)
		}
	}
}

func TestParseParallelism(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil || cfg.Parallelism != 3 {
		t.Fatalf("default parallelism = %d, %v; want 3", cfg.Parallelism, err)
	}
	cfg, err = Parse([]byte(minimal + "parallelism: 5\n"))
	if err != nil || cfg.Parallelism != 5 {
		t.Fatalf("parallelism = %d, %v; want 5", cfg.Parallelism, err)
	}
	if _, err := Parse([]byte(minimal + "parallelism: -1\n")); err == nil || !strings.Contains(err.Error(), "parallelism") {
		t.Fatalf("negative parallelism: error = %v", err)
	}
}
