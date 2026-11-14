package skills

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
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
		"A `shell` call goes to `gabo check --pre-command` with the tool-call JSON on stdin",
		"both run with `GABO_ENGINE=opencode`",
		"exit 3 adds that OpenCode shows no prompt for an ask",
		"any other exit or a signal blocks the call and names the failure",
	} {
		if !strings.Contains(steps, want) {
			t.Errorf("## Build Steps: missing %q", want)
		}
	}
	for _, stale := range []string{"or exits otherwise, the call runs", "A `shell` call that writes files is not gated"} {
		if strings.Contains(steps, stale) {
			t.Errorf("## Build Steps: still says %q", stale)
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

// fakeGabo records its stdin in <log>.stdin (shell builtins only: the test
// PATH holds the fake bin alone), each argument on its own line,
// then the cwd and GABO_ENGINE and a "--" line; it prints $FAKE_GABO_OUT and
// exits with $FAKE_GABO_EXIT, or kills itself with SIGKILL when that is
// "killed".
const fakeGabo = `#!/bin/sh
while IFS= read -r line || [ -n "$line" ]; do printf '%s\n' "$line"; done > "$FAKE_GABO_LOG.stdin"
for a in "$@"; do printf '%s\n' "$a" >> "$FAKE_GABO_LOG"; done
printf 'cwd=%s\nengine=%s\ntask=%s\noutcome=%s\n--\n' "$PWD" "$GABO_ENGINE" "$GABO_TASK" "$GABO_OUTCOME_FILE" >> "$FAKE_GABO_LOG"
printf '%s' "$FAKE_GABO_OUT"
if [ "$FAKE_GABO_EXIT" = killed ]; then kill -KILL $$; fi
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
	engines  []string
	tasks    []string
	outcomes []string
	stdin    string
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
		// "noexec" leaves the fake without its execute bit: the spawn fails with EACCES.
		mode := os.FileMode(0o755)
		if exit == "noexec" {
			mode = 0o644
		}
		if err := os.WriteFile(filepath.Join(bin, "gabo"), []byte(fakeGabo), mode); err != nil {
			t.Fatal(err)
		}
	}
	log := filepath.Join(dir, "argv.log")
	callJSON, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	// The driver never outlives the test: a timeout kills node, and WaitDelay
	// stops Wait from blocking on a gabo that kept the output pipes open.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, driver, plugin, string(callJSON))
	cmd.WaitDelay = 5 * time.Second
	// The session directory differs from the process cwd, as after a session
	// move: gabo must run in the session directory.
	session := filepath.Join(dir, "session")
	if err := os.Mkdir(session, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd.Dir = dir
	// The ambient GABO_ENGINE names another engine: the plugin must still
	// tell gabo the call comes from opencode.
	cmd.Env = []string{"SESSION_DIR=" + session, "PATH=" + bin, "HOME=" + dir, "GABO_ENGINE=claude", "GABO_TASK=I0028/T13", "GABO_OUTCOME_FILE=/nonexistent/outcome.jsonl", "FAKE_GABO_LOG=" + log, "FAKE_GABO_EXIT=" + exit, "FAKE_GABO_OUT=" + out}
	stdout, err := cmd.Output()
	if ctx.Err() != nil {
		t.Fatalf("node driver timed out after 60s")
	}
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
			if v, ok := strings.CutPrefix(l, "engine="); ok {
				r.engines = append(r.engines, v)
				continue
			}
			if v, ok := strings.CutPrefix(l, "task="); ok {
				r.tasks = append(r.tasks, v)
				continue
			}
			if v, ok := strings.CutPrefix(l, "outcome="); ok {
				r.outcomes = append(r.outcomes, v)
				continue
			}
			cur = append(cur, l)
		}
	}
	if b, err := os.ReadFile(log + ".stdin"); err == nil {
		r.stdin = strings.TrimSuffix(string(b), "\n")
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

// The execute.before hook relays each call to gabo with GABO_ENGINE=opencode:
// an edit, write or patch call goes to gabo check --pre-write with its target
// paths; a shell call goes to gabo check --pre-command with the tool-call JSON
// on stdin. Exit 0 runs the call; exit 1 and exit 3 block it with gabo's lines
// (exit 3 names the missing prompt: OpenCode shows none for an ask, probe item
// (d) is no); any other exit or a signal blocks it and names the failure; no
// gabo on PATH runs it with one warning. Other tools never call gabo (S3, S10,
// S12, S15, D10).
func TestStageGatePluginRelaysCallsToGabo(t *testing.T) {
	t.Parallel()
	node := requireNode(t)
	const target = "docs/product/intents/0001-x/notes.md"
	const shellCmd = "git push origin feat/x"
	patch := "*** Begin Patch\n*** Add File: a/new.txt\n+hi\n*** Update File: b/old.txt\n*** Move to: c/moved.txt\n@@\n-x\n+y\n*** Delete File: d/gone.txt\n*** End Patch"
	const noPrompt = "OpenCode shows no prompt for an ask, so the call is blocked (intent 0028 S12)"
	cases := []struct {
		name      string
		call      map[string]any
		withGabo  bool
		exit, out string
		wantCalls [][]string
		wantStdin string
		wantThrow bool
		wantMsg   string
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
			wantCalls: [][]string{{"check", "--pre-write", target}}, wantThrow: true, wantMsg: "✗ " + target + ": blocked by gabo"},
		{name: "write ask is blocked with no prompt", call: map[string]any{"tool": "write", "args": map[string]any{"path": target}}, withGabo: true, exit: "3", out: "✗ " + target + ": needs an answer\n",
			wantCalls: [][]string{{"check", "--pre-write", target}}, wantThrow: true, wantMsg: "✗ " + target + ": needs an answer\n" + noPrompt},
		{name: "write gabo usage error blocks", call: map[string]any{"tool": "write", "args": map[string]any{"path": target}}, withGabo: true, exit: "2", out: "usage\n",
			wantCalls: [][]string{{"check", "--pre-write", target}}, wantThrow: true, wantMsg: "gabo check --pre-write failed (exit 2); the call is blocked\nusage"},
		{name: "write gabo killed by a signal blocks", call: map[string]any{"tool": "write", "args": map[string]any{"path": target}}, withGabo: true, exit: "killed",
			wantCalls: [][]string{{"check", "--pre-write", target}}, wantThrow: true, wantMsg: "gabo check --pre-write was killed by signal SIGKILL; the call is blocked"},
		{name: "write gabo not executable blocks", call: map[string]any{"tool": "write", "args": map[string]any{"path": target}}, withGabo: true, exit: "noexec",
			wantThrow: true, wantMsg: "gabo check --pre-write could not run (spawn gabo EACCES); the call is blocked"},
		{name: "write gabo missing runs with one warning", call: map[string]any{"tool": "edit", "args": map[string]any{"path": target}}, withGabo: false, wantWarn: 1},
		{name: "shell passes", call: map[string]any{"tool": "shell", "args": map[string]any{"command": shellCmd}}, withGabo: true, exit: "0",
			wantCalls: [][]string{{"check", "--pre-command"}}, wantStdin: `{"tool_input":{"command":"git push origin feat/x"}}`},
		{name: "shell blocked", call: map[string]any{"tool": "shell", "args": map[string]any{"command": shellCmd}}, withGabo: true, exit: "1", out: "✗ shell_guard: a push is blocked\n",
			wantCalls: [][]string{{"check", "--pre-command"}}, wantStdin: `{"tool_input":{"command":"git push origin feat/x"}}`, wantThrow: true, wantMsg: "✗ shell_guard: a push is blocked"},
		{name: "shell ask is blocked with no prompt", call: map[string]any{"tool": "shell", "args": map[string]any{"command": shellCmd}}, withGabo: true, exit: "3", out: "✗ shell_guard: asks\n",
			wantCalls: [][]string{{"check", "--pre-command"}}, wantStdin: `{"tool_input":{"command":"git push origin feat/x"}}`, wantThrow: true, wantMsg: "✗ shell_guard: asks\n" + noPrompt},
		{name: "shell gabo usage error blocks", call: map[string]any{"tool": "shell", "args": map[string]any{"command": shellCmd}}, withGabo: true, exit: "2", out: "usage\n",
			wantCalls: [][]string{{"check", "--pre-command"}}, wantStdin: `{"tool_input":{"command":"git push origin feat/x"}}`, wantThrow: true, wantMsg: "gabo check --pre-command failed (exit 2); the call is blocked\nusage"},
		{name: "shell gabo missing runs with one warning", call: map[string]any{"tool": "shell", "args": map[string]any{"command": shellCmd}}, withGabo: false, wantWarn: 1},
		{name: "read never calls gabo", call: map[string]any{"tool": "read", "args": map[string]any{"path": target}}, withGabo: true, exit: "1"},
		{name: "grep never calls gabo", call: map[string]any{"tool": "grep", "args": map[string]any{"pattern": "x", "path": target}}, withGabo: true, exit: "1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := runStageGate(t, node, tc.call, tc.withGabo, tc.exit, tc.out)
			if !slices.EqualFunc(r.calls, tc.wantCalls, slices.Equal) {
				t.Errorf("gabo calls = %q, want %q", r.calls, tc.wantCalls)
			}
			for _, engine := range r.engines {
				if engine != "opencode" {
					t.Errorf("gabo ran with GABO_ENGINE=%q, want opencode", engine)
				}
			}
			if len(r.engines) != len(r.calls) {
				t.Errorf("gabo ran %d times with %d engine records", len(r.calls), len(r.engines))
			}
			for _, task := range r.tasks {
				if task != "I0028/T13" {
					t.Errorf("gabo ran with GABO_TASK=%q, want the cycle's task", task)
				}
			}
			for _, outcome := range r.outcomes {
				if outcome != "/nonexistent/outcome.jsonl" {
					t.Errorf("gabo ran with GABO_OUTCOME_FILE=%q, want the cycle's outcome file", outcome)
				}
			}
			if len(r.tasks) != len(r.calls) || len(r.outcomes) != len(r.calls) {
				t.Errorf("gabo ran %d times with %d task and %d outcome records", len(r.calls), len(r.tasks), len(r.outcomes))
			}
			if r.stdin != tc.wantStdin {
				t.Errorf("gabo stdin = %q, want %q", r.stdin, tc.wantStdin)
			}
			for _, cwd := range r.cwds {
				if filepath.Base(cwd) != "session" {
					t.Errorf("gabo ran in %q, want the session directory (ctx.location.directory)", cwd)
				}
			}
			if r.Threw != tc.wantThrow {
				t.Errorf("threw = %v (message %q), want %v", r.Threw, r.Message, tc.wantThrow)
			}
			if tc.wantMsg != "" && r.Message != tc.wantMsg {
				t.Errorf("thrown message = %q, want %q", r.Message, tc.wantMsg)
			}
			if len(r.Warnings) != tc.wantWarn {
				t.Errorf("warnings = %q, want %d", r.Warnings, tc.wantWarn)
			}
		})
	}
}

// The plugin's header no longer says a shell call is not gated: the shell
// call reaches gabo's pre-command mode (S3).
func TestStageGatePluginGatesShellCalls(t *testing.T) {
	t.Parallel()
	src := string(stageGateSource(t))
	if strings.Contains(src, "not gated (documented limit)") {
		t.Errorf("%s still says a shell call is not gated", stageGatePlugin)
	}
	if !strings.Contains(src, "gabo check --pre-command") {
		t.Errorf("%s does not name gabo check --pre-command", stageGatePlugin)
	}
}
