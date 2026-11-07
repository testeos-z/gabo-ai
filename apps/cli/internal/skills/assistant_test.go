package skills

import (
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
		// its engine.
		"Session engine (intent 0013 S6, S12)", "`gabo-engine:`", "`{{engine}}`",
		// unclaimed only, at start and when chaining.
		"intent 0013 S8", "the earliest claim not followed by its own release holds the intent",
		"An intent another session holds is not eligible",
		"`next`: start the next eligible intent as in Wave Start, only an unclaimed one",
		// resume.
		"`resume`",
		// report holders.
		"which session holds which intent",
		"`<I####>: session <session id> (engine <engine>) since <UTC time>`", "`<I####>: unclaimed`",
		// owner release.
		"only on the owner's own instruction",
		"`intent released by owner for session <session id>`", "`log progress`", "`read one`",
	})
	_, gates, _ := strings.Cut(doc, "## Decision Gates")
	gates, _, _ = strings.Cut(gates, "\n## ")
	if !strings.Contains(gates, "`resume`") {
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
