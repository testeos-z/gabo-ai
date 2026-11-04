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

// stageGateDriver loads the plugin with a stubbed Bun `$` and calls its
// tool.execute.before hook once. The stub runs commands without a shell,
// expands an interpolated array into one argument per element, implements
// Bun's `which` builtin, and answers a missing command like Bun does: exit 1
// with "bun: command not found: <cmd>". It prints the outcome as JSON.
const stageGateDriver = `
const { spawnSync } = await import("node:child_process");
const fs = await import("node:fs");
const path = await import("node:path");
const [pluginPath, callJSON] = process.argv.slice(2);
const call = JSON.parse(callJSON);

function findOnPath(cmd) {
  for (const dir of (process.env.PATH || "").split(":")) {
    if (!dir) continue;
    const p = path.join(dir, cmd);
    try { fs.accessSync(p, fs.constants.X_OK); return p; } catch {}
  }
  return null;
}

function run(argv) {
  if (argv[0] === "which") {
    const p = findOnPath(argv[1]);
    return p ? { exitCode: 0, stdout: Buffer.from(p + "\n"), stderr: Buffer.alloc(0) }
             : { exitCode: 1, stdout: Buffer.from(argv[1] + " not found\n"), stderr: Buffer.alloc(0) };
  }
  const r = spawnSync(argv[0], argv.slice(1), { env: process.env });
  if (r.error && r.error.code === "ENOENT") {
    return { exitCode: 1, stdout: Buffer.alloc(0), stderr: Buffer.from("bun: command not found: " + argv[0] + "\n") };
  }
  return { exitCode: r.status, stdout: r.stdout, stderr: r.stderr };
}

function $(strings, ...values) {
  const argv = [];
  strings.forEach((s, i) => {
    argv.push(...s.split(/\s+/).filter(Boolean));
    if (i < values.length) {
      const v = values[i];
      if (Array.isArray(v)) argv.push(...v.map(String)); else argv.push(String(v));
    }
  });
  let nothrow = false;
  const shell = {
    nothrow() { nothrow = true; return shell; },
    quiet() { return shell; },
    then(resolve, reject) {
      const r = run(argv);
      if (r.exitCode !== 0 && !nothrow) {
        const e = new Error("ShellError: exit " + r.exitCode);
        Object.assign(e, r);
        return Promise.reject(e).then(resolve, reject);
      }
      return Promise.resolve(r).then(resolve, reject);
    },
  };
  return shell;
}

const warnings = [];
console.warn = (...a) => warnings.push(a.join(" "));
const mod = await import(pluginPath);
const out = { threw: false, message: "", warnings, plugins: Object.keys(mod).length };
for (const make of Object.values(mod)) {
  const hooks = await make({ $, directory: process.cwd(), worktree: process.cwd() });
  const hook = hooks["tool.execute.before"];
  if (!hook) continue;
  try {
    await hook({ tool: call.tool, sessionID: "s", callID: "c" }, { args: call.args });
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
printf -- '--\n' >> "$FAKE_GABO_LOG"
printf '%s' "$FAKE_GABO_OUT"
exit "$FAKE_GABO_EXIT"
`

type stageGateRun struct {
	Threw    bool     `json:"threw"`
	Message  string   `json:"message"`
	Warnings []string `json:"warnings"`
	Plugins  int      `json:"plugins"`
	calls    [][]string
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
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + bin, "HOME=" + dir, "FAKE_GABO_LOG=" + log, "FAKE_GABO_EXIT=" + exit, "FAKE_GABO_OUT=" + out}
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
	if r.Plugins == 0 {
		t.Fatalf("%s exports no plugin function", stageGatePlugin)
	}
	if b, err := os.ReadFile(log); err == nil {
		var cur []string
		for _, l := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
			if l == "--" {
				r.calls = append(r.calls, cur)
				cur = nil
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

// tool.execute.before passes each target path of an edit, write or patch
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
