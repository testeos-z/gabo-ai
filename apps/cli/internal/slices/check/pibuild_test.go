package check

import (
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// piConfig runs every role on Pi and points two slots at catalog adapters.
const piConfig = `repo:
  name: o/r
  integration_branch: dev
runtimes:
  coder: pi
  reviewer: pi
  assistant: pi
parallelism: 3
review:
  checks: [go test ./...]
  human_only_paths: [.github/]
slots:
  reviewer_placement:
    adapter: .gabo/skills/gabo-reviewer-placement-local/SKILL.md
  agent_host:
    adapter: .gabo/skills/gabo-agent-host-local/SKILL.md
`

// piTools maps each spec capability to its Pi tools (gabo-runtime-pi, build
// step Agents, with gentle-pi as the subagent package and no web package).
var piTools = map[string][]string{
	"read": {"read"}, "skills": {"read"},
	"search": {"grep", "find", "ls"},
	"edit":   {"edit", "write"},
	"shell":  {"bash"}, "git": {"bash"}, "tracker": {"bash"}, "repository-host": {"bash"}, "memory": {"bash"},
	"subagents": {"subagent_run", "subagent_status", "subagent_result", "subagent_continue", "subagent_reply", "subagent_cancel", "subagent_list_agents"},
}

// fixedNameLinks are the template slots that name a fixed-name link, with the
// link name a runtime skill writes for them, in the order it lists them.
var fixedNameLinks = [][2]string{{"{{methodology}}", "gabo-methodology"}, {"{{adapter:tasks}}", "gabo-tasks"}, {"{{adapter:repository_host}}", "gabo-repository-host"}, {"{{adapter:memory}}", "gabo-memory"}}

// agentSkillList is what a runtime skill lists for agent a from the real
// catalog: the spec skills, then the fixed-name links its template names.
func agentSkillList(t *testing.T, a agentspec.Agent) (list []string, rest string) {
	t.Helper()
	tmpl, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, a.Template))
	if err != nil {
		t.Fatal(err)
	}
	_, rest, _ = strings.Cut(string(tmpl), "\n")
	list = slices.Clone(a.Skills)
	for _, l := range fixedNameLinks {
		if strings.Contains(string(tmpl), l[0]) {
			list = append(list, l[1])
		}
	}
	return list, strings.TrimPrefix(rest, "\n")
}

// skillsLine is the Skills: body line the Pi and OpenCode runtime skills
// write: each listed skill is read at where, with <name> in it; "" for an
// empty list (the line is omitted).
func skillsLine(where string, list []string) string {
	if len(list) == 0 {
		return ""
	}
	return "Skills: before work read " + where + " for each of " + strings.Join(list, ", ") + ".\n"
}

// setSkillsLine replaces the Skills: line of an agent file with skillsLine,
// or drops it for an empty list.
func setSkillsLine(t *testing.T, proj fstest.MapFS, name, where string, list []string) {
	t.Helper()
	var out []string
	found := false
	for line := range strings.SplitAfterSeq(string(proj[name].Data), "\n") {
		if strings.HasPrefix(line, "Skills: ") {
			found = true
			line = skillsLine(where, list)
		}
		out = append(out, line)
	}
	if !found {
		t.Fatalf("%s: no Skills: line", name)
	}
	proj[name] = file(strings.Join(out, ""))
}

// piWhere is the path gabo-runtime-pi writes in the Skills: line.
const piWhere = "`.pi/skills/<name>/SKILL.md`"

