package engineconfig

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

const (
	opencodeInPlace    = "stage-gate plugin in place (built by gabo-runtime-opencode)"
	opencodeMissing    = "no .opencode/plugins/gabo-stage-gate.js yet; nothing written"
	opencodeMissingNow = "no .opencode/plugins/gabo-stage-gate.js yet"
	wantOpencodeStep   = "ask your agent to build the agents with gabo-runtime-opencode (it installs .opencode/plugins/gabo-stage-gate.js)"
	opencodeBuildStep  = "ask your agent to build the agents (the main session needs .opencode/agents/gabo-assistant.md)"
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

// Intent 0010 S8: the OpenCode step names only the stage-gate plugin.
func TestOpencodeStepNamesOnlyTheStageGate(t *testing.T) {
	if strings.Contains(opencodeStep, "subagent_depth") || !strings.Contains(opencodeStep, OpencodeStageGate) {
		t.Errorf("opencodeStep = %q", opencodeStep)
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

const jsoncWithComments = "{\n" +
	"  // OpenCode config\n" +
	"  \"$schema\": \"https://opencode.ai/config.json\",\n" +
	"  /* the model */\n" +
	"  \"model\": \"anthropic/claude\", // pinned\n" +
	"  \"agent\": {\n" +
	"    \"build\": {\"mode\": \"primary\",},\n" +
	"  },\n" +
	"}\n"

// Intent 0010 S2: with OpenCode as the assistant the rule merges
// default_agent into the first existing config, else a new
// .opencode/opencode.json; a second run is byte-identical; the user's own
// value is kept; a file that does not parse writes nothing.
func TestOpencodeRuleDefaultAgent(t *testing.T) {
	stage := map[string][]byte{OpencodeStageGate: []byte("x")}
	cases := []struct {
		name   string
		files  map[string][]byte
		built  bool
		target string // the file written, "" for none
		out    string
		lines  []string
		steps  []string
	}{
		{
			name: "no config creates .opencode/opencode.json", files: stage,
			target: ".opencode/opencode.json", out: "{\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote .opencode/opencode.json: added default_agent", opencodeInPlace},
			steps: []string{opencodeBuildStep},
		},
		{
			name: "jsonc with comments gains only the key", files: map[string][]byte{"opencode.jsonc": []byte(jsoncWithComments)},
			target: "opencode.jsonc",
			out:    strings.TrimSuffix(jsoncWithComments, "}\n") + "  \"default_agent\": \"gabo-assistant\",\n}\n",
			lines:  []string{"wrote opencode.jsonc: added default_agent", opencodeMissingNow},
			steps:  []string{wantOpencodeStep, opencodeBuildStep},
		},
		{
			name: "built agent has no build step", files: map[string][]byte{"opencode.json": []byte("{\n  \"model\": \"x\"\n}\n"), OpencodeStageGate: nil}, built: true,
			target: "opencode.json", out: "{\n  \"model\": \"x\",\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote opencode.json: added default_agent", opencodeInPlace},
		},
		{
			name: "first existing file wins", files: map[string][]byte{
				".opencode/opencode.jsonc": []byte("{}\n"),
				".opencode/opencode.json":  []byte("{\"model\": \"x\"}\n"),
				OpencodeStageGate:          nil,
			},
			target: ".opencode/opencode.json", out: "{\"model\": \"x\", \"default_agent\": \"gabo-assistant\"}\n",
			lines: []string{"wrote .opencode/opencode.json: added default_agent", opencodeInPlace},
			steps: []string{opencodeBuildStep},
		},
		{
			name: "already set is unchanged", files: map[string][]byte{"opencode.json": []byte("{\"default_agent\": \"gabo-assistant\"}\n"), OpencodeStageGate: nil},
			lines: []string{"unchanged opencode.json", opencodeInPlace},
			steps: []string{opencodeBuildStep},
		},
		{
			name: "the user's own value is kept", files: map[string][]byte{"opencode.json": []byte("{\"default_agent\": \"gentle-orchestrator\"}\n"), OpencodeStageGate: nil},
			lines: []string{`kept opencode.json default_agent: "gentle-orchestrator" (your setting; gabo leaves it)`, opencodeInPlace},
		},
		{
			name: "invalid JSONC writes nothing", files: map[string][]byte{"opencode.json": []byte("{\"default_agent\": "), OpencodeStageGate: nil},
			lines: []string{"✗ opencode.json: invalid JSONC (unexpected end of JSON input); fix it and run gabo init again", opencodeInPlace},
		},
		{
			name: "top level not an object writes nothing", files: map[string][]byte{"opencode.jsonc": []byte("[] // x\n"), OpencodeStageGate: nil},
			lines: []string{"✗ opencode.jsonc: the top level is not a JSON object; fix it and run gabo init again", opencodeInPlace},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := maps.Clone(c.files)
			if c.built {
				files[OpencodeAgentFile] = nil
			}
			res := opencodeRule(Input{Files: files, Version: "0.2.6", Assistant: Opencode})
			if !slices.Equal(res.Section.Lines, c.lines) {
				t.Errorf("lines =\n%q\nwant\n%q", res.Section.Lines, c.lines)
			}
			if !slices.Equal(res.Section.Steps, c.steps) {
				t.Errorf("steps =\n%q\nwant\n%q", res.Section.Steps, c.steps)
			}
			if c.target == "" {
				if len(res.Writes) != 0 {
					t.Fatalf("writes = %q, want none", res.Writes)
				}
				return
			}
			if got, ok := res.Writes[c.target]; !ok || len(res.Writes) != 1 || string(got) != c.out {
				t.Fatalf("writes = %q, want only %s =\n%s", res.Writes, c.target, c.out)
			}
			files[c.target] = res.Writes[c.target]
			again := opencodeRule(Input{Files: files, Version: "0.2.6", Assistant: Opencode})
			if len(again.Writes) != 0 || again.Section.Lines[0] != "unchanged "+c.target {
				t.Errorf("second run writes %q lines %q, want none and unchanged", again.Writes, again.Section.Lines)
			}
		})
	}
}

