package initcmd

import (
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/")

const (
	claudeFile = root + "/" + engineconfig.ClaudeSettings
	codexFile  = root + "/" + engineconfig.CodexConfigPath
	piFile     = root + "/" + engineconfig.PiSettings
	stageGate  = root + "/" + engineconfig.OpencodeStageGate
)

// golden compares got with testdata/<name>, or rewrites it with -update.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Fatalf("%s differs:\n--- got\n%s\n--- want\n%s", name, got, want)
	}
}

func engineStep(s *memStore, set ...string) engineconfig.Report {
	return NewUseCase(s, catalog, Settings{ProjectDir: root, Version: "0.2.6"}).engineStep(set, "", "0.2.6")
}

func sectionOf(t *testing.T, r engineconfig.Report, engine string) engineconfig.Section {
	t.Helper()
	for _, s := range r.Sections {
		if s.Engine == engine {
			return s
		}
	}
	t.Fatalf("no %s section in %+v", engine, r.Sections)
	return engineconfig.Section{}
}

func failures(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(l, engineconfig.FailurePrefix) {
			out = append(out, l)
		}
	}
	return out
}

// All four full-tier engines on an empty project: each engine file is
// written once, inside the project, and the block matches the golden (S9).
func TestEngineStepAllFourEnginesGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := engineStep(s, engineconfig.Set("pi", "opencode", "codex", "claude")...)
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	golden(t, "engines-all.golden", r.Render())
	if want := slices.Sorted(slices.Values([]string{claudeFile, codexFile, opencodeNew, piFile})); !slices.Equal(slices.Sorted(slices.Values(s.Writes)), want) {
		t.Fatalf("writes = %v, want %v", s.Writes, want)
	}
	if _, ok := s.Files[stageGate]; ok {
		t.Fatal("opencode stage gate written")
	}
}

// Only changed files are written: a second run writes nothing and every
// written engine reports unchanged.
func TestEngineStepWritesOnlyChangedFiles(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	set := engineconfig.Set("claude", "codex", "pi")
	engineStep(s, set...)
	before := maps.Clone(s.Files)
	s.Writes = nil
	r := engineStep(s, set...)
	if len(s.Writes) != 0 || !maps.Equal(s.Files, before) {
		t.Fatalf("second run wrote %v", s.Writes)
	}
	for _, e := range []string{"claude", "codex", "pi"} {
		if !strings.Contains(strings.Join(sectionOf(t, r, e).Lines, "\n"), "unchanged") {
			t.Fatalf("%s section lacks unchanged:\n%s", e, r.Render())
		}
	}
}

// A flexible-tier engine (S8) gets one line after the full-tier sections,
// no step, no file and no failure; claude is still written.
func TestEngineStepFlexibleTierEngine(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := engineStep(s, engineconfig.Set("acme", "claude")...)
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	if !slices.Equal(s.Writes, []string{claudeFile}) {
		t.Fatalf("writes = %v, want only %s", s.Writes, claudeFile)
	}
	var engines []string
	for _, sec := range r.Sections {
		engines = append(engines, sec.Engine)
	}
	if !slices.Equal(engines, []string{"claude", "acme"}) {
		t.Fatalf("sections = %v, want claude then acme", engines)
	}
	acme := sectionOf(t, r, "acme")
	if !slices.Equal(acme.Lines, []string{"acme: flexible tier: skills and configuration only; nothing written"}) || len(acme.Steps) != 0 {
		t.Fatalf("acme section = %+v", acme)
	}
	if strings.Contains(r.Render(), "- acme:") {
		t.Fatalf("acme step listed:\n%s", r.Render())
	}
}

// The opencode rule sees the stage-gate plugin when it is in the project.
func TestEngineStepLoadsOpencodeStageGate(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{stageGate: "export default {}\n", opencodeRootCfg: depthOK})
	r := engineStep(s, "opencode")
	if got := sectionOf(t, r, "opencode").Lines; !slices.Equal(got, []string{"stage-gate plugin in place (built by gabo-runtime-opencode)"}) {
		t.Fatalf("opencode lines = %q", got)
	}
	if len(s.Writes) != 0 {
		t.Fatalf("writes = %v", s.Writes)
	}
}

