package skills

import (
	"regexp"
	"strings"
	"testing"
)

// Intent 0025 S1 (plan T6): the wave run id `<I####>-<UTC start>` keys the
// trace, the ledger and `--wave`; the claim comment carries it; a resume reads
// it from the newest claim comment, then by `mem_search`, then the old key.
func TestWaveRunIDAssistant(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant", doc, []string{
		"note the wave start time once (UTC ISO 8601) and build the run id from it as `<I####>-<YYYYMMDDTHHMMSSZ>`",
		"the comment `intent claimed by session <session id> (engine <engine>) run <run id>` claims it",
		"the run id is the one at the end of the newest claim comment of the intent issue (`run <run id>`)",
		"run `read wave trace` with `mem_search` for `gabo/trace/<I####>-`",
		"take the observation whose topic key starts with that text and sorts last",
		"read `gabo/trace/<I####>` (a trace written before the run id)",
		"`gabo/trace/<I####>/planning`",
		"`gabo/trace/<run id>`",
		"--wave <run id>",
	})
}

func TestWaveRunIDCycleAndReport(t *testing.T) {
	t.Parallel()
	cycle := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle", cycle, []string{
		"`gabo/trace/<run id>`", "`gabo/ledger/<run id>`", "--wave <run id>",
		"`intent claimed by session <session id> (engine <engine>) run <run id>`",
		"`gabo/trace/<I####>/planning`",
		"the worktree namespace",
		"extended with `save wave trace` (search the key, then `mem_update`, else `mem_save`)",
		"no save of one run overwrites another's",
		"gabo/waves/<run id>/stamps",
	})
	report := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report", report, []string{
		"`gabo/trace/<run id>`", "--wave <run id>", "run id (`<I####>-<YYYYMMDDTHHMMSSZ>`)",
	})
}

// No trace or ledger key, and no `--wave` argument, keeps `<wave>`.
func TestWaveRunIDNoWaveLeftInKeys(t *testing.T) {
	t.Parallel()
	bad := regexp.MustCompile("gabo/(trace|ledger)/<wave>|--wave <wave>|gabo/waves/<wave>")
	for _, p := range []string{"gabo-assistant/SKILL.md", "gabo-cycle/SKILL.md", "gabo-wave-report/SKILL.md"} {
		for _, m := range bad.FindAllString(readCatalog(t, p), -1) {
			t.Errorf("%s: %q still keys by <wave>", p, m)
		}
	}
	if !strings.Contains(readCatalog(t, "gabo-cycle/SKILL.md"), "-worktrees/<wave>/<task>") {
		t.Error("gabo-cycle: the worktree namespace keeps the intent id")
	}
}