// Intent 0010 S4, D2: another assistant engine removes default_agent from
// each of the four files that holds exactly the gabo value, only that
// member, whether or not opencode is still in runtimes.*.
func TestOpencodeDefaultAgentRemoval(t *testing.T) {
	gabo := "{\n  \"model\": \"x\", // pinned\n  \"default_agent\": \"gabo-assistant\"\n}\n"
	clean := "{\n  \"model\": \"x\" // pinned\n}\n"
	files := map[string][]byte{}
	for _, p := range OpencodeConfigs {
		files[p] = []byte(gabo)
	}
	files["opencode.jsonc"] = []byte(`{"default_agent": "gentle-orchestrator"}`)
	var removed []string
	for _, p := range OpencodeConfigs {
		if p != "opencode.jsonc" {
			removed = append(removed, "removed "+p+` default_agent: "gabo-assistant" (runtimes.assistant is claude)`)
		}
	}
	check := func(t *testing.T, writes map[string][]byte, lines []string) {
		t.Helper()
		if len(lines) < len(removed) || !slices.Equal(lines[:len(removed)], removed) {
			t.Errorf("lines = %q, want %q first", lines, removed)
		}
		if len(writes) != 3 {
			t.Fatalf("writes = %q, want three", writes)
		}
		for p, b := range writes {
			if p == "opencode.jsonc" || string(b) != clean {
				t.Errorf("%s =\n%s", p, b)
			}
		}
	}
	t.Run("opencode in runtimes", func(t *testing.T) {
		res := opencodeRule(Input{Files: maps.Clone(files), Assistant: Claude})
		check(t, res.Writes, res.Section.Lines)
		if !slices.Equal(res.Section.Lines[len(removed):], []string{opencodeMissingNow}) || !slices.Equal(res.Section.Steps, []string{wantOpencodeStep}) {
			t.Errorf("section = %+v", res.Section)
		}
	})
	t.Run("opencode not in runtimes", func(t *testing.T) {
		report, writes := Remove(Set("claude"), Input{Files: maps.Clone(files), Assistant: Claude})
		if len(report.Sections) != 1 || report.Sections[0].Engine != Opencode || len(report.Sections[0].Steps) != 0 {
			t.Fatalf("sections = %+v", report.Sections)
		}
		check(t, writes, report.Sections[0].Lines)
		if report.Failed() {
			t.Error("Failed() = true")
		}
	})
	t.Run("unparsable file outside the set is skipped", func(t *testing.T) {
		in := Input{Files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant"`)}, Assistant: Claude}
		if report, writes := Remove(Set("claude"), in); len(writes) != 0 || len(report.Sections) != 0 {
			t.Errorf("writes %q sections %+v", writes, report.Sections)
		}
	})
	t.Run("unparsable file in the set fails", func(t *testing.T) {
		in := Input{Files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant"`), ".opencode/opencode.json": []byte(gabo)}, Assistant: Claude}
		res := opencodeRule(in)
		if len(res.Writes) != 0 || len(failuresOf(res.Section.Lines)) != 1 {
			t.Errorf("writes %q lines %q, want none and one ✗", res.Writes, res.Section.Lines)
		}
	})
	t.Run("no assistant leaves it alone", func(t *testing.T) {
		if res := opencodeRule(Input{Files: maps.Clone(files)}); len(res.Writes) != 0 {
			t.Errorf("writes %q", res.Writes)
		}
		if report, writes := Remove(Set("claude"), Input{Files: maps.Clone(files)}); len(writes) != 0 || len(report.Sections) != 0 {
			t.Errorf("writes %q sections %+v", writes, report.Sections)
		}
	})
}

func failuresOf(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(l, FailurePrefix) {
			out = append(out, l)
		}
	}
	return out
}
