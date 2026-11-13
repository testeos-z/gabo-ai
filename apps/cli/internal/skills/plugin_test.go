package skills

import (
	"encoding/json"
	"os"
	"os/exec"
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

// plugin.json declares the checks option as a boolean that is on by default,
// so one /config row turns both hooks off (intent 0001 D9).
func TestPluginDeclaresChecksOption(t *testing.T) {
	t.Parallel()
	p := readJSON(t, pluginDir+"/.claude-plugin/plugin.json")
	cfg, _ := p["userConfig"].(map[string]any)
	checks, ok := cfg["checks"].(map[string]any)
	if !ok {
		t.Fatalf("plugin.json userConfig %v has no checks option (D9)", p["userConfig"])
	}
	if checks["type"] != "boolean" || checks["default"] != true {
		t.Fatalf("userConfig.checks type %v default %v, want boolean true (D9)", checks["type"], checks["default"])
	}
	for _, key := range []string{"title", "description"} {
		if s, _ := checks[key].(string); s == "" {
			t.Fatalf("userConfig.checks has no %s", key)
		}
	}
}

// hooks.json wires the post-write hook: PostToolUse on Write|Edit runs
// scripts/gabo-check.sh from the plugin root (intent 0001 S6, D8).
func TestPluginHooksWirePostWriteCheck(t *testing.T) {
	t.Parallel()
	h := readJSON(t, pluginDir+"/hooks/hooks.json")
	hooks, _ := h["hooks"].(map[string]any)
	post, _ := hooks["PostToolUse"].([]any)
	if len(post) != 1 {
		t.Fatalf("hooks.json PostToolUse %v, want exactly one matcher group", hooks["PostToolUse"])
	}
	group, _ := post[0].(map[string]any)
	if group["matcher"] != "Write|Edit" {
		t.Fatalf("PostToolUse matcher %v, want Write|Edit", group["matcher"])
	}
	cmds, _ := group["hooks"].([]any)
	if len(cmds) != 1 {
		t.Fatalf("PostToolUse hooks %v, want exactly one command", group["hooks"])
	}
	cmd, _ := cmds[0].(map[string]any)
	want := `bash "${CLAUDE_PLUGIN_ROOT}/scripts/gabo-check.sh"`
	if cmd["type"] != "command" || cmd["command"] != want {
		t.Fatalf("PostToolUse hook %v, want command %s", cmd, want)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, pluginDir, "scripts", "gabo-check.sh")); err != nil {
		t.Fatal(err)
	}
}

// The post-write hook script passes its shell tests with a fake gabo on PATH
// (intent 0001 S6, S7, S8).
func TestPluginPostWriteHookScript(t *testing.T) {
	if testing.Short() {
		t.Skip("shell tests skipped in -short mode")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not found: %v", err)
	}
	test := filepath.Join(repoRoot, pluginDir, "scripts", "tests", "gabo-check.test.sh")
	out, err := exec.Command("bash", test).CombinedOutput()
	if err != nil {
		t.Fatalf("post-write hook shell tests: %v\n%s", err, out)
	}
}

// hooks.json wires the pre-write hook: PreToolUse on Write|Edit runs
// scripts/gabo-pre-write.sh from the plugin root (intent 0001 S12, D8).
func TestPluginHooksWirePreWriteCheck(t *testing.T) {
	t.Parallel()
	h := readJSON(t, pluginDir+"/hooks/hooks.json")
	hooks, _ := h["hooks"].(map[string]any)
	pre, _ := hooks["PreToolUse"].([]any)
	if len(pre) != 1 {
		t.Fatalf("hooks.json PreToolUse %v, want exactly one matcher group", hooks["PreToolUse"])
	}
	group, _ := pre[0].(map[string]any)
	if group["matcher"] != "Write|Edit" {
		t.Fatalf("PreToolUse matcher %v, want Write|Edit", group["matcher"])
	}
	cmds, _ := group["hooks"].([]any)
	if len(cmds) != 1 {
		t.Fatalf("PreToolUse hooks %v, want exactly one command", group["hooks"])
	}
	cmd, _ := cmds[0].(map[string]any)
	want := `bash "${CLAUDE_PLUGIN_ROOT}/scripts/gabo-pre-write.sh"`
	if cmd["type"] != "command" || cmd["command"] != want {
		t.Fatalf("PreToolUse hook %v, want command %s", cmd, want)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, pluginDir, "scripts", "gabo-pre-write.sh")); err != nil {
		t.Fatal(err)
	}
}

// The pre-write hook script passes its shell tests with a fake gabo on PATH
// that records its argv and stdin (intent 0001 S7, S12).
func TestPluginPreWriteHookScript(t *testing.T) {
	if testing.Short() {
		t.Skip("shell tests skipped in -short mode")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skipf("bash not found: %v", err)
	}
	test := filepath.Join(repoRoot, pluginDir, "scripts", "tests", "gabo-pre-write.test.sh")
	out, err := exec.Command("bash", test).CombinedOutput()
	if err != nil {
		t.Fatalf("pre-write hook shell tests: %v\n%s", err, out)
	}
}

// releaseVersion is the gabo release the plugin manifests and the Pi package
// carry (intent 0006 S14).
const releaseVersion = "0.4.0"

// The Claude Code and Codex plugin manifests and the @gabo-ai/pi package all
// read the current release version (intent 0006 S14).
func TestManifestsCarryReleaseVersion(t *testing.T) {
	t.Parallel()
	for _, rel := range []string{
		"plugins/claude-code/.claude-plugin/plugin.json",
		"plugins/codex/.codex-plugin/plugin.json",
		"packages/pi/package.json",
	} {
		if v := readJSON(t, rel)["version"]; v != releaseVersion {
			t.Errorf("%s version %v, want %s", rel, v, releaseVersion)
		}
	}
	if n := readJSON(t, "packages/pi/package.json")["name"]; n != "@gabo-ai/pi" {
		t.Errorf("packages/pi/package.json name %v, want @gabo-ai/pi", n)
	}
}
