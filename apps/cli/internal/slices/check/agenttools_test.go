package check

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// toolsSkill is a Pi runtime skill that declares where an agent file lists
// its tools and the tools each capability requires (intent 0013 D11).
const toolsSkill = "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n  agent_skills: frontmatter:skills\n" +
	"  agent_tools: frontmatter:tools\n" + toolsReqs + "---\n"

// toolsReqs is the agent_tool_requirements block of toolsSkill.
const toolsReqs = "  agent_tool_requirements:\n    read: [read]\n" +
	"    memory: [mem_search, mem_save, mem_update, mem_get_observation]\n    web: []\n"

// toolsSpec gives alpha the memory capability.
var toolsSpec = strings.Replace(spec, "    tools: [read]\n    launched_by: main-session\n",
	"    tools: [read, web, memory]\n    launched_by: main-session\n", 1)

// piToolsProject is piCoder with toolsSkill, toolsSpec, alpha listing tools
// and beta listing read.
func piToolsProject(t *testing.T, skill, alphaTools string) Report {
	t.Helper()
	if toolsSpec == spec {
		t.Fatal("spec has no alpha tools line to replace")
	}
	proj, cfg := piCoder("---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: pi\n" + alphaTools + "---\nbody\n")
	proj = with(proj, map[string]string{
		".gabo/skills/gabo-runtime-pi/SKILL.md":         skill,
		".gabo/skills/gabo-agent-templates/agents.yaml": toolsSpec,
		".pi/agents/beta.md":                            "---\nname: beta\ngabo-version: 1.2.3\ngabo-engine: pi\ntools: [read]\n---\nbody\n",
	})
	return run(t, proj, configFiles{projectFile: cfg}, Settings{})
}

// toolLines returns the tools-missing lines of r.
func toolLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		if strings.Contains(l, "tools missing") {
			out = append(out, l)
		}
	}
	return out
}

// Intent 0013 S17: a built agent that lacks a tool its spec capabilities
// require gives exactly one line, missing tools in requirement order, and
// the rebuild hint; an mcp__<server>__ name or wildcard meets a mem_ tool.
func TestAgentToolsMissingFails(t *testing.T) {
	t.Parallel()
	all := "tools: [read, mem_search, mem_save, mem_update, mem_get_observation]\n"
	tests := []struct {
		name, tools string
		want        string
	}{
		{"one missing", "tools: [read, mem_search, mem_update, mem_get_observation]\n", "mem_save"},
		{"several in requirement order", "tools: [mem_get_observation, mem_update, read]\n", "mem_search, mem_save"},
		{"none listed", "", "read, mem_search, mem_save, mem_update, mem_get_observation"},
		{"all listed", all, ""},
		{"comma string", "tools: read, mem_search, mem_save, mem_update, mem_get_observation\n", ""},
		{"mcp name", "tools: [read, mem_search, mcp__engram__mem_save, mem_update, mem_get_observation]\n", ""},
		{"mcp wildcard", "tools: [read, mcp__engram__*]\n", ""},
		{"mcp wildcard does not meet a plain tool", "tools: [mcp__engram__*]\n", "read"},
		{"mcp name without a server", "tools: [read, mcp____mem_save, mem_search, mem_update, mem_get_observation]\n", "mem_save"},
		{"mcp name of another tool", "tools: [read, mcp__engram__mem_search, mem_update, mem_get_observation]\n", "mem_save"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := piToolsProject(t, toolsSkill, tt.tools)
			got := toolLines(r)
			if tt.want == "" {
				wantPassed(t, r, PassedLine)
				return
			}
			line := "✗ .pi/agents/alpha.md: tools missing for its spec capabilities: " + tt.want
			if !slices.Equal(got, []string{line}) {
				t.Fatalf("tools lines = %q, want exactly %q:\n%s", got, line, joined(r))
			}
			i := slices.Index(r.Lines, line)
			if i+1 >= len(r.Lines) || r.Lines[i+1] != NextRebuild {
				t.Fatalf("line %q not followed by %q:\n%s", line, NextRebuild, joined(r))
			}
			wantFailed(t, r)
		})
	}
}

// An engine whose runtime skill declares no tools gives no line.
func TestAgentToolsNotCheckedWithoutDeclaration(t *testing.T) {
	t.Parallel()
	noDecl := "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n  agent_skills: frontmatter:skills\n---\n"
	r := piToolsProject(t, noDecl, "")
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "tools missing")
	noLineWith(t, r, "agent_tool")
	r = run(t, project(), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}

