package check

import (
	"io/fs"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"
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

// entryLines returns the lines naming the gabo assistant entry, a
// main_session declaration or a main-session key holding another value.
func entryLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		if strings.Contains(l, "gabo assistant entry") || strings.Contains(l, "main_session") ||
			strings.HasPrefix(l, "✗ ") && strings.HasSuffix(l, `, not "`+assistantID+`"`) {
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

// The next lines of intent 0010 S6, as the spec words them.
func initMerges(key, file string) string {
	return `next: run gabo init (it merges "` + key + `": "gabo-assistant" into ` + file + `), then rerun gabo check`
}

func initRemoves(key, file string) string {
	return `next: run gabo init (it removes "` + key + `": "gabo-assistant" from ` + file + `), then rerun gabo check`
}

func changeKey(key, file string) string {
	return `next: change ` + key + ` to "gabo-assistant" in ` + file + `, or remove it and run gabo init, then rerun gabo check`
}

// wantEntry fails unless the entry lines are exactly the ✗ lines of pairs,
// in order, each followed by its next line, and the run failed.
func wantEntry(t *testing.T, r Report, pairs ...[2]string) {
	t.Helper()
	var want []string
	for _, p := range pairs {
		want = append(want, p[0])
	}
	if got := entryLines(r); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("entry lines:\n%s\nwant:\n%s\noutput:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"), joined(r))
	}
	wantFailed(t, r)
	for _, p := range pairs {
		wantConsecutive(t, r, p[0], p[1])
	}
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
	// line is a piece whose fix is a rebuild; key is the config piece of a
	// full-tier kind agent entry, whose fix is gabo init (intent 0010 S6).
	line := func(file, word, engine string) [2]string {
		return [2]string{"✗ " + file + ": gabo assistant entry " + word + " for " + engine, NextRebuild}
	}
	key := func(file, word, engine, key string) [2]string {
		return [2]string{"✗ " + file + ": gabo assistant entry " + word + " for " + engine, initMerges(key, file)}
	}
	cases := []struct {
		name   string
		engine string
		edit   func(*testing.T, fstest.MapFS)
		want   [][2]string
	}{
		{"claude link", "claude", func(_ *testing.T, p fstest.MapFS) { delete(p, claudeLink) },
			[][2]string{line(claudeLink, "incomplete", "claude")}},
		{"claude agent file", "claude", func(_ *testing.T, p fstest.MapFS) { delete(p, claudeAgent) },
			[][2]string{line(claudeAgent, "incomplete", "claude")}},
		{"claude key", "claude", func(_ *testing.T, p fstest.MapFS) { p[claudeKeys] = file(claudeKeysJSON) },
			[][2]string{key(claudeKeys, "incomplete", "claude", "agent")}},
		{"claude settings missing", "claude", func(_ *testing.T, p fstest.MapFS) { delete(p, claudeKeys) },
			[][2]string{key(claudeKeys, "incomplete", "claude", "agent")}},
		// gabo init refuses a file it cannot parse strictly, so the hint is
		// its reason, never a gabo init run that would loop (intent 0010 D7).
		{"claude settings not JSON", "claude", func(_ *testing.T, p fstest.MapFS) { p[claudeKeys] = file("{") },
			[][2]string{{line(claudeKeys, "incomplete", "claude")[0],
				"next: fix .claude/settings.json (invalid JSON (unexpected end of JSON input)), then run gabo init and rerun gabo check"}}},
		{"claude settings with a comment", "claude", func(_ *testing.T, p fstest.MapFS) { p[claudeKeys] = file("// mine\n" + claudeKeysJSON) },
			[][2]string{{line(claudeKeys, "incomplete", "claude")[0],
				"next: fix .claude/settings.json (invalid JSON (invalid character '/' looking for beginning of value)), then run gabo init and rerun gabo check"}}},
		{"unparsable opencode config fails the check", "opencode", func(_ *testing.T, p fstest.MapFS) {
			p[ocConfig] = file(`{"default_agent": `)
		}, [][2]string{{line(ocConfig, "incomplete", "opencode")[0],
			"next: fix " + ocConfig + " (invalid JSONC (unexpected end of JSON input)), then run gabo init and rerun gabo check"}}},
		{"claude user's own agent value", "claude", func(_ *testing.T, p fstest.MapFS) {
			p[claudeKeys] = file(withJSONKey(claudeKeysJSON, "agent", "my-agent"))
		}, [][2]string{{`✗ .claude/settings.json: agent is "my-agent", not "gabo-assistant"`, changeKey("agent", claudeKeys)}}},
		{"claude agent value not a string", "claude", func(_ *testing.T, p fstest.MapFS) {
			p[claudeKeys] = file(strings.Replace(claudeKeysJSON, "{", "{\n  \"agent\": [\"<a>\"],", 1))
		}, [][2]string{{`✗ .claude/settings.json: agent is ["<a>"], not "gabo-assistant"`, changeKey("agent", claudeKeys)}}},
		{"claude nothing", "claude", func(_ *testing.T, p fstest.MapFS) {
			for _, sk := range []string{"gabo-assistant", "gabo-cycle", "gabo-wave-report"} {
				delete(p, ".claude/skills/"+sk)
			}
			delete(p, claudeAgent)
			p[claudeKeys] = file(claudeKeysJSON)
		}, [][2]string{line(claudeLink, "missing", "claude"), line(".claude/skills/gabo-cycle", "missing", "claude"), line(".claude/skills/gabo-wave-report", "missing", "claude"), line(claudeAgent, "missing", "claude"), key(claudeKeys, "missing", "claude", "agent")}},
		{"opencode agent file and key", "opencode", func(t *testing.T, p fstest.MapFS) {
			delete(p, ocAgent)
			p[ocConfig] = file(readTestdata(t, "opencode.jsonc"))
		}, [][2]string{line(ocAgent, "incomplete", "opencode"), key(ocConfig, "incomplete", "opencode", "default_agent")}},
		{"opencode key", "opencode", func(t *testing.T, p fstest.MapFS) { p[ocConfig] = file(readTestdata(t, "opencode.jsonc")) },
			[][2]string{key(ocConfig, "incomplete", "opencode", "default_agent")}},
		{"opencode user's own default_agent", "opencode", func(t *testing.T, p fstest.MapFS) {
			p[ocConfig] = file(withJSONKey(readTestdata(t, "opencode.jsonc"), "default_agent", "build"))
		}, [][2]string{{`✗ .opencode/opencode.jsonc: default_agent is "build", not "gabo-assistant"`, changeKey("default_agent", ocConfig)}}},
		// gabo init creates .opencode/opencode.json when no config exists,
		// so the hint names that one file.
		{"opencode no config", "opencode", func(_ *testing.T, p fstest.MapFS) { delete(p, ocConfig) },
			[][2]string{{line("opencode.json or opencode.jsonc or .opencode/opencode.json or .opencode/opencode.jsonc", "incomplete", "opencode")[0],
				initMerges("default_agent", ".opencode/opencode.json")}}},
		// Intent 0015 S7: gabo-cycle and gabo-wave-report are main_session.skills.
		{"claude cycle link", "claude", func(_ *testing.T, p fstest.MapFS) { delete(p, ".claude/skills/gabo-cycle") },
			[][2]string{line(".claude/skills/gabo-cycle", "incomplete", "claude")}},
		{"codex wave-report link", "codex", func(_ *testing.T, p fstest.MapFS) { delete(p, ".agents/skills/gabo-wave-report") },
			[][2]string{line(".agents/skills/gabo-wave-report", "incomplete", "codex")}},
		{"opencode link", "opencode", func(_ *testing.T, p fstest.MapFS) { delete(p, sharedLink) },
			[][2]string{line(sharedLink, "incomplete", "opencode")}},
		{"codex block", "codex", func(_ *testing.T, p fstest.MapFS) { p["AGENTS.md"] = file("# Project\n") },
			[][2]string{line("AGENTS.md", "incomplete", "codex")}},
		{"codex nothing", "codex", func(_ *testing.T, p fstest.MapFS) {
			for _, sk := range []string{"gabo-assistant", "gabo-cycle", "gabo-wave-report"} {
				delete(p, ".agents/skills/"+sk)
			}
			delete(p, "AGENTS.md")
		}, [][2]string{line(sharedLink, "missing", "codex"), line(".agents/skills/gabo-cycle", "missing", "codex"), line(".agents/skills/gabo-wave-report", "missing", "codex"), line("AGENTS.override.md or AGENTS.md", "missing", "codex")}},
		// The block must be in the first existing file of the order.
		{"codex block below an override", "codex", func(_ *testing.T, p fstest.MapFS) { p["AGENTS.override.md"] = file("# Local\n") },
			[][2]string{line("AGENTS.override.md", "incomplete", "codex")}},
		{"codex half a block", "codex", func(t *testing.T, p fstest.MapFS) {
			p["AGENTS.md"] = file(strings.Replace(assistantBlock(t), assistantEnd, "", 1))
		}, [][2]string{line("AGENTS.md", "incomplete", "codex")}},
		{"pi two blocks", "pi", func(t *testing.T, p fstest.MapFS) {
			p["AGENTS.md"] = file(assistantBlock(t) + "\n" + assistantBlock(t))
		}, [][2]string{line("AGENTS.md", "duplicated", "pi")}},
		{"pi block in CLAUDE.md only", "pi", func(t *testing.T, p fstest.MapFS) {
			delete(p, "AGENTS.md")
			p["CLAUDE.md"] = file("# Claude\n\n" + assistantBlock(t))
		}, nil},
		{"pi link", "pi", func(_ *testing.T, p fstest.MapFS) { delete(p, sharedLink) },
			[][2]string{line(sharedLink, "incomplete", "pi")}},
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
				wantEntry(t, r, c.want...)
			}
		})
	}
}

