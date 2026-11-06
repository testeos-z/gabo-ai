package check

import (
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/agentspec"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// assistantID is the main_session id of the embedded agents.yaml.
const assistantID = "gabo-assistant"

// mainSessionTemplate reads a main_session template of the embedded catalog.
func mainSessionTemplate(t *testing.T, field func(agentspec.MainSessionEntry) string) string {
	t.Helper()
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	if spec.MainSession == nil || spec.MainSession.ID != assistantID {
		t.Fatalf("embedded main_session = %+v, want id %s", spec.MainSession, assistantID)
	}
	b, err := fs.ReadFile(skills.Catalog(), path.Join(agentspec.Dir, field(*spec.MainSession)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// assistantBody is the assistant agent template without its first line.
func assistantBody(t *testing.T) string {
	t.Helper()
	_, rest, _ := strings.Cut(mainSessionTemplate(t, func(m agentspec.MainSessionEntry) string { return m.Template }), "\n")
	return strings.TrimPrefix(rest, "\n")
}

// assistantBlock is the scoped block template, copied verbatim.
func assistantBlock(t *testing.T) string {
	t.Helper()
	return mainSessionTemplate(t, func(m agentspec.MainSessionEntry) string { return m.Block })
}

// withJSONKey adds "key": "value" as the first member of the JSON (or
// JSONC) object doc.
func withJSONKey(doc, key, value string) string {
	return strings.Replace(doc, "{", "{\n  \""+key+"\": \""+value+"\",", 1)
}

// entryLines returns the lines naming the gabo assistant entry.
func entryLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		if strings.Contains(l, "gabo assistant entry") || strings.Contains(l, "main_session") {
			out = append(out, l)
		}
	}
	return out
}

// wantEntryLines fails unless the entry lines are exactly want, in order,
// and the run failed with the rebuild step.
func wantEntryLines(t *testing.T, r Report, want ...string) {
	t.Helper()
	got := entryLines(r)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("entry lines:\n%s\nwant:\n%s\noutput:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"), joined(r))
	}
	wantFailed(t, r, NextRebuild)
}

// builtEngine runs gabo check on a correct build of one engine as
// runtimes.assistant.
type builtEngine struct {
	engine string
	build  func(*testing.T, string) fstest.MapFS
	run    func(*testing.T, fs.FS) Report
}

func builtEngines() []builtEngine {
	return []builtEngine{
		{"claude", claudeBuilt, func(t *testing.T, p fs.FS) Report {
			return run(t, p, configFiles{projectFile: validConfig}, Settings{})
		}},
		{"codex", codexBuilt, func(t *testing.T, p fs.FS) Report { return runCodex(t, p, codexUser(), "1.2.3") }},
		{"opencode", opencodeBuilt, func(t *testing.T, p fs.FS) Report { return runOpencode(t, p, "1.2.3") }},
		{"pi", piBuilt, func(t *testing.T, p fs.FS) Report { return runPi(t, p, Settings{}) }},
	}
}

// S5: a correct build of each of the four engines prints no entry line.
func TestMainSessionCorrectBuildPrintsNoLine(t *testing.T) {
	t.Parallel()
	for _, e := range builtEngines() {
		t.Run(e.engine, func(t *testing.T) {
			t.Parallel()
			r := e.run(t, e.build(t, "1.2.3"))
			if got := entryLines(r); len(got) != 0 {
				t.Fatalf("entry lines %q:\n%s", got, joined(r))
			}
			wantPassed(t, r, PassedLine)
		})
	}
}

