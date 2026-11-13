package skills

import "testing"

// Intent 0024 (plan T13, PD15): the saved wave trace carries what the script
// printed: the stamps file content, each coder's `session started` line and
// the ledger lines of the wave start entry save.
func TestCycleWaveTraceCarriesStampsSessionLinesAndStartLedger(t *testing.T) {
	cycle := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle", cycle, []string{
		"The entry saved to `gabo/trace/<run id>` at the report step holds the whole output of `cat` of the stamps file exactly as `wave-record.sh` printed it, never a summary or a rewrite, and the report is not returned before that entry is saved and read back.",
		"Each coder's `session started <agent> <task id> <UTC time>` line is saved to the trace in the same step as its `stamp session-started`, so a trace read back shows one such line per coder.",
		"The wave start entry save is followed by `sh .gabo/skills/gabo-wave-report/scripts/wave-record.sh --wave <run id> saves add <topic key> <observation id>` and, after its read-back matched, `sh .gabo/skills/gabo-wave-report/scripts/wave-record.sh --wave <run id> saves read <topic key>`, the same as the report entry.",
	})
}

func TestAssistantWaveStartNamesTheStartEntryLedgerLines(t *testing.T) {
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant", doc, []string{
		"followed by `saves add '<topic key>#<entry>' <observation id>` and, after its read-back showed that entry, `saves read '<topic key>#<entry>'` of `wave-record.sh`",
	})
}
