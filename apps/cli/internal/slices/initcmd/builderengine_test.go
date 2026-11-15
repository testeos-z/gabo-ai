package initcmd

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

// builderSkill is the runtime skill a runtime builder writes for the engine
// foo (intent 0017 D1 (a)).
const builderSkill = "---\nname: gabo-runtime-foo\nmetadata:\n  agent_dir: .foo/agents\n---\nbody\n"

// builderProject is a project whose runtimes.coder is coder, reviewer and
// assistant are claude, and whose runtime skill for foo is in .gabo/runtimes.
func builderProject(t *testing.T, coder string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(ConfigFile, "runtimes:\n  coder: "+coder+"\n  reviewer: claude\n  assistant: claude\n")
	write(".gabo/runtimes/gabo-runtime-foo/SKILL.md", builderSkill)
	return dir
}

// projectTree returns every file under dir by slash path, with its bytes,
// except gabo.yaml, which differs by the runtime name.
func projectTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == ConfigFile {
			return nil
		}
		b, err := os.ReadFile(p)
		out[rel] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// S4 (intent 0017): gabo init prints the flexible-tier line for a builder
// engine and writes nothing for it: the project tree equals the one a
// gabo.yaml naming claude in the same role gets.
func TestInitWritesNothingForABuilderEngine(t *testing.T) {
	t.Parallel()
	foo := builderProject(t, "foo")
	claude := builderProject(t, "claude")
	out, err := runCommand(t, foo)
	if err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	if n := strings.Count(out, "foo: flexible tier: skills and configuration only; nothing written"); n != 1 {
		t.Fatalf("flexible-tier line %d times, want once:\n%s", n, out)
	}
	if _, err := runCommand(t, claude); err != nil {
		t.Fatalf("init with claude: %v", err)
	}
	if got, want := projectTree(t, foo), projectTree(t, claude); !maps.Equal(got, want) {
		t.Fatalf("project tree for foo differs from the claude tree:\nfoo: %v\nclaude: %v", keysOf(got), keysOf(want))
	}
}

// The four named engines keep the full tier in gabo init: none is reported
// as flexible tier, and each one writes its own file (intent 0006 S2, S3, S7).
func TestBuilderEngineLeavesTheNamedEnginesFullTier(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := engineStep(s, engineconfig.Set("claude", "codex", "pi", "opencode")...)
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	if strings.Contains(r.Render(), "flexible tier") {
		t.Fatalf("a named engine reported as flexible tier:\n%s", r.Render())
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
