package initcmd

import (
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/engineconfig"
)

const guardFile = root + "/" + engineconfig.GuardrailsFile

// guardrailsStep runs the engine step of set in mode (gabo.yaml mode) for
// version, as withEngines does.
func guardrailsStep(s *memStore, mode, version string, set ...string) engineconfig.Report {
	return NewUseCase(s, catalog, Settings{ProjectDir: root, Version: version}).engineStep(set, nil, mode, version)
}

// guardLines returns the report lines that name the guardrails file, one per
// line with a final newline, the form of the goldens.
func guardLines(t *testing.T, r engineconfig.Report) string {
	t.Helper()
	var out []string
	for _, s := range r.Sections {
		for _, l := range s.Lines {
			if strings.Contains(l, engineconfig.GuardrailsFile) {
				out = append(out, l)
			}
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

// S5: mode auto, pi listed and no file: the one key is written and the line
// is printed (golden).
func TestGuardrailsWrittenGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := guardrailsStep(s, "auto", "0.2.6", "pi")
	golden(t, "guardrails-written.golden", guardLines(t, r))
	if got := string(s.Files[guardFile]); got != "{\"autonomousMode\": true}\n" {
		t.Fatalf("file = %q", got)
	}
}

// S5: an existing key of another name is kept with its bytes.
func TestGuardrailsKeepsOtherKeys(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{guardFile: "{\"other\": 1}\n"})
	r := guardrailsStep(s, "auto", "0.2.6", "pi")
	if got := string(s.Files[guardFile]); got != "{\"other\": 1, \"autonomousMode\": true}\n" {
		t.Fatalf("file = %q", got)
	}
	if !strings.Contains(guardLines(t, r), "wrote "+engineconfig.GuardrailsFile+": autonomousMode true") {
		t.Fatalf("no wrote line:\n%s", r.Render())
	}
}

// S5: an existing false is kept and the report says kept (golden).
func TestGuardrailsKeptFalseGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{guardFile: "{\"autonomousMode\": false}\n"})
	r := guardrailsStep(s, "auto", "0.2.6", "pi")
	golden(t, "guardrails-kept.golden", guardLines(t, r))
	if slices.Contains(s.Writes, guardFile) {
		t.Fatalf("kept file was written: %v", s.Writes)
	}
	if got := string(s.Files[guardFile]); got != "{\"autonomousMode\": false}\n" {
		t.Fatalf("file = %q", got)
	}
}

// S5: a second run changes no byte and says already set (golden).
func TestGuardrailsSecondRunGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	guardrailsStep(s, "auto", "0.2.6", "pi")
	before := string(s.Files[guardFile])
	s.Writes = nil
	r := guardrailsStep(s, "auto", "0.2.6", "pi")
	golden(t, "guardrails-already-set.golden", guardLines(t, r))
	if len(s.Writes) != 0 || string(s.Files[guardFile]) != before {
		t.Fatalf("second run wrote %v", s.Writes)
	}
}

// S5: mode manual writes no file and prints no line.
func TestGuardrailsManualWritesNothing(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := guardrailsStep(s, "manual", "0.2.6", "pi")
	if guardLines(t, r) != "" || slices.Contains(s.Writes, guardFile) {
		t.Fatalf("manual mode touched the guardrails file: writes %v\n%s", s.Writes, r.Render())
	}
}

// S5: pi not listed writes nothing, even in mode auto.
func TestGuardrailsWithoutPiWritesNothing(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := guardrailsStep(s, "auto", "0.2.6", "claude")
	if guardLines(t, r) != "" || slices.Contains(s.Writes, guardFile) {
		t.Fatalf("no pi, but the guardrails file was touched: writes %v", s.Writes)
	}
}

// S5: invalid JSON leaves the file unchanged and prints one failure line
// (golden).
func TestGuardrailsInvalidJSONGolden(t *testing.T) {
	t.Parallel()
	const bad = "{\"autonomousMode\": "
	s := newStore(map[string]string{guardFile: bad})
	r := guardrailsStep(s, "auto", "0.2.6", "pi")
	golden(t, "guardrails-invalid-json.golden", guardLines(t, r))
	if slices.Contains(s.Writes, guardFile) || string(s.Files[guardFile]) != bad {
		t.Fatalf("invalid file was changed: writes %v", s.Writes)
	}
	if !r.Failed() {
		t.Fatalf("invalid JSON is not a failure:\n%s", r.Render())
	}
}

