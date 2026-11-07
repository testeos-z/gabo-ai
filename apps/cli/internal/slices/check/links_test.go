package check

import (
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// linkedSkill is a runtime skill that declares where its project skill links
// and its user-level reviewer link live, as gabo-runtime-claude does.
const linkedSkill = "---\nname: gabo-runtime-claude\nmetadata:\n  agent_dir: .claude/agents\n  skill_dir: .claude/skills\n  agent_skills: frontmatter:skills\n  reviewer_link: ${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer\n---\nbody\n"

// linkedSpec lists catalog skills: sk-b twice, so it is checked once.
const linkedSpec = `agents:
  - id: alpha
    role: first
    skills: [sk-a, sk-b]
    tools: [read]
    launched_by: main-session
    template: agents/alpha.md
  - id: beta
    role: second
    skills: [sk-b]
    tools: [read]
    launched_by: alpha
    template: agents/beta.md
`

// linkedConfig adds a skill-folder slot adapter, a bare methodology skill
// and an adapter that is not a skill folder (needs no link).
const linkedConfig = validConfig + `slots:
  tasks:
    adapter: .claude/skills/gabo-tasks-github/SKILL.md
  methodology:
    adapter: sk-method
  reviewer_placement:
    adapter: .claude/agents/gabo-reviewer.md
`

const (
	home        = "/home/o"
	reviewerDir = "/home/o/.local/share/gabo/skills/gabo-reviewer"
	userLink    = "/home/o/.claude/skills/gabo-reviewer"
)

func symlink(target string) *fstest.MapFile {
	return &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte(target)}
}

// linkedProject passes: every spec skill and slot link exists and resolves.
func linkedProject() fstest.MapFS {
	p := with(project(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md":              linkedSkill,
		".gabo/skills/gabo-agent-templates/agents.yaml":          linkedSpec,
		".gabo/skills/sk-a/SKILL.md":                             "# A\n",
		".gabo/skills/sk-b/SKILL.md":                             "# B\n",
		".gabo/skills/sk-method/SKILL.md":                        "# M\n",
		".gabo/skills/gabo-contract-tasks/SKILL.md":              "## Operations\n",
		".gabo/skills/gabo-contract-methodology/SKILL.md":        "## Operations\n",
		".gabo/skills/gabo-contract-reviewer-placement/SKILL.md": "## Operations\n",
		".claude/skills/gabo-tasks-github/SKILL.md":              "# T\n",
		".claude/agents/gabo-reviewer.md":                        "---\nname: gabo-reviewer\ngabo-version: 1.2.3\n---\n",
		".claude/agents/alpha.md":                                "---\nname: alpha\nskills:\n  - sk-a\n  - sk-b\ngabo-version: 1.2.3\n---\nbody\n",
		".claude/agents/beta.md":                                 "---\nname: beta\nskills: [sk-b]\ngabo-version: 1.2.3\n---\nbody\n",
	})
	p[".claude/skills/sk-a"] = symlink("../../.gabo/skills/sk-a")
	p[".claude/skills/sk-b"] = symlink("../../.gabo/skills/sk-b")
	p[".claude/skills/gabo-tasks"] = symlink("gabo-tasks-github")
	p[".claude/skills/gabo-methodology"] = symlink("../../.gabo/skills/sk-method")
	return p
}

// userFS is a hand-written in-memory UserFS: links map a path to its raw
// target, dirs and files are the real entries.
type userFS struct {
	links map[string]string
	dirs  map[string]bool
	files map[string]bool
}

func goodUser() userFS {
	return userFS{
		links: map[string]string{userLink: reviewerDir},
		dirs:  map[string]bool{reviewerDir: true, reviewerDir + "/scripts": true},
		files: map[string]bool{reviewerDir + "/scripts/reviewer.sh": true},
	}
}

func (u userFS) clone() userFS {
	return userFS{links: maps.Clone(u.links), dirs: maps.Clone(u.dirs), files: maps.Clone(u.files)}
}

func notExist(name string) error {
	return &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
}

// resolve follows every link on the way to name.
func (u userFS) resolve(name string) string {
	for range 8 {
		moved := false
		for l, target := range u.links {
			if name == l || strings.HasPrefix(name, l+"/") {
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(l), target)
				}
				name, moved = target+name[len(l):], true
				break
			}
		}
		if !moved {
			return name
		}
	}
	return name
}

