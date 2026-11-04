package engineconfig

import (
	"slices"
	"strings"
	"testing"
)

const (
	opencodeInPlace  = "stage-gate plugin in place (built by gabo-runtime-opencode)"
	opencodeMissing  = "no .opencode/plugins/gabo-stage-gate.js yet; nothing written"
	wantOpencodeStep = "ask your agent to build the agents with gabo-runtime-opencode (it installs .opencode/plugins/gabo-stage-gate.js and the project-level experimental.subagent_depth key)"
)

func TestOpencodeRule(t *testing.T) {
	cases := []struct {
		name  string
		files map[string][]byte
		lines []string
		steps []string
	}{
		{"plugin present", map[string][]byte{".opencode/plugins/gabo-stage-gate.js": []byte("export default {}\n")}, []string{opencodeInPlace}, nil},
		{"plugin present and empty", map[string][]byte{".opencode/plugins/gabo-stage-gate.js": nil}, []string{opencodeInPlace}, nil},
		{"plugin missing", map[string][]byte{}, []string{opencodeMissing}, []string{wantOpencodeStep}},
		{"other opencode files only", map[string][]byte{"opencode.json": []byte("{}\n"), ".opencode/plugins/other.js": []byte("x")}, []string{opencodeMissing}, []string{wantOpencodeStep}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res := opencodeRule(Input{Files: c.files, Version: "0.2.6"})
			if len(res.Writes) != 0 {
				t.Errorf("writes %q, want none", res.Writes)
			}
			if !slices.Equal(res.Section.Lines, c.lines) {
				t.Errorf("lines = %q, want %q", res.Section.Lines, c.lines)
			}
			if !slices.Equal(res.Section.Steps, c.steps) {
				t.Errorf("steps = %q, want %q", res.Section.Steps, c.steps)
			}
		})
	}
}

func TestOpencodeIsInTheRuleTable(t *testing.T) {
	for _, c := range []struct {
		name     string
		files    map[string][]byte
		section  string
		withStep bool
	}{
		{"present", map[string][]byte{OpencodeStageGate: []byte("x")}, "  opencode:\n    " + opencodeInPlace + "\n", false},
		{"missing", map[string][]byte{}, "  opencode:\n    " + opencodeMissing + "\n", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			report, writes := Integrate(Set("opencode"), Input{Files: c.files, Version: "0.2.6"})
			if len(writes) != 0 {
				t.Errorf("writes %q, want none", writes)
			}
			if report.Failed() {
				t.Error("Failed() = true, want false")
			}
			out := report.Render()
			if !strings.Contains(out, c.section) {
				t.Errorf("Render() lacks section %q:\n%s", c.section, out)
			}
			hasStep := strings.Contains(out, "you still need to:\n  - opencode: "+wantOpencodeStep+"\n")
			if hasStep != c.withStep {
				t.Errorf("step listed = %v, want %v:\n%s", hasStep, c.withStep, out)
			}
		})
	}
}
