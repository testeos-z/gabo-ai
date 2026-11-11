package skills

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The Pi runtime skill (T053, TRD D21, D24) declares where gabo check finds
// its agents and carries the verified Pi and gentle-pi facts it builds from.
func TestRuntimePiSkill(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), "gabo-runtime-pi/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)

	rest, _ := bytes.CutPrefix(b, []byte("---\n"))
	front, _, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		t.Fatal("gabo-runtime-pi/SKILL.md: no frontmatter")
	}
	var fm struct {
		Metadata struct {
			AgentDir       string `yaml:"agent_dir"`
			SkillDir       string `yaml:"skill_dir"`
			FormerSkillDir string `yaml:"former_skill_dir"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(front, &fm); err != nil {
		t.Fatalf("gabo-runtime-pi/SKILL.md frontmatter: %v", err)
	}
	if fm.Metadata.AgentDir != ".pi/agents" {
		t.Errorf("metadata.agent_dir = %q, want .pi/agents", fm.Metadata.AgentDir)
	}
	// intent 0008 S3, S4: links live in the shared .agents/skills; .pi/skills
	// is where the links were made before the move.
	if fm.Metadata.SkillDir != ".agents/skills" {
		t.Errorf("metadata.skill_dir = %q, want .agents/skills", fm.Metadata.SkillDir)
	}
	if fm.Metadata.FormerSkillDir != ".pi/skills" {
		t.Errorf("metadata.former_skill_dir = %q, want .pi/skills", fm.Metadata.FormerSkillDir)
	}

	for _, s := range []string{
		// prerequisite: a subagent package, checked read-only, never configured.
		"`pi list`", "`npm i -g gentle-pi`", "`pi install npm:gentle-pi`", "another subagent package",
		"Never edit", "`subagents.json`", "refuse",
		// refusal without a subagent package: no fallback, recommends gentle-pi and the full Gentle AI framework (S15).
		"Pi has no subagents, so gabo cannot work on Pi without a subagent package",
		"the full Gentle AI framework", "https://github.com/Gentleman-Programming/gentle-ai",
		"`gentle-ai install --agent pi`", "Then ask me to rebuild",
		// inputs read from the project copy.
		"`.gabo/skills/gabo-agent-templates/agents.yaml`", "`.gabo/VERSION`", "`runtimes.*`", "`slots.<slot>.adapter`",
		// agents: one file per spec id, version line in the frontmatter.
		"`.pi/agents/<id>.md`", "gabo-version: <version>", "`tools`",
		"subagent_run",
		// skills and fixed-name links in the shared .agents/skills (intent 0008 S3).
		"`.agents/skills/<skill>`", "`../../.gabo/skills/<skill>`", "`.agents/skills/gabo-methodology`", "`gabo-<slot>`",
		// reviewer stays at user level, linked into the active Pi agent home.
		"Never place `gabo-reviewer` in the project",
		"`GENTLE_PI_AGENT_HOME`", "`PI_CODING_AGENT_DIR`", "`~/.pi/agent`", "`~/.gentle-shell/agent`",
		"`${XDG_DATA_HOME:-~/.local/share}/gabo/skills/gabo-reviewer`",
		`${GENTLE_PI_AGENT_HOME:-${PI_CODING_AGENT_DIR:-$HOME/.pi/agent}}/skills/gabo-reviewer/scripts/reviewer.sh`,
		// probe and check.
		"probe agent", "`gabo check`",
		// sources and versions (former TRD 9.6 rows), and the confirmed git clone path.
		"https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/skills.md",
		"https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md",
		"Pi 1.4.0", "gentle-pi 4.5.1", "`<agent-home>/git/<host>/<path>`", "`.pi/git/<host>/<path>`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-runtime-pi/SKILL.md: missing %q", s)
		}
	}
	if strings.Contains(doc, "unverified") {
		t.Errorf("gabo-runtime-pi/SKILL.md: still calls a location unverified")
	}
}

// The Pi skill carries the narrowed engine-config rule (intent 0004 S8, D7),
// says `gabo init` writes every engine config key (intent 0010 S5) and keeps its no-install rule and the subagent package's own config off
// limits.
func TestRuntimePiNarrowsTheEngineConfigRule(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), "gabo-runtime-pi/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	assertEngineConfigRule(t, "gabo-runtime-pi/SKILL.md", doc,
		"Never edit Pi's or the subagent package's own configuration",
		"Pi needs no engine key today, so this skill writes no Pi config",
		"`gabo init` writes the project-level gabo integration keys (intent 0006)",
	)
	for _, s := range []string{
		"`gabo init` writes every engine config key (intent 0010 S5); this skill writes none, no Pi config included, and never runs `pi install`.",
		"Never run `pi install`, `pi config` or `npm i` yourself; the user installs.",
		"`subagents.json`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-runtime-pi/SKILL.md: missing %q", s)
		}
	}
}

// intent 0008 S2, D1.
func TestRuntimePiDeclaresAgentPlaces(t *testing.T) {
	t.Parallel()
	skills, name := runtimeAgentPlaces(t, "gabo-runtime-pi/SKILL.md")
	if skills != "line:Skills:" || name != "frontmatter:name" {
		t.Fatalf("agent_skills, agent_name = %q, %q; want line:Skills:, frontmatter:name", skills, name)
	}
}

// intent 0008 S3, S4: the Pi build places every link in the shared
// .agents/skills, names it in the Skills: line, and migrates the links it
// made in .pi/skills: it removes only its own symlinks and leaves and reports
// anything else.
func TestRuntimePiMovesToAgentsSkills(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), "gabo-runtime-pi/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)
	for _, s := range []string{
		"Skills: before work read `.agents/skills/<name>/SKILL.md` for each of",
		"relative to `.agents/skills/`",
		"`.agents/skills/gabo-<slot>/SKILL.md`",
		// migration of the former links (S4).
		"remove each `.pi/skills/gabo-*` symlink that this skill made",
		"one that points into `../../.gabo/skills/` or to a slot adapter folder of `gabo.yaml`",
		"Leave and report any other entry of `.pi/skills/`, and never remove a real folder or file.",
		"`gabo check` fails on such a leftover: `✗ .pi/skills/<name>: gabo link left after the move to .agents/skills`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-runtime-pi/SKILL.md: missing %q", s)
		}
	}
	for _, s := range []string{"`.pi/skills/<skill>`", "`.pi/skills/<name>/SKILL.md` for each", "`.pi/skills/gabo-methodology`"} {
		if strings.Contains(doc, s) {
			t.Errorf("gabo-runtime-pi/SKILL.md: still places links in .pi/skills: %q", s)
		}
	}
}

// intent 0008 S5: removing unneeded links in the shared directory keeps
// every link a declared engine needs, the assistant link included.
func TestRuntimePiKeepsSharedLinks(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), "gabo-runtime-pi/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	assertSharedRemovalRule(t, "gabo-runtime-pi/SKILL.md", string(b))
}

const runtimePi = "gabo-runtime-pi/SKILL.md"

func runtimePiDoc(t *testing.T) string {
	t.Helper()
	b, err := fs.ReadFile(Catalog(), runtimePi)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// piSection returns the body of the heading line head up to the next "## "
// or "### " heading.
func piSection(t *testing.T, doc, head string) string {
	t.Helper()
	_, s, ok := strings.Cut(doc, "\n"+head+"\n")
	if !ok {
		t.Fatalf("%s: no %s section", runtimePi, head)
	}
	s, _, _ = strings.Cut(s, "\n## ")
	s, _, _ = strings.Cut(s, "\n### ")
	return s
}

func mustContainIn(t *testing.T, where, doc string, want ...string) {
	t.Helper()
	for _, s := range want {
		if !strings.Contains(doc, s) {
			t.Errorf("%s %s: missing %q", runtimePi, where, s)
		}
	}
}

// intent 0009 S1, S3, D1, D6: when runtimes.assistant is pi, the build links
// gabo-assistant into the shared .agents/skills and writes the T1 block into
// the first existing file of the order the T5 probe confirmed.
func TestRuntimePiBuildsTheMainSessionEntry(t *testing.T) {
	t.Parallel()
	sec := piSection(t, runtimePiDoc(t), "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"intent 0009 S1, S3, S4, D1, D3",
		"the Main-Session Entry table of `gabo-agent-templates`",
		"When `pi` is listed in `runtimes.assistant` (one engine or a list, intent 0013 S4)",
		"make `.agents/skills/gabo-assistant` a relative symlink to `../../.gabo/skills/gabo-assistant`",
		"copy the `main_session.block` of `.gabo/skills/gabo-agent-templates/agents.yaml` verbatim, from `<!-- GABO_ASSISTANT_START -->` to `<!-- GABO_ASSISTANT_END -->`",
		"into the first existing file of `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md` at the repository root, in that order",
		"`AGENTS.md` created when none exists",
		"`.gabo/skills/gabo-assistant/SKILL.md`",
		"at most 2048 bytes",
		"Keep exactly one block",
	)
}

// intent 0009 S3: the same marker, size, idempotence and symlink rules as the
// Codex block (T6).
func TestRuntimePiWritesTheBlockInPlace(t *testing.T) {
	t.Parallel()
	sec := piSection(t, runtimePiDoc(t), "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"replace only the bytes between the markers",
		"append it after one blank line when absent",
		"keep every other byte, so a second build leaves the file byte-identical",
		"A file whose markers are unpaired: change nothing and report it.",
		"write through a symlinked file only when its target stays inside the repository",
	)
}

// intent 0009 S1, S4: when runtimes.assistant is not pi, the build removes its
// block and the shared link only under the shared rule of T6, so a Pi coder
// keeps the Codex assistant's link.
func TestRuntimePiRemovesItsEntryWhenNotAssistant(t *testing.T) {
	t.Parallel()
	sec := piSection(t, runtimePiDoc(t), "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"When `pi` is not listed in `runtimes.assistant`, remove its block from `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`",
		"a block is removed only when no engine listed in `runtimes.assistant` uses its file (Codex uses the same files)",
		"removes the shared link only when no engine listed in `runtimes.assistant` uses `.agents/skills`",
		"With `runtimes.assistant: codex` and `runtimes.coder: pi`, a Pi rebuild keeps `.agents/skills/gabo-assistant`",
	)
}

// intent 0009 S4: on every build the skill removes the gabo entry of every
// other full-tier engine of the T1 table that is not runtimes.assistant,
// recognized only by its gabo identity.
func TestRuntimePiRemovesOtherEnginesEntries(t *testing.T) {
	t.Parallel()
	sec := piSection(t, runtimePiDoc(t), "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"On every build, remove the gabo entry of every other full-tier engine in the Main-Session Entry table of `gabo-agent-templates` that is not listed in `runtimes.assistant`",
		"except any link or block a listed engine's row writes",
		"recognized only by its gabo identity",
		"`.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"the `.claude/skills/gabo-assistant` symlink to the catalog folder",
		"`.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"for Codex, its block in `AGENTS.override.md` and `AGENTS.md`",
		"keep and report it",
	)
}

