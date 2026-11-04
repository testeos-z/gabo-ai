package skills

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const runtimeOpenCode = "gabo-runtime-opencode/SKILL.md"

func runtimeOpenCodeDoc(t *testing.T) string {
	t.Helper()
	b, err := content.ReadFile(runtimeOpenCode)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// section returns the body of "## <name>" up to the next "## " heading.
func openCodeSection(t *testing.T, doc, name string) string {
	t.Helper()
	_, s, ok := strings.Cut(doc, "\n## "+name+"\n")
	if !ok {
		t.Fatalf("%s: no ## %s section", runtimeOpenCode, name)
	}
	s, _, _ = strings.Cut(s, "\n## ")
	return s
}

// The OpenCode runtime skill ships in the catalog (intent 0004 S2, D1).
func TestRuntimeOpenCodeIsACatalogSkill(t *testing.T) {
	t.Parallel()
	if !slices.Contains(Names(), "gabo-runtime-opencode") {
		t.Fatalf("Names() = %v, lacks gabo-runtime-opencode", Names())
	}
}

// gabo check reads agent_dir and skill_dir from the frontmatter; OpenCode
// has no reviewer link, so reviewer_link is absent (S2, D5).
func TestRuntimeOpenCodeDeclaresItsDirs(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	rest, _ := strings.CutPrefix(doc, "---\n")
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("%s: no frontmatter", runtimeOpenCode)
	}
	var fm struct {
		Name     string         `yaml:"name"`
		Metadata map[string]any `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(head), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Name != "gabo-runtime-opencode" {
		t.Errorf("name = %q, want gabo-runtime-opencode", fm.Name)
	}
	if got := fm.Metadata["agent_dir"]; got != ".opencode/agents" {
		t.Errorf("metadata.agent_dir = %v, want .opencode/agents", got)
	}
	if got := fm.Metadata["skill_dir"]; got != ".opencode/skills" {
		t.Errorf("metadata.skill_dir = %v, want .opencode/skills", got)
	}
	if got, ok := fm.Metadata["reviewer_link"]; ok {
		t.Errorf("metadata.reviewer_link = %v, want no such key (D5)", got)
	}
}

// The skill has the sections of its siblings, in this order (S2).
func TestRuntimeOpenCodeSections(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	var got []string
	for _, l := range strings.Split(doc, "\n") {
		if h, ok := strings.CutPrefix(l, "## "); ok {
			got = append(got, h)
		}
	}
	want := []string{"Activation Contract", "Hard Rules", "Sources", "Tool Map", "Fixed-Name Links", "Build Steps", "Output Contract"}
	if !slices.Equal(got, want) {
		t.Errorf("sections = %v, want %v", got, want)
	}
}

// Sources are dated, name OpenCode 2.0.32 and carry the recheck rule that
// stops the build when the version differs and a row no longer holds (S2).
func TestRuntimeOpenCodeCitesItsSources(t *testing.T) {
	t.Parallel()
	s := openCodeSection(t, runtimeOpenCodeDoc(t), "Sources")
	for _, want := range []string{
		"2026-11-03", "OpenCode 2.0.32", "`opencode --version`",
		"When it is not 2.0.32", "A row that no longer holds: stop, build nothing",
		"https://opencode.ai/docs/agents", "https://opencode.ai/docs/skills",
		"https://opencode.ai/docs/permissions", "https://opencode.ai/docs/tools",
		"(binary)",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("## Sources: missing %q", want)
		}
	}
}

var markdownFence = regexp.MustCompile("(?ms)^( *)```markdown\n(.*?)\n *```$")

// The agent file holds only description (double-quoted role), mode: subagent
// and permission, with no allow; its body starts with the version comment,
// then the Skills: line (S3, D2, D3).
func TestRuntimeOpenCodeWritesTheAgentShape(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	m := markdownFence.FindStringSubmatch(doc)
	if m == nil {
		t.Fatalf("%s: no ```markdown agent block", runtimeOpenCode)
	}
	lines := strings.Split(m[2], "\n")
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, m[1])
	}
	block := strings.Join(lines, "\n")
	rest, ok := strings.CutPrefix(block, "---\n")
	if !ok {
		t.Fatalf("agent block does not start with frontmatter:\n%s", block)
	}
	front, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("agent block frontmatter not closed:\n%s", block)
	}
	var keys []string
	for _, l := range strings.Split(front, "\n") {
		if l != "" && l[0] != ' ' {
			k, _, _ := strings.Cut(l, ":")
			keys = append(keys, k)
		}
	}
	if want := []string{"description", "mode", "permission"}; !slices.Equal(keys, want) {
		t.Errorf("frontmatter keys = %v, want %v", keys, want)
	}
	for _, want := range []string{"description: \"<role>\"", "mode: subagent", "gabo-reviewer: deny"} {
		if !strings.Contains(front, want) {
			t.Errorf("frontmatter lacks %q:\n%s", want, front)
		}
	}
	if strings.Contains(block, "allow") {
		t.Errorf("agent block sets allow (D3 is deny-only):\n%s", block)
	}
	bl := strings.Split(body, "\n")
	if len(bl) < 2 || !strings.HasPrefix(bl[0], "<!-- gabo-version: ") || !strings.HasSuffix(bl[0], " -->") {
		t.Errorf("body first line = %q, want <!-- gabo-version: … -->", bl[0])
	}
	if len(bl) < 2 || !strings.HasPrefix(bl[1], "Skills: ") {
		t.Errorf("body second line does not start with Skills: :\n%s", body)
	}
	for _, want := range []string{"`.opencode/agents/<id>.md`", "no other key", "double quotes", "`mode: subagent`"} {
		if !strings.Contains(doc, want) {
			t.Errorf("%s: missing %q", runtimeOpenCode, want)
		}
	}
}

