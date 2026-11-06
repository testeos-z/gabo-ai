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
		{name: "duplicate", list: append(slices.Clone(pmSkills), "gabo-pm"),
			want: []string{line("gabo-pm not in the agent spec")}},
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

// Under GABO_UPGRADE_PR=1 no per-agent skill line prints (intent 0008 S9).
func TestAgentSkillsSkippedForUpgradePR(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	setClaudeSkills(t, proj, "gabo-pm", nil)
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	if got := skillLines(r); len(got) != 0 {
		t.Fatalf("skill lines under GABO_UPGRADE_PR=1: %q", got)
	}
	wantLines(t, r, UpgradePRLine)
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
		name          string
		got           []string
		missing, extr []string
	}{
		{name: "exact", got: []string{"a", "gabo-memory"}},
		{name: "adapter name", got: []string{"gabo-memory-engram", "a"}},
		{name: "link and adapter name", got: []string{"a", "gabo-memory", "gabo-memory-engram"}, extr: []string{"gabo-memory-engram"}},
		{name: "duplicate", got: []string{"a", "a", "gabo-memory"}, extr: []string{"a"}},
		{name: "missing and extra", got: []string{"b"}, missing: []string{"a", "gabo-memory"}, extr: []string{"b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			missing, extra := compareSkills(want, tt.got)
			if !slices.Equal(missing, tt.missing) || !slices.Equal(extra, tt.extr) {
				t.Fatalf("compareSkills(%q) = %q, %q; want %q, %q", tt.got, missing, extra, tt.missing, tt.extr)
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
