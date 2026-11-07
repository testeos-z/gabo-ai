package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// minimal sets every required key and nothing else; review stays last so a
// test can append a review entry.
const minimal = `
repo:
  name: cappy-lab/vulcam
  integration_branch: dev
runtimes:
  coder: claude
  reviewer: claude
  assistant: claude
parallelism: 3
review:
  checks: [go test ./...]
  human_only_paths: [.github/]
`

// with returns minimal overridden by extra, merged like a higher layer.
func with(extra string) string {
	var lower, higher map[string]any
	if err := yaml.Unmarshal([]byte(minimal), &lower); err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal([]byte(extra), &higher); err != nil {
		panic(err)
	}
	b, err := yaml.Marshal(Merge(lower, higher))
	if err != nil {
		panic(err)
	}
	return string(b)
}

func TestParseRepoSplit(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Repo.Owner() != "cappy-lab" || cfg.Repo.Repo() != "vulcam" {
		t.Errorf("repo split = %q/%q", cfg.Repo.Owner(), cfg.Repo.Repo())
	}
}

func TestParseKeepsExplicitValues(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(with(`
runtimes:
  coder: pi
  reviewer: claude
`)))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Runtimes.Coder != RuntimePi || cfg.Runtimes.Reviewer != RuntimeClaude {
		t.Errorf("runtimes = %+v", cfg.Runtimes)
	}
}

func TestParseValidationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"bad repo", strings.Replace(minimal, "cappy-lab/vulcam", "vulcam", 1), "repo.name"},
		{"bad runtime", with("runtimes:\n  coder: foo\n"), "runtimes.coder"},
		{"relative state dir", with("review:\n  state_dir: state\n"), "review.state_dir"},
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
	cfg, err := Load(filepath.Join("..", "..", "gabo.example.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Repo.Name != "cappy-lab/vulcam" || cfg.Repo.IntegrationBranch != "dev" {
		t.Fatalf("example config = %+v", cfg)
	}
	if cfg.Wave.After != AfterWait {
		t.Errorf("example wave.after = %q, want %q", cfg.Wave.After, AfterWait)
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
	if again.Repo != cfg.Repo || again.Parallelism != cfg.Parallelism {
		t.Fatalf("round trip = %+v, want %+v", again, cfg)
	}
}

func TestParseReviewAutoMergeIsOptional(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Review.AutoMerge {
		t.Error("auto_merge must be false when unset")
	}
}

func TestParseReviewExplicit(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(with("review:\n  checks: [\"go test ./...\"]\n  auto_merge: true\n  human_only_paths: [secret/]\n  pi_credential_env: OPENAI_API_KEY\n")))
	if err != nil {
		t.Fatal(err)
	}
	r := cfg.Review
	if len(r.Checks) != 1 || !r.AutoMerge || r.HumanOnlyPaths[0] != "secret/" || r.PiCredentialEnv != "OPENAI_API_KEY" {
		t.Fatalf("review = %+v", r)
	}
}

func TestParseReviewCheckEnv(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(with("review:\n  check_env: [AISHARE_TEST_DATABASE_URL]\n")))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Review.CheckEnv; len(got) != 1 || got[0] != "AISHARE_TEST_DATABASE_URL" {
		t.Fatalf("check_env = %q", got)
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
		"check env value": {"review:\n  check_env: [\"A=B\"]\n", "review.check_env: entry \"A=B\" must be an environment variable name"},
		"reserved env":    {"review:\n  check_env: [GH_TOKEN]\n", "review.check_env: entry \"GH_TOKEN\" is reserved"},
		"pi credential":   {"review:\n  pi_credential_env: PI_KEY\n  check_env: [DB_URL, PI_KEY]\n", "review.check_env: entry \"PI_KEY\" is review.pi_credential_env"},
	} {
		if _, err := Parse([]byte(with(tt.yaml))); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error = %v, want %q", name, err, tt.want)
		}
	}
}

func TestParseCoderLogin(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil || cfg.Coder.Login != "" {
		t.Fatalf("default coder = %+v, %v", cfg.Coder, err)
	}
	cfg, err = Parse([]byte(minimal + "coder:\n  login: vulcam-coder\n"))
	if err != nil || cfg.Coder.Login != "vulcam-coder" {
		t.Fatalf("explicit coder = %+v, %v", cfg.Coder, err)
	}
	if _, err := Parse([]byte(minimal + "coder:\n  login: \"bad login\"\n")); err == nil || !strings.Contains(err.Error(), "coder.login") {
		t.Errorf("bad coder.login: error = %v", err)
	}
}

