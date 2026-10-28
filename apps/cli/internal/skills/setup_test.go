package skills

import (
	"strings"
	"testing"
)

const setupSkill = "gabo-setup/SKILL.md"

func setupDoc(t *testing.T) string {
	t.Helper()
	b, err := content.ReadFile(setupSkill)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func requireAll(t *testing.T, doc string, want []string) {
	t.Helper()
	for _, s := range want {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", setupSkill, s)
		}
	}
}

// The steps run in the J1 order (T019): ask, adapter, write the config, build
// the agents, gabo check, per-slot smoke test.
func TestSetupFollowsTheJ1Order(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	_, steps, ok := strings.Cut(doc, "## Steps")
	if !ok {
		t.Fatalf("%s: no ## Steps section", setupSkill)
	}
	steps, _, _ = strings.Cut(steps, "\n## ")
	order := []string{
		"**Ask.**", "**Adapter.**", "**Write the config.**", "**Build the agents.**", "**Check.**", "**Smoke test.**",
	}
	at := 0
	for _, s := range order {
		i := strings.Index(steps[at:], s)
		if i < 0 {
			t.Fatalf("%s: step %s missing or out of order", setupSkill, s)
		}
		at += i + len(s)
	}
}

// Setup delegates every write and build to the skills that own it.
func TestSetupUsesTheOwningSkills(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{
		"`gabo-adapter-writer`", "`gabo-config-writer`", "`gabo-agent-templates`",
		"`gabo-runtime-claude`", "`gabo-runtime-pi`", "`gabo-runtime-builder`",
		"`gabo check`", "`gabo init`", "`gabo-reviewer`",
		"fix every failure", "rerun `gabo check` until it passes",
	})
}

// The questions asked once (T019), with no default for the reviewer runtime.
func TestSetupAsksTheTeamQuestions(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{
		"`mode`", "`plan_change.scope`", "`plan_change.global_without_owner`",
		"`runtimes.coder`", "`runtimes.reviewer`", "`runtimes.assistant`", "no default",
		"`language`", "once", "where each agent runs", "`slots.agent_host`",
		"which tracker", "`slots.tasks`", "how the agents communicate",
	})
}

// The recommended stacks are offered, never required.
func TestSetupOffersRecommendedNotRequired(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{
		"gentle-ai", "gentle-pi", "recommended, not required", "only if the owner accepts",
	})
}

// No value without the owner's answer (R12), no secret (R21).
func TestSetupAppliesOnlyOwnerAnswers(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{"R12", "owner's answer", "R21", "env var name"})
}

// The CLI has three commands (D24); setup uses no other and no gabo MCP tool.
func TestSetupUsesNoRemovedCommand(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	requireAll(t, doc, []string{"no gabo MCP tool", "`gabo version`"})
	for _, s := range []string{"generate", "config set", "mcp__gabo", "gbai"} {
		if strings.Contains(doc, s) {
			t.Errorf("%s: names removed command or tool %q", setupSkill, s)
		}
	}
}

// A failed smoke test leaves the slot unverified (APP_FLOW A3); the setup
// start time is saved in memory (R28).
func TestSetupVerifiesSlotsAndRecordsTheStart(t *testing.T) {
	t.Parallel()
	requireAll(t, setupDoc(t), []string{
		"subagent", "harmless", "unverified", "A3", "names the failing operation",
		"R28", "setup start", "save decision", "after the `memory` slot is verified",
	})
}

// After a gabo upgrade the rebuild step refreshes the catalog and rebuilds
// every agent (TRD 9.4, PRD open question 11), keyed on gabo check's lines.
func TestSetupHasTheRebuildStep(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	_, rebuild, ok := strings.Cut(doc, "## Rebuild")
	if !ok {
		t.Fatalf("%s: no ## Rebuild section", setupSkill)
	}
	rebuild, _, _ = strings.Cut(rebuild, "\n## ")
	for _, s := range []string{
		"this project uses gabo <v>; run gabo init to upgrade it", "✗ no stamp: run gabo init",
		"built with <v>, catalog is <w>: ask your agent to rebuild",
		"run `gabo init`", "never touches the config", "user-level `gabo-reviewer`",
		"rebuild every agent", "rerun `gabo check`", "is newer than gabo",
	} {
		if !strings.Contains(rebuild, s) {
			t.Errorf("%s ## Rebuild: missing %q", setupSkill, s)
		}
	}
}
