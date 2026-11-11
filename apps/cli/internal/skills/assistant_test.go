package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Spec 0009 S7: the gabo-assistant Activation Contract names its main-session
// entry per engine (default agent for Claude Code and OpenCode, scoped block
// for Codex and Pi) and keeps role and headless sessions in their role.
func TestAssistantActivationNamesTheMainSessionEntry(t *testing.T) {
	t.Parallel()
	b, err := content.ReadFile("gabo-assistant/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	_, contract, ok := strings.Cut(string(b), "## Activation Contract")
	if !ok {
		t.Fatal("gabo-assistant/SKILL.md: no ## Activation Contract section")
	}
	contract, _, _ = strings.Cut(contract, "\n## ")
	for _, s := range []string{
		"`runtimes.assistant`", "Main-Session Entry", "`gabo-agent-templates`",
		"Claude Code and OpenCode: the default agent `gabo-assistant`",
		"`agent` in `.claude/settings.json`", "`default_agent` in the OpenCode project config",
		"Codex and Pi: a scoped block between `<!-- GABO_ASSISTANT_START -->` and `<!-- GABO_ASSISTANT_END -->`",
		"`AGENTS.override.md`", "`AGENTS.md`", "`CLAUDE.md`",
		"started with a role prompt", "run headless", "keeps its role",
	} {
		if !strings.Contains(contract, s) {
			t.Errorf("gabo-assistant Activation Contract: missing %q", s)
		}
	}
}

// Spec 0010 S5: where the Activation Contract says who writes the
// main-session key, it names `gabo init`; the runtime skill builds the rest.
func TestAssistantActivationNamesGaboInitAsKeyWriter(t *testing.T) {
	t.Parallel()
	b, err := content.ReadFile("gabo-assistant/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	_, contract, _ := strings.Cut(string(b), "## Activation Contract")
	contract, _, _ = strings.Cut(contract, "\n## ")
	for _, s := range []string{
		"keys that `gabo init` writes",
		"the main-session entry that `gabo init` and the runtime skill of `runtimes.assistant` build",
	} {
		if !strings.Contains(contract, s) {
			t.Errorf("gabo-assistant Activation Contract: missing %q", s)
		}
	}
	assertKeyLinesNameGaboInit(t, "gabo-assistant/SKILL.md", contract)
}

// Intent 0013 S8, S12 (plan T9): the assistant knows its engine, starts waves
// and chains `wave.after: next` only on unclaimed intents, reports holders on
// request and posts the owner's release only on the owner's instruction.
func TestAssistantStartsWavesOnlyOnUnclaimedIntents(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		// its engine (the heading and sources: TestAssistantResolvesItsEngineOnEveryEntry).
		"`gabo-engine:`", "`{{engine}}`",
		// unclaimed only, at start and when chaining.
		"intent 0013 S8", "the earliest claim not followed by its own release holds the intent",
		"An intent another session holds is not eligible",
		"`next`: start the next eligible intent as in Wave Start, only an unclaimed one",
		// resume: only a wave launched in this same session; the trace is
		// shared, so another session's wave needs the owner's release first.
		"`resume <session id>`", "A wave you ran in this same session",
		"After a session restart, the holder's resume from the trace (D17)",
		"resume only when the owner asks you to resume that wave",
		"Never resume another session's wave on your own",
		"does not stop a wave that is still running",
		// a refused session writes nothing to the holder's trace.
		"Before any trace write, `read one` the intent issue",
		"write nothing to `gabo/trace/<wave>`", "a refused session never writes the holder's trace",
		// stale or orphan claims.
		"may be stale",
		// report holders.
		"which session holds which intent",
		"`<I####>: session <session id> (engine <engine>) since <UTC time>`", "`<I####>: unclaimed`",
		// owner release.
		"only on the owner's own instruction",
		"`intent released by owner for session <session id>`", "`log progress`", "`read one`",
	})
	_, gates, _ := strings.Cut(doc, "## Decision Gates")
	gates, _, _ = strings.Cut(gates, "\n## ")
	if !strings.Contains(gates, "`resume <session id>`") {
		t.Error("gabo-assistant Decision Gates: the cycle launch does not pass `resume`")
	}
	const line = "- Start a wave only on an intent no other session has claimed; the skill holds the claim rules."
	tpls := agentTemplates(t)
	for _, name := range []string{mainTemplate, mainBlock} {
		if !strings.Contains(tpls[name], line) {
			t.Errorf("%s: missing %q", name, line)
		}
	}
}

