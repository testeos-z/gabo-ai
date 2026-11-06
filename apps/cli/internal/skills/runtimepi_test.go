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
			AgentDir string `yaml:"agent_dir"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(front, &fm); err != nil {
		t.Fatalf("gabo-runtime-pi/SKILL.md frontmatter: %v", err)
	}
	if fm.Metadata.AgentDir != ".pi/agents" {
		t.Errorf("metadata.agent_dir = %q, want .pi/agents", fm.Metadata.AgentDir)
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
		// skills and fixed-name links.
		"`.pi/skills/<skill>`", "`../../.gabo/skills/<skill>`", "`.pi/skills/gabo-methodology`", "`gabo-<slot>`",
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
