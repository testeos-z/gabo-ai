package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/check"
)

// piFacts is a gabo_facts stdin object from the Pi extension.
func piFacts(t *testing.T, event string, tools []string, targets ...string) string {
	t.Helper()
	m := map[string]any{"gabo_facts": 1, "event": event, "engine": "pi", "tools": tools, "package_version": "1.2.3"}
	if len(targets) > 0 {
		m["target_paths"] = targets
	}
	return hookJSON(t, m)
}

// With --pre-write, no path and a gabo_facts object on stdin, gabo applies
// the subagent rule (intent 0003 S4, S5) and prints one JSON object; it
// writes nothing.
func TestCheckPreWriteFacts(t *testing.T) {
	refusal := hookJSON(t, check.SubagentRefusal)
	warn := `{"notify":[{"level":"warning","text":` + refusal + `}],"block":null}` + "\n"
	block := `{"notify":[],"block":{"reason":` + refusal + `}}` + "\n"
	pass := `{"notify":[],"block":null}` + "\n"
	pi := []string{"read", "bash", "edit", "write"}
	gentle := []string{"read", "subagent_run", "write"}
	tests := []struct {
		name  string
		stdin func(project string) string
		out   string
		code  int
	}{
		{name: "session_start without subagent_run", stdin: func(string) string { return piFacts(t, "session_start", pi) }, out: warn},
		{name: "session_start with subagent_run", stdin: func(string) string { return piFacts(t, "session_start", gentle) }, out: pass},
		{name: "tool_call on .pi/agents without subagent_run", stdin: func(string) string { return piFacts(t, "tool_call", pi, ".pi/agents/gabo-coder.md") }, out: block, code: 1},
		{name: "tool_call on .pi/agents, absolute path", stdin: func(p string) string {
			return piFacts(t, "tool_call", pi, filepath.Join(p, ".pi", "agents", "gabo-coder.md"))
		}, out: block, code: 1},
		{name: "tool_call on .pi/agents with subagent_run", stdin: func(string) string { return piFacts(t, "tool_call", gentle, ".pi/agents/gabo-coder.md") }, out: pass},
		{name: "tool_call on another path", stdin: func(string) string { return piFacts(t, "tool_call", pi, "README.md") }, out: pass},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := preWriteProject(t, acceptedMD)
			out, code := runPreWriteStdinIn(t, project, tt.stdin(project))
			if code != tt.code || out != tt.out {
				t.Fatalf("exit %d, output %q; want exit %d, output %q", code, out, tt.code, tt.out)
			}
		})
	}
}

// Bad session facts exit 2 with one usage line on stderr and no stdout.
func TestCheckPreWriteBadFactsExitTwo(t *testing.T) {
	project := preWriteProject(t, acceptedMD)
	for _, stdin := range []string{
		`{"gabo_facts":1,"event":"session_start"`,
		`{"gabo_facts":9,"event":"session_start","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"session_start","engine":"claude","tools":[]}`,
		`{"gabo_facts":1,"event":"turn_end","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[]}`,
	} {
		out, err := runCheckStdin(t, project, "1.2.3", stdin, "--pre-write")
		var msg bytes.Buffer
		if code := exitCode(&msg, err); code != 2 || out != "" {
			t.Fatalf("%s: exit %d, stdout %q, want 2 and no stdout", stdin, code, out)
		}
		if line := msg.String(); !strings.HasPrefix(line, "gabo: usage: gabo check --pre-write") || strings.Count(line, "\n") != 1 {
			t.Fatalf("%s: stderr %q, want one usage line", stdin, line)
		}
	}
}

// A tool_call on an intent plan.md is judged with the stage gate: with an
// unaccepted or unreadable spec block.reason equals the "✗" line the
// path-argument mode prints for that path, with an accepted spec nothing
// blocks (the unreadable spec is tested in the check slice); a call blocked by both rules gets one reason, refusal first.
func TestCheckPreWriteFactsStageGate(t *testing.T) {
	plan := "docs/product/intents/0009-x/plan.md"
	gentle := []string{"write", "subagent_run"}
	for _, tt := range []struct {
		name, spec string
		tools      []string
		blocked    bool
	}{
		{name: "unaccepted spec", spec: unfilledMD, tools: gentle, blocked: true},
		{name: "accepted spec", spec: acceptedMD, tools: gentle},
		{name: "both rules", spec: unfilledMD, tools: []string{"write"}, blocked: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := preWriteProject(t, tt.spec)
			argOut, argErr := runCheckStdin(t, project, "1.2.3", "", "--pre-write", plan)
			targets := []string{plan}
			if !slices.Contains(tt.tools, "subagent_run") {
				targets = []string{".pi/agents/gabo-coder.md", plan}
			}
			out, code := runPreWriteStdinIn(t, project, piFacts(t, "tool_call", tt.tools, targets...))
			want := `{"notify":[],"block":null}` + "\n"
			wantCode := 0
			if tt.blocked {
				reasons := []string{}
				if !slices.Contains(tt.tools, "subagent_run") {
					reasons = append(reasons, check.SubagentRefusal)
				}
				if exitOf(argErr) == 1 {
					reasons = append(reasons, strings.TrimSuffix(argOut, "\n"))
				}
				want = `{"notify":[],"block":{"reason":` + hookJSON(t, strings.Join(reasons, "\n")) + `}}` + "\n"
				wantCode = 1
			}
			if code != wantCode || out != want {
				t.Fatalf("exit %d, output %q; want exit %d, output %q", code, out, wantCode, want)
			}
		})
	}
}

// On session_start the package version fact is compared with the
// project's .gabo/VERSION: one info notification when they differ, none
// when equal or when .gabo/VERSION is missing.
func TestCheckPreWriteFactsVersionRule(t *testing.T) {
	gentle := []string{"read", "subagent_run"}
	for _, tt := range []struct {
		name, stamp, out string
	}{
		{name: "different", stamp: "0.4.0\n", out: `{"notify":[{"level":"info","text":"The gabo Pi package is 0.5.0 but this project's .gabo/VERSION is 0.4.0; next: run gabo init with the gabo CLI 0.5.0 to update the project's catalog."}],"block":null}` + "\n"},
		{name: "equal", stamp: "0.5.0\n", out: `{"notify":[],"block":null}` + "\n"},
		{name: "missing", out: `{"notify":[],"block":null}` + "\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := preWriteProject(t, acceptedMD)
			if tt.stamp != "" {
				if err := os.MkdirAll(filepath.Join(project, ".gabo"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(project, ".gabo", "VERSION"), []byte(tt.stamp), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			stdin := hookJSON(t, map[string]any{"gabo_facts": 1, "event": "session_start", "engine": "pi", "tools": gentle, "package_version": "0.5.0"})
			out, code := runPreWriteStdinIn(t, project, stdin)
			if code != 0 || out != tt.out {
				t.Fatalf("exit %d, output %q; want exit 0, output %q", code, out, tt.out)
			}
		})
	}
}
