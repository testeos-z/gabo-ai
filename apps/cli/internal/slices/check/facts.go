package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
)

// FactsKey is the top-level stdin key that marks a session-facts object
// (intent 0003 D9) and holds its schema version.
const FactsKey = "gabo_facts"

// FactsSchema is the one session-facts schema version this CLI reads.
const FactsSchema = 1

// Session-facts events.
const (
	EventSessionStart = "session_start"
	EventToolCall     = "tool_call"
)

// Notification levels.
const (
	LevelWarning = "warning"
	LevelInfo    = "info"
)

// factsEngines are the engines that send session facts.
var factsEngines = []string{"pi"}

// SubagentTools are the registered tool names that tell a subagent package
// is loaded in the session (intent 0003 D2): gentle-pi's subagent_run.
// Adding a package is a CLI release.
var SubagentTools = []string{"subagent_run"}

// SubagentRefusal is the one source of the refusal text (intent 0003 D7):
// the quoted block of gabo-runtime-pi Prerequisite step 4, kept equal to
// the catalog skill by a test.
const SubagentRefusal = "Pi has no subagents, so gabo cannot work on Pi without a subagent package. For subagents I recommend gentle-pi, installed with `npm i -g gentle-pi` (adds the `gentle-shell` launcher) or `pi install npm:gentle-pi` (inside plain Pi); another subagent package is acceptable. As the recommended setup I recommend the full Gentle AI framework (https://github.com/Gentleman-Programming/gentle-ai): its methodology with RDD and the Engram memory stack, set up for Pi with `gentle-ai install --agent pi`. Then ask me to rebuild."

// piAgentsDir is the project directory Pi's subagent packages read agents
// from; gabo writes them only with a subagent package loaded.
const piAgentsDir = ".pi/agents"

// Facts are the session facts an engine extension passes on stdin.
type Facts struct {
	Event          string   `json:"event"`
	Engine         string   `json:"engine"`
	Tools          []string `json:"tools"`
	PackageVersion string   `json:"package_version"`
	TargetPaths    []string `json:"target_paths"`
}

// Notification is one message the extension shows unchanged.
type Notification struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// Block is the reason a tool call is blocked.
type Block struct {
	Reason string `json:"reason"`
}

// FactsResult is the one JSON object the session-facts mode prints. Notify
// is never null; Block is null when the call may go ahead.
type FactsResult struct {
	Notify []Notification `json:"notify"`
	Block  *Block         `json:"block"`
}

// IsFacts reports whether stdin is one JSON object with the top-level key
// gabo_facts, whatever its value: such input is judged as session facts,
// any other input as an engine hook's tool call.
func IsFacts(data []byte) bool {
	var obj map[string]json.RawMessage
	if err := decodeOne(data, &obj); err != nil {
		return false
	}
	_, ok := obj[FactsKey]
	return ok
}

// ParseFacts reads a session-facts object (intent 0003 D9): gabo_facts 1,
// engine pi, event session_start or tool_call, tools a list of strings,
// an optional string package_version and, for tool_call, a non-empty list
// of non-empty target_paths. Anything else is a usage error
// (ErrPreWriteUsage) of one line. Unknown keys are ignored.
func ParseFacts(data []byte) (Facts, error) {
	var raw struct {
		Version        json.RawMessage `json:"gabo_facts"`
		Event          string          `json:"event"`
		Engine         string          `json:"engine"`
		Tools          *[]string       `json:"tools"`
		PackageVersion string          `json:"package_version"`
		TargetPaths    []string        `json:"target_paths"`
	}
	if err := decodeOne(data, &raw); err != nil {
		return Facts{}, badFacts(err)
	}
	if string(raw.Version) != fmt.Sprint(FactsSchema) {
		return Facts{}, badFacts(fmt.Errorf("unknown %s schema version %s", FactsKey, raw.Version))
	}
	if !slices.Contains(factsEngines, raw.Engine) {
		return Facts{}, badFacts(fmt.Errorf("engine %q is not %s", raw.Engine, strings.Join(factsEngines, ", ")))
	}
	if raw.Event != EventSessionStart && raw.Event != EventToolCall {
		return Facts{}, badFacts(fmt.Errorf("unknown event %q", raw.Event))
	}
	if raw.Tools == nil {
		return Facts{}, badFacts(errors.New("no tools list"))
	}
	f := Facts{Event: raw.Event, Engine: raw.Engine, Tools: *raw.Tools, PackageVersion: raw.PackageVersion}
	if raw.Event == EventToolCall {
		if len(raw.TargetPaths) == 0 || slices.Contains(raw.TargetPaths, "") {
			return Facts{}, badFacts(errors.New("tool_call needs target_paths, a list of non-empty paths"))
		}
		f.TargetPaths = raw.TargetPaths
	}
	return f, nil
}

