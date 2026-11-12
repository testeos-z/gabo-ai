package engineconfig

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

const wantWroteLine = "wrote .pi/gentle-ai/runtime-guardrails.json: autonomousMode true (mode auto; gentle-pi will not ask before git push; the file is project config, commit it or set GENTLE_PI_AUTONOMOUS_MODE=1 for the session instead)"

// S5 examples of the guardrails rule, one case per example. The file bytes
// are the input; want is the new file (nil: no write) and the report lines.
func TestGuardrailsRuleGoldenCases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		mode    string
		version string
		files   map[string][]byte
		want    []byte
		lines   []string
	}{
		{
			name:  "absent file holds exactly the one key",
			mode:  "auto",
			want:  []byte("{\"autonomousMode\": true}\n"),
			lines: []string{wantWroteLine},
		},
		{
			name:  "other key is kept with its bytes",
			mode:  "auto",
			files: map[string][]byte{GuardrailsFile: []byte("{\"other\": 1}\n")},
			want:  []byte("{\"other\": 1, \"autonomousMode\": true}\n"),
			lines: []string{wantWroteLine},
		},
		{
			name:  "false is kept as the owner left it",
			mode:  "auto",
			files: map[string][]byte{GuardrailsFile: []byte("{\"autonomousMode\": false}\n")},
			lines: []string{"kept .pi/gentle-ai/runtime-guardrails.json: autonomousMode false by the user's choice"},
		},
		{
			name:  "manual mode writes nothing and prints nothing",
			mode:  "manual",
			lines: nil,
		},
		{
			name:  "second run changes no byte and says already set",
			mode:  "auto",
			files: map[string][]byte{GuardrailsFile: []byte("{\"autonomousMode\": true}\n")},
			lines: []string{"unchanged .pi/gentle-ai/runtime-guardrails.json: autonomousMode true already set"},
		},
		{
			name:    "development build writes the file too",
			mode:    "auto",
			version: "0.0.0-dev",
			want:    []byte("{\"autonomousMode\": true}\n"),
			lines:   []string{wantWroteLine},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res := Guardrails(Input{Files: c.files, Mode: c.mode, Version: c.version})
			got, wrote := res.Writes[GuardrailsFile]
			if c.want == nil {
				if len(res.Writes) != 0 {
					t.Fatalf("writes = %v, want none", res.Writes)
				}
			} else if !wrote || !bytes.Equal(got, c.want) {
				t.Fatalf("file = %q, want %q", got, c.want)
			}
			if !slices.Equal(res.Section.Lines, c.lines) {
				t.Fatalf("lines = %q, want %q", res.Section.Lines, c.lines)
			}
		})
	}
}

// An existing file that is not valid JSON, or is not an object, is left
// unchanged and gets one failure line in the .pi/settings.json style (S5).
func TestGuardrailsRuleRefusesBadFile(t *testing.T) {
	t.Parallel()
	for _, data := range []string{"{\"autonomousMode\": ", "[1]\n"} {
		res := Guardrails(Input{Files: map[string][]byte{GuardrailsFile: []byte(data)}, Mode: "auto"})
		if len(res.Writes) != 0 {
			t.Fatalf("%q: writes = %v, want none", data, res.Writes)
		}
		if len(res.Section.Lines) != 1 {
			t.Fatalf("%q: lines = %q, want one failure line", data, res.Section.Lines)
		}
		line := res.Section.Lines[0]
		prefix := FailurePrefix + " .pi/gentle-ai/runtime-guardrails.json: "
		if !strings.HasPrefix(line, prefix+"invalid JSON (") && !strings.HasPrefix(line, prefix+"not a JSON object") {
			t.Fatalf("%q: line = %q", data, line)
		}
		if !strings.HasSuffix(line, "; fix it and run gabo init again") {
			t.Fatalf("%q: line lacks the fix step: %q", data, line)
		}
	}
}

// An existing autonomousMode that is neither true nor false (a string or
// null) is left unchanged and gets one failure line that says the value must
// be true or false (intent 0023 S5, owner decision on #617).
func TestGuardrailsRuleRefusesNonBoolean(t *testing.T) {
	t.Parallel()
	const want = FailurePrefix + " .pi/gentle-ai/runtime-guardrails.json: autonomousMode must be true or false; fix it and run gabo init again"
	for _, data := range []string{"{\"autonomousMode\": \"true\"}\n", "{\"autonomousMode\": null}\n"} {
		res := Guardrails(Input{Files: map[string][]byte{GuardrailsFile: []byte(data)}, Mode: "auto"})
		if len(res.Writes) != 0 {
			t.Fatalf("%q: writes = %v, want none", data, res.Writes)
		}
		if !slices.Equal(res.Section.Lines, []string{want}) {
			t.Fatalf("%q: lines = %q, want %q", data, res.Section.Lines, want)
		}
	}
}