func (u userFS) mode(name string) (fs.FileMode, error) {
	switch {
	case u.dirs[name]:
		return fs.ModeDir | 0o755, nil
	case u.files[name]:
		return 0o644, nil
	}
	return 0, notExist(name)
}

func (u userFS) Lstat(name string) (fs.FileMode, error) {
	if _, ok := u.links[name]; ok {
		return fs.ModeSymlink | 0o777, nil
	}
	return u.mode(u.resolve(name))
}

func (u userFS) Stat(name string) (fs.FileMode, error) { return u.mode(u.resolve(name)) }

func (u userFS) ReadLink(name string) (string, error) {
	if t, ok := u.links[name]; ok {
		return t, nil
	}
	return "", &fs.PathError{Op: "readlink", Path: name, Err: errors.New("invalid argument")}
}

func env(kv ...string) func(string) string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return func(k string) string { return m[k] }
}

func linkedSettings() Settings {
	return Settings{Getenv: env("HOME", home), ReviewerDir: reviewerDir}
}

func runLinked(t *testing.T, proj fs.FS, user UserFS, s Settings) Report {
	t.Helper()
	if s.Version == "" {
		s.Version = "1.2.3"
	}
	s.Layers = layers()
	rep, err := NewUseCase(proj, configFiles{projectFile: linkedConfig}, user, s).Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return rep
}

func TestLinkedProjectPasses(t *testing.T) {
	t.Parallel()
	r := runLinked(t, linkedProject(), goodUser(), linkedSettings())
	wantPassed(t, r, PassedLine)
}

func TestMissingCatalogSkillLinkFails(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	delete(p, ".claude/skills/sk-b")
	r := runLinked(t, p, goodUser(), linkedSettings())
	wantFailed(t, r,
		"✗ skill link .claude/skills/sk-b missing",
		"why: agent alpha lists sk-b; link it to ../../.gabo/skills/sk-b (runtime claude)",
		NextLink)
	if n := strings.Count(joined(r), "skill link .claude/skills/sk-b missing"); n != 1 {
		t.Fatalf("reported %d times, want once:\n%s", n, joined(r))
	}
}

// Item 4 of #135: a link to the CLI source is not reached by gabo init.
func TestSkillLinkToAnotherFolderFails(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p["apps/cli/internal/skills/sk-a/SKILL.md"] = file("# A\n")
	p[".claude/skills/sk-a"] = symlink("../../apps/cli/internal/skills/sk-a")
	r := runLinked(t, p, goodUser(), linkedSettings())
	wantFailed(t, r,
		"✗ skill link .claude/skills/sk-a points to ../../apps/cli/internal/skills/sk-a, not ../../.gabo/skills/sk-a",
		NextLink)
}

func TestSkillLinkAsRealFolderFails(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	delete(p, ".claude/skills/sk-a")
	p[".claude/skills/sk-a/SKILL.md"] = file("# copy\n")
	r := runLinked(t, p, goodUser(), linkedSettings())
	wantFailed(t, r, "✗ skill link .claude/skills/sk-a is not a link to ../../.gabo/skills/sk-a", NextLink)
}

func TestMissingSlotLinksFail(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	delete(p, ".claude/skills/gabo-tasks")
	delete(p, ".claude/skills/gabo-methodology")
	r := runLinked(t, p, goodUser(), linkedSettings())
	wantFailed(t, r,
		"✗ skill link .claude/skills/gabo-tasks missing",
		"why: slots.tasks.adapter is .claude/skills/gabo-tasks-github/SKILL.md; link it to gabo-tasks-github (runtime claude)",
		"✗ skill link .claude/skills/gabo-methodology missing",
		"why: slots.methodology.adapter is sk-method; link it to ../../.gabo/skills/sk-method (runtime claude)")
	if strings.Contains(joined(r), "gabo-reviewer-placement") {
		t.Fatalf("an adapter that is not a skill folder needs no link:\n%s", joined(r))
	}
}

func TestBrokenSlotLinkFails(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p[".claude/skills/gabo-tasks"] = symlink("gabo-tasks-gitlab")
	r := runLinked(t, p, goodUser(), linkedSettings())
	wantFailed(t, r, "✗ skill link .claude/skills/gabo-tasks points to gabo-tasks-gitlab, not gabo-tasks-github", NextLink)
}