// intent 0013 S4: a Pi build never removes a listed engine's entry, so
// rebuilding Claude Code and Pi in either order leaves both entries; Codex
// and Pi listed together share one block in the file Codex writes.
func TestRuntimePiKeepsEveryListedEntry(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	sec := piSection(t, doc, "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"A build never removes the entry of another engine listed in `runtimes.assistant`, so rebuilding the listed engines in any order leaves every listed entry in place",
		"With `runtimes.assistant: [claude, pi]`, a Pi build keeps `.claude/agents/gabo-assistant.md` and the `.claude/skills/gabo-assistant` link",
		"When `codex` is listed too, write the block into `AGENTS.override.md` when it exists, else `AGENTS.md`, created when neither exists",
		"the one block serves both",
		"neither build removes the other's block",
	)
	mustLackIn(t, runtimePi+" ### Main-session entry", sec,
		"When `runtimes.assistant` is `pi`",
		"When `runtimes.assistant` is not `pi`",
		"is runtimes.assistant",
	)
	mustContainIn(t, "## Build", doc,
		"with a list, while any engine listed in `runtimes.assistant` uses it (intent 0013 S4)",
	)
}

// intent 0013 S6: every Pi agent names pi in its frontmatter, right after
// its version line, and the body loses both lines.
func TestRuntimePiFillsTheEngineSlot(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mustContainIn(t, "## Build", doc,
		"gabo-version: <version>\ngabo-engine: pi\n",
		"| `{{"+engineSlot+"}}` | `pi`",
		"`gabo-engine: pi` is the template's engine line",
		"<template body after its first two lines `gabo-version: {{gabo_version}}` and `gabo-engine: {{engine}}` and the blank line that follows, slots filled>",
	)
	mustLackIn(t, runtimePi, doc, "after its first line `gabo-version: {{gabo_version}}` and the blank line that follows")
}

