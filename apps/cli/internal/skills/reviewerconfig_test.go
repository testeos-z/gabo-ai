package skills

import (
	"io/fs"
	"os/exec"
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
		// isolation test: only codex and opencode, through the script, once, at
		// configuration time; a fail refuses the engine (plan T5).
		"`claude` and `pi` need no isolation test", "Isolation test (`codex`, `opencode` only)", "pass marker",
		"record no pass marker", "leave `runtimes.reviewer` unchanged",
		"`bash <skill dir>/scripts/isolation-test.sh <engine>`", "exactly once", "never during a wave",
		"`${XDG_CONFIG_HOME:-$HOME/.config}/gabo/reviewer-isolation/<engine>`",
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

// The isolation-test script and its shell tests ship with the catalog skill,
// so the project copy that gabo init writes can run it (plan T5).
func TestReviewerConfigShipsIsolationScript(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	for _, want := range []string{
		"gabo-reviewer-config/scripts/isolation-test.sh",
		"gabo-reviewer-config/scripts/tests/isolation.test.sh",
	} {
		if !slices.Contains(paths, want) {
			t.Errorf("Files() misses %s", want)
		}
	}
}

// The shell tests of the isolation-test script run against fake codex and
// opencode with their own HOME, XDG_CONFIG_HOME and TMPDIR, so they run with
// the Go suite (and in review.checks) wherever bash and git exist.
func TestReviewerConfigIsolationShellTests(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("shell tests skipped in -short mode")
	}
	for _, tool := range []string{"bash", "git", "od"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not found: %v", tool, err)
		}
	}
	out, err := exec.Command("bash", "gabo-reviewer-config/scripts/tests/isolation.test.sh").CombinedOutput()
	if err != nil {
		t.Fatalf("isolation shell tests: %v\n%s", err, out)
	}
}

// Intent 0013 S11: the owner may answer session; then the CLI check and the
// isolation test run once for each codex or opencode engine listed in
// runtimes.assistant, and any failure leaves runtimes.reviewer unchanged.
func TestReviewerConfigAcceptsSession(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), "gabo-reviewer-config/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	for _, s := range []string{
		"the four engines of the table or `session` as the only answers",
		"`session` reviews each wave on the engine of the main session that starts it, one of the engines listed in `runtimes.assistant`",
		"for each engine listed in `runtimes.assistant`",
		"once for each listed `codex` or `opencode` engine",
		"If any of them fails, leave `runtimes.reviewer` unchanged",
		"Set `runtimes.reviewer` to the engine, or to `session`,",
		"the isolation test result per listed engine",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-reviewer-config/SKILL.md: missing %q", s)
		}
	}
}
