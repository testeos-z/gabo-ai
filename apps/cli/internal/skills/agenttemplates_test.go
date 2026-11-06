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
// identity, the version line, the methodology link and the adapter of each
// slot. A runtime skill fills exactly these and nothing else.
var agentSlots = []string{
	"id", "role", "gabo_version", "methodology",
	"adapter:tasks", "adapter:repository_host", "adapter:memory",
	"reviewer_script",
}

const versionLine = "gabo-version: {{gabo_version}}"

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
		if strings.HasPrefix(lines[1], "---") {
			t.Errorf("%s: frontmatter is the runtime skill's job", p)
		}
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
		"named after its spec `id`", "## Slots",
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
	if strings.HasPrefix(lines[1], "---") {
		t.Errorf("%s: frontmatter is the runtime skill's job", mainTemplate)
	}
	for _, m := range slotPattern.FindAllStringSubmatch(doc, -1) {
		if m[1] != "gabo_version" {
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
// lives in gabo-agent-templates and is built only for runtimes.assistant.
func TestAgentTemplatesSkillHoldsTheMainSessionEntryTable(t *testing.T) {
	t.Parallel()
	skill := embeddedByPath(t)[agentspec.Dir+"/SKILL.md"]
	for _, s := range []string{
		"## Main-Session Entry", "only for `runtimes.assistant`", "`main_session`",
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
		"the runtime skill of that engine builds the agent file and the link",
		"`gabo init` removes the key of each other engine",
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
