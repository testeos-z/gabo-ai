package check

import (
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// opencodeTree is the testdata OpenCode tree: a gabo.yaml running every role
// on OpenCode, a stand-in gabo-runtime-opencode skill (only its metadata is
// read) and a project opencode.jsonc with comments and trailing commas.
const opencodeTree = "testdata/opencode"

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(path.Join(opencodeTree, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// opencodeBuilt is a project with the real embedded catalog (its own
// gabo-runtime-opencode, which declares metadata.main_session), and every
// spec agent built in the shape of spec 0004 S3
// (.opencode/agents/<id>.md: frontmatter description, mode and permission;
// the body starts with <!-- gabo-version: x.y.z -->, then the Skills: line
// of build step 7, omitted for an empty list);
// each spec skill linked in the shared .agents/skills (intent 0008 S3) as
// .agents/skills/<name> -> ../../.gabo/skills/<name>, the fixed-name links
// of the tree's skill adapters, the tree's .opencode/opencode.jsonc with
// default_agent, and the main-session entry (spec 0009 S1, S2).
func opencodeBuilt(t *testing.T, version string) fstest.MapFS {
	t.Helper()
	proj := fstest.MapFS{".gabo/VERSION": file(version + "\n")}
	files, err := skills.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		proj[path.Join(CatalogDir, f.Path)] = &fstest.MapFile{Data: f.Data}
	}
	proj[".opencode/opencode.jsonc"] = file(withJSONKey(readTestdata(t, "opencode.jsonc"), "default_agent", assistantID))
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range spec.Agents {
		list, rest := agentSkillList(t, a)
		proj[".opencode/agents/"+a.ID+".md"] = file("---\n" +
			`description: "` + strings.ReplaceAll(a.Role, `"`, `\"`) + "\"\n" +
			"mode: subagent\npermission:\n  skill:\n    gabo-reviewer: deny\n---\n" +
			"<!-- gabo-version: " + version + " -->\n" +
			skillsLine(opencodeWhere, list) + rest)
		for _, name := range a.Skills {
			proj[".agents/skills/"+name] = symlink("../../.gabo/skills/" + name)
		}
	}
	proj[".agents/skills/gabo-reviewer-placement"] = symlink("../../.gabo/skills/gabo-reviewer-placement-local")
	proj[".agents/skills/gabo-agent-host"] = symlink("../../.gabo/skills/gabo-agent-host-local")
	// The main-session entry of spec 0009 S1, S2: link, primary agent and
	// the default_agent key above.
	proj[".agents/skills/gabo-assistant"] = symlink("../../.gabo/skills/gabo-assistant")
	proj[".opencode/agents/"+assistantID+".md"] = file("---\ndescription: \"The gabo assistant\"\nmode: primary\n---\n" +
		"<!-- gabo-version: " + version + " -->\n" + assistantBody(t))
	return proj
}

// opencodeWhere is the path gabo-runtime-opencode writes in the Skills: line:
// the shared .agents/skills (intent 0008 S3).
const opencodeWhere = ".agents/skills/<name>/SKILL.md"

func runOpencode(t *testing.T, proj fs.FS, version string) Report {
	t.Helper()
	rep, err := NewUseCase(proj, configFiles{projectFile: readTestdata(t, "gabo.yaml")}, goodUser(),
		Settings{Version: version, Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
	if err != nil {
		t.Fatal(err)
	}
	return rep
}

func noLineWith(t *testing.T, r Report, part string) {
	t.Helper()
	for _, l := range r.Lines {
		if strings.Contains(l, part) {
			t.Errorf("unexpected line %q in:\n%s", l, joined(r))
		}
	}
}

// S1, S3, S11: the built OpenCode project with every role on opencode
// passes, with no runtime-value, agent, link or reviewer link failure.
func TestOpencodeBuiltSetPassesCheck(t *testing.T) {
	t.Parallel()
	proj := opencodeBuilt(t, "1.2.3")
	rep := runOpencode(t, proj, "1.2.3")
	wantPassed(t, rep, PassedLine)
	noLineWith(t, rep, "runtimes.")
	noLineWith(t, rep, "reviewer link")
	noLineWith(t, rep, "skill link")
	if _, err := fs.Lstat(proj, ".agents/skills/gabo-reviewer"); err == nil {
		t.Fatal(".agents/skills/gabo-reviewer exists in the project")
	}
}

// The body comment of S3 is the agent's version.
func TestVersionLineReadsTheOpencodeComment(t *testing.T) {
	t.Parallel()
	doc := "---\ndescription: \"x\"\nmode: subagent\n---\n<!-- gabo-version: 1.2.3 -->\nSkills: a\n"
	if v := versionLine([]byte(doc)); v != "1.2.3" {
		t.Fatalf("versionLine = %q, want 1.2.3", v)
	}
}

// S11: one agent built with an older version fails naming it.
func TestOpencodeAgentFromAnotherVersionFails(t *testing.T) {
	t.Parallel()
	proj := opencodeBuilt(t, "1.2.4")
	coder := string(proj[".opencode/agents/gabo-coder.md"].Data)
	proj[".opencode/agents/gabo-coder.md"] = file(strings.Replace(coder, "<!-- gabo-version: 1.2.4 -->", "<!-- gabo-version: 1.2.3 -->", 1))
	rep := runOpencode(t, proj, "1.2.4")
	wantFailed(t, rep,
		"✗ agent gabo-coder built with 1.2.3, catalog is 1.2.4: ask your agent to rebuild",
		"why: .opencode/agents/gabo-coder.md")
	if n := strings.Count(joined(rep), "✗ agent "); n != 1 {
		t.Fatalf("%d agent failures, want 1:\n%s", n, joined(rep))
	}
}

// S11: a missing link in the shared .agents/skills fails naming it
// (intent 0008 S3).
func TestOpencodeMissingLinkFails(t *testing.T) {
	t.Parallel()
	proj := opencodeBuilt(t, "1.2.3")
	delete(proj, ".agents/skills/gabo-wave-report")
	delete(proj, ".agents/skills/gabo-agent-host")
	rep := runOpencode(t, proj, "1.2.3")
	wantFailed(t, rep,
		"✗ skill link .agents/skills/gabo-wave-report missing",
		"✗ skill link .agents/skills/gabo-agent-host missing")
}

// Intent 0008 S4: a link gabo-runtime-opencode left in .opencode/skills
// fails with the leftover line; a user's own entry there is not reported.
func TestOpencodeBuiltLeftoverLinkFails(t *testing.T) {
	t.Parallel()
	proj := opencodeBuilt(t, "1.2.3")
	proj[".opencode/skills/my-skill"] = symlink("../../.gabo/skills/gabo-pm")
	rep := runOpencode(t, proj, "1.2.3")
	wantPassed(t, rep, PassedLine)
	noLineWith(t, rep, ".opencode/skills/my-skill")
	proj[".opencode/skills/gabo-pm"] = symlink("../../.gabo/skills/gabo-pm")
	wantFailed(t, runOpencode(t, proj, "1.2.3"),
		"✗ .opencode/skills/gabo-pm: gabo link left after the move to .agents/skills", NextRebuild)
}

// codexPiOpencodeConfig declares codex, pi and opencode: all three read the
// shared .agents/skills.
var codexPiOpencodeConfig = strings.Replace(strings.Replace(codexConfig,
	"  reviewer: codex\n", "  reviewer: pi\n", 1),
	"  assistant: codex\n", "  assistant: opencode\n", 1)

// Intent 0008 S5: with runtimes naming codex, pi and opencode, one set of
// links in .agents/skills serves all three and gabo check passes; a missing
// shared link is reported once.
func TestCodexPiAndOpencodeShareOneSetOfLinks(t *testing.T) {
	t.Parallel()
	if !strings.Contains(codexPiOpencodeConfig, "reviewer: pi") || !strings.Contains(codexPiOpencodeConfig, "assistant: opencode") {
		t.Fatalf("codexConfig lacks the runtime lines to replace:\n%s", codexConfig)
	}
	proj := codexBuilt(t, "1.2.3")
	for _, built := range []fstest.MapFS{piBuilt(t, "1.2.3"), opencodeBuilt(t, "1.2.3")} {
		for name, f := range built {
			if prev, ok := proj[name]; ok && strings.HasPrefix(name, ".agents/skills/") && string(prev.Data) != string(f.Data) {
				t.Fatalf("%s: links %q and %q", name, prev.Data, f.Data)
			}
			proj[name] = f
		}
	}
	delete(proj, "AGENTS.md") // the Codex and Pi entry; the assistant is opencode
	run := func(p fstest.MapFS) Report {
		rep, err := NewUseCase(p, configFiles{projectFile: codexPiOpencodeConfig}, codexUser(),
			Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	rep := run(proj)
	wantPassed(t, rep, PassedLine)
	noLineWith(t, rep, "skill link")
	delete(proj, ".agents/skills/gabo-pm")
	rep = run(proj)
	wantFailed(t, rep, "✗ skill link .agents/skills/gabo-pm missing")
	if n := strings.Count(joined(rep), "✗ skill link .agents/skills/gabo-pm missing"); n != 1 {
		t.Errorf("the shared link is reported %d times, want once:\n%s", n, joined(rep))
	}
}

// depthLines returns the ✗ lines naming experimental.subagent_depth.
func depthLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		if strings.HasPrefix(l, "✗ ") && strings.Contains(l, "experimental.subagent_depth") {
			out = append(out, l)
		}
	}
	return out
}

// Intent 0015 S8, D6: gabo no longer depends on an engine's nesting depth,
// so gabo check prints no line naming experimental.subagent_depth whatever
// the project's OpenCode config holds (missing, below 3, not a number, no
// file at all, or 3), and the rest of the output is the built project's.
func TestOpencodeSubagentDepthIsNotChecked(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name  string
		files map[string]string
	}{
		{"no project config", nil},
		{"key missing", map[string]string{"opencode.json": `{"theme": "x"}`}},
		{"below 3", map[string]string{".opencode/opencode.jsonc": `{"experimental": {"subagent_depth": 2}}`}},
		{"not a number", map[string]string{"opencode.json": `{"experimental": {"subagent_depth": "3"}}`}},
		{"3 as before", map[string]string{"opencode.json": `{"experimental": {"subagent_depth": 3}}`}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			proj := opencodeBuilt(t, "1.2.3")
			delete(proj, ".opencode/opencode.jsonc")
			for k, v := range c.files {
				proj[k] = file(withJSONKey(v, "default_agent", assistantID))
			}
			rep := runOpencode(t, proj, "1.2.3")
			noLineWith(t, rep, "subagent_depth")
			if len(c.files) > 0 {
				wantPassed(t, rep, PassedLine)
			}
		})
	}
}

// The JSONC reader keeps comment markers inside strings and rejects what is
// not JSONC.
func TestParseJSONC(t *testing.T) {
	t.Parallel()
	v, err := parseJSONC([]byte(readTestdata(t, "opencode.jsonc")))
	if err != nil {
		t.Fatal(err)
	}
	if v["theme"] != "a // not a comment /* nor this */" {
		t.Fatalf("theme = %q", v["theme"])
	}
	for _, doc := range []string{`{"a": 1`, `{"a": /* open`, `[1]`, `{"a": "x\"}`} {
		if _, err := parseJSONC([]byte(doc)); err == nil {
			t.Errorf("parseJSONC(%q) passed", doc)
		}
	}
	if keys := slices.Sorted(maps.Keys(v)); !slices.Equal(keys, []string{"$schema", "experimental", "theme"}) {
		t.Fatalf("keys %v", keys)
	}
}

// Intent 0008 S1 on the agents gabo-runtime-opencode builds: the Skills:
// line holds exactly the spec skills plus the template's fixed-name links,
// each once; gabo-dev-lead keeps gabo-plan-reviewer throughout.
func TestOpencodeBuiltAgentSkills(t *testing.T) {
	t.Parallel()
	for _, tt := range agentSkillsCases("opencode") {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			proj := opencodeBuilt(t, "1.2.3")
			setSkillsLine(t, proj, ".opencode/agents/gabo-pm.md", opencodeWhere, tt.list)
			wantSkillLines(t, runOpencode(t, proj, "1.2.3"), ".opencode/agents/gabo-pm.md", tt.want)
		})
	}
}

