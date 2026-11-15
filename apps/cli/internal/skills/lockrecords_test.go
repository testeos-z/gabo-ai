package skills

import (
	"strings"
	"testing"
)

// Intent 0028 S14, S16 (plan T14): the cycle gives each coder the outcome
// file, the task and the engine that its hook records carry, and makes one
// scratch directory per wave outside the project tree.
func TestCycleGivesEachCoderTheLockRecordEnvironment(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle lock record environment", doc, []string{
		"`mktemp -d`",
		"outside the project tree",
		"`outcomes.jsonl`",
		"`outcome file`",
		"`gabo/ledger/<run id>`",
		"`GABO_OUTCOME_FILE`",
		"`GABO_TASK`",
		"`GABO_ENGINE`",
	})
}

// Intent 0028 S14 (plan T14): a lock on an engine whose probe item (a) is no
// (or not testable, never observed) shows `not run: <reason>` in the trace,
// and no task is stopped for it.
func TestCycleWritesNotRunFromTheProbeRecord(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle not run lines", doc, []string{
		"`probe.md` in `docs/product/intents/<NNNN>-<slug>/`",
		"item (a) `no` or `not testable`",
		"outcome not-run <lock> <task id> <session engine> <reason>",
		"`probe item (a) is <no|not testable> on <engine>`",
		"`yes`",
		"stops no task",
	})
}

// Intent 0028 S16 (plan T14): the report step copies the lock records into
// the trace as one entry before the report entry, and a missing record reads
// `none`.
func TestCycleCopiesTheLockRecordsAtTheReport(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle lock record entry", doc, []string{
		"`outcomes` for the lock records entry",
		"wave-record.sh --wave <run id> outcomes <outcome file>",
		"`none` when it prints nothing",
		"before the report entry",
		"the outcome directory is removed",
	})
}

// Intent 0028 S16 (plan T14): the wave report's Tasks section shows the lock
// records of the trace; `asked` reads `not reported` when no answer arrived
// (PD11), and the section says that no task was stopped by a `not run` line.
func TestWaveReportShowsTheLockRecords(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	_, rest, ok := strings.Cut(doc, "2. **Tasks**")
	if !ok {
		t.Fatal("gabo-wave-report/SKILL.md: no section 2 (Tasks)")
	}
	tasks, _, _ := strings.Cut(rest, "\n3. **Questions**")
	requireIn(t, "gabo-wave-report lock records", tasks, []string{
		"lock records of the trace",
		"`blocked`",
		"`warned`",
		"`asked`",
		"`not run`",
		"`answer: not reported`",
		"`none` when the trace holds none",
		"`not run: <reason>`",
		"no task was stopped",
	})
}
