package engineconfig

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

// piTail is what every pi section holds after its file line: the refusal
// statement and the refusal text (S7).
var piTail = []string{"gabo refuses to build on Pi without a subagent package:", SubagentRefusal}

// piSteps are the pi items of `you still need to:` (S9).
var piSteps = []string{
	"trust the project in Pi so it installs and loads @gabo-ai/pi from .pi/settings.json",
	"install a subagent package (see the refusal above)",
}

func golden(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "pi", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPiRule(t *testing.T) {
	cases := []struct {
		name    string
		in      string // testdata/pi input; "" means no file
		version string
		want    string // testdata/pi golden output; "" means no write
		line    string
		steps   []string
	}{
		{name: "missing file", version: "0.2.6", want: "missing.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "go install version with a v prefix pins the plain version", version: "v0.2.6", want: "missing.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "other keys and entries kept in order, entry appended", in: "merge.json", version: "0.2.6", want: "merge.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "missing packages key added", in: "nopackages.json", version: "0.2.6", want: "nopackages.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "empty packages array", in: "emptypackages.json", version: "0.2.6", want: "emptypackages.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "empty object", in: "emptyobject.json", version: "0.2.6", want: "emptyobject.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "one-line file with an object entry", in: "oneline.json", version: "0.2.6", want: "oneline.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "other packages of the scope are not gabo", in: "lookalike.json", version: "0.2.6", want: "lookalike.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "escapes and brackets inside strings", in: "escapes.json", version: "0.2.6", want: "escapes.want.json",
			line: "wrote .pi/settings.json: added packages entry npm:@gabo-ai/pi@0.2.6"},
		{name: "escaped source naming the package", in: "kept-escaped.json", version: "0.2.6",
			line: "kept .pi/settings.json packages entry npm:@gabo-ai/pi by the user's choice (this gabo is 0.2.6)"},
		{name: "entry pinned to another version", in: "kept-string.json", version: "0.2.6",
			line: "kept .pi/settings.json packages entry npm:@gabo-ai/pi@0.2.5 by the user's choice (this gabo is 0.2.6)"},
		{name: "object entry pinned to another version", in: "kept-object.json", version: "0.2.6",
			line: "kept .pi/settings.json packages entry npm:@gabo-ai/pi@0.2.5 by the user's choice (this gabo is 0.2.6)"},
		{name: "entry present", in: "present.json", version: "0.2.6",
			line: "unchanged .pi/settings.json: packages entry npm:@gabo-ai/pi@0.2.6 present"},
		{name: "invalid JSON", in: "invalid.json", version: "0.2.6",
			line: "✗ .pi/settings.json: invalid JSON (unexpected end of JSON input); fix it and run gabo init again"},
		{name: "not an object", in: "notobject.json", version: "0.2.6",
			line: "✗ .pi/settings.json: not a JSON object; fix it and run gabo init again"},
		{name: "packages not an array", in: "packagesnotarray.json", version: "0.2.6",
			line: "✗ .pi/settings.json: packages is not an array; fix it and run gabo init again"},
		{name: "source build, no file", version: "dev",
			line:  "pi: development build dev; no published @gabo-ai/pi to pin; nothing written",
			steps: []string{`add "npm:@gabo-ai/pi@<version>" to .pi/settings.json packages by hand, with <version> a released gabo version`}},
		{name: "source build leaves a file alone", in: "merge.json", version: "0.2.6-3-gabc1234",
			line:  "pi: development build 0.2.6-3-gabc1234; no published @gabo-ai/pi to pin; nothing written",
			steps: []string{`add "npm:@gabo-ai/pi@<version>" to .pi/settings.json packages by hand, with <version> a released gabo version`}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := map[string][]byte{}
			var before []byte
			if c.in != "" {
				before = golden(t, c.in)
				files[".pi/settings.json"] = slices.Clone(before)
			}
			res := rules[Pi](Input{Files: files, Version: c.version})

			if c.want == "" {
				if len(res.Writes) != 0 {
					t.Errorf("writes = %q, want none", res.Writes)
				}
			} else {
				want := golden(t, c.want)
				if len(res.Writes) != 1 || string(res.Writes[".pi/settings.json"]) != string(want) {
					t.Errorf("writes = %q, want only .pi/settings.json =\n%s", res.Writes, want)
				}
			}
			if c.in != "" && string(files[".pi/settings.json"]) != string(before) {
				t.Error("the input bytes were modified")
			}
			if wantLines := append([]string{c.line}, piTail...); !slices.Equal(res.Section.Lines, wantLines) {
				t.Errorf("lines =\n%q\nwant\n%q", res.Section.Lines, wantLines)
			}
			if wantSteps := append(slices.Clone(c.steps), piSteps...); !slices.Equal(res.Section.Steps, wantSteps) {
				t.Errorf("steps =\n%q\nwant\n%q", res.Section.Steps, wantSteps)
			}
		})
	}
}