// S5: a symlinked or wrong-kind engine file or engine directory is left
// untouched with one ✗ line; the other engines still run.
func TestEngineStepRefusesSymlinksAndWrongKinds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		files  map[string]string
		modes  map[string]fs.FileMode
		engine string
		target string
		reason string
	}{
		{"claude file symlink", map[string]string{claudeFile: "{}"}, map[string]fs.FileMode{claudeFile: fs.ModeSymlink}, "claude", claudeFile, "symlink"},
		{"claude file a directory", nil, map[string]fs.FileMode{root + "/.claude": fs.ModeDir, claudeFile: fs.ModeDir}, "claude", claudeFile, "not a regular file"},
		{"claude file a fifo", nil, map[string]fs.FileMode{root + "/.claude": fs.ModeDir, claudeFile: fs.ModeNamedPipe}, "claude", claudeFile, "not a regular file"},
		{".claude symlink", nil, map[string]fs.FileMode{root + "/.claude": fs.ModeSymlink | fs.ModeDir}, "claude", claudeFile, ".claude is a symlink"},
		{".claude a file", nil, map[string]fs.FileMode{root + "/.claude": 0}, "claude", claudeFile, ".claude is not a directory"},
		{".codex symlink", nil, map[string]fs.FileMode{root + "/.codex": fs.ModeSymlink}, "codex", codexFile, ".codex is a symlink"},
		{".pi symlink", nil, map[string]fs.FileMode{root + "/.pi": fs.ModeSymlink}, "pi", piFile, ".pi is a symlink"},
		{"invalid json", map[string]string{claudeFile: "{"}, nil, "claude", claudeFile, "fix it"},
		{"invalid toml", map[string]string{codexFile: "[x"}, nil, "codex", codexFile, "invalid TOML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := maps.Clone(tt.files)
			if files == nil {
				files = map[string]string{}
			}
			s := newStore(files)
			s.Modes = tt.modes
			before, had := s.Files[tt.target]
			r := engineStep(s, engineconfig.Set("claude", "codex", "pi")...)
			if after, ok := s.Files[tt.target]; ok != had || after != before || slices.Contains(s.Writes, tt.target) {
				t.Fatalf("%s touched: %q → %q, writes %v", tt.target, before, after, s.Writes)
			}
			got := failures(sectionOf(t, r, tt.engine).Lines)
			rel := strings.TrimPrefix(tt.target, root+"/")
			if len(got) != 1 || !strings.HasPrefix(got[0], "✗ "+rel+": ") || !strings.Contains(got[0], tt.reason) {
				t.Fatalf("✗ lines = %q, want one for %s naming %q", got, rel, tt.reason)
			}
			if !r.Failed() {
				t.Fatal("Failed() = false")
			}
			for _, other := range []string{claudeFile, codexFile, piFile} {
				if other != tt.target && !slices.Contains(s.Writes, other) {
					t.Fatalf("%s not written; writes %v", other, s.Writes)
				}
			}
		})
	}
}

// Engine step writes stay inside the project and never reach a user-level
// engine directory (S2, S3, S7).
func TestEngineStepWritesOnlyInsideTheProject(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	engineStep(s, engineconfig.Set("claude", "codex", "opencode", "pi")...)
	for _, w := range s.Writes {
		if !strings.HasPrefix(w, root+"/") {
			t.Fatalf("write outside the project: %s", w)
		}
	}
}

