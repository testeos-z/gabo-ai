package engineconfig

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
)

// ClaudeSettings is the project's committed Claude Code settings file, the
// only file the Claude Code rule writes (S2, D3).
const ClaudeSettings = ".claude/settings.json"

// ClaudeAgentFile is the agent file the Claude Code main session runs; the
// runtime skill builds it (intent 0010 S8).
const ClaudeAgentFile = ".claude/agents/gabo-assistant.md"

// The top-level main-session key and its gabo value (intent 0010 S1).
const (
	claudeAgentKey   = "agent"
	claudeAgentValue = "gabo-assistant"
)

// claudeBuildStep is the S8 step while the main session's agent is not built.
const claudeBuildStep = "ask your agent to build the agents (the main session needs " + ClaudeAgentFile + ")"

// claudeKey is one key the Claude Code rule makes the settings file hold:
// member child of the top-level object parent, with value; an empty parent
// makes child a top-level member.
type claudeKey struct {
	parent, child string
	value         any
}

// claudeKeys are the S2 keys, in the order they are named and inserted.
var claudeKeys = []claudeKey{
	{"extraKnownMarketplaces", "gabo-ai", jsonObj{{"source", jsonObj{{"source", "github"}, {"repo", "testeos-z/gabo-ai"}}}}},
	{"enabledPlugins", "gabo@gabo-ai", true},
}

// claudeSteps are the S9 steps outside the project for Claude Code.
var claudeSteps = []string{
	"trust this project folder in Claude Code so the gabo-ai marketplace and the gabo plugin apply",
}

func (k claudeKey) name() string {
	if k.parent == "" {
		return k.child
	}
	return k.parent + `."` + k.child + `"`
}

// wantedKeys are the keys the rule writes for in: the S2 keys, plus the
// main-session key when Claude Code is the assistant engine (intent 0010 S1).
func wantedKeys(in Input) []claudeKey {
	keys := slices.Clone(claudeKeys)
	if in.Assistant == Claude {
		keys = append(keys, claudeKey{"", claudeAgentKey, claudeAgentValue})
	}
	return keys
}

// claudeRule merges the S2 keys into .claude/settings.json, keeping every
// existing byte (D5): a missing key is inserted, a key with another value is
// kept and reported (S4, D6), and a file that does not parse is left alone
// with one ✗ line (S5). The top-level agent key follows the assistant
// engine (intent 0010 S1, S4): Claude Code merges "gabo-assistant" with the
// same rule; another engine removes exactly that value.
func claudeRule(in Input) Result {
	res := Result{Section: Section{Steps: append([]string(nil), claudeSteps...)}}
	fail := func(reason string) Result {
		res.Section.Lines = []string{FailurePrefix + " " + ClaudeSettings + ": " + reason + "; fix it and run gabo init again"}
		return res
	}

	src, exists := in.Files[ClaudeSettings]
	if !exists {
		top := jsonObj{}
		var added []string
		for _, k := range wantedKeys(in) {
			if k.parent == "" {
				top = append(top, jsonKV{k.child, k.value})
			} else {
				top = append(top, jsonKV{k.parent, jsonObj{{k.child, k.value}}})
			}
			added = append(added, k.name())
		}
		res.Writes = map[string][]byte{ClaudeSettings: []byte(renderJSON(top, "", "  ", true) + "\n")}
		res.Section.Lines = []string{wroteLine(added)}
		res.Section.Steps = appendBuildStep(res.Section.Steps, in, true)
		return res
	}

	var doc any
	if err := json.Unmarshal(src, &doc); err != nil {
		return fail("invalid JSON (" + err.Error() + ")")
	}
	if _, ok := doc.(map[string]any); !ok {
		return fail("the top level is not a JSON object")
	}

	// Decide every key before writing, so a failure writes nothing.
	var missing []claudeKey
	var kept []string
	agentIsGabo := true
	for _, k := range wantedKeys(in) {
		top := scanObject(src, skipWS(src, 0))
		if k.parent == "" {
			m, ok := top.member(src, k.child)
			if !ok {
				missing = append(missing, k)
			} else if raw := src[m.valStart:m.valEnd]; !sameJSON(raw, k.value) {
				kept = append(kept, keptLine(k, raw))
				agentIsGabo = false
			}
			continue
		}
		parent, ok := top.member(src, k.parent)
		if !ok {
			missing = append(missing, k)
			continue
		}
		if src[parent.valStart] != '{' {
			return fail(k.parent + " is not a JSON object")
		}
		child, ok := scanObject(src, parent.valStart).member(src, k.child)
		if !ok {
			missing = append(missing, k)
			continue
		}
		if raw := src[child.valStart:child.valEnd]; !sameJSON(raw, k.value) {
			kept = append(kept, keptLine(k, raw))
		}
	}
	res.Section.Steps = appendBuildStep(res.Section.Steps, in, agentIsGabo)

	out := bytes.Clone(src)
	var added []string
	for _, k := range missing {
		top := scanObject(out, skipWS(out, 0))
		if k.parent == "" {
			out = insertMember(out, top, jsonKV{k.child, k.value}, indentUnit(out, top))
		} else if parent, ok := top.member(out, k.parent); ok {
			out = insertMember(out, scanObject(out, parent.valStart), jsonKV{k.child, k.value}, indentUnit(out, top))
		} else {
			out = insertMember(out, top, jsonKV{k.parent, jsonObj{{k.child, k.value}}}, indentUnit(out, top))
		}
		added = append(added, k.name())
	}
	var lines []string
	if len(added) > 0 {
		lines = append(lines, wroteLine(added))
	}
	if removesAgent(in) {
		var removed bool
		if out, removed = removeGaboAgent(out); removed {
			lines = append(lines, removedLine(in))
		}
	}
	if len(lines) == 0 {
		if len(kept) == 0 {
			kept = []string{"unchanged " + ClaudeSettings}
		}
		res.Section.Lines = kept
		return res
	}
	res.Writes = map[string][]byte{ClaudeSettings: out}
	res.Section.Lines = append(lines, kept...)
	return res
}

