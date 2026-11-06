package check

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// claudeBuilt is a project with the real embedded catalog and every spec
// agent built in the shape gabo-runtime-claude writes: a frontmatter with
// name, description and the gabo-version line, then the filled body; and
// each spec skill linked as .claude/skills/<name> -> ../../.gabo/skills/<name>
// (build step 2).
func claudeBuilt(t *testing.T, version string) fstest.MapFS {
	t.Helper()
	proj := fstest.MapFS{".gabo/VERSION": file(version + "\n"), ".claude/settings.json": file(claudeKeysJSON)}
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
		body, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, a.Template))
		if err != nil {
			t.Fatal(err)
		}
		_, rest, _ := strings.Cut(string(body), "\n")
		proj[".claude/agents/"+a.ID+".md"] = file(claudeFrontmatter(a, string(body), version) + strings.TrimPrefix(rest, "\n"))
		for _, name := range a.Skills {
			proj[".claude/skills/"+name] = symlink("../../.gabo/skills/" + name)
		}
	}
	return proj
}

// claudeTools is the Tool Map of gabo-runtime-claude, with memory wired to
// one MCP server.
var claudeTools = [][2]string{
	{"read", "Read"}, {"search", "Grep, Glob"}, {"edit", "Edit, Write"}, {"shell", "Bash"},
	{"git", "Bash"}, {"tracker", "Bash"}, {"repository-host", "Bash"}, {"memory", "mcp__engram__*"},
	{"web", "WebFetch, WebSearch"}, {"skills", "Skill"}, {"subagents", "Agent"},
}

// claudeFrontmatter writes the frontmatter of build step 6: name,
// single-quoted description, mapped tools, spec skills then the fixed-name
// links the template names, and the unquoted gabo-version.
func claudeFrontmatter(a agentspec.Agent, template, version string) string {
	var tools []string
	for _, m := range claudeTools {
		if slices.Contains(a.Tools, m[0]) {
			for _, tool := range strings.Split(m[1], ", ") {
				if !slices.Contains(tools, tool) {
					tools = append(tools, tool)
				}
			}
		}
	}
	list := slices.Clone(a.Skills)
	for _, l := range [][2]string{{"{{methodology}}", "gabo-methodology"}, {"{{adapter:tasks}}", "gabo-tasks"}, {"{{adapter:repository_host}}", "gabo-repository-host"}, {"{{adapter:memory}}", "gabo-memory"}} {
		if strings.Contains(template, l[0]) {
			list = append(list, l[1])
		}
	}
	var b strings.Builder
	b.WriteString("---\nname: " + a.ID + "\ndescription: '" + strings.ReplaceAll(a.Role, "'", "''") + "'\ntools: " + strings.Join(tools, ", ") + "\n")
	if len(list) > 0 {
		b.WriteString("skills:\n")
		for _, s := range list {
			b.WriteString("  - " + s + "\n")
		}
	}
	b.WriteString("gabo-version: " + version + "\n---\n")
	return b.String()
}

// Every built agent's frontmatter parses as YAML with the keys Claude Code
// needs and gabo-version as the stamp string.
func TestClaudeBuiltFrontmatterParses(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		doc := string(proj[".claude/agents/"+a.ID+".md"].Data)
		head, _, ok := strings.Cut(strings.TrimPrefix(doc, "---\n"), "\n---\n")
		if !ok {
			t.Fatalf("%s: no frontmatter", a.ID)
		}
		var fm struct {
			Name        string   `yaml:"name"`
			Description string   `yaml:"description"`
			Tools       string   `yaml:"tools"`
			Skills      []string `yaml:"skills"`
			Version     string   `yaml:"gabo-version"`
		}
		if err := yaml.Unmarshal([]byte(head), &fm); err != nil {
			t.Fatalf("%s: %v", a.ID, err)
		}
		if fm.Name != a.ID || fm.Description != a.Role || fm.Tools == "" || fm.Version != "1.2.3" {
			t.Errorf("%s: frontmatter %+v", a.ID, fm)
		}
		if slices.Contains(fm.Skills, "gabo-reviewer") {
			t.Errorf("%s: preloads gabo-reviewer", a.ID)
		}
	}
}

