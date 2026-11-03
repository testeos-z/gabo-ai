package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// patchHeaders are the Codex apply_patch headers that name a file the
// patch writes: a new file, a changed file and a changed file's new name.
var patchHeaders = []string{"*** Add File:", "*** Update File:", "*** Move to:"}

// hookCall is the part of an engine hook's tool-call JSON the hook mode
// reads.
type hookCall struct {
	ToolInput *struct {
		FilePath json.RawMessage `json:"file_path"`
		Command  json.RawMessage `json:"command"`
	} `json:"tool_input"`
}

// PreWriteTargets returns the paths a hook's tool-call JSON would write, in
// order (intent 0001 S11): Claude Code's tool_input.file_path, or else the
// Add File, Update File and Move to headers of a Codex apply_patch in
// tool_input.command (a string, or an argv whose every element is
// scanned). A patch that writes no file gives no path. Empty input, input
// that is not one JSON object, and a tool_input with neither key, a
// non-string value or an empty path are usage errors (ErrPreWriteUsage).
func PreWriteTargets(data []byte) ([]string, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("%w: no path given and stdin is empty", ErrPreWriteUsage)
	}
	var call hookCall
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&call); err != nil {
		return nil, notToolCall(err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, notToolCall(errors.New("data after the JSON object"))
	}
	if call.ToolInput == nil {
		return nil, notToolCall(errors.New("no tool_input object"))
	}
	if set(call.ToolInput.FilePath) {
		var p string
		if err := json.Unmarshal(call.ToolInput.FilePath, &p); err != nil || p == "" {
			return nil, notToolCall(errors.New("tool_input.file_path is not a path"))
		}
		return []string{p}, nil
	}
	if !set(call.ToolInput.Command) {
		return nil, notToolCall(errors.New("tool_input has no file_path or command"))
	}
	var argv []string
	var one string
	if err := json.Unmarshal(call.ToolInput.Command, &one); err == nil {
		argv = []string{one}
	} else if err := json.Unmarshal(call.ToolInput.Command, &argv); err != nil {
		return nil, notToolCall(errors.New("tool_input.command is neither a string nor a list of strings"))
	}
	paths := []string{}
	for _, arg := range argv {
		for line := range strings.Lines(arg) {
			line = strings.TrimSpace(line)
			for _, h := range patchHeaders {
				if rest, ok := strings.CutPrefix(line, h); ok {
					p := strings.TrimSpace(rest)
					if p == "" {
						return nil, notToolCall(fmt.Errorf("%q names no path", h))
					}
					paths = append(paths, p)
				}
			}
		}
	}
	return paths, nil
}

// set reports whether a JSON value is present and not null.
func set(v json.RawMessage) bool {
	return len(v) > 0 && string(v) != "null"
}

// notToolCall is the one-line usage error of stdin that is no tool call.
func notToolCall(err error) error {
	return fmt.Errorf("%w: stdin holds no tool-call JSON (%v)", ErrPreWriteUsage, err)
}
