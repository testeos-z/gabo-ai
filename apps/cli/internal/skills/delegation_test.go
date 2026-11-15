package skills

import (
	"strings"
	"testing"
)

// Intent 0021 (plan T1): the assistant skill states the default (do the work
// yourself), the four reasons to hand work off, role ownership and the
// handoff's first line.
func TestAssistantHandsOffOnlyForANamedReason(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	_, hard, _ := strings.Cut(doc, "## Hard Rules")
	hard, _, _ = strings.Cut(hard, "\n## ")
	requireIn(t, "gabo-assistant Hard Rules", hard, []string{
		// S1: the default and the four reasons.
		"do the work yourself",
		"hand work off only when one of four reasons holds",
		"independent units of behavior with disjoint edit surfaces, each needing several minutes, launched together",
		"a map that needs more than one bounded read batch (about 3 reads or searches and about 10k tokens of output)",
		"a long suite, build or wait that would block the main thread",
		"an independent verifier, reviewer or plan reviewer, because a fresh context is the point",
		// S3: role ownership.
		"role ownership",
		"`gabo-pm` for the PRD, intents and specs",
		"`gabo-dev-lead` for the TRD, plans and task issues",
		"`gabo-db-master` and `gabo-frontend` for their documents",
		"a fresh `gabo-plan-reviewer` when the owner is absent",
		"not delegation by default",
		// S2: the handoff's first line and the labels.
		"`Reason: <label>`",
		"`(a) parallel units`",
		"`(b) map beyond one read batch`",
		"`(c) long run off the main thread`",
		"`(d) fresh context`",
		"`role ownership`",
		"Name that label when you tell the owner what you handed off",
		"when the owner asks why",
		"no handoff is announced",
		// PD7: resume and replacement continue the same unit.
		"repeats the reason of its first launch and is not a new handoff",
		// kept semantics.
		"never block on long work",
	})
	for _, old := range []string{
		"delegates all heavy work",
		"hand writing, exploring and long commands",
		"delegate exploring, writing documents, writing code and long commands",
	} {
		if strings.Contains(doc, old) {
			t.Errorf("gabo-assistant/SKILL.md still holds %q", old)
		}
	}
}

// The Decision Gates keep their rows and gain an intro naming the reasons.
func TestAssistantDecisionGatesIntroNamesTheReasons(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	_, gates, _ := strings.Cut(doc, "## Decision Gates")
	gates, _, _ = strings.Cut(gates, "\n## ")
	intro, table, ok := strings.Cut(gates, "| Need |")
	if !ok {
		t.Fatal("Decision Gates has no table")
	}
	requireIn(t, "Decision Gates intro", intro, []string{"role ownership", "named reason"})
	requireIn(t, "Decision Gates table", table, []string{
		"`gabo-pm`", "`gabo-dev-lead`", "`gabo-db-master`", "`gabo-frontend`",
		"no delegate: load the `gabo-cycle` skill yourself",
		"`gabo-plan-reviewer`",
	})
}
