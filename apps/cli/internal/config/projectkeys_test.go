package config

import (
	"strings"
	"testing"
)

// dbSchemaExample is the project gbai.yaml example of DB_SCHEMA 4.2.
const dbSchemaExample = `# yaml-language-server: $schema=https://raw.githubusercontent.com/testeos-z/gbai-team/main/schemas/gbai.project.schema.json
version: 1
mode: manual

repo:
  name: testeos-z/gbai-team
  integration_branch: dev

runtimes:
  coder: claude
  assistant: claude
  reviewer: pi

generation:
  targets: [claude, pi]

slots:
  tasks:
    adapter: .gbai/adapters/tasks-gh.md
  memory:
    adapter: .gbai/adapters/memory-engram.md
  notifications:
    adapter: .gbai/adapters/notify-task-comment.md
  direct_sessions:
    adapter: .gbai/adapters/sessions-orca.md
  methodology:
    adapter: .gbai/adapters/methodology-tdd.md
  repository_host:
    adapter: .gbai/adapters/host-github.md
  reviewer_placement:
    adapter: .gbai/adapters/reviewer-local.md

reviewer:
  placement: local
  signing:
    marker: gbai-review

review:
  checks:
    - go test ./...
    - go vet ./...
  auto_merge: false
  human_only_paths: [.github/, AGENTS.md, CLAUDE.md]

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
	if got := cfg.Slots[SlotMethodology].Adapter; got != ".gbai/adapters/methodology-tdd.md" {
		t.Errorf("slots.methodology.adapter = %q", got)
	}
	if cfg.Reviewer.Placement != PlacementLocal || cfg.Reviewer.Signing.Marker != "gbai-review" {
		t.Errorf("reviewer = %+v", cfg.Reviewer)
	}
	if cfg.Developers["leo@example.com"] != "Leo" {
		t.Errorf("developers = %v", cfg.Developers)
	}
	if cfg.PlanChange.Scope != ScopeGroup || cfg.PlanChange.GlobalWithoutOwner {
		t.Errorf("plan_change = %+v", cfg.PlanChange)
	}
	if got := cfg.Generation.Targets; len(got) != 2 || got[0] != RuntimeClaude || got[1] != RuntimePi {
		t.Errorf("generation.targets = %v", got)
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
		b.WriteString("  " + s + ":\n    adapter: .gbai/adapters/" + s + ".md\n")
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
		{"bad target", "generation:\n  targets: [claude, opencode]\n", []string{"generation.targets", "opencode"}},
		{"duplicate target", "generation:\n  targets: [pi, pi]\n", []string{"generation.targets", "duplicate"}},
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
	if cfg.Mode != "" || cfg.Slots != nil || cfg.Reviewer.Placement != "" || cfg.Generation.Targets != nil {
		t.Errorf("new keys got values without being set: %+v", cfg)
	}
}
