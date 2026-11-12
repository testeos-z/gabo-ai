package skills

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// stageGatePlugin is the catalog source of the OpenCode stage-gate plugin
// (intent 0004 S10, D10; plan item "Plugin source location").
const stageGatePlugin = "gabo-runtime-opencode/plugins/gabo-stage-gate.js"

func stageGateSource(t *testing.T) []byte {
	t.Helper()
	b, err := content.ReadFile(stageGatePlugin)
	if err != nil {
		t.Fatalf("catalog misses %s: %v", stageGatePlugin, err)
	}
	return b
}

// The plugin ships with the catalog, so gabo init copies it to
// .gabo/skills/ where the build step finds it.
func TestStageGatePluginShipsWithTheCatalog(t *testing.T) {
	t.Parallel()
	files, err := Files()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(files, func(f File) bool { return f.Path == stageGatePlugin }) {
		t.Fatalf("Files() misses %s", stageGatePlugin)
	}
}

// The plugin is a thin connection layer: plain JavaScript that loads no
// module and holds no stage rule; the rule lives in the Go hook mode
// (S10, D10).
func TestStageGatePluginHoldsNoRule(t *testing.T) {
	t.Parallel()
	src := string(stageGateSource(t))
	for _, re := range []string{`\bimport\b`, `\brequire\b`, `spec\.md`, `Acceptance`, `plan\.md`} {
		if loc := regexp.MustCompile(re).FindStringIndex(src); loc != nil {
			t.Errorf("%s holds %q at byte %d; it must load no module and hold no rule", stageGatePlugin, src[loc[0]:loc[1]], loc[0])
		}
	}
}

// OpenCode 2.2.6 refuses a V1 function export ("Plugin must export a default
// definition with an id and an effect or setup function"), so the source is a
// default definition and holds no V1 hook key (intent 0015 T17, probe on the
// 2.2.6 binary).
func TestStageGatePluginIsAV2Definition(t *testing.T) {
	t.Parallel()
	src := string(stageGateSource(t))
	for _, want := range []string{"export default {", `id: "gabo-stage-gate"`, "async setup(ctx)", `ctx.tool.hook("execute.before"`} {
		if !strings.Contains(src, want) {
			t.Errorf("%s: missing %q", stageGatePlugin, want)
		}
	}
	for _, bad := range []string{`"tool.execute.before"`, "export const", "output.args"} {
		if strings.Contains(src, bad) {
			t.Errorf("%s: still holds the V1 shape %q", stageGatePlugin, bad)
		}
	}
}