// claudeRemoval removes the gabo main-session value from the settings file
// of a project whose engine set no longer holds Claude Code (intent 0010
// S4, D2). Only that member changes; a missing or unreadable file, any
// other value, or no assistant engine changes nothing.
func claudeRemoval(in Input) Result {
	src, ok := in.Files[ClaudeSettings]
	if !ok || !removesAgent(in) {
		return Result{}
	}
	var doc any
	if json.Unmarshal(src, &doc) != nil {
		return Result{}
	}
	if _, ok := doc.(map[string]any); !ok {
		return Result{}
	}
	out, removed := removeGaboAgent(src)
	if !removed {
		return Result{}
	}
	return Result{Section: Section{Lines: []string{removedLine(in)}}, Writes: map[string][]byte{ClaudeSettings: out}}
}

// removesAgent reports whether another engine is the assistant, so the gabo
// main-session value must leave the Claude Code settings (intent 0010 S4).
func removesAgent(in Input) bool { return in.Assistant != "" && in.Assistant != Claude }

// removeGaboAgent deletes the top-level agent member whose value is exactly
// the gabo value, with its separating comma; every other byte is kept. src
// must be valid JSON with an object at the top level.
func removeGaboAgent(src []byte) ([]byte, bool) {
	top := scanObject(src, skipWS(src, 0))
	for i, m := range top.members {
		if m.key != claudeAgentKey || !sameJSON(src[m.valStart:m.valEnd], claudeAgentValue) {
			continue
		}
		var from, to int
		switch {
		case i+1 < len(top.members): // the comma after it and the space up to the next key
			from, to = m.keyStart, top.members[i+1].keyStart
		case i > 0: // the last member: the comma before it
			from, to = top.members[i-1].valEnd, m.valEnd
		default: // the only member
			from, to = top.open+1, top.close
		}
		return append(bytes.Clone(src[:from]), src[to:]...), true
	}
	return src, false
}

// appendBuildStep adds the S8 build step when the main session is gabo's
// agent and its agent file is not in the project yet.
func appendBuildStep(steps []string, in Input, agentIsGabo bool) []string {
	if _, built := in.Files[ClaudeAgentFile]; in.Assistant == Claude && agentIsGabo && !built {
		return append(steps, claudeBuildStep)
	}
	return steps
}

func wroteLine(added []string) string {
	return "wrote " + ClaudeSettings + ": added " + strings.Join(added, ", ")
}

func keptLine(k claudeKey, raw []byte) string {
	var compact bytes.Buffer
	_ = json.Compact(&compact, raw)
	return "kept " + ClaudeSettings + " " + k.name() + ": " + compact.String() + " (your setting; gabo leaves it)"
}

func removedLine(in Input) string {
	return "removed " + ClaudeSettings + " " + claudeAgentKey + ": " + quoteJSON(claudeAgentValue) + " (runtimes.assistant is " + in.Assistant + ")"
}

// sameJSON reports whether raw decodes to the same value as want.
func sameJSON(raw []byte, want any) bool {
	var got, exp any
	if json.Unmarshal(raw, &got) != nil {
		return false
	}
	_ = json.Unmarshal([]byte(renderJSON(want, "", "", false)), &exp)
	return reflect.DeepEqual(got, exp)
}

// jsonObj is a JSON object with its members in order; jsonKV is one member.
// Values are jsonObj, string or bool.
type jsonObj []jsonKV

type jsonKV struct {
	key   string
	value any
}

