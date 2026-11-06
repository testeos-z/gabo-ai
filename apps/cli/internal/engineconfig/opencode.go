package engineconfig

import (
	"bytes"
	"encoding/json"
	"maps"
	"math"
	"strconv"
	"strings"
)

// OpencodeStageGate is the stage-gate plugin that gabo-runtime-opencode
// writes on every build (intent 0004). The OpenCode rule only reports
// whether it exists; gabo init never writes it (intent 0006 S6).
const OpencodeStageGate = ".opencode/plugins/gabo-stage-gate.js"

// OpencodeConfigs is OpenCode's project config lookup order (spec 0004 S8):
// the main-session key goes into the first existing file, and gabo's value
// is removed from each of them (intent 0010 S2, S4).
var OpencodeConfigs = []string{"opencode.json", "opencode.jsonc", ".opencode/opencode.json", ".opencode/opencode.jsonc"}

// OpencodeNewConfig is the config gabo init creates when none exists.
const OpencodeNewConfig = ".opencode/opencode.json"

// OpencodeAgentFile is the agent file the OpenCode main session runs; the
// runtime skill builds it (intent 0010 S8).
const OpencodeAgentFile = ".opencode/agents/gabo-assistant.md"

// The top-level main-session key and its gabo value (intent 0010 S2).
const (
	opencodeAgentKey   = "default_agent"
	opencodeAgentValue = "gabo-assistant"
)

// opencodeStep is the step outside the project for OpenCode when the
// plugin is missing; it names only the plugin (intent 0010 S8).
const opencodeStep = "ask your agent to build the agents with gabo-runtime-opencode (it installs " + OpencodeStageGate + ")"

// opencodeMainSessionStep is the S8 step while the main session's agent is
// not built.
const opencodeMainSessionStep = "ask your agent to build the agents (the main session needs " + OpencodeAgentFile + ")"

// opencodeRule writes gabo's OpenCode keys into the S2 file, the first
// existing config, else a new .opencode/opencode.json, keeping every other
// byte and comment (intent 0010 S2, S3, S4): it raises
// experimental.subagent_depth to at least 3; with OpenCode a listed
// assistant engine it merges "default_agent": "gabo-assistant", with OpenCode
// not listed it removes exactly that value from each config (intent 0013
// S3, D7). Anything it cannot merge
// without loss writes nothing and gets one ✗ line. Then it reports whether
// the stage-gate plugin is in the project.
func opencodeRule(in Input) Result {
	res, agentIsGabo := opencodeKeys(in)
	failed := len(res.Section.Lines) > 0 && strings.HasPrefix(res.Section.Lines[0], FailurePrefix)
	if _, ok := in.Files[OpencodeStageGate]; ok {
		res.Section.Lines = append(res.Section.Lines, "stage-gate plugin in place (built by gabo-runtime-opencode)")
	} else {
		line := "no " + OpencodeStageGate + " yet"
		if len(res.Writes) == 0 && !failed {
			line += "; nothing written"
		}
		res.Section.Lines = append(res.Section.Lines, line)
		res.Section.Steps = append(res.Section.Steps, opencodeStep)
	}
	if _, built := in.Files[OpencodeAgentFile]; agentIsGabo && !failed && !built {
		res.Section.Steps = append(res.Section.Steps, opencodeMainSessionStep)
	}
	return res
}

// The nested key gabo raises and its floor (intent 0010 S3, D5).
const (
	opencodeExperimental = "experimental"
	opencodeDepthKey     = "subagent_depth"
	opencodeDepthName    = opencodeExperimental + "." + opencodeDepthKey
	opencodeMinDepth     = 3
)

// opencodeKeys decides every key before it writes, so a failure writes
// nothing, and reports whether the main session runs gabo's agent.
func opencodeKeys(in Input) (Result, bool) {
	file := opencodeConfig(in)
	fail := func(reason string) (Result, bool) {
		return Result{Section: Section{Lines: []string{opencodeFailure(file, reason)}}}, false
	}
	writes := map[string][]byte{}
	var removed []string
	if in.unlists(Opencode) {
		rem := removeDefaultAgent(in, true)
		if len(rem.Section.Lines) > 0 && strings.HasPrefix(rem.Section.Lines[0], FailurePrefix) {
			return rem, false
		}
		maps.Copy(writes, rem.Writes)
		removed = rem.Section.Lines
	}

	src, exists := writes[file]
	if !exists {
		src, exists = in.Files[file]
	}
	var parts, kept []string
	agentIsGabo := false
	if !exists {
		obj := jsonObj{{opencodeExperimental, jsonObj{{opencodeDepthKey, opencodeMinDepth}}}}
		parts = append(parts, opencodeDepthName+" unset → "+strconv.Itoa(opencodeMinDepth))
		if in.lists(Opencode) {
			obj = append(obj, jsonKV{opencodeAgentKey, opencodeAgentValue})
			parts = append(parts, "added "+opencodeAgentKey)
			agentIsGabo = true
		}
		src = []byte(renderJSON(obj, "", "  ", true) + "\n")
	} else {
		top, reason := opencodeTop(src)
		if reason != "" {
			return fail(reason)
		}
		out, part, reason := raiseDepth(src, top)
		if reason != "" {
			return fail(reason)
		}
		if part != "" {
			parts = append(parts, part)
		}
		if in.lists(Opencode) {
			top = scanObject(out, top.open)
			switch m, ok := top.member(out, opencodeAgentKey); {
			case !ok:
				out = insertMember(out, top, jsonKV{opencodeAgentKey, opencodeAgentValue}, indentUnit(out, top))
				parts = append(parts, "added "+opencodeAgentKey)
				agentIsGabo = true
			case sameJSON(out[m.valStart:m.valEnd], opencodeAgentValue):
				agentIsGabo = true
			default:
				kept = append(kept, "kept "+file+" "+opencodeAgentKey+": "+compactJSON(out[m.valStart:m.valEnd])+" (your setting; gabo leaves it)")
			}
		}
		src = out
	}

	var res Result
	if len(parts) > 0 {
		writes[file] = src
		res.Section.Lines = append(res.Section.Lines, "wrote "+file+": "+strings.Join(parts, "; "))
	}
	res.Section.Lines = append(res.Section.Lines, removed...)
	res.Section.Lines = append(res.Section.Lines, kept...)
	if len(res.Section.Lines) == 0 && in.lists(Opencode) {
		res.Section.Lines = []string{"unchanged " + file}
	}
	if len(writes) > 0 {
		res.Writes = writes
	}
	return res, agentIsGabo
}

