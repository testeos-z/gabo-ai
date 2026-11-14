package check

import "testing"

// I0028 S7, S13: a path is human-only when it lies under a folder entry of
// review.human_only_paths, is a file entry at the project root, or is
// gabo.yaml, which is protected even when the list omits it. A nested file of
// the same name is not, and a path outside the root is never human-only.
func TestHumanOnlyPathMatchesEntries(t *testing.T) {
	t.Parallel()
	list := []string{".github/", ".claude/", "AGENTS.md", "CLAUDE.md"}
	tests := []struct {
		path string
		want bool
	}{
		{".github/workflows/ci.yml", true},
		{".github", true},
		{"./.github/ci.yml", true},
		{".claude/settings.json", true},
		{"AGENTS.md", true},
		{"docs/AGENTS.md", false},
		{"gabo.yaml", true},
		{"sub/gabo.yaml", false},
		{"/proj/.github/ci.yml", true},
		{"/elsewhere/.github/ci.yml", false},
		{"../proj/.github/ci.yml", false},
		{"apps/cli/main.go", false},
		{"AGENTS.mdx", false},
		{".githubx/a", false},
		{".gi*/ci.yml", true},
		{"*.md", true},
		{"apps/cli/*.go", false},
		{"build/*", false},
		{"AGENTS.*", true},
	}
	for _, tt := range tests {
		if got := HumanOnlyPath(tt.path, "/proj", false, list); got != tt.want {
			t.Errorf("HumanOnlyPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// I0028 S13: gabo.yaml is human-only even when review.human_only_paths is empty.
func TestHumanOnlyPathProtectsGaboYamlAlways(t *testing.T) {
	t.Parallel()
	if !HumanOnlyPath("gabo.yaml", "/proj", false, nil) {
		t.Fatalf("gabo.yaml with an empty list: not human-only; want human-only")
	}
}
