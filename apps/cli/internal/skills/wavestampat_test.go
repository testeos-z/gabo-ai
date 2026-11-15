package skills

import (
	"strings"
	"testing"
)

// Intent 0025 plan T8 (S3): the cycle stamps pr-opened and pr-merged with the
// PR's own times and wave-reported first in the report step.
func TestCycleStampsPREventsWithTheirOwnTimes(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	outcome := section(t, doc, "\n6. On each outcome", "\n7. The wave closes")
	requireIn(t, "gabo-cycle step 6", outcome, []string{
		"`repository_host` read PR status (GitHub: `gh pr view <pr> --json createdAt,mergedAt`)",
		waveRecord + " stamp pr-opened <pr> --at <createdAt>`",
		waveRecord + " stamp pr-merged <pr> --at <mergedAt>`",
	})
	for _, bad := range []string{" stamp pr-opened <pr>`", " stamp pr-merged <pr>`"} {
		if strings.Contains(doc, bad) {
			t.Errorf("gabo-cycle still stamps a PR event without --at: %s", bad)
		}
	}
}

func TestCycleStampsWaveReportedFirstWithoutAt(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	report := section(t, doc, "\n8. Report:", "\n## Output Contract")
	requireIn(t, "gabo-cycle step 8", report, []string{
		"Report: as the first command of this step, run `sh .gabo/skills/gabo-wave-report/scripts/wave-record.sh --wave <run id> stamp wave-reported` (never with `--at`)",
	})
	if strings.Contains(doc, "stamp wave-reported --at") {
		t.Error("gabo-cycle stamps wave-reported with --at")
	}
}

// The report and assistant skills restate no PR stamp call without the PR's time.
func TestReportAndAssistantRestateNoPlainPRStamp(t *testing.T) {
	t.Parallel()
	for _, p := range []string{"gabo-wave-report/SKILL.md", "gabo-assistant/SKILL.md"} {
		doc := readCatalog(t, p)
		for _, bad := range []string{"stamp pr-opened <pr>", "stamp pr-merged <pr>"} {
			if strings.Contains(doc, bad) {
				t.Errorf("%s restates %q without --at", p, bad)
			}
		}
	}
}

// The ledger and the trace rows of the memory adapters key by the run id; the
// decision row keeps the wave (intent) id.
func TestMemoryRecordsRowsKeyByRunID(t *testing.T) {
	t.Parallel()
	for name, doc := range map[string]string{
		liveMemoryAdapter:    readRepoDoc(t, liveMemoryAdapter),
		catalogMemoryExample: readCatalog(t, catalogMemoryExample),
	} {
		requireIn(t, name, doc, []string{"`gabo/ledger/<run id>`", "`gabo/trace/<run id>`", "`gabo/decision/<wave>/"})
		for _, bad := range []string{"`gabo/ledger/<wave>`", "`gabo/trace/<wave>`"} {
			if strings.Contains(doc, bad) {
				t.Errorf("%s: a Records row still keys by %s", name, bad)
			}
		}
	}
}
