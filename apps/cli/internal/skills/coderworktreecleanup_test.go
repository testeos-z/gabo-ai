package skills

import (
	"strings"
	"testing"
)

// Intent 0033 S1 (plan T1): the Task Close removes only a merged worktree and
// keeps it on any real unsaved work; untracked or ignored content of unproven
// provenance counts as unsaved work regardless of author, and a filename or
// glob match alone never proves disposability.
func TestCoderMergedOnlyRemovalGateConservativeProvenance(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"### Cleanup gate (intent 0033 S1",
		"step 3 removes only a worktree whose PR merged into the integration branch or the intent branch (verified in step 1)",
		"an unmerged or open-PR worktree is never removed",
		"Real unsaved work keeps the worktree regardless of author: unpushed commits, staged changes, tracked modifications, or any untracked or ignored content whose provenance or disposability is not proven",
		"A filename or glob match alone never proves disposability",
	})
	_, closing, ok := strings.Cut(doc, "## Task Close")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Task Close section")
	}
	closing, _, _ = strings.Cut(closing, "## Output Contract")
	for _, s := range []string{"Cleanup gate (intent 0033", "real unsaved work", "never proves disposability"} {
		if !strings.Contains(closing, s) {
			t.Errorf("gabo-coder/SKILL.md: the Task Close does not hold %q", s)
		}
	}
}

// Intent 0033 S2 (plan T2): verifier scratch lives outside the worktree and
// never blocks the gate; ignored `.atl/` blocks removal unless proven
// rebuildable by rebuild provenance, never by the path alone.
func TestCoderIgnoredScratchRebuildProof(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"the verifier integration names its scratch path outside the worktree (a `mktemp -d` dir, never inside the worktree)",
		"Ignored `.atl/` content proven rebuildable",
		"regenerable by re-running the verifier with no user input inside, provenance proven by that rebuild and never by the path alone",
		"does not block removal",
		"Ignored scratch that cannot be proven rebuildable keeps the worktree and the reason names it",
	})
}

// Intent 0033 S3, S4 (plan T3): untracked generated output, `*cwd-placeholder`
// files and wave folders block removal unless their provenance is proven;
// filename-only matches, non-empty folders and folders of another wave keep
// the worktree with the item named.
func TestCoderGeneratedOutputPlaceholderRules(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"Generated output, placeholders and empty folders (S3, S4)",
		"reproducible by its build with no hand input inside, proven and never assumed from the path",
		"A `*cwd-placeholder` file blocks removal unless its flow ownership and orphaned provenance are both proven, never by the filename alone",
		"A wave folder blocks removal unless it is actually empty and owned by this same wave",
		"a non-empty folder or a folder not owned by this wave keeps the worktree",
		"Every kept item is named in the reason; doubt keeps the worktree",
	})
}

// Intent 0033 S5 (plan T4, coder side): a gone ledger path with no flow record
// and no proof of who removed it is reported as missing with unknown
// provenance, never as a flow cleanup and never naming an actor.
func TestCoderMissingPathRecordRule(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", doc, []string{
		"Missing path record (S5)",
		"missing with unknown provenance",
		"kept as history, reported separately from flow cleanups and from proven external removals, and a missing path alone never names an actor",
		"A removal proven external records the evidence cited; a flow-owned removal records the owner and the reason",
	})
}