// piBuilt is a project with the real embedded catalog (gabo-runtime-pi
// included) and every spec agent built in the shape gabo-runtime-pi writes:
// .pi/agents/<id>.md with frontmatter name, description, gabo-version and
// tools, then the Skills: body line (omitted for an empty list) and the
// filled template body; plus the .pi/settings.json gabo init writes.
func piBuilt(t *testing.T, version string) fstest.MapFS {
	t.Helper()
	proj := fstest.MapFS{".gabo/VERSION": file(version + "\n"), ".pi/settings.json": file(piKeysJSON)}
	files, err := skills.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		proj[path.Join(CatalogDir, f.Path)] = &fstest.MapFile{Data: f.Data}
	}
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		list, rest := agentSkillList(t, a)
		var tools []string
		for _, c := range a.Tools {
			for _, tool := range piTools[c] {
				if !slices.Contains(tools, tool) {
					tools = append(tools, tool)
				}
			}
		}
		front := "---\nname: " + a.ID + "\n" +
			`description: "` + strings.ReplaceAll(a.Role, `"`, `\"`) + "\"\n" +
			"gabo-version: " + version + "\ntools:\n"
		for _, tool := range tools {
			front += "  - " + tool + "\n"
		}
		body := strings.NewReplacer("{{id}}", a.ID, "{{role}}", a.Role).Replace(rest)
		if line := skillsLine(piWhere, list); line != "" {
			body = line + "\n" + body
		}
		proj[".pi/agents/"+a.ID+".md"] = file(front + "---\n" + body)
	}
	return proj
}

func runPi(t *testing.T, proj fs.FS, s Settings) Report {
	t.Helper()
	return run(t, proj, configFiles{projectFile: piConfig}, s)
}

// The agents gabo-runtime-pi builds pass gabo check, with no skill line.
func TestPiBuiltSetPassesCheck(t *testing.T) {
	t.Parallel()
	rep := runPi(t, piBuilt(t, "1.2.3"), Settings{})
	wantSkillLines(t, rep, ".pi/agents/gabo-pm.md", nil)
	wantPassed(t, rep, PassedLine)
}

// Intent 0008 S1 on the agents gabo-runtime-pi builds: the Skills: line
// holds exactly the spec skills plus the template's fixed-name links, each
// once; gabo-dev-lead keeps gabo-plan-reviewer throughout.
func TestPiBuiltAgentSkills(t *testing.T) {
	t.Parallel()
	for _, tt := range agentSkillsCases("pi") {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proj := piBuilt(t, "1.2.3")
			setSkillsLine(t, proj, ".pi/agents/gabo-pm.md", piWhere, tt.list)
			wantSkillLines(t, runPi(t, proj, Settings{}), ".pi/agents/gabo-pm.md", tt.want)
		})
	}
}

// The agent with an empty spec list and no Skills: line passes.
func TestPiBuiltReviewerHasNoSkillsLine(t *testing.T) {
	t.Parallel()
	proj := piBuilt(t, "1.2.3")
	if strings.Contains(string(proj[".pi/agents/gabo-reviewer.md"].Data), "\nSkills: ") {
		t.Fatalf("gabo-reviewer has a Skills: line:\n%s", proj[".pi/agents/gabo-reviewer.md"].Data)
	}
	wantSkillLines(t, runPi(t, proj, Settings{}), ".pi/agents/gabo-reviewer.md", nil)
}

// The directory the Skills: line names does not change the result.
func TestPiBuiltAgentSkillsIgnoreTheDirectory(t *testing.T) {
	t.Parallel()
	missing := slices.DeleteFunc(slices.Clone(pmSkills), func(s string) bool { return s == "gabo-plan-reviewer" })
	for _, where := range []string{"`.pi/skills/<name>/SKILL.md`", ".opencode/skills/<name>/SKILL.md", "`.agents/skills/<name>/SKILL.md`"} {
		t.Run(where, func(t *testing.T) {
			t.Parallel()
			proj := piBuilt(t, "1.2.3")
			setSkillsLine(t, proj, ".pi/agents/gabo-pm.md", where, pmSkills)
			wantSkillLines(t, runPi(t, proj, Settings{}), ".pi/agents/gabo-pm.md", nil)
			setSkillsLine(t, proj, ".pi/agents/gabo-pm.md", where, missing)
			wantSkillLines(t, runPi(t, proj, Settings{}), ".pi/agents/gabo-pm.md",
				[]string{"✗ agent gabo-pm (pi): skill gabo-plan-reviewer missing"})
		})
	}
}