// The init slice runs no process and looks for no engine on the machine
// (S1, D1): no source imports os/exec or takes a kernel.Runner.
func TestInitSourcesRunNoProcess(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		for _, bad := range []string{`"os/exec"`, "kernel.Runner", "exec.LookPath", "UserHomeDir"} {
			if strings.Contains(string(src), bad) {
				t.Errorf("%s contains %s", f, bad)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no source checked")
	}
}

func engineStepAs(s *memStore, assistant string, set ...string) engineconfig.Report {
	return NewUseCase(s, catalog, Settings{ProjectDir: root, Version: "0.2.6"}).engineStep(set, assistant, "0.2.6")
}

const (
	claudeAgent      = root + "/" + engineconfig.ClaudeAgentFile
	integrationOnly  = "{\n  \"enabledPlugins\": {\n    \"gabo@gabo-ai\": true\n  },\n  \"extraKnownMarketplaces\": {\n    \"gabo-ai\": {\n      \"source\": {\n        \"source\": \"github\",\n        \"repo\": \"testeos-z/gabo-ai\"\n      }\n    }\n  }\n}\n"
	withGaboAgent    = "{\n  \"enabledPlugins\": {\n    \"gabo@gabo-ai\": true\n  },\n  \"extraKnownMarketplaces\": {\n    \"gabo-ai\": {\n      \"source\": {\n        \"source\": \"github\",\n        \"repo\": \"testeos-z/gabo-ai\"\n      }\n    }\n  },\n  \"agent\": \"gabo-assistant\"\n}\n"
	buildAgentStep   = "  - claude: ask your agent to build the agents (the main session needs .claude/agents/gabo-assistant.md)\n"
	removedGaboAgent = `removed .claude/settings.json agent: "gabo-assistant" (runtimes.assistant is codex)`
)

// Intent 0010 S1, S8: with Claude Code as the assistant a fresh project gets
// the agent key with the integration keys, and the report names agent and
// the build step while the agent file is missing.
func TestEngineStepClaudeAssistantGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := engineStepAs(s, "claude", engineconfig.Set("codex", "claude", "claude")...)
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	golden(t, "engines-claude-assistant.golden", r.Render())
	if !strings.Contains(s.Files[claudeFile], `"agent": "gabo-assistant"`) {
		t.Fatalf("%s =\n%s", claudeFile, s.Files[claudeFile])
	}
	// Once the agent is built, the next run changes nothing and drops the step.
	s.Files[claudeAgent] = "---\nname: gabo-assistant\n---\n"
	s.Writes = nil
	again := engineStepAs(s, "claude", engineconfig.Set("codex", "claude")...)
	if len(s.Writes) != 0 || strings.Contains(again.Render(), "main session needs") {
		t.Fatalf("second run wrote %v:\n%s", s.Writes, again.Render())
	}
}

// Intent 0010 S1: the existing integration keys keep every byte, the user's
// own agent is kept and reported, and an unreadable or symlinked file is
// refused with one ✗ line.
func TestEngineStepClaudeAgentKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		files  map[string]string
		modes  map[string]fs.FileMode
		want   string // the file after the run
		lines  []string
		failed bool
	}{
		{"adds agent", map[string]string{claudeFile: integrationOnly}, nil, withGaboAgent,
			[]string{"wrote .claude/settings.json: added agent"}, false},
		{"already set", map[string]string{claudeFile: withGaboAgent}, nil, withGaboAgent,
			[]string{"unchanged .claude/settings.json"}, false},
		{"user's agent", map[string]string{claudeFile: strings.Replace(withGaboAgent, "gabo-assistant", "my-agent", 1)}, nil,
			strings.Replace(withGaboAgent, "gabo-assistant", "my-agent", 1),
			[]string{`kept .claude/settings.json agent: "my-agent" (your setting; gabo leaves it)`}, false},
		{"invalid JSON", map[string]string{claudeFile: `{"agent": `}, nil, `{"agent": `,
			[]string{"✗ .claude/settings.json: invalid JSON (unexpected end of JSON input); fix it and run gabo init again"}, true},
		{"top level not an object", map[string]string{claudeFile: `[]`}, nil, `[]`,
			[]string{"✗ .claude/settings.json: the top level is not a JSON object; fix it and run gabo init again"}, true},
		{"symlink", map[string]string{claudeFile: integrationOnly}, map[string]fs.FileMode{claudeFile: fs.ModeSymlink}, integrationOnly,
			[]string{"✗ .claude/settings.json: it is a symlink; gabo init writes only inside the project, so replace it with a real file and run gabo init again"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(maps.Clone(tt.files))
			s.Modes = tt.modes
			r := engineStepAs(s, "claude", "claude")
			if got := s.Files[claudeFile]; got != tt.want {
				t.Fatalf("%s =\n%s\nwant\n%s", claudeFile, got, tt.want)
			}
			if got := sectionOf(t, r, "claude").Lines; !slices.Equal(got, tt.lines) {
				t.Fatalf("lines = %q, want %q", got, tt.lines)
			}
			if r.Failed() != tt.failed {
				t.Fatalf("Failed() = %v, want %v", r.Failed(), tt.failed)
			}
		})
	}
}