// The agent directory declared by the shipped gabo-runtime-claude skill is
// where gabo check finds the agents that skill builds (T052).
func TestClaudeBuiltSetPassesCheck(t *testing.T) {
	t.Parallel()
	r := run(t, claudeBuilt(t, "1.2.3"), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}

// Spec 0009 D4: the main-session agent file beside the spec agents is no
// spec agent, so gabo check reports no missing or extra agent for it.
func TestClaudeBuiltSetWithTheMainSessionAgentPassesCheck(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	if spec.MainSession == nil {
		t.Fatal("embedded spec has no main_session entry")
	}
	body, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, spec.MainSession.Template))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, _ := strings.Cut(string(body), "\n")
	proj[".claude/agents/"+spec.MainSession.ID+".md"] = file("---\nname: " + spec.MainSession.ID +
		"\ndescription: '" + strings.ReplaceAll(spec.MainSession.Role, "'", "''") + "'\nskills:\n  - gabo-assistant\ngabo-version: 1.2.3\n---\n" +
		strings.TrimPrefix(rest, "\n"))
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	for _, line := range r.Lines {
		if strings.Contains(line, "agent") && strings.Contains(line, spec.MainSession.ID) {
			t.Errorf("unexpected line %q", line)
		}
	}
	wantPassed(t, r, PassedLine)
}

func TestClaudeBuiltSetFromAnotherVersionFails(t *testing.T) {
	t.Parallel()
	r := run(t, with(claudeBuilt(t, "1.2.3"), map[string]string{".gabo/VERSION": "1.2.4\n"}), configFiles{projectFile: validConfig}, Settings{Version: "1.2.4"})
	wantFailed(t, r, "✗ agent gabo-coder built with 1.2.3, catalog is 1.2.4: ask your agent to rebuild")
}

// #135: a spec skill without its .claude/skills link (gabo-wave-report) and
// a link left pointing at the CLI source fail the shipped runtime's check.
func TestClaudeBuiltSetWithMissingOrStaleLinksFails(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	delete(proj, ".claude/skills/gabo-wave-report")
	proj[".claude/skills/gabo-coder"] = symlink("../../apps/cli/internal/skills/gabo-coder")
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r,
		"✗ skill link .claude/skills/gabo-wave-report missing",
		"✗ skill link .claude/skills/gabo-coder points to ../../apps/cli/internal/skills/gabo-coder, not ../../.gabo/skills/gabo-coder")
}

// The shipped runtime declares the user-level reviewer link the reviewer
// agent runs through; without it the check fails.
func TestClaudeBuiltSetWithoutReviewerLinkFails(t *testing.T) {
	t.Parallel()
	rep, err := NewUseCase(claudeBuilt(t, "1.2.3"), configFiles{projectFile: validConfig}, userFS{},
		Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
	if err != nil {
		t.Fatal(err)
	}
	wantFailed(t, rep, "✗ reviewer link "+userLink+" missing", NextLink)
}

// setClaudeSkills rewrites the frontmatter skills list of a built agent.
func setClaudeSkills(t *testing.T, proj fstest.MapFS, id string, list []string) {
	t.Helper()
	name := ".claude/agents/" + id + ".md"
	var out []string
	for line := range strings.SplitSeq(string(proj[name].Data), "\n") {
		switch {
		case line == "skills:" || strings.HasPrefix(line, "  - "):
		case strings.HasPrefix(line, "gabo-version:"):
			if len(list) > 0 {
				out = append(out, "skills:")
				for _, s := range list {
					out = append(out, "  - "+s)
				}
			}
			out = append(out, line)
		default:
			out = append(out, line)
		}
	}
	proj[name] = file(strings.Join(out, "\n"))
}

// Intent 0008 S1 on the agents gabo-runtime-claude builds: the frontmatter
// skills list holds exactly the spec skills plus the template's fixed-name
// links, each once; gabo-dev-lead keeps gabo-plan-reviewer throughout.
func TestClaudeBuiltAgentSkills(t *testing.T) {
	t.Parallel()
	for _, tt := range agentSkillsCases("claude") {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proj := claudeBuilt(t, "1.2.3")
			setClaudeSkills(t, proj, "gabo-pm", tt.list)
			r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
			wantSkillLines(t, r, ".claude/agents/gabo-pm.md", tt.want)
		})
	}
}

// The adapter's frontmatter name passes in place of its fixed-name link.
func TestClaudeBuiltAgentSkillsAcceptAdapterName(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	slots := memoryAdapterSlot(t, proj, ".claude/skills")
	setClaudeSkills(t, proj, "gabo-pm", []string{"gabo-pm", "gabo-plan-reviewer", "gabo-tasks", "gabo-repository-host", "gabo-memory-engram"})
	r := run(t, proj, configFiles{projectFile: validConfig + "slots:\n" + slots}, Settings{})
	wantSkillLines(t, r, ".claude/agents/gabo-pm.md", nil)
	wantPassed(t, r, PassedLine)
}

// The fixed-name links come from the template in the project catalog copy:
// a gabo-pm template without {{adapter:memory}} makes gabo-memory extra.
func TestClaudeBuiltAgentSkillsFollowTheProjectTemplate(t *testing.T) {
	t.Parallel()
	proj := withoutMemorySlot(t, claudeBuilt(t, "1.2.3"))
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantSkillLines(t, r, ".claude/agents/gabo-pm.md", []string{"✗ agent gabo-pm (claude): skill gabo-memory not in the agent spec"})
}
