package check

import (
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"path"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the testdata/golden files")

// TestCommandGolden pins `gabo check` without --pre-write: its output and
// exit code on every intent fixture, on the plain project and on usage
// errors equal the golden files written before the hook mode existed.
func TestCommandGolden(t *testing.T) {
	t.Parallel()
	fixtures, err := os.ReadDir(path.Join("testdata", "intents"))
	if err != nil {
		t.Fatal(err)
	}
	type tc struct {
		name string
		args []string
		tree string
		// cfg is the project gabo.yaml; empty means validConfig.
		cfg string
		// edit changes project files; an empty value removes the file.
		edit map[string]string
	}
	cases := []tc{
		{name: "no-intents"},
		{name: "usage-extra-arg", args: []string{"extra"}},
		{name: "usage-unknown-flag", args: []string{"--bogus"}},
		{name: "versions-no-stamp", edit: map[string]string{".gabo/VERSION": ""}},
		{name: "versions-no-version-line", edit: map[string]string{".claude/agents/beta.md": "---\nname: beta\n---\nbody\n"}},
		{name: "agent-skills-extra", edit: map[string]string{".claude/agents/alpha.md": "---\nname: alpha\nskills:\n  - sk-x\ngabo-version: 1.2.3\n---\nbody\n"}},
		// Spec 0014 S1: a runtime skill declaring agent_name and an agent
		// file that declares no name.
		{name: "agent-name-missing", edit: map[string]string{
			".gabo/skills/gabo-runtime-claude/SKILL.md": "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n" +
				"  agent_skills: frontmatter:skills\n  agent_name: frontmatter:name\n---\nbody\n",
			".claude/agents/alpha.md": "---\ngabo-version: 1.2.3\n---\nbody\n",
		}},
	}
	// Spec 0009 S5 on the plain project with a main_session entry: nothing of
	// the claude entry built, then a correct entry beside a Codex block.
	cases = append(cases,
		tc{name: "main-session-missing", edit: mainSessionEdit(nil)},
		tc{name: "main-session-stale", edit: mainSessionEdit(map[string]string{
			".claude/agents/gabo-assistant.md":         "---\nname: gabo-assistant\ngabo-version: 1.2.3\n---\nbody\n",
			".claude/settings.json":                    withJSONKey(claudeKeysJSON, "agent", "gabo-assistant"),
			".gabo/skills/gabo-runtime-codex/SKILL.md": "---\nname: gabo-runtime-codex\nmetadata:\n  main_session:\n    kind: block\n    files: [AGENTS.override.md, AGENTS.md]\n---\n",
			"AGENTS.md": "# Project\n\n" + assistantStart + "\nYou are the gabo assistant.\n" + assistantEnd + "\n",
		})},
	)
	// Intent 0010 S6: the lines whose fix is gabo init name it, with the key
	// and file of metadata.main_session.
	assistantAgent := map[string]string{".claude/agents/gabo-assistant.md": "---\nname: gabo-assistant\ngabo-version: 1.2.3\n---\nbody\n"}
	cases = append(cases,
		tc{name: "main-session-key-only", edit: mainSessionEdit(assistantAgent)},
		tc{name: "main-session-other-value", edit: mainSessionEdit(withEdits(assistantAgent, map[string]string{
			".claude/settings.json": withJSONKey(claudeKeysJSON, "agent", "my-agent"),
		}))},
		tc{name: "main-session-stale-key", edit: mainSessionEdit(withEdits(assistantAgent, map[string]string{
			".claude/settings.json": withJSONKey(claudeKeysJSON, "agent", "gabo-assistant"),
			".gabo/skills/gabo-runtime-opencode/SKILL.md": "---\nname: gabo-runtime-opencode\nmetadata:\n  main_session:\n    kind: agent\n" +
				"    config: [opencode.json, opencode.jsonc, .opencode/opencode.json, .opencode/opencode.jsonc]\n    key: default_agent\n---\n",
			"opencode.json": `{"default_agent": "gabo-assistant"}`,
		}))},
	)
	// Intent 0013 S5: assistant [claude, pi], both entries built; then the Pi
	// block removed; then a leftover OpenCode key, whose stale line names the
	// list.
	listedCfg := strings.Replace(validConfig, "  assistant: claude\n", "  assistant: [claude, pi]\n", 1)
	listed := withEdits(assistantAgent, map[string]string{
		".claude/settings.json": withJSONKey(claudeKeysJSON, "agent", "gabo-assistant"),
		".gabo/skills/gabo-runtime-pi/SKILL.md": "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n  agent_skills: frontmatter:skills\n" +
			"  main_session:\n    kind: block\n    files: [AGENTS.md]\n---\n",
		".pi/agents/alpha.md": "---\nname: alpha\ngabo-version: 1.2.3\n---\nbody\n",
		".pi/agents/beta.md":  "---\nname: beta\ngabo-version: 1.2.3\n---\nbody\n",
		".pi/settings.json":   piKeysJSON,
		"AGENTS.md":           "# Project\n\n" + assistantStart + "\nYou are the gabo assistant.\n" + assistantEnd + "\n",
	})
	cases = append(cases,
		tc{name: "main-session-list-built", cfg: listedCfg, edit: mainSessionEdit(listed)},
		tc{name: "main-session-list-pi-block-removed", cfg: listedCfg, edit: mainSessionEdit(withEdits(listed, map[string]string{
			"AGENTS.md": "# Project\n",
		}))},
		tc{name: "main-session-list-stale-key", cfg: listedCfg, edit: mainSessionEdit(withEdits(listed, map[string]string{
			".gabo/skills/gabo-runtime-opencode/SKILL.md": "---\nname: gabo-runtime-opencode\nmetadata:\n  main_session:\n    kind: agent\n" +
				"    config: [opencode.json, opencode.jsonc, .opencode/opencode.json, .opencode/opencode.jsonc]\n    key: default_agent\n---\n",
			"opencode.json": `{"default_agent": "gabo-assistant"}`,
		}))},
	)
	// Intent 0013 S6: a Claude Code agent copied into .pi/agents.
	builtFor, builtForCfg := piCoder("---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: claude\n---\nbody\n")
	cases = append(cases, tc{name: "agent-built-for-other-engine", cfg: builtForCfg, edit: mapFSEdit(builtFor)})
	for _, f := range fixtures {
		cases = append(cases, tc{name: "intents-" + f.Name(), tree: f.Name()})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			proj := project()
			if c.edit != nil {
				proj = with(proj, c.edit)
			}
			if c.tree != "" {
				proj = withTree(t, proj, c.tree)
			}
			// A nil slice would make cobra read os.Args (the test flags).
			cfg := c.cfg
			if cfg == "" {
				cfg = validConfig
			}
			out, err := runCommandConfig(t, proj, cfg, append([]string{}, c.args...)...)
			code := 0
			var ee *ExitError
			if errors.As(err, &ee) {
				code = ee.Code
			} else if err != nil {
				t.Fatalf("err = %v, want an ExitError", err)
			}
			msg := ""
			if err != nil {
				msg = err.Error()
			}
			got := fmt.Sprintf("exit %d\nerror: %q\n--- stdout\n%s", code, msg, out)
			file := path.Join("testdata", "golden", c.name+".golden")
			if *update {
				if err := os.MkdirAll(path.Dir(file), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read golden (run with -update on the base to write it): %v", err)
			}
			if got != string(want) {
				t.Fatalf("output changed:\n got: %s\nwant: %s", got, want)
			}
		})
	}
}

