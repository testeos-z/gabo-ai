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

// The CLI has four commands (D24, R31); setup uses no other and no gabo MCP tool.
func TestSetupUsesNoRemovedCommand(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	requireAll(t, doc, []string{"no gabo MCP tool", "`gabo version`", "`gabo prompt`", "Four commands"})
	for _, s := range []string{"generate", "config set", "mcp__gabo", "gbai"} {
		if strings.Contains(doc, s) {
			t.Errorf("%s: names removed command or tool %q", setupSkill, s)
		}
	}
	// Built from parts so a repository grep for the stale rule stays empty.
	if stale := "Three" + " commands"; strings.Contains(doc, stale) {
		t.Errorf("%s: still says %q", setupSkill, stale)
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

// sharedMemory returns the ## Shared memory section of the setup skill.
func sharedMemory(t *testing.T) string {
	t.Helper()
	_, sec, ok := strings.Cut(setupDoc(t), "## Shared memory")
	if !ok {
		t.Fatalf("%s: no ## Shared memory section", setupSkill)
	}
	sec, _, _ = strings.Cut(sec, "\n## ")
	return sec
}

// Before the memory slot is configured, setup explains the shared-memory
// recommendation (T070, R37, R38): one backend and one project name for every
// agent except the reviewer (R9, D6), local-only accepted.
func TestSetupExplainsSharedMemoryBeforeTheMemorySlot(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	_, steps, _ := strings.Cut(doc, "## Steps")
	ask, _, _ := strings.Cut(steps, "**Adapter.**")
	if !strings.Contains(ask, "explain **Shared memory** first") {
		t.Errorf("%s: the Ask step does not explain Shared memory before the memory slot", setupSkill)
	}
	sec := sharedMemory(t)
	for _, s := range []string{
		"R37", "R38", "one memory backend and one memory project name", "every agent of the project",
		"local and cloud", "every developer", "except the reviewer", "never reads or writes it",
		"R9", "D6", "no memory slot", "local-only memory is acceptable", "solo developer",
	} {
		if !strings.Contains(sec, s) {
			t.Errorf("%s ## Shared memory: missing %q", setupSkill, s)
		}
	}
}

// Engram with Engram Cloud sync comes first with its reason; claude-mem and
// Mem0 are listed with their sharing capability as user-supplied adapters
// (PRD "Memory backends researched", open question 13); nothing is applied
// without the owner's answer (R12).
func TestSetupOffersEngramCloudFirstAndListsTheOthers(t *testing.T) {
	t.Parallel()
	sec := sharedMemory(t)
	engram := strings.Index(sec, "Engram with Engram Cloud sync")
	claudeMem := strings.Index(sec, "claude-mem")
	mem0 := strings.Index(sec, "Mem0")
	if engram < 0 || claudeMem < 0 || mem0 < 0 || engram > claudeMem || engram > mem0 {
		t.Fatalf("%s ## Shared memory: Engram with Engram Cloud sync must come before claude-mem and Mem0", setupSkill)
	}
	for _, s := range []string{
		"recommended", "because", "only use Claude Code", "across machines", "across developers",
		"user-supplied adapters", "open question 13", "R12", "owner's answer",
	} {
		if !strings.Contains(sec, s) {
			t.Errorf("%s ## Shared memory: missing %q", setupSkill, s)
		}
	}
}

// Setup records only the adapter and the memory project name, in the
// project-owned adapter file, never in the catalog copy; a write under a
// human-only path needs a human merge; no backend setup, no secret.
func TestSetupRecordsOnlyTheMemoryAdapterAndProjectName(t *testing.T) {
	t.Parallel()
	sec := sharedMemory(t)
	for _, s := range []string{
		"`slots.memory.adapter`", "memory project name", "project-owned", "never in the catalog copy",
		"`.gabo/skills/`", "`gabo init` replaces", "no new config key", "`review.human_only_paths`",
		"human merge", "never auto-merges", "never sets up the backend", "credentials", "access", "no secret",
	} {
		if !strings.Contains(sec, s) {
			t.Errorf("%s ## Shared memory: missing %q", setupSkill, s)
		}
	}
}

// The memory smoke test writes and reads only under a scratch topic and
// deletes the probe afterwards, so nothing stays in the project memory.
func TestSetupMemoryProbeUsesAScratchTopicAndDeletesIt(t *testing.T) {
	t.Parallel()
	doc := setupDoc(t)
	_, smoke, ok := strings.Cut(doc, "**Smoke test.**")
	if !ok {
		t.Fatalf("%s: no smoke test step", setupSkill)
	}
	smoke, _, _ = strings.Cut(smoke, "\n7. ")
	at := 0
	for _, s := range []string{"`gabo/setup-probe/<timestamp>`", "read it back", "delete it", "reads back nothing"} {
		i := strings.Index(smoke[at:], s)
		if i < 0 {
			t.Fatalf("%s smoke test: %q missing or out of order", setupSkill, s)
		}
		at += i + len(s)
	}
	for _, s := range []string{
		"only under", "nothing stays in the project memory", "stays unverified",
		"backend's own memory tool", "no delete operation", "writes nothing", "read-only `search`",
	} {
		if !strings.Contains(smoke, s) {
			t.Errorf("%s smoke test: missing %q", setupSkill, s)
		}
	}
}

// The memory contract defines no delete, so the probe cannot be deleted
// through a contract operation: the smoke test must name the backend's own
// tool for it. This test fails if the contract gains a delete, so the smoke
// step is revisited then.
func TestMemoryProbeDeletionMatchesTheContract(t *testing.T) {
	t.Parallel()
	b, err := content.ReadFile("gabo-contract-memory/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "### ") && strings.Contains(strings.ToLower(line), "delete") {
			t.Fatalf("gabo-contract-memory now has %q: make the setup smoke test delete the probe through it", line)
		}
	}
}

// A catalog memory adapter is copied to a project-owned file before the
// memory project name is written (never into the catalog copy).
func TestSetupCopiesACatalogMemoryAdapterBeforeWriting(t *testing.T) {
	t.Parallel()
	sec := sharedMemory(t)
	for _, s := range []string{"points into the catalog copy", "project-owned copy", "`gabo-adapter-writer`", "points `slots.memory.adapter` at the copy"} {
		if !strings.Contains(sec, s) {
			t.Errorf("%s ## Shared memory: missing %q", setupSkill, s)
		}
	}
}
