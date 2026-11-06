package skills

import (
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
)

// agentSlots are the named slots an agent template may carry (TRD 8.1, D24):
// identity, the version line, the engine line (intent 0013 S6), the
// methodology link and the adapter of each slot. A runtime skill fills
// exactly these and nothing else.
var agentSlots = []string{
	"id", "role", "gabo_version", engineSlot, "methodology",
	"adapter:tasks", "adapter:repository_host", "adapter:memory",
	"reviewer_script",
}

const versionLine = "gabo-version: {{gabo_version}}"

// engineSlot is the slot of the engine line (intent 0013 S6); every runtime
// skill fills it (plan 0013 T6, T7), so it is one of agentSlots.
const engineSlot = "engine"

// engineLine names the engine a built agent was built for (intent 0013 S6,
// D3); it follows the version line in every template that carries one.
const engineLine = "gabo-engine: {{engine}}"

// engineSlotRow is the slot table's meaning of {{engine}} (intent 0013 S6).
const engineSlotRow = "| `{{engine}}` | the engine the agent is built for, as written in `gabo.yaml`"

var slotPattern = regexp.MustCompile(`\{\{([^{}]*)\}\}`)

// engineSyntax are fragments that tie a body to one engine: frontmatter keys,
// engine tool names, engine directories and MCP tool ids.
var engineSyntax = []string{
	"tools:", "skills:", "model:", "mcp__", ".claude", ".pi/", "Claude", "subagent_type",
	"Bash", "WebFetch", "WebSearch", "Grep", "Glob", "$ARGUMENTS",
}

const templatesPrefix = agentspec.Dir + "/agents/"

func agentTemplates(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for p, doc := range embeddedByPath(t) {
		if strings.HasPrefix(p, templatesPrefix) {
			out[p] = doc
		}
	}
	if len(out) == 0 {
		t.Fatal("no agent templates embedded")
	}
	return out
}