// S5: for the engine in runtimes.assistant, each missing piece of its entry
// prints its file, "incomplete" while another piece is there, "missing"
// when none is, "duplicated" for two blocks, and exits 1.
func TestMainSessionEntryPiecesFail(t *testing.T) {
	t.Parallel()
	const (
		claudeLink  = ".claude/skills/gabo-assistant"
		claudeAgent = ".claude/agents/gabo-assistant.md"
		claudeKeys  = ".claude/settings.json"
		sharedLink  = ".agents/skills/gabo-assistant"
		ocAgent     = ".opencode/agents/gabo-assistant.md"
		ocConfig    = ".opencode/opencode.jsonc"
	)
	line := func(file, word, engine string) string {
		return "✗ " + file + ": gabo assistant entry " + word + " for " + engine
	}
	cases := []struct {
		name   string
		engine string
		edit   func(*testing.T, fstest.MapFS)
		want   []string
	}{
		{"claude link", "claude", func(_ *testing.T, p fstest.MapFS) { delete(p, claudeLink) },
			[]string{line(claudeLink, "incomplete", "claude")}},
		{"claude agent file", "claude", func(_ *testing.T, p fstest.MapFS) { delete(p, claudeAgent) },
			[]string{line(claudeAgent, "incomplete", "claude")}},
		{"claude key", "claude", func(_ *testing.T, p fstest.MapFS) { p[claudeKeys] = file(claudeKeysJSON) },
			[]string{line(claudeKeys, "incomplete", "claude")}},
		{"claude user's own agent value", "claude", func(_ *testing.T, p fstest.MapFS) {
			p[claudeKeys] = file(withJSONKey(claudeKeysJSON, "agent", "my-agent"))
		}, []string{line(claudeKeys, "incomplete", "claude")}},
		{"claude nothing", "claude", func(_ *testing.T, p fstest.MapFS) {
			delete(p, claudeLink)
			delete(p, claudeAgent)
			p[claudeKeys] = file(claudeKeysJSON)
		}, []string{line(claudeLink, "missing", "claude"), line(claudeAgent, "missing", "claude"), line(claudeKeys, "missing", "claude")}},
		{"opencode agent file and key", "opencode", func(t *testing.T, p fstest.MapFS) {
			delete(p, ocAgent)
			p[ocConfig] = file(readTestdata(t, "opencode.jsonc"))
		}, []string{line(ocAgent, "incomplete", "opencode"), line(ocConfig, "incomplete", "opencode")}},
		{"opencode link", "opencode", func(_ *testing.T, p fstest.MapFS) { delete(p, sharedLink) },
			[]string{line(sharedLink, "incomplete", "opencode")}},
		{"codex block", "codex", func(_ *testing.T, p fstest.MapFS) { p["AGENTS.md"] = file("# Project\n") },
			[]string{line("AGENTS.md", "incomplete", "codex")}},
		{"codex nothing", "codex", func(_ *testing.T, p fstest.MapFS) {
			delete(p, sharedLink)
			delete(p, "AGENTS.md")
		}, []string{line(sharedLink, "missing", "codex"), line("AGENTS.override.md or AGENTS.md", "missing", "codex")}},
		// The block must be in the first existing file of the order.
		{"codex block below an override", "codex", func(_ *testing.T, p fstest.MapFS) { p["AGENTS.override.md"] = file("# Local\n") },
			[]string{line("AGENTS.override.md", "incomplete", "codex")}},
		{"codex half a block", "codex", func(t *testing.T, p fstest.MapFS) {
			p["AGENTS.md"] = file(strings.Replace(assistantBlock(t), assistantEnd, "", 1))
		}, []string{line("AGENTS.md", "incomplete", "codex")}},
		{"pi two blocks", "pi", func(t *testing.T, p fstest.MapFS) {
			p["AGENTS.md"] = file(assistantBlock(t) + "\n" + assistantBlock(t))
		}, []string{line("AGENTS.md", "duplicated", "pi")}},
		{"pi block in CLAUDE.md only", "pi", func(t *testing.T, p fstest.MapFS) {
			delete(p, "AGENTS.md")
			p["CLAUDE.md"] = file("# Claude\n\n" + assistantBlock(t))
		}, nil},
		{"pi link", "pi", func(_ *testing.T, p fstest.MapFS) { delete(p, sharedLink) },
			[]string{line(sharedLink, "incomplete", "pi")}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			for _, e := range builtEngines() {
				if e.engine != c.engine {
					continue
				}
				proj := e.build(t, "1.2.3")
				c.edit(t, proj)
				r := e.run(t, proj)
				if c.want == nil {
					if got := entryLines(r); len(got) != 0 {
						t.Fatalf("entry lines %q", got)
					}
					wantPassed(t, r, PassedLine)
					return
				}
				wantEntryLines(t, r, c.want...)
			}
		})
	}
}

