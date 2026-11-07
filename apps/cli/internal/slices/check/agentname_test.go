package check

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// replaceIn rewrites old to repl once in the project file, which must hold old.
func replaceIn(t *testing.T, proj fstest.MapFS, name, old, repl string) fstest.MapFS {
	t.Helper()
	f, ok := proj[name]
	if !ok || !strings.Contains(string(f.Data), old) {
		t.Fatalf("%s lacks %q", name, old)
	}
	proj[name] = file(strings.Replace(string(f.Data), old, repl, 1))
	return proj
}

// wantConsecutive requires the lines in this order, one after the other.
func wantConsecutive(t *testing.T, r Report, lines ...string) {
	t.Helper()
	i := slices.Index(r.Lines, lines[0])
	if i < 0 || i+len(lines) > len(r.Lines) || !slices.Equal(r.Lines[i:i+len(lines)], lines) {
		t.Fatalf("output lacks consecutive lines %q:\n%s", lines, joined(r))
	}
}

// S6: an agent whose declared name differs from its spec id fails, for each
// engine whose runtime skill declares metadata.agent_name.
func TestDeclaredAgentNameDiffersFails(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T) Report
		want []string
	}{
		{"codex", func(t *testing.T) Report {
			proj := replaceIn(t, codexBuilt(t, "1.2.3"), ".codex/agents/gabo-pm.toml", `name = "gabo-pm"`, `name = "pm"`)
			return runCodex(t, proj, codexUser(), "1.2.3")
		}, []string{"✗ agent gabo-pm (codex): declares name pm", "why: .codex/agents/gabo-pm.toml", NextRebuild}},
		{"claude", func(t *testing.T) Report {
			proj := replaceIn(t, claudeBuilt(t, "1.2.3"), ".claude/agents/gabo-pm.md", "name: gabo-pm\n", "name: pm\n")
			return run(t, proj, configFiles{projectFile: validConfig}, Settings{})
		}, []string{"✗ agent gabo-pm (claude): declares name pm", "why: .claude/agents/gabo-pm.md", NextRebuild}},
		{"pi", func(t *testing.T) Report {
			proj := replaceIn(t, piBuilt(t, "1.2.3"), ".pi/agents/gabo-pm.md", "name: gabo-pm\n", "name: pm\n")
			return runPi(t, proj, Settings{})
		}, []string{"✗ agent gabo-pm (pi): declares name pm", "why: .pi/agents/gabo-pm.md", NextRebuild}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := tt.run(t)
			if !r.Failed {
				t.Fatalf("Failed = false, want true:\n%s", joined(r))
			}
			wantConsecutive(t, r, tt.want...)
			if !strings.HasPrefix(NextRebuild, "next: ask your agent to rebuild the agents") {
				t.Fatalf("NextRebuild = %q", NextRebuild)
			}
		})
	}
}

// S6: a correct build of each engine prints no name line; OpenCode declares
// no agent_name, so even a frontmatter name that differs prints none.
func TestDeclaredAgentNameMatchesPrintsNothing(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		run  func(t *testing.T) Report
	}{
		{"codex", func(t *testing.T) Report { return runCodex(t, codexBuilt(t, "1.2.3"), codexUser(), "1.2.3") }},
		{"claude", func(t *testing.T) Report {
			return run(t, claudeBuilt(t, "1.2.3"), configFiles{projectFile: validConfig}, Settings{})
		}},
		{"pi", func(t *testing.T) Report { return runPi(t, piBuilt(t, "1.2.3"), Settings{}) }},
		{"opencode", func(t *testing.T) Report { return runOpencode(t, opencodeBuilt(t, "1.2.3"), "1.2.3") }},
		{"opencode with another name", func(t *testing.T) Report {
			proj := replaceIn(t, opencodeBuilt(t, "1.2.3"), ".opencode/agents/gabo-pm.md", "---\n", "---\nname: pm\n")
			return runOpencode(t, proj, "1.2.3")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			noLineWith(t, tt.run(t), "declares name")
		})
	}
}

