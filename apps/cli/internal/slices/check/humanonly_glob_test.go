package check

import "testing"

// I0028 S7, S13 (review round 5): a glob that can match a file entry in a
// nested folder is judged, as the literal rule judges the nested file: a file
// entry covers any path that ends with its name, so a pattern whose last
// components match the entry can write a human-only path. gabo.yaml is the
// root file only, so a nested gabo.yaml glob stays allowed.
func TestHumanOnlyShellWritesGlobNestedFileEntry(t *testing.T) {
	t.Parallel()
	u := humanUseCase(Settings{Root: "/proj"})
	list := []string{"AGENTS.md", "opencode.json"}
	blocked := []string{
		"rm -f apps/cli/*.md",
		"sed -i s/a/b/ docs/AGENTS.*",
		"rm */*.json",
		"rm -f apps/*",
		"rm -f gab*.yaml",
	}
	for _, c := range blocked {
		if got := u.humanOnlyShellWrites(c, true, list); len(got) == 0 {
			t.Errorf("coder: %q is allowed; want blocked", c)
		}
	}
	allowed := []string{
		"rm -f apps/cli/*.go",
		"rm -f build/*.log",
		"rm -f */gabo.yaml",
		"rm -f one.txt",
	}
	for _, c := range allowed {
		if got := u.humanOnlyShellWrites(c, true, list); len(got) != 0 {
			t.Errorf("coder: %q is blocked by %v; want allowed", c, got)
		}
	}
}

// I0028 S13 (review round 6): on a case-insensitive filesystem a glob is
// matched without case, as a literal path is: rm -f AGENTS.* removes AGENTS.md,
// so it is blocked; the fold fails closed.
func TestHumanOnlyShellWritesGlobFoldsCase(t *testing.T) {
	t.Parallel()
	u := humanUseCase(Settings{Root: "/proj", FoldCase: true})
	blocked := []string{"rm -f AGENTS.*", "sed -i s/a/b/ CLAUDE.*", "rm -f GAB*.YAML"}
	for _, c := range blocked {
		if got := u.humanOnlyShellWrites(c, true, humanList); len(got) == 0 {
			t.Errorf("fold: %q is allowed; want blocked", c)
		}
	}
	if got := u.humanOnlyShellWrites("rm -f APPS/*.GO", true, humanList); len(got) != 0 {
		t.Errorf("fold: APPS/*.GO is blocked by %v; want allowed", got)
	}
}
