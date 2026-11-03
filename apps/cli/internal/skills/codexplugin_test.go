package skills

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
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

// The Codex plugin skills directory holds exactly the catalog skills the
// Claude Code plugin ships (every skill gabo init copies minus the
// review-running ones), each as a real copy, never a symlink, byte-identical
// to its catalog folder: codex-cli leaves symlinked skill folders and files out
// of its plugin cache (intent 0002 S1, S3, D4, spec open question 5 option c).
func TestCodexPluginSkillsCopyTheCatalog(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(repoRoot, codexPluginDir, "skills")
	got := dirNames(t, dir)
	var want []string
	for _, n := range Names() {
		if !slices.Contains(pluginExcluded, n) {
			want = append(want, n)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Codex plugin skills\n got %v\nwant %v", got, want)
	}
	if !slices.Contains(got, "gabo-reviewer-config") {
		t.Fatal("Codex plugin skills miss gabo-reviewer-config (S3)")
	}
	for _, name := range pluginExcluded {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("Codex plugin ships excluded skill %s (err %v)", name, err)
		}
	}
	for _, name := range got {
		requireCopy(t, filepath.Join(dir, name), name)
	}
}

// requireCopy fails unless copyPath, a file or directory, is a real copy of
// src (both relative to the catalog package directory): no symlink at or
// below copyPath, the same set of files and directories as src, and each file
// byte-identical with the same mode as git records it (regular file, executable
// or not). It names the first drifted path and how to refresh the copy. It
// takes a file or a directory so later hook-script copies reuse it.
func requireCopy(t *testing.T, copyPath, src string) {
	t.Helper()
	cp, err := treeEntries(copyPath)
	if err != nil {
		t.Fatalf("Codex plugin copy %s: %v", copyPath, err)
	}
	sp, err := treeEntries(src)
	if err != nil {
		t.Fatalf("copy source %s: %v", src, err)
	}
	fix := fmt.Sprintf("refresh it from the catalog: rm -r %[1]s && cp -r %[2]s %[1]s (run from apps/cli/internal/skills)", copyPath, src)
	for _, rel := range sortedUnion(cp, sp) {
		c, inCopy := cp[rel]
		s, inSrc := sp[rel]
		at := filepath.Join(copyPath, rel)
		switch {
		case !inCopy:
			t.Fatalf("%s: missing from the copy of %s; %s", at, filepath.Join(src, rel), fix)
		case !inSrc:
			t.Fatalf("%s: not in the source %s; %s", at, src, fix)
		case c.mode != s.mode:
			t.Fatalf("%s: mode %v, source %s has %v; %s", at, c.mode, filepath.Join(src, rel), s.mode, fix)
		case !bytes.Equal(c.data, s.data):
			t.Fatalf("%s: content differs from %s; %s", at, filepath.Join(src, rel), fix)
		}
	}
}

// treeEntry is one file or directory of a copied tree with its mode as git
// records it: fs.ModeDir for a directory, 0o755 or 0o644 for a regular file
// (git tracks only the executable bit; the rest follows the umask). data is
// nil for directories.
type treeEntry struct {
	mode fs.FileMode
	data []byte
}

// treeEntries reads root (a file or a directory) into entries keyed by the
// path relative to root ("." for root itself). Any symlink is an error.
func treeEntries(root string) (map[string]treeEntry, error) {
	out := map[string]treeEntry{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		info, err := os.Lstat(p)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink, want a real copy", p)
		case info.IsDir():
			out[rel] = treeEntry{mode: fs.ModeDir}
		case info.Mode().IsRegular():
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			mode := fs.FileMode(0o644)
			if info.Mode()&0o100 != 0 {
				mode = 0o755
			}
			out[rel] = treeEntry{mode: mode, data: b}
		default:
			return fmt.Errorf("%s is not a regular file or directory (mode %v)", p, info.Mode())
		}
		return nil
	})
	return out, err
}

// sortedUnion returns the keys of a and b, sorted and without duplicates.
func sortedUnion(a, b map[string]treeEntry) []string {
	keys := slices.Collect(maps.Keys(a))
	for k := range b {
		if _, ok := a[k]; !ok {
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return keys
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