// A malformed declaration gives one line per runtime with the existing
// NextRestoreCatalog hint and checks no agent.
func TestAgentToolsMalformedDeclarationFails(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, skill, line string }{
		{"requirements not a map",
			strings.Replace(toolsSkill, toolsReqs, "  agent_tool_requirements: read\n", 1),
			"✗ runtime pi: gabo-runtime-pi metadata.agent_tool_requirements: want a map of each capability to a list of tool names"},
		{"unknown form",
			strings.Replace(toolsSkill, "agent_tools: frontmatter:tools", "agent_tools: json:tools", 1),
			`✗ runtime pi: gabo-runtime-pi metadata.agent_tools "json:tools": unknown form, want frontmatter:<key>, toml:<key> or line:<prefix>`},
		{"only agent_tools",
			strings.Replace(toolsSkill, toolsReqs, "", 1),
			"✗ runtime pi: gabo-runtime-pi metadata.agent_tools and metadata.agent_tool_requirements: declare both or neither"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := piToolsProject(t, tt.skill, "")
			wantFailed(t, r, tt.line, NextRestoreCatalog)
			if i := slices.Index(r.Lines, tt.line); r.Lines[i+1] != NextRestoreCatalog {
				t.Fatalf("line %q not followed by %q:\n%s", tt.line, NextRestoreCatalog, joined(r))
			}
			if n := len(slices.DeleteFunc(slices.Clone(r.Lines), func(l string) bool { return !strings.Contains(l, "metadata.agent_tool") })); n != 1 {
				t.Fatalf("declaration reported %d times, want once:\n%s", n, joined(r))
			}
			noLineWith(t, r, "tools missing")
		})
	}
}

// The command output of a Pi agent without mem_save, pinned as
// testdata/golden/agent-tools-missing.golden (go test -update writes it).
func TestAgentToolsMissingGolden(t *testing.T) {
	t.Parallel()
	proj, cfg := piCoder("---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: pi\n" +
		"tools: [read, mem_search, mem_update, mem_get_observation]\n---\nbody\n")
	proj = with(proj, map[string]string{
		".gabo/skills/gabo-runtime-pi/SKILL.md":         toolsSkill,
		".gabo/skills/gabo-agent-templates/agents.yaml": toolsSpec,
		".pi/agents/beta.md":                            "---\nname: beta\ngabo-version: 1.2.3\ngabo-engine: pi\ntools: [read]\n---\nbody\n",
	})
	commandGolden(t, proj, cfg, "agent-tools-missing.golden")
}

// unavailableSkill is toolsSkill declaring that a subagent never has a
// subagent_* tool (intent 0013 S20).
var unavailableSkill = strings.TrimSuffix(toolsSkill, "---\n") + "  agent_tools_unavailable: [subagent_*, task]\n---\n"

// unavailableLines returns the never-has lines of r.
func unavailableLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		if strings.Contains(l, "never has") {
			out = append(out, l)
		}
	}
	return out
}

// Intent 0013 S20 (plan T22): a built agent that lists a tool its runtime
// skill says a subagent never has gives exactly one line, names in file
// order, and the rebuild hint; a trailing * is a prefix match.
func TestAgentToolsUnavailableFails(t *testing.T) {
	t.Parallel()
	if !strings.HasSuffix(unavailableSkill, "  agent_tools_unavailable: [subagent_*, task]\n---\n") {
		t.Fatalf("unavailableSkill = %q", unavailableSkill)
	}
	all := "read, mem_search, mem_save, mem_update, mem_get_observation"
	tests := []struct{ name, tools, want string }{
		{"one listed", "tools: [" + all + ", subagent_run]\n", "subagent_run"},
		{"several in file order", "tools: [subagent_status, " + all + ", task, subagent_run]\n", "subagent_status, task, subagent_run"},
		{"listed twice", "tools: [subagent_run, " + all + ", subagent_run]\n", "subagent_run"},
		{"comma string", "tools: " + all + ", subagent_list_agents\n", "subagent_list_agents"},
		{"none listed", "tools: [" + all + "]\n", ""},
		{"exact name is not a prefix", "tools: [" + all + ", tasks]\n", ""},
		{"prefix needs the whole stem", "tools: [" + all + ", subagent, subagentrun]\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := piToolsProject(t, unavailableSkill, tt.tools)
			noLineWith(t, r, "tools missing")
			got := unavailableLines(r)
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("never-has lines = %q, want none", got)
				}
				wantPassed(t, r, PassedLine)
				return
			}
			line := "✗ .pi/agents/alpha.md: tools a subagent never has: " + tt.want
			if !slices.Equal(got, []string{line}) {
				t.Fatalf("never-has lines = %q, want exactly %q:\n%s", got, line, joined(r))
			}
			i := slices.Index(r.Lines, line)
			if i+1 >= len(r.Lines) || r.Lines[i+1] != NextRebuild {
				t.Fatalf("line %q not followed by %q:\n%s", line, NextRebuild, joined(r))
			}
			wantFailed(t, r)
		})
	}
}

