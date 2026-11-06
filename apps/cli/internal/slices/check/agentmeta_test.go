package check

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// runtimeDoc is a runtime skill whose metadata holds the given lines.
func runtimeDoc(meta ...string) []byte {
	return []byte("---\nname: gabo-runtime-x\nmetadata:\n  agent_dir: .x/agents\n" +
		strings.Join(meta, "\n") + "\n---\nbody\n")
}

// D1: agent_skills and agent_name are <form>:<argument> strings in one of
// three forms.
func TestParseRuntimeMetaReadsAgentPlaces(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		meta      []string
		skills    agentPlace
		agentName agentPlace
		hasSkills bool
		hasName   bool
	}{
		{"frontmatter", []string{"  agent_skills: frontmatter:skills", "  agent_name: frontmatter:name"},
			agentPlace{"frontmatter", "skills"}, agentPlace{"frontmatter", "name"}, true, true},
		{"toml and line", []string{`  agent_skills: "line:Load these skills by name before work:"`, "  agent_name: toml:name"},
			agentPlace{"line", "Load these skills by name before work:"}, agentPlace{"toml", "name"}, true, true},
		{"line only", []string{`  agent_skills: "line:Skills:"`},
			agentPlace{"line", "Skills:"}, agentPlace{}, true, false},
		{"none", nil, agentPlace{}, agentPlace{}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m, err := parseRuntimeMeta(runtimeDoc(tt.meta...))
			if err != nil {
				t.Fatalf("parseRuntimeMeta() error = %v", err)
			}
			if m.AgentSkills != tt.skills || m.AgentName != tt.agentName {
				t.Fatalf("places = %+v, %+v; want %+v, %+v", m.AgentSkills, m.AgentName, tt.skills, tt.agentName)
			}
			if m.AgentSkills.declared() != tt.hasSkills || m.AgentName.declared() != tt.hasName {
				t.Fatalf("declared = %v, %v; want %v, %v", m.AgentSkills.declared(), m.AgentName.declared(), tt.hasSkills, tt.hasName)
			}
			if m.AgentDir != ".x/agents" {
				t.Fatalf("agent_dir = %q", m.AgentDir)
			}
		})
	}
}

func TestParseRuntimeMetaRefusesUnknownForm(t *testing.T) {
	t.Parallel()
	for _, meta := range []string{
		"  agent_skills: json:skills",
		"  agent_skills: skills",
		"  agent_skills: \"frontmatter:\"",
		"  agent_name: yaml:name",
	} {
		t.Run(meta, func(t *testing.T) {
			t.Parallel()
			_, err := parseRuntimeMeta(runtimeDoc(meta))
			if err == nil {
				t.Fatal("parseRuntimeMeta() error = nil, want an unknown-form error")
			}
			key := strings.TrimSpace(strings.SplitN(meta, ":", 2)[0])
			if !strings.Contains(err.Error(), "metadata."+key) {
				t.Fatalf("error %q does not name metadata.%s", err, key)
			}
		})
	}
}