// Intent 0013 S16, D8 (plan T17): Claude Code and OpenCode read the engine
// from the agent file's `gabo-engine:` line, Codex and Pi from the block's
// fixed line; a session with neither refuses.
func TestAssistantResolvesItsEngineOnEveryEntry(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	requireIn(t, "gabo-assistant/SKILL.md", doc, []string{
		"Session engine (intent 0013 S6, S12, S16)",
		"Claude Code and OpenCode: the `gabo-engine:` line of your built agent file (`{{engine}}`)",
		"Codex and Pi: the fixed session-engine line in the opening scope paragraph of the block (`codex` when Codex loaded it, `pi` when Pi loaded it)",
		"`session engine unknown (rebuild the agents)`",
	})
}

// Intent 0013 S19, D13 (plan T20): a user file the engine denies, or whose
// read waits for an approval nobody gives, is unreadable, not missing: the
// assistant runs no `gabo-setup` for it, goes on to step 2 without the stable
// layer and reports one exact line; an absent file still runs `gabo-setup`.
func TestAssistantSessionStartToleratesAnUnreadableUserFile(t *testing.T) {
	t.Parallel()
	b, err := content.ReadFile("gabo-assistant/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	_, start, ok := strings.Cut(string(b), "## Session Start")
	if !ok {
		t.Fatal("gabo-assistant/SKILL.md: no ## Session Start section")
	}
	start, _, _ = strings.Cut(start, "\n## ")
	step1, _, ok := strings.Cut(start, "\n2. ")
	if !ok {
		t.Fatal("gabo-assistant Session Start: no step 2")
	}
	for _, s := range []string{
		"Absent file ⇒ note the setup start time (UTC ISO 8601) in the session, run `gabo-setup`",
		"a read the engine denies, or one that waits for an approval nobody can give",
		"is unreadable, not missing",
		"do not run `gabo-setup` for it",
		"continue with step 2 without the stable layer",
		"report exactly `user config unreadable (<reason>): continuing without alias, language, tone and interaction level`",
	} {
		if !strings.Contains(step1, s) {
			t.Errorf("gabo-assistant Session Start step 1: missing %q", s)
		}
	}
	if strings.Contains(step1, "Missing config ⇒") {
		t.Error("gabo-assistant Session Start step 1: \"Missing config ⇒\" no longer separates an absent file from an unreadable one")
	}
}

// Intent 0013 S19, D13 (plan T20): the Pi section of docs/compatibility.md
// names the gentle-pi permission for the read of the global user file.
func TestCompatibilityPiNamesTheUserFilePermission(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs/compatibility.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, pi, ok := strings.Cut(string(b), "\n## Pi package\n")
	if !ok {
		t.Fatal("docs/compatibility.md: no ## Pi package section")
	}
	pi, _, _ = strings.Cut(pi, "\n## ")
	for _, s := range []string{
		"`~/.config/gabo/config.yaml`", "gentle-pi",
		"approve that read once in an interactive session",
		"unattended Pi waves", "`/gentle:mode`",
		"user config unreadable (<reason>): continuing without alias, language, tone and interaction level",
	} {
		if !strings.Contains(pi, s) {
			t.Errorf("docs/compatibility.md Pi section: missing %q", s)
		}
	}
}

// Intent 0015 S1, S4, D1 (plan T4; supersedes intent 0013 S18 for every
// engine): the main session loads the gabo-cycle skill and runs the wave on
// every engine; no sentence launches a `gabo-cycle` subagent and Wave Start
// names no engine.
func TestAssistantRunsTheWaveInItsOwnSession(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	_, start, ok := strings.Cut(doc, "## Wave Start (J3)")
	if !ok {
		t.Fatal("gabo-assistant/SKILL.md: no Wave Start section")
	}
	start, _, _ = strings.Cut(start, "\n## ")
	requireIn(t, "gabo-assistant Wave Start", start, []string{
		"load the `gabo-cycle` skill (`.gabo/skills/gabo-cycle/SKILL.md`) in this main session and run it with the wave id, the mode and the wave start time, or `resume <session id>` when you resume",
		"each coder is one first-level in-process subagent of this session",
		"End your turn while the coders run; never poll",
	})
	for _, s := range []string{`\bPi\b`, `Claude Code`, `Codex`, `OpenCode`, `subagent_run`, `gentle-pi`} {
		if regexp.MustCompile(s).MatchString(start) {
			t.Errorf("gabo-assistant Wave Start: must not name %q", s)
		}
	}
	for _, s := range []string{"launch `gabo-cycle`", "`gabo-cycle` subagent", "launch no `gabo-cycle`", "subagent `gabo-cycle`"} {
		if strings.Contains(doc, s) {
			t.Errorf("gabo-assistant/SKILL.md: must not say %q", s)
		}
	}
	_, gates, _ := strings.Cut(doc, "## Decision Gates")
	gates, _, _ = strings.Cut(gates, "\n## ")
	if strings.Contains(gates, "resume a wave | `gabo-cycle`") {
		t.Error("gabo-assistant Decision Gates: a row still delegates the wave to `gabo-cycle`")
	}
	_, act, _ := strings.Cut(doc, "## Activation Contract")
	act, _, _ = strings.Cut(act, "\n## ")
	requireIn(t, "gabo-assistant Activation Contract", act, []string{
		"directiva (`gabo-pm`, `gabo-dev-lead`), `gabo-plan-reviewer` and the coders are your native subagents",
		"you run the wave yourself with the `gabo-cycle` skill",
		"the cycle runs in your session and its session engine is the engine of your main-session entry",
	})
	_, hard, _ := strings.Cut(doc, "## Hard Rules")
	hard, _, _ = strings.Cut(hard, "\n## ")
	requireIn(t, "gabo-assistant Hard Rules", hard, []string{
		"One exception: you run the `gabo-cycle` steps yourself; exploring, writing documents and writing code still go to subagents",
		"the session holds the wave's context",
		"the trace lets it compact or be resumed",
	})
	_, qs, _ := strings.Cut(doc, "## Questions by Mode (J4)")
	qs, _, _ = strings.Cut(qs, "\n## ")
	requireIn(t, "gabo-assistant Questions by Mode", qs, []string{"travels coder → assistant"})
	_, inter, _ := strings.Cut(doc, "## Intervene in a Task (J5)")
	inter, _, _ = strings.Cut(inter, "\n## ")
	requireIn(t, "gabo-assistant Intervene", inter, []string{
		"pass the owner's instruction straight to the running coder, or resume it by its ledger handle once it is HANDED",
		"between coder rounds end your turn and never poll",
	})
}

// Intent 0015 S5, PD11 (plan T16): the assistant resumes a coder that can be
// resumed and otherwise runs the cycle's new-coder rule; no task is left open.
func TestAssistantReplacesACoderThatCannotBeResumed(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-assistant/SKILL.md")
	merge := lineWith(t, doc, "After every merge the coder did not see itself")
	requireIn(t, "gabo-assistant merge rule", merge, []string{
		"resume the owning coder by its handle (wave report or ledger) with `PR #<n> merged` so it closes its task (D23)",
		"If it cannot be resumed (no transcript after a session close or restart), launch a new `gabo-coder` on the ledger worktree and branch (`gabo-cycle` step 3) with `PR #<n> merged`, only while this session holds the intent",
		"when another session holds it or nobody does, launch nothing and report the task and the holder to the owner. The new coder closes the task, and you never write DONE",
	})
	if strings.Contains(merge, "the task stays open") || strings.Contains(merge, "leave its resources to the `gabo-cycle` fallback sweep") {
		t.Errorf("gabo-assistant merge rule still leaves the task open: %q", merge)
	}
	resume := lineWith(t, doc, "5. **Resume from the traces.**")
	requireIn(t, "gabo-assistant Intervene resume", resume, []string{
		"A coder that cannot be resumed is replaced by a new `gabo-coder` as in Hard Rules, under the same holder condition",
	})
	if strings.Contains(resume, "is reported to the owner") {
		t.Errorf("gabo-assistant Intervene resume still only reports it: %q", resume)
	}
	res := lineWith(t, doc, "2. **Resume.**")
	requireIn(t, "gabo-assistant Resume", res, []string{
		"Never resume another session's wave on your own",
		"resume only when the owner asks you to resume that wave",
		"When the holder cannot resume a ledger coder, the cycle's step 3 launches a new `gabo-coder` for it",
	})
}
