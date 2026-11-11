package agentspec

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

const validSpec = `agents:
  - id: lead
    role: Plans the work.
    skills: [skill-a]
    tools: [read, tracker]
    launched_by: main-session
    template: agents/lead.md
  - id: worker
    role: Does the work.
    skills: [skill-a, skill-b]
    tools: [read, edit, git]
    launched_by: lead
    template: agents/worker.md
  - id: checker
    role: Checks the work.
    skills: []
    tools: [shell]
    launched_by: worker
    template: agents/checker.md
`

// catalog builds a catalog root holding the spec, its templates and two skills.
func catalog(spec string, drop ...string) fstest.MapFS {
	fsys := fstest.MapFS{
		File:                       {Data: []byte(spec)},
		Dir + "/agents/lead.md":    {Data: []byte("lead")},
		Dir + "/agents/worker.md":  {Data: []byte("worker")},
		Dir + "/agents/checker.md": {Data: []byte("checker")},
		"skill-a/SKILL.md":         {Data: []byte("a")},
		"skill-b/SKILL.md":         {Data: []byte("b")},
		"not-a-skill/README.md":    {Data: []byte("x")},
	}
	for _, p := range drop {
		delete(fsys, p)
	}
	return fsys
}

func TestLoadValidSpec(t *testing.T) {
	t.Parallel()
	spec, err := Load(catalog(validSpec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := spec.IDs(); !slices.Equal(got, []string{"lead", "worker", "checker"}) {
		t.Fatalf("IDs() = %v", got)
	}
	w := spec.Agents[1]
	if w.Role != "Does the work." || w.LaunchedBy != "lead" || w.Template != "agents/worker.md" ||
		!slices.Equal(w.Skills, []string{"skill-a", "skill-b"}) || !slices.Equal(w.Tools, []string{"read", "edit", "git"}) {
		t.Fatalf("worker = %+v", w)
	}
}

func TestLoadRejectsInvalidSpecs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		spec string
		drop []string
		want string
	}{
		{"missing file", validSpec, []string{File}, "read " + File},
		{"not yaml", "agents: [", nil, "parse " + File},
		{"unknown field", strings.Replace(validSpec, "    role: Plans the work.\n", "    role: Plans the work.\n    model: big\n", 1), nil, "field model not found"},
		{"no agents", "agents: []\n", nil, "no agents"},
		{"missing id", strings.Replace(validSpec, "id: checker", "id: ''", 1), nil, "agent 3: missing id"},
		{"missing role", strings.Replace(validSpec, "role: Checks the work.", "role: ''", 1), nil, "agent checker: missing role"},
		{"missing tools", strings.Replace(validSpec, "tools: [shell]", "tools: []", 1), nil, "agent checker: missing tools"},
		{"missing launched_by", strings.Replace(validSpec, "launched_by: worker", "launched_by: ''", 1), nil, "agent checker: missing launched_by"},
		{"missing template", strings.Replace(validSpec, "template: agents/checker.md", "template: ''", 1), nil, "agent checker: missing template"},
		{"duplicate id", strings.Replace(validSpec, "id: checker", "id: worker", 1), nil, "agent worker: duplicate id"},
		{"launched_by unknown", strings.Replace(validSpec, "launched_by: worker", "launched_by: ghost", 1), nil, `agent checker: launched_by "ghost" is not a spec agent or main-session`},
		{"template missing", validSpec, []string{Dir + "/agents/checker.md"}, `agent checker: template "agents/checker.md" not found`},
		{"template escapes", strings.Replace(validSpec, "template: agents/checker.md", "template: ../skill-a/SKILL.md", 1), nil, `agent checker: template "../skill-a/SKILL.md" must be a relative path inside ` + Dir},
		{"template absolute", strings.Replace(validSpec, "template: agents/checker.md", "template: /etc/passwd", 1), nil, `agent checker: template "/etc/passwd" must be a relative path inside ` + Dir},
		{"skill unknown", strings.Replace(validSpec, "skills: []", "skills: [skill-z]", 1), nil, `agent checker: skill "skill-z" not in the catalog`},
		{"skill without SKILL.md", strings.Replace(validSpec, "skills: []", "skills: [not-a-skill]", 1), nil, `agent checker: skill "not-a-skill" not in the catalog`},
		{"self launch", strings.Replace(validSpec, "launched_by: worker", "launched_by: checker", 1), nil, "launched_by cycle: checker -> checker"},
		{"cycle", strings.Replace(validSpec, "launched_by: main-session", "launched_by: checker", 1), nil, "launched_by cycle: lead -> checker -> worker -> lead"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(catalog(tt.spec, tt.drop...))
			if err == nil {
				t.Fatalf("Load succeeded, want error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q\nwant it to contain %q", err, tt.want)
			}
		})
	}
}

func TestLoadReportsEveryProblem(t *testing.T) {
	t.Parallel()
	spec := strings.Replace(validSpec, "skills: []", "skills: [skill-z]", 1)
	spec = strings.Replace(spec, "role: Plans the work.", "role: ''", 1)
	_, err := Load(catalog(spec))
	if err == nil {
		t.Fatal("Load succeeded")
	}
	for _, want := range []string{"agent lead: missing role", `agent checker: skill "skill-z" not in the catalog`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q misses %q", err, want)
		}
	}
}

const mainSessionSpec = `main_session:
  id: assistant
  role: Talks to the owner.
  skills: [skill-a]
  template: agents/assistant.md
  block: agents/assistant-block.md
`

