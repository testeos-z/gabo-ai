package main

import (
	"bytes"
	"path/filepath"
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
