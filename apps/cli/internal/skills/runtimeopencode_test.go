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
	want := []string{"Activation Contract", "Hard Rules", "Prerequisites", "Sources", "Tool Map", "Fixed-Name Links", "Build Steps", "Output Contract"}
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

// numberedItems returns the top-level "N. " items of a section, in order.
func numberedItems(s string) []string {
	var items []string
	for _, l := range strings.Split(s, "\n") {
		if len(l) > 2 && l[0] >= '1' && l[0] <= '9' && strings.HasPrefix(l[1:], ". ") {
			items = append(items, l)
		}
	}
	return items
}

// The build first checks opencode --version, before anything is built or
// written; build step 1 runs the prerequisites (S8).
func TestRuntimeOpenCodeChecksOpenCodeFirst(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	pre := numberedItems(openCodeSection(t, doc, "Prerequisites"))
	if len(pre) == 0 || !strings.Contains(pre[0], "`opencode --version` must succeed") {
		t.Fatalf("## Prerequisites: first item does not check `opencode --version`: %v", pre)
	}
	steps := numberedItems(openCodeSection(t, doc, "Build Steps"))
	if len(steps) == 0 || !strings.HasPrefix(steps[0], "1. Run the Prerequisites") {
		t.Errorf("## Build Steps: step 1 does not run the Prerequisites first: %v", steps)
	}
}

// The depth rule writes only experimental.subagent_depth: 3, at project level,
// into the first existing config of OpenCode's lookup order (the order gabo
// check reads), else a new .opencode/opencode.json; it merges, keeps 3 or
// more, reports the change, and refuses on a missing opencode or an
// unmergeable file (S8, D7).
func TestRuntimeOpenCodeSubagentDepthRule(t *testing.T) {
	t.Parallel()
	pre := openCodeSection(t, runtimeOpenCodeDoc(t), "Prerequisites")
	for _, want := range []string{
		"the first existing of `opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `.opencode/opencode.jsonc` (OpenCode's own lookup order, the same one `gabo check` reads), else a new `.opencode/opencode.json`",
		"Write only the key `experimental.subagent_depth`, with the value `3`",
		"`{\"experimental\": {\"subagent_depth\": 3}}`",
		"keeping every other key and value",
		"the comments of a `.jsonc` file",
		"a whole number of 3 or more: leave the file unchanged",
		"Never write `~/.config/opencode/` or `.opencode/package.json`",
		"Report the file, the key, the old value (or `unset`) and the new value",
		"build nothing, write nothing and print `set experimental.subagent_depth to 3 in <file>`",
		"`opencode` is missing",
		"cannot be parsed and merged without loss",
	} {
		if !strings.Contains(pre, want) {
			t.Errorf("## Prerequisites: missing %q", want)
		}
	}
}

// The write list of S9 is the only one: the skill lists what it writes once,
// never changes another project config key and collides with no agent (S9, D8).
func TestRuntimeOpenCodeWriteList(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	const list = "Write only `.opencode/agents/<id>.md` for spec ids, `.opencode/skills/` links, the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js`, the probe files you delete, and the project-level key `experimental.subagent_depth` of the Prerequisites, merged into one project config file."
	rules := openCodeSection(t, doc, "Hard Rules")
	if !strings.Contains(rules, list) {
		t.Errorf("## Hard Rules: missing the write list %q", list)
	}
	if n := strings.Count(doc, "Write only `"); n != 1 {
		t.Errorf("%s: %d write lists, want exactly the one of S9", runtimeOpenCode, n)
	}
	for _, want := range []string{
		"never change any other key of a project config",
		"`gentle-orchestrator`",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("## Hard Rules: missing %q", want)
		}
	}
}

var textFence = regexp.MustCompile("(?ms)^( *)```text\n(.*?)\n *```$")

// The built gabo-coder body gains an OpenCode-only first step that moves the
// session to the launch-prompt worktree, else uses shell workdir and
// absolute paths (S7, D6).
func TestRuntimeOpenCodeCoderWorktreeStep(t *testing.T) {
	t.Parallel()
	steps := openCodeSection(t, runtimeOpenCodeDoc(t), "Build Steps")
	if !strings.Contains(steps, "For `gabo-coder` only, write this paragraph between the `Skills:` line and `<filled body>`") {
		t.Errorf("## Build Steps: does not place the coder worktree step in the gabo-coder body")
	}
	m := textFence.FindStringSubmatch(steps)
	if m == nil {
		t.Fatalf("## Build Steps: no ```text coder worktree step")
	}
	for _, want := range []string{
		"First step, before any other tool call",
		"`opencode.session_move` with the worktree path from your launch prompt",
		"not callable",
		"pass that worktree path as `workdir` on every `shell` call",
		"absolute paths",
		"`git rev-parse --show-toplevel`",
	} {
		if !strings.Contains(m[2], want) {
			t.Errorf("coder worktree step: missing %q:\n%s", want, m[2])
		}
	}
}

// The binary facts behind the depth rule and the worktree step are cited (S2).
func TestRuntimeOpenCodeCitesPrerequisiteFacts(t *testing.T) {
	t.Parallel()
	s := openCodeSection(t, runtimeOpenCodeDoc(t), "Sources")
	for _, want := range []string{"`experimental.subagent_depth`", "`opencode.session_move`", "`workdir`"} {
		found := false
		for _, l := range strings.Split(s, "\n") {
			if strings.Contains(l, want) && strings.Contains(l, "(binary)") {
				found = true
			}
		}
		if !found {
			t.Errorf("## Sources: no (binary) row for %s", want)
		}
	}
}