// intent 0010 S4, S5, D3 A: gabo init writes and removes every engine config
// key, so the Pi skill no longer removes Claude Code's `agent` or OpenCode's
// `default_agent`; it keeps removing other engines' agent files, links and
// blocks and keeps writing its own block.
func TestRuntimePiLeavesEngineConfigKeysToGaboInit(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	sec := piSection(t, doc, "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"Never remove Claude Code's `agent` key or OpenCode's `default_agent` key: `gabo init` removes them (intent 0010 S4)",
		"copy the `main_session.block` of `.gabo/skills/gabo-agent-templates/agents.yaml` verbatim",
	)
	mustLackIn(t, runtimePi+" ### Main-session entry", sec,
		"the key `agent` in `.claude/settings.json` only when its value is exactly `gabo-assistant`",
		"the key `default_agent` only when its value is exactly `gabo-assistant`",
		"For a key, remove only that member",
		"a user's own `agent` or `default_agent` value",
	)
	assertKeyLinesNameGaboInit(t, runtimePi, doc)
}

// intent 0009 S7, D3, D6: the write list names the entry, the build runs it,
// the T5 probe is a source and the output reports the entry.
func TestRuntimePiNamesTheMainSessionEntry(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mustContainIn(t, "## Hard Rules", piSection(t, doc, "## Hard Rules"),
		"the `.agents/skills/gabo-assistant` link and the scoped block of the Main-session entry",
		"the main-session entries of other engines that section removes",
	)
	mustContainIn(t, "## Sources", piSection(t, doc, "## Sources"),
		"Pi 1.4.0 loads only the first existing file per directory",
		"`AGENTS.override.md`, then `AGENTS.md`, then `CLAUDE.md`",
		"`dist/core/resource-loader.js`",
		"https://github.com/testeos-z/gabo-ai/issues/324#issuecomment-6690677042",
	)
	mustContainIn(t, "## Output Contract", piSection(t, doc, "## Output Contract"),
		"the main-session entry",
		"each other engine's entry removed or kept",
	)
}

