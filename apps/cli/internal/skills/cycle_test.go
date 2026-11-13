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
		"exactly `PR #<n> head <40-hex pushed sha>`",
		"`PR #<n> head <40-hex pushed sha> engine <engine>`",
		"only when `runtimes.reviewer` in `gabo.yaml` is `session`",
		"your session engine", "`gabo-engine:`",
	})
	tmpl := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	requireIn(t, "gabo-coder template", tmpl, []string{
		"build the request `PR #<n> head <40-hex sha>`",
		"`PR #<n> head <40-hex sha> engine {{engine}}`",
		"only when `runtimes.reviewer` in `gabo.yaml` is `session`",
	})
}

// Intent 0013 S11 (plan T8): the coder and the reviewer notes
// describe the optional engine field; the coder passes it as an argument.
func TestReviewerDescribesTheOptionalEngineField(t *testing.T) {
	t.Parallel()
	tmpl := agentTemplates(t)[templatesPrefix+"gabo-coder.md"]
	requireIn(t, "gabo-coder template", tmpl, []string{
		"`PR #<n> head <40-hex sha> engine {{engine}}`",
		"`^PR #([1-9][0-9]*) head ([0-9a-f]{40}) engine (claude|codex|pi|opencode)$`",
		"only when `runtimes.reviewer` in `gabo.yaml` is `session`",
		"`bash \"{{reviewer_script}}\" head <n> <sha> engine <engine>`",
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
		claim   = "`intent claimed by session <session id> (engine <engine>) run <run id>`"
		refusal = "`intent <I####> is claimed by session <session id> (engine <engine>) since <UTC time>; it is released when that wave closes`"
		release = "`intent released by session <session id>`"
		owner   = "`intent released by owner for session <session id>`"
	)
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"intent 0013 S8",
		// the session id: made once per wave start, kept in the trace.
		"`<engine>-<8 hex>`", "once per wave start", "`gabo/trace/<run id>`",
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

// Intent 0013 S9, D6 (plan T10): each wave keeps its own resources. Every
// ledger line names the session id, the worktrees live under a per-wave
// folder, and the fallback sweep touches only this wave's ledger: a resource
// of another wave is neither removed nor reported as a leftover of this wave.
func TestCycleKeepsAndCleansOnlyItsOwnWave(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"intent 0013 S9", "D6",
		// the ledger line names the session id.
		"One line per resource: kind, name or path, creator, task, the session id of this wave, state (live, removed, leftover)",
		// the worktree namespace.
		"`<repo-parent>/<repo-name>-worktrees/<wave>/<task>`",
		"`git worktree add <repo-parent>/<repo-name>-worktrees/<wave>/<task> -b <branch> origin/dev`",
		// the sweep touches only this wave's ledger.
		"only entries of this wave's ledger (`gabo/ledger/<run id>`)",
		"A resource of another wave",
		"is neither removed nor reported as a leftover of this wave",
	})
	if strings.Contains(doc, "`git worktree add <path>") {
		t.Error("gabo-cycle/SKILL.md: the worktree path must be the per-wave path, not a free `<path>`")
	}
	rule := lineWith(t, doc, "Resource ledger")
	if !strings.Contains(rule, "session id") {
		t.Errorf("gabo-cycle/SKILL.md: the ledger rule does not name the session id: %q", rule)
	}
	sweep := lineWith(t, doc, "fallback sweep) runs only")
	for _, s := range []string{"this wave's ledger", "another wave"} {
		if !strings.Contains(sweep, s) {
			t.Errorf("gabo-cycle/SKILL.md: the fallback sweep rule does not say %q", s)
		}
	}
}

// lineWith returns the first line of doc containing s.
func lineWith(t *testing.T, doc, s string) string {
	t.Helper()
	for _, l := range strings.Split(doc, "\n") {
		if strings.Contains(l, s) {
			return l
		}
	}
	t.Fatalf("no line contains %q", s)
	return ""
}

// Intent 0015 S1, S4, S5, D1 (plan T4; supersedes intent 0013 S18 for every
// engine): the main session loads this skill on every engine; "return to the
// assistant" means continuing with its Wave Close; each coder is one
// first-level in-process subagent; no sentence names a `gabo-cycle` subagent
// or agent file, and no rule on where the cycle runs names one engine.
func TestCycleRunsInTheMainSession(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	_, act, _ := strings.Cut(doc, "## Activation Contract")
	act, _, _ = strings.Cut(act, "\n## ")
	requireIn(t, "gabo-cycle Activation Contract", act, []string{
		"Load in the main session, on every engine: `gabo-assistant` loads this skill there and runs it with the same inputs",
		"\"return to the assistant\" means continue with the assistant's Wave Close in the same session",
	})
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"the engine of your main-session entry: the `gabo-engine:` line of your built agent file (`{{engine}}`, intent 0013 S6) or the fixed session-engine line of the block",
		"each coder is one first-level in-process subagent of the main session, and its handle goes to the ledger",
		"never a named teammate in its own pane unless the owner asks",
		"a compacted or restarted main session resumes the wave from the tracker, the ledger and the trace, without asking the owner to repeat anything",
		"Between coders the session ends its turn and never polls",
	})
	for _, s := range []string{
		"`gabo-cycle` subagent", "`gabo-cycle` agent", "subagent_run", "subagent_continue", "gentle-pi", "On a Pi main session",
		"on a Pi main session", "Claude Code, Codex and OpenCode",
	} {
		if strings.Contains(doc, s) {
			t.Errorf("gabo-cycle/SKILL.md: must not say %q", s)
		}
	}
	step5 := lineWith(t, doc, "5. Per task:")
	if !strings.Contains(step5, "launch `gabo-coder` as a first-level in-process subagent of this session") {
		t.Error("gabo-cycle step 5: the coder launch does not name a first-level in-process subagent")
	}
}

