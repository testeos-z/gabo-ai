package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

// repoRoot is the repository root seen from this package directory.
const repoRoot = "../../../.."

// pluginDir is the plugin tree the marketplace entry points at.
const pluginDir = "plugins/claude-code"

// pluginExcluded are the catalog skills that run reviews; the plugin never
// ships them (intent 0001 S3, D20).
var pluginExcluded = []string{"gabo-reviewer", "gabo-reviewer-placement-local"}

func readJSON(t *testing.T, rel string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, rel))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("%s: %v", rel, err)
	}
	return m
}

// The repository is the Claude Code marketplace gabo-ai listing exactly one
// plugin gabo at ./plugins/claude-code, with no version in the entry
// (intent 0001 S1, S5, D1, D2).
func TestMarketplaceListsOnlyThePlugin(t *testing.T) {
	t.Parallel()
	m := readJSON(t, ".claude-plugin/marketplace.json")
	if m["name"] != "gabo-ai" {
		t.Fatalf("marketplace name %v, want gabo-ai", m["name"])
	}
	plugins, ok := m["plugins"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("marketplace plugins %v, want exactly one entry", m["plugins"])
	}
	entry, ok := plugins[0].(map[string]any)
	if !ok {
		t.Fatalf("marketplace entry %v is not an object", plugins[0])
	}
	if entry["name"] != "gabo" || entry["source"] != "./"+pluginDir {
		t.Fatalf("marketplace entry name %v source %v, want gabo ./%s", entry["name"], entry["source"], pluginDir)
	}
	if _, has := entry["version"]; has {
		t.Fatalf("marketplace entry declares version %v; the version lives in plugin.json only (S5)", entry["version"])
	}
}

// plugin.json names the plugin gabo, carries the bare release version x.y.z
// that gabo init writes to .gabo/VERSION (S5, D5), and declares no agents,
// commands or MCP servers (S4, D6, D7).
func TestPluginManifest(t *testing.T) {
	t.Parallel()
	p := readJSON(t, pluginDir+"/.claude-plugin/plugin.json")
	if p["name"] != "gabo" {
		t.Fatalf("plugin name %v, want gabo", p["name"])
	}
	v, _ := p["version"].(string)
	if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(v) {
		t.Fatalf("plugin version %q is not a bare release version x.y.z", v)
	}
	for _, key := range []string{"agents", "commands", "mcpServers"} {
		if _, has := p[key]; has {
			t.Fatalf("plugin.json declares %s; the plugin ships none (S4)", key)
		}
	}
}

// The plugin tree has no agents, commands, bin or MCP config (S4).
func TestPluginShipsNoAgentsCommandsBinOrMCP(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"agents", "commands", "bin", ".mcp.json"} {
		if _, err := os.Lstat(filepath.Join(repoRoot, pluginDir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s/%s exists (err %v); the plugin ships none (S4)", pluginDir, name, err)
		}
	}
}

// The plugin skills are exactly the catalog minus the review-running skills,
// each a relative symlink into the catalog under its catalog name, so the
// plugin carries the catalog byte for byte (S3, D3).
func TestPluginSkillsLinkTheCatalog(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(repoRoot, pluginDir, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	var want []string
	for _, n := range Names() {
		if !slices.Contains(pluginExcluded, n) {
			want = append(want, n)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("plugin skills\n got %v\nwant %v", got, want)
	}
	if !slices.Contains(got, "gabo-reviewer-config") {
		t.Fatal("plugin skills miss gabo-reviewer-config (S10)")
	}

	catalog := absReal(t, ".")
	for _, name := range got {
		link := filepath.Join(dir, name)
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("%s is not a symlink: %v", name, err)
		}
		if filepath.IsAbs(target) {
			t.Fatalf("%s links to absolute %s, want a relative link", name, target)
		}
		resolved := absReal(t, link)
		if want := filepath.Join(catalog, name); resolved != want {
			t.Fatalf("%s resolves to %s, want %s", name, resolved, want)
		}
		if _, err := os.Stat(filepath.Join(link, "SKILL.md")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	for _, name := range pluginExcluded {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("plugin ships excluded skill %s (err %v)", name, err)
		}
	}
}

// absReal returns the absolute path of p with every symlink resolved.
func absReal(t *testing.T, p string) string {
	t.Helper()
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		t.Fatal(err)
	}
	return real
}
