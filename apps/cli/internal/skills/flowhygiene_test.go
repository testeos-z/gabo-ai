package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// closingKeyword matches a GitHub closing keyword that links an issue, such as
// `Closes #592` or `fixes #592`. A closing keyword on the default branch closes
// the task before the Task Close, so the coder PR body never holds one.
var closingKeyword = regexp.MustCompile(`(?i)\b(close[sd]?|fix(e[sd])?|resolve[sd]?)\s+#`)

// Intent 0023 S8, D7 (plan T8, finding #565 (b)): the coder's PR body links
// its task as `Refs #<n>` on every base and never with a closing keyword,
// because the Task Close is the only writer of DONE (TRD D23). The catalog
// source and its Codex mirror (PD10) carry the same text.
func TestCoderPRBodyLinksTaskWithRefsNeverClosing(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-coder/SKILL.md")
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-coder/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != catalog {
		t.Error("plugins/codex/skills/gabo-coder/SKILL.md differs from the catalog source (S8, PD10)")
	}
	requireIn(t, "gabo-coder/SKILL.md", catalog, []string{
		"links the task as `Refs #<n>` on every base, never with a closing keyword",
		"a closing keyword that merges into the default branch closes the task before the Task Close removes its worktree and labels",
		"the Task Close is the only writer of DONE (TRD D23)",
		"the PR that carries `Refs #<n>`",
	})
	if closingKeyword.MatchString(catalog) {
		t.Errorf("gabo-coder/SKILL.md: a closing keyword still links the task (S8): %q",
			closingKeyword.FindString(catalog))
	}
	if strings.Contains(catalog, "`Closes #<n>`") {
		t.Error("gabo-coder/SKILL.md: `Closes #<n>` still names the task link (S8)")
	}
}

// Intent 0023 S7 (plan T9, finding #565 (a)): closing a task removes every
// status label it carries, in the contract example and its Codex copy (PD10:
// the `.gabo/` catalog copy is never edited). The claim and progress commands
// of an open task touch no label they do not add.
func TestContractTasksCloseRemovesEveryStatusLabel(t *testing.T) {
	t.Parallel()
	example := readCatalog(t, "gabo-contract-tasks/examples/tasks-gh.md")
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-contract-tasks/examples/tasks-gh.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != example {
		t.Error("plugins/codex/skills/gabo-contract-tasks/examples/tasks-gh.md differs from the catalog source (S7, PD10)")
	}
	done := lineWith(t, example, "- DONE, only after the PR merged:")
	requireIn(t, "tasks-gh.md DONE line", done, []string{
		"--remove-label status:in-progress,status:blocked,needs-user",
		"gh issue close <n> -R $R --reason completed",
	})
	requireIn(t, "tasks-gh.md", example, []string{
		"gh issue edit <n> -R $R --add-assignee @me --add-label status:in-progress",
		"gh issue edit <n> -R $R --add-label status:blocked --remove-label status:in-progress",
		"gh issue comment <n> -R $R --body \"<!-- gabo-progress --> <one line>\"",
	})
	for _, heading := range []string{"### claim", "### log progress", "### read one"} {
		sec := operationSection(example, heading)
		if sec == "" {
			t.Errorf("tasks-gh.md: no %s section", heading)
		}
		if strings.Contains(sec, "--remove-label") {
			t.Errorf("tasks-gh.md: %s removes a label of an open task (S7): %q", heading, sec)
		}
	}
}

// Intent 0023 S7 (plan T9): the coder's Task Close step 2 closes the task,
// re-reads it, and reports a status label that survives under `unverified:`
// (retry once, as today). Source and Codex copy are the same text (PD10).
func TestCoderTaskCloseRemovesStatusLabelsAndReportsSurvivors(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-coder/SKILL.md")
	_, closing, ok := strings.Cut(catalog, "## Task Close")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Task Close section")
	}
	closing, _, _ = strings.Cut(closing, "## Output Contract")
	step := lineWith(t, closing, "2. Close the task issue")
	requireIn(t, "gabo-coder Task Close step 2", step, []string{
		"removes every status label it carries (`status:in-progress`, `status:blocked`, `needs-user`)",
		"re-read it: it must show state closed and no status label",
		"Retry once, then report it under `unverified:`, naming each status label that survives",
	})
}

// Intent 0023 S9 (plan T10, finding #565 (c), D8): when the engine asks about
// or refuses deleting the merged remote branch, the coder's Task Close does not
// retry it by another route, lists it under `leftovers:` with the exact command
// for the owner, and the task still closes. Source and Codex copy are the same
// text (PD10).
func TestCoderTaskCloseLeavesARefusedRemoteBranchWithItsCommand(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-coder/SKILL.md")
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-coder/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != catalog {
		t.Error("plugins/codex/skills/gabo-coder/SKILL.md differs from the catalog source (S9, PD10)")
	}
	_, closing, ok := strings.Cut(catalog, "## Task Close")
	if !ok {
		t.Fatal("gabo-coder/SKILL.md: no ## Task Close section")
	}
	closing, _, _ = strings.Cut(closing, "## Output Contract")
	step := lineWith(t, closing, "4. Delete the branch")
	requireIn(t, "gabo-coder Task Close step 4", step, []string{
		"the engine asks about deleting the merged remote branch or refuses it",
		"do not retry it by another route (no API call, no force push, no other delete)",
		"list it under `leftovers:` as `merged remote branch left (<reason>)`",
		"with the exact command `git push origin --delete <branch>` for the owner",
		"the task still closes (step 2 unchanged)",
		"A delete that succeeds leaves no leftover.",
	})
	leftovers := lineWith(t, catalog, "leftovers: <none")
	requireIn(t, "gabo-coder leftovers line", leftovers, []string{
		"merged remote branch left (<reason>) with its command",
	})
}

// Intent 0023 PD6 (plan T10): gabo-wave-report/SKILL.md is not edited. Its
// Leftovers item still names the coder task-close leftovers, which carry a
// refused remote branch to the wave report (S9). This is a guard: it passes on
// the unchanged text.
func TestWaveReportLeftoversStillNamesCoderTaskCloseLeftovers(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	leftovers := lineWith(t, doc, "7. **Leftovers**")
	requireIn(t, "gabo-wave-report Leftovers item", leftovers, []string{
		"leftovers from coder task closes",
	})
}

// operationSection returns the text of one `### ` section, up to the next
// heading of any level.
func operationSection(doc, heading string) string {
	_, rest, ok := strings.Cut(doc, heading+"\n")
	if !ok {
		return ""
	}
	sec, _, _ := strings.Cut(rest, "\n#")
	return sec
}
