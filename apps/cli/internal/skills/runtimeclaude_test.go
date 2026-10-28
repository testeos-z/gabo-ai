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
			AgentDir string `yaml:"agent_dir"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(head), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Metadata.AgentDir != ".claude/agents" {
		t.Fatalf("metadata.agent_dir = %q, want .claude/agents", fm.Metadata.AgentDir)
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
		"$HOME/.claude/skills/gabo-reviewer/scripts/reviewer.sh",
		"leave it and report it",
		"`gabo check`",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("%s: missing %q", runtimeClaude, s)
		}
	}
}
