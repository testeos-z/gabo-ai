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
// name, description and the gabo-version line, then the filled body.
func claudeBuilt(t *testing.T, version string) fstest.MapFS {
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
		body, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, a.Template))
		if err != nil {
			t.Fatal(err)
		}
		_, rest, _ := strings.Cut(string(body), "\n")
		proj[".claude/agents/"+a.ID+".md"] = file(claudeFrontmatter(a, string(body), version) + strings.TrimPrefix(rest, "\n"))
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

func TestClaudeBuiltSetFromAnotherVersionFails(t *testing.T) {
	t.Parallel()
	r := run(t, with(claudeBuilt(t, "1.2.3"), map[string]string{".gabo/VERSION": "1.2.4\n"}), configFiles{projectFile: validConfig}, Settings{Version: "1.2.4"})
	wantFailed(t, r, "✗ agent gabo-coder built with 1.2.3, catalog is 1.2.4: ask your agent to rebuild")
}