// The agent with an empty spec list and no Skills: line passes.
func TestOpencodeBuiltReviewerHasNoSkillsLine(t *testing.T) {
	t.Parallel()
	proj := opencodeBuilt(t, "1.2.3")
	if strings.Contains(string(proj[".opencode/agents/gabo-reviewer.md"].Data), "\nSkills: ") {
		t.Fatalf("gabo-reviewer has a Skills: line:\n%s", proj[".opencode/agents/gabo-reviewer.md"].Data)
	}
	wantSkillLines(t, runOpencode(t, proj, "1.2.3"), ".opencode/agents/gabo-reviewer.md", nil)
}

// The directory the Skills: line names does not change the result.
func TestOpencodeBuiltAgentSkillsIgnoreTheDirectory(t *testing.T) {
	t.Parallel()
	missing := slices.DeleteFunc(slices.Clone(pmSkills), func(s string) bool { return s == "gabo-plan-reviewer" })
	for _, where := range []string{".opencode/skills/<name>/SKILL.md", "`.pi/skills/<name>/SKILL.md`", ".agents/skills/<name>/SKILL.md"} {
		t.Run(where, func(t *testing.T) {
			t.Parallel()
			proj := opencodeBuilt(t, "1.2.3")
			setSkillsLine(t, proj, ".opencode/agents/gabo-pm.md", where, pmSkills)
			wantSkillLines(t, runOpencode(t, proj, "1.2.3"), ".opencode/agents/gabo-pm.md", nil)
			setSkillsLine(t, proj, ".opencode/agents/gabo-pm.md", where, missing)
			wantSkillLines(t, runOpencode(t, proj, "1.2.3"), ".opencode/agents/gabo-pm.md",
				[]string{"✗ agent gabo-pm (opencode): skill gabo-plan-reviewer missing"})
		})
	}
}

// Intent 0013 S5: a fixed reviewer engine beside listed assistants is in
// the set: reviewer opencode with no OpenCode agents fails with the agents
// line for opencode; session is no engine.
func TestOpencodeReviewerBesideListedAssistantsNeedsItsAgents(t *testing.T) {
	t.Parallel()
	p := with(project(), map[string]string{
		".gabo/skills/gabo-runtime-pi/SKILL.md":       strings.Replace(claudeSkill, "gabo-runtime-claude", "gabo-runtime-pi", 1),
		".gabo/skills/gabo-runtime-opencode/SKILL.md": strings.NewReplacer("gabo-runtime-claude", "gabo-runtime-opencode", ".claude/agents", ".opencode/agents").Replace(claudeSkill),
		".pi/settings.json":                           piKeysJSON,
	})
	cfg := withRuntimes(t, validConfig, "  coder: session\n  reviewer: opencode\n  assistant: [claude, pi]\n")
	r := run(t, p, configFiles{projectFile: cfg}, Settings{})
	wantFailed(t, r, "✗ agent alpha missing", "why: no file named alpha in .opencode/agents (runtime opencode)")
	noLineWith(t, r, "runtime session")
}
