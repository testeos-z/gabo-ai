package check

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
)

// toolRequirement is one entry of metadata.agent_tool_requirements: the
// tools an agent whose spec lists capability must list (intent 0013 D11).
type toolRequirement struct {
	capability string
	tools      []string
}

// errToolsHalf is a runtime skill declaring one of the two tool keys only.
var errToolsHalf = errors.New("metadata.agent_tools and metadata.agent_tool_requirements: declare both or neither")

// parseToolDeclaration reads metadata.agent_tools (an agent place, intent
// 0008 D1 forms) and metadata.agent_tool_requirements (each capability to a
// list of tool names, in declaration order). Absent keys are not declared;
// a malformed or half declaration is an error.
func parseToolDeclaration(tools, reqs yaml.Node) (agentPlace, []toolRequirement, error) {
	if tools.Kind == 0 && reqs.Kind == 0 {
		return agentPlace{}, nil, nil
	}
	if tools.Kind == 0 || reqs.Kind == 0 {
		return agentPlace{}, nil, errToolsHalf
	}
	if tools.Kind != yaml.ScalarNode {
		return agentPlace{}, nil, errors.New("metadata.agent_tools: want a string <form>:<argument>")
	}
	place, err := parseAgentPlace("agent_tools", tools.Value)
	if err != nil {
		return agentPlace{}, nil, err
	}
	if !place.declared() {
		return agentPlace{}, nil, errToolsHalf
	}
	if reqs.Kind != yaml.MappingNode {
		return agentPlace{}, nil, errors.New("metadata.agent_tool_requirements: want a map of each capability to a list of tool names")
	}
	out := []toolRequirement{}
	for i := 0; i+1 < len(reqs.Content); i += 2 {
		k, v := reqs.Content[i], reqs.Content[i+1]
		capability := strings.TrimSpace(k.Value)
		if k.Kind != yaml.ScalarNode || capability == "" || v.Kind != yaml.SequenceNode {
			return agentPlace{}, nil, fmt.Errorf("metadata.agent_tool_requirements.%s: want a list of tool names", k.Value)
		}
		req := toolRequirement{capability: capability}
		for _, item := range v.Content {
			name := strings.TrimSpace(item.Value)
			if item.Kind != yaml.ScalarNode || name == "" {
				return agentPlace{}, nil, fmt.Errorf("metadata.agent_tool_requirements.%s: want a list of tool names", capability)
			}
			req.tools = append(req.tools, name)
		}
		out = append(out, req)
	}
	return place, out, nil
}

// parseUnavailableTools reads metadata.agent_tools_unavailable, the tools
// a subagent of the engine never has, in declaration order (intent 0013
// S20). Absent is not declared; it needs the agent_tools declaration
// (declared) to know where an agent file lists its tools.
func parseUnavailableTools(node yaml.Node, declared bool) ([]string, error) {
	if node.Kind == 0 {
		return nil, nil
	}
	if !declared {
		return nil, errors.New("metadata.agent_tools_unavailable: needs metadata.agent_tools and metadata.agent_tool_requirements")
	}
	if node.Kind != yaml.SequenceNode {
		return nil, errors.New("metadata.agent_tools_unavailable: want a list of tool names")
	}
	out := []string{}
	for _, item := range node.Content {
		name := strings.TrimSpace(item.Value)
		if item.Kind != yaml.ScalarNode || name == "" {
			return nil, errors.New("metadata.agent_tools_unavailable: want a list of tool names")
		}
		out = append(out, name)
	}
	return out, nil
}

// unavailableListed returns the tools of listed that match a pattern of
// unavailable (a trailing * is a prefix match), in file order, each once.
func unavailableListed(unavailable, listed []string) []string {
	var out []string
	for _, l := range listed {
		if slices.Contains(out, l) {
			continue
		}
		for _, p := range unavailable {
			if prefix, ok := strings.CutSuffix(p, "*"); l == p || ok && strings.HasPrefix(l, prefix) {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

// declaresTools reports whether the runtime skill declares both tool keys
// well formed.
func (m runtimeMeta) declaresTools() bool {
	return m.ToolsErr == nil && m.AgentTools.declared() && m.ToolRequirements != nil
}

// toolListed reports whether listed meets the required tool: by its name,
// or for a required mem_x by mcp__<server>__mem_x or mcp__<server>__*
// (intent 0013 D11).
func toolListed(required string, listed []string) bool {
	for _, l := range listed {
		if l == required {
			return true
		}
		if !strings.HasPrefix(required, "mem_") {
			continue
		}
		rest, ok := strings.CutPrefix(l, "mcp__")
		if !ok {
			continue
		}
		server, tool, ok := strings.Cut(rest, "__")
		if ok && server != "" && (tool == required || tool == "*") {
			return true
		}
	}
	return false
}

// missingTools returns the tools capabilities require that listed does not
// meet, in requirement order, each once.
func missingTools(reqs []toolRequirement, capabilities, listed []string) []string {
	var out []string
	for _, req := range reqs {
		if !slices.Contains(capabilities, req.capability) {
			continue
		}
		for _, tool := range req.tools {
			if !slices.Contains(out, tool) && !toolListed(tool, listed) {
				out = append(out, tool)
			}
		}
	}
	return out
}

// checkAgentTools fails, for a runtime skill that declares
// metadata.agent_tools and metadata.agent_tool_requirements, each built spec
// agent whose file lacks a tool its spec capabilities require, with one line
// per file (intent 0013 S17, D11); for one that also declares
// metadata.agent_tools_unavailable, each built agent file that lists a tool
// a subagent never has, with one line per file (S20). A malformed
// declaration fails once with NextRestoreCatalog; a runtime without the
// declaration is not checked.
func (u *UseCase) checkAgentTools(r *Report, rt string, meta runtimeMeta, spec agentspec.Spec) error {
	if meta.ToolsErr != nil {
		r.fail("✗ runtime "+rt+": gabo-runtime-"+rt+" "+meta.ToolsErr.Error(), NextRestoreCatalog)
		return nil
	}
	if !meta.declaresTools() || !insideRepo(meta.AgentDir) {
		return nil // agentDir reports a directory outside the repository
	}
	dir := path.Clean(meta.AgentDir)
	files, err := u.agentFiles(dir)
	if err != nil {
		return err
	}
	for _, a := range spec.Agents {
		for _, name := range files[a.ID] {
			file := path.Join(dir, name)
			b, err := fs.ReadFile(u.project, file)
			if err != nil {
				return fmt.Errorf("read %s: %w", file, err)
			}
			listed := meta.AgentTools.skills(b)
			if missing := missingTools(meta.ToolRequirements, a.Tools, listed); len(missing) > 0 {
				r.fail("✗ "+file+": tools missing for its spec capabilities: "+strings.Join(missing, ", "), NextRebuild)
			}
			if never := unavailableListed(meta.UnavailableTools, listed); len(never) > 0 {
				r.fail("✗ "+file+": tools a subagent never has: "+strings.Join(never, ", "), NextRebuild)
			}
		}
	}
	return nil
}
