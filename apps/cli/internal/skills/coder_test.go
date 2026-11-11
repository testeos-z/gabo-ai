package skills

import (
	"strings"
	"testing"
)

// Intent 0013 S9, D6 (plan T10): the coder works in the per-wave worktree the
// cycle gives it and its Task Close touches only entries of its own wave's
// ledger; a resource of another wave is neither removed nor reported as a
// leftover of this wave.
func TestCoderClosesOnlyItsOwnWaveResources(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"intent 0013 S9", "D6",
		"`<repo-parent>/<repo-name>-worktrees/<wave>/<task>`",
		"only entries of your own wave's ledger (`gabo/ledger/<wave>`)",
		"A resource of another wave",
		"is neither removed nor reported as a leftover of this wave",
		// the scope keeps a kept worktree and a refused branch as leftovers.
		"`leftovers:` lists only resources of this task (its worktree and branch)",
	})
	if strings.Contains(doc, "lists only paths of your own worktree") {
		t.Error("gabo-coder/SKILL.md: `leftovers:` must keep a refused or kept branch (Task Close steps 3 and 4)")
	}
	_, closing, ok := strings.Cut(doc, "## Task Close")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Task Close section")
	}
	closing, _, _ = strings.Cut(closing, "## Output Contract")
	for _, s := range []string{"own wave's ledger", "another wave"} {
		if !strings.Contains(closing, s) {
			t.Errorf("gabo-coder/SKILL.md: the Task Close does not say %q", s)
		}
	}
}

// Intent 0015 S3, S4, D5 (plan T5): the coder runs the reviewer script itself
// on every engine; its catalog skill has one review rule and no Pi exception,
// and neither it nor the notes send the coder to a relay agent.
func TestCoderRunsTheReviewerScriptOnEveryEngine(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"After every push, run the user-level reviewer script yourself from the main checkout, never from your worktree",
		"exactly as your agent file says, and act on its `VERDICT` line",
		"Questions go back to the assistant in your outcome",
	})
	mustLackIn(t, "gabo-coder/SKILL.md", doc,
		"Pi exception", "gabo-reviewer` subagent", "launched by `gabo-cycle`", "back to the cycle", "intent 0013 S20")
	notes := readCatalog(t, "gabo-reviewer-notes/SKILL.md")
	requireIn(t, "gabo-reviewer-notes/SKILL.md", notes, []string{
		"the coder runs the scripts of the user-level `gbai-reviewer` skill itself",
		"validates `PR #<n> head <40-hex sha>`",
	})
	mustLackIn(t, "gabo-reviewer-notes/SKILL.md", notes,
		"Pi exception", "subagent only validates", "relays their final", "ask the `gabo-reviewer` subagent", "The relay passes")
}
