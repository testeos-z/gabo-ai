package check

import (
	"io/fs"
	"path"
	"strings"
	"testing"
	"testing/fstest"

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
		front := "---\nname: " + a.ID + "\ndescription: '" + strings.ReplaceAll(a.Role, "'", "''") + "'\ntools: Bash\ngabo-version: " + version + "\n---\n"
		proj[".claude/agents/"+a.ID+".md"] = file(front + strings.TrimPrefix(rest, "\n"))
	}
	return proj
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
