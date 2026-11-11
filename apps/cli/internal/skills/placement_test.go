package skills

import (
	"slices"
	"strings"
	"testing"
)

// placementOps are the stable reviewer_placement operation headings; the
// engine field changes no heading, so slot coverage in gabo check is unchanged
// (intent 0013 plan T16).
var placementOps = []string{"trigger review", "report placement", "return verdict"}

// The local reviewer_placement adapter accepts the review engine only under
// runtimes.reviewer session, refuses every other shape before running any
// command and passes the engine to the user-level scripts as an argument,
// never as environment (intent 0013 S11, D5, plan T16).
func TestReviewerPlacementLocalEngineField(t *testing.T) {
	t.Parallel()
	const name = "gabo-reviewer-placement-local"
	data, err := content.ReadFile(name + "/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if got := operations(doc); !slices.Equal(got, placementOps) {
		t.Errorf("%s/SKILL.md: operations %q, want %q", name, got, placementOps)
	}
	for _, s := range []string{
		// the two accepted request forms.
		"`PR #<n> head <40-hex sha>`",
		"`PR #<n> head <40-hex sha> engine <engine>`",
		"only when `runtimes.reviewer` is `session`",
		// every refusal, before any command runs.
		"an `engine` field under a fixed `runtimes.reviewer`",
		"a missing `engine` under `session`",
		"extra words",
		"refuse it before running any command",
		// the engine reaches the scripts as an argument, never as environment.
		`bash "${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh" head <n> <sha> engine <engine>`,
		"as an argument, never as environment",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s/SKILL.md: missing %q", name, s)
		}
	}
	for _, s := range []string{"REVIEWER_ENGINE=", "ENGINE=<engine>"} {
		if strings.Contains(doc, s) {
			t.Errorf("%s/SKILL.md: passes the engine as environment (%q)", name, s)
		}
	}
}

// The reviewer_placement contract names the optional engine field and keeps
// its operation headings (intent 0013 S11, plan T16).
func TestReviewerPlacementContractEngineField(t *testing.T) {
	t.Parallel()
	const name = "gabo-contract-reviewer-placement"
	data, err := content.ReadFile(name + "/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)
	if got := operations(doc); !slices.Equal(got, placementOps) {
		t.Errorf("%s/SKILL.md: operations %q, want %q", name, got, placementOps)
	}
	for _, s := range []string{
		"`PR #<n> head <40-hex sha>`",
		"optional `engine` field",
		"`PR #<n> head <40-hex sha> engine <engine>`",
		"only when `runtimes.reviewer` is `session`",
		"as an argument, never as environment",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s/SKILL.md: missing %q", name, s)
		}
	}
}

// Intent 0015 S4 (plan T5): the coder is the caller of the reviewer
// placement; no relay agent runs the scripts.
func TestReviewerPlacementNamesTheCoderAsCaller(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{
		"gabo-reviewer-placement-local":    "The coder runs the scripts itself",
		"gabo-contract-reviewer-placement": "The coder that runs the review never reviews",
	} {
		doc := readCatalog(t, name+"/SKILL.md")
		if !strings.Contains(doc, want) {
			t.Errorf("%s/SKILL.md: missing %q", name, want)
		}
		for _, bad := range []string{"relaying agent", "agent relaying", "reviewer agent", "relay"} {
			if strings.Contains(doc, bad) {
				t.Errorf("%s/SKILL.md: still says %q", name, bad)
			}
		}
	}
}
