package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

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

// shellHookCall is the part of the tool-call JSON the shell hook mode reads.
type shellHookCall struct {
	AgentID   string `json:"agent_id"`
	ToolInput *struct {
		Command json.RawMessage `json:"command"`
	} `json:"tool_input"`
}

// plainWord matches a word a shell reads as itself, so it needs no quotes.
var plainWord = regexp.MustCompile(`^[A-Za-z0-9_./=:,+@%-]+$`)

// ParseShellCall reads one engine tool-call JSON object for the shell hook
// mode: tool_input.command, a string or an argv (each argument quoted, so the
// argv reads as the one command line the shell would run), and the agent_id
// of a subagent call. Empty or malformed stdin, stdin with data after the
// object, and a call with no command are usage errors (ErrPreCommandUsage).
func ParseShellCall(data []byte) (ShellCall, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return ShellCall{}, fmt.Errorf("%w: stdin is empty", ErrPreCommandUsage)
	}
	var call shellHookCall
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&call); err != nil {
		return ShellCall{}, fmt.Errorf("%w: stdin holds no tool-call JSON (%v)", ErrPreCommandUsage, err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return ShellCall{}, fmt.Errorf("%w: stdin holds data after the JSON object", ErrPreCommandUsage)
	}
	if call.ToolInput == nil || !set(call.ToolInput.Command) {
		return ShellCall{}, fmt.Errorf("%w: tool_input has no command", ErrPreCommandUsage)
	}
	command, err := commandLine(call.ToolInput.Command)
	if err != nil {
		return ShellCall{}, err
	}
	return ShellCall{Command: command, AgentID: call.AgentID}, nil
}

// commandLine reads tool_input.command: a string as it is, or an argv as one
// command line with each word quoted.
func commandLine(raw json.RawMessage) (string, error) {
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		return one, nil
	}
	var argv []string
	if err := json.Unmarshal(raw, &argv); err != nil {
		return "", fmt.Errorf("%w: tool_input.command is neither a string nor a list of strings", ErrPreCommandUsage)
	}
	words := make([]string, len(argv))
	for i, a := range argv {
		if plainWord.MatchString(a) {
			words[i] = a
		} else {
			words[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
	}
	return strings.Join(words, " "), nil
}

// CoderSession reports whether a shell call belongs to a coder session (intent
// 0028 S4, decision 13). A call with an agent id is a subagent's, judged with
// the coder rules. A call without one is the main session of claude or codex,
// the two engines that give the id to subagents only (intent 0028 probe, S1
// b). Every other engine, and an unset engine, gets the coder rules, the
// stricter reading where an engine gives no id.
func CoderSession(agentID, engine string) bool {
	if agentID != "" {
		return true
	}
	switch engine {
	case "claude", "codex":
		return false
	}
	return true
}