// The build copies the plugin to .opencode/plugins/ and writes no
// package.json (S9, S10).
func TestRuntimeOpenCodeInstallsTheStageGate(t *testing.T) {
	t.Parallel()
	doc := runtimeOpenCodeDoc(t)
	steps := openCodeSection(t, doc, "Build Steps")
	for _, want := range []string{
		"copy `.gabo/skills/gabo-runtime-opencode/plugins/gabo-stage-gate.js` byte for byte to `.opencode/plugins/gabo-stage-gate.js`",
		"on every build",
		"write no `package.json`",
		"a default definition with an `id` and a `setup` function",
		"`ctx.tool.hook(\"execute.before\", ...)`",
		"`event.tool` and `event.input`",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	if !strings.Contains(openCodeSection(t, doc, "Output Contract"), "`.opencode/plugins/gabo-stage-gate.js`") {
		t.Errorf("## Output Contract: does not report the stage-gate plugin")
	}
	if !strings.Contains(openCodeSection(t, doc, "Hard Rules"), "the links this skill makes and `.opencode/plugins/gabo-stage-gate.js`") {
		t.Errorf("## Hard Rules: the overwrite rule does not name the stage-gate plugin")
	}
}

// stageGateDriver loads the plugin like OpenCode 2.2.6 does: the default
// export is a definition with an `id` and a `setup(ctx)` function; setup
// registers its hook with ctx.tool.hook("execute.before", handler). The
// driver calls the handler once with the event shape probed on the 2.2.6
// binary ({tool, sessionID, agent, messageID, id, input}) and prints the
// outcome as JSON. The plugin runs its real child process: the test puts a
// fake gabo on PATH, or none.
const stageGateDriver = `
const [pluginPath, callJSON] = process.argv.slice(2);
const call = JSON.parse(callJSON);
const warnings = [];
console.warn = (...a) => warnings.push(a.join(" "));
const mod = await import(pluginPath);
const def = mod.default;
const out = { threw: false, message: "", warnings, id: def && def.id, setup: typeof (def && def.setup), hooks: [] };
const handlers = {};
const ctx = { location: { directory: process.env.SESSION_DIR }, tool: { hook(name, fn) { out.hooks.push(name); handlers[name] = fn; } } };
await def.setup(ctx);
const hook = handlers["execute.before"];
if (hook) {
  try {
    await hook({ tool: call.tool, sessionID: "s", agent: "build", messageID: "m", id: "c", input: call.args });
  } catch (e) {
    out.threw = true;
    out.message = String(e && e.message);
  }
}
process.stdout.write(JSON.stringify(out));
`

// fakeGabo records each argument on its own line, then a "--" line, prints
// $FAKE_GABO_OUT and exits with $FAKE_GABO_EXIT.
const fakeGabo = `#!/bin/sh
for a in "$@"; do printf '%s\n' "$a" >> "$FAKE_GABO_LOG"; done
printf 'cwd=%s\n--\n' "$PWD" >> "$FAKE_GABO_LOG"
printf '%s' "$FAKE_GABO_OUT"
exit "$FAKE_GABO_EXIT"
`

type stageGateRun struct {
	Threw    bool     `json:"threw"`
	Message  string   `json:"message"`
	Warnings []string `json:"warnings"`
	ID       string   `json:"id"`
	Setup    string   `json:"setup"`
	Hooks    []string `json:"hooks"`
	calls    [][]string
	cwds     []string
}

// runStageGate runs the plugin hook once under node for one tool call, with
// a fake gabo on PATH (or none when withGabo is false).
func runStageGate(t *testing.T, node string, call map[string]any, withGabo bool, exit, out string) stageGateRun {
	t.Helper()
	dir := t.TempDir()
	plugin := filepath.Join(dir, "gabo-stage-gate.mjs")
	if err := os.WriteFile(plugin, stageGateSource(t), 0o644); err != nil {
		t.Fatal(err)
	}
	driver := filepath.Join(dir, "driver.mjs")
	if err := os.WriteFile(driver, []byte(stageGateDriver), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if withGabo {
		if err := os.WriteFile(filepath.Join(bin, "gabo"), []byte(fakeGabo), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	log := filepath.Join(dir, "argv.log")
	callJSON, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, driver, plugin, string(callJSON))
	// The session directory differs from the process cwd, as after a session
	// move: gabo must run in the session directory.
	session := filepath.Join(dir, "session")
	if err := os.Mkdir(session, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd.Dir = dir
	cmd.Env = []string{"SESSION_DIR=" + session, "PATH=" + bin, "HOME=" + dir, "FAKE_GABO_LOG=" + log, "FAKE_GABO_EXIT=" + exit, "FAKE_GABO_OUT=" + out}
	stdout, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		t.Fatalf("node driver: %v\n%s", err, stderr)
	}
	var r stageGateRun
	if err := json.Unmarshal(stdout, &r); err != nil {
		t.Fatalf("driver output %q: %v", stdout, err)
	}
	if r.ID != "gabo-stage-gate" || r.Setup != "function" {
		t.Fatalf("%s default export has id %q and setup %q, want a V2 definition (id gabo-stage-gate, setup function)", stageGatePlugin, r.ID, r.Setup)
	}
	if !slices.Equal(r.Hooks, []string{"execute.before"}) {
		t.Fatalf("%s registers hooks %q, want [execute.before]", stageGatePlugin, r.Hooks)
	}
	if b, err := os.ReadFile(log); err == nil {
		var cur []string
		for _, l := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
			if l == "--" {
				r.calls = append(r.calls, cur)
				cur = nil
				continue
			}
			if v, ok := strings.CutPrefix(l, "cwd="); ok {
				r.cwds = append(r.cwds, v)
				continue
			}
			cur = append(cur, l)
		}
	}
	return r
}

func requireNode(t *testing.T) string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH: the stage-gate plugin test runs the plugin with node (intent 0004 plan, Plugin test runner)")
	}
	return node
}

