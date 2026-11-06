package skills

import (
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
)

const runtimeClaude = "gabo-runtime-claude/SKILL.md"

func runtimeClaudeDoc(t *testing.T) string {
	t.Helper()
	b, err := content.ReadFile(runtimeClaude)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// gabo check reads the agent directory from the runtime skill's frontmatter
// (TRD 9.4, T047), so the Claude Code skill must declare it.
func TestRuntimeClaudeDeclaresItsAgentDir(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	rest, _ := strings.CutPrefix(doc, "---\n")
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("%s: no frontmatter", runtimeClaude)
	}
	var fm struct {
		Metadata struct {
			AgentDir     string `yaml:"agent_dir"`
			SkillDir     string `yaml:"skill_dir"`
			ReviewerLink string `yaml:"reviewer_link"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(head), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Metadata.AgentDir != ".claude/agents" {
		t.Fatalf("metadata.agent_dir = %q, want .claude/agents", fm.Metadata.AgentDir)
	}
	// gabo check requires the skill links and the reviewer link there (#135).
	if fm.Metadata.SkillDir != ".claude/skills" {
		t.Fatalf("metadata.skill_dir = %q, want .claude/skills", fm.Metadata.SkillDir)
	}
	if want := "${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer"; fm.Metadata.ReviewerLink != want {
		t.Fatalf("metadata.reviewer_link = %q, want %q", fm.Metadata.ReviewerLink, want)
	}
}

var toolRow = regexp.MustCompile("(?m)^\\| `([a-z-]+)` \\| ([^|]+) \\|")

// Every engine-neutral capability, including every one the spec uses, maps
// to Claude Code tools, so no agent gets a capability the skill cannot write.
func TestRuntimeClaudeMapsEveryCapability(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	_, section, ok := strings.Cut(doc, "## Tool Map")
	if !ok {
		t.Fatalf("%s: no ## Tool Map section", runtimeClaude)
	}
	section, _, _ = strings.Cut(section, "\n## ")
	mapped := map[string]bool{}
	for _, m := range toolRow.FindAllStringSubmatch(section, -1) {
		if strings.TrimSpace(m[2]) == "" {
			t.Errorf("capability %s maps to no tool", m[1])
		}
		mapped[m[1]] = true
	}
	need := map[string]bool{}
	for _, c := range []string{"read", "search", "edit", "shell", "git", "tracker", "repository-host", "memory", "web", "skills", "subagents"} {
		need[c] = true
	}
	spec, err := agentspec.Load(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		for _, c := range a.Tools {
			need[c] = true
		}
	}
	for c := range need {
		if !mapped[c] {
			t.Errorf("capability %s has no row in ## Tool Map", c)
		}
	}
}

// The skill says how to fill every template slot (T049) and where each
// fixed-name link points.
func TestRuntimeClaudeFillsEverySlot(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	for _, s := range agentSlots {
		if !strings.Contains(doc, "`{{"+s+"}}`") {
			t.Errorf("%s: does not say how to fill {{%s}}", runtimeClaude, s)
		}
	}
	for _, s := range []string{"`gabo-methodology`", "`gabo-tasks`", "`gabo-repository-host`", "`gabo-memory`", "gabo-version:"} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %s", runtimeClaude, s)
		}
	}
}

// Every location cites the Claude Code docs and the version it was checked
// against, with the recheck commands (former TRD 9.6 rows).
func TestRuntimeClaudeCitesItsSources(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	for _, s := range []string{
		"https://code.claude.com/docs/en/sub-agents", "https://code.claude.com/docs/en/skills",
		"Claude Code 2.1.302", "`claude --version`", "`claude --help`",
		"`.claude/agents/<id>.md`", "`.claude/skills/<name>`", "`~/.claude/skills/gabo-reviewer`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeClaude, s)
		}
	}
}

// The reviewer scripts never enter the project (D20): the skill only links
// the user-level copy and refuses to overwrite what it did not create.
func TestRuntimeClaudeKeepsTheReviewerOutOfTheProject(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	for _, s := range []string{
		"Never place `gabo-reviewer` in the project",
		"${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer",
		"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer/scripts/reviewer.sh",
		"leave it and report it",
		"`gabo check`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeClaude, s)
		}
	}
}

// The probe checks every fixed-name link on every build, has a fallback when a
// link does not preload, always removes its file, and the verify step does not
// loop on agents the skill skipped (review of PR #127).
func TestRuntimeClaudeProbesEveryLink(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	for _, s := range []string{
		"Probe, on every build", "every fixed-name link made in step 3",
		"delete the file whatever the result", "the adapter's frontmatter `name` instead of the link name",
		"still failing, stop and report the link", "probe not run: restart Claude Code and rebuild",
		"Only a probe that ran in such a session can fail a link", "repeat step 7 and probe once more",
		"remove each `.claude/skills/gabo-*` symlink that no spec skill or slot asks for any more", "except the `✗ agent <id> missing` lines of agents you skipped",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeClaude, s)
		}
	}
}

// The Claude Code skill carries the narrowed engine-config rule (intent 0004
// S8, D7): `gabo init` writes the project-level gabo integration keys
// (intent 0006 S13), and the skill itself writes only the main-session key
// (spec 0009 S2, D3).
func TestRuntimeClaudeNarrowsTheEngineConfigRule(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	assertEngineConfigRule(t, runtimeClaude, doc,
		"Claude Code needs no engine key today, so this skill writes no settings file",
		"this skill writes no settings file",
	)
	for _, s := range []string{
		"Never install or upgrade Claude Code", "`~/.claude/settings.json`",
		"`gabo init` writes the project-level gabo integration keys (intent 0006); this skill writes only the one main-session key `agent` in `.claude/settings.json` (Main-Session Entry).",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeClaude, s)
		}
	}
}

// runtimeAgentPlaces returns metadata.agent_skills and metadata.agent_name
// of a catalog runtime skill (intent 0008 D1), each "" when absent.
func runtimeAgentPlaces(t *testing.T, file string) (skills, name string) {
	t.Helper()
	b, err := content.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	rest, _ := strings.CutPrefix(string(b), "---\n")
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("%s: no frontmatter", file)
	}
	var fm struct {
		Metadata struct {
			AgentSkills *string `yaml:"agent_skills"`
			AgentName   *string `yaml:"agent_name"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(head), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Metadata.AgentSkills != nil {
		skills = *fm.Metadata.AgentSkills
	}
	if fm.Metadata.AgentName != nil {
		name = *fm.Metadata.AgentName
	}
	return skills, name
}

// gabo check reads each agent's skill list and declared name where the
// runtime skill says (intent 0008 S2, D1).
func TestRuntimeClaudeDeclaresAgentPlaces(t *testing.T) {
	t.Parallel()
	skills, name := runtimeAgentPlaces(t, runtimeClaude)
	if skills != "frontmatter:skills" || name != "frontmatter:name" {
		t.Fatalf("agent_skills, agent_name = %q, %q; want frontmatter:skills, frontmatter:name", skills, name)
	}
}

// mainSessionSection returns the ## Main-Session Entry section of the Claude
// Code skill.
func mainSessionSection(t *testing.T) string {
	t.Helper()
	_, section, ok := strings.Cut(runtimeClaudeDoc(t), "\n## Main-Session Entry\n")
	if !ok {
		t.Fatalf("%s: no ## Main-Session Entry section", runtimeClaude)
	}
	section, _, _ = strings.Cut(section, "\n## ")
	return section
}

// When runtimes.assistant is claude the skill writes the Claude Code entry:
// the link, the agent file from the T1 template in the form the T2 probe
// chose (a short prompt, choice A), and only the `agent` key (spec 0009 S1,
// S2, D2).
func TestRuntimeClaudeWritesTheMainSessionEntry(t *testing.T) {
	t.Parallel()
	section := mainSessionSection(t)
	for _, s := range []string{
		"`runtimes.assistant` is `claude`",
		"`.claude/skills/gabo-assistant`", "`../../.gabo/skills/gabo-assistant`",
		"`.claude/agents/gabo-assistant.md`", "`main_session.template`", "`main_session.role`",
		"gabo-version: <x.y.z, unquoted>", "the filled body without its first line `gabo-version: …`",
		"never an empty body",
		"`agent: \"gabo-assistant\"`", "`.claude/settings.json`", "keep every other byte",
		"a user's own `agent` value", "keep it and report it",
		"`.gabo/skills/gabo-assistant/SKILL.md`",
	} {
		if !strings.Contains(section, s) {
			t.Errorf("%s Main-Session Entry: missing %q", runtimeClaude, s)
		}
	}
}

// When runtimes.assistant is another engine the skill removes its own entry,
// recognized only by its gabo identity (spec 0009 S4).
func TestRuntimeClaudeRemovesItsOwnEntry(t *testing.T) {
	t.Parallel()
	section := mainSessionSection(t)
	for _, s := range []string{
		"`runtimes.assistant` is not `claude`",
		"the `.claude/skills/gabo-assistant` symlink",
		"`.claude/agents/gabo-assistant.md` when it carries a `gabo-version:` line",
		"the `agent` key only when its value is exactly `gabo-assistant`",
	} {
		if !strings.Contains(section, s) {
			t.Errorf("%s Main-Session Entry: missing %q", runtimeClaude, s)
		}
	}
}

// On every build the skill removes the gabo entry of every other full-tier
// engine, read from the one table in gabo-agent-templates (spec 0009 S4, D4),
// and never a piece the assistant engine's own entry uses.
func TestRuntimeClaudeRemovesOtherEnginesEntries(t *testing.T) {
	t.Parallel()
	section := mainSessionSection(t)
	for _, s := range []string{
		"On every build", "Main-Session Entry table of `gabo-agent-templates`",
		"`<!-- GABO_ASSISTANT_START -->`", "`<!-- GABO_ASSISTANT_END -->`",
		"`AGENTS.override.md`", "`AGENTS.md`", "`CLAUDE.md`",
		"`default_agent: \"gabo-assistant\"`", "`opencode.json`", "`opencode.jsonc`",
		"`.opencode/opencode.json`", "`.opencode/opencode.jsonc`",
		"`.opencode/agents/gabo-assistant.md`", "`.agents/skills/gabo-assistant`",
		"never a piece the entry of `runtimes.assistant` also uses",
	} {
		if !strings.Contains(section, s) {
			t.Errorf("%s Main-Session Entry: missing %q", runtimeClaude, s)
		}
	}
	// Every file of the T1 table appears here, so the two cannot drift.
	tb, err := content.ReadFile("gabo-agent-templates/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	_, table, _ := strings.Cut(string(tb), "## Main-Session Entry")
	table, _, _ = strings.Cut(table, "\n## ")
	for _, m := range regexp.MustCompile("`([^`]*(?:\\.md|gabo-assistant))`").FindAllStringSubmatch(table, -1) {
		if !strings.Contains(section, "`"+m[1]+"`") {
			t.Errorf("%s Main-Session Entry: T1 table names %q, the skill does not", runtimeClaude, m[1])
		}
	}
}

// The probe of build step 8 gains the main-session check (spec 0009 D2, S2):
// a fresh session runs as the assistant, a role session keeps its role.
func TestRuntimeClaudeProbesTheMainSession(t *testing.T) {
	t.Parallel()
	doc := runtimeClaudeDoc(t)
	_, step8, ok := strings.Cut(doc, "\n8. Probe, on every build.")
	if !ok {
		t.Fatalf("%s: no build step 8", runtimeClaude)
	}
	step8, _, _ = strings.Cut(step8, "\n## ")
	for _, s := range []string{
		"Main-session check", "`claude --print` with no `--agent`",
		"`claude --print --agent gabo-coder`", "runs as `gabo-coder`",
		"main-session probe failed",
	} {
		if !strings.Contains(step8, s) {
			t.Errorf("%s step 8: missing %q", runtimeClaude, s)
		}
	}
}