// Intent 0010 S4, D2: another assistant engine removes only the gabo value
// and its comma, also when claude is no longer in runtimes.*; the user's own
// agent stays; a file of an engine outside the set that gabo cannot read is
// left alone without a ✗ line.
func TestEngineStepRemovesTheGaboAgent(t *testing.T) {
	t.Parallel()
	t.Run("claude still in runtimes", func(t *testing.T) {
		t.Parallel()
		s := newStore(map[string]string{claudeFile: withGaboAgent})
		r := engineStepAs(s, "codex", engineconfig.Set("claude", "codex")...)
		if s.Files[claudeFile] != integrationOnly {
			t.Fatalf("%s =\n%s", claudeFile, s.Files[claudeFile])
		}
		if got := sectionOf(t, r, "claude").Lines; !slices.Equal(got, []string{removedGaboAgent}) {
			t.Fatalf("lines = %q", got)
		}
	})
	t.Run("claude no longer in runtimes", func(t *testing.T) {
		t.Parallel()
		s := newStore(map[string]string{claudeFile: withGaboAgent})
		r := engineStepAs(s, "codex", "codex")
		if s.Files[claudeFile] != integrationOnly {
			t.Fatalf("%s =\n%s", claudeFile, s.Files[claudeFile])
		}
		golden(t, "engines-agent-removed.golden", r.Render())
		if r.Failed() {
			t.Fatal("Failed() = true")
		}
		s.Writes = nil
		again := engineStepAs(s, "codex", "codex")
		if slices.Contains(s.Writes, claudeFile) || strings.Contains(again.Render(), "claude") {
			t.Fatalf("second run wrote %v:\n%s", s.Writes, again.Render())
		}
	})
	for name, tt := range map[string]struct {
		file  string
		modes map[string]fs.FileMode
	}{
		"user's agent stays":  {strings.Replace(withGaboAgent, "gabo-assistant", "my-agent", 1), nil},
		"invalid JSON stays":  {`{"agent": "gabo-assistant"`, nil},
		"symlink is not read": {withGaboAgent, map[string]fs.FileMode{claudeFile: fs.ModeSymlink}},
		".claude symlink":     {withGaboAgent, map[string]fs.FileMode{root + "/.claude": fs.ModeSymlink | fs.ModeDir}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := newStore(map[string]string{claudeFile: tt.file})
			s.Modes = tt.modes
			r := engineStepAs(s, "codex", "codex")
			if s.Files[claudeFile] != tt.file || slices.Contains(s.Writes, claudeFile) {
				t.Fatalf("%s touched: %q, writes %v", claudeFile, s.Files[claudeFile], s.Writes)
			}
			if r.Failed() || strings.Contains(r.Render(), "claude") {
				t.Fatalf("report:\n%s", r.Render())
			}
		})
	}
}

const (
	opencodeAgent   = root + "/" + engineconfig.OpencodeAgentFile
	opencodeNew     = root + "/.opencode/opencode.json"
	opencodeRootCfg = root + "/opencode.json"
	opencodeJSONC   = root + "/opencode.jsonc"
	gaboDefault     = "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"model\": \"x\", // pinned\n  \"default_agent\": \"gabo-assistant\"\n}\n"
	noDefault       = "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"model\": \"x\" // pinned\n}\n"
	depthOK         = "{\"experimental\": {\"subagent_depth\": 3}}\n"
)

// Intent 0010 S2, S8: with OpenCode as the assistant a fresh project gets a
// new .opencode/opencode.json with default_agent, and the report names
// default_agent and the build step while the agent file is missing.
func TestEngineStepOpencodeAssistantGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := engineStepAs(s, "opencode", engineconfig.Set("claude", "opencode")...)
	if r.Failed() {
		t.Fatalf("failed:\n%s", r.Render())
	}
	golden(t, "engines-opencode-assistant.golden", r.Render())
	if got := s.Files[opencodeNew]; got != "{\n  \"experimental\": {\n    \"subagent_depth\": 3\n  },\n  \"default_agent\": \"gabo-assistant\"\n}\n" {
		t.Fatalf("%s =\n%s", opencodeNew, got)
	}
	s.Files[opencodeAgent] = "---\nname: gabo-assistant\n---\n"
	s.Writes = nil
	again := engineStepAs(s, "opencode", engineconfig.Set("claude", "opencode")...)
	if len(s.Writes) != 0 || strings.Contains(again.Render(), "main session needs") {
		t.Fatalf("second run wrote %v:\n%s", s.Writes, again.Render())
	}
}