// Each form reads an agent file in the shape its runtime skill writes it.
func TestAgentPlaceReadsAgentFiles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		skills, own agentPlace
		file        string
		wantSkills  []string
		wantName    string
	}{
		{
			name: "claude frontmatter list", skills: agentPlace{"frontmatter", "skills"}, own: agentPlace{"frontmatter", "name"},
			file:       "---\nname: gabo-pm\ndescription: 'Product: PRD'\ntools: Read, Edit\nskills:\n  - gabo-pm\n  - gabo-plan-reviewer\n  - gabo-memory\ngabo-version: 1.2.3\n---\nbody\nSkills: not this\n",
			wantSkills: []string{"gabo-pm", "gabo-plan-reviewer", "gabo-memory"}, wantName: "gabo-pm",
		},
		{
			name: "frontmatter scalar", skills: agentPlace{"frontmatter", "skills"}, own: agentPlace{"frontmatter", "name"},
			file:       "---\nname: gabo-pm\nskills: gabo-pm, gabo-memory\n---\n",
			wantSkills: []string{"gabo-pm", "gabo-memory"}, wantName: "gabo-pm",
		},
		{
			name: "codex developer_instructions line", skills: agentPlace{"line", "Load these skills by name before work:"}, own: agentPlace{"toml", "name"},
			file:       "# gabo-version: 1.2.3\nname = \"gabo-pm\"\ndescription = \"Product\"\ndeveloper_instructions = \"\"\"\nYou are the PM.\n\nLoad these skills by name before work: gabo-pm, gabo-plan-reviewer, gabo-memory.\n\"\"\"\n",
			wantSkills: []string{"gabo-pm", "gabo-plan-reviewer", "gabo-memory"}, wantName: "gabo-pm",
		},
		{
			name: "opencode body line", skills: agentPlace{"line", "Skills:"},
			file:       "---\ndescription: \"Product\"\nmode: subagent\n---\n<!-- gabo-version: 1.2.3 -->\nSkills: before work read .opencode/skills/<name>/SKILL.md for each of gabo-pm, gabo-plan-reviewer.\n\nbody\n",
			wantSkills: []string{"gabo-pm", "gabo-plan-reviewer"},
		},
		{
			name: "pi body line with backticks", skills: agentPlace{"line", "Skills:"}, own: agentPlace{"frontmatter", "name"},
			file:       "---\nname: gabo-pm\ndescription: \"Product\"\ngabo-version: 1.2.3\ntools:\n  - read\n---\nSkills: before work read `.pi/skills/<name>/SKILL.md` for each of `gabo-pm`, gabo-memory.\n\nbody\n",
			wantSkills: []string{"gabo-pm", "gabo-memory"}, wantName: "gabo-pm",
		},
		{
			name: "no matching place", skills: agentPlace{"line", "Skills:"}, own: agentPlace{"toml", "name"},
			file: "---\nname: gabo-pm\n---\nbody\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := []byte(tt.file)
			if got := tt.skills.skills(b); !slices.Equal(got, tt.wantSkills) {
				t.Fatalf("skills() = %q, want %q", got, tt.wantSkills)
			}
			if got := tt.own.name(b); got != tt.wantName {
				t.Fatalf("name() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

// S2: a full-tier runtime skill without agent_skills fails.
func TestFullTierRuntimeWithoutAgentSkillsFails(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n  agent_name: frontmatter:name\n---\n",
	})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ runtime claude: gabo-runtime-claude declares no metadata.agent_skills", NextRestoreCatalog)
	if !strings.HasPrefix(NextRestoreCatalog, "next: run gabo init to restore the catalog") {
		t.Fatalf("NextRestoreCatalog = %q", NextRestoreCatalog)
	}
}

// S2: a full-tier runtime skill that declares agent_skills passes, also
// without agent_name.
func TestFullTierRuntimeWithAgentSkillsPasses(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n  agent_skills: frontmatter:skills\n---\n",
	})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "agent_skills")
}

// S2, TRD D28: a flexible-tier engine without agent_skills gets one warning
// naming the engine and no failure.
func TestFlexibleTierRuntimeWithoutAgentSkillsWarns(t *testing.T) {
	t.Parallel()
	var r Report
	checkAgentMeta(&r, "zed", runtimeMeta{AgentDir: ".zed/agents"})
	if r.Failed {
		t.Fatalf("Failed = true:\n%s", joined(r))
	}
	if len(r.Lines) != 1 || !strings.HasPrefix(r.Lines[0], "! ") || !strings.Contains(r.Lines[0], "zed") {
		t.Fatalf("lines = %q, want one ! line naming zed", r.Lines)
	}

	r = Report{}
	checkAgentMeta(&r, "zed", runtimeMeta{AgentDir: ".zed/agents", AgentSkills: agentPlace{"line", "Skills:"}})
	if r.Failed || len(r.Lines) != 0 {
		t.Fatalf("declared: lines = %q, failed = %v", r.Lines, r.Failed)
	}
}

// S9: under GABO_UPGRADE_PR=1 the missing-declaration line never prints.
func TestMissingAgentSkillsSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n---\n",
	})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	noLineWith(t, r, "agent_skills")
}

