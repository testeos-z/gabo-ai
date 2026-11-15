package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Issue #619 (I0023 T18, owner decision on #575: PRs merge into dev with a
// merge commit, `gh pr merge --merge`, never a squash). The repository-host
// contract example and its Codex mirror carry the merge command, and no
// embedded skill instructs a squash merge.
func TestHostMergesWithMergeCommitNeverSquash(t *testing.T) {
	t.Parallel()
	const merge = "gh pr merge <pr> -R $R --merge --match-head-commit <sha>"

	example := readCatalog(t, "gabo-contract-repository-host/examples/host-github.md")
	mirror, err := os.ReadFile(filepath.Join(repoRoot, "plugins/codex/skills/gabo-contract-repository-host/examples/host-github.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mirror) != example {
		t.Error("plugins/codex/skills/gabo-contract-repository-host/examples/host-github.md differs from the catalog source (PD10)")
	}
	requireIn(t, "host-github.md", example, []string{merge})
	if strings.Contains(example, "--squash") {
		t.Error("host-github.md: the contract example still merges with --squash (#619)")
	}
	if strings.Contains(strings.ToLower(example), "squash") {
		t.Error("host-github.md: squash wording remains in the contract example (#619)")
	}

	claude, err := os.ReadFile("../../../../.claude/skills/gabo-repository-host-github/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	requireIn(t, ".claude gabo-repository-host-github/SKILL.md", string(claude), []string{merge})
	if strings.Contains(strings.ToLower(string(claude)), "squash") {
		t.Error(".claude gabo-repository-host-github/SKILL.md: squash wording remains (#619)")
	}

	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(string(f.Data), "--squash") {
			t.Errorf("%s: instructs a squash merge (#619)", f.Path)
		}
	}
}