// S5: a development build writes the file too; the settings file still names
// no published package (golden).
func TestGuardrailsDevelopmentBuildGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	r := guardrailsStep(s, "auto", "0.0.0-dev", "pi")
	golden(t, "guardrails-dev-build.golden", guardLines(t, r))
	if !slices.Contains(s.Writes, guardFile) {
		t.Fatalf("dev build did not write the file: %v", s.Writes)
	}
}

// Like every engine file, the guardrails file is not written through a
// symlinked engine directory (S5, refuseEngineFile); the refusal is a ✗ line
// (golden).
func TestGuardrailsSymlinkDirRefusedGolden(t *testing.T) {
	t.Parallel()
	s := newStore(map[string]string{})
	s.Modes = map[string]fs.FileMode{root + "/.pi/gentle-ai": fs.ModeSymlink | fs.ModeDir}
	r := guardrailsStep(s, "auto", "0.0.0-dev", "pi")
	golden(t, "guardrails-symlink.golden", guardLines(t, r))
	if slices.Contains(s.Writes, guardFile) {
		t.Fatalf("symlinked directory was written through: %v", s.Writes)
	}
}

// S5 through gabo init: the mode key of gabo.yaml decides the write, and the
// report line reaches the output only when the file is written (D4).
func TestInitWritesGuardrailsFromGaboYamlMode(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		mode     string
		wantFile bool
	}{{"auto", true}, {"manual", false}} {
		t.Run(c.mode, func(t *testing.T) {
			t.Parallel()
			cfg := "mode: " + c.mode + "\nrepo:\n  name: o/r\nruntimes:\n  coder: pi\n  reviewer: claude\n  assistant: claude\n"
			s := newStore(map[string]string{config: cfg})
			out, err := run(t, s, "1.2.3")
			if err != nil {
				t.Fatal(err)
			}
			got, ok := s.Files[guardFile]
			if ok != c.wantFile {
				t.Fatalf("guardrails file present = %v, want %v", ok, c.wantFile)
			}
			hasLine := strings.Contains(joined(out), "wrote "+engineconfig.GuardrailsFile+": autonomousMode true")
			if c.wantFile && (got != "{\"autonomousMode\": true}\n" || !hasLine) {
				t.Fatalf("file = %q, line = %v\n%s", got, hasLine, joined(out))
			}
			if !c.wantFile && (hasLine || strings.Contains(joined(out), engineconfig.GuardrailsFile)) {
				t.Fatalf("manual mode printed the guardrails file:\n%s", joined(out))
			}
		})
	}
}

// S5 and the #617 decision: an existing autonomousMode that is a string or
// null is not a choice. gabo init writes nothing, prints one ✗ line, leaves
// the file bytes unchanged and exits 1 (golden per value).
func TestGuardrailsNonBooleanGolden(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, value string }{
		{"string-true", "{\"autonomousMode\": \"true\"}\n"},
		{"null", "{\"autonomousMode\": null}\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := newStore(map[string]string{guardFile: c.value})
			r := guardrailsStep(s, "auto", "0.2.6", "pi")
			golden(t, "guardrails-"+c.name+".golden", guardLines(t, r))
			if slices.Contains(s.Writes, guardFile) || string(s.Files[guardFile]) != c.value {
				t.Fatalf("non-boolean file was changed: writes %v", s.Writes)
			}
			if !r.Failed() {
				t.Fatalf("non-boolean autonomousMode is not a failure:\n%s", r.Render())
			}
		})
	}
}

// S5 through gabo init: a non-boolean autonomousMode makes the run fail with
// exit 1 and leaves the guardrails file unchanged.
func TestInitNonBooleanGuardrailsExitsOne(t *testing.T) {
	t.Parallel()
	const value = "{\"autonomousMode\": \"true\"}\n"
	cfg := "mode: auto\nrepo:\n  name: o/r\nruntimes:\n  coder: pi\n  reviewer: claude\n  assistant: claude\n"
	s := newStore(map[string]string{config: cfg, guardFile: value})
	out, err := run(t, s, "1.2.3")
	if err == nil {
		t.Fatalf("non-boolean autonomousMode exited 0:\n%s", joined(out))
	}
	if slices.Contains(s.Writes, guardFile) || string(s.Files[guardFile]) != value {
		t.Fatalf("file = %q, writes %v", s.Files[guardFile], s.Writes)
	}
}