// S2, TRD D28: a flexible-tier engine without agent_skills is never failed by
// the name rule, even when it declares agent_name.
func TestDeclaredAgentNameSkippedForFlexibleTierWithoutAgentSkills(t *testing.T) {
	t.Parallel()
	proj := fstest.MapFS{
		CatalogDir + "/gabo-runtime-zed/SKILL.md": file("---\nname: gabo-runtime-zed\nmetadata:\n  agent_dir: .zed/agents\n  agent_name: frontmatter:name\n---\n"),
		".zed/agents/alpha.md":                    file("---\nname: pm\n---\n"),
	}
	var r Report
	(&UseCase{project: proj}).checkAgentName(&r, "alpha", "zed", ".zed/agents/alpha.md")
	if r.Failed || len(r.Lines) != 0 {
		t.Fatalf("lines = %q, failed = %v", r.Lines, r.Failed)
	}
}

// S9, spec 0014 S1: under GABO_UPGRADE_PR=1 the name line never prints, for
// a name that differs, a nameless agent file and a nameless gabo-assistant
// file.
func TestDeclaredAgentNameSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	t.Run("differs", func(t *testing.T) {
		t.Parallel()
		proj := replaceIn(t, piBuilt(t, "1.2.3"), ".pi/agents/gabo-pm.md", "name: gabo-pm\n", "name: pm\n")
		r := runPi(t, proj, Settings{UpgradePR: true})
		noLineWith(t, r, "declares name")
		wantLines(t, r, UpgradePRLine)
	})
	t.Run("nameless", func(t *testing.T) {
		t.Parallel()
		proj := replaceIn(t, claudeBuilt(t, "1.2.3"), ".claude/agents/gabo-pm.md", "name: gabo-pm\n", "")
		proj = replaceIn(t, proj, ".claude/agents/"+assistantID+".md", "name: "+assistantID+"\n", "")
		r := run(t, proj, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
		noLineWith(t, r, "declares")
		wantLines(t, r, UpgradePRLine)
	})
}

// Spec 0014 S1, D1, D2: an agent file that declares no name (absent, blank,
// non-string, unparsable frontmatter or TOML) fails, for each engine whose
// runtime skill declares metadata.agent_name.
func TestDeclaredAgentNameMissingFails(t *testing.T) {
	t.Parallel()
	codexLine := []string{"✗ agent gabo-pm (codex): declares no name", "why: .codex/agents/gabo-pm.toml", NextRebuild}
	claudeLine := []string{"✗ agent gabo-pm (claude): declares no name", "why: .claude/agents/gabo-pm.md", NextRebuild}
	piLine := []string{"✗ agent gabo-pm (pi): declares no name", "why: .pi/agents/gabo-pm.md", NextRebuild}
	codex := func(old, repl string) func(t *testing.T) Report {
		return func(t *testing.T) Report {
			proj := replaceIn(t, codexBuilt(t, "1.2.3"), ".codex/agents/gabo-pm.toml", old, repl)
			return runCodex(t, proj, codexUser(), "1.2.3")
		}
	}
	claude := func(old, repl string) func(t *testing.T) Report {
		return func(t *testing.T) Report {
			proj := replaceIn(t, claudeBuilt(t, "1.2.3"), ".claude/agents/gabo-pm.md", old, repl)
			return run(t, proj, configFiles{projectFile: validConfig}, Settings{})
		}
	}
	tests := []struct {
		name string
		run  func(t *testing.T) Report
		want []string
	}{
		{"codex without name", codex(`name = "gabo-pm"`+"\n", ""), codexLine},
		{"codex blank name", codex(`name = "gabo-pm"`, `name = "  "`), codexLine},
		{"codex non-string name", codex(`name = "gabo-pm"`, `name = 7`), codexLine},
		{"codex unparsable toml", codex(`name = "gabo-pm"`, `name = "gabo-pm`), codexLine},
		{"claude without name", claude("name: gabo-pm\n", ""), claudeLine},
		{"claude non-string name", claude("name: gabo-pm\n", "name: [gabo-pm]\n"), claudeLine},
		{"claude unparsable frontmatter", claude("name: gabo-pm\n", "name: gabo-pm\n: [\n"), claudeLine},
		{"pi without name", func(t *testing.T) Report {
			proj := replaceIn(t, piBuilt(t, "1.2.3"), ".pi/agents/gabo-pm.md", "name: gabo-pm\n", "")
			return runPi(t, proj, Settings{})
		}, piLine},
		{"claude main-session file without name", func(t *testing.T) Report {
			proj := replaceIn(t, claudeBuilt(t, "1.2.3"), ".claude/agents/"+assistantID+".md", "name: "+assistantID+"\n", "")
			return run(t, proj, configFiles{projectFile: validConfig}, Settings{})
		}, []string{"✗ agent gabo-assistant (claude): declares no name", "why: .claude/agents/gabo-assistant.md", NextRebuild}},
		{"claude main-session file with another name", func(t *testing.T) Report {
			proj := replaceIn(t, claudeBuilt(t, "1.2.3"), ".claude/agents/"+assistantID+".md", "name: "+assistantID+"\n", "name: assistant\n")
			return run(t, proj, configFiles{projectFile: validConfig}, Settings{})
		}, []string{"✗ agent gabo-assistant (claude): declares name assistant", "why: .claude/agents/gabo-assistant.md", NextRebuild}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := tt.run(t)
			if !r.Failed {
				t.Fatalf("Failed = false, want true:\n%s", joined(r))
			}
			wantConsecutive(t, r, tt.want...)
		})
	}
}

