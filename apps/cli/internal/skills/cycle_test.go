package skills

import (
	"io/fs"
	"strings"
	"testing"
)

// readCatalog returns one catalog file as text.
func readCatalog(t *testing.T, path string) string {
	t.Helper()
	b, err := fs.ReadFile(Catalog(), path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// requireIn fails for every want string missing from doc.
func requireIn(t *testing.T, name, doc string, want []string) {
	t.Helper()
	for _, s := range want {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", name, s)
		}
	}
}

// Intent 0013 S7 (plan T8): the cycle runs the wave on its session engine,
// read from its built agent's `{{engine}}`; in-process coders always run on
// that engine, and a fixed other coder engine is named in the wave report
// (open question 2, owner); `parallelism` bounds each wave (open question 3,
// owner).
func TestCycleRunsTheWaveOnItsSessionEngine(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		// where the session engine comes from.
		"session engine", "`gabo-engine:`", "`{{engine}}`", "intent 0013 S6",
		// coders: session or equal ⇒ in-process as today.
		"`runtimes.coder` is `session` or equal to your session engine",
		"in-process subagents as today",
		// another fixed engine ⇒ still in-process, and the report says so.
		"coders still run in-process on the session engine",
		"`coders ran on <session engine> (runtimes.coder is <engine>; subagents inherit the session engine)`",
		// parallelism: today's text kept, bounded per wave.
		"`parallelism` bounds the coders in parallel within this intent",
		"bounds this wave only", "no project-wide counter",
		// the report names the session engine.
		"`session engine: <engine>`",
	})
}

// Intent 0013 S7 and S11 (plan T8): the reviewer gate resolves the review
// engine first (`session` is the session engine) and runs the preflight with
// that engine as an argument under `session` when it is `codex` or
// `opencode`, before any claim; single values keep today's step.
func TestCycleResolvesTheReviewEngineBeforeThePreflight(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	const withEngine = "`bash \"${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh\" preflight engine <engine>`"
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"resolve the review engine first",
		"`runtimes.reviewer` `session` is your session engine",
		"a fixed `runtimes.reviewer` is that engine",
		withEngine,
		"only under `session`",
		// single values: today's step 1 kept word for word.
		"1. Reviewer preflight, only when `runtimes.reviewer` is `codex` or `opencode`, auto mode included: run the user-level copy, never a copy from a repository checkout, with no extra arguments or environment: `bash \"${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh\" preflight`.",
	})
	steps := doc[strings.Index(doc, "## Execution Steps"):]
	pre, claim := strings.Index(steps, withEngine), strings.Index(steps, "Claim each")
	if pre < 0 || claim < 0 || pre > claim {
		t.Errorf("gabo-cycle/SKILL.md: the session preflight (at %d) must come before `Claim each` (at %d)", pre, claim)
	}
	for _, s := range []string{"REVIEWER_ENGINE", "ENGINE=<engine>"} {
		if strings.Contains(doc, s) {
			t.Errorf("gabo-cycle/SKILL.md: passes the engine as environment (%q)", s)
		}
	}
}

// Intent 0013 S11 (plan T8): the coder adds `engine <engine>` to its review
// request only when `runtimes.reviewer` is `session`; the plain request stays.
func TestCoderSendsTheEngineOnlyUnderSession(t *testing.T) {
	t.Parallel()
	skill := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", skill, []string{
		"request one review from the `gabo-reviewer` subagent with exactly `PR #<n> head <40-hex pushed sha>`",
		"`PR #<n> head <40-hex pushed sha> engine <engine>`",
		"only when `runtimes.reviewer` in `gabo.yaml` is `session`",
		"your session engine", "`gabo-engine:`",
	})
	tmpl := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	requireIn(t, "gabo-coder template", tmpl, []string{
		"launch it with exactly `PR #<n> head <40-hex sha>`",
		"`PR #<n> head <40-hex sha> engine {{engine}}`",
		"only when `runtimes.reviewer` in `gabo.yaml` is `session`",
	})
}

