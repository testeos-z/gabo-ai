package config

import (
	"strings"
	"testing"
)

// dbSchemaExample is the project gabo.yaml example of DB_SCHEMA 4.2, with a
// Claude reviewer (the DB_SCHEMA wording follows in T075).
const dbSchemaExample = `# yaml-language-server: $schema=https://raw.githubusercontent.com/testeos-z/gabo-ai/main/schemas/gabo.project.schema.json
version: 1
mode: manual

repo:
  name: testeos-z/gabo-ai
  integration_branch: dev

runtimes:
  coder: claude
  assistant: claude
  reviewer: claude

slots:
  tasks:
    adapter: .gabo/adapters/tasks-gh.md
  memory:
    adapter: .gabo/adapters/memory-engram.md
  notifications:
    adapter: .gabo/adapters/notify-task-comment.md
  direct_sessions:
    adapter: .gabo/adapters/sessions-orca.md
  methodology:
    adapter: .gabo/adapters/methodology-tdd.md
  repository_host:
    adapter: .gabo/adapters/host-github.md
  reviewer_placement:
    adapter: .gabo/adapters/reviewer-local.md

reviewer:
  placement: local
  signing:
    marker: gabo-review

review:
  checks:
    - go test ./...
    - go vet ./...
  auto_merge: false
  human_only_paths: [.github/, AGENTS.md, CLAUDE.md]
  pi_credential_env: ANTHROPIC_API_KEY

developers:
  "leo@example.com": Leo

plan_change:
  scope: group
  global_without_owner: false

parallelism: 3
`

func TestParseDBSchemaExample(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(dbSchemaExample))
	if err != nil {
		t.Fatalf("Parse(DB_SCHEMA 4.2 example) error = %v", err)
	}
	if cfg.Mode != ModeManual {
		t.Errorf("mode = %q, want manual", cfg.Mode)
	}
	if got := len(cfg.Slots); got != 7 {
		t.Errorf("slots = %d, want 7", got)
	}
	if got := cfg.Slots[SlotMethodology].Adapter; got != ".gabo/adapters/methodology-tdd.md" {
		t.Errorf("slots.methodology.adapter = %q", got)
	}
	if cfg.Reviewer.Placement != PlacementLocal || cfg.Reviewer.Signing.Marker != "gabo-review" {
		t.Errorf("reviewer = %+v", cfg.Reviewer)
	}
	if cfg.Developers["leo@example.com"] != "Leo" {
		t.Errorf("developers = %v", cfg.Developers)
	}
	if cfg.PlanChange.Scope != ScopeGroup || cfg.PlanChange.GlobalWithoutOwner {
		t.Errorf("plan_change = %+v", cfg.PlanChange)
	}
}

func TestParseAcceptsEverySlotName(t *testing.T) {
	t.Parallel()
	if len(SlotNames) != 8 {
		t.Fatalf("SlotNames = %v, want the eight TRD 9.1 slots", SlotNames)
	}
	var b strings.Builder
	b.WriteString(minimal + "slots:\n")
	for _, s := range SlotNames {
		b.WriteString("  " + s + ":\n    adapter: .gabo/adapters/" + s + ".md\n")
	}
	cfg, err := Parse([]byte(b.String()))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Slots) != 8 {
		t.Fatalf("slots = %v", cfg.Slots)
	}
}

func TestParseRejectsBadProjectKeys(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{"bad mode", "mode: yolo\n", []string{"mode", "manual", "auto", "yolo"}},
		{"unknown slot", "slots:\n  tracker:\n    adapter: a.md\n", []string{"slots.tracker", "unknown slot"}},
		{"empty adapter", "slots:\n  tasks:\n    adapter: \"\"\n", []string{"slots.tasks.adapter"}},
		{"absolute adapter", "slots:\n  tasks:\n    adapter: /etc/a.md\n", []string{"slots.tasks.adapter"}},
		{"escaping adapter", "slots:\n  tasks:\n    adapter: ../a.md\n", []string{"slots.tasks.adapter"}},
		{"unknown slot field", "slots:\n  tasks:\n    path: a.md\n", []string{"path"}},
		{"later placement", "reviewer:\n  placement: server\n", []string{"reviewer.placement", "later release"}},
		{"ci placement", "reviewer:\n  placement: github-actions\n", []string{"reviewer.placement", "later release"}},
		{"bad placement", "reviewer:\n  placement: cloud\n", []string{"reviewer.placement", "cloud"}},
		{"bad marker", "reviewer:\n  signing:\n    marker: \"a b\"\n", []string{"reviewer.signing.marker"}},
		{"unknown signing field", "reviewer:\n  signing:\n    key: x\n", []string{"key"}},
		{"bad developer email", "developers:\n  leo: Leo\n", []string{"developers", "leo"}},
		{"blank developer alias", "developers:\n  \"leo@example.com\": \"\"\n", []string{"developers.leo@example.com"}},
		{"bad scope", "plan_change:\n  scope: everything\n", []string{"plan_change.scope", "everything"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(minimal + tt.yaml))
			if err == nil {
				t.Fatalf("Parse() error = nil, want error mentioning %q", tt.want)
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("Parse() error = %v, want it to mention %q", err, w)
				}
			}
		})
	}
}

