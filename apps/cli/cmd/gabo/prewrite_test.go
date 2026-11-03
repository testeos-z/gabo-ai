package main

import (
	"bytes"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/kerneltest"
)

const (
	prePlan    = "docs/product/intents/0009-x/plan.md"
	acceptedMD = "## Acceptance\n\n| accepted by | date | notes |\n|---|---|---|\n| owner | 2026-11-03 | ok |\n"
	unfilledMD = "## Acceptance\n\n| accepted by | date | notes |\n|---|---|---|\n|  |  |  |\n"
)

// preWriteProject is a bare project directory with intent 0009-x and,
// unless specMD is empty, its spec.md. The hook mode needs no gabo.yaml.
func preWriteProject(t *testing.T, specMD string) string {
	t.Helper()
	project := t.TempDir()
	dir := filepath.Join(project, filepath.FromSlash("docs/product/intents/0009-x"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "intent.md"), []byte("# intent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if specMD != "" {
		if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(specMD), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return project
}

// runPreWriteIn runs `gabo check --pre-write paths...` in project and
// fails when the run changed the project tree or its modification times.
func runPreWriteIn(t *testing.T, project string, paths ...string) (string, int) {
	t.Helper()
	before, beforeTimes := snapshot(t, project), modes(t, project)
	out, err := runCheck(t, project, "1.2.3", append([]string{"--pre-write"}, paths...)...)
	if !maps.Equal(before, snapshot(t, project)) || !maps.Equal(beforeTimes, modes(t, project)) {
		t.Fatal("check --pre-write changed the project tree or modification times")
	}
	if _, err := os.Lstat(filepath.Join(project, filepath.FromSlash(prePlan))); err == nil {
		t.Fatal("check --pre-write created plan.md")
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return out + msg, exitOf(err)
}

func TestCheckPreWrite(t *testing.T) {
	notAccepted := "✗ " + prePlan + ": spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan\n"
	noSpec := "✗ " + prePlan + ": no spec.md; a plan starts only after a human accepts the spec\n"
	tests := []struct {
		name   string
		specMD string
		paths  func(project string) []string
		out    string
		code   int
	}{
		{name: "unaccepted spec", specMD: unfilledMD, paths: rel(prePlan), out: notAccepted, code: 1},
		{name: "unaccepted spec, absolute path", specMD: unfilledMD, paths: abs(prePlan), out: notAccepted, code: 1},
		{name: "filled Acceptance row", specMD: acceptedMD, paths: rel(prePlan)},
		{name: "filled Acceptance row, absolute path", specMD: acceptedMD, paths: abs(prePlan)},
		{name: "no spec.md", paths: rel(prePlan), out: noSpec, code: 1},
		{name: "non-plan path", paths: rel("docs/product/intents/0009-x/intent.md", "README.md")},
		{name: "path outside the root", paths: func(p string) []string {
			return []string{"../x/" + prePlan, filepath.Join(filepath.Dir(p), "other", filepath.FromSlash(prePlan))}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := preWriteProject(t, tt.specMD)
			out, code := runPreWriteIn(t, project, tt.paths(project)...)
			if code != tt.code || out != tt.out {
				t.Fatalf("exit %d, output %q; want exit %d, output %q", code, out, tt.code, tt.out)
			}
		})
	}
}

func TestCheckPreWriteUnreadableSpecFailsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads unreadable files")
	}
	project := preWriteProject(t, acceptedMD)
	spec := filepath.Join(project, filepath.FromSlash("docs/product/intents/0009-x/spec.md"))
	if err := os.Chmod(spec, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(spec, 0o644) })
	// The unreadable spec cannot be snapshotted by content: compare modes
	// and modification times only.
	before := modes(t, project)
	got, err := runCheck(t, project, "1.2.3", "--pre-write", prePlan)
	if !maps.Equal(before, modes(t, project)) {
		t.Fatal("check --pre-write changed the project tree or modification times")
	}
	out, code := got, exitOf(err)
	if err != nil {
		out += err.Error()
	}
	if code != 1 || !strings.HasPrefix(out, "✗ "+prePlan+": spec.md cannot be read (") || !strings.HasSuffix(out, "); it counts as not accepted\n") || strings.Count(out, "\n") != 1 {
		t.Fatalf("exit %d, output %q", code, out)
	}
}