// mainSessionEdit gives the plain project a main_session entry in its spec,
// its templates and skill, and metadata.main_session in its Claude Code
// runtime skill, plus extra.
func mainSessionEdit(extra map[string]string) map[string]string {
	edit := map[string]string{
		".gabo/skills/gabo-agent-templates/agents.yaml": spec + "main_session:\n  id: gabo-assistant\n  role: assistant\n" +
			"  skills: [gabo-assistant]\n  template: agents/gabo-assistant.md\n  block: agents/gabo-assistant-block.md\n",
		".gabo/skills/gabo-agent-templates/agents/gabo-assistant.md":       "gabo-version: {{gabo_version}}\n",
		".gabo/skills/gabo-agent-templates/agents/gabo-assistant-block.md": assistantStart + "\n" + assistantEnd + "\n",
		".gabo/skills/gabo-assistant/SKILL.md":                             "---\nname: gabo-assistant\n---\n",
		".gabo/skills/gabo-runtime-claude/SKILL.md": strings.Replace(claudeSkill, "  agent_skills: frontmatter:skills\n",
			"  agent_skills: frontmatter:skills\n  main_session:\n    kind: agent\n    config: [.claude/settings.json]\n    key: agent\n", 1),
	}
	maps.Copy(edit, extra)
	return edit
}

// withEdits returns a copy of base with extra added.
func withEdits(base, extra map[string]string) map[string]string {
	out := maps.Clone(base)
	maps.Copy(out, extra)
	return out
}
