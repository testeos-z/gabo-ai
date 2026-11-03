package skills

import (
	"io/fs"
	"slices"
	"strings"
	"testing"
)

// The reviewer-configuration skill (intent 0001 S10, D4, plan T4) is the single
// catalog source every engine ships: it configures any of the four engines as
// reviewer, installs nothing itself, never runs a review, and leaves placing
// the reviewer scripts at user level to gabo init (D20).
func TestReviewerConfigSkill(t *testing.T) {
	t.Parallel()
	const name = "gabo-reviewer-config"
	if !slices.Contains(Names(), name) {
		t.Fatalf("Names() misses %s", name)
	}
	b, err := fs.ReadFile(Catalog(), name+"/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	fm, ok := frontmatter(doc)
	if !ok || fm["name"] != name {
		t.Fatalf("%s/SKILL.md: frontmatter name %q, want %q", name, fm["name"], name)
	}

	for _, s := range []string{
		// the four engines and the key that picks one.
		"`claude`", "`codex`", "`pi`", "`opencode`", "`runtimes.reviewer`",
		// missing CLI: name the engine and the CLI, install or write nothing.
		"on `PATH`", "`command -v <cli>`", "name the engine and its missing CLI",
		"install nothing and write nothing",
		// the scripts reach user level only through gabo init (D20).
		"only through `gabo init`", "`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`", "D20",
		// isolation test: only codex and opencode, never claude or pi.
		"`claude` and `pi` need no isolation test", "isolation test", "pass marker",
		"leave `runtimes.reviewer` unchanged",
		// the skill never reviews.
		"never runs a review",
		"`gabo-config-writer`", "`gabo check`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s/SKILL.md: missing %q", name, s)
		}
	}
	if strings.Contains(doc, "reviewer.sh") {
		t.Errorf("%s/SKILL.md: must not invoke reviewer.sh", name)
	}
}