// Intent 0010 S2: initcmd reads the OpenCode configs: the first existing one
// gains only the key, comments kept; the user's own value is kept; a
// symlinked target or .opencode directory writes nothing and prints one ✗.
func TestEngineStepOpencodeDefaultAgent(t *testing.T) {
	t.Parallel()
	jsonc := "{\n  // mine\n  \"model\": \"x\" /* pinned */\n}\n"
	tests := []struct {
		name   string
		files  map[string]string
		modes  map[string]fs.FileMode
		target string
		want   string
		lines  []string
		failed bool
	}{
		{"jsonc with comments", map[string]string{opencodeJSONC: jsonc, opencodeNew: "{}\n"}, nil, opencodeJSONC,
			"{\n  // mine\n  \"model\": \"x\", /* pinned */\n  \"experimental\": {\n    \"subagent_depth\": 3\n  },\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			[]string{"wrote opencode.jsonc: experimental.subagent_depth unset → 3; added default_agent", "no .opencode/plugins/gabo-stage-gate.js yet"}, false},
		{"user's value", map[string]string{opencodeRootCfg: `{"default_agent": "gentle-orchestrator", "experimental": {"subagent_depth": 3}}`}, nil, opencodeRootCfg,
			`{"default_agent": "gentle-orchestrator", "experimental": {"subagent_depth": 3}}`,
			[]string{`kept opencode.json default_agent: "gentle-orchestrator" (your setting; gabo leaves it)`, "no .opencode/plugins/gabo-stage-gate.js yet; nothing written"}, false},
		{"symlinked target", map[string]string{opencodeRootCfg: "{}"}, map[string]fs.FileMode{opencodeRootCfg: fs.ModeSymlink}, opencodeRootCfg, "{}",
			[]string{"✗ opencode.json: it is a symlink; gabo init writes only inside the project, so replace it with a real file and run gabo init again"}, true},
		{"symlinked .opencode", nil, map[string]fs.FileMode{root + "/.opencode": fs.ModeSymlink | fs.ModeDir}, opencodeNew, "",
			[]string{"✗ .opencode/opencode.json: .opencode is a symlink; gabo init writes only inside the project, so replace it with a real directory and run gabo init again"}, true},
		{"unparsable", map[string]string{opencodeRootCfg: "{,}"}, nil, opencodeRootCfg, "{,}",
			[]string{"✗ opencode.json: invalid JSONC (invalid character ',' looking for beginning of object key string); fix it and run gabo init again", "no .opencode/plugins/gabo-stage-gate.js yet"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			files := maps.Clone(tt.files)
			if files == nil {
				files = map[string]string{}
			}
			s := newStore(files)
			s.Modes = tt.modes
			r := engineStepAs(s, "opencode", "opencode")
			if got := s.Files[tt.target]; got != tt.want {
				t.Fatalf("%s =\n%s\nwant\n%s", tt.target, got, tt.want)
			}
			for _, w := range s.Writes {
				if w != tt.target || tt.failed {
					t.Fatalf("writes = %v", s.Writes)
				}
			}
			if got := sectionOf(t, r, "opencode").Lines; !slices.Equal(got, tt.lines) {
				t.Fatalf("lines =\n%q\nwant\n%q", got, tt.lines)
			}
			if r.Failed() != tt.failed {
				t.Fatalf("Failed() = %v", r.Failed())
			}
		})
	}
}

// Intent 0010 S4, D2: with another assistant engine each of the four OpenCode
// configs holding the gabo default_agent loses only that member, reported,
// also when opencode is no longer in runtimes.*; a symlinked file of an
// engine outside the set is left alone without a line.
func TestEngineStepRemovesTheGaboDefaultAgent(t *testing.T) {
	t.Parallel()
	all := func() map[string]string {
		files := map[string]string{}
		for _, p := range engineconfig.OpencodeConfigs {
			files[root+"/"+p] = gaboDefault
		}
		return files
	}
	t.Run("opencode no longer in runtimes", func(t *testing.T) {
		t.Parallel()
		s := newStore(all())
		r := engineStepAs(s, "claude", "claude")
		if r.Failed() {
			t.Fatalf("failed:\n%s", r.Render())
		}
		golden(t, "engines-default-agent-removed.golden", r.Render())
		for _, p := range engineconfig.OpencodeConfigs {
			if got := s.Files[root+"/"+p]; got != noDefault {
				t.Fatalf("%s =\n%s", p, got)
			}
		}
		s.Writes = nil
		again := engineStepAs(s, "claude", "claude")
		if len(s.Writes) != 0 || strings.Contains(again.Render(), "opencode") {
			t.Fatalf("second run wrote %v:\n%s", s.Writes, again.Render())
		}
	})
	t.Run("opencode still in runtimes", func(t *testing.T) {
		t.Parallel()
		s := newStore(all())
		r := engineStepAs(s, "claude", engineconfig.Set("claude", "opencode")...)
		lines := sectionOf(t, r, "opencode").Lines
		if len(lines) != 5 || lines[0] != `removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is claude)` || len(failures(lines)) != 0 {
			t.Fatalf("lines = %q", lines)
		}
		for _, p := range engineconfig.OpencodeConfigs {
			if got := s.Files[root+"/"+p]; got != noDefault {
				t.Fatalf("%s =\n%s", p, got)
			}
		}
	})
	t.Run("symlink outside the set is not read", func(t *testing.T) {
		t.Parallel()
		s := newStore(map[string]string{opencodeRootCfg: gaboDefault})
		s.Modes = map[string]fs.FileMode{opencodeRootCfg: fs.ModeSymlink}
		r := engineStepAs(s, "claude", "claude")
		if s.Files[opencodeRootCfg] != gaboDefault || slices.Contains(s.Writes, opencodeRootCfg) || strings.Contains(r.Render(), "opencode") || r.Failed() {
			t.Fatalf("writes %v:\n%s", s.Writes, r.Render())
		}
	})
}

// Intent 0010 S3, S8: with opencode in runtimes.* initcmd raises
// experimental.subagent_depth in the first existing OpenCode config, keeps
// its comments, reports the old and new value, and no step names the key.
func TestEngineStepOpencodeSubagentDepth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		files  map[string]string
		target string
		want   string
		line   string
		failed bool
	}{
		{"absent key in a new file", map[string]string{}, opencodeNew,
			"{\n  \"experimental\": {\n    \"subagent_depth\": 3\n  }\n}\n", "wrote .opencode/opencode.json: experimental.subagent_depth unset → 3", false},
		{"1 becomes 3", map[string]string{opencodeJSONC: "{\n  // mine\n  \"experimental\": {\"subagent_depth\": 1}\n}\n", opencodeNew: "{}\n"}, opencodeJSONC,
			"{\n  // mine\n  \"experimental\": {\"subagent_depth\": 3}\n}\n", "wrote opencode.jsonc: experimental.subagent_depth 1 → 3", false},
		{"5 is kept", map[string]string{opencodeRootCfg: `{"experimental": {"subagent_depth": 5}}`}, opencodeRootCfg,
			`{"experimental": {"subagent_depth": 5}}`, "no .opencode/plugins/gabo-stage-gate.js yet; nothing written", false},
		{"experimental not an object", map[string]string{opencodeRootCfg: `{"experimental": 3}`}, opencodeRootCfg,
			`{"experimental": 3}`, "✗ opencode.json: experimental is not a JSON object; fix it and run gabo init again", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(maps.Clone(tt.files))
			r := engineStep(s, engineconfig.Set("claude", "opencode")...)
			if got := s.Files[tt.target]; got != tt.want {
				t.Fatalf("%s =\n%s\nwant\n%s", tt.target, got, tt.want)
			}
			if got := sectionOf(t, r, "opencode").Lines; got[0] != tt.line {
				t.Fatalf("lines = %q, want %q first", got, tt.line)
			}
			if r.Failed() != tt.failed || (tt.failed && slices.Contains(s.Writes, tt.target)) {
				t.Fatalf("Failed() = %v, writes %v", r.Failed(), s.Writes)
			}
			if _, steps, _ := strings.Cut(r.Render(), "you still need to:"); strings.Contains(steps, "subagent_depth") {
				t.Fatalf("a step names subagent_depth:\n%s", r.Render())
			}
		})
	}
}