// Spec 0014 S1: a declared name that differs still gives only "declares
// name", never "declares no name".
func TestDeclaredAgentNameDiffersGivesNoMissingLine(t *testing.T) {
	t.Parallel()
	proj := replaceIn(t, codexBuilt(t, "1.2.3"), ".codex/agents/gabo-pm.toml", `name = "gabo-pm"`, `name = "pm"`)
	r := runCodex(t, proj, codexUser(), "1.2.3")
	wantConsecutive(t, r, "✗ agent gabo-pm (codex): declares name pm", "why: .codex/agents/gabo-pm.toml", NextRebuild)
	noLineWith(t, r, "declares no name")
}

// Spec 0014 S1: OpenCode declares no agent_name, so a nameless OpenCode agent
// file gives no name line; a flexible-tier engine without agent_skills is
// skipped (intent 0008 S2) even for a nameless file.
func TestDeclaredAgentNameMissingSkipped(t *testing.T) {
	t.Parallel()
	t.Run("opencode", func(t *testing.T) {
		t.Parallel()
		proj := opencodeBuilt(t, "1.2.3")
		if strings.Contains(string(proj[".opencode/agents/gabo-pm.md"].Data), "\nname:") {
			t.Fatal("opencode fixture declares a name")
		}
		noLineWith(t, runOpencode(t, proj, "1.2.3"), "declares no name")
	})
	t.Run("flexible tier without agent_skills", func(t *testing.T) {
		t.Parallel()
		proj := fstest.MapFS{
			CatalogDir + "/gabo-runtime-zed/SKILL.md": file("---\nname: gabo-runtime-zed\nmetadata:\n  agent_dir: .zed/agents\n  agent_name: frontmatter:name\n---\n"),
			".zed/agents/alpha.md":                    file("---\ngabo-version: 1.2.3\n---\n"),
		}
		var r Report
		(&UseCase{project: proj}).checkAgentName(&r, "alpha", "zed", ".zed/agents/alpha.md")
		if r.Failed || len(r.Lines) != 0 {
			t.Fatalf("lines = %q, failed = %v", r.Lines, r.Failed)
		}
	})
}
