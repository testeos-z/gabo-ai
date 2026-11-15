package check

import (
	"errors"
	"testing"
)

// I0028 S2: the shell hook reads one tool-call JSON object: the command of
// tool_input.command (a string, or an argv that is read as one command line)
// and the agent id of a subagent call.
func TestParseShellCall(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want ShellCall
	}{
		{"Claude Code command string", `{"tool_name":"Bash","tool_input":{"command":"git status"}}`, ShellCall{Command: "git status"}},
		{"subagent call carries its agent id", `{"agent_id":"a1","agent_type":"gabo-coder","tool_input":{"command":"git status"}}`, ShellCall{Command: "git status", AgentID: "a1"}},
		{"argv is read as one command line", `{"tool_input":{"command":["bash","-lc","git push origin main"]}}`, ShellCall{Command: "bash -lc 'git push origin main'"}},
		{"empty command is a call with nothing to judge", `{"tool_input":{"command":""}}`, ShellCall{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseShellCall([]byte(tt.in))
			if err != nil {
				t.Fatalf("ParseShellCall() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseShellCall() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Stdin that is no shell tool call is a usage error (exit 2 at the command).
func TestParseShellCallRejectsNonShellInput(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		"", " \n", "{", "not json", "[]", `"x"`, `{}`,
		`{"tool_input":{}}`,
		`{"tool_input":{"command":7}}`,
		`{"tool_input":{"command":["bash",7]}}`,
		`{"tool_input":{"file_path":"a"}}`,
		`{"tool_input":{"command":"a"}} {}`,
	} {
		if _, err := ParseShellCall([]byte(in)); !errors.Is(err, ErrPreCommandUsage) {
			t.Errorf("%q: err = %v, want ErrPreCommandUsage", in, err)
		}
	}
}

// I0028 S4 (session kind): a call with an agent id is a coder session; a call
// without one is the main session of claude or codex, which give that id only
// to subagents. Every other engine, and an unset engine, gets the coder rules.
func TestCoderSession(t *testing.T) {
	t.Parallel()
	tests := []struct {
		agentID, engine string
		want            bool
	}{
		{"a1", "claude", true},
		{"a1", "codex", true},
		{"", "claude", false},
		{"", "codex", false},
		{"", "pi", true},
		{"", "opencode", true},
		{"", "", true},
	}
	for _, tt := range tests {
		if got := CoderSession(tt.agentID, tt.engine); got != tt.want {
			t.Errorf("CoderSession(%q, %q) = %v, want %v", tt.agentID, tt.engine, got, tt.want)
		}
	}
}
