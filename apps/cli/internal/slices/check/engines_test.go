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
)

// Engine files of a project integrated by gabo init (intent 0006 S2, S3, S7).
const (
	claudeKeysJSON = `{
  "extraKnownMarketplaces": {
    "gabo-ai": {
      "source": {
        "source": "github",
        "repo": "testeos-z/gabo-ai"
      }
    }
  },
  "enabledPlugins": {
    "gabo@gabo-ai": true
  }
}
`
	codexKeysTOML = `[marketplaces.gabo-ai]
source_type = "git"
source = "https://github.com/testeos-z/gabo-ai.git"

[plugins."gabo@gabo-ai"]
enabled = true
`
	piKeysJSON = `{"packages": ["npm:@gabo-ai/pi@0.2.6"]}
`
)

// Exact failure lines of spec S11.
const (
	claudeMissingPlugin = `✗ .claude/settings.json: enabledPlugins."gabo@gabo-ai" missing; run gabo init`
	claudeMissingMarket = `✗ .claude/settings.json: extraKnownMarketplaces."gabo-ai" missing; run gabo init`
	piMissing           = `✗ .pi/settings.json: packages entry npm:@gabo-ai/pi missing; run gabo init (a release build) or add "npm:@gabo-ai/pi@<version>" to packages`
)

func codexMissing(key string) string {
	return "✗ .codex/config.toml: " + key + " missing; run gabo init"
}