// An agent both missing a required tool and listing an unavailable one
// gets both lines, the missing line first.
func TestAgentToolsMissingAndUnavailable(t *testing.T) {
	t.Parallel()
	r := piToolsProject(t, unavailableSkill, "tools: [read, subagent_run]\n")
	missing := "✗ .pi/agents/alpha.md: tools missing for its spec capabilities: mem_search, mem_save, mem_update, mem_get_observation"
	never := "✗ .pi/agents/alpha.md: tools a subagent never has: subagent_run"
	wantFailed(t, r, missing, never)
	if slices.Index(r.Lines, missing) > slices.Index(r.Lines, never) {
		t.Fatalf("never-has line before the missing line:\n%s", joined(r))
	}
}

// An engine whose runtime skill does not declare agent_tools_unavailable
// gives no never-has line, whatever its agents list.
func TestAgentToolsUnavailableNotCheckedWithoutDeclaration(t *testing.T) {
	t.Parallel()
	r := piToolsProject(t, toolsSkill, "tools: [read, mem_search, mem_save, mem_update, mem_get_observation, subagent_run]\n")
	wantPassed(t, r, PassedLine)
	noLineWith(t, r, "never has")
}

// A malformed agent_tools_unavailable gives one line with the existing
// NextRestoreCatalog hint and checks no agent.
func TestAgentToolsUnavailableMalformedFails(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, skill, line string }{
		{"not a list",
			strings.Replace(unavailableSkill, "[subagent_*, task]", "subagent_*", 1),
			"✗ runtime pi: gabo-runtime-pi metadata.agent_tools_unavailable: want a list of tool names"},
		{"without agent_tools",
			strings.Replace(strings.Replace(unavailableSkill, toolsReqs, "", 1), "  agent_tools: frontmatter:tools\n", "", 1),
			"✗ runtime pi: gabo-runtime-pi metadata.agent_tools_unavailable: needs metadata.agent_tools and metadata.agent_tool_requirements"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := piToolsProject(t, tt.skill, "tools: [subagent_run]\n")
			wantFailed(t, r, tt.line, NextRestoreCatalog)
			if i := slices.Index(r.Lines, tt.line); r.Lines[i+1] != NextRestoreCatalog {
				t.Fatalf("line %q not followed by %q:\n%s", tt.line, NextRestoreCatalog, joined(r))
			}
			if n := len(slices.DeleteFunc(slices.Clone(r.Lines), func(l string) bool { return !strings.Contains(l, "metadata.agent_tool") })); n != 1 {
				t.Fatalf("declaration reported %d times, want once:\n%s", n, joined(r))
			}
			noLineWith(t, r, "never has")
			noLineWith(t, r, "tools missing")
		})
	}
}

// The command output of a Pi agent listing subagent_run, pinned as
// testdata/golden/agent-tools-unavailable.golden (go test -update writes it).
func TestAgentToolsUnavailableGolden(t *testing.T) {
	t.Parallel()
	proj, cfg := piCoder("---\nname: alpha\ngabo-version: 1.2.3\ngabo-engine: pi\n" +
		"tools: [read, mem_search, mem_save, mem_update, mem_get_observation, subagent_run]\n---\nbody\n")
	proj = with(proj, map[string]string{
		".gabo/skills/gabo-runtime-pi/SKILL.md":         unavailableSkill,
		".gabo/skills/gabo-agent-templates/agents.yaml": toolsSpec,
		".pi/agents/beta.md":                            "---\nname: beta\ngabo-version: 1.2.3\ngabo-engine: pi\ntools: [read]\n---\nbody\n",
	})
	commandGolden(t, proj, cfg, "agent-tools-unavailable.golden")
}

// commandGolden runs gabo check on proj and compares exit code, error and
// stdout with testdata/golden/<name> (go test -update writes it).
func commandGolden(t *testing.T, proj fstest.MapFS, cfg, name string) {
	t.Helper()
	// A nil slice would make cobra read os.Args (the test flags).
	out, err := runCommandConfig(t, proj, cfg, []string{}...)
	code := 0
	var ee *ExitError
	if errors.As(err, &ee) {
		code = ee.Code
	} else if err != nil {
		t.Fatalf("err = %v, want an ExitError", err)
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	got := fmt.Sprintf("exit %d\nerror: %q\n--- stdout\n%s", code, msg, out)
	file := path.Join("testdata", "golden", name)
	if *update {
		if err := os.WriteFile(file, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read golden (run with -update to write it): %v", err)
	}
	if got != string(want) {
		t.Fatalf("output changed:\n got: %s\nwant: %s", got, want)
	}
}
