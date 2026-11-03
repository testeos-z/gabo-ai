package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// codexPluginDir is the Codex plugin tree the Codex marketplace entry points
// at, beside the Claude Code plugin (intent 0002 D4).
const codexPluginDir = "plugins/codex"

// The repository is the Codex marketplace gabo-ai listing exactly one plugin
// gabo with a local source at ./plugins/codex and no version in the entry
// (intent 0002 S2, D3, D8).
func TestCodexMarketplaceListsOnlyThePlugin(t *testing.T) {
	t.Parallel()
	m := readJSON(t, ".agents/plugins/marketplace.json")
	if m["name"] != "gabo-ai" {
		t.Fatalf("Codex marketplace name %v, want gabo-ai", m["name"])
	}
	plugins, ok := m["plugins"].([]any)
	if !ok || len(plugins) != 1 {
		t.Fatalf("Codex marketplace plugins %v, want exactly one entry", m["plugins"])
	}
	entry, ok := plugins[0].(map[string]any)
	if !ok {
		t.Fatalf("Codex marketplace entry %v is not an object", plugins[0])
	}
	if entry["name"] != "gabo" {
		t.Fatalf("Codex marketplace entry name %v, want gabo", entry["name"])
	}
	if got := codexSourcePath(t, entry); got != "./"+codexPluginDir {
		t.Fatalf("Codex marketplace entry source path %q, want ./%s", got, codexPluginDir)
	}
	if _, has := entry["version"]; has {
		t.Fatalf("Codex marketplace entry declares version %v; the version lives in plugin.json only (D8)", entry["version"])
	}
}

// codexSourcePath returns the path of a Codex marketplace entry's local
// source object ({"source": "local", "path": "./..."}).
func codexSourcePath(t *testing.T, entry map[string]any) string {
	t.Helper()
	src, ok := entry["source"].(map[string]any)
	if !ok {
		t.Fatalf("Codex marketplace entry source %v is not a source object", entry["source"])
	}
	if src["source"] != "local" {
		t.Fatalf("Codex marketplace entry source kind %v, want local", src["source"])
	}
	p, _ := src["path"].(string)
	return p
}

// Both marketplace files name the one plugin gabo, each with the source path
// of its own tool's plugin directory (intent 0002 D3).
func TestBothMarketplacesNameTheSamePlugin(t *testing.T) {
	t.Parallel()
	claude := readJSON(t, ".claude-plugin/marketplace.json")
	codex := readJSON(t, ".agents/plugins/marketplace.json")
	if claude["name"] != codex["name"] {
		t.Fatalf("marketplace names differ: Claude Code %v, Codex %v", claude["name"], codex["name"])
	}
	ce, _ := claude["plugins"].([]any)
	xe, _ := codex["plugins"].([]any)
	if len(ce) != 1 || len(xe) != 1 {
		t.Fatalf("marketplaces list %d and %d plugins, want one each", len(ce), len(xe))
	}
	c, _ := ce[0].(map[string]any)
	x, _ := xe[0].(map[string]any)
	if c["name"] != "gabo" || x["name"] != "gabo" {
		t.Fatalf("marketplace plugin names Claude Code %v, Codex %v, want gabo in both", c["name"], x["name"])
	}
	if c["source"] != "./"+pluginDir {
		t.Fatalf("Claude Code marketplace source %v, want ./%s", c["source"], pluginDir)
	}
	if got := codexSourcePath(t, x); got != "./"+codexPluginDir {
		t.Fatalf("Codex marketplace source path %q, want ./%s", got, codexPluginDir)
	}
}

// The Codex manifest names the plugin gabo, carries the bare release version
// x.y.z equal to the Claude Code manifest (the CLI release version, bumped
// together on release), points skills at ./skills/ inside the plugin root and
// declares no apps or MCP servers (intent 0002 S1, S11, D8).
func TestCodexPluginManifest(t *testing.T) {
	t.Parallel()
	p := readJSON(t, codexPluginDir+"/.codex-plugin/plugin.json")
	if p["name"] != "gabo" {
		t.Fatalf("Codex plugin name %v, want gabo", p["name"])
	}
	v, _ := p["version"].(string)
	if !regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`).MatchString(v) {
		t.Fatalf("Codex plugin version %q is not a bare release version x.y.z", v)
	}
	claude := readJSON(t, pluginDir+"/.claude-plugin/plugin.json")
	if v != claude["version"] {
		t.Fatalf("Codex plugin version %q differs from the Claude Code plugin version %v (S11)", v, claude["version"])
	}
	if p["skills"] != "./skills/" {
		t.Fatalf("Codex plugin skills %v, want ./skills/", p["skills"])
	}
	root := absReal(t, filepath.Join(repoRoot, codexPluginDir))
	skills := absReal(t, filepath.Join(root, "skills"))
	if !strings.HasPrefix(skills, root+string(filepath.Separator)) {
		t.Fatalf("Codex plugin skills path %s resolves outside the plugin root %s", skills, root)
	}
	for _, key := range []string{"apps", "mcpServers"} {
		if _, has := p[key]; has {
			t.Fatalf("Codex plugin.json declares %s; the plugin ships none (S1)", key)
		}
	}
	if _, err := os.Lstat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("%s/.mcp.json exists (err %v); the plugin ships no MCP servers (S1)", codexPluginDir, err)
	}
}

// The Codex plugin skills directory lists exactly the entries of the Claude
// Code plugin skills directory, and each entry is a relative symlink that
// resolves to the catalog folder of the same name, so both plugins carry one
// skill source (intent 0002 S3, D4).
func TestCodexPluginSkillsMatchTheClaudeCodePlugin(t *testing.T) {
	t.Parallel()
	got := dirNames(t, filepath.Join(repoRoot, codexPluginDir, "skills"))
	want := dirNames(t, filepath.Join(repoRoot, pluginDir, "skills"))
	if !slices.Equal(got, want) {
		t.Fatalf("Codex plugin skills\n got %v\nwant %v", got, want)
	}
	if !slices.Contains(got, "gabo-reviewer-config") {
		t.Fatal("Codex plugin skills miss gabo-reviewer-config (S3)")
	}
	for _, name := range pluginExcluded {
		if slices.Contains(got, name) {
			t.Fatalf("Codex plugin ships excluded skill %s (S3)", name)
		}
	}

	catalog := absReal(t, ".")
	dir := filepath.Join(repoRoot, codexPluginDir, "skills")
	for _, name := range got {
		link := filepath.Join(dir, name)
		target, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("%s is not a symlink: %v", name, err)
		}
		if filepath.IsAbs(target) {
			t.Fatalf("%s links to absolute %s, want a relative link", name, target)
		}
		if resolved, want := absReal(t, link), filepath.Join(catalog, name); resolved != want {
			t.Fatalf("%s resolves to %s, want %s", name, resolved, want)
		}
		if _, err := os.Stat(filepath.Join(link, "SKILL.md")); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// dirNames lists the entry names of dir in directory order.
func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
