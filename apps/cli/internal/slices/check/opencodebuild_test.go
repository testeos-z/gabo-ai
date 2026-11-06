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

// opencodeBuilt is a project with the real embedded catalog, the stand-in
// runtime skill, and every spec agent built in the shape of spec 0004 S3
// (.opencode/agents/<id>.md: frontmatter description, mode and permission;
// the body starts with <!-- gabo-version: x.y.z -->, then the Skills: line
// of build step 7, omitted for an empty list);
// each spec skill linked in the shared .agents/skills (intent 0008 S3) as
// .agents/skills/<name> -> ../../.gabo/skills/<name>, the fixed-name links
// of the tree's skill adapters, and the tree's .opencode/opencode.jsonc.
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
	proj[CatalogDir+"/gabo-runtime-opencode/SKILL.md"] = file(readTestdata(t, "gabo-runtime-opencode/SKILL.md"))
	proj[".opencode/opencode.jsonc"] = file(readTestdata(t, "opencode.jsonc"))
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
// passes, with no runtime-value, agent, link, reviewer link or subagent
// depth failure.
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

// S8, S11: with no project config, or experimental.subagent_depth below 3
// (or missing, or unreadable) in the first existing file of the S8 order,
// gabo check fails with one ✗ line naming the key, the value 3 and the file.
func TestOpencodeSubagentDepthBelowThreeFails(t *testing.T) {
	t.Parallel()
	const (
		root   = "opencode.json"
		rootC  = "opencode.jsonc"
		nested = ".opencode/opencode.json"
		jsonc  = ".opencode/opencode.jsonc"
	)
	for _, c := range []struct {
		name  string
		files map[string]string
		file  string
		why   string
	}{
		{"no project config", nil, nested, "no project config"},
		{"below 3", map[string]string{jsonc: `{"experimental": {"subagent_depth": 2}}`}, jsonc, "it is 2"},
		{"key missing", map[string]string{root: `{"theme": "x"}`}, root, "not set"},
		{"experimental not an object", map[string]string{root: `{"experimental": true}`}, root, "experimental is not an object"},
		{"not a number", map[string]string{root: `{"experimental": {"subagent_depth": "3"}}`}, root, `it is "3"`},
		{"not a whole number", map[string]string{root: `{"experimental": {"subagent_depth": 3.5}}`}, root, "it is 3.5"},
		{"does not parse", map[string]string{rootC: `{"experimental": {"subagent_depth": 3}`}, rootC, "the file does not parse"},
		// Only the first existing file counts, as in OpenCode's lookup order.
		{"first file wins", map[string]string{rootC: `{"experimental": {"subagent_depth": 1}}`, nested: `{"experimental": {"subagent_depth": 3}}`}, rootC, "it is 1"},
		{"root before nested", map[string]string{root: `{}`, jsonc: `{"experimental": {"subagent_depth": 5}}`}, root, "not set"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			proj := opencodeBuilt(t, "1.2.3")
			delete(proj, ".opencode/opencode.jsonc")
			for k, v := range c.files {
				proj[k] = file(v)
			}
			rep := runOpencode(t, proj, "1.2.3")
			want := "✗ experimental.subagent_depth must be 3 or more in " + c.file + ": " + c.why
			got := depthLines(rep)
			if len(got) != 1 || !strings.HasPrefix(got[0], want) {
				t.Fatalf("depth lines %q, want one starting with %q:\n%s", got, want, joined(rep))
			}
			wantFailed(t, rep, "next: set experimental.subagent_depth to 3 in "+c.file+", then rerun gabo check")
		})
	}
}

// S11: 3 or more in the first existing file passes, in any of the four
// files, JSONC comments and trailing commas included.
func TestOpencodeSubagentDepthThreeOrMorePasses(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ file, doc string }{
		{"opencode.json", `{"experimental": {"subagent_depth": 3}}`},
		{"opencode.jsonc", "// c\n{\"experimental\": {\"subagent_depth\": 4,},}\n"},
		{".opencode/opencode.json", "\ufeff{\"experimental\": {\"subagent_depth\": 10}}"},
		{".opencode/opencode.jsonc", readTestdata(t, "opencode.jsonc")},
	} {
		t.Run(c.file, func(t *testing.T) {
			t.Parallel()
			proj := opencodeBuilt(t, "1.2.3")
			delete(proj, ".opencode/opencode.jsonc")
			proj[c.file] = file(c.doc)
			rep := runOpencode(t, proj, "1.2.3")
			wantPassed(t, rep, PassedLine)
			if got := depthLines(rep); len(got) != 0 {
				t.Fatalf("depth lines %q", got)
			}
		})
	}
}

// A project without an opencode runtime never reads an OpenCode config.
func TestSubagentDepthOnlyForOpencode(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{"opencode.json": `{"experimental": {"subagent_depth": 1}}`})
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	if got := depthLines(r); len(got) != 0 {
		t.Fatalf("depth lines %q", got)
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