// piCoder is the plain project with Pi as its coder engine: a Pi runtime
// skill and both spec agents built in .pi/agents, alpha with alphaPi.
func piCoder(alphaPi string) (fstest.MapFS, string) {
	cfg := strings.Replace(validConfig, "  coder: claude\n", "  coder: pi\n", 1)
	return with(project(), map[string]string{
		".gabo/skills/gabo-runtime-pi/SKILL.md": "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n  agent_skills: frontmatter:skills\n---\n",
		".pi/agents/alpha.md":                   alphaPi,
		".pi/agents/beta.md":                    "---\nname: beta\ngabo-version: 1.2.3\ngabo-engine: pi\n---\nbody\n",
		".pi/settings.json":                     piKeysJSON,
	}), cfg
}

// Intent 0013 S6: a built agent whose filled engine differs from the engine
// of its agent directory fails with the rebuild hint; a matching engine and
// a file with no engine line (a 0.2.10 build) pass.
func TestAgentBuiltForAnotherEngineFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, alpha string
		fail        bool
	}{
		{"claude agent copied into .pi/agents", "---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: claude\n---\nbody\n", true},
		{"quoted value", "---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: \"claude\"\n---\nbody\n", true},
		{"matching engine", "---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: pi\n---\nbody\n", false},
		{"no engine line", "---\nname: alpha\ngabo-version: 1.2.3\n---\nbody\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proj, cfg := piCoder(tt.alpha)
			r := run(t, proj, configFiles{projectFile: cfg}, Settings{})
			if tt.fail {
				wantFailed(t, r, "✗ .pi/agents/alpha.md: built for claude, not pi", NextRebuild)
				if !strings.HasPrefix(NextRebuild, "next: ask your agent to rebuild the agents") {
					t.Fatalf("NextRebuild = %q", NextRebuild)
				}
				return
			}
			wantPassed(t, r, PassedLine)
			noLineWith(t, r, "built for")
		})
	}
}

// Intent 0013 S6: the Claude Code agents of the plain project carry their
// own engine and pass beside the Pi ones; only the Pi file built for Claude
// fails.
func TestAgentBuiltForChecksEachAgentDirectory(t *testing.T) {
	t.Parallel()
	proj, cfg := piCoder("---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: claude\n---\nbody\n")
	proj = with(proj, map[string]string{
		".claude/agents/alpha.md": "---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: claude\n---\nbody\n",
		".claude/agents/beta.md":  "---\nname: beta\ngabo-version: 1.2.3\ngabo-engine: pi\n---\nbody\n",
	})
	r := run(t, proj, configFiles{projectFile: cfg}, Settings{})
	wantFailed(t, r, "✗ .pi/agents/alpha.md: built for claude, not pi", "✗ .claude/agents/beta.md: built for pi, not claude")
	noLineWith(t, r, ".claude/agents/alpha.md: built for")
}

// mapFSEdit turns a project into the edit map of the golden harness.
func mapFSEdit(proj fstest.MapFS) map[string]string {
	edit := map[string]string{}
	for name, f := range proj {
		edit[name] = string(f.Data)
	}
	return edit
}

// Intent 0013 S6: the help names the built-for rule.
func TestCommandHelpNamesTheBuiltForRule(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{
		"  - each agent file's filled gabo-engine line names the engine of the agent",
		"    directory it is in (a file with no engine line passes);",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

// Intent 0013 S6: the main-session agent file is a built agent too.
func TestMainSessionAgentBuiltForAnotherEngineFails(t *testing.T) {
	t.Parallel()
	proj := with(project(), mainSessionEdit(map[string]string{
		".claude/agents/gabo-assistant.md": "---\nname: gabo-assistant\ngabo-version: 1.2.3\ngabo-engine: opencode\n---\nbody\n",
		".claude/settings.json":            withJSONKey(claudeKeysJSON, "agent", "gabo-assistant"),
	}))
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ .claude/agents/gabo-assistant.md: built for opencode, not claude", NextRebuild)
}
