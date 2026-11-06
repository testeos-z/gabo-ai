package skills

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const runtimeCodex = "gabo-runtime-codex/SKILL.md"

func runtimeCodexDoc(t *testing.T) string {
	t.Helper()
	b, err := content.ReadFile(runtimeCodex)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The Codex runtime skill ships in the catalog (intent 0002 S4).
func TestRuntimeCodexIsACatalogSkill(t *testing.T) {
	t.Parallel()
	if !slices.Contains(Names(), "gabo-runtime-codex") {
		t.Fatalf("Names() = %v, lacks gabo-runtime-codex", Names())
	}
}

// gabo check reads the agent directory, the skill link directory and the
// user-level reviewer link from the runtime skill's frontmatter (D2, S5).
func TestRuntimeCodexDeclaresItsDirs(t *testing.T) {
	t.Parallel()
	doc := runtimeCodexDoc(t)
	rest, _ := strings.CutPrefix(doc, "---\n")
	head, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("%s: no frontmatter", runtimeCodex)
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
	if fm.Metadata.AgentDir != ".codex/agents" {
		t.Errorf("metadata.agent_dir = %q, want .codex/agents", fm.Metadata.AgentDir)
	}
	if fm.Metadata.SkillDir != ".agents/skills" {
		t.Errorf("metadata.skill_dir = %q, want .agents/skills", fm.Metadata.SkillDir)
	}
	if want := "$HOME/.agents/skills/gabo-reviewer"; fm.Metadata.ReviewerLink != want {
		t.Errorf("metadata.reviewer_link = %q, want %q", fm.Metadata.ReviewerLink, want)
	}
}

var tomlFence = regexp.MustCompile("(?ms)^( *)```toml\n(.*?)\n *```$")

// tomlBlock returns the first ```toml code block of the skill without the
// indentation of its fence.
func tomlBlock(t *testing.T, doc string) string {
	t.Helper()
	m := tomlFence.FindStringSubmatch(doc)
	if m == nil {
		t.Fatalf("%s: no ```toml block", runtimeCodex)
	}
	lines := strings.Split(m[2], "\n")
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, m[1])
	}
	return strings.Join(lines, "\n")
}

var tomlKey = regexp.MustCompile(`(?m)^\s*\[?([A-Za-z0-9_.-]+)\]?\s*(=|$)`)

// The agent file has the shape of spec section 2: the gabo-version comment,
// then name, description and developer_instructions ending with the skill
// line; no model, sandbox_mode or features key (D6).
func TestRuntimeCodexWritesTheSpecTOMLShape(t *testing.T) {
	t.Parallel()
	block := tomlBlock(t, runtimeCodexDoc(t))
	lines := strings.Split(block, "\n")
	if !strings.HasPrefix(lines[0], "# gabo-version: ") {
		t.Errorf("first line %q, want the # gabo-version: comment", lines[0])
	}
	for i, want := range []string{`name = "`, `description = "`, `developer_instructions = """`} {
		if i+1 >= len(lines) || !strings.HasPrefix(lines[i+1], want) {
			t.Errorf("line %d does not start with %q:\n%s", i+2, want, block)
		}
	}
	if !strings.Contains(block, "\nLoad these skills by name before work: ") || !strings.HasSuffix(block, "\n\"\"\"") {
		t.Errorf("developer_instructions lacks the closing skill line:\n%s", block)
	}
	// Only the three keys, outside the multi-line string.
	_, after, _ := strings.Cut(block, `"""`)
	_, outside, _ := strings.Cut(after, `"""`)
	header, _, _ := strings.Cut(block, `developer_instructions`)
	var keys []string
	for _, m := range tomlKey.FindAllStringSubmatch(header+outside, -1) {
		keys = append(keys, m[1])
	}
	if want := []string{"name", "description"}; !slices.Equal(keys, want) {
		t.Errorf("keys before and after developer_instructions = %v, want %v", keys, want)
	}
	doc := runtimeCodexDoc(t)
	for _, s := range []string{"No `model`, `sandbox_mode` or `features` key", "`.codex/agents/<id>.toml`"} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeCodex, s)
		}
	}
}

// The skill says how to fill every template slot and names the fixed-name
// links and the escaping that keeps every body valid TOML.
func TestRuntimeCodexFillsEverySlot(t *testing.T) {
	t.Parallel()
	doc := runtimeCodexDoc(t)
	for _, s := range agentSlots {
		if !strings.Contains(doc, "`{{"+s+"}}`") {
			t.Errorf("%s: does not say how to fill {{%s}}", runtimeCodex, s)
		}
	}
	for _, s := range []string{
		"`gabo-methodology`", "`gabo-tasks`", "`gabo-repository-host`", "`gabo-memory`",
		"`.agents/skills/<name>`", "`../../.gabo/skills/<name>`", "`.agents/skills/gabo-<name>`",
		"each `\\` as `\\\\`", "each `\"\"\"` as `\"\"\\\"`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeCodex, s)
		}
	}
}

// The reviewer stays out of the project (D20) and the skill writes no
// user-level file but the reviewer link: nothing under ~/.codex.
func TestRuntimeCodexKeepsUserFilesAndTheReviewerOut(t *testing.T) {
	t.Parallel()
	doc := runtimeCodexDoc(t)
	for _, s := range []string{
		"Never place `gabo-reviewer` in the project",
		"no `.agents/skills/gabo-reviewer`",
		"${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer",
		"$HOME/.agents/skills/gabo-reviewer/scripts/reviewer.sh",
		"Never write under `~/.codex` or `$CODEX_HOME`",
		"probe not run: run gabo-runtime-codex inside Codex",
		"`.agents/skills/` is shared with other tools, so leave and report any other `gabo-*` symlink",
		"no user-level file but the reviewer link",
		"leave it and report it",
		"`gabo check`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeCodex, s)
		}
	}
}

// Every location cites the Codex docs and the version it was checked
// against, with the recheck command.
func TestRuntimeCodexCitesItsSources(t *testing.T) {
	t.Parallel()
	doc := runtimeCodexDoc(t)
	for _, s := range []string{
		"https://learn.chatgpt.com/docs/agent-configuration/subagents",
		"https://learn.chatgpt.com/docs/build-skills",
		"codex-cli 0.163.0", "`codex --version`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeCodex, s)
		}
	}
}

// intent 0008 S2, D1.
func TestRuntimeCodexDeclaresAgentPlaces(t *testing.T) {
	t.Parallel()
	skills, name := runtimeAgentPlaces(t, runtimeCodex)
	if skills != "line:Load these skills by name before work:" || name != "toml:name" {
		t.Fatalf("agent_skills, agent_name = %q, %q; want line:Load these skills by name before work:, toml:name", skills, name)
	}
}
