package check

import "testing"

// humanUseCase is the use case of the human-only tests: the project at root,
// the working directory work (empty means the root), and fold for a
// case-insensitive filesystem. No config or outcome is read.
func humanUseCase(s Settings) *UseCase {
	return NewUseCase(project(), nil, nil, s)
}

// I0028 S7, S13: a path is human-only when it lies under a folder entry of
// review.human_only_paths, is a file entry at the project root, or is
// gabo.yaml, which is protected even when the list omits it. A nested file of
// the same name is not, and a path outside the root is never human-only.
func TestHumanOnlyPathMatchesEntries(t *testing.T) {
	t.Parallel()
	list := []string{".github/", ".claude/", "AGENTS.md", "CLAUDE.md"}
	u := humanUseCase(Settings{Root: "/proj"})
	tests := []struct {
		path string
		want bool
	}{
		{".github/workflows/ci.yml", true},
		{".github", true},
		{"./.github/ci.yml", true},
		{".claude/settings.json", true},
		{"AGENTS.md", true},
		{"./CLAUDE.md", true},
		{"docs/AGENTS.md", false},
		{"gabo.yaml", true},
		{"sub/gabo.yaml", false},
		{"/proj/.github/ci.yml", true},
		{"/elsewhere/.github/ci.yml", false},
		{"../other/.github/ci.yml", false},
		{"../proj/gabo.yaml", true},
		{"apps/cli/main.go", false},
		{"AGENTS.mdx", false},
		{".githubx/a", false},
		{".gi*/ci.yml", true},
		{"*.md", true},
		{"AGENTS.*", true},
		{"apps/cli/*.go", false},
		{"build/*", false},
	}
	for _, tt := range tests {
		if got := u.HumanOnlyPath(tt.path, list); got != tt.want {
			t.Errorf("HumanOnlyPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// I0028 S13: gabo.yaml is human-only even when review.human_only_paths is empty.
func TestHumanOnlyPathProtectsGaboYamlAlways(t *testing.T) {
	t.Parallel()
	if !humanUseCase(Settings{Root: "/proj"}).HumanOnlyPath("gabo.yaml", nil) {
		t.Fatalf("gabo.yaml with an empty list: not human-only; want human-only")
	}
}

// I0028 S7: a relative path is read from the working directory, as PreWrite
// reads it: from apps/cli, ../../gabo.yaml is the root's gabo.yaml and
// AGENTS.md is apps/cli/AGENTS.md.
func TestHumanOnlyPathReadsTheWorkingDirectory(t *testing.T) {
	t.Parallel()
	list := []string{".github/", "AGENTS.md"}
	u := humanUseCase(Settings{Root: "/proj", WorkDir: "/proj/apps/cli"})
	tests := []struct {
		path string
		want bool
	}{
		{"../../gabo.yaml", true},
		{"../../.github/ci.yml", true},
		{"../../AGENTS.md", true},
		{"AGENTS.md", false},
		{"../../apps/cli/main.go", false},
		{"../../../proj/gabo.yaml", true},
		{"../../../other/gabo.yaml", false},
	}
	for _, tt := range tests {
		if got := u.HumanOnlyPath(tt.path, list); got != tt.want {
			t.Errorf("from apps/cli: HumanOnlyPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// I0028 S7: on a case-insensitive filesystem the names match without case; on
// a case-sensitive one they do not.
func TestHumanOnlyPathFoldsCase(t *testing.T) {
	t.Parallel()
	list := []string{".github/", "AGENTS.md"}
	tests := []struct {
		name, path string
		fold, want bool
	}{
		{"folded gabo.yaml", "GABO.yaml", true, true},
		{"folded folder", ".GitHub/workflows/ci.yml", true, true},
		{"folded file", "Agents.md", true, true},
		{"sensitive gabo.yaml", "GABO.yaml", false, false},
		{"sensitive folder", ".GitHub/ci.yml", false, false},
	}
	for _, tt := range tests {
		u := humanUseCase(Settings{Root: "/proj", FoldCase: tt.fold})
		if got := u.HumanOnlyPath(tt.path, list); got != tt.want {
			t.Errorf("%s: HumanOnlyPath(%q) = %v, want %v", tt.name, tt.path, got, tt.want)
		}
	}
}

// I0028 S7: a symlink in the tree is followed as PreWrite follows it: a write
// to gh/ci.yml, where gh is a link to .github, is a write to .github/ci.yml.
func TestHumanOnlyPathFollowsSymlinks(t *testing.T) {
	t.Parallel()
	u := humanUseCase(Settings{
		Root:    "/proj",
		Resolve: fakeResolve(map[string]string{"/proj/gh": "/proj/.github"}, "/proj", "/proj/.github", "/proj/.github/ci.yml"),
	})
	list := []string{".github/"}
	if !u.HumanOnlyPath("gh/ci.yml", list) {
		t.Fatalf("gh/ci.yml through a link to .github: not human-only; want human-only")
	}
	if u.HumanOnlyPath("apps/main.go", list) {
		t.Fatalf("apps/main.go: human-only; want allowed")
	}
}
