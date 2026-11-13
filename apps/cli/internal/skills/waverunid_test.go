package skills

import (
	"regexp"
	"strings"
	"testing"
)

// Intent 0025 S1 (plan T6): the wave run id `<I####>-<UTC start>` keys the
// trace, the ledger and `--wave`; the claim comment carries it; a resume reads
// it from the claim comment that holds the intent, then by `mem_search`, then the old key.
func TestWaveRunIDAssistant(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant", doc, []string{
		"note the wave start time once (UTC ISO 8601) and build the run id from it as `<I####>-<YYYYMMDDTHHMMSSZ>`",
		"the comment `intent claimed by session <session id> (engine <engine>) run <run id>` claims it",
		"the run id is the one at the end of the claim comment of the session being resumed",
		"run `read wave trace` with `mem_search` for `gabo/trace/<I####>-`, asking for 20 hits, the tool's maximum",
		"take the one that sorts last, the latest run",
		"read `gabo/trace/<I####>` (a trace written before the run id)",
		"`gabo/trace/<I####>/planning`",
		"`resume <session id>` together with the run id you read when you resume",
		"then the run id is the intent id `<I####>`, so the ledger is `gabo/ledger/<I####>` and `--wave <I####>`",
		"`gabo/trace/<run id>`",
		"--wave <run id>",
		"the claim comment of the session being resumed, the claim that holds the intent (the earliest claim not followed by its own release), never the newest comment",
		"a refused session posts a claim of its own with its own run id before its release",
		"read the session id and the run id (the run id of the claim that holds the intent, as Wave Start says)",
	})
	// The resume reads the holder's claim, never the newest comment.
	if strings.Contains(doc, "run id is the one at the end of the newest claim comment") {
		t.Error("gabo-assistant: the resume must not read the newest claim comment")
	}
}

// Intent 0026 S15 (plan T8): the run-id fallback considers only the TRD 5.1
// run-id keys, compared case-insensitively, asks for the tool's maximum of 20
// hits, and reads a full page without a run-id key as incomplete, never as no run.
// Both the resume fallback and the memory example carry the same rule.
var runIDFallbackPins = []string{
	"asking for 20 hits, the tool's maximum",
	"`gabo/trace/<I####>-<8 digits>T<6 digits>Z`",
	"`<8 digits>` the UTC date YYYYMMDD, `<6 digits>` the UTC time HHMMSS",
	"compared case-insensitively because Engram lowercases topic keys",
	"take the one that sorts last, the latest run",
	"`gabo/trace/i0001-w8-t001`",
	"`gabo/trace/i0001-20261113t145700z`",
	"`gabo/trace/i0001-20261113t150111z`",
	"it takes `gabo/trace/i0001-20261113t150111z`",
	"a full page of 20 hits with no run-id key is an incomplete lookup, never no run",
}

func TestRunIDFallbackMatchesOnlyRunIDKeys(t *testing.T) {
	t.Parallel()
	for _, p := range []string{"gabo-assistant/SKILL.md", "gabo-contract-memory/examples/memory-engram.md"} {
		requireIn(t, p, readCatalog(t, p), runIDFallbackPins)
	}
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
		"the cycle builds the run id itself and returns it to the assistant on a `run id: <run id>` line just before the `session id:` line",
		"a new run id built from it as `<I####>-<YYYYMMDDTHHMMSSZ>`, never adopting the trace's session id or the passed run id",
		"the stamps are kept per run id, so a new start never finds an earlier run's stamps",
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
