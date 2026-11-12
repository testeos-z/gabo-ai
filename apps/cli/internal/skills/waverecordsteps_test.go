package skills

import (
	"strings"
	"testing"
)

const waveRecord = "`.gabo/skills/gabo-wave-report/scripts/wave-record.sh"

// failureRule is the sentence every skill carries for a missing or failing
// wave-record.sh (intent 0024 S1).
const failureRule = "report `unverified: wave-record <subcommand> failed (<reason>)`, continue the wave and never write the line the script would print"

// section returns doc from the first occurrence of from up to the first later
// occurrence of to (the end of doc when to is empty).
func section(t *testing.T, doc, from, to string) string {
	t.Helper()
	i := strings.Index(doc, from)
	if i < 0 {
		t.Fatalf("missing %q", from)
	}
	rest := doc[i:]
	if to == "" {
		return rest
	}
	j := strings.Index(rest[len(from):], to)
	if j < 0 {
		t.Fatalf("missing %q after %q", to, from)
	}
	return rest[:len(from)+j]
}

// Intent 0024 plan T3 (S1, S2, S4): the cycle runs the stamps and the saves
// ledger of wave-record.sh and pastes the output unchanged.
func TestCycleRunsWaveRecordStamps(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	steps := section(t, doc, "\n2. Intent claim", "\n3. Resume")
	requireIn(t, "gabo-cycle step 2", steps, []string{waveRecord + " stamp wave-started`", "paste its output unchanged"})
	launch := section(t, doc, "\n5. Per task:", "\n6. On each outcome")
	requireIn(t, "gabo-cycle step 5", launch, []string{
		waveRecord + " stamp session-started <agent> <task id>` before the launch call",
		"a coder resumed by its handle runs none",
	})
	outcome := section(t, doc, "\n6. On each outcome", "\n7. The wave closes")
	requireIn(t, "gabo-cycle step 6", outcome, []string{
		waveRecord + " stamp pr-opened <pr>`", waveRecord + " stamp pr-merged <pr>`", "paste its output unchanged",
	})
	report := section(t, doc, "\n8. Report:", "\n## Output Contract")
	requireIn(t, "gabo-cycle step 8", report, []string{waveRecord + " stamp wave-reported`"})
	trace := section(t, doc, "- Trace (D17)", "\n\n")
	requireIn(t, "gabo-cycle Trace", trace, []string{
		"saves the content of the stamps file as the wave trace",
		waveRecord + " saves add <topic key> <observation id>` after each `mem_save`",
		waveRecord + " saves read <topic key>` after each read-back that matched",
	})
	requireIn(t, "gabo-cycle", doc, []string{failureRule})
}

func TestWaveReportMeasuresAndSavesComeFromTheScript(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	measures := section(t, doc, "9. **Measures**", "\n10. ")
	requireIn(t, "gabo-wave-report Measures", measures, []string{
		"the output of " + waveRecord + " measures` pasted unchanged",
		"never a figure with `~`",
	})
	unverified := section(t, doc, "5. **Unverified writes**", "\n6. ")
	requireIn(t, "gabo-wave-report Unverified writes", unverified, []string{
		"the output of " + waveRecord + " saves unverified`",
		"the final answer is not written before it ran",
	})
	requireIn(t, "gabo-wave-report", doc, []string{failureRule})
}

func TestAssistantWaveCloseRunsSavesUnverified(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	step1 := section(t, doc, "\n1. Before anything else, read", "\n2. **In the first wave report**")
	requireIn(t, "gabo-assistant Wave Close step 1", step1, []string{
		"run " + waveRecord + " saves unverified`",
		"before the final answer",
		failureRule,
	})
}