func TestCheckPreWriteUsageExitsTwo(t *testing.T) {
	project := preWriteProject(t, "")
	for _, args := range [][]string{{"--pre-write"}, {"--pre-write", ""}, {"--bogus", "--pre-write", prePlan}} {
		_, err := runCheck(t, project, "1.2.3", args...)
		if exitOf(err) != 2 || !strings.Contains(err.Error(), "usage: gabo check --pre-write <path>...") {
			t.Fatalf("check %q: exit %d (%v), want 2", args, exitOf(err), err)
		}
	}
}

func rel(paths ...string) func(string) []string {
	return func(string) []string { return paths }
}

func abs(paths ...string) func(string) []string {
	return func(project string) []string {
		out := make([]string, len(paths))
		for i, p := range paths {
			out[i] = filepath.Join(project, filepath.FromSlash(p))
		}
		return out
	}
}

// An absolute path that reaches the project through a symlinked directory
// is still judged: the gate is not skipped as "outside the root".
func TestCheckPreWriteThroughASymlink(t *testing.T) {
	project := preWriteProject(t, unfilledMD)
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(project, alias); err != nil {
		t.Fatal(err)
	}
	want := "✗ " + prePlan + ": spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan\n"
	out, code := runPreWriteIn(t, project, filepath.Join(alias, filepath.FromSlash(prePlan)))
	if code != 1 || out != want {
		t.Fatalf("exit %d, output %q; want exit 1, output %q", code, out, want)
	}
}

// A dangling symlink is judged at the target a write through it would
// create: here an unaccepted intent's plan.md.
func TestCheckPreWriteThroughADanglingSymlink(t *testing.T) {
	project := preWriteProject(t, acceptedMD)
	other := "docs/product/intents/0010-y/plan.md"
	if err := os.Symlink(filepath.FromSlash(other), filepath.Join(project, "notes.md")); err != nil {
		t.Fatal(err)
	}
	want := "✗ " + other + ": no spec.md; a plan starts only after a human accepts the spec\n"
	out, code := runPreWriteIn(t, project, "notes.md")
	if code != 1 || out != want {
		t.Fatalf("exit %d, output %q; want exit 1, output %q", code, out, want)
	}
}

// A relative path is taken from the working directory, here a
// subdirectory of the project (git's top level).
func TestCheckPreWriteRelativeToTheWorkingDirectory(t *testing.T) {
	project := preWriteProject(t, unfilledMD)
	sub := filepath.Join(project, "docs", "product")
	runner := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(project + "\n")}}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GABO_UPGRADE_PR", "")
	t.Chdir(sub)
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3", runner: runner})
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"check", "--pre-write", "intents/0009-x/plan.md"})
	err := root.Execute()
	want := "✗ " + prePlan + ": spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan\n"
	if exitOf(err) != 1 || out.String() != want {
		t.Fatalf("exit %d, output %q; want exit 1, output %q", exitOf(err), out.String(), want)
	}
}

// A ".." after a real symlink climbs from the link's target: lnk points at
// another intent folder, so lnk/../0009-x/plan.md writes intent 0009-x's
// plan.md and is judged as that plan.
func TestCheckPreWriteDotDotAfterASymlink(t *testing.T) {
	project := preWriteProject(t, unfilledMD)
	other := filepath.Join(project, filepath.FromSlash("docs/product/intents/0010-y"))
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(project, "lnk")); err != nil {
		t.Fatal(err)
	}
	want := "✗ " + prePlan + ": spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan\n"
	out, code := runPreWriteIn(t, project, "lnk/../0009-x/plan.md")
	if code != 1 || out != want {
		t.Fatalf("exit %d, output %q; want exit 1, output %q", code, out, want)
	}
}

// foldsCase tells a case-insensitive filesystem by the root spelled in
// another case naming the same directory; the test machine's temporary
// directory is case-sensitive, and a path without letters cannot be
// spelled otherwise.
func TestFoldsCase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if foldsCase(dir) {
		t.Fatalf("foldsCase(%s) = true on a case-sensitive filesystem", dir)
	}
	if foldsCase(string(filepath.Separator)) {
		t.Fatal("foldsCase(/) = true")
	}
	if got, want := swapLastLetterCase("/a/Proj1"), "/a/ProJ1"; got != want {
		t.Fatalf("swapLastLetterCase = %q, want %q", got, want)
	}
}