// The cloud keys were removed from the config types with the work-time
// commands that read them (T042): a file still carrying one fails as an
// unknown key instead of being read through a legacy accessor.
func TestParseRejectsRemovedCloudKeys(t *testing.T) {
	t.Parallel()
	for key, doc := range map[string]string{
		"server":   "server:\n  railway:\n    project_id: p1\n",
		"users":    "users:\n  orca:\n    name: orca\n",
		"bot":      "bot:\n  login: owner\n",
		"sessions": "sessions:\n  local_orca: orca\n",
	} {
		if _, err := Parse([]byte(minimal + doc)); err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("%s: error = %v, want unknown key", key, err)
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
	cfg, err := Parse([]byte(with("parallelism: 5\n")))
	if err != nil || cfg.Parallelism != 5 {
		t.Fatalf("parallelism = %d, %v; want 5", cfg.Parallelism, err)
	}
	if _, err := Parse([]byte(with("parallelism: -1\n"))); err == nil || !strings.Contains(err.Error(), "parallelism") {
		t.Fatalf("negative parallelism: error = %v", err)
	}
}

// runtimesDoc returns minimal with its runtimes block replaced by block.
func runtimesDoc(block string) string {
	return strings.Replace(minimal, "runtimes:\n  coder: claude\n  reviewer: claude\n  assistant: claude\n", "runtimes:\n"+block, 1)
}

// linesNaming returns the lines of err that name key.
func linesNaming(err error, key string) []string {
	var out []string
	if err == nil {
		return nil
	}
	for _, l := range strings.Split(err.Error(), "\n") {
		if strings.Contains(l, key) {
			out = append(out, l)
		}
	}
	return out
}

// Intent 0013 S1, D1: runtimes.assistant takes one engine or a list, and
// runtimes.coder and runtimes.reviewer take one engine or session.
func TestParseAcceptsAssistantListAndSession(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(runtimesDoc("  assistant: [claude, pi]\n  coder: session\n  reviewer: pi\n")))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := Runtimes{Coder: RuntimeSession, Reviewer: RuntimePi, Assistant: RuntimeList{RuntimeClaude, RuntimePi}}
	if !reflect.DeepEqual(cfg.Runtimes, want) {
		t.Fatalf("runtimes = %+v, want %+v", cfg.Runtimes, want)
	}
	cfg, err = Parse([]byte(runtimesDoc("  assistant: [opencode]\n  coder: claude\n  reviewer: session\n")))
	if err != nil || cfg.Runtimes.Reviewer != RuntimeSession {
		t.Fatalf("reviewer session: runtimes = %+v, %v", cfg.Runtimes, err)
	}
}

// A scalar assistant decodes to a one-item list.
func TestParseScalarAssistantIsOneItemList(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Runtimes.Assistant, RuntimeList{RuntimeClaude}) {
		t.Fatalf("assistant = %#v, want [claude]", cfg.Runtimes.Assistant)
	}
	if got := cfg.Runtimes.Assistant.Single(); got != RuntimeClaude {
		t.Errorf("Single() = %q, want claude", got)
	}
	if got := (RuntimeList{RuntimeClaude, RuntimePi}).Single(); got != "" {
		t.Errorf("Single() of a list = %q, want empty", got)
	}
}

// Each refused assistant shape gives exactly one failure line naming
// runtimes.assistant, with its next step.
func TestParseRefusesBadAssistantList(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct{ block, want string }{
		"empty list": {"  coder: claude\n  reviewer: claude\n  assistant: []\n",
			`runtimes.assistant: must list at least one engine; next: set one of "claude", "codex", "pi", "opencode" or a list of them`},
		"duplicate": {"  coder: claude\n  reviewer: claude\n  assistant: [pi, pi]\n",
			`runtimes.assistant: "pi" is listed more than once; next: list each engine once`},
		"unknown engine": {"  coder: claude\n  reviewer: claude\n  assistant: [claude, gpt]\n",
			`runtimes.assistant: entry "gpt" must be one of "claude", "codex", "pi", "opencode"; next: replace or remove it`},
		"session": {"  coder: claude\n  reviewer: claude\n  assistant: session\n",
			`runtimes.assistant: "session" is valid only in runtimes.coder and runtimes.reviewer; next: set the engine of the main session`},
	} {
		_, err := Parse([]byte(runtimesDoc(tt.block)))
		if got := linesNaming(err, "runtimes.assistant"); len(got) != 1 || got[0] != tt.want {
			t.Errorf("%s: lines naming runtimes.assistant = %q, want [%q]", name, got, tt.want)
		}
	}
}

// A single value keeps its 0.2.10 meaning and messages.
func TestParseSingleValueMessagesUnchanged(t *testing.T) {
	t.Parallel()
	for _, role := range []string{"coder", "reviewer", "assistant"} {
		_, err := Parse([]byte(strings.Replace(minimal, role+": claude", role+": foo", 1)))
		want := `runtimes.` + role + `: must be one of "claude", "codex", "pi", "opencode", got "foo"`
		if got := linesNaming(err, "runtimes."+role); len(got) != 1 || got[0] != want {
			t.Errorf("%s foo: lines = %q, want [%q]", role, got, want)
		}
	}
	_, err := Parse([]byte(strings.Replace(minimal, "assistant: claude", `assistant: ""`, 1)))
	if got := linesNaming(err, "runtimes.assistant"); len(got) != 1 || got[0] != "✗ runtimes.assistant missing" {
		t.Errorf(`assistant "": lines = %q, want the missing line`, got)
	}
}

// Marshal writes a one-item assistant as a scalar and an unset one as null,
// so a round trip keeps the single-value shape and the missing key.
func TestMarshalAssistantShapes(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		in   RuntimeList
		want string
	}{
		{RuntimeList{RuntimeClaude}, "assistant: claude\n"},
		{RuntimeList{RuntimeClaude, RuntimePi}, "assistant:\n        - claude\n        - pi\n"},
		{nil, "assistant: null\n"},
	} {
		b, err := Config{Runtimes: Runtimes{Assistant: tt.in}}.Marshal()
		if err != nil || !strings.Contains(string(b), tt.want) {
			t.Errorf("Marshal(%q) = %s, %v; want it to contain %q", tt.in, b, err, tt.want)
		}
	}
}
