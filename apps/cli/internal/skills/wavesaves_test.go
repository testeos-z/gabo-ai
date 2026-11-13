package skills

import (
	"strings"
	"testing"
)

// Intent 0025 S2 (plan T7, PD6): every memory write of a wave is a ledger entry
// `<topic key>#<entry>` in the saves file, so `saves unverified` covers all of
// them; a read counts only for the entry whose text the read-back shows.
const (
	savesAdd  = waveRecord + " saves add '<topic key>#<entry>' <observation id>`"
	savesRead = waveRecord + " saves read '<topic key>#<entry>'`"
)

func TestCycleTraceRecordsEveryMemoryWriteByEntry(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	trace := section(t, doc, "- Trace (D17)", "\n\n")
	requireIn(t, "gabo-cycle Trace", trace, []string{
		savesAdd + " after every memory write of the wave",
		savesRead + " after each read-back whose observation shows that entry's text",
		"the label of the line written",
		"`start` for the wave start entry",
		"the task id for a task's line",
		"`stamps` for the stamps entry",
		"`report` for the report entry of `gabo/trace/<run id>`",
		"`ledger` for each ledger entry of `gabo/ledger/<run id>`",
		"two entries of one observation are two ledger entries",
		"the wave start entry calls of step 2 included",
	})
	for _, old := range []string{"saves add <topic key> <observation id>` after each `mem_save`", "saves read <topic key>` after each read-back that matched"} {
		if strings.Contains(trace, old) {
			t.Errorf("gabo-cycle Trace still has %q", old)
		}
	}
}

func TestAssistantRecordsItsMemoryWritesByEntry(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant", doc, []string{
		"followed by `saves add '<topic key>#<entry>' <observation id>` and, after its read-back showed that entry, `saves read '<topic key>#<entry>'` of `wave-record.sh` (entry `start`; the entry labels are the ones of `gabo-cycle` Trace)",
		"each save of the report entry again (the comfort answer) is followed by the same `saves add` and `saves read` calls with the entry `report`",
	})
}

// The report's Unverified writes line stays the script's output, unchanged.
func TestUnverifiedWritesLineStaysTheScriptOutput(t *testing.T) {
	t.Parallel()
	report := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report", report, []string{waveRecord + " saves unverified` pasted unchanged"})
	requireIn(t, "gabo-assistant", readCatalog(t, "gabo-assistant/SKILL.md"), []string{waveRecord + " saves unverified` before the final answer, pasting its output unchanged"})
	if strings.Contains(report, "saves add") {
		t.Error("gabo-wave-report restates saves add; only the cycle and the assistant record writes")
	}
}

// The memory example keys the ledger and the trace operations by the run id and
// finds the newest run of an intent by prefix (S1); no operation keeps `<wave>`.
func TestMemoryExampleOperationsUseTheRunID(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-contract-memory/examples/memory-engram.md")
	_, ops, ok := strings.Cut(doc, "## Operations")
	if !ok {
		t.Fatal("memory-engram.md: no Operations section")
	}
	for _, bad := range []string{"gabo/ledger/<wave>", "gabo/trace/<wave>"} {
		if strings.Contains(ops, bad) {
			t.Errorf("memory-engram.md Operations still keys by %s", bad)
		}
	}
	requireIn(t, "memory-engram.md", doc, []string{
		"`mem_search` for `gabo/ledger/<run id>`",
		"`topic_key: gabo/ledger/<run id>`",
		"`<wave>` is the wave's run id `<I####>-<YYYYMMDDTHHMMSSZ>`",
	})
}
