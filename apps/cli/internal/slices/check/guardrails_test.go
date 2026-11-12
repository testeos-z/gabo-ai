package check

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"testing/fstest"
)

// guardFile is the Pi guardrails file gabo init writes (intent 0023 S5).
const guardFile = ".pi/gentle-ai/runtime-guardrails.json"

// guardCase is one project of the guardrails golden file: its gabo.yaml and
// its files.
type guardCase struct {
	name string
	cfg  string
	proj fstest.MapFS
}

// guardCases are the S6 examples of gabo check, one per case. A Pi project in
// mode auto warns while the guardrails file is missing or autonomousMode is
// not true, and fails on invalid JSON; every other case prints nothing about
// the file.
func guardCases(t *testing.T) []guardCase {
	t.Helper()
	auto := piConfig + "mode: auto\n"
	manual := piConfig + "mode: manual\n"
	autoClaude := validConfig + "mode: auto\n"
	piWith := func(content string) fstest.MapFS {
		return with(piBuilt(t, "1.2.3"), map[string]string{guardFile: content})
	}
	return []guardCase{
		{name: "auto pi no file", cfg: auto, proj: piBuilt(t, "1.2.3")},
		{name: "auto pi autonomousMode true", cfg: auto, proj: piWith("{\"autonomousMode\": true}\n")},
		{name: "auto pi invalid JSON", cfg: auto, proj: piWith("{\"autonomousMode\": \n")},
		{name: "auto pi autonomousMode false", cfg: auto, proj: piWith("{\"autonomousMode\": false}\n")},
		{name: "auto pi not an object", cfg: auto, proj: piWith("[1]\n")},
		{name: "manual pi no file", cfg: manual, proj: piBuilt(t, "1.2.3")},
		{name: "manual pi autonomousMode false", cfg: manual, proj: piWith("{\"autonomousMode\": false}\n")},
		{name: "auto without pi no file", cfg: autoClaude, proj: project()},
	}
}

// renderGuard is the golden form of one run: the exit code, the error text,
// then stdout, as golden_test.go renders the other gabo check goldens.
func renderGuard(t *testing.T, c guardCase) string {
	t.Helper()
	out, err := runCommandConfig(t, c.proj, c.cfg)
	code := 0
	var ee *ExitError
	if errors.As(err, &ee) {
		code = ee.Code
	} else if err != nil {
		t.Fatalf("%s: err = %v, want an ExitError", c.name, err)
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return fmt.Sprintf("exit %d\nerror: %q\n--- stdout\n%s", code, msg, out)
}

// guardSections splits the golden file into its cases by their "=== <name>"
// header lines.
func guardSections(text string) map[string]string {
	out := map[string]string{}
	name := ""
	var b strings.Builder
	flush := func() {
		if name != "" {
			out[name] = b.String()
		}
		b.Reset()
	}
	for _, line := range strings.SplitAfter(text, "\n") {
		if rest, ok := strings.CutPrefix(line, "=== "); ok {
			flush()
			name = strings.TrimSuffix(rest, "\n")
			continue
		}
		b.WriteString(line)
	}
	flush()
	return out
}

// TestGuardrailsCheckGolden pins gabo check on the S6 projects against
// testdata/guardrails/guardrails.golden: each case is compared on its own, so
// every case that differs is reported. With -update it rewrites the file.
func TestGuardrailsCheckGolden(t *testing.T) {
	t.Parallel()
	cases := guardCases(t)
	got := map[string]string{}
	for _, c := range cases {
		got[c.name] = renderGuard(t, c)
	}
	file := path.Join("testdata", "guardrails", "guardrails.golden")
	if *update {
		var b strings.Builder
		for _, c := range cases {
			fmt.Fprintf(&b, "=== %s\n%s", c.name, got[c.name])
		}
		if err := os.MkdirAll(path.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read golden (run with -update on the base to write it): %v", err)
	}
	want := guardSections(string(raw))
	for _, c := range cases {
		if got[c.name] != want[c.name] {
			t.Errorf("%s: output changed:\n got: %q\nwant: %q", c.name, got[c.name], want[c.name])
		}
	}
}