// opencodeConfig is the S2 file: the first existing config, else the new
// one gabo init creates.
func opencodeConfig(in Input) string {
	for _, p := range OpencodeConfigs {
		if _, ok := in.Files[p]; ok {
			return p
		}
	}
	return OpencodeNewConfig
}

// raiseDepth makes experimental.subagent_depth at least 3 in src: it
// returns the new bytes and the report part ("" when the value is already a
// whole number of 3 or more), or why it cannot be merged without loss.
func raiseDepth(src []byte, top spanObject) ([]byte, string, string) {
	unset := opencodeDepthName + " unset → " + strconv.Itoa(opencodeMinDepth)
	switch top.count(opencodeExperimental) {
	case 0:
		return insertMember(src, top, jsonKV{opencodeExperimental, jsonObj{{opencodeDepthKey, opencodeMinDepth}}}, indentUnit(src, top)), unset, ""
	case 1: // merged below
	default:
		return nil, "", opencodeExperimental + " appears more than once"
	}
	exp, _ := top.member(src, opencodeExperimental)
	if src[exp.valStart] != '{' {
		return nil, "", opencodeExperimental + " is not a JSON object"
	}
	obj := scanObject(src, exp.valStart)
	switch obj.count(opencodeDepthKey) {
	case 0:
		return insertMember(src, obj, jsonKV{opencodeDepthKey, opencodeMinDepth}, indentUnit(src, top)), unset, ""
	case 1: // checked below
	default:
		return nil, "", opencodeDepthName + " appears more than once"
	}
	m, _ := obj.member(src, opencodeDepthKey)
	raw := src[m.valStart:m.valEnd]
	var v any
	n, isNum := 0.0, false
	if json.Unmarshal(raw, &v) == nil {
		n, isNum = v.(float64)
	}
	switch {
	case !isNum || n != math.Trunc(n):
		return nil, "", opencodeDepthName + " is " + compactJSON(raw) + ", not a whole number"
	case n >= opencodeMinDepth:
		return src, "", ""
	}
	return replaceValue(src, m, opencodeMinDepth), opencodeDepthName + " " + compactJSON(raw) + " → " + strconv.Itoa(opencodeMinDepth), ""
}

// compactJSON renders a raw value on one line, as it is when it does not
// compact.
func compactJSON(raw []byte) string {
	var b bytes.Buffer
	if json.Compact(&b, raw) != nil {
		return string(raw)
	}
	return b.String()
}

// opencodeRemoval removes the gabo default_agent from the configs of a
// project whose engine set no longer holds OpenCode (intent 0010 S4, D2);
// a config that does not parse is left alone without a line.
func opencodeRemoval(in Input) Result {
	if !in.unlists(Opencode) {
		return Result{}
	}
	return removeDefaultAgent(in, false)
}

// removeDefaultAgent deletes default_agent from each config whose value is
// exactly gabo's. With strict, a config that does not parse fails the
// whole removal with one ✗ line and nothing is written; without, it is
// skipped.
func removeDefaultAgent(in Input, strict bool) Result {
	var res Result
	for _, file := range OpencodeConfigs {
		src, ok := in.Files[file]
		if !ok {
			continue
		}
		top, reason := opencodeTop(src)
		if reason != "" {
			if strict {
				return Result{Section: Section{Lines: []string{opencodeFailure(file, reason)}}}
			}
			continue
		}
		for i, m := range top.members {
			if m.key != opencodeAgentKey || !sameJSON(src[m.valStart:m.valEnd], opencodeAgentValue) {
				continue
			}
			if res.Writes == nil {
				res.Writes = map[string][]byte{}
			}
			res.Writes[file] = removeMember(src, top, i)
			res.Section.Lines = append(res.Section.Lines, removedMainSessionLine(file, opencodeAgentKey, opencodeAgentValue, in))
			break
		}
	}
	return res
}

// opencodeTop scans the top-level object of a JSONC config, or returns why
// it cannot be merged.
func opencodeTop(src []byte) (spanObject, string) {
	if err := checkJSONC(src); err != nil {
		return spanObject{}, "invalid JSONC (" + err.Error() + ")"
	}
	start := skipWS(src, len(src)-len(bytes.TrimPrefix(src, utf8BOM)))
	if start >= len(src) || src[start] != '{' {
		return spanObject{}, "the top level is not a JSON object"
	}
	return scanObject(src, start), ""
}

func opencodeFailure(file, reason string) string {
	return FailurePrefix + " " + file + ": " + reason + "; fix it and run gabo init again"
}