func TestIntegrateRunsThePiRule(t *testing.T) {
	report, writes := Integrate(Set("pi"), Input{Version: "0.2.6"})
	if string(writes[".pi/settings.json"]) != string(golden(t, "missing.want.json")) || len(writes) != 1 {
		t.Errorf("writes = %q", writes)
	}
	if len(report.Sections) != 1 || report.Sections[0].Engine != Pi || report.Failed() {
		t.Errorf("report = %+v", report)
	}
}

// The pi section holds the refusal text byte-equal to the quoted block of
// the embedded catalog's gabo-runtime-pi Prerequisite step 4 (S7).
func TestPiSectionHoldsTheRuntimePiRefusal(t *testing.T) {
	data, err := fs.ReadFile(skills.Catalog(), "gabo-runtime-pi/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	_, prereq, ok := strings.Cut(string(data), "\n## Prerequisite\n")
	if !ok {
		t.Fatal("gabo-runtime-pi/SKILL.md has no ## Prerequisite section")
	}
	prereq, _, _ = strings.Cut(prereq, "\n## ")
	_, step4, ok := strings.Cut(prereq, "\n4. ")
	if !ok {
		t.Fatal("Prerequisite has no step 4")
	}
	var quoted []string
	for line := range strings.Lines(step4) {
		if q, ok := strings.CutPrefix(strings.TrimLeft(line, " "), "> "); ok {
			quoted = append(quoted, strings.TrimSuffix(q, "\n"))
		} else if len(quoted) > 0 {
			break
		}
	}
	block := strings.Join(quoted, "\n")
	if block == "" {
		t.Fatal("Prerequisite step 4 has no quoted block")
	}
	for _, version := range []string{"0.2.6", "dev"} {
		res := rules[Pi](Input{Version: version})
		if !slices.Contains(res.Section.Lines, block) {
			t.Errorf("version %s: pi section lines %q do not hold the quoted block %q", version, res.Section.Lines, block)
		}
	}
}

// S1, D1: with a subagent package present the Pi section keeps the trust
// step and drops the refusal line, the refusal text and the install step.
func TestPiRuleOmitsTheRefusalWhenAPackageIsPresent(t *testing.T) {
	res := piRule(Input{Version: "0.2.6", PiPackage: true})
	for _, l := range res.Section.Lines {
		if l == piTail[0] || l == SubagentRefusal || strings.Contains(l, "subagent package") {
			t.Errorf("refusal line with a package present: %q", l)
		}
	}
	if !slices.Equal(res.Section.Steps, piSteps[:1]) {
		t.Errorf("steps = %q, want only the trust step", res.Section.Steps)
	}
	if string(res.Writes[PiSettings]) != string(golden(t, "missing.want.json")) {
		t.Errorf("the package check changed the written file: %q", res.Writes[PiSettings])
	}
}

// S2: without a package the refusal stays exactly as today.
func TestPiRuleKeepsTheRefusalWithoutAPackage(t *testing.T) {
	res := piRule(Input{Version: "0.2.6"})
	if !slices.Equal(res.Section.Lines[len(res.Section.Lines)-2:], piTail) || !slices.Equal(res.Section.Steps, piSteps) {
		t.Errorf("section = %+v, want the refusal tail and both steps", res.Section)
	}
}

// S1, PD4: a line of `pi list` names a subagent package when its source is
// npm:gentle-pi (with or without a version) or a git: source of gentle-pi;
// npm:@gabo-ai/pi and other packages never count.
func TestPiListNamesASubagentPackage(t *testing.T) {
	cases := []struct {
		list string
		want bool
	}{
		{"npm:gentle-pi\n", true},
		{"npm:gentle-pi@1.2.3\n", true},
		{"  npm:gentle-pi  \n", true},
		{"git:github.com/someone/gentle-pi\n", true},
		{"git:https://example.com/gentle-pi.git@main\n", true},
		{"npm:@gabo-ai/pi\n", false},
		{"npm:@gabo-ai/pi@0.2.6\n", false},
		{"npm:gentle-pi-extra\n", false},
		{"git:github.com/someone/other\n", false},
		{"", false},
	}
	for _, c := range cases {
		if got := PiListNamesPackage([]byte(c.list)); got != c.want {
			t.Errorf("PiListNamesPackage(%q) = %v, want %v", c.list, got, c.want)
		}
	}
}
