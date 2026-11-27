package gabocycle

import (
	"os"
	"strings"
	"testing"
)

func readSkill(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func requireIn(t *testing.T, doc string, want []string) {
	t.Helper()
	for _, s := range want {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-cycle/SKILL.md: missing %q", s)
		}
	}
}

// Intent 0032 landing branches S1-S10: the cycle cuts <NNNN-slug> from dev,
// task PRs target the intent branch, the intent PR targets dev, and S9 always
// runs the <NNNN-slug>-sync reverse merge even with no divergence.
func TestLandingBranches(t *testing.T) {
	doc := readSkill(t)
	requireIn(t, doc, []string{
		// S1: two landing-branch kinds.
		"two landing-branch kinds",
		"Task PRs target the intent branch, never `dev` directly",
		// S2+S7: finished intents pour into dev through one intent PR.
		"exactly one intent PR targeting `dev`",
		"coders never open the intent PR",
		// S3: intent branch works as the working branch works today.
		"works as the working branch works today",
		// S4: name, cut point, ruleset, no key, main/master refused.
		"named exactly like the intent folder",
		"Cut the intent branch from `dev` when the wave opens",
		"No new `gabo.yaml` key",
		"`main` and `master` stay refused as landing targets",
		"same ruleset as `dev`",
		// S5: same gate, human-only already approved, no direct change.
		"same gate",
		"human-only paths already approved",
		"never a direct change to `repo.integration_branch`",
		// S6: folder source, handoff, contract ops never host CLIs.
		"pass the folder in each coder handoff",
		"name contract operations never host CLIs",
		// S7: pour via repository_host contract ops.
		"Pour through `repository_host` contract operations only",
		"delete the intent branch under the merged-only rule",
		"a blocking verdict runs a fix round on the intent branch",
		// S8: folder before branch.
		"folder exists before the branch is cut",
		// S9: always sync even with no divergence, never rebase/force-push.
		"always run the reverse-merge sync even with no divergence",
		"`<NNNN-slug>-sync` from the intent branch",
		"merge `repo.integration_branch` into it",
		"PR `<NNNN-slug>-sync` into the intent branch through the normal review",
		"Never rebase, never force-push",
		// S10: parallel intents pour one by one after their own sync.
		"each with its own branch and its own wave",
		"pour one by one, each after its own S9 sync",
	})
}

// The sync-branch name derives from the intent folder name (fixture).
func TestLandingSyncBranchNames(t *testing.T) {
	b, err := os.ReadFile("testdata/landing-branches.txt")
	if err != nil {
		t.Fatal(err)
	}
	fixture := string(b)
	doc := readSkill(t)
	for _, line := range strings.Split(fixture, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			t.Fatalf("fixture line must be '<intent-folder> <sync-branch>': %q", line)
		}
		folder, sync := parts[0], parts[1]
		if sync != folder+"-sync" {
			t.Errorf("fixture: sync branch %q must be folder %q plus -sync", sync, folder)
		}
	}
	if !strings.Contains(doc, "`<NNNN-slug>-sync`") {
		t.Error("gabo-cycle/SKILL.md: missing `<NNNN-slug>-sync` naming rule")
	}
}