// mainCatalog is catalog with a main_session entry and its two files.
func mainCatalog(entry string, drop ...string) fstest.MapFS {
	fsys := catalog(validSpec + entry)
	fsys[Dir+"/agents/assistant.md"] = &fstest.MapFile{Data: []byte("assistant")}
	fsys[Dir+"/agents/assistant-block.md"] = &fstest.MapFile{Data: []byte("block")}
	for _, p := range drop {
		delete(fsys, p)
	}
	return fsys
}

// Spec 0009 D4: the main-session entry parses beside the agents and is never
// one of them.
func TestLoadMainSessionEntry(t *testing.T) {
	t.Parallel()
	spec, err := Load(mainCatalog(mainSessionSpec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	m := spec.MainSession
	if m == nil {
		t.Fatal("MainSession is nil")
	}
	if m.ID != "assistant" || m.Role != "Talks to the owner." || !slices.Equal(m.Skills, []string{"skill-a"}) ||
		m.Template != "agents/assistant.md" || m.Block != "agents/assistant-block.md" {
		t.Fatalf("MainSession = %+v", *m)
	}
	if got := spec.IDs(); !slices.Equal(got, []string{"lead", "worker", "checker"}) {
		t.Fatalf("IDs() = %v", got)
	}
	if len(spec.Agents) != 3 {
		t.Fatalf("Agents has %d entries, want 3", len(spec.Agents))
	}
}

func TestLoadWithoutMainSessionEntry(t *testing.T) {
	t.Parallel()
	spec, err := Load(catalog(validSpec))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if spec.MainSession != nil {
		t.Fatalf("MainSession = %+v, want nil", *spec.MainSession)
	}
}

func TestLoadRejectsInvalidMainSessionEntries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		entry string
		drop  []string
		want  string
	}{
		{"unknown key", strings.Replace(mainSessionSpec, "  role:", "  model: big\n  role:", 1), nil, "field model not found"},
		{"missing id", strings.Replace(mainSessionSpec, "id: assistant", "id: ''", 1), nil, "main_session: missing id"},
		{"id of a spec agent", strings.Replace(mainSessionSpec, "id: assistant", "id: worker", 1), nil, "main_session: id worker is a spec agent"},
		{"missing role", strings.Replace(mainSessionSpec, "role: Talks to the owner.", "role: ''", 1), nil, "main_session: missing role"},
		{"skill unknown", strings.Replace(mainSessionSpec, "skills: [skill-a]", "skills: [skill-z]", 1), nil, `main_session: skill "skill-z" not in the catalog`},
		{"missing template", strings.Replace(mainSessionSpec, "template: agents/assistant.md", "template: ''", 1), nil, "main_session: missing template"},
		{"template not found", mainSessionSpec, []string{Dir + "/agents/assistant.md"}, `main_session: template "agents/assistant.md" not found`},
		{"template escapes", strings.Replace(mainSessionSpec, "template: agents/assistant.md", "template: ../skill-a/SKILL.md", 1), nil, `main_session: template "../skill-a/SKILL.md" must be a relative path inside ` + Dir},
		{"missing block", strings.Replace(mainSessionSpec, "block: agents/assistant-block.md", "block: ''", 1), nil, "main_session: missing block"},
		{"block not found", mainSessionSpec, []string{Dir + "/agents/assistant-block.md"}, `main_session: block "agents/assistant-block.md" not found`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Load(mainCatalog(tt.entry, tt.drop...))
			if err == nil {
				t.Fatalf("Load succeeded, want error containing %q", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q\nwant it to contain %q", err, tt.want)
			}
		})
	}
}

// The spec shipped in the embedded catalog lists every gabo agent that runs
// from an agent file (TRD 4.3) and validates against that catalog.
func TestEmbeddedSpecIsValid(t *testing.T) {
	t.Parallel()
	spec, err := Load(skills.Catalog())
	if err != nil {
		t.Fatalf("embedded spec: %v", err)
	}
	want := []string{
		"gabo-pm", "gabo-dev-lead", "gabo-db-master", "gabo-frontend",
		"gabo-plan-reviewer", "gabo-coder",
	}
	if got := spec.IDs(); !slices.Equal(got, want) {
		t.Fatalf("IDs() = %v\nwant  %v", got, want)
	}
	for _, a := range spec.Agents {
		if a.Template != "agents/"+a.ID+".md" {
			t.Errorf("agent %s: template %q, want agents/%s.md", a.ID, a.Template, a.ID)
		}
		if a.LaunchedBy != MainSession {
			t.Errorf("agent %s: launched_by %q, want %s", a.ID, a.LaunchedBy, MainSession)
		}
		if slices.Contains(a.Tools, "subagents") {
			t.Errorf("agent %s: tools %v has subagents", a.ID, a.Tools)
		}
	}
	m := spec.MainSession
	if m == nil {
		t.Fatal("embedded spec has no main_session entry")
	}
	if m.ID != "gabo-assistant" || !slices.Equal(m.Skills, []string{"gabo-assistant", "gabo-cycle", "gabo-wave-report"}) ||
		m.Template != "agents/gabo-assistant.md" || m.Block != "agents/gabo-assistant-block.md" {
		t.Fatalf("main_session = %+v", *m)
	}
}

// The project copy under .gabo/skills/ never holds the reviewer skill, so the
// embedded catalog root must not expose it either.
func TestEmbeddedCatalogHidesTheReviewerSkill(t *testing.T) {
	t.Parallel()
	spec := strings.Replace(validSpec, "skills: []", "skills: [gabo-reviewer]", 1)
	fsys := catalog(spec)
	fsys["gabo-reviewer/SKILL.md"] = &fstest.MapFile{Data: []byte("r")}
	if _, err := Load(fsys); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	if _, err := skills.Catalog().Open("gabo-reviewer/SKILL.md"); err == nil {
		t.Fatal("Catalog() exposes gabo-reviewer/SKILL.md")
	}
}
