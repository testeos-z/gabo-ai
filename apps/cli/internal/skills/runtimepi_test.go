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
// says `gabo init` writes the project-level keys (intent 0006 S13) and keeps its no-install rule and the subagent package's own config off
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
	)
	for _, s := range []string{
		"`gabo init` writes the project-level gabo integration keys (intent 0006); this skill writes no Pi config, and never runs `pi install`.",
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
		"When `runtimes.assistant` is `pi`",
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
		"When `runtimes.assistant` is not `pi`, remove its block from `AGENTS.override.md`, `AGENTS.md` and `CLAUDE.md`",
		"except from the file the row of the `runtimes.assistant` engine writes its own block to (Codex uses the same files)",
		"removes the shared link only when no declared engine using .agents/skills is runtimes.assistant",
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
		"On every build, remove the gabo entry of every other full-tier engine in the Main-Session Entry table of `gabo-agent-templates` that is not `runtimes.assistant`",
		"recognized only by its gabo identity",
		"`.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"the `.claude/skills/gabo-assistant` symlink to the catalog folder",
		"the key `agent` in `.claude/settings.json` only when its value is exactly `gabo-assistant`",
		"`.opencode/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"the key `default_agent` only when its value is exactly `gabo-assistant`",
		"for Codex, its block in `AGENTS.override.md` and `AGENTS.md`",
		"keep and report it",
	)
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
