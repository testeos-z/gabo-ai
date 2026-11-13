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
		"only entries of your own wave's ledger (`gabo/ledger/<run id>`)",
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

// decisionGateRow returns the row of the `## Decision Gates` table that starts
// with the given verdict cell.
func decisionGateRow(t *testing.T, doc, cell string) string {
	t.Helper()
	_, gates, ok := strings.Cut(doc, "## Decision Gates")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Decision Gates section")
	}
	gates, _, _ = strings.Cut(gates, "\n## ")
	for _, line := range strings.Split(gates, "\n") {
		if strings.HasPrefix(line, "| "+cell+" |") {
			return line
		}
	}
	t.Fatalf("gabo-coder/SKILL.md: no Decision Gates row for %s", cell)
	return ""
}

// Intent 0015 S3, S5, PD10 (plan T15): a clean verdict whose merge was skipped
// because the PR is CONFLICTING is one fix round: merge origin/dev, push, ask
// for a new review.
func TestCoderMergesDevOnACleanVerdictWithAConflictingPR(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	row := decisionGateRow(t, doc, "`verdict=clean merge=skipped`, PR `CONFLICTING`")
	for _, s := range []string{
		"read PR status", "`mergeable`", "`CONFLICTING`",
		"merge `origin/dev` into the task branch in the worktree",
		"never a rebase, never a force-push",
		"resolve the conflict", "run the task's checks", "push",
		"request a new review for the new head",
		"one fix round",
		"cannot resolve", "BLOCKED with the question, never HANDED",
	} {
		if !strings.Contains(row, s) {
			t.Errorf("gabo-coder/SKILL.md: the merge=skipped row lacks %q", s)
		}
	}
	if !strings.Contains(doc, "Max 4 fix rounds; then label the issue `needs-user`, report BLOCKED with the question.") {
		t.Error("gabo-coder/SKILL.md: the Max 4 fix rounds line changed")
	}
	// the new row precedes the other-merge row.
	if strings.Index(doc, "`verdict=clean merge=skipped`, PR `CONFLICTING`") > strings.Index(doc, "| `verdict=clean` other merge |") {
		t.Error("gabo-coder/SKILL.md: the merge=skipped row must come before the other-merge row")
	}
}

// Intent 0015 S3, S5, PD10 (plan T15): the other rows keep their text.
func TestCoderDecisionGateRowsKeepTheirText(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	for cell, want := range map[string]string{
		"`verdict=clean merge=merged`": "Run the Task Close now.",
		"`verdict=clean` other merge":  "Report \"needs human merge\"; HANDED. You are resumed with `PR #<n> merged` once it merges.",
		"`verdict=blocking`":           "Fix round: read the latest marker comment, fix, push, request review for the new head.",
		"`verdict=error`":              "Request once more; a second error ⇒ BLOCKED.",
		"`verdict=skipped`":            "The head moved; request review for the current head.",
	} {
		if got := decisionGateRow(t, doc, cell); got != "| "+cell+" | "+want+" |" {
			t.Errorf("gabo-coder/SKILL.md: row %s is %q", cell, got)
		}
	}
}

// Intent 0015 S5, PD11 (plan T16): a new coder launched after a session
// restart can run the Task Close when its worktree or branch is already gone.
func TestCoderTaskCloseWithoutAWorktree(t *testing.T) {
	t.Parallel()
	want := "If your worktree or branch is already gone (a new coder launched after a session restart), the caller gives you the main checkout path in your input: run steps 1 and 2 from it without `git -C <wt>`, skip steps 3 and 4 for what is gone, and list nothing as a leftover for it. With the PR still open and only the worktree gone, recreate it there with `git worktree add <worktree path> <branch>` and continue; with the branch gone too, return the question."
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	_, closing, _ := strings.Cut(doc, "## Task Close")
	closing, _, _ = strings.Cut(closing, "## Output Contract")
	requireIn(t, "gabo-coder Task Close", closing, []string{want})
	tpl := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	requireIn(t, "gabo-coder template", tpl, []string{"already gone", "from it without `git -C <wt>`"})
}
