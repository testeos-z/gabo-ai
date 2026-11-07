package check

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// pmSkills is what a runtime skill writes for gabo-pm from the real catalog:
// the spec skills, then the fixed-name links its template names.
var pmSkills = []string{"gabo-pm", "gabo-plan-reviewer", "gabo-tasks", "gabo-repository-host", "gabo-memory"}

// agentSkillsCase edits the skill list of the gabo-pm agent file; want holds
// the per-agent skill lines the check prints for it (none: it passes).
type agentSkillsCase struct {
	name string
	list []string
	want []string
}

// agentSkillsCases are the intent 0008 S1 cases for one runtime.
func agentSkillsCases(rt string) []agentSkillsCase {
	line := func(s string) string { return "✗ agent gabo-pm (" + rt + "): skill " + s }
	return []agentSkillsCase{
		{name: "as built", list: pmSkills},
		{name: "any order", list: []string{"gabo-memory", "gabo-tasks", "gabo-plan-reviewer", "gabo-repository-host", "gabo-pm"}},
		{name: "missing", list: slices.DeleteFunc(slices.Clone(pmSkills), func(s string) bool { return s == "gabo-plan-reviewer" }),
			want: []string{line("gabo-plan-reviewer missing")}},
		{name: "missing fixed-name link", list: pmSkills[:4], want: []string{line("gabo-memory missing")}},
		{name: "extra", list: append(slices.Clone(pmSkills), "gabo-frontend"),
			want: []string{line("gabo-frontend not in the agent spec")}},
		// Spec 0014 S2: a spec skill listed more than once gets one "listed
		// twice" line per name, between the missing and the extra lines.
		{name: "duplicate", list: append(slices.Clone(pmSkills), "gabo-pm"),
			want: []string{line("gabo-pm listed twice")}},
		{name: "three copies", list: append(slices.Clone(pmSkills), "gabo-pm", "gabo-pm"),
			want: []string{line("gabo-pm listed twice")}},
		{name: "duplicate and extra", list: append(slices.Clone(pmSkills), "gabo-frontend", "gabo-tasks"),
			want: []string{line("gabo-tasks listed twice"), line("gabo-frontend not in the agent spec")}},
		{name: "missing and duplicate", list: append(slices.DeleteFunc(slices.Clone(pmSkills), func(s string) bool { return s == "gabo-plan-reviewer" }), "gabo-pm"),
			want: []string{line("gabo-plan-reviewer missing"), line("gabo-pm listed twice")}},
		{name: "extra twice", list: append(slices.Clone(pmSkills), "gabo-frontend", "gabo-frontend"),
			want: []string{line("gabo-frontend not in the agent spec"), line("gabo-frontend not in the agent spec")}},
		{name: "empty", list: nil, want: []string{
			line("gabo-pm missing"), line("gabo-plan-reviewer missing"), line("gabo-tasks missing"),
			line("gabo-repository-host missing"), line("gabo-memory missing"),
		}},
	}
}

// skillLines returns the per-agent skill lines of a report, in order.
func skillLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		if strings.HasPrefix(l, "✗ agent ") && strings.Contains(l, "): skill ") {
			out = append(out, l)
		}
	}
	return out
}

// wantSkillLines requires exactly the want skill lines, each followed by the
// agent file and the rebuild step, and exit 1 when there is any.
func wantSkillLines(t *testing.T, r Report, file string, want []string) {
	t.Helper()
	if got := skillLines(r); !slices.Equal(got, want) {
		t.Fatalf("skill lines = %q, want %q; output:\n%s", got, want, joined(r))
	}
	if len(want) == 0 {
		if r.Failed {
			t.Fatalf("Failed = true, want false; output:\n%s", joined(r))
		}
		return
	}
	if !r.Failed {
		t.Fatalf("Failed = false, want true; output:\n%s", joined(r))
	}
	for i, l := range r.Lines {
		if slices.Contains(want, l) && (i+2 >= len(r.Lines) || r.Lines[i+1] != "why: "+file || r.Lines[i+2] != NextRebuild) {
			t.Fatalf("%q not followed by why: %s and %q:\n%s", l, file, NextRebuild, joined(r))
		}
	}
}

// withoutMemorySlot rewrites the project's gabo-pm template so that it no
// longer names {{adapter:memory}}.
func withoutMemorySlot(t *testing.T, proj fstest.MapFS) fstest.MapFS {
	t.Helper()
	name := path.Join(CatalogDir, "gabo-agent-templates", "agents", "gabo-pm.md")
	tmpl, err := fs.ReadFile(proj, name)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tmpl), "{{adapter:memory}}") {
		t.Fatalf("%s names no {{adapter:memory}}", name)
	}
	return with(proj, map[string]string{name: strings.ReplaceAll(string(tmpl), "{{adapter:memory}}", "memory")})
}