// JudgeFacts applies the rules of intent 0003 to the facts and writes
// nothing.
//
// The subagent rule (S4, S5): while no tool in Facts.Tools is in
// SubagentTools, session_start returns one warning with SubagentRefusal,
// and a tool_call any of whose targets lies under .pi/agents/ of the
// project (as written or with its symlinks resolved, as PreWrite resolves
// paths) is blocked with SubagentRefusal as the reason. A target whose
// symlinks cannot be resolved counts as under .pi/agents/ (fail closed).
//
// The version rule (S7): session_start then adds one info notification when
// the package version differs from the project's .gabo/VERSION (versionNote).
//
// The stage gate (S6): a tool_call's targets are judged by PreWrite, the
// path-argument hook mode of intent 0001 S11, and each of its "✗" lines is
// a block reason after the refusal; a PreWrite error blocks too (fail
// closed). Several reasons are joined with newlines.
func (u *UseCase) JudgeFacts(f Facts) FactsResult {
	r := FactsResult{Notify: []Notification{}}
	subagents := hasSubagentTool(f.Tools)
	switch f.Event {
	case EventSessionStart:
		if !subagents {
			r.Notify = append(r.Notify, Notification{Level: LevelWarning, Text: SubagentRefusal})
		}
		if note, ok := u.versionNote(f.PackageVersion); ok {
			r.Notify = append(r.Notify, Notification{Level: LevelInfo, Text: note})
		}
	case EventToolCall:
		var reasons []string
		if !subagents && u.writesPiAgent(f.TargetPaths) {
			reasons = append(reasons, SubagentRefusal)
		}
		rep, err := u.PreWrite(f.TargetPaths)
		if err != nil {
			reasons = append(reasons, err.Error())
		}
		reasons = append(reasons, rep.Lines...)
		if len(reasons) > 0 {
			r.Block = &Block{Reason: strings.Join(reasons, "\n")}
		}
	}
	return r
}

// writesPiAgent reports whether a write to any of paths lands under
// .pi/agents/, or has symlinks that cannot be resolved.
func (u *UseCase) writesPiAgent(paths []string) bool {
	for _, p := range paths {
		targets, err := u.targetsOf(p)
		if err != nil || slices.ContainsFunc(targets, u.isPiAgent) {
			return true
		}
	}
	return false
}

// versionNote compares the package version fact with the project's
// .gabo/VERSION (a leading v ignored) and, when they differ, returns the
// text naming both and the next step: gabo init when the package is newer,
// a package update when the stamp is newer, either when one is no version
// x.y.z. No package version, no project, or a .gabo/VERSION that is
// missing or cannot be read gives none: the version rule only informs.
func (u *UseCase) versionNote(pkg string) (string, bool) {
	pkg = strings.TrimPrefix(strings.TrimSpace(pkg), "v")
	if pkg == "" || u.project == nil {
		return "", false
	}
	b, err := fs.ReadFile(u.project, StampFile)
	if err != nil {
		return "", false
	}
	stamp := strings.TrimPrefix(strings.TrimSpace(string(b)), "v")
	if stamp == pkg {
		return "", false
	}
	next := "run gabo init or update the gabo Pi package so both match"
	pv, okP := parseVersion(pkg)
	sv, okS := parseVersion(stamp)
	switch {
	case okP && okS && pv.compare(sv) > 0:
		next = "run gabo init with the gabo CLI " + pkg + " to update the project's catalog"
	case okP && okS && pv.compare(sv) < 0:
		next = "update the gabo Pi package to " + stamp
	case okP && okS:
		return "", false
	}
	return fmt.Sprintf("The gabo Pi package is %s but this project's %s is %s; next: %s.", pkg, StampFile, stamp, next), true
}

// hasSubagentTool reports whether a registered tool is a subagent tool.
func hasSubagentTool(tools []string) bool {
	return slices.ContainsFunc(tools, func(t string) bool { return slices.Contains(SubagentTools, t) })
}

// isPiAgent reports whether the project-relative path rel lies under
// .pi/agents/, without case on a case-insensitive filesystem.
func (u *UseCase) isPiAgent(rel string) bool {
	if len(rel) <= len(piAgentsDir) || rel[len(piAgentsDir)] != '/' {
		return false
	}
	if u.s.FoldCase {
		return strings.EqualFold(rel[:len(piAgentsDir)], piAgentsDir)
	}
	return rel[:len(piAgentsDir)] == piAgentsDir
}

// decodeOne decodes data as exactly one JSON value into v.
func decodeOne(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(v); err != nil {
		return err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return errors.New("data after the JSON object")
	}
	return nil
}

// badFacts is the one-line usage error of a bad session-facts object.
func badFacts(err error) error {
	return fmt.Errorf("%w: stdin holds no valid %s object (%v)", ErrPreWriteUsage, FactsKey, err)
}