// With --pre-write and no path the hook's tool-call JSON comes on stdin.
// A Claude Code PreToolUse JSON whose tool_input.file_path is the plan.md
// gives the path-argument result; a Codex apply_patch gives one result per
// target in tool_input.command; malformed or empty stdin exits 2 with one
// usage line.
func TestCheckPreWriteFromStdin(t *testing.T) {
	notAccepted := "✗ " + prePlan + ": spec.md is not accepted; a human fills a row (accepted by, date) of its Acceptance table before the plan\n"
	noSpecY := "✗ docs/product/intents/0010-y/plan.md: no spec.md; a plan starts only after a human accepts the spec\n"
	claude := func(project string) string {
		return hookJSON(t, map[string]any{"hook_event_name": "PreToolUse", "tool_name": "Write", "cwd": project,
			"tool_input": map[string]any{"file_path": filepath.Join(project, filepath.FromSlash(prePlan)), "content": "# plan\n"}})
	}
	codex := func(string) string {
		return hookJSON(t, map[string]any{"hook_event_name": "PreToolUse", "tool_name": "apply_patch", "tool_input": map[string]any{"command": "*** Begin Patch\n" +
			"*** Add File: " + prePlan + "\n+# plan\n" +
			"*** Update File: README.md\n@@\n-a\n+b\n" +
			"*** Add File: docs/product/intents/0010-y/plan.md\n+# plan\n" +
			"*** End Patch\n"}})
	}
	tests := []struct {
		name   string
		specMD string
		stdin  func(project string) string
		out    string
		code   int
	}{
		{name: "Claude Code, unaccepted spec", specMD: unfilledMD, stdin: claude, out: notAccepted, code: 1},
		{name: "Claude Code, filled Acceptance row", specMD: acceptedMD, stdin: claude},
		{name: "Claude Code, no spec.md", stdin: claude, out: "✗ " + prePlan + ": no spec.md; a plan starts only after a human accepts the spec\n", code: 1},
		{name: "Codex apply_patch, unaccepted spec", specMD: unfilledMD, stdin: codex, out: notAccepted + noSpecY, code: 1},
		{name: "Codex apply_patch, filled Acceptance row", specMD: acceptedMD, stdin: codex, out: noSpecY, code: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := preWriteProject(t, tt.specMD)
			stdinOut, stdinCode := runPreWriteStdinIn(t, project, tt.stdin(project))
			if stdinCode != tt.code || stdinOut != tt.out {
				t.Fatalf("exit %d, output %q; want exit %d, output %q", stdinCode, stdinOut, tt.code, tt.out)
			}
		})
	}
	// The Claude Code case equals the path-argument case.
	project := preWriteProject(t, unfilledMD)
	argOut, argCode := runPreWriteIn(t, project, prePlan)
	stdinOut, stdinCode := runPreWriteStdinIn(t, project, claude(project))
	if argOut != stdinOut || argCode != stdinCode {
		t.Fatalf("stdin: exit %d, output %q; path argument: exit %d, output %q", stdinCode, stdinOut, argCode, argOut)
	}
}

func TestCheckPreWriteBadStdinExitsTwo(t *testing.T) {
	project := preWriteProject(t, unfilledMD)
	for _, stdin := range []string{"", "{\"tool_input\":", "not json"} {
		_, err := runCheckStdin(t, project, "1.2.3", stdin, "--pre-write")
		var msg bytes.Buffer
		if code := exitCode(&msg, err); code != 2 {
			t.Fatalf("%q: exit %d (%v), want 2", stdin, code, err)
		}
		if line := msg.String(); !strings.HasPrefix(line, "gabo: usage: gabo check --pre-write <path>...") || strings.Count(line, "\n") != 1 {
			t.Fatalf("%q: stderr %q, want one usage line", stdin, line)
		}
	}
}

// runPreWriteStdinIn runs `gabo check --pre-write` with stdin in project,
// like runPreWriteIn.
func runPreWriteStdinIn(t *testing.T, project, stdin string) (string, int) {
	t.Helper()
	before, beforeTimes := snapshot(t, project), modes(t, project)
	out, err := runCheckStdin(t, project, "1.2.3", stdin, "--pre-write")
	if !maps.Equal(before, snapshot(t, project)) || !maps.Equal(beforeTimes, modes(t, project)) {
		t.Fatal("check --pre-write changed the project tree or modification times")
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return out + msg, exitOf(err)
}

func hookJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
