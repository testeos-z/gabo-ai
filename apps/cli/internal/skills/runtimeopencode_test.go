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
	// intent 0008 S3, S4: links live in the shared .agents/skills;
	// .opencode/skills is where the links were made before the move.
	if got := fm.Metadata["skill_dir"]; got != ".agents/skills" {
		t.Errorf("metadata.skill_dir = %v, want .agents/skills", got)
	}
	if got := fm.Metadata["former_skill_dir"]; got != ".opencode/skills" {
		t.Errorf("metadata.former_skill_dir = %v, want .opencode/skills", got)
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
	if len(bl) < 3 || bl[1] != "<!-- gabo-engine: opencode -->" {
		t.Errorf("body second line = %q, want <!-- gabo-engine: opencode --> (intent 0013 S6)", bl[1])
	}
	if len(bl) < 3 || !strings.HasPrefix(bl[2], "Skills: ") {
		t.Errorf("body third line does not start with Skills: :\n%s", body)
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
		"never create `.agents/skills/gabo-reviewer`",
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
// relative symlink in the shared .agents/skills (intent 0008 S3), the
// reviewer-configuration skill is linked like a spec skill and survives every
// rebuild, any other gabo-* link this skill made is removed under the
// shared-directory rule, and gabo-reviewer is never linked (S5, S14, D11, D12).
func TestRuntimeOpenCodeLinkRules(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	steps := openCodeSection(t, doc, "Build Steps")
	for _, want := range []string{
		"make `.agents/skills/<name>` a relative symlink to `../../.gabo/skills/<name>`",
		"make `.agents/skills/gabo-reviewer-config` a relative symlink to `../../.gabo/skills/gabo-reviewer-config`",
		"on every build, also when it already exists",
		"remove each `.agents/skills/gabo-*` symlink that no spec skill or slot asks for any more and that this skill made",
		"except `.agents/skills/gabo-reviewer-config`",
		"never create `.agents/skills/gabo-reviewer`",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	fixed := openCodeSection(t, doc, "Fixed-Name Links")
	for _, want := range []string{
		"whose adapter is a `<folder>/SKILL.md` gets one link `.agents/skills/gabo-<slot>` (`_` written `-`)",
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
	const want = "Skills: before work read .agents/skills/<name>/SKILL.md for each of <spec skills, then the fixed-name links the body names>."
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

// intent 0010 S3, S5: gabo init writes experimental.subagent_depth, so the
// Prerequisites keep only the opencode check; the skill reads the project
// config in OpenCode's lookup order (the order gabo check reads) and
// reports `run gabo init` when a key is missing or low.
func TestRuntimeOpenCodeReadsTheEngineConfigKeys(t *testing.T) {
	t.Parallel()
	pre := openCodeSection(t, runtimeOpenCodeDoc(t), "Prerequisites")
	if n := len(numberedItems(pre)); n != 1 {
		t.Errorf("## Prerequisites: %d numbered steps, want only the opencode check", n)
	}
	mustContainIn(t, "## Prerequisites", pre,
		"the first existing of `opencode.json`, `opencode.jsonc`, `.opencode/opencode.json`, `.opencode/opencode.jsonc` (OpenCode's own lookup order, the same one `gabo check` reads)",
		"Call it `<file>`",
		"`gabo init` writes `experimental.subagent_depth` (at least 3) and `default_agent` into `<file>` (intent 0010 S3, S4); this skill only reads them and writes no key",
		"report `run gabo init`",
		"below 3",
		"Never write `~/.config/opencode/` or `.opencode/package.json`",
	)
	mustLackIn(t, "## Prerequisites", pre,
		"Write only the key `experimental.subagent_depth`, with the value `3`",
		"`{\"experimental\": {\"subagent_depth\": 3}}`",
		"print `set experimental.subagent_depth to 3 in <file>`",
		"merge: edit the text in place",
		"else a new `.opencode/opencode.json`",
	)
}

// The write list of S9 is the only one: the skill lists what it writes once,
// never changes another project config key but the one main-session key of
// intent 0009 S7, and collides with no agent (S9, D8).
func TestRuntimeOpenCodeWriteList(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	const list = "Write only `.opencode/agents/<id>.md` for spec ids and `.opencode/agents/gabo-assistant.md`, `.agents/skills/` links, the stage-gate plugin `.opencode/plugins/gabo-stage-gate.js`, the probe files you delete and the removals of other engines' main-session entries of build step 9."
	rules := openCodeSection(t, doc, "Hard Rules")
	if !strings.Contains(rules, list) {
		t.Errorf("## Hard Rules: missing the write list %q", list)
	}
	if n := strings.Count(doc, "Write only `"); n != 1 {
		t.Errorf("%s: %d write lists, want exactly the one of S9", runtimeOpenCode, n)
	}
	for _, want := range []string{
		"never write or remove a key of a project config",
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

// intent 0008 S2, D1: OpenCode names an agent by its file name, so there is
// no agent_name.
func TestRuntimeOpenCodeDeclaresAgentPlaces(t *testing.T) {
	t.Parallel()
	skills, name := runtimeAgentPlaces(t, "gabo-runtime-opencode/SKILL.md")
	if skills != "line:Skills:" || name != "" {
		t.Fatalf("agent_skills, agent_name = %q, %q; want line:Skills: and none", skills, name)
	}
}

// intent 0008 S3, S4: the OpenCode build places every link in the shared
// .agents/skills, names it in the Skills: line, and migrates the links it
// made in .opencode/skills: it removes only its own symlinks and leaves and
// reports anything else.
func TestRuntimeOpenCodeMovesToAgentsSkills(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	steps := openCodeSection(t, doc, "Build Steps")
	for _, s := range []string{
		"remove each `.opencode/skills/gabo-*` symlink that this skill made",
		"one that points into `../../.gabo/skills/` or to a slot adapter folder of `gabo.yaml`",
		"`.opencode/skills/gabo-reviewer-config` included",
		"Leave and report any other entry of `.opencode/skills/`, and never remove a real folder or file.",
		"`gabo check` fails on such a leftover: `✗ .opencode/skills/<name>: gabo link left after the move to .agents/skills`",
	} {
		if !strings.Contains(steps, s) {
			t.Errorf("## Build Steps: missing %q", s)
		}
	}
	if out := openCodeSection(t, doc, "Output Contract"); !strings.Contains(out, "each `.opencode/skills/` link removed and each entry left with why") {
		t.Errorf("## Output Contract does not report the migration")
	}
	for _, s := range []string{"`.opencode/skills/<name>`", "read .opencode/skills/<name>/SKILL.md", "`.opencode/skills/gabo-<slot>`", "`.opencode/skills/` links"} {
		if strings.Contains(doc, s) {
			t.Errorf("%s: still places links in .opencode/skills: %q", runtimeOpenCode, s)
		}
	}
}

// intent 0008 S5: removing unneeded links in the shared directory keeps
// every link a declared engine needs, the assistant link included.
func TestRuntimeOpenCodeKeepsSharedLinks(t *testing.T) {
	t.Parallel()
	assertSharedRemovalRule(t, runtimeOpenCode, runtimeOpenCodeDoc(t))
}

// intent 0008 S5, D6: the Sources cite the T9 probe of OpenCode with the same
// skill in .claude/skills and .agents/skills.
func TestRuntimeOpenCodeCitesTheDuplicateProbe(t *testing.T) {
	t.Parallel()
	s := openCodeSection(t, runtimeOpenCodeDoc(t), "Sources")
	for _, want := range []string{
		"https://github.com/testeos-z/gabo-ai/issues/315#issuecomment-6689595417",
		"lists it once",
		"`.claude/skills` before `.agents/skills`",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("## Sources: missing %q", want)
		}
	}
}

// intent 0009 S1, S2, S4, D3: when runtimes.assistant is opencode, the build
// links gabo-assistant into the shared .agents/skills, writes the primary
// agent .opencode/agents/gabo-assistant.md from the T1 template and merges
// reports `run gabo init` when default_agent is missing, which gabo init
// writes (intent 0010 S4); it never merges the key itself.
func TestRuntimeOpenCodeBuildsTheMainSessionEntry(t *testing.T) {
	t.Parallel()
	steps := openCodeSection(t, runtimeOpenCodeDoc(t), "Build Steps")
	for _, want := range []string{
		"Main-session entry (intent 0009 S1, S2, S4, D3)",
		"the Main-Session Entry table of `gabo-agent-templates`",
		"When `opencode` is listed in `runtimes.assistant`",
		"make `.agents/skills/gabo-assistant` a relative symlink to `../../.gabo/skills/gabo-assistant`",
		"write `.opencode/agents/gabo-assistant.md` from `main_session.template`",
		"`mode: primary`",
		"`gabo init` writes `default_agent` with the value `\"gabo-assistant\"` into `<file>` (intent 0010 S4)",
		"when it is missing, report `run gabo init`",
		"A `default_agent` with another value is the owner's own: leave it and report it",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	mustLackIn(t, "## Build Steps", steps,
		"merge only the key `default_agent`",
		"naming `default_agent`",
	)
}

// intent 0009 S1, S4: when runtimes.assistant is not opencode, the build
// removes its own entry, and the shared link only under the shared rule.
func TestRuntimeOpenCodeRemovesItsEntryWhenNotAssistant(t *testing.T) {
	t.Parallel()
	steps := openCodeSection(t, runtimeOpenCodeDoc(t), "Build Steps")
	for _, want := range []string{
		"When `opencode` is not listed in `runtimes.assistant`",
		"remove `.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"removes the shared `.agents/skills/gabo-assistant` link only when every declared engine using `.agents/skills` is not listed in `runtimes.assistant`",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	mustLackIn(t, "## Build Steps", steps,
		"remove the key `default_agent` from `<file>`",
	)
}

// intent 0009 S4: on every build the skill removes the gabo entry of every
// other full-tier engine of the T1 table, recognized only by its gabo
// identity, and never the entry of the engine in runtimes.assistant.
func TestRuntimeOpenCodeRemovesOtherEnginesEntries(t *testing.T) {
	t.Parallel()
	steps := openCodeSection(t, runtimeOpenCodeDoc(t), "Build Steps")
	for _, want := range []string{
		"On every build, remove the gabo main-session entry of every other full-tier engine of that table that is not listed in `runtimes.assistant`",
		"recognized only by its gabo identity",
		"`.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"the `.claude/skills/gabo-assistant` symlink to the catalog folder",
		"Never remove Claude Code's `agent` key or OpenCode's `default_agent` key: `gabo init` removes them (intent 0010 S4)",
		"the bytes from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`",
		"`AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`",
		"keep a block in a file the entry of an engine listed in `runtimes.assistant` uses",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	mustLackIn(t, "## Build Steps", steps,
		"the key `agent` in `.claude/settings.json` only when its value is exactly `gabo-assistant`",
		"a key with another value",
	)
}

// intent 0009 S7, D3 and intent 0010 S5: the build writes the files of the
// entry and no key; the only primary agent it writes is gabo-assistant.
func TestRuntimeOpenCodeAllowsTheMainSessionKey(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	rules := openCodeSection(t, doc, "Hard Rules")
	for _, want := range []string{
		"`.opencode/agents/gabo-assistant.md`",
		"Every spec agent is `mode: subagent`",
		"the only primary agent a build writes is `gabo-assistant`, and only when `opencode` is listed in `runtimes.assistant`",
	} {
		if !strings.Contains(rules, want) {
			t.Errorf("## Hard Rules: missing %q", want)
		}
	}
	if strings.Contains(doc, "no build sets a default agent") {
		t.Errorf("%s: still says no build sets a default agent", runtimeOpenCode)
	}
	src := openCodeSection(t, doc, "Sources")
	for _, want := range []string{"https://opencode.ai/docs/config", "`default_agent`", "must be a primary agent"} {
		if !strings.Contains(src, want) {
			t.Errorf("## Sources: missing %q", want)
		}
	}
	out := openCodeSection(t, doc, "Output Contract")
	for _, want := range []string{"the main-session entry", "`default_agent`", "each other engine's entry removed"} {
		if !strings.Contains(out, want) {
			t.Errorf("## Output Contract: missing %q", want)
		}
	}
}

// intent 0010 S5: the engine-config rule says gabo init writes every engine
// config key and the skill writes none; every line naming a key says so.
func TestRuntimeOpenCodeLeavesEngineConfigKeysToGaboInit(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	assertEngineConfigRule(t, runtimeOpenCode, doc)
	mustContainIn(t, "## Hard Rules", openCodeSection(t, doc, "Hard Rules"),
		"`gabo init` writes every engine config key (intent 0010 S5), `experimental.subagent_depth` and `default_agent` included; this skill writes none",
		"the stage-gate plugin",
	)
	assertKeyLinesNameGaboInit(t, runtimeOpenCode, skillBody(t, doc))
}

// gabo check reads the main-session entry of OpenCode from its runtime skill
// (spec 0009 D5): default_agent in the project config, in OpenCode's lookup
// order.
func TestRuntimeOpencodeDeclaresMainSession(t *testing.T) {
	t.Parallel()
	wantMainSession(t, runtimeOpenCode, mainSessionMeta{Kind: "agent",
		Config: []string{"opencode.json", "opencode.jsonc", ".opencode/opencode.json", ".opencode/opencode.jsonc"}, Key: "default_agent"})
}

// intent 0013 S6: the skill fills {{engine}} with `opencode`, as written in
// gabo.yaml, and writes it as an HTML comment after the version comment of
// every agent file, the main-session agent included.
func TestRuntimeOpenCodeFillsTheEngineSlot(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	steps := openCodeSection(t, doc, "Build Steps")
	mustContainIn(t, "## Build Steps", steps,
		"| `{{"+engineSlot+"}}` | `opencode`, the engine name as written in `gabo.yaml` |",
		"`<!-- gabo-engine: opencode -->` is the template's engine line moved into an HTML comment",
		"`<filled body>` is the filled template without its first two lines `gabo-version: …` and `gabo-engine: …` and the blank line after them",
		"the body is the template filled with `{{gabo_version}}` and `{{engine}}`",
	)
	mustLackIn(t, runtimeOpenCode, doc, "without its first line `gabo-version: …`")
}

// intent 0013 S4: runtimes.assistant may list several engines; the skill
// builds the OpenCode entry when opencode is listed, removes another
// full-tier engine's entry only when that engine is not listed, and keeps the
// shared link while any listed engine uses .agents/skills.
func TestRuntimeOpenCodeKeepsEveryListedEntry(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	mustContainIn(t, "## Build Steps", openCodeSection(t, doc, "Build Steps"),
		"`runtimes.assistant` is one engine or a list (intent 0013 S4)",
		"A build removes another full-tier engine's entry only when it is not listed in `runtimes.assistant`, never the entry of another listed engine",
		"it is removed only when every engine that uses `.agents/skills` is not listed in `runtimes.assistant` (intent 0013 S4)",
	)
	mustLackIn(t, runtimeOpenCode, doc,
		"`runtimes.assistant` is `opencode`", "`runtimes.assistant` is not `opencode`",
		"that is not `runtimes.assistant`", "the engine in `runtimes.assistant`",
		"is `runtimes.assistant`",
	)
	assertRemovalRulesSayNotListed(t, runtimeOpenCode, skillBody(t, doc))
}
