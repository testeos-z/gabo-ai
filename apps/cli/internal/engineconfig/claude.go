package engineconfig

import (
	"bytes"
	"encoding/json"
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
// main-session key when Claude Code is a listed assistant engine (intent
// 0010 S1; intent 0013 S3).
func wantedKeys(in Input) []claudeKey {
	keys := slices.Clone(claudeKeys)
	if in.lists(Claude) {
		keys = append(keys, claudeKey{"", claudeAgentKey, claudeAgentValue})
	}
	return keys
}

// claudeRule merges the S2 keys into .claude/settings.json, keeping every
// existing byte (D5): a missing key is inserted, a key with another value is
// kept and reported (S4, D6), and a file that does not parse is left alone
// with one ✗ line (S5). The top-level agent key follows the assistant
// engines (intent 0010 S1, S4; intent 0013 S3, D7): Claude Code listed
// merges "gabo-assistant" with the same rule; Claude Code not listed removes
// exactly that value.
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
			lines = append(lines, removedMainSessionLine(ClaudeSettings, claudeAgentKey, claudeAgentValue, in))
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
	return Result{Section: Section{Lines: []string{removedMainSessionLine(ClaudeSettings, claudeAgentKey, claudeAgentValue, in)}}, Writes: map[string][]byte{ClaudeSettings: out}}
}

// removesAgent reports whether Claude Code is not a listed assistant engine
// while another is, so the gabo main-session value must leave the Claude
// Code settings (intent 0010 S4; intent 0013 D7).
func removesAgent(in Input) bool { return in.unlists(Claude) }

// removeGaboAgent deletes the top-level agent member whose value is exactly
// the gabo value, with its separating comma; every other byte is kept. src
// must be valid JSON with an object at the top level.
func removeGaboAgent(src []byte) ([]byte, bool) {
	top := scanObject(src, skipWS(src, 0))
	for i, m := range top.members {
		if m.key == claudeAgentKey && sameJSON(src[m.valStart:m.valEnd], claudeAgentValue) {
			return removeMember(src, top, i), true
		}
	}
	return src, false
}

// appendBuildStep adds the S8 build step when the main session is gabo's
// agent and its agent file is not in the project yet.
func appendBuildStep(steps []string, in Input, agentIsGabo bool) []string {
	if _, built := in.Files[ClaudeAgentFile]; in.lists(Claude) && agentIsGabo && !built {
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
