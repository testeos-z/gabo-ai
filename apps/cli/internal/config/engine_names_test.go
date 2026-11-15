package config

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Intent 0017 (spec S1, S3, S5, S6; plan T1): config accepts any well-formed
// engine name in runtimes.coder and runtimes.assistant, refuses a malformed
// one with the PD3 line of its role, and keeps the four reviewer engines.

// engineNameLine is the PD3 line for a malformed value of runtimes.<role>.
func engineNameLine(role, v string) string {
	return fmt.Sprintf(`runtimes.%s: must be an engine name of lower case letters, digits and "-", starting with a letter, got %q`, role, v)
}

// engineEntryLine is the PD3 line for a malformed entry of runtimes.assistant.
func engineEntryLine(v string) string {
	return fmt.Sprintf(`runtimes.assistant: entry %q must be an engine name of lower case letters, digits and "-", starting with a letter; next: replace or remove it`, v)
}

// reviewerLine is the PD3 line for a runtimes.reviewer value outside the
// four engines the reviewer scripts run (TRD D27).
func reviewerLine(v string) string {
	return fmt.Sprintf(`runtimes.reviewer: must be one of "claude", "codex", "pi", "opencode", got %q; next: set one of them (the reviewer scripts run only these engines) or ask your agent to configure the reviewer with gabo-reviewer-config`, v)
}

// readLog is a Reader that records every file it is asked for.
type readLog struct {
	files
	names []string
}

func (r *readLog) ReadFile(name string) ([]byte, error) {
	r.names = append(r.names, name)
	return r.files.ReadFile(name)
}

// loadProject loads one project layer through LoadLayers and returns the
// error and the files read.
func loadProject(doc string) (error, []string) {
	r := &readLog{files: files{"/src/p/gabo.yaml": doc}}
	_, err := LoadLayers(r, []Layer{{Name: LayerProject, Path: "/src/p/gabo.yaml"}})
	return err, r.names
}

// Spec S1, S6, D4: a well-formed engine name passes Parse, LoadLayers and the
// three schema entry points, and config reads no file besides the layers.
func TestWellFormedEngineNamesPassEveryValidation(t *testing.T) {
	t.Parallel()
	for name, block := range map[string]string{
		"coder name":      "  coder: foo\n  reviewer: claude\n  assistant: claude\n",
		"assistant name":  "  coder: claude\n  reviewer: claude\n  assistant: foo\n",
		"assistant names": "  coder: claude\n  reviewer: claude\n  assistant: [claude, foo]\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc := runtimesDoc(block)
			if _, err := Parse([]byte(doc)); err != nil {
				t.Errorf("Parse() error = %v", err)
			}
			err, read := loadProject(doc)
			if err != nil {
				t.Errorf("LoadLayers() error = %v", err)
			}
			if !slices.Equal(read, []string{"/src/p/gabo.yaml"}) {
				t.Errorf("files read = %q, want only the project layer", read)
			}
			for _, s := range []Schema{SchemaFile, SchemaLayer, SchemaProject} {
				if err := ValidateSchema(s, []byte(doc)); err != nil {
					t.Errorf("ValidateSchema(%s) error = %v", s, err)
				}
			}
		})
	}
}

// Spec S6 and PD2: a malformed name is refused with the PD3 line of its role,
// one line per bad value, in Parse and in LoadLayers (gabo check), and the
// schema refuses it too.
func TestMalformedEngineNamesAreRefusedWithTheRoleLine(t *testing.T) {
	t.Parallel()
	const rest = "  reviewer: claude\n"
	for name, tt := range map[string]struct{ block, key, want string }{
		"coder with a space":     {"  coder: \"Foo Bar\"\n" + rest + "  assistant: claude\n", "runtimes.coder", engineNameLine("coder", "Foo Bar")},
		"coder starting digit":   {"  coder: \"1abc\"\n" + rest + "  assistant: claude\n", "runtimes.coder", engineNameLine("coder", "1abc")},
		"coder with underscore":  {"  coder: foo_bar\n" + rest + "  assistant: claude\n", "runtimes.coder", engineNameLine("coder", "foo_bar")},
		"assistant scalar space": {"  coder: claude\n" + rest + "  assistant: \"foo bar\"\n", "runtimes.assistant", engineNameLine("assistant", "foo bar")},
		"assistant list entry":   {"  coder: claude\n" + rest + "  assistant: [claude, foo_bar]\n", "runtimes.assistant", engineEntryLine("foo_bar")},
		"assistant empty entry":  {"  coder: claude\n" + rest + "  assistant: [claude, \"\"]\n", "runtimes.assistant", engineEntryLine("")},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			doc := runtimesDoc(tt.block)
			_, err := Parse([]byte(doc))
			if got := linesNaming(err, tt.key); !slices.Contains(got, tt.want) {
				t.Errorf("Parse() lines naming %s = %q, want %q", tt.key, got, tt.want)
			}
			loadErr, _ := loadProject(doc)
			if loadErr == nil || strings.Count(loadErr.Error(), tt.want) != 1 {
				t.Errorf("LoadLayers() error = %v, want the line %q once", loadErr, tt.want)
			}
			if err := ValidateSchema(SchemaFile, []byte(doc)); err == nil {
				t.Errorf("ValidateSchema(SchemaFile) error = nil, want a refusal")
			}
		})
	}
}