func TestParseNewProjectKeysOptional(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Mode != "" || cfg.Slots != nil || cfg.Reviewer.Placement != "" {
		t.Errorf("new keys got values without being set: %+v", cfg)
	}
}

// S11: wave.after says what the assistant does when a wave (one intent)
// closes: start the next eligible intent or wait. Unset reads as wait.
func TestParseWaveAfter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, yaml, set, want string
	}{
		{"next", "wave:\n  after: next\n", AfterNext, AfterNext},
		{"wait", "wave:\n  after: wait\n", AfterWait, AfterWait},
		{"missing key", "", "", AfterWait},
		{"empty wave", "wave: {}\n", "", AfterWait},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Parse([]byte(minimal + tt.yaml))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if cfg.Wave.After != tt.set {
				t.Errorf("wave.after = %q, want %q (Parse fills no value)", cfg.Wave.After, tt.set)
			}
			if got := cfg.AfterWave(); got != tt.want {
				t.Errorf("AfterWave() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRejectsBadWaveAfter(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(minimal + "wave:\n  after: later\n"))
	if err == nil {
		t.Fatal("Parse() error = nil, want wave.after rejected")
	}
	want := `wave.after: must be "next" or "wait", got "later"`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("Parse() error = %v, want it to contain %q", err, want)
	}
}

// A config written before wave.after existed loads unchanged and marshals
// without a wave key.
func TestConfigWithoutWaveLoadsUnchanged(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(dbSchemaExample))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Wave != (Wave{}) {
		t.Errorf("wave = %+v, want zero", cfg.Wave)
	}
	if cfg.Parallelism != 3 || cfg.PlanChange.Scope != ScopeGroup {
		t.Errorf("parallelism = %d, plan_change = %+v", cfg.Parallelism, cfg.PlanChange)
	}
	out, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "wave") {
		t.Errorf("Marshal() added a wave key:\n%s", out)
	}
	again, err := Parse(out)
	if err != nil {
		t.Fatalf("Parse(Marshal()) error = %v", err)
	}
	if again.AfterWave() != AfterWait {
		t.Errorf("AfterWave() after round trip = %q", again.AfterWave())
	}
}

// Intent 0020 S1: three optional integer caps under wave. Unset is nil and
// changes nothing; no default fills a value.
func TestParseWaveCaps(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal + "wave:\n  max_coder_turns: 30\n  max_coders: 1\n  max_fix_rounds: 4\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	w := cfg.Wave
	if w.MaxCoderTurns == nil || *w.MaxCoderTurns != 30 ||
		w.MaxCoders == nil || *w.MaxCoders != 1 ||
		w.MaxFixRounds == nil || *w.MaxFixRounds != 4 {
		t.Errorf("wave = %+v, want 30, 1, 4", w)
	}
	cfg, err = Parse([]byte(minimal + "wave:\n  after: next\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Wave.MaxCoderTurns != nil || cfg.Wave.MaxCoders != nil || cfg.Wave.MaxFixRounds != nil {
		t.Errorf("unset caps got values: %+v", cfg.Wave)
	}
	out, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "max_") {
		t.Errorf("Marshal() added a cap key:\n%s", out)
	}
}

func TestParseRejectsBadWaveCaps(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, yaml, want string }{
		{"coders zero", "max_coders: 0", "wave.max_coders: must be at least 1, got 0"},
		{"turns zero", "max_coder_turns: 0", "wave.max_coder_turns: must be at least 1, got 0"},
		{"turns negative", "max_coder_turns: -2", "wave.max_coder_turns: must be at least 1, got -2"},
		{"rounds zero", "max_fix_rounds: 0", "wave.max_fix_rounds: must be from 1 to 4, got 0"},
		{"rounds five", "max_fix_rounds: 5", "wave.max_fix_rounds: must be from 1 to 4, got 5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(minimal + "wave:\n  " + tt.yaml + "\n"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
	if _, err := Parse([]byte(minimal + "wave:\n  max_coders: many\n")); err == nil {
		t.Error("Parse() error = nil for a non-integer cap")
	}
}