// gabo check reads the main-session entry of Pi from its runtime skill (spec
// 0009 D5): the scoped block in the order T5 confirmed.
func TestRuntimePiDeclaresMainSession(t *testing.T) {
	t.Parallel()
	wantMainSession(t, runtimePi, mainSessionMeta{Kind: "block", Files: []string{"AGENTS.override.md", "AGENTS.md", "CLAUDE.md"}})
}

// piCapabilityRow returns the Pi tool cell of the capability table row whose
// first cell is cell.
func piCapabilityRow(t *testing.T, doc, cell string) string {
	t.Helper()
	_, row, ok := strings.Cut(doc, "\n| "+cell+" | ")
	if !ok {
		t.Fatalf("%s: no capability row %s", runtimePi, cell)
	}
	row, _, _ = strings.Cut(row, "\n")
	return row
}

// intent 0013 S17: gentle-pi starts a subagent as `pi --tools <its tools>`,
// which declares no tool it does not name, so the memory row lists the four
// Engram tools the memory adapter calls: plain names from gentle-engram, the
// mcp__<server>__ form from a Pi MCP server, else omitted and reported.
func TestRuntimePiListsTheMemoryToolsExplicitly(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	row := piCapabilityRow(t, doc, "`memory`")
	mustContainIn(t, "memory row", row,
		"`bash`, `mem_search`, `mem_save`, `mem_update`, `mem_get_observation` when `pi list` shows `npm:gentle-engram`",
		"`mcp__<server>__mem_search`, `mcp__<server>__mem_save`, `mcp__<server>__mem_update`, `mcp__<server>__mem_get_observation` when Pi's MCP config has an Engram server named `<server>`",
		"neither: `bash` only, omit the memory tools and report it",
	)
	mustLackIn(t, runtimePi, doc, "need no entry", "Pi keeps MCP tools whatever `--tools` lists")
	mustContainIn(t, "## Build", doc,
		"gentle-pi starts a subagent as `pi --tools <its tools list>`",
		"every tool the list does not name is not declared to the subagent, MCP and extension tools included",
	)
}