// S5: another engine's entry while runtimes.assistant is claude fails with
// the stale line, once per file, naming the first runtime skill in catalog
// order that declares it.
func TestMainSessionStaleEntriesFail(t *testing.T) {
	t.Parallel()
	block := assistantBlock(t)
	proj := claudeBuilt(t, "1.2.3")
	proj["AGENTS.md"] = file("# Project\n\n" + block)
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantEntryLines(t, r, "✗ AGENTS.md: gabo assistant entry for codex, but runtimes.assistant is claude")

	proj[".agents/skills/gabo-assistant"] = symlink("../../.gabo/skills/gabo-assistant")
	proj[".opencode/agents/gabo-assistant.md"] = file("---\nmode: primary\n---\n<!-- gabo-version: 1.2.3 -->\nbody\n")
	proj["opencode.json"] = file(`{"default_agent": "gabo-assistant"}`)
	proj["CLAUDE.md"] = file(block)
	r = run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantEntryLines(t, r,
		"✗ .agents/skills/gabo-assistant: gabo assistant entry for codex, but runtimes.assistant is claude",
		"✗ AGENTS.md: gabo assistant entry for codex, but runtimes.assistant is claude",
		"✗ .opencode/agents/gabo-assistant.md: gabo assistant entry for opencode, but runtimes.assistant is claude",
		"✗ opencode.json: gabo assistant entry for opencode, but runtimes.assistant is claude",
		"✗ CLAUDE.md: gabo assistant entry for pi, but runtimes.assistant is claude")
}

// A user's own files are never stale: an agent file without a gabo-version
// line, another default_agent value, a link elsewhere, no markers.
func TestMainSessionUserFilesAreNotStale(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	proj["AGENTS.md"] = file("# Project\n")
	proj[".agents/skills/gabo-assistant"] = symlink("../../my/gabo-assistant")
	proj[".opencode/agents/gabo-assistant.md"] = file("---\nmode: primary\n---\nmine\n")
	proj["opencode.json"] = file(`{"default_agent": "build"}`)
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	if got := entryLines(r); len(got) != 0 {
		t.Fatalf("entry lines %q", got)
	}
	wantPassed(t, r, PassedLine)
}

// Auto decision remove-other-entries-not-assistant: Codex and Pi share
// AGENTS.md, so with either as runtimes.assistant the block there is its own
// and never stale for the other.
func TestMainSessionSharedFileIsNotStale(t *testing.T) {
	t.Parallel()
	proj := codexBuilt(t, "1.2.3")
	for name, f := range piBuilt(t, "1.2.3") {
		proj[name] = f
	}
	for _, cfg := range []string{codexConfig, codexPiConfig} {
		rep, err := NewUseCase(proj, configFiles{projectFile: cfg}, codexUser(),
			Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}).Run()
		if err != nil {
			t.Fatal(err)
		}
		if got := entryLines(rep); len(got) != 0 {
			t.Fatalf("entry lines %q", got)
		}
		wantPassed(t, rep, PassedLine)
	}
}

// D5: the stale rule reads metadata.main_session of every runtime skill in
// the catalog copy, so a catalog copy declaring another file moves the rule.
func TestMainSessionStaleRuleReadsTheCatalogCopy(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	skill := CatalogDir + "/gabo-runtime-codex/SKILL.md"
	proj[skill] = file(strings.Replace(string(proj[skill].Data), "files: [AGENTS.override.md, AGENTS.md]", "files: [NOTES.md]", 1))
	proj[CatalogDir+"/gabo-runtime-pi/SKILL.md"] = file("---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n---\n")
	proj["AGENTS.md"] = file(assistantBlock(t))
	proj["NOTES.md"] = file(assistantBlock(t))
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantEntryLines(t, r, "✗ NOTES.md: gabo assistant entry for codex, but runtimes.assistant is claude")
}

