package skills

import (
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/agentspec"
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
	for p := range tpls {
		if !used[p] {
			t.Errorf("%s: template has no spec entry", p)
		}
	}
}

func TestAgentTemplatesUseNamedSlotsOnly(t *testing.T) {
	t.Parallel()
	for p, doc := range agentTemplates(t) {
		id := strings.TrimSuffix(path.Base(p), ".md")
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
		"Fill only the named slots", "invent nothing", ".gbai/VERSION", versionLine,
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