// The Tool Map has every spec capability with the deny keys of spec D3 and
// never an allow (S4, D3).
func TestRuntimeOpenCodeToolMap(t *testing.T) {
	t.Parallel()
	s := openCodeSection(t, runtimeOpenCodeDoc(t), "Tool Map")
	rows := map[string][]string{
		"read":            {"`read`"},
		"search":          {"`grep`", "`glob`"},
		"edit":            {"`edit`"},
		"shell":           {"`bash`"},
		"git":             {"`bash`"},
		"tracker":         {"`bash`"},
		"repository-host": {"`bash`"},
		"memory":          {"`<server>_*`"},
		"web":             {"`webfetch`", "`websearch`"},
		"skills":          {"`skill`"},
		"subagents":       {"`task`"},
	}
	for c, keys := range rows {
		row := ""
		for _, l := range strings.Split(s, "\n") {
			if strings.HasPrefix(l, "| `"+c+"`") || strings.Contains(l, ", `"+c+"`") && strings.HasPrefix(l, "| `") {
				row = l
				break
			}
		}
		if row == "" {
			t.Errorf("capability %s has no row in ## Tool Map", c)
			continue
		}
		for _, k := range keys {
			if !strings.Contains(row, k) {
				t.Errorf("Tool Map row for %s lacks deny key %s: %s", c, k, row)
			}
		}
	}
	if strings.Contains(s, "allow") {
		t.Errorf("## Tool Map mentions allow (D3 is deny-only)")
	}
	for _, want := range []string{"`deny`", "left at the owner's setting"} {
		if !strings.Contains(s, want) {
			t.Errorf("## Tool Map: missing %q", want)
		}
	}
}

// The reviewer stays user level: no skill link, every other agent denies
// the gabo-reviewer skill, and the script path is written literally (S6, D5).
func TestRuntimeOpenCodeReviewerRules(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	for _, want := range []string{
		"Never place `gabo-reviewer` in the project",
		"never create `.opencode/skills/gabo-reviewer`",
		"no reviewer link",
		"every agent but `gabo-reviewer`",
		"`gabo-reviewer: deny`",
		"`${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh`, written literally",
		"Never write under `~/.config/opencode`",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("%s: missing %q", runtimeOpenCode, want)
		}
	}
	if strings.Contains(doc, "reviewer_link:") {
		t.Errorf("%s: declares a reviewer_link", runtimeOpenCode)
	}
}

// The skill says how to fill every template slot.
func TestRuntimeOpenCodeFillsEverySlot(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	for _, s := range agentSlots {
		if !strings.Contains(doc, "`{{"+s+"}}`") {
			t.Errorf("%s: does not say how to fill {{%s}}", runtimeOpenCode, s)
		}
	}
}

// The link rules: each spec skill and each SKILL.md slot adapter is a
// relative symlink in .opencode/skills, the reviewer-configuration skill is
// linked like a spec skill and survives every rebuild, any other gabo-* link
// is removed, and gabo-reviewer is never linked (S5, S14, D11, D12).
func TestRuntimeOpenCodeLinkRules(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	steps := openCodeSection(t, doc, "Build Steps")
	for _, want := range []string{
		"make `.opencode/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`",
		"make `.opencode/skills/gabo-reviewer-config` a relative symlink to `../../.gabo/skills/gabo-reviewer-config`",
		"on every build, also when it already exists",
		"remove each `.opencode/skills/gabo-*` symlink that no spec skill or slot asks for any more, except `.opencode/skills/gabo-reviewer-config`",
		"never create `.opencode/skills/gabo-reviewer`",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	fixed := openCodeSection(t, doc, "Fixed-Name Links")
	for _, want := range []string{
		"whose adapter is a `<folder>/SKILL.md` gets one link `.opencode/skills/gabo-<slot>` (`_` written `-`)",
		"as a relative symlink",
	} {
		if !strings.Contains(fixed, want) {
			t.Errorf("## Fixed-Name Links: missing %q", want)
		}
	}
	for _, want := range []string{
		"Make every link of steps 2, 3 and 4 on every build",
		"`gabo-reviewer-config` link",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("%s: missing %q", runtimeOpenCode, want)
		}
	}
	if out := openCodeSection(t, doc, "Output Contract"); !strings.Contains(out, "the `gabo-reviewer-config` link") {
		t.Errorf("## Output Contract does not report the gabo-reviewer-config link")
	}
}

// Each agent names its skills by path in one line (S5, D4).
func TestRuntimeOpenCodeSkillsLine(t *testing.T) {
	t.Parallel()
	const want = "Skills: before work read .opencode/skills/<name>/SKILL.md for each of <spec skills, then the fixed-name links the body names>."
	doc := runtimeOpenCodeDoc(t)
	m := markdownFence.FindStringSubmatch(doc)
	if m == nil {
		t.Fatalf("%s: no ```markdown agent block", runtimeOpenCode)
	}
	var got string
	for _, l := range strings.Split(m[2], "\n") {
		if l = strings.TrimPrefix(l, m[1]); strings.HasPrefix(l, "Skills: ") {
			got = l
		}
	}
	if got != want {
		t.Errorf("Skills: line = %q, want %q", got, want)
	}
	if strings.Contains(doc, "gabo-reviewer-config/SKILL.md") {
		t.Errorf("%s: an agent's Skills: line names gabo-reviewer-config; it is for the main session (S14)", runtimeOpenCode)
	}
}
