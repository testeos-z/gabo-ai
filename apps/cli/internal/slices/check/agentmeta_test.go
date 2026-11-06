package check

import (
	"slices"
	"strings"
	"testing"
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
