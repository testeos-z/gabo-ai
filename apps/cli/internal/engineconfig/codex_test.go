package engineconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const codexAll = `marketplaces.gabo-ai.source_type, marketplaces.gabo-ai.source, plugins."gabo@gabo-ai".enabled`

func codexGolden(t *testing.T, name, file string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "codex", name, file))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func codexIn(t *testing.T, name string) Input {
	t.Helper()
	if name == "missing" {
		return Input{Files: map[string][]byte{}, Version: "0.2.6"}
	}
	return Input{Files: map[string][]byte{CodexConfigPath: codexGolden(t, name, "in.toml")}, Version: "0.2.6"}
}

func TestCodexWritesTablesAppendOnly(t *testing.T) {
	cases := []struct {
		name string
		line string
	}{
		{"missing", "wrote .codex/config.toml: added " + codexAll},
		{"append", "wrote .codex/config.toml: added " + codexAll},
		{"nonewline", "wrote .codex/config.toml: added " + codexAll},
		{"partial", `wrote .codex/config.toml: added marketplaces.gabo-ai.source, plugins."gabo@gabo-ai".enabled`},
		{"partial-crlf", `wrote .codex/config.toml: added marketplaces.gabo-ai.source, plugins."gabo@gabo-ai".enabled`},
		{"header-eof", "wrote .codex/config.toml: added " + codexAll},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := codexIn(t, c.name)
			res := codexRule(in)
			got, ok := res.Writes[CodexConfigPath]
			if !ok || len(res.Writes) != 1 {
				t.Fatalf("Writes = %q, want only %s", res.Writes, CodexConfigPath)
			}
			if want := codexGolden(t, c.name, "want.toml"); !bytes.Equal(got, want) {
				t.Errorf("new file =\n%s\nwant\n%s", got, want)
			}
			if old, ok := in.Files[CodexConfigPath]; ok && !strings.HasPrefix(c.name, "partial") && c.name != "header-eof" && !bytes.HasPrefix(got, old) {
				t.Errorf("existing bytes are not a prefix of the new file")
			}
			if !slices.Equal(res.Section.Lines, []string{c.line}) {
				t.Errorf("Lines = %q, want %q", res.Section.Lines, c.line)
			}
		})
	}
}

func TestCodexBothTablesPresentIsUnchanged(t *testing.T) {
	for _, name := range []string{"present", "dotted"} {
		t.Run(name, func(t *testing.T) {
			res := codexRule(codexIn(t, name))
			if len(res.Writes) != 0 {
				t.Errorf("Writes = %q, want none", res.Writes)
			}
			if want := []string{".codex/config.toml unchanged"}; !slices.Equal(res.Section.Lines, want) {
				t.Errorf("Lines = %q, want %q", res.Section.Lines, want)
			}
		})
	}
}

func TestCodexKeepsAnotherValue(t *testing.T) {
	res := codexRule(codexIn(t, "disabled"))
	if len(res.Writes) != 0 {
		t.Errorf("Writes = %q, want none", res.Writes)
	}
	want := []string{`kept plugins."gabo@gabo-ai".enabled = false in .codex/config.toml (your choice)`}
	if !slices.Equal(res.Section.Lines, want) {
		t.Errorf("Lines = %q, want %q", res.Section.Lines, want)
	}
	if (Report{Sections: []Section{res.Section}}).Failed() {
		t.Error("a kept key is not a failure")
	}
}

func TestCodexKeepsAnotherSourceAndAddsTheRest(t *testing.T) {
	in := Input{Files: map[string][]byte{CodexConfigPath: []byte("[marketplaces.gabo-ai]\nsource_type = \"local\"\nsource = \"./gabo\"\n")}}
	res := codexRule(in)
	want := []string{
		`kept marketplaces.gabo-ai.source_type = "local" in .codex/config.toml (your choice)`,
		`kept marketplaces.gabo-ai.source = "./gabo" in .codex/config.toml (your choice)`,
		`wrote .codex/config.toml: added plugins."gabo@gabo-ai".enabled`,
	}
	if !slices.Equal(res.Section.Lines, want) {
		t.Errorf("Lines = %q, want %q", res.Section.Lines, want)
	}
	if got := string(res.Writes[CodexConfigPath]); !strings.HasPrefix(got, string(in.Files[CodexConfigPath])) || !strings.HasSuffix(got, "\n[plugins.\"gabo@gabo-ai\"]\nenabled = true\n") {
		t.Errorf("new file =\n%s", got)
	}
}