func TestEverySpecAgentHasATemplateAndEveryTemplateASpecEntry(t *testing.T) {
	t.Parallel()
	spec, err := agentspec.Load(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	tpls := agentTemplates(t)
	used := map[string]bool{}
	for _, a := range spec.Agents {
		p := path.Join(agentspec.Dir, a.Template)
		if want := "agents/" + a.ID + ".md"; a.Template != want {
			t.Errorf("agent %s: template %q, want %q (named after the spec id)", a.ID, a.Template, want)
		}
		if _, ok := tpls[p]; !ok {
			t.Errorf("agent %s: template %s not embedded", a.ID, p)
		}
		used[p] = true
	}
	if m := spec.MainSession; m != nil {
		for _, f := range []string{m.Template, m.Block} {
			p := path.Join(agentspec.Dir, f)
			if _, ok := tpls[p]; !ok {
				t.Errorf("main session %s: %s not embedded", m.ID, p)
			}
			used[p] = true
		}
	}
	for p := range tpls {
		if !used[p] {
			t.Errorf("%s: template has no spec entry", p)
		}
	}
}

func TestAgentTemplatesUseNamedSlotsOnly(t *testing.T) {
	t.Parallel()
	spec, err := agentspec.Load(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	tpls := agentTemplates(t)
	for _, a := range spec.Agents {
		p := path.Join(agentspec.Dir, a.Template)
		doc := tpls[p]
		id := a.ID
		lines := strings.Split(doc, "\n")
		if lines[0] != versionLine {
			t.Errorf("%s: first line %q, want %q", p, lines[0], versionLine)
		}
		assertEngineLineNextToVersion(t, p, doc)
		if !strings.Contains(doc, "{{id}}") {
			t.Errorf("%s: missing the identity slot {{id}}", p)
		}
		if strings.Contains(doc, id) {
			t.Errorf("%s: hardcodes its id %q; use {{id}}", p, id)
		}
		for _, m := range slotPattern.FindAllStringSubmatch(doc, -1) {
			if !slices.Contains(agentSlots, m[1]) {
				t.Errorf("%s: unknown slot %q", p, m[0])
			}
		}
		rest := slotPattern.ReplaceAllString(doc, "")
		if strings.Contains(rest, "{{") || strings.Contains(rest, "}}") {
			t.Errorf("%s: malformed slot braces", p)
		}
		for _, s := range append(engineSyntax, "Placeholder") {
			if strings.Contains(doc, s) {
				t.Errorf("%s: engine-specific or placeholder text %q", p, s)
			}
		}
		if strings.HasPrefix(lines[2], "---") {
			t.Errorf("%s: frontmatter is the runtime skill's job", p)
		}
	}
}

// assertEngineLineNextToVersion requires exactly one engine line, right after
// the version line (intent 0013 S6, D3).
func assertEngineLineNextToVersion(t *testing.T, p, doc string) {
	t.Helper()
	lines := strings.Split(doc, "\n")
	if len(lines) < 3 || lines[1] != engineLine {
		t.Errorf("%s: second line %q, want %q next to the version line", p, lines[min(1, len(lines)-1)], engineLine)
	}
	if n := strings.Count(doc, "gabo-engine:"); n != 1 {
		t.Errorf("%s: %d engine lines, want exactly one", p, n)
	}
}

// Intent 0013 S6, D3: every template under agents/ that carries the version
// line also carries one engine line next to it; the block, which has no
// version line and no slots, carries none.
func TestEveryAgentTemplateNamesItsEngine(t *testing.T) {
	t.Parallel()
	n := 0
	for p, doc := range agentTemplates(t) {
		if p == mainBlock {
			if strings.Contains(doc, "gabo-engine:") {
				t.Errorf("%s: the block is copied verbatim and has no slots; it carries no engine line", p)
			}
			continue
		}
		n++
		if !strings.HasPrefix(doc, versionLine+"\n") {
			t.Errorf("%s: does not start with %q", p, versionLine)
		}
		assertEngineLineNextToVersion(t, p, doc)
	}
	if n == 0 {
		t.Fatal("no agent template with a version line")
	}
}

func TestReviewerTemplateIsARelay(t *testing.T) {
	t.Parallel()
	doc := agentTemplates(t)[templatesPrefix+"gabo-reviewer.md"]
	for _, s := range []string{
		"`PR #<n> head <sha> ready for review`", "positive integer", "40-hex",
		"`^PR #([1-9][0-9]*) head ([0-9a-f]{40})( ready for review)?$`",
		"{{reviewer_script}}", "`verdict=retry`", "exactly once", "`VERDICT `",
		"never review", "edit", "comment", "merge",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-reviewer template: missing %q", s)
		}
	}
}

func TestAgentTemplatesSkillTellsToFillOnlyTheSlots(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)[agentspec.Dir+"/SKILL.md"]
	want := []string{
		"Fill only the named slots", "invent nothing", ".gabo/VERSION", versionLine,
		"named after its spec `id`", "## Slots", engineSlotRow, "`" + engineLine + "`",
		"built for <value>, not <engine>", "no engine line passes",
	}
	for _, s := range agentSlots {
		want = append(want, "`{{"+s+"}}`")
	}
	for _, s := range want {
		if !strings.Contains(skill, s) {
			t.Errorf("%s/SKILL.md: missing %q", agentspec.Dir, s)
		}
	}
}

const (
	mainTemplate = templatesPrefix + "gabo-assistant.md"
	mainBlock    = templatesPrefix + "gabo-assistant-block.md"
	blockStart   = "<!-- GABO_ASSISTANT_START -->"
	blockEnd     = "<!-- GABO_ASSISTANT_END -->"
)

// mainSessionContent is what spec 0009 S2 asks every main-session entry to
// tell the session.
var mainSessionContent = []string{
	"the gabo assistant", "load the `gabo-assistant` skill", "follow its Session Start", "before your first reply",
	"`.gabo/skills/gabo-assistant/SKILL.md`", "single point of contact", "never block on long work",
	"`gabo-pm`", "`gabo-dev-lead`", "`gabo-cycle`", "`gabo-plan-reviewer`",
}

func containsFold(doc, s string) bool {
	return strings.Contains(strings.ToLower(doc), strings.ToLower(s))
}

