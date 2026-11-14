package check

import "errors"

// ErrPreCommandUsage marks a usage error of the shell hook mode.
var ErrPreCommandUsage = errors.New("usage: gabo check --pre-command (the hook's tool-call JSON on stdin)")

// ShellCall is the part of an engine's shell tool call the shell guard reads.
type ShellCall struct {
	// Command is the shell command line.
	Command string
	// AgentID is the engine's agent id of a subagent call; empty for the
	// main session.
	AgentID string
}

// ParseShellCall reads one tool-call JSON object for the shell hook mode. Stub: RED.
func ParseShellCall(data []byte) (ShellCall, error) {
	return ShellCall{}, nil
}

// CoderSession reports whether a shell call belongs to a coder session. Stub: RED.
func CoderSession(agentID, engine string) bool {
	return false
}