// D5, TRD D28: a full-tier runtime skill without metadata.main_session fails;
// a flexible-tier one gets one warning line and no failure.
func TestMainSessionDeclaration(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	skill := CatalogDir + "/gabo-runtime-claude/SKILL.md"
	doc := string(proj[skill].Data)
	start := strings.Index(doc, "  main_session:\n")
	end := strings.Index(doc, "    key: agent\n") + len("    key: agent\n")
	if start < 0 || end < start {
		t.Fatalf("%s has no main_session block to drop", skill)
	}
	proj[skill] = file(doc[:start] + doc[end:])
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, "✗ runtime claude: gabo-runtime-claude declares no metadata.main_session", NextRestoreCatalog)

	var fr Report
	if declaresMainSession(&fr, "zed", runtimeMeta{AgentDir: ".zed/agents"}) {
		t.Fatal("zed declares main_session")
	}
	if fr.Failed || len(fr.Lines) != 1 || !strings.HasPrefix(fr.Lines[0], "! ") || !strings.Contains(fr.Lines[0], "zed") {
		t.Fatalf("flexible tier: lines = %q, failed = %v; want one ! line naming zed", fr.Lines, fr.Failed)
	}
}

// D5: metadata.main_session parses in both kinds and refuses what is not.
func TestParseMainSessionMeta(t *testing.T) {
	t.Parallel()
	meta := func(body string) string { return "---\nname: x\nmetadata:\n  main_session:\n" + body + "---\n" }
	m, err := parseRuntimeMeta([]byte(meta("    kind: agent\n    config: [ ./a.json ]\n    key: agent\n")))
	if err != nil || m.MainSession == nil || m.MainSession.Kind != kindAgent || m.MainSession.Config[0] != "a.json" {
		t.Fatalf("agent: %+v, %v", m.MainSession, err)
	}
	m, err = parseRuntimeMeta([]byte(meta("    kind: block\n    files: [A.md, B.md]\n")))
	if err != nil || m.MainSession == nil || len(m.MainSession.Files) != 2 {
		t.Fatalf("block: %+v, %v", m.MainSession, err)
	}
	for _, bad := range []string{
		"    kind: hook\n",
		"    kind: agent\n    key: agent\n",
		"    kind: agent\n    config: [a.json]\n",
		"    kind: block\n",
		"    kind: block\n    files: [../A.md]\n",
		"    kind: block\n    files: [/A.md]\n",
	} {
		if _, err := parseRuntimeMeta([]byte(meta(bad))); err == nil {
			t.Errorf("%q parsed, want an error", bad)
		}
	}
	if m, err := parseRuntimeMeta([]byte("---\nname: x\nmetadata:\n  agent_dir: a\n---\n")); err != nil || m.MainSession != nil {
		t.Fatalf("absent: %+v, %v", m.MainSession, err)
	}
}

// S5, intent 0008 S9: under GABO_UPGRADE_PR=1 no entry line prints.
func TestMainSessionSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	delete(proj, ".claude/agents/"+assistantID+".md")
	proj["AGENTS.md"] = file(assistantBlock(t))
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	if got := entryLines(r); len(got) != 0 {
		t.Fatalf("entry lines %q", got)
	}
	wantPassed(t, r, UpgradePRLine)
}

// S5: gabo check writes nothing: the project tree is byte-equal after a run
// with entry failures.
func TestMainSessionWritesNothing(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	delete(proj, ".claude/skills/gabo-assistant")
	proj["AGENTS.md"] = file(assistantBlock(t))
	snapshot := map[string]string{}
	for k, v := range proj {
		snapshot[k] = v.Mode.String() + string(v.Data)
	}
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, NextRebuild)
	if len(proj) != len(snapshot) {
		t.Fatalf("project has %d files after the run, want %d", len(proj), len(snapshot))
	}
	for k, v := range proj {
		if snapshot[k] != v.Mode.String()+string(v.Data) {
			t.Fatalf("%s changed", k)
		}
	}
}

// Intent 0008 T1 grep: the main-session rule names no engine in Go; what
// each engine's entry is comes from its runtime skill's metadata.
func TestMainSessionNamesNoEngine(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile("mainsession.go")
	if err != nil {
		t.Fatal(err)
	}
	if m := regexp.MustCompile(`"(claude|codex|pi|opencode)"`).Find(b); m != nil {
		t.Fatalf("mainsession.go names engine %s", m)
	}
}

// The help names the main-session rule.
func TestCommandHelpNamesTheMainSessionRule(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"the main-session entry against runtimes.assistant", "metadata.main_session"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}