// memoryAdapterSlot adds a memory slot adapter skill named
// gabo-memory-engram in skillDir with its fixed-name link gabo-memory, and
// returns the config lines that select it.
func memoryAdapterSlot(t *testing.T, proj fstest.MapFS, skillDir string) string {
	t.Helper()
	contract, err := fs.ReadFile(proj, path.Join(CatalogDir, "gabo-contract-memory", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	adapter := path.Join(skillDir, "gabo-memory-engram", "SKILL.md")
	proj[adapter] = file("---\nname: gabo-memory-engram\n---\n" + string(contract))
	proj[path.Join(skillDir, "gabo-memory")] = symlink("gabo-memory-engram")
	return "  memory:\n    adapter: " + adapter + "\n"
}

// Under GABO_UPGRADE_PR=1 no per-agent skill line prints (intent 0008 S9),
// a skill listed twice included (spec 0014 S3).
func TestAgentSkillsSkippedForUpgradePR(t *testing.T) {
	t.Parallel()
	for name, list := range map[string][]string{"empty": nil, "duplicate": append(slices.Clone(pmSkills), "gabo-pm")} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			proj := claudeBuilt(t, "1.2.3")
			setClaudeSkills(t, proj, "gabo-pm", list)
			r := run(t, proj, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
			if got := skillLines(r); len(got) != 0 {
				t.Fatalf("skill lines under GABO_UPGRADE_PR=1: %q", got)
			}
			wantLines(t, r, UpgradePRLine)
		})
	}
}

// Spec 0014 S2 (open question 2): a fixed-name link listed under its link
// name and its adapter's name is listed twice, named as the second entry
// writes it.
func TestAgentSkillsLinkAndAdapterNameListedTwice(t *testing.T) {
	t.Parallel()
	list := append(slices.Clone(pmSkills), "gabo-memory-engram")
	t.Run("claude", func(t *testing.T) {
		t.Parallel()
		proj := claudeBuilt(t, "1.2.3")
		slots := memoryAdapterSlot(t, proj, ".claude/skills")
		setClaudeSkills(t, proj, "gabo-pm", list)
		r := run(t, proj, configFiles{projectFile: validConfig + "slots:\n" + slots}, Settings{})
		wantSkillLines(t, r, ".claude/agents/gabo-pm.md", []string{"✗ agent gabo-pm (claude): skill gabo-memory-engram listed twice"})
	})
	t.Run("codex", func(t *testing.T) {
		t.Parallel()
		proj := codexBuilt(t, "1.2.3")
		slots := memoryAdapterSlot(t, proj, ".agents/skills")
		setCodexSkills(t, proj, "gabo-pm", list)
		r, err := NewUseCase(proj, configFiles{projectFile: codexConfig + slots}, codexUser(),
			Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
		if err != nil {
			t.Fatal(err)
		}
		wantSkillLines(t, r, ".codex/agents/gabo-pm.toml", []string{"✗ agent gabo-pm (codex): skill gabo-memory-engram listed twice"})
	})
}

// A runtime skill without metadata.agent_skills gets no per-agent skill line
// (checkAgentMeta reports it instead).
func TestAgentSkillsNeedTheDeclaration(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	setClaudeSkills(t, proj, "gabo-pm", nil)
	skill := path.Join(CatalogDir, "gabo-runtime-claude", "SKILL.md")
	b, err := fs.ReadFile(proj, skill)
	if err != nil {
		t.Fatal(err)
	}
	proj = with(proj, map[string]string{skill: strings.Replace(string(b), "  agent_skills: frontmatter:skills\n", "", 1)})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	if got := skillLines(r); len(got) != 0 {
		t.Fatalf("skill lines without metadata.agent_skills: %q", got)
	}
}

func TestCompareSkills(t *testing.T) {
	t.Parallel()
	want := []expectedSkill{{name: "a"}, {name: "gabo-memory", alt: "gabo-memory-engram"}}
	tests := []struct {
		name               string
		got                []string
		missing, dup, extr []string
	}{
		{name: "exact", got: []string{"a", "gabo-memory"}},
		{name: "adapter name", got: []string{"gabo-memory-engram", "a"}},
		{name: "link and adapter name", got: []string{"a", "gabo-memory", "gabo-memory-engram"}, dup: []string{"gabo-memory-engram"}},
		{name: "duplicate", got: []string{"a", "a", "gabo-memory"}, dup: []string{"a"}},
		{name: "three copies", got: []string{"a", "a", "gabo-memory", "a"}, dup: []string{"a"}},
		{name: "duplicates in list order", got: []string{"gabo-memory", "a", "gabo-memory-engram", "a"}, dup: []string{"gabo-memory-engram", "a"}},
		{name: "extra twice", got: []string{"a", "gabo-memory", "b", "b"}, extr: []string{"b", "b"}},
		{name: "missing and extra", got: []string{"b"}, missing: []string{"a", "gabo-memory"}, extr: []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			missing, dup, extra := compareSkills(want, tt.got)
			if !slices.Equal(missing, tt.missing) || !slices.Equal(dup, tt.dup) || !slices.Equal(extra, tt.extr) {
				t.Fatalf("compareSkills(%q) = %q, %q, %q; want %q, %q, %q", tt.got, missing, dup, extra, tt.missing, tt.dup, tt.extr)
			}
		})
	}
}

func TestTemplateLinks(t *testing.T) {
	t.Parallel()
	got := templateLinks([]byte("x {{methodology}} `{{adapter:repository_host}}` {{adapter:tasks}} {{methodology}} {{id}} {{reviewer_script}}"))
	want := []templateLink{{slot: "methodology", name: "gabo-methodology"}, {slot: "repository_host", name: "gabo-repository-host"}, {slot: "tasks", name: "gabo-tasks"}}
	if !slices.Equal(got, want) {
		t.Fatalf("templateLinks = %+v, want %+v", got, want)
	}
}
