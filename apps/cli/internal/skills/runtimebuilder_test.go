package skills

import (
	"slices"
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
// the UX_UI section 4 message, and the engine is never installed or upgraded.
func TestRuntimeBuilderRefusesUnsupportedEngines(t *testing.T) {
	t.Parallel()
	mustContain(t, runtimeBuilderDoc(t),
		"✗ cannot add a runtime for <engine>",
		"why: it has no subagent and skill support, or an equivalent",
		"next: use Claude Code, Pi or another engine with subagents and skills",
		"Never install or upgrade the engine",
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
	for _, s := range append(slices.Clone(agentSlots), engineSlot) {
		if !strings.Contains(shape, "`{{"+s+"}}`") {
			t.Errorf("## Runtime Skill Shape: does not say how to fill {{%s}}", s)
		}
	}
}

// engineConfigRule is the narrowed engine-config rule every runtime skill
// carries (intent 0004 S8, D7, section 5): gabo writes only the engine keys it
// needs to operate, at project level only, merged, reported and verified by
// gabo check, and never the user's global engine config.
var engineConfigRule = []string{
	"gabo writes only the engine keys it needs to operate",
	"at project level only",
	"merged with the existing project config",
	"reports what it wrote",
	"verified by `gabo check`",
	"never the user's global engine config",
}

// assertEngineConfigRule checks that doc carries the narrowed rule and none
// of the retired blanket bans.
func assertEngineConfigRule(t *testing.T, name, doc string, retired ...string) {
	t.Helper()
	for _, s := range engineConfigRule {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", name, s)
		}
	}
	for _, s := range retired {
		if strings.Contains(doc, s) {
			t.Errorf("%s: still carries the retired rule %q", name, s)
		}
	}
}

// The builder's engine-config rule is narrowed (intent 0004 S8, D7): no
// install or upgrade, only the project-level keys gabo needs to operate.
func TestRuntimeBuilderNarrowsTheEngineConfigRule(t *testing.T) {
	t.Parallel()
	doc := runtimeBuilderDoc(t)
	assertEngineConfigRule(t, runtimeBuilder, doc,
		"Never edit the engine's own config files, project or global",
		"Never install, upgrade or configure the engine",
	)
	mustContain(t, doc, "Never install or upgrade the engine")
}

// A runtime skill the builder writes declares where its agent files keep
// their skill list and name, in the forms gabo check reads (intent 0008 D1).
func TestRuntimeBuilderTeachesAgentPlaces(t *testing.T) {
	t.Parallel()
	doc := runtimeBuilderDoc(t)
	_, shape, ok := strings.Cut(doc, "## Runtime Skill Shape")
	if !ok {
		t.Fatalf("%s: no ## Runtime Skill Shape section", runtimeBuilder)
	}
	shape, _, _ = strings.Cut(shape, "\n## ")
	mustContain(t, shape, "`metadata.agent_skills`", "`metadata.agent_name`",
		"`frontmatter:<key>`", "`toml:<key>`", "`line:<prefix>`")
}

// Spec 0009 D5 and S7: a new runtime skill declares its main-session entry in
// metadata.main_session, as the four shipped runtime skills do, or reports
// that its engine has none (a flexible-tier engine then gets one warning).
func TestRuntimeBuilderTeachesTheMainSessionMetadata(t *testing.T) {
	t.Parallel()
	mustContain(t, runtimeBuilderDoc(t),
		"`metadata.main_session`",
		"`kind: agent` with `config` (the project config files in resolution order) and `key` (the JSON key)",
		"`kind: block` with `files` (the instructions files in resolution order)",
		"`<agent_dir>/gabo-assistant.<ext>`",
		"joined with \" or \"",
		"report that `<engine>` has no main-session entry",
		"`! runtime <engine>: gabo-runtime-<engine> declares no metadata.main_session; main-session entry not checked for <engine> (flexible tier)`",
		"`✗ <file>: gabo assistant entry missing for <engine>`",
		"Main-Session Entry", "`main_session.template`", "`main_session.block`",
		"removes the gabo entry of every other full-tier engine",
	)
}

// runtimeBuilderLine returns the first line of the builder skill that holds
// marker, or fails the test.
func runtimeBuilderLine(t *testing.T, marker string) string {
	t.Helper()
	for _, l := range strings.Split(runtimeBuilderDoc(t), "\n") {
		if strings.Contains(l, marker) {
			return l
		}
	}
	t.Fatalf("%s: no line holds %q", runtimeBuilder, marker)
	return ""
}

// intent 0010 S5, D4 A: gabo init writes the engine config keys of the
// full-tier engines; a builder-written runtime skill keeps writing its own
// main-session key for kind: agent, the one exception (flexible tier, PRD
// R32, TRD D28), and never removes another engine's key, which gabo init
// removes.
func TestRuntimeBuilderKeepsTheFlexibleTierKeyWrite(t *testing.T) {
	t.Parallel()
	rule := runtimeBuilderLine(t, "Engine config (intent 0004 S8, D7)")
	for _, s := range []string{
		"`gabo init` writes the engine config keys of the full-tier engines (intent 0010 S5)",
		"step 6",
	} {
		if !strings.Contains(rule, s) {
			t.Errorf("engine-config rule: missing %q", s)
		}
	}
	step := runtimeBuilderLine(t, "6. **Main-session entry.**")
	for _, s := range []string{
		"the one key merged into the first existing file of `config`",
		"the one exception to \"`gabo init` writes every engine config key\"",
		"flexible tier", "PRD R32", "TRD D28", "intent 0010 D4",
		"never removes Claude Code's `agent` key or OpenCode's `default_agent` key: `gabo init` removes them",
	} {
		if !strings.Contains(step, s) {
			t.Errorf("step 6: missing %q", s)
		}
	}
}

// intent 0013 S4, S6: a builder-written runtime skill fills {{engine}} with
// its own engine, builds its main-session entry when its engine is listed in
// runtimes.assistant (keeping its own key write, intent 0010 D4) and never
// removes a listed engine's entry, so rebuilds in either order keep both.
func TestRuntimeBuilderBuildsEveryListedEntry(t *testing.T) {
	t.Parallel()
	step5 := runtimeBuilderLine(t, "5. **Spec agents.**")
	for _, s := range []string{
		"`gabo-engine: <engine>`",
		"without its first two lines `gabo-version: …` and `gabo-engine: …` and the blank line after them",
	} {
		if !strings.Contains(step5, s) {
			t.Errorf("step 5: missing %q", s)
		}
	}
	step6 := runtimeBuilderLine(t, "6. **Main-session entry.**")
	for _, s := range []string{
		"When `<engine>` is listed in `runtimes.assistant` (one engine or a list, intent 0013 S4)",
		"every other full-tier engine not listed in `runtimes.assistant`",
		"never removes the entry of another engine listed in `runtimes.assistant`",
		"with `runtimes.assistant: [claude, <engine>]`, rebuilding either engine first leaves both entries",
		"keeps writing its own key while `<engine>` is listed (intent 0010 D4)",
	} {
		if !strings.Contains(step6, s) {
			t.Errorf("step 6: missing %q", s)
		}
	}
	mustContain(t, runtimeBuilderDoc(t),
		"`{{engine}}` as `<engine>`, the engine as written in `gabo.yaml`",
		"the main-session entry when `<engine>` is listed in `runtimes.assistant` and the removal of every other full-tier engine's gabo entry not listed in `runtimes.assistant`",
	)
	mustLackIn(t, runtimeBuilder, runtimeBuilderDoc(t), "When `runtimes.assistant` is `<engine>`")
}
