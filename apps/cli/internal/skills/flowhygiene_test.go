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