// An adapter folder that already has the fixed name needs no link.
func TestSlotAdapterUnderItsFixedNameNeedsNoLink(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	delete(p, ".claude/skills/gabo-tasks")
	delete(p, ".claude/skills/gabo-tasks-github/SKILL.md")
	p[".claude/skills/gabo-tasks/SKILL.md"] = file("# T\n")
	cfg := strings.Replace(linkedConfig, "gabo-tasks-github/SKILL.md", "gabo-tasks/SKILL.md", 1)
	rep, err := NewUseCase(p, configFiles{projectFile: cfg}, goodUser(), Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
	if err != nil {
		t.Fatal(err)
	}
	wantPassed(t, rep, PassedLine)
}

func TestSkillDirOutsideTheRepositoryFails(t *testing.T) {
	t.Parallel()
	p := with(linkedProject(), map[string]string{
		".gabo/skills/gabo-runtime-claude/SKILL.md": strings.Replace(linkedSkill, "skill_dir: .claude/skills", "skill_dir: ../skills", 1),
	})
	r := runLinked(t, p, goodUser(), linkedSettings())
	wantFailed(t, r, "✗ runtime claude: skill_dir must stay inside the repository")
}

func TestReviewerLinkMissingFails(t *testing.T) {
	t.Parallel()
	u := goodUser().clone()
	delete(u.links, userLink)
	r := runLinked(t, linkedProject(), u, linkedSettings())
	wantFailed(t, r,
		"✗ reviewer link "+userLink+" missing",
		"why: the gabo-reviewer agent runs "+userLink+"/scripts/reviewer.sh; link it to "+reviewerDir+" (runtime claude)",
		NextLink)
}

func TestReviewerLinkToAnotherFolderFails(t *testing.T) {
	t.Parallel()
	u := goodUser().clone()
	u.links[userLink] = "/home/o/.claude/skills/gbai-reviewer"
	u.dirs["/home/o/.claude/skills/gbai-reviewer"] = true
	r := runLinked(t, linkedProject(), u, linkedSettings())
	wantFailed(t, r, "✗ reviewer link "+userLink+" points to /home/o/.claude/skills/gbai-reviewer, not "+reviewerDir, NextLink)
}

func TestReviewerLinkAsRealFolderFails(t *testing.T) {
	t.Parallel()
	u := goodUser().clone()
	delete(u.links, userLink)
	u.dirs[userLink] = true
	r := runLinked(t, linkedProject(), u, linkedSettings())
	wantFailed(t, r, "✗ reviewer link "+userLink+" is not a link to "+reviewerDir, NextLink)
}

func TestBrokenReviewerLinkFails(t *testing.T) {
	t.Parallel()
	u := goodUser().clone()
	delete(u.files, reviewerDir+"/scripts/reviewer.sh")
	r := runLinked(t, linkedProject(), u, linkedSettings())
	wantFailed(t, r,
		"✗ reviewer link "+userLink+" is broken: "+reviewerDir+"/scripts/reviewer.sh missing",
		"next: run gabo init to install the reviewer skill, then rerun gabo check")
}

// A relative link target resolves from the link's directory.
func TestRelativeReviewerLinkPasses(t *testing.T) {
	t.Parallel()
	u := goodUser().clone()
	u.links[userLink] = "../../.local/share/gabo/skills/gabo-reviewer"
	r := runLinked(t, linkedProject(), u, linkedSettings())
	wantPassed(t, r, PassedLine)
}

func TestReviewerLinkFollowsClaudeConfigDir(t *testing.T) {
	t.Parallel()
	u := goodUser().clone()
	delete(u.links, userLink)
	u.links["/cfg/skills/gabo-reviewer"] = reviewerDir
	s := linkedSettings()
	s.Getenv = env("HOME", home, "CLAUDE_CONFIG_DIR", "/cfg")
	wantPassed(t, runLinked(t, linkedProject(), u, s), PassedLine)
	s.Getenv = env("HOME", home)
	wantFailed(t, runLinked(t, linkedProject(), u, s), "✗ reviewer link "+userLink+" missing")
}

// The reviewer scripts and CI run gabo check with their own HOME, where the
// owner's user-level link never exists: there it is not checked.
func TestReviewerLinkNotCheckedInCI(t *testing.T) {
	t.Parallel()
	s := linkedSettings()
	s.CI = true
	r := runLinked(t, linkedProject(), userFS{}, s)
	wantPassed(t, r, "! reviewer link not checked in CI: it is user-level", PassedLine)
}

func TestReviewerLinkWithoutAHomeFails(t *testing.T) {
	t.Parallel()
	s := linkedSettings()
	s.Getenv = env()
	r := runLinked(t, linkedProject(), goodUser(), s)
	wantFailed(t, r, "✗ reviewer link ${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gabo-reviewer is not an absolute path here",
		"next: set HOME to an absolute path, then rerun gabo check")
	s = linkedSettings()
	s.ReviewerDir = ""
	r = runLinked(t, linkedProject(), goodUser(), s)
	wantFailed(t, r, "✗ reviewer skill dir unknown: set XDG_DATA_HOME or HOME to an absolute path")
}

// Only the reviewer runtime's skill decides the reviewer link: a pi coder
// runtime declaring another reviewer link is not checked for it.
func TestReviewerLinkOnlyForTheReviewerRuntime(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p[".gabo/skills/gabo-runtime-pi/SKILL.md"] = file("---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .claude/agents\n  agent_skills: frontmatter:skills\n  reviewer_link: $HOME/.pi/skills/gabo-reviewer\n---\n")
	p[".pi/settings.json"] = file(piKeysJSON)
	cfg := strings.Replace(linkedConfig, "  coder: claude", "  coder: pi", 1)
	rep, err := NewUseCase(p, configFiles{projectFile: cfg}, goodUser(), Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
	if err != nil {
		t.Fatal(err)
	}
	wantPassed(t, rep, PassedLine)
}

// A pi reviewer runtime decides the reviewer link from its own skill: one
// that declares none needs no link and the project passes gabo check (T073),
// with or without review.pi_credential_env.
func TestPiReviewerRuntimePassesCheck(t *testing.T) {
	t.Parallel()
	p := linkedProject()
	p[".gabo/skills/gabo-runtime-pi/SKILL.md"] = file("---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .claude/agents\n  agent_skills: \"line:Skills:\"\n---\n")
	p[".pi/settings.json"] = file(piKeysJSON)
	cfg := strings.Replace(linkedConfig, "  reviewer: claude", "  reviewer: pi", 1)
	u := goodUser().clone()
	delete(u.links, userLink)
	for name, c := range map[string]string{
		"without env": cfg,
		"with env":    strings.Replace(cfg, "  human_only_paths: [.github/]\n", "  human_only_paths: [.github/]\n  pi_credential_env: PI_TOKEN\n", 1),
	} {
		rep, err := NewUseCase(p, configFiles{projectFile: c}, u, Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
		if err != nil {
			t.Fatal(err)
		}
		if rep.Failed {
			t.Errorf("%s: gabo check failed: %v", name, rep.Lines)
		}
		wantPassed(t, rep, PassedLine)
	}
}

func TestExpand(t *testing.T) {
	t.Parallel()
	get := env("HOME", "/h", "CFG", "/c")
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"${CFG:-$HOME/.claude}/skills", "/c/skills", true},
		{"${UNSET:-$HOME/.claude}/skills", "/h/.claude/skills", true},
		{"${UNSET:-$NOPE/.claude}", "/.claude", false},
		{"$HOME/x", "/h/x", true},
		{"${HOME}/x", "/h/x", true},
		{"$NOPE/x", "/x", false},
		{"/plain/$", "/plain/$", true},
	}
	for _, c := range cases {
		got, ok := expand(c.in, get)
		if got != c.want || ok != c.ok {
			t.Errorf("expand(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

// singleRuntimes is the runtimes block of validConfig and linkedConfig.
const singleRuntimes = "  coder: claude\n  reviewer: claude\n  assistant: claude\n"

// sessionRuntimes runs the coder and the reviewer in the main session and
// lists two assistant engines (intent 0013 S1, S2).
const sessionRuntimes = "  coder: session\n  reviewer: session\n  assistant: [claude, pi]\n"

// withRuntimes replaces the runtimes block of cfg.
func withRuntimes(t *testing.T, cfg, runtimes string) string {
	t.Helper()
	if !strings.Contains(cfg, singleRuntimes) {
		t.Fatalf("config has no %q block", singleRuntimes)
	}
	return strings.Replace(cfg, singleRuntimes, runtimes, 1)
}

// piLinkedSkill is a Pi runtime skill sharing the Claude agent and skill
// dirs, with its own user-level reviewer link.
const piLinkedSkill = "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .claude/agents\n  skill_dir: .claude/skills\n  agent_skills: frontmatter:skills\n  reviewer_link: $HOME/.pi/skills/gabo-reviewer\n---\n"

const piUserLink = "/home/o/.pi/skills/gabo-reviewer"

// listedProject is linkedProject with the Pi runtime skill and Pi keys, for
// sessionRuntimes.
func listedProject() fstest.MapFS {
	return with(linkedProject(), map[string]string{
		".gabo/skills/gabo-runtime-pi/SKILL.md": piLinkedSkill,
		".pi/settings.json":                     piKeysJSON,
	})
}

// Intent 0013 S5: with runtimes.reviewer session the reviewer link of every
// listed assistant engine is checked, each once, and session is no engine.
func TestReviewerLinkForEveryListedAssistantUnderSession(t *testing.T) {
	t.Parallel()
	cfg := withRuntimes(t, linkedConfig, sessionRuntimes)
	s := Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}
	rep, err := NewUseCase(listedProject(), configFiles{projectFile: cfg}, goodUser(), s).Run()
	if err != nil {
		t.Fatal(err)
	}
	wantFailed(t, rep, "✗ reviewer link "+piUserLink+" missing",
		"why: the gabo-reviewer agent runs "+piUserLink+"/scripts/reviewer.sh; link it to "+reviewerDir+" (runtime pi)")
	noLineWith(t, rep, "reviewer link "+userLink)
	noLineWith(t, rep, "runtime session")

	u := goodUser().clone()
	u.links[piUserLink] = reviewerDir
	rep, err = NewUseCase(listedProject(), configFiles{projectFile: cfg}, u, s).Run()
	if err != nil {
		t.Fatal(err)
	}
	wantPassed(t, rep, PassedLine)
}

// Intent 0013 S5: the skill links are required in the skill dir of every
// listed assistant engine.
func TestSkillLinksForEveryListedAssistant(t *testing.T) {
	t.Parallel()
	p := with(listedProject(), map[string]string{".gabo/skills/gabo-runtime-pi/SKILL.md": otherSkill})
	cfg := withRuntimes(t, linkedConfig, sessionRuntimes)
	rep, err := NewUseCase(p, configFiles{projectFile: cfg}, goodUser(), Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
	if err != nil {
		t.Fatal(err)
	}
	wantFailed(t, rep, "✗ skill link .other/skills/sk-a missing", "why: agent alpha lists sk-a; link it to ../../.gabo/skills/sk-a (runtime pi)")
	noLineWith(t, rep, "runtime session")
}

// Intent 0013 D11: parseRuntimeMeta reads metadata.agent_tools (an agent
// place, intent 0008 D1 forms) and metadata.agent_tool_requirements (each
// capability to its tool names, in declaration order).
func TestParseRuntimeMetaReadsToolDeclaration(t *testing.T) {
	t.Parallel()
	m, err := parseRuntimeMeta(runtimeDoc("  agent_tools: frontmatter:tools", "  agent_tool_requirements:",
		"    memory: [mem_search, mem_save]", "    search: [grep, find]", "    web: []"))
	if err != nil {
		t.Fatalf("parseRuntimeMeta() error = %v", err)
	}
	if m.AgentTools != (agentPlace{"frontmatter", "tools"}) || !m.declaresTools() || m.ToolsErr != nil {
		t.Fatalf("agent_tools = %+v, declared %v, err %v", m.AgentTools, m.declaresTools(), m.ToolsErr)
	}
	want := []toolRequirement{{"memory", []string{"mem_search", "mem_save"}}, {"search", []string{"grep", "find"}}, {"web", nil}}
	if len(m.ToolRequirements) != len(want) {
		t.Fatalf("requirements = %+v, want %+v", m.ToolRequirements, want)
	}
	for i, w := range want {
		g := m.ToolRequirements[i]
		if g.capability != w.capability || strings.Join(g.tools, ",") != strings.Join(w.tools, ",") {
			t.Fatalf("requirement %d = %+v, want %+v", i, g, w)
		}
	}
	m, err = parseRuntimeMeta(runtimeDoc("  agent_skills: frontmatter:skills"))
	if err != nil || m.declaresTools() || m.ToolsErr != nil {
		t.Fatalf("no declaration: declared %v, err %v, %v", m.declaresTools(), err, m.ToolsErr)
	}
}

// A malformed tool declaration does not fail the whole frontmatter: the
// other metadata stays readable and ToolsErr names the key.
func TestParseRuntimeMetaKeepsAMalformedToolDeclaration(t *testing.T) {
	t.Parallel()
	reqs := []string{"  agent_tool_requirements:", "    memory: [mem_save]"}
	for name, meta := range map[string][]string{
		"unknown form":              append([]string{"  agent_tools: json:tools"}, reqs...),
		"agent_tools is a list":     append([]string{"  agent_tools: [tools]"}, reqs...),
		"requirements not a map":    {"  agent_tools: frontmatter:tools", "  agent_tool_requirements: memory"},
		"requirement not a list":    {"  agent_tools: frontmatter:tools", "  agent_tool_requirements:", "    memory: mem_save"},
		"tool not a name":           {"  agent_tools: frontmatter:tools", "  agent_tool_requirements:", "    memory: [{a: b}]"},
		"only agent_tools":          {"  agent_tools: frontmatter:tools"},
		"only tool requirements":    reqs,
		"empty tool name in a list": {"  agent_tools: frontmatter:tools", "  agent_tool_requirements:", `    memory: [""]`},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m, err := parseRuntimeMeta(runtimeDoc(append([]string{"  agent_skills: frontmatter:skills"}, meta...)...))
			if err != nil {
				t.Fatalf("parseRuntimeMeta() error = %v, want the rest of the metadata read", err)
			}
			if m.ToolsErr == nil || !strings.Contains(m.ToolsErr.Error(), "metadata.agent_tool") {
				t.Fatalf("ToolsErr = %v, want an error naming the key", m.ToolsErr)
			}
			if m.declaresTools() || m.AgentDir != ".x/agents" || !m.AgentSkills.declared() {
				t.Fatalf("meta = %+v", m)
			}
		})
	}
}

// Intent 0013 S20 (plan T22): parseRuntimeMeta reads
// metadata.agent_tools_unavailable, the tools a subagent never has, in
// declaration order; absent, it declares none.
func TestParseRuntimeMetaReadsUnavailableTools(t *testing.T) {
	t.Parallel()
	decl := []string{"  agent_tools: frontmatter:tools", "  agent_tool_requirements:", "    memory: [mem_save]"}
	m, err := parseRuntimeMeta(runtimeDoc(append(slices.Clone(decl), "  agent_tools_unavailable: [subagent_*, task]")...))
	if err != nil || m.ToolsErr != nil {
		t.Fatalf("parseRuntimeMeta() error = %v, %v", err, m.ToolsErr)
	}
	if strings.Join(m.UnavailableTools, ",") != "subagent_*,task" || !m.declaresTools() {
		t.Fatalf("agent_tools_unavailable = %q, declared %v", m.UnavailableTools, m.declaresTools())
	}
	m, err = parseRuntimeMeta(runtimeDoc(decl...))
	if err != nil || m.ToolsErr != nil || m.UnavailableTools != nil {
		t.Fatalf("no declaration: %q, err %v, %v", m.UnavailableTools, err, m.ToolsErr)
	}
	for name, meta := range map[string][]string{
		"not a list":          append(slices.Clone(decl), "  agent_tools_unavailable: subagent_*"),
		"empty name":          append(slices.Clone(decl), `  agent_tools_unavailable: [""]`),
		"name not a scalar":   append(slices.Clone(decl), "  agent_tools_unavailable: [{a: b}]"),
		"without agent_tools": {"  agent_tools_unavailable: [subagent_*]"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			m, err := parseRuntimeMeta(runtimeDoc(append([]string{"  agent_skills: frontmatter:skills"}, meta...)...))
			if err != nil {
				t.Fatalf("parseRuntimeMeta() error = %v, want the rest of the metadata read", err)
			}
			if m.ToolsErr == nil || !strings.Contains(m.ToolsErr.Error(), "metadata.agent_tools_unavailable") {
				t.Fatalf("ToolsErr = %v, want an error naming the key", m.ToolsErr)
			}
			if m.declaresTools() || !m.AgentSkills.declared() {
				t.Fatalf("meta = %+v", m)
			}
		})
	}
}