// engineFixture reads one file of testdata/engines.
func engineFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(path.Join("testdata", "engines", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// engineLines returns the report lines naming an engine file.
func engineLines(r Report) []string {
	var out []string
	for _, l := range r.Lines {
		for _, f := range []string{".claude/settings.json", ".codex/config.toml", ".pi/settings.json"} {
			if strings.Contains(l, f) {
				out = append(out, l)
				break
			}
		}
	}
	return out
}

func wantEngineLines(t *testing.T, r Report, want ...string) {
	t.Helper()
	if got := engineLines(r); !slices.Equal(got, want) {
		t.Fatalf("engine-key lines = %q, want %q; output:\n%s", got, want, joined(r))
	}
}

// S11: a missing Claude Code key fails with its own line; a key present with
// another value (false) passes (D6); a complete file passes.
func TestEngineKeysClaude(t *testing.T) {
	t.Parallel()
	noPlugin := `{
  "extraKnownMarketplaces": {"gabo-ai": {"source": {"source": "github", "repo": "testeos-z/gabo-ai"}}},
  "permissions": {"allow": ["Bash(ls)"]}
}
`
	r := run(t, with(project(), map[string]string{".claude/settings.json": noPlugin}), configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, claudeMissingPlugin)
	wantEngineLines(t, r, claudeMissingPlugin)

	r = run(t, with(project(), map[string]string{".claude/settings.json": ""}), configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, claudeMissingMarket, claudeMissingPlugin)
	wantEngineLines(t, r, claudeMissingMarket, claudeMissingPlugin)

	off := strings.Replace(claudeKeysJSON, `"gabo@gabo-ai": true`, `"gabo@gabo-ai": false`, 1)
	r = run(t, with(project(), map[string]string{".claude/settings.json": off}), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	wantEngineLines(t, r)

	r = run(t, with(project(), map[string]string{".claude/settings.json": claudeKeysJSON}), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}

// A settings file gabo init cannot read fails with gabo init's own line.
func TestEngineKeysClaudeInvalidFile(t *testing.T) {
	t.Parallel()
	r := run(t, with(project(), map[string]string{".claude/settings.json": "{"}), configFiles{projectFile: validConfig}, Settings{})
	if !r.Failed {
		t.Fatalf("Failed = false:\n%s", joined(r))
	}
	lines := engineLines(r)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "✗ .claude/settings.json: invalid JSON") {
		t.Fatalf("engine-key lines = %q, want one invalid JSON failure", lines)
	}
}

// S11: the same for both Codex tables.
func TestEngineKeysCodex(t *testing.T) {
	t.Parallel()
	proj := codexBuilt(t, "1.2.3")
	run := func(t *testing.T, proj fs.FS, _ configFiles, _ Settings) Report {
		return runCodex(t, proj, codexUser(), "1.2.3")
	}
	var cfg configFiles

	r := run(t, with(proj, map[string]string{".codex/config.toml": ""}), cfg, Settings{})
	all := []string{
		codexMissing("marketplaces.gabo-ai.source_type"),
		codexMissing("marketplaces.gabo-ai.source"),
		codexMissing(`plugins."gabo@gabo-ai".enabled`),
	}
	wantFailed(t, r, all...)
	wantEngineLines(t, r, all...)

	noPlugin := "model = \"o3\"\n\n[marketplaces.gabo-ai]\nsource_type = \"git\"\nsource = \"https://github.com/testeos-z/gabo-ai.git\"\n"
	r = run(t, with(proj, map[string]string{".codex/config.toml": noPlugin}), cfg, Settings{})
	wantFailed(t, r, codexMissing(`plugins."gabo@gabo-ai".enabled`))
	wantEngineLines(t, r, codexMissing(`plugins."gabo@gabo-ai".enabled`))

	noMarket := "[plugins.\"gabo@gabo-ai\"]\nenabled = true\n"
	r = run(t, with(proj, map[string]string{".codex/config.toml": noMarket}), cfg, Settings{})
	wantFailed(t, r, all[0], all[1])
	wantEngineLines(t, r, all[0], all[1])

	off := strings.Replace(codexKeysTOML, "enabled = true", "enabled = false", 1)
	r = run(t, with(proj, map[string]string{".codex/config.toml": off}), cfg, Settings{})
	wantPassed(t, r, PassedLine)
	wantEngineLines(t, r)
}

// S11: Pi needs a packages entry naming @gabo-ai/pi, any version.
func TestEngineKeysPi(t *testing.T) {
	t.Parallel()
	cfg := configFiles{projectFile: strings.Replace(validConfig, "coder: claude", "coder: pi", 1)}
	proj := with(project(), map[string]string{
		".claude/settings.json":                   claudeKeysJSON,
		".gabo/runtimes/gabo-runtime-pi/SKILL.md": "---\nname: gabo-runtime-pi\nmetadata:\n  agent_dir: .pi/agents\n---\n",
		".pi/agents/alpha.md":                     "---\ngabo-version: 1.2.3\n---\n",
		".pi/agents/beta.md":                      "---\ngabo-version: 1.2.3\n---\n",
	})
	for name, settings := range map[string]string{
		"missing file":     "",
		"no packages":      `{"theme": "dark"}`,
		"other packages":   `{"packages": ["npm:gentle-pi", {"source": "npm:@gabo-ai/pi-extra"}]}`,
		"another scope pi": `{"packages": ["npm:@other/pi@0.2.6"]}`,
	} {
		r := run(t, with(proj, map[string]string{".pi/settings.json": settings}), cfg, Settings{})
		if !r.Failed {
			t.Fatalf("%s: Failed = false:\n%s", name, joined(r))
		}
		if got := engineLines(r); !slices.Equal(got, []string{piMissing}) {
			t.Fatalf("%s: engine-key lines = %q, want %q", name, got, piMissing)
		}
	}
	for name, settings := range map[string]string{
		"this release":   piKeysJSON,
		"older release":  `{"packages": ["npm:gentle-pi", "npm:@gabo-ai/pi@0.2.5"]}`,
		"no version":     `{"packages": ["npm:@gabo-ai/pi"]}`,
		"object entry":   `{"packages": [{"source": "npm:@gabo-ai/pi@0.3.0", "extensions": []}]}`,
		"source version": `{"packages": ["npm:@gabo-ai/pi@0.2.6-dev"]}`,
	} {
		r := run(t, with(proj, map[string]string{".pi/settings.json": settings}), cfg, Settings{Version: "dev"})
		if got := engineLines(r); len(got) != 0 {
			t.Fatalf("%s: engine-key lines = %q, want none:\n%s", name, got, joined(r))
		}
	}
}

// S11: with GABO_UPGRADE_PR=1 the rule is skipped with the other
// spec-dependent checks.
func TestEngineKeysSkippedOnUpgradePR(t *testing.T) {
	t.Parallel()
	r := run(t, with(project(), map[string]string{".claude/settings.json": ""}), configFiles{projectFile: validConfig}, Settings{UpgradePR: true})
	wantPassed(t, r, UpgradePRLine)
	wantEngineLines(t, r)
}

// S11: engines not in the set are not checked; OpenCode, in the full tier,
// has no key of S2, S3 or S7.
func TestEngineKeysOnlyForTheSet(t *testing.T) {
	t.Parallel()
	// claude only: no .codex/config.toml and no .pi/settings.json.
	r := run(t, with(project(), map[string]string{".claude/settings.json": claudeKeysJSON}), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	wantEngineLines(t, r)
}

// recordingFS records every path read.
type recordingFS struct {
	fstest.MapFS
	read *[]string
}

func (f recordingFS) ReadFile(name string) ([]byte, error) {
	*f.read = append(*f.read, name)
	return f.MapFS.ReadFile(name)
}

// S11 (amended 2026-11-04): a flexible-tier engine gets no engine-key check
// and no engine file read. Unit test of the rule: the config schema rejects
// acme until intent 0005.
func TestEngineKeysFlexibleTierIsNotChecked(t *testing.T) {
	t.Parallel()
	var read []string
	proj := recordingFS{MapFS: fstest.MapFS{".claude/settings.json": file(claudeKeysJSON)}, read: &read}
	lines, err := engineKeyFailures(proj, []string{"claude", "acme"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 0 {
		t.Fatalf("lines = %q, want none", lines)
	}
	if !slices.Equal(read, []string{".claude/settings.json"}) {
		t.Fatalf("read %q, want only .claude/settings.json", read)
	}

	read = nil
	lines, err = engineKeyFailures(recordingFS{MapFS: fstest.MapFS{}, read: &read}, []string{"acme"})
	if err != nil || len(lines) != 0 || len(read) != 0 {
		t.Fatalf("acme alone: lines = %q, read = %q, err = %v; want nothing", lines, read, err)
	}
}

// gabo check writes nothing: the project tree is byte-equal after a run
// with failing engine keys.
func TestEngineKeysWriteNothing(t *testing.T) {
	t.Parallel()
	proj := with(project(), map[string]string{".claude/settings.json": `{"permissions": {}}`})
	before := maps.Clone(proj)
	snapshot := map[string]string{}
	for k, v := range before {
		snapshot[k] = string(v.Data)
	}
	r := run(t, proj, configFiles{projectFile: validConfig}, Settings{})
	wantFailed(t, r, claudeMissingMarket, claudeMissingPlugin)
	if len(proj) != len(snapshot) {
		t.Fatalf("project has %d files after the run, want %d", len(proj), len(snapshot))
	}
	for k, v := range proj {
		if snapshot[k] != string(v.Data) {
			t.Fatalf("%s changed", k)
		}
	}
}

// gabo-ai's own .claude/settings.json (T12, a copy in testdata) passes.
func TestEngineKeysGaboAiSettingsPass(t *testing.T) {
	t.Parallel()
	r := run(t, with(project(), map[string]string{".claude/settings.json": engineFixture(t, "gabo-ai-claude-settings.json")}), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
}
