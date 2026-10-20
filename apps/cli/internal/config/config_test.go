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
	if u := cfg.Users.Reviewer; u.Name != "reviewer" || u.Home != "/data/reviewer" || u.UID != 2000 {
		t.Errorf("reviewer = %+v", u)
	}
	if u := cfg.Users.ReviewerCI; u.Name != "reviewer-ci" || u.Home != "/data/reviewer-ci" || u.UID != 2001 {
		t.Errorf("reviewer-ci = %+v", u)
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
  reviewer:
    home: /srv/reviewer
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Runtimes.Coder != RuntimePi || cfg.Runtimes.Reviewer != RuntimeClaude {
		t.Errorf("runtimes = %+v", cfg.Runtimes)
	}
	if cfg.Users.Reviewer.Home != "/srv/reviewer" || cfg.Users.Reviewer.UID != 2000 {
		t.Errorf("reviewer = %+v", cfg.Users.Reviewer)
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
		{"missing bot", strings.Replace(minimal, "login: vulcambot", "login: \"\"", 1), "bot.login"},
		{"bad runtime", minimal + "runtimes:\n  coder: opencode\n", "runtimes.coder"},
		{"relative home", minimal + "users:\n  orca:\n    home: data/home\n", "users.orca.home"},
		{"unsafe user name", minimal + "users:\n  reviewer:\n    name: \"rev;id\"\n", "users.reviewer.name"},
		{"duplicate user", minimal + "users:\n  reviewer_ci:\n    name: reviewer\n", "duplicate"},
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
	if again.Repo != cfg.Repo || again.Users.Reviewer.UID != cfg.Users.Reviewer.UID {
		t.Fatalf("round trip = %+v, want %+v", again, cfg)
	}
}