// Intent 0013 S11 (plan T8): the reviewer relay and the reviewer notes
// describe the optional engine field; the relay passes it as an argument.
func TestReviewerDescribesTheOptionalEngineField(t *testing.T) {
	t.Parallel()
	tmpl := agentTemplates(t)[templatesPrefix+"gabo-reviewer.md"]
	requireIn(t, "gabo-reviewer template", tmpl, []string{
		"`PR #<n> head <sha> engine <engine>`",
		"`^PR #([1-9][0-9]*) head ([0-9a-f]{40}) engine (claude|codex|pi|opencode)$`",
		"only when `runtimes.reviewer` in `gabo.yaml` is `session`",
		"`bash \"{{reviewer_script}}\" head <n> <sha> engine <engine>`",
		"as an argument, never as environment",
	})
	notes := readCatalog(t, "gabo-reviewer-notes/SKILL.md")
	requireIn(t, "gabo-reviewer-notes/SKILL.md", notes, []string{
		"`PR #<n> head <40-hex sha> engine <engine>`",
		"only when `runtimes.reviewer` is `session`",
		"the engine of the wave that requests the review",
		"a fixed `runtimes.reviewer` reviews every wave on that engine",
	})
}

// Intent 0013 S7 (plan T8): the wave report names the session engine and,
// for a fixed other coder engine, the in-process line.
func TestWaveReportNamesTheSessionEngine(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report/SKILL.md", doc, []string{
		"`session engine: <engine>`",
		"`coders ran on <session engine> (runtimes.coder is <engine>; subagents inherit the session engine)`",
	})
}

// Intent 0013 S8, D4 (plan T9): the wave claims its intent through the
// existing `tasks` operations on the intent issue, after the reviewer
// preflight and before any task claim; the earliest claim not followed by its
// own release holds it; a non-holder releases, claims nothing and refuses
// naming the holder; the holder releases at wave close and keeps its claim on
// resume (open question 1, owner: no timeout, owner release).
func TestCycleClaimsItsIntentBeforeAnyTask(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	const (
		claim   = "`intent claimed by session <session id> (engine <engine>)`"
		refusal = "`intent <I####> is claimed by session <session id> (engine <engine>) since <UTC time>; it is released when that wave closes`"
		release = "`intent released by session <session id>`"
		owner   = "`intent released by owner for session <session id>`"
	)
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"intent 0013 S8",
		// the session id: made once per wave start, kept in the trace.
		"`<engine>-<8 hex>`", "once per wave start", "`gabo/trace/<wave>`",
		// the claim goes through existing operations on the intent issue.
		"`log progress`", "`read one`", "on the intent issue", claim,
		"no new `tasks` operation (D4)",
		// who holds it.
		"the earliest claim not followed by its own release holds the intent",
		"in the tracker's comment order",
		// a non-holder.
		"posts its release", "claims nothing", refusal,
		"The owner can release it by telling any assistant to release session <session id>.",
		// the holder.
		"keeps its claim on resume", release, "at wave close",
		// a stale claim: no timeout, owner release.
		"no timeout", owner, "orphan claim",
		// the trace is shared: a resume keeps a claim only with its own id,
		// and a refused session writes nothing to it.
		"`resume <session id>`", "never adopted on its own",
		"that id equals the session id in the trace", "never adopting the trace's id",
		"`session id: <session id>`", "a refused session writes nothing to the trace",
		// one refusal form; an owner release stops the next coder launch.
		"returns only three lines", "re-checks it with `read one` before each coder launch",
		"when you no longer hold the intent (an owner release), launch no more coders",
	})
	_, steps, ok := strings.Cut(doc, "## Execution Steps")
	if !ok {
		t.Fatal("gabo-cycle/SKILL.md: no ## Execution Steps section")
	}
	pre := strings.Index(steps, "preflight engine <engine>")
	at := strings.Index(steps, "Intent claim")
	task := strings.Index(steps, "Claim each")
	if pre < 0 || at < 0 || task < 0 || !(pre < at && at < task) {
		t.Errorf("gabo-cycle/SKILL.md: want the preflight (%d), then the intent claim (%d), then `Claim each` (%d)", pre, at, task)
	}
	_, closing, ok := strings.Cut(steps, "The wave closes")
	if !ok || !strings.Contains(closing, release) {
		t.Errorf("gabo-cycle/SKILL.md: wave close does not post %s", release)
	}
	// D4: the claim adds no operation to the `tasks` contract.
	contract := readCatalog(t, "gabo-contract-tasks/SKILL.md")
	if strings.Contains(strings.ToLower(contract), "### claim intent") {
		t.Error("gabo-contract-tasks/SKILL.md: D4 adds no `claim intent` operation")
	}
}