// Spec S3, D3, PD3: runtimes.reviewer accepts only the four engines and
// session; any other value gets the reviewer line with its next step.
func TestReviewerAcceptsOnlyTheFourEnginesAndSession(t *testing.T) {
	t.Parallel()
	refused := runtimesDoc("  coder: claude\n  reviewer: bar\n  assistant: claude\n")
	_, err := Parse([]byte(refused))
	if got := linesNaming(err, "runtimes.reviewer"); !slices.Equal(got, []string{reviewerLine("bar")}) {
		t.Errorf("Parse() reviewer bar: lines = %q, want [%q]", got, reviewerLine("bar"))
	}
	if loadErr, _ := loadProject(refused); loadErr == nil || !strings.Contains(loadErr.Error(), reviewerLine("bar")) {
		t.Errorf("LoadLayers() reviewer bar: error = %v, want the line %q", loadErr, reviewerLine("bar"))
	}
	if err := ValidateSchema(SchemaFile, []byte(refused)); err == nil {
		t.Errorf("ValidateSchema(SchemaFile) reviewer bar: error = nil, want a refusal")
	}
	// A malformed reviewer name gets the same line: it is not one of the four.
	if _, err := Parse([]byte(runtimesDoc("  coder: claude\n  reviewer: \"Foo Bar\"\n  assistant: claude\n"))); err == nil || !strings.Contains(err.Error(), reviewerLine("Foo Bar")) {
		t.Errorf("reviewer Foo Bar: error = %v, want the line %q", err, reviewerLine("Foo Bar"))
	}
	for _, rv := range []string{"claude", "codex", "pi", "opencode", "session"} {
		doc := runtimesDoc("  coder: claude\n  reviewer: " + rv + "\n  assistant: claude\n")
		if _, err := Parse([]byte(doc)); err != nil {
			t.Errorf("Parse() reviewer %s: error = %v, want accepted", rv, err)
		}
		if loadErr, _ := loadProject(doc); loadErr != nil {
			t.Errorf("LoadLayers() reviewer %s: error = %v, want accepted", rv, loadErr)
		}
	}
}

// Spec S5 (D4 consequence): session keeps its meaning. It names the main
// session in coder and reviewer and is refused in assistant, with its line.
func TestSessionStaysRefusedInAssistant(t *testing.T) {
	t.Parallel()
	const want = `runtimes.assistant: "session" is valid only in runtimes.coder and runtimes.reviewer; next: set the engine of the main session`
	for name, block := range map[string]string{
		"scalar": "  coder: claude\n  reviewer: claude\n  assistant: session\n",
		"list":   "  coder: claude\n  reviewer: claude\n  assistant: [claude, session]\n",
	} {
		doc := runtimesDoc(block)
		if _, err := Parse([]byte(doc)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: Parse() error = %v, want the line %q", name, err, want)
		}
		if loadErr, _ := loadProject(doc); loadErr == nil || !strings.Contains(loadErr.Error(), want) {
			t.Errorf("%s: LoadLayers() error = %v, want the line %q", name, loadErr, want)
		}
		if err := ValidateSchema(SchemaFile, []byte(doc)); err == nil {
			t.Errorf("%s: ValidateSchema(SchemaFile) error = nil, want a refusal", name)
		}
	}
}

