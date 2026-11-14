package skills

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Intent 0029 T2 (spec S1, S2, S4, S5; decisions D1, D2, D3, D6): the
// repository_host contract and both GitHub adapters carry the `protect
// branches` operation, with the rule list, the read-back, the three results
// and the idempotent rerun. The live host run is owner work (T5), not pinned here.
const protectOperation = "protect branches"

// protectRuleList is what every copy of the operation must state (S2, D3, D6).
var protectRuleList = []string{
	"a pull request before a merge", "no force-push", "no deletion",
	"`reviewer/tests`", "`reviewer/review`", "`repo.integration_branch` only",
	"administrator role", "`main` requires no status",
	"repo.protection.required_approvals.integration", "repo.protection.required_approvals.main",
	"zero when the key is absent",
}

func TestProtectBranchesContractStatesTheRuleList(t *testing.T) {
	t.Parallel()
	const name = "gabo-contract-repository-host/SKILL.md"
	doc := readCatalog(t, name)
	if !slices.Contains(operations(doc), protectOperation) {
		t.Fatalf("%s: no `### %s` heading under `## Operations` (S1)", name, protectOperation)
	}
	requireIn(t, name, doc, protectRuleList)
	requireIn(t, name, doc, []string{
		"`set`", "`missing`", "`not offered`", "host's reason", "read back",
		"the other rules are still set", "no implementation yet", "states that the operation is not offered",
	})
}

func TestProtectBranchesGitHubAdaptersCoverTheOperation(t *testing.T) {
	t.Parallel()
	example := readCatalog(t, "gabo-contract-repository-host/examples/host-github.md")
	claude, err := os.ReadFile(filepath.Join(repoRoot, ".claude/skills/gabo-repository-host-github/SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	adapters := map[string]string{"host-github.md": example, ".claude gabo-repository-host-github/SKILL.md": string(claude)}
	for name, doc := range adapters {
		if !slices.Contains(operations(doc), protectOperation) {
			t.Errorf("%s: no `### %s` heading under `## Operations` (S4)", name, protectOperation)
		}
		// The rule list is stated in each adapter too (S2, D3, D6).
		requireIn(t, name, doc, protectRuleList)
		// D2: rulesets are the default mechanism; the host acceptance is checked by T1, not by this text.
		requireIn(t, name, doc, []string{
			"repository rulesets", "gabo-protect-", "/rulesets", "\"type\": \"pull_request\"",
			"\"type\": \"non_fast_forward\"", "\"type\": \"deletion\"", "\"type\": \"required_status_checks\"",
			"\"actor_id\": 5, \"actor_type\": \"RepositoryRole\", \"bypass_mode\": \"always\"",
			"`not offered`", "host's reason",
		})
		// S4: the rerun finds its ruleset by name, updates it, and creates no second one.
		requireIn(t, name, doc, []string{
			"found by name", "no second ruleset", "read-back is identical",
		})
	}
}

// The contract and its two copies stay byte-identical with the project source
// (`.gabo/skills/`), so the operation text has one author.
func TestProtectBranchesCopiesAreByteIdentical(t *testing.T) {
	t.Parallel()
	copies := map[string][]string{
		"gabo-contract-repository-host/SKILL.md": {
			".gabo/skills/gabo-contract-repository-host/SKILL.md",
			"plugins/codex/skills/gabo-contract-repository-host/SKILL.md",
		},
		"gabo-contract-repository-host/examples/host-github.md": {
			".gabo/skills/gabo-contract-repository-host/examples/host-github.md",
			"plugins/codex/skills/gabo-contract-repository-host/examples/host-github.md",
		},
	}
	for catalogPath, projectPaths := range copies {
		embedded := readCatalog(t, catalogPath)
		for _, p := range projectPaths {
			b, err := os.ReadFile(filepath.Join(repoRoot, p))
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != embedded {
				t.Errorf("%s differs from the catalog copy %s (intent 0029 T2)", p, catalogPath)
			}
		}
	}
	for _, p := range []string{
		"gabo-contract-repository-host/SKILL.md",
		"gabo-contract-repository-host/examples/host-github.md",
	} {
		if !strings.Contains(readCatalog(t, p), protectOperation) {
			t.Errorf("%s does not carry the %q operation", p, protectOperation)
		}
	}
}