// the execute.before hook passes each target path of an edit, write or patch
// call to gabo check --pre-write, throws with gabo's output on exit 1, and
// on exit 2 or no gabo on PATH lets the call run with one warning; other
// tools never call gabo (S10, D10).
func TestStageGatePluginRelaysWritesToGabo(t *testing.T) {
	t.Parallel()
	node := requireNode(t)
	const target = "docs/product/intents/0001-x/notes.md"
	patch := "*** Begin Patch\n*** Add File: a/new.txt\n+hi\n*** Update File: b/old.txt\n*** Move to: c/moved.txt\n@@\n-x\n+y\n*** Delete File: d/gone.txt\n*** End Patch"
	cases := []struct {
		name      string
		call      map[string]any
		withGabo  bool
		exit, out string
		wantCalls [][]string
		wantThrow bool
		wantWarn  int
	}{
		{name: "write passes", call: map[string]any{"tool": "write", "args": map[string]any{"path": target, "content": "x"}}, withGabo: true, exit: "0",
			wantCalls: [][]string{{"check", "--pre-write", target}}},
		{name: "edit passes", call: map[string]any{"tool": "edit", "args": map[string]any{"path": target, "oldString": "a", "newString": "b"}}, withGabo: true, exit: "0",
			wantCalls: [][]string{{"check", "--pre-write", target}}},
		{name: "edit with v1 filePath", call: map[string]any{"tool": "edit", "args": map[string]any{"filePath": "/abs/" + target}}, withGabo: true, exit: "0",
			wantCalls: [][]string{{"check", "--pre-write", "/abs/" + target}}},
		{name: "patch passes every target", call: map[string]any{"tool": "patch", "args": map[string]any{"patchText": patch}}, withGabo: true, exit: "0",
			wantCalls: [][]string{{"check", "--pre-write", "a/new.txt", "b/old.txt", "c/moved.txt"}}},
		{name: "write blocked", call: map[string]any{"tool": "write", "args": map[string]any{"path": target}}, withGabo: true, exit: "1", out: "✗ " + target + ": blocked by gabo\n",
			wantCalls: [][]string{{"check", "--pre-write", target}}, wantThrow: true},
		{name: "gabo usage error", call: map[string]any{"tool": "write", "args": map[string]any{"path": target}}, withGabo: true, exit: "2", out: "usage\n",
			wantCalls: [][]string{{"check", "--pre-write", target}}, wantWarn: 1},
		{name: "gabo missing", call: map[string]any{"tool": "edit", "args": map[string]any{"path": target}}, withGabo: false, wantWarn: 1},
		{name: "read never calls gabo", call: map[string]any{"tool": "read", "args": map[string]any{"path": target}}, withGabo: true, exit: "1"},
		{name: "shell never calls gabo", call: map[string]any{"tool": "shell", "args": map[string]any{"command": "touch " + target}}, withGabo: true, exit: "1"},
		{name: "grep never calls gabo", call: map[string]any{"tool": "grep", "args": map[string]any{"pattern": "x", "path": target}}, withGabo: true, exit: "1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := runStageGate(t, node, tc.call, tc.withGabo, tc.exit, tc.out)
			if !slices.EqualFunc(r.calls, tc.wantCalls, slices.Equal) {
				t.Errorf("gabo calls = %q, want %q", r.calls, tc.wantCalls)
			}
			for _, cwd := range r.cwds {
				if filepath.Base(cwd) != "session" {
					t.Errorf("gabo ran in %q, want the session directory (ctx.location.directory)", cwd)
				}
			}
			if r.Threw != tc.wantThrow {
				t.Errorf("threw = %v (message %q), want %v", r.Threw, r.Message, tc.wantThrow)
			}
			if tc.wantThrow && !strings.Contains(r.Message, strings.TrimSpace(tc.out)) {
				t.Errorf("thrown message %q lacks gabo's output %q", r.Message, tc.out)
			}
			if len(r.Warnings) != tc.wantWarn {
				t.Errorf("warnings = %q, want %d", r.Warnings, tc.wantWarn)
			}
		})
	}
}
