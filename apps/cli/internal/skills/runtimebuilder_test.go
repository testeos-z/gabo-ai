package skills

import (
	"strings"
	"testing"
)

const runtimeBuilder = "gabo-runtime-builder/SKILL.md"

func runtimeBuilderDoc(t *testing.T) string {
	t.Helper()
	b, err := content.ReadFile(runtimeBuilder)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustContain(t *testing.T, doc string, want ...string) {
	t.Helper()
	for _, s := range want {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeBuilder, s)
		}
	}
}

// The builder researches from verified sources only (PRD R35): official docs
// first, rechecked against the installed version and its CLI help, a cited
// source and version for every fact, one concrete question when unclear, and
// doc links asked for when it has no web access or documentation MCP.
func TestRuntimeBuilderResearchesVerifiedSources(t *testing.T) {
	t.Parallel()
	mustContain(t, runtimeBuilderDoc(t),
		"official documentation first", "`<engine> --version`", "`<engine> --help`",
		"source URL and the version", "one concrete question",
		"no web access and no documentation MCP", "ask the owner for the documentation links",
		"never from memory",
	)
}

// An engine without subagents and skills, or an equivalent, is refused with
// the UX_UI section 4 message, and the engine is never installed or configured.
func TestRuntimeBuilderRefusesUnsupportedEngines(t *testing.T) {
	t.Parallel()
	mustContain(t, runtimeBuilderDoc(t),
		"✗ cannot add a runtime for <engine>",
		"why: it has no subagent and skill support, or an equivalent",
		"next: use Claude Code, Pi or another engine with subagents and skills",
		"Never install, upgrade or configure the engine",
	)
}

// The builder proves the build before it saves the skill: probe agent, then
// the spec agents, then gabo check, and only then the runtime skill, written
// outside the catalog copy so gabo init never removes it, with its agent_dir.
func TestRuntimeBuilderBuildsBeforeItWrites(t *testing.T) {
	t.Parallel()
	doc := runtimeBuilderDoc(t)
	mustContain(t, doc,
		"`.gabo/runtimes/gabo-runtime-<engine>/SKILL.md`", "`metadata.agent_dir`",
		"`gabo init` never removes it", "Never place `gabo-reviewer` in the project",
		"`gabo-runtime-claude`", "`gabo-runtime-pi`", "no Go change",
	)
	_, steps, ok := strings.Cut(doc, "## Build Steps")
	if !ok {
		t.Fatalf("%s: no ## Build Steps section", runtimeBuilder)
	}
	steps, _, _ = strings.Cut(steps, "\n## ")
	last := -1
	for _, s := range []string{"**Probe.**", "**Spec agents.**", "**Check.**", "**Write the runtime skill.**"} {
		i := strings.Index(steps, s)
		if i < 0 {
			t.Errorf("## Build Steps: missing step %s", s)
			continue
		}
		if i < last {
			t.Errorf("## Build Steps: step %s out of order", s)
		}
		last = i
	}
}

// The written runtime skill has the shape of the shipped ones (T052, T053):
// the same sections, a tool map row for every capability and a fill for
// every template slot.
func TestRuntimeBuilderNamesTheRuntimeSkillShape(t *testing.T) {
	t.Parallel()
	doc := runtimeBuilderDoc(t)
	_, shape, ok := strings.Cut(doc, "## Runtime Skill Shape")
	if !ok {
		t.Fatalf("%s: no ## Runtime Skill Shape section", runtimeBuilder)
	}
	shape, _, _ = strings.Cut(shape, "\n## ")
	for _, s := range []string{
		"`## Activation Contract`", "`## Hard Rules`", "`## Sources`", "`## Tool Map`",
		"`## Fixed-Name Links`", "`## Build Steps`", "`## Output Contract`",
		"`gabo-version`", "`metadata.agent_dir`",
	} {
		if !strings.Contains(shape, s) {
			t.Errorf("## Runtime Skill Shape: missing %s", s)
		}
	}
	for _, c := range []string{"read", "search", "edit", "shell", "git", "tracker", "repository-host", "memory", "web", "skills", "subagents"} {
		if !strings.Contains(shape, "`"+c+"`") {
			t.Errorf("## Runtime Skill Shape: capability %s not named", c)
		}
	}
	for _, s := range agentSlots {
		if !strings.Contains(shape, "`{{"+s+"}}`") {
			t.Errorf("## Runtime Skill Shape: does not say how to fill {{%s}}", s)
		}
	}
}