// Spec S5, PD2: the four named engines keep full validation in every role.
func TestNamedEnginesValidateInEveryRoleThroughEveryLayer(t *testing.T) {
	t.Parallel()
	for _, rt := range []string{"claude", "codex", "pi", "opencode"} {
		doc := runtimesDoc("  coder: " + rt + "\n  reviewer: " + rt + "\n  assistant: [" + rt + "]\n")
		if _, err := Parse([]byte(doc)); err != nil {
			t.Errorf("%s: Parse() error = %v", rt, err)
		}
		if loadErr, _ := loadProject(doc); loadErr != nil {
			t.Errorf("%s: LoadLayers() error = %v", rt, loadErr)
		}
		if err := ValidateSchema(SchemaFile, []byte(doc)); err != nil {
			t.Errorf("%s: ValidateSchema(SchemaFile) error = %v", rt, err)
		}
	}
}

// PD2: the Go expression and the schema pattern accept and refuse the same
// names, checked through Parse (Go) and the layer schema for runtimes.coder.
func TestEngineNameGoAndSchemaAgree(t *testing.T) {
	t.Parallel()
	accepted := []string{"claude", "codex", "pi", "opencode", "foo", "gabo-runtime-1", "a", "x-y-z"}
	refused := []string{"Foo Bar", "1abc", "foo_bar", "foo bar", "", "-x", "X", "foo.bar", "foo/bar"}
	schemaPattern := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	check := func(name string, want bool) {
		doc := runtimesDoc("  coder: " + strconv.Quote(name) + "\n  reviewer: claude\n  assistant: claude\n")
		_, goErr := Parse([]byte(doc))
		schemaErr := ValidateSchema(SchemaLayer, []byte(doc))
		if (goErr == nil) != want || (schemaErr == nil) != want {
			t.Errorf("%q: want accepted=%v; Go error = %v; schema error = %v", name, want, goErr, schemaErr)
		}
		if got := schemaPattern.MatchString(name); got != want {
			t.Errorf("%q: schema pattern match = %v, want %v", name, got, want)
		}
	}
	for _, n := range accepted {
		check(n, true)
	}
	for _, n := range refused {
		check(n, false)
	}
}

// PD4: the schema lists the four known engines as examples of engine and
// runtime, and has the reviewer definition.
func TestEngineSchemaDefinitions(t *testing.T) {
	t.Parallel()
	project := readSchemaJSON(t, ProjectSchemaFile)
	defs, _ := project["$defs"].(map[string]any)
	const pattern = "^[a-z][a-z0-9-]*$"
	for _, name := range []string{"engine", "runtime"} {
		def, _ := defs[name].(map[string]any)
		if def["pattern"] != pattern {
			t.Errorf("$defs/%s pattern = %v, want %s", name, def["pattern"], pattern)
		}
		examples, _ := def["examples"].([]any)
		var got []string
		for _, e := range examples {
			s, _ := e.(string)
			got = append(got, s)
		}
		if !slices.Equal(got, []string{"claude", "codex", "pi", "opencode"}) {
			t.Errorf("$defs/%s examples = %q, want the four engines", name, got)
		}
	}
	if _, ok := defs["reviewerRuntime"]; !ok {
		t.Error("$defs/reviewerRuntime is missing (PD4)")
	}
}

// A null runtimes value in one layer is not a name: the merged result judges
// it as missing, as before (intent 0017 S1, D4). Each bad list entry gets
// its own line (one line per bad value).
func TestNullAndMultipleBadEntriesKeepTheirShape(t *testing.T) {
	t.Parallel()
	err, _ := loadProject(strings.Replace(minimal, "coder: claude", "coder: null", 1))
	if err == nil || !strings.Contains(err.Error(), "✗ runtimes.coder missing") || strings.Contains(err.Error(), "engine name") {
		t.Errorf("null coder: error = %v, want the missing line and no name refusal", err)
	}
	doc := runtimesDoc("  coder: claude\n  reviewer: claude\n  assistant: [claude, \"Foo Bar\", \"1abc\"]\n")
	_, perr := Parse([]byte(doc))
	want := []string{engineEntryLine("Foo Bar"), engineEntryLine("1abc")}
	if got := linesNaming(perr, "runtimes.assistant"); !slices.Equal(got, want) {
		t.Errorf("two bad entries: lines = %q, want %q", got, want)
	}
}
