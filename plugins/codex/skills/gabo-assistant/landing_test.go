package gaboassistant

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

// Intent 0032 landing branches (assistant side): folder = branch name,
// folder before branch, one wave per branch, pour into dev.
func TestLandingBranches(t *testing.T) {
	doc := readSkill(t)
	for _, s := range []string{
		// S4/S8: branch name is the intent folder name; folder first.
		"The landing branch is the intent folder name under `docs/product/intents/<NNNN-slug>/`",
		"The folder exists on `repo.integration_branch` before `gabo-cycle` cuts the branch",
		// S1/S2: task PRs target the intent branch; the intent pours into dev.
		"Task PRs target the intent branch",
		"pours into `dev` through one intent PR",
		// S10: one wave per branch, parallel intents.
		"One wave per intent branch",
		"each with its own branch and wave",
		// Handoff carries the branch.
		"pass the intent folder and branch to `gabo-cycle`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-assistant/SKILL.md: missing %q", s)
		}
	}
}

// Fixture lists intent folders; each must read as its own branch name.
func TestLandingBranchFixtureNames(t *testing.T) {
	b, err := os.ReadFile("testdata/landing-branches.txt")
	if err != nil {
		t.Fatal(err)
	}
	doc := readSkill(t)
	if !strings.Contains(doc, "`docs/product/intents/<NNNN-slug>/`") {
		t.Error("gabo-assistant/SKILL.md: missing intent folder path rule")
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, " ") {
			t.Errorf("fixture: assistant folder line must be a bare folder name: %q", line)
		}
	}
}