// Spec 0009 S2, D4: the assistant agent template is engine-neutral, starts
// with the version line and tells the session it is the gabo assistant.
func TestMainSessionTemplateMakesTheSessionTheAssistant(t *testing.T) {
	t.Parallel()
	doc, ok := agentTemplates(t)[mainTemplate]
	if !ok {
		t.Fatalf("%s not embedded", mainTemplate)
	}
	lines := strings.Split(doc, "\n")
	if lines[0] != versionLine {
		t.Errorf("%s: first line %q, want %q", mainTemplate, lines[0], versionLine)
	}
	assertEngineLineNextToVersion(t, mainTemplate, doc)
	if strings.HasPrefix(lines[2], "---") {
		t.Errorf("%s: frontmatter is the runtime skill's job", mainTemplate)
	}
	for _, m := range slotPattern.FindAllStringSubmatch(doc, -1) {
		if m[1] != "gabo_version" && m[1] != engineSlot {
			t.Errorf("%s: slot %q; the main-session body is the same in the block, which has no slots", mainTemplate, m[0])
		}
	}
	for _, s := range mainSessionContent {
		if !containsFold(doc, s) {
			t.Errorf("%s: missing %q", mainTemplate, s)
		}
	}
	for _, s := range append(engineSyntax, "Placeholder") {
		if strings.Contains(doc, s) {
			t.Errorf("%s: engine-specific or placeholder text %q", mainTemplate, s)
		}
	}
}

// Spec 0009 S3: the scoped block opens with the role-prompt and headless
// rule, then carries the template body byte for byte, between the markers,
// in at most 2 KiB.
func TestMainSessionBlockIsScopedAndSmall(t *testing.T) {
	t.Parallel()
	tpls := agentTemplates(t)
	doc, ok := tpls[mainBlock]
	if !ok {
		t.Fatalf("%s not embedded", mainBlock)
	}
	if n := len(doc); n > 2048 {
		t.Errorf("%s: %d bytes, want at most 2048", mainBlock, n)
	}
	if !strings.HasPrefix(doc, blockStart+"\n") || !strings.HasSuffix(doc, "\n"+blockEnd+"\n") {
		t.Fatalf("%s: not wrapped in %s ... %s", mainBlock, blockStart, blockEnd)
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(doc, blockStart+"\n"), blockEnd+"\n")
	if strings.Count(doc, blockStart) != 1 || strings.Count(doc, blockEnd) != 1 {
		t.Errorf("%s: markers must appear once each", mainBlock)
	}
	rule, body, ok := strings.Cut(inner, "\n\n")
	if !ok {
		t.Fatalf("%s: no blank line after the scope rule", mainBlock)
	}
	for _, s := range []string{"role prompt", "headless", "`codex exec`", "`pi --print`", "ignores this block", "keeps its role"} {
		if !strings.Contains(rule, s) {
			t.Errorf("%s: scope rule misses %q", mainBlock, s)
		}
	}
	_, tplBody, _ := strings.Cut(tpls[mainTemplate], "\n\n")
	if body != tplBody {
		t.Errorf("%s: content after the scope rule differs from %s after its version line\nblock:\n%s\ntemplate:\n%s", mainBlock, mainTemplate, body, tplBody)
	}
	if strings.Contains(doc, "{{") || strings.Contains(doc, "}}") {
		t.Errorf("%s: the block is written verbatim; it has no slots", mainBlock)
	}
	for _, s := range mainSessionContent {
		if !containsFold(body, s) {
			t.Errorf("%s: missing %q", mainBlock, s)
		}
	}
}

// Spec 0009 S1 to S4, D4: the one table of each engine's main-session entry
// lives in gabo-agent-templates and is built for every engine listed in
// runtimes.assistant (intent 0013 S4).
func TestAgentTemplatesSkillHoldsTheMainSessionEntryTable(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)[agentspec.Dir+"/SKILL.md"]
	for _, s := range []string{
		"## Main-Session Entry", "for every engine listed in `runtimes.assistant`", "`main_session`",
		"`.claude/skills/gabo-assistant`", "`.agents/skills/gabo-assistant`", "`.gabo/skills/gabo-assistant/SKILL.md`",
		"`.claude/agents/gabo-assistant.md`", "`.claude/settings.json`", "`agent: \"gabo-assistant\"`",
		"`.opencode/agents/gabo-assistant.md`", "`mode: primary`", "`default_agent: \"gabo-assistant\"`",
		"`AGENTS.override.md`", "`AGENTS.md`", "`CLAUDE.md`", "`" + blockStart + "`", "`" + blockEnd + "`",
		"2048 bytes", "| Claude Code |", "| OpenCode |", "| Codex |", "| Pi |",
	} {
		if !strings.Contains(skill, s) {
			t.Errorf("%s/SKILL.md: missing %q", agentspec.Dir, s)
		}
	}
}

