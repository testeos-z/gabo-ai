package skills

import (
	"strings"
	"testing"
)

const oldBulletPhrase = "hand writing, exploring and long commands"

// Intent 0021 T2 (S2, S3, S5, S6, S7): the neutral template and the block say
// to do the work yourself and hand off only for a named reason.
func TestAssistantTemplateAndBlockNarrowTheDelegationRule(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"gabo-agent-templates/agents/gabo-assistant.md",
		"gabo-agent-templates/agents/gabo-assistant-block.md",
	} {
		doc := readCatalog(t, path)
		requireIn(t, path, doc, []string{
			"Do the work yourself",
			"only for a named reason",
			"`gabo-assistant` skill",
			"name the reason in the handoff",
			"never block on long work",
			"`.gabo/skills/gabo-assistant/SKILL.md`",
			"single point of contact",
		})
		if strings.Contains(doc, oldBulletPhrase) {
			t.Errorf("%s: still has %q", path, oldBulletPhrase)
		}
		// The handoff sentence itself names no roles; roles sit in the
		// separate role-ownership line the pinned strings of
		// agenttemplates_test.go require.
		for _, line := range strings.Split(doc, "\n") {
			if strings.Contains(line, "hand it off only") && strings.Contains(line, "`gabo-p") {
				t.Errorf("%s: the handoff sentence names roles: %q", path, line)
			}
		}
	}
	block := readCatalog(t, "gabo-agent-templates/agents/gabo-assistant-block.md")
	requireIn(t, "block", block, []string{
		"<!-- GABO_ASSISTANT_START -->", "<!-- GABO_ASSISTANT_END -->",
		"Scope: this block applies only to the owner's interactive main session. A session started with a role prompt (any gabo agent, a reviewer prompt) or run headless (`codex exec`, `pi --print`) ignores this block and keeps its role.",
		"Session engine: loaded by Codex, this session's engine is `codex`; loaded by Pi, it is `pi`.",
	})
}

func TestAssistantRoleLineNoLongerDelegatesAllHeavyWork(t *testing.T) {
	t.Parallel()
	doc := readCatalog(t, "gabo-agent-templates/agents.yaml")
	if strings.Contains(doc, "delegates all heavy work") {
		t.Error("agents.yaml: role line still says it delegates all heavy work")
	}
	requireIn(t, "agents.yaml", doc, []string{
		"single point of contact in the main session; hands work off only for a named reason.",
	})
}