// renderJSON renders v. Multiline puts each member on its own line at
// indent+unit and the closing brace at indent; inline uses ": " and ", ".
func renderJSON(v any, indent, unit string, multiline bool) string {
	obj, ok := v.(jsonObj)
	if !ok {
		b, _ := json.Marshal(v)
		return string(b)
	}
	parts := make([]string, len(obj))
	for i, kv := range obj {
		parts[i] = quoteJSON(kv.key) + ": " + renderJSON(kv.value, indent+unit, unit, multiline)
	}
	if !multiline {
		return "{" + strings.Join(parts, ", ") + "}"
	}
	if len(parts) == 0 {
		return "{}"
	}
	return "{\n" + indent + unit + strings.Join(parts, ",\n"+indent+unit) + "\n" + indent + "}"
}

func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// spanObject is the byte layout of one JSON object in a valid document.
type spanObject struct {
	open, close int // offsets of '{' and '}'
	members     []spanMember
}

// spanMember is one member: where its key starts and its value's span.
type spanMember struct {
	key              string
	keyStart         int
	valStart, valEnd int
}

func (o spanObject) member(src []byte, key string) (spanMember, bool) {
	for _, m := range o.members {
		if m.key == key {
			return m, true
		}
	}
	return spanMember{}, false
}

// scanObject reads the object starting at src[i] == '{'. src must be valid
// JSON (checked by the caller with json.Unmarshal).
func scanObject(src []byte, i int) spanObject {
	o := spanObject{open: i}
	i = skipWS(src, i+1)
	for src[i] != '}' {
		if src[i] == ',' {
			i = skipWS(src, i+1)
			continue
		}
		keyEnd := skipValue(src, i)
		var key string
		_ = json.Unmarshal(src[i:keyEnd], &key)
		valStart := skipWS(src, skipWS(src, keyEnd)+1) // past ':'
		valEnd := skipValue(src, valStart)
		o.members = append(o.members, spanMember{key: key, keyStart: i, valStart: valStart, valEnd: valEnd})
		i = skipWS(src, valEnd)
	}
	o.close = i
	return o
}

func skipWS(src []byte, i int) int {
	for i < len(src) && strings.IndexByte(" \t\r\n", src[i]) >= 0 {
		i++
	}
	return i
}

// skipValue returns the offset just past the value starting at src[i].
func skipValue(src []byte, i int) int {
	switch src[i] {
	case '"':
		for i++; src[i] != '"'; i++ {
			if src[i] == '\\' {
				i++
			}
		}
		return i + 1
	case '{', '[':
		depth := 0
		for ; ; i++ {
			switch src[i] {
			case '"':
				i = skipValue(src, i) - 1
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	default:
		for i < len(src) && strings.IndexByte(",}] \t\r\n", src[i]) < 0 {
			i++
		}
		return i
	}
}

// lineIndent returns the leading whitespace of the line holding src[i].
func lineIndent(src []byte, i int) string {
	start := bytes.LastIndexByte(src[:i], '\n') + 1
	end := start
	for end < len(src) && (src[end] == ' ' || src[end] == '\t') {
		end++
	}
	return string(src[start:end])
}

// multiline reports whether the object puts its members on their own lines;
// an empty object is multiline unless the whole document is one line.
func multiline(src []byte, o spanObject) bool {
	if len(o.members) == 0 {
		return bytes.Contains(bytes.TrimSpace(src), []byte("\n")) || len(bytes.TrimSpace(src)) == 2
	}
	return bytes.Contains(src[o.open:o.members[0].keyStart], []byte("\n"))
}

// indentUnit is the file's indent step, taken from the top-level object's
// first member; two spaces when the file shows none.
func indentUnit(src []byte, top spanObject) string {
	if len(top.members) > 0 && multiline(src, top) {
		if unit := strings.TrimPrefix(lineIndent(src, top.members[0].keyStart), lineIndent(src, top.open)); unit != "" {
			return unit
		}
	}
	return "  "
}

// insertMember adds kv as the last member of o, in the object's layout:
// after the last member on a new line at its indent, or inline after ", ".
func insertMember(src []byte, o spanObject, kv jsonKV, unit string) []byte {
	ml := multiline(src, o)
	var at, end int
	var text string
	if len(o.members) == 0 {
		at, end = o.open+1, o.close
		if ml {
			indent := lineIndent(src, o.open)
			text = "\n" + indent + unit + quoteJSON(kv.key) + ": " + renderJSON(kv.value, indent+unit, unit, true) + "\n" + indent
		} else {
			text = quoteJSON(kv.key) + ": " + renderJSON(kv.value, "", "", false)
		}
	} else {
		last := o.members[len(o.members)-1]
		at, end = last.valEnd, last.valEnd
		if ml {
			indent := lineIndent(src, last.keyStart)
			text = ",\n" + indent + quoteJSON(kv.key) + ": " + renderJSON(kv.value, indent, unit, true)
		} else {
			text = ", " + quoteJSON(kv.key) + ": " + renderJSON(kv.value, "", "", false)
		}
	}
	out := make([]byte, 0, len(src)+len(text))
	out = append(out, src[:at]...)
	out = append(out, text...)
	return append(out, src[end:]...)
}