// Intent 0010 S6, D4: the gabo init hint is for full-tier engines only; a
// flexible-tier engine's builder-written runtime skill writes its own key,
// so its config piece keeps the rebuild hint, and another value of its key
// stays an incomplete entry.
func TestMainSessionFlexibleTierKeepsTheRebuildHint(t *testing.T) {
	t.Parallel()
	spec, err := agentspec.Load(skills.Catalog())
	if err != nil {
		t.Fatal(err)
	}
	meta := runtimeMeta{MainSession: &mainSessionMeta{Kind: kindAgent, Config: []string{".zed/settings.json"}, Key: "agent"}}
	for _, doc := range []string{"", `{}`, `{"agent": "mine"}`} {
		proj := fstest.MapFS{}
		if doc != "" {
			proj[".zed/settings.json"] = file(doc)
		}
		var r Report
		if err := NewUseCase(proj, nil, nil, Settings{}).checkAssistantEntry(&r, []string{"zed"}, []string{"zed"}, "zed", meta, spec); err != nil {
			t.Fatal(err)
		}
		want := []string{"✗ .zed/settings.json: gabo assistant entry missing for zed", NextRebuild}
		if strings.Join(r.Lines, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%q: lines %q, want %q", doc, r.Lines, want)
		}
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
	proj[".agents/skills/gabo-cycle"] = symlink("../../.gabo/skills/gabo-cycle")
	proj[".agents/skills/gabo-wave-report"] = symlink("../../.gabo/skills/gabo-wave-report")
	proj[".opencode/agents/gabo-assistant.md"] = file("---\nmode: primary\n---\n<!-- gabo-version: 1.2.3 -->\nbody\n")
	proj["opencode.json"] = file(`{"default_agent": "gabo-assistant"}`)
	proj["CLAUDE.md"] = file(block)
	r = run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	stale := func(file, rt string) string {
		return "✗ " + file + ": gabo assistant entry for " + rt + ", but runtimes.assistant is claude"
	}
	// Intent 0010 S6: the stale key is gabo init's to remove; the other
	// pieces keep the rebuild hint.
	wantEntry(t, r,
		[2]string{stale(".agents/skills/gabo-assistant", "codex"), NextRebuild},
		[2]string{stale(".agents/skills/gabo-cycle", "codex"), NextRebuild},
		[2]string{stale(".agents/skills/gabo-wave-report", "codex"), NextRebuild},
		[2]string{stale("AGENTS.md", "codex"), NextRebuild},
		[2]string{stale(".opencode/agents/gabo-assistant.md", "opencode"), NextRebuild},
		[2]string{stale("opencode.json", "opencode"), initRemoves("default_agent", "opencode.json")},
		[2]string{stale("CLAUDE.md", "pi"), NextRebuild})
}

// Intent 0010 S6, D7: a leftover Claude Code agent key while OpenCode is the
// assistant names gabo init when gabo init removes it, and removing it by
// hand when gabo init would leave the file alone (it reads strict JSON).
func TestMainSessionStaleClaudeKey(t *testing.T) {
	t.Parallel()
	const settings = ".claude/settings.json"
	for _, c := range []struct{ doc, next string }{
		{`{"agent": "gabo-assistant"}`, initRemoves("agent", settings)},
		{"// mine\n{\"agent\": \"gabo-assistant\"}\n", `next: remove "agent": "gabo-assistant" from .claude/settings.json, then rerun gabo check`},
	} {
		proj := opencodeBuilt(t, "1.2.3")
		proj[settings] = file(c.doc)
		r := runOpencode(t, proj, "1.2.3")
		wantEntry(t, r, [2]string{"✗ " + settings + ": gabo assistant entry for claude, but runtimes.assistant is opencode", c.next})
	}
}

// Intent 0010 D7: with OpenCode in the engine set but not the assistant,
// gabo init refuses every OpenCode change while one of its configs does not
// parse, so the stale-key line names that file to fix.
func TestMainSessionHintNamesTheFileGaboInitRefuses(t *testing.T) {
	t.Parallel()
	proj := claudeBuilt(t, "1.2.3")
	proj["opencode.json"] = file(`{"default_agent": "gabo-assistant"}`)
	proj[".opencode/opencode.jsonc"] = file("{")
	cfg := strings.Replace(validConfig, "  coder: claude\n", "  coder: opencode\n", 1)
	r := run(t, proj, configFiles{projectFile: cfg}, Settings{})
	fix := "next: fix .opencode/opencode.jsonc ("
	for _, l := range []string{
		"✗ opencode.json: gabo assistant entry for opencode, but runtimes.assistant is claude",
	} {
		i := slices.Index(r.Lines, l)
		if i < 0 || i+1 >= len(r.Lines) || !strings.HasPrefix(r.Lines[i+1], fix) {
			t.Fatalf("want %q followed by a line starting %q:\n%s", l, fix, joined(r))
		}
	}
}

// Intent 0010 S6: a leftover default_agent while runtimes.assistant is
// claude is the only line, with the gabo init hint, in each OpenCode config.
func TestMainSessionStaleKeyNamesGaboInit(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"opencode.json", "opencode.jsonc", ".opencode/opencode.json", ".opencode/opencode.jsonc"} {
		proj := claudeBuilt(t, "1.2.3")
		proj[name] = file("// mine\n{\"default_agent\": \"gabo-assistant\",}\n")
		r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
		wantEntry(t, r, [2]string{"✗ " + name + ": gabo assistant entry for opencode, but runtimes.assistant is claude",
			initRemoves("default_agent", name)})
	}
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
	proj[".claude/settings.json"] = file(withJSONKey(claudeKeysJSON, "agent", "my-agent"))
	proj["opencode.json"] = file(`{"default_agent": "gabo-assistant"}`)
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
	for _, want := range []string{"the main-session entry of every engine listed in runtimes.assistant", "metadata.main_session"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

// listedRuntimes lists two assistant engines (intent 0013 S1).
const listedRuntimes = "  coder: claude\n  reviewer: claude\n  assistant: [claude, pi]\n"

// listedBuilt is a correct build of Claude Code and Pi, both listed in
// runtimes.assistant: each engine's agents, links, keys and main-session
// entry.
func listedBuilt(t *testing.T) fstest.MapFS {
	t.Helper()
	proj := claudeBuilt(t, "1.2.3")
	maps.Copy(proj, piBuilt(t, "1.2.3"))
	return proj
}

func runListed(t *testing.T, proj fs.FS) Report {
	t.Helper()
	return run(t, proj, configFiles{projectFile: withRuntimes(t, validConfig, listedRuntimes)}, Settings{})
}

// Intent 0013 S5: the main-session entry of every engine listed in
// runtimes.assistant is checked; one built entry never covers another, and
// a listed engine's entry is never stale (Pi's AGENTS.md block is not
// Codex's).
func TestMainSessionEveryListedAssistant(t *testing.T) {
	t.Parallel()
	r := runListed(t, listedBuilt(t))
	if got := entryLines(r); len(got) != 0 {
		t.Fatalf("entry lines %q:\n%s", got, joined(r))
	}
	wantPassed(t, r, PassedLine)

	proj := listedBuilt(t)
	proj["AGENTS.md"] = file("# Project\n")
	wantEntry(t, runListed(t, proj), [2]string{"✗ AGENTS.md: gabo assistant entry incomplete for pi", NextRebuild})

	proj = listedBuilt(t)
	proj[".claude/settings.json"] = file(claudeKeysJSON)
	wantEntry(t, runListed(t, proj), [2]string{"✗ .claude/settings.json: gabo assistant entry incomplete for claude",
		initMerges("agent", ".claude/settings.json")})
}

// Intent 0013 S5: a stale entry fails only for a full-tier engine that is
// not listed, and the stale line names the listed engines joined with ", ".
func TestMainSessionStaleLineNamesTheList(t *testing.T) {
	t.Parallel()
	proj := listedBuilt(t)
	proj["opencode.json"] = file(`{"default_agent": "gabo-assistant"}`)
	wantEntry(t, runListed(t, proj), [2]string{
		"✗ opencode.json: gabo assistant entry for opencode, but runtimes.assistant is claude, pi",
		initRemoves("default_agent", "opencode.json")})
}