// intent 0015 S7 d: the `subagents` capability is gone (no spec agent has it),
// and the sentence on subagents getting no `subagent_*` tools stays with
// `agent_tools_unavailable`.
func TestRuntimePiHasNoSubagentsCapability(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mustLackIn(t, runtimePi, doc, "| `subagents` |", "    subagents: []")
	mustContainIn(t, "## Build", doc,
		"`metadata.agent_tools_unavailable` the tools a subagent never has",
		"every `.pi/agents` file runs as a gentle-pi subagent",
	)
}

// intent 0013 S17, D11: the frontmatter declares where a Pi agent file lists
// its tools and the tools each spec capability requires, for gabo check (T19).
func TestRuntimePiDeclaresToolRequirements(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), runtimePi)
	if err != nil {
		t.Fatal(err)
	}
	rest, _ := bytes.CutPrefix(b, []byte("---\n"))
	front, _, ok := bytes.Cut(rest, []byte("\n---\n"))
	if !ok {
		t.Fatalf("%s: no frontmatter", runtimePi)
	}
	var fm struct {
		Metadata struct {
			AgentTools       string              `yaml:"agent_tools"`
			ToolRequirements map[string][]string `yaml:"agent_tool_requirements"`
			Unavailable      []string            `yaml:"agent_tools_unavailable"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(front, &fm); err != nil {
		t.Fatalf("%s frontmatter: %v", runtimePi, err)
	}
	if fm.Metadata.AgentTools != "frontmatter:tools" {
		t.Errorf("metadata.agent_tools = %q, want frontmatter:tools", fm.Metadata.AgentTools)
	}
	want := map[string][]string{
		"read":            {"read"},
		"skills":          {"read"},
		"search":          {"grep", "find", "ls"},
		"edit":            {"edit", "write"},
		"shell":           {"bash"},
		"git":             {"bash"},
		"tracker":         {"bash"},
		"repository-host": {"bash"},
		"memory":          {"mem_search", "mem_save", "mem_update", "mem_get_observation"},
		"web":             {},
	}
	// intent 0013 S20 (plan T22): every .pi/agents file runs as a gentle-pi
	// subagent, which never has a subagent_* tool.
	if strings.Join(fm.Metadata.Unavailable, ",") != "subagent_*" {
		t.Errorf("metadata.agent_tools_unavailable = %q, want [subagent_*]", fm.Metadata.Unavailable)
	}
	got := fm.Metadata.ToolRequirements
	if len(got) != len(want) {
		t.Errorf("metadata.agent_tool_requirements has %d capabilities, want %d: %v", len(got), len(want), got)
	}
	for c, tools := range want {
		g, ok := got[c]
		if !ok {
			t.Errorf("metadata.agent_tool_requirements: no %s", c)
			continue
		}
		if strings.Join(g, ",") != strings.Join(tools, ",") {
			t.Errorf("metadata.agent_tool_requirements.%s = %v, want %v", c, g, tools)
		}
	}
}

// intent 0013 S17: the sources of the tool facts.
func TestRuntimePiNamesTheToolSources(t *testing.T) {
	t.Parallel()
	mustContainIn(t, "## Sources", piSection(t, runtimePiDoc(t), "## Sources"),
		"gentle-engram 0.4.2 `index.ts` `ENGRAM_TOOLS`",
		"Pi 1.4.0 `docs/cli.md` (Tools)",
		"gentle-pi 4.5.1 `lib/agents-runner.ts` `childArguments`",
		"`extensions/gentle-agents.ts`",
	)
}

// Intent 0015 S1, D7 (plan T9): the Pi-only wave rule is gone because the main
// session runs the wave on every engine.
func TestRuntimePiHasNoPiOnlyWaveRule(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mustLackIn(t, runtimePi, doc,
		"A wave started from a Pi main session runs in that main session",
		"Build the `gabo-cycle` agent anyway",
		"loads the `gabo-cycle` skill itself instead of launching",
		"Build `gabo-reviewer.md` anyway",
	)
}

// Owner order 2026-11-11 (plan T9): on Pi the main session launches each coder
// as one first-level `subagent_run` of `gabo-coder` and resumes it with
// `subagent_continue`; the run id is the handle. No depth concept.
func TestRuntimePiSaysHowTheMainSessionLaunchesAndResumesCoders(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	sec := piSection(t, doc, "## Launching Agents")
	mustContainIn(t, "## Launching Agents", sec,
		"the main session launches each coder as one first-level `subagent_run` of `gabo-coder`",
		"resumes it with `subagent_continue`",
		"the run id is the handle",
	)
	mustLackIn(t, runtimePi+" ## Launching Agents", sec, "depth", "nest")
}

// intent 0015 S6, D3: every main_session.skills entry is linked while pi is
// listed, and goes with the entry when pi is not listed.
func TestRuntimePiLinksEveryMainSessionSkill(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	sec := piSection(t, doc, "### Main-session entry")
	mustContainIn(t, "### Main-session entry", sec,
		"every other `main_session.skills` entry",
		"`.agents/skills/<name>`", "`../../.gabo/skills/<name>`",
		"`gabo-assistant`, `gabo-cycle` and `gabo-wave-report`",
		"the `.agents/skills/<name>` link of every other `main_session.skills` entry",
		"while another spec agent's `skills` lists it",
	)
	mustContainIn(t, "## Build", doc,
		"also keep every `main_session.skills` link while `runtimes.assistant` names an engine that uses `.agents/skills`")
}

// intent 0015 S6, D4: the two former agent files go on every build, only with
// a gabo-version line.
func TestRuntimePiRemovesTheFormerAgentFiles(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mustContainIn(t, "", doc,
		"`.pi/agents/gabo-cycle.md` and `.pi/agents/gabo-reviewer.md`",
		"only when the file carries a `gabo-version:` line",
		"a file without one: leave it and report it",
	)
}

// reviewerPatterns are the two request patterns the coder validates, pinned
// as literals (they were compared with the removed gabo-reviewer template).
func reviewerPatterns(t *testing.T) []string {
	t.Helper()
	return []string{
		`^PR #([1-9][0-9]*) head ([0-9a-f]{40})( ready for review)?$`,
		`^PR #([1-9][0-9]*) head ([0-9a-f]{40}) engine (claude|codex|pi|opencode)$`,
	}
}

// Intent 0015 D5 (plan T5, supersedes intent 0013 D15): the Pi coder body is
// the template, so the Build no longer rewrites the coder's review bullet;
// the {{reviewer_script}} row of the Slots table stays.
func TestRuntimePiDoesNotRewriteTheCoderReviewBullet(t *testing.T) {
	t.Parallel()
	doc := runtimePiDoc(t)
	mustLackIn(t, runtimePi, doc, "### Coder review", "in place of the template's bullet", "The only agent you launch is")
	mustContainIn(t, "Slots table", doc, "`{{reviewer_script}}`", "the Pi coder body is the template", "filled in `gabo-coder`, the only template that carries the slot")
}
