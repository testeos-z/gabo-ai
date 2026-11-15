package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// incidentRulesSection returns the assistant's incident and repeated-mistake
// section, from its heading to the next "## " heading. Each rule test reads
// the catalog copy, which the engine and plugin copies must match (see
// TestAssistantIncidentMirrorsAreIdentical).
func incidentRulesSection(t *testing.T) string {
	t.Helper()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	_, section, ok := strings.Cut(doc, "## Incidents and Repeated Mistakes (intent 0030 S13, S15, S17, S18; D9)")
	if !ok {
		t.Fatal("gabo-assistant/SKILL.md: no ## Incidents and Repeated Mistakes section")
	}
	section, _, _ = strings.Cut(section, "\n## ")
	return section
}

// Intent 0030 S13 and D9: an incident is a coder run that reported done and
// was later found wrong; a BLOCKED or refused task is not one. The assistant
// records one trace line, one lessons entry and asks gabo-pm for the draft.
func TestAssistantIncidentDefinitionAndRecord(t *testing.T) {
	t.Parallel()
	section := incidentRulesSection(t)
	requireIn(t, "gabo-assistant incident rule", section, []string{
		"an incident is a coder run that reported done and was later found wrong by the owner, the assistant or the reviewer",
		"A task that was BLOCKED or refused before it reported done is not an incident",
		"one incident line in the wave trace",
		"naming the task issue, its PR and who found it",
		"one entry in `docs/product/LESSONS.md` under `## Incidents`",
		"`eval task: pending fix`",
		"asks `gabo-pm` for a Draft `intent.md`",
		"`Reason: role ownership`",
		"naming the task, its PR and what was found wrong",
		"the coder that made the mistake never writes it",
	})
}

// Intent 0030 S15: one eval task is proposed when the fix of an incident
// merges, citing the lessons entry and the fix PR; the set's maximum is named.
func TestAssistantEvalTaskOnFix(t *testing.T) {
	t.Parallel()
	section := incidentRulesSection(t)
	requireIn(t, "gabo-assistant eval-task rule", section, []string{
		"When the PR that fixes an incident is merged",
		"propose one eval task for that incident",
		"cites the lessons entry and the fix PR",
		"it joins the set only when the owner accepts it",
		"While no fix has shipped, no task is proposed and the entry keeps `eval task: pending fix`",
		"When the set is at its maximum (S2), the proposal says so",
		"until the owner raises the maximum or removes a task",
	})
}

// Intent 0030 S17 and S18 (D8): a mistake seen once is recorded and proposes
// nothing; a second occurrence opens one proposal PR that names both, which
// no agent or reviewer merges; a decline is recorded in the lessons file.
func TestAssistantTwoTimesRule(t *testing.T) {
	t.Parallel()
	section := incidentRulesSection(t)
	requireIn(t, "gabo-assistant two-times rule", section, []string{
		"A mistake seen once goes to `## Mistakes seen once` in the lessons file",
		"No proposal and no instruction-file change for a first occurrence",
		"Two-times rule",
		"open one proposal PR",
		"adds its entry to the `## Repeated mistakes` section of the instruction file",
		"names both occurrences (the PR or task of each)",
		"the reviewer never merges it",
		"`review.human_only_paths` in `gabo.yaml`",
		"the owner merges it or declines it",
		"the instruction file stays unchanged",
		"- proposal: declined (PR #<n>, <UTC date>)",
	})
}

// Intent 0030 S13, S15, S17, S18: the catalog copy and the two engine copies
// of gabo-assistant stay byte-identical, as the existing mirror tests require.
func TestAssistantIncidentMirrorsAreIdentical(t *testing.T) {
	t.Parallel()
	catalog := readCatalog(t, "gabo-assistant/SKILL.md")
	for _, rel := range []string{
		".gabo/skills/gabo-assistant/SKILL.md",
		"plugins/codex/skills/gabo-assistant/SKILL.md",
	} {
		b, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b, []byte(catalog)) {
			t.Errorf("%s differs from the catalog copy apps/cli/internal/skills/gabo-assistant/SKILL.md", rel)
		}
	}
}