func TestCodexInvalidTOMLIsOneFailureLine(t *testing.T) {
	res := codexRule(codexIn(t, "invalid"))
	if len(res.Writes) != 0 {
		t.Errorf("Writes = %q, want none", res.Writes)
	}
	if len(res.Section.Lines) != 1 || !strings.HasPrefix(res.Section.Lines[0], "✗ .codex/config.toml: invalid TOML") || !strings.HasSuffix(res.Section.Lines[0], "; fix it and run gabo init again") {
		t.Errorf("Lines = %q, want one ✗ .codex/config.toml: invalid TOML line", res.Section.Lines)
	}
	if !(Report{Sections: []Section{res.Section}}).Failed() {
		t.Error("Failed() = false, want true")
	}
}

// A missing key whose table cannot take an appended or inserted line (here
// a table defined by dotted keys) is refused, never written broken.
func TestCodexRefusesAKeyItCannotAddSafely(t *testing.T) {
	in := Input{Files: map[string][]byte{CodexConfigPath: []byte("marketplaces.gabo-ai.source_type = \"git\"\n")}}
	res := codexRule(in)
	if len(res.Writes) != 0 {
		t.Errorf("Writes = %q, want none", res.Writes)
	}
	want := []string{`✗ .codex/config.toml: cannot add marketplaces.gabo-ai.source, plugins."gabo@gabo-ai".enabled without rewriting your text; add them by hand and run gabo init again`}
	if !slices.Equal(res.Section.Lines, want) {
		t.Errorf("Lines = %q, want %q", res.Section.Lines, want)
	}
}

func TestCodexStepsWithTheFallbackFlag(t *testing.T) {
	steps := []string{
		"trust the project in Codex so it loads .codex/config.toml",
		"trust both gabo hooks in /hooks",
		"if Codex does not list the gabo plugin after trust, run: codex plugin marketplace add testeos-z/gabo-ai",
		"if Codex does not list the gabo plugin after trust, run: codex plugin add gabo@gabo-ai",
		"while testeos-z/gabo-ai is private, set up git credentials that can read it",
	}
	if !codexFallback {
		t.Fatal("the fallback flag is set until the S12 smoke confirms Codex installs from the project tables (D4)")
	}
	for _, name := range []string{"missing", "present", "disabled", "invalid"} {
		if got := codexRule(codexIn(t, name)).Section.Steps; !slices.Equal(got, steps) {
			t.Errorf("%s: Steps = %q, want %q", name, got, steps)
		}
	}

	codexFallback = false
	t.Cleanup(func() { codexFallback = true })
	got := codexRule(codexIn(t, "missing")).Section.Steps
	if want := []string{steps[0], steps[1], steps[4]}; !slices.Equal(got, want) {
		t.Errorf("without the flag Steps = %q, want %q", got, want)
	}
}

func TestIntegrateRunsTheCodexRule(t *testing.T) {
	report, writes := Integrate(Set("codex"), Input{Files: map[string][]byte{}})
	if len(report.Sections) != 1 || report.Sections[0].Engine != Codex {
		t.Fatalf("Sections = %+v, want one codex section", report.Sections)
	}
	if want := codexGolden(t, "missing", "want.toml"); !bytes.Equal(writes[CodexConfigPath], want) || len(writes) != 1 {
		t.Errorf("writes = %q, want only %s", writes, CodexConfigPath)
	}
}

// A kept value of any TOML type never breaks the rule (S4), and the kept
// line names it as TOML.
func TestCodexKeepsArrayAndTableValues(t *testing.T) {
	cases := []struct {
		name, in string
		lines    []string
		suffix   string
	}{
		{
			"array", "[marketplaces.gabo-ai]\nsource_type = \"git\"\nsource = [\"x\"]\n",
			[]string{`kept marketplaces.gabo-ai.source = ['x'] in .codex/config.toml (your choice)`, `wrote .codex/config.toml: added plugins."gabo@gabo-ai".enabled`},
			"\n\n[plugins.\"gabo@gabo-ai\"]\nenabled = true\n",
		},
		{
			"table", "plugins.\"gabo@gabo-ai\".enabled.x = 1\n",
			[]string{`kept plugins."gabo@gabo-ai".enabled = {x = 1} in .codex/config.toml (your choice)`, `wrote .codex/config.toml: added marketplaces.gabo-ai.source_type, marketplaces.gabo-ai.source`},
			"\n\n[marketplaces.gabo-ai]\nsource_type = \"git\"\nsource = \"https://github.com/testeos-z/gabo-ai.git\"\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := codexRule(Input{Files: map[string][]byte{CodexConfigPath: []byte(c.in)}})
			if !slices.Equal(res.Section.Lines, c.lines) {
				t.Errorf("Lines = %q, want %q", res.Section.Lines, c.lines)
			}
			got, wrote := res.Writes[CodexConfigPath]
			if c.suffix == "" {
				if wrote {
					t.Errorf("wrote %q, want no write", got)
				}
				return
			}
			if string(got) != c.in+strings.TrimPrefix(c.suffix, "\n") {
				t.Errorf("new file = %q, want %q", got, c.in+strings.TrimPrefix(c.suffix, "\n"))
			}
		})
	}
}
