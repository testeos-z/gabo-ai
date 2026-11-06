package check

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// piConfig runs every role on Pi.
const piConfig = `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: pi
  reviewer: pi
  assistant: pi
parallelism: 3
review:
  checks: [go test ./...]
  human_only_paths: [.github/]
`

// piBuilt is a project with the real embedded catalog and every spec agent
// built as .pi/agents/<id>.md with a frontmatter name, a gabo-version line
// and a Skills: line.
func piBuilt(t *testing.T, version string) fstest.MapFS {
	t.Helper()
	proj := fstest.MapFS{".gabo/VERSION": file(version + "\n")}
	files, err := skills.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		proj[path.Join(CatalogDir, f.Path)] = &fstest.MapFile{Data: f.Data}
	}
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		proj[".pi/agents/"+a.ID+".md"] = file("---\nname: " + a.ID + "\ndescription: " + a.Role + "\n---\n" +
			"<!-- gabo-version: " + version + " -->\nSkills: " + strings.Join(a.Skills, ", ") + "\n")
	}
	return proj
}

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

func runPi(t *testing.T, proj fs.FS, s Settings) Report {
	t.Helper()
	return run(t, proj, configFiles{projectFile: piConfig}, s)
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

// S9: under GABO_UPGRADE_PR=1 the name line never prints.
func TestDeclaredAgentNameSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	proj := replaceIn(t, piBuilt(t, "1.2.3"), ".pi/agents/gabo-pm.md", "name: gabo-pm\n", "name: pm\n")
	r := runPi(t, proj, Settings{UpgradePR: true})
	noLineWith(t, r, "declares name")
	wantLines(t, r, UpgradePRLine)
}
