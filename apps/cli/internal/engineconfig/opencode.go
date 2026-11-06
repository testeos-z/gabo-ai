package engineconfig

import (
	"bytes"
	"encoding/json"
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

// opencodeRule keeps default_agent in step with the assistant engine
// (intent 0010 S2, S4): with OpenCode as the assistant it merges
// "gabo-assistant" into the first existing config, else a new
// .opencode/opencode.json, keeping every other byte and comment; with
// another assistant it removes exactly that value from each config. A
// config that does not parse writes nothing and gets one ✗ line. Then it
// reports whether the stage-gate plugin is in the project.
func opencodeRule(in Input) Result {
	var res Result
	agentIsGabo := false
	switch {
	case in.Assistant == Opencode:
		res, agentIsGabo = mergeDefaultAgent(in)
	case in.Assistant != "":
		res = removeDefaultAgent(in, true)
	}
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

// opencodeRemoval removes the gabo default_agent from the configs of a
// project whose engine set no longer holds OpenCode (intent 0010 S4, D2);
// a config that does not parse is left alone without a line.
func opencodeRemoval(in Input) Result {
	if in.Assistant == "" || in.Assistant == Opencode {
		return Result{}
	}
	return removeDefaultAgent(in, false)
}

// mergeDefaultAgent writes default_agent into the first existing config and
// reports whether the main session runs gabo's agent.
func mergeDefaultAgent(in Input) (Result, bool) {
	file := OpencodeNewConfig
	for _, p := range OpencodeConfigs {
		if _, ok := in.Files[p]; ok {
			file = p
			break
		}
	}
	src, exists := in.Files[file]
	if !exists {
		out := renderJSON(jsonObj{{opencodeAgentKey, opencodeAgentValue}}, "", "  ", true) + "\n"
		return Result{
			Section: Section{Lines: []string{"wrote " + file + ": added " + opencodeAgentKey}},
			Writes:  map[string][]byte{file: []byte(out)},
		}, true
	}
	top, reason := opencodeTop(src)
	if reason != "" {
		return Result{Section: Section{Lines: []string{opencodeFailure(file, reason)}}}, false
	}
	m, ok := top.member(src, opencodeAgentKey)
	switch {
	case !ok:
		out := insertMember(src, top, jsonKV{opencodeAgentKey, opencodeAgentValue}, indentUnit(src, top))
		return Result{
			Section: Section{Lines: []string{"wrote " + file + ": added " + opencodeAgentKey}},
			Writes:  map[string][]byte{file: out},
		}, true
	case sameJSON(src[m.valStart:m.valEnd], opencodeAgentValue):
		return Result{Section: Section{Lines: []string{"unchanged " + file}}}, true
	default:
		var compact bytes.Buffer
		if json.Compact(&compact, src[m.valStart:m.valEnd]) != nil {
			compact.Write(src[m.valStart:m.valEnd])
		}
		return Result{Section: Section{Lines: []string{"kept " + file + " " + opencodeAgentKey + ": " + compact.String() + " (your setting; gabo leaves it)"}}}, false
	}
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
			res.Section.Lines = append(res.Section.Lines, "removed "+file+" "+opencodeAgentKey+": "+quoteJSON(opencodeAgentValue)+" (runtimes.assistant is "+in.Assistant+")")
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