// Intent 0015 S1: the wave report loads in the main session.
func TestWaveReportLoadsInTheMainSession(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-wave-report/SKILL.md")
	requireIn(t, "gabo-wave-report/SKILL.md", doc, []string{"Load in the main session when it runs the `gabo-cycle` skill"})
	if strings.Contains(doc, "Load in `gabo-cycle`") {
		t.Error("gabo-wave-report/SKILL.md: still loads in `gabo-cycle`")
	}
}

// Intent 0015 S3, S5, PD10 (plan T15): a wave that returns with HANDED tasks
// says that it keeps its intent claim.
func TestCycleReturnWithHandedTasksKeepsTheClaim(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"A wave that returns with HANDED tasks says that it keeps its intent claim and that the claim ends with the wave close or the owner's release.",
	})
	// the rule it states already holds.
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"The wave closes when every task of that intent is DONE or BLOCKED",
	})
}

// Intent 0015 S5, PD11 (plan T16): a resumed holder launches a new coder for
// a coder that cannot be resumed; the sweep keeps the entries of open tasks.
func TestCycleResumeLaunchesANewCoderForAnUnresumableOne(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	step3 := lineWith(t, doc, "3. Resume (D17)")
	requireIn(t, "gabo-cycle step 3", step3, []string{
		"a coder handle that cannot be resumed (no transcript after a session close or restart)",
		"launch a new `gabo-coder` on the ledger worktree and branch with the same input as a first launch (task, worktree path, branch, mode)",
		"`PR #<n> merged` when `repository_host` shows the PR merged",
		"when the PR is `CONFLICTING`, with the instruction to run the coder's `CONFLICTING` fix round (merge `origin/dev` into the task branch, never a rebase, never a force-push, resolve the conflict, run the task's checks, push, request a new review)",
		"read the PR status and the latest reviewer comment and continue from them",
		"When the worktree or the branch is already gone and the PR is merged, the new coder runs the Task Close from the main checkout (verify the merge through `repository_host`, close the issue with DONE via PR; nothing else to remove)",
		"give it the main checkout path (your own working directory) in its input",
		"the new coder recreates it in the main checkout with `git worktree add <worktree path> <branch>`",
		"The new coder is the only writer of DONE",
	})
	coder := lineWith(t, doc, "The coder owns its task close")
	if strings.Contains(coder, "its merged task stays open") || strings.Contains(coder, "leaves its entry to the fallback sweep") {
		t.Errorf("gabo-cycle coder rule still leaves a merged task open: %q", coder)
	}
	requireIn(t, "gabo-cycle coder rule", coder, []string{"the holder launches a new `gabo-coder` for it (step 3)"})
}

// The sweep keeps its 0.2.10 rule (only entries no coder could close, only
// after the PR is merged) and never takes the entry of a task still open.
func TestCycleSweepKeepsOpenTasks(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	sweep := lineWith(t, doc, "fallback sweep) runs only")
	requireIn(t, "gabo-cycle sweep rule", sweep, []string{
		"only for entries no coder could close, only after `repository_host` shows the PR merged",
		"never removes the worktree or the branch of a task that is neither DONE nor BLOCKED, even when its PR is merged (its issue is open and a coder can still close it, step 3)",
		"so it takes an entry only after its task is DONE or BLOCKED; a BLOCKED task's issue stays open, and its entries are swept only when its PR is merged, as before",
	})
	step7 := lineWith(t, doc, "7. The wave closes when")
	requireIn(t, "gabo-cycle step 7", step7, []string{
		"The wave closes when every task of that intent is DONE or BLOCKED",
		"skip the entries of a task that is neither DONE nor BLOCKED",
	})
}

// Existing rules the T16 change must not touch.
func TestCycleResumeRulesStayPinned(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-cycle/SKILL.md")
	requireIn(t, "gabo-cycle/SKILL.md", doc, []string{
		"3. Resume (D17), only as the holder: read `gabo/trace/<run id>` and `gabo/ledger/<run id>`",
		"The cycle skips DONE tasks (closed issues).",
		"An open PR (`repository_host` read status) or a live ledger worktree ⇒ reattach: resume its coder by its handle, or launch a new coder on that worktree and branch.",
		"When tracker, host and trace disagree (for example the trace says HANDED and the issue has no PR), record the disagreement in the report and leave that task alone; never guess.",
		"The coder owns its task close and is the only writer of DONE (D23); the cycle never writes DONE. A coder that gets `merge=merged` closes on its own",
		"a resume keeps a claim only when the assistant passes `resume <session id>` and that id equals the session id in the trace",
		"The holder keeps its claim on resume",
		"Skip `worktree.keep` entries. A removal that fails is REFUSED",
	})
	coder := readCatalog(t, "gabo-coder/SKILL.md")
	requireIn(t, "gabo-coder/SKILL.md", coder, []string{
		"`verdict=clean merge=skipped`, PR `CONFLICTING`",
		"`verdict=clean merge=merged`",
		"run it from the main checkout, the parent of",
	})
}
