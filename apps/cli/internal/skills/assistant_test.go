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