// secretPatterns match common credential shapes; none may ship in the skill.
var secretPatterns = regexp.MustCompile(
	`gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|sk-[A-Za-z0-9_-]{20,}|` +
		`AKIA[0-9A-Z]{16}|xox[abposr]-[A-Za-z0-9-]{10,}|-----BEGIN [A-Z ]*PRIVATE KEY-----`)

func TestAgentTemplatesSkillHasNoSecrets(t *testing.T) {
	t.Parallel()
	if !secretPatterns.MatchString("ghp_" + strings.Repeat("a", 36)) {
		t.Fatal("secret pattern does not match a sample token")
	}
	for p, doc := range embeddedByPath(t) {
		if strings.HasPrefix(p, agentspec.Dir+"/") && secretPatterns.MatchString(doc) {
			t.Errorf("%s: matches a secret pattern", p)
		}
	}
}

// The plan reviewer and the directiva agents that open plan changes load the
// one plan-change reviewer skill (T032, D24).
func TestPlanChangeAgentsLoadThePlanReviewerSkill(t *testing.T) {
	t.Parallel()
	spec, err := agentspec.Load(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	need := map[string]bool{"gabo-plan-reviewer": true, "gabo-pm": true, "gabo-dev-lead": true}
	for _, a := range spec.Agents {
		if need[a.ID] {
			delete(need, a.ID)
			if !slices.Contains(a.Skills, "gabo-plan-reviewer") {
				t.Errorf("agent %s: skills %v miss gabo-plan-reviewer", a.ID, a.Skills)
			}
		}
	}
	for id := range need {
		t.Errorf("agent %s: not in the spec", id)
	}
}

// Spec 0010 S5: the Main-Session Entry table names `gabo init` as the writer of
// `agent` and `default_agent`; the runtime skill builds the agent file and
// link, and writes no engine config key.
func TestAgentTemplatesMainSessionEntryNamesGaboInitAsKeyWriter(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)[agentspec.Dir+"/SKILL.md"]
	_, sec, ok := strings.Cut(skill, "## Main-Session Entry")
	if !ok {
		t.Fatalf("%s/SKILL.md: no ## Main-Session Entry section", agentspec.Dir)
	}
	sec, _, _ = strings.Cut(sec, "\n## ")
	for _, s := range []string{
		"`gabo init` writes the one key `agent: \"gabo-assistant\"`",
		"`gabo init` writes the one key `default_agent: \"gabo-assistant\"`",
		"the runtime skill of each listed engine builds the agent file and the link",
		"`gabo init` removes the key of each full-tier engine not listed in `runtimes.assistant`",
	} {
		if !strings.Contains(sec, s) {
			t.Errorf("%s/SKILL.md ## Main-Session Entry: missing %q", agentspec.Dir, s)
		}
	}
	for _, s := range []string{"merged into `.claude/settings.json`", "Merge only the one key", "writes and refreshes its row"} {
		if strings.Contains(sec, s) {
			t.Errorf("%s/SKILL.md ## Main-Session Entry: still says the runtime skill writes the key: %q", agentspec.Dir, s)
		}
	}
	assertKeyLinesNameGaboInit(t, agentspec.Dir+"/SKILL.md", sec)
}

// Intent 0013 S4: the Main-Session Entry rules build the entry for every
// listed assistant engine, remove another full-tier engine's entry only when
// that engine is not listed, and keep a shared link or block while any
// listed engine uses it.
func TestAgentTemplatesMainSessionEntryFollowsTheAssistantList(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)[agentspec.Dir+"/SKILL.md"]
	_, sec, ok := strings.Cut(skill, "## Main-Session Entry")
	if !ok {
		t.Fatalf("%s/SKILL.md: no ## Main-Session Entry section", agentspec.Dir)
	}
	sec, _, _ = strings.Cut(sec, "\n## ")
	for _, s := range []string{
		"Build the entry for every engine listed in `runtimes.assistant`",
		"each other full-tier engine not listed in `runtimes.assistant`",
		"A shared link or block stays while any listed engine uses it",
		"never removes the entry of another listed engine",
	} {
		if !strings.Contains(sec, s) {
			t.Errorf("%s/SKILL.md ## Main-Session Entry: missing %q", agentspec.Dir, s)
		}
	}
	for _, s := range []string{"only for `runtimes.assistant`", "is `runtimes.assistant`", "of each other engine"} {
		if strings.Contains(sec, s) {
			t.Errorf("%s/SKILL.md ## Main-Session Entry: still reads one assistant engine: %q", agentspec.Dir, s)
		}
	}
}
