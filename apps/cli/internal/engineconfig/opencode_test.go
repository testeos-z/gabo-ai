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
		{"plugin present", map[string][]byte{".opencode/plugins/gabo-stage-gate.js": []byte("export default {}\n"), "opencode.json": []byte(depthOK)}, []string{opencodeInPlace}, nil},
		{"plugin present and empty", map[string][]byte{".opencode/plugins/gabo-stage-gate.js": nil, "opencode.json": []byte(depthOK)}, []string{opencodeInPlace}, nil},
		{"plugin missing", map[string][]byte{"opencode.json": []byte(depthOK)}, []string{opencodeMissing}, []string{wantOpencodeStep}},
		{"other opencode files only", map[string][]byte{"opencode.json": []byte(depthOK), ".opencode/plugins/other.js": []byte("x")}, []string{opencodeMissing}, []string{wantOpencodeStep}},
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
		{"present", map[string][]byte{OpencodeStageGate: []byte("x"), "opencode.json": []byte(depthOK)}, "  opencode:\n    " + opencodeInPlace + "\n", false},
		{"missing", map[string][]byte{"opencode.json": []byte(depthOK)}, "  opencode:\n    " + opencodeMissing + "\n", true},
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
			name: "built agent has no build step", files: map[string][]byte{"opencode.json": []byte("{\n  \"model\": \"x\",\n  \"experimental\": {\"subagent_depth\": 3}\n}\n"), OpencodeStageGate: nil}, built: true,
			target: "opencode.json", out: "{\n  \"model\": \"x\",\n  \"experimental\": {\"subagent_depth\": 3},\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote opencode.json: added default_agent", opencodeInPlace},
		},
		{
			name: "first existing file wins", files: map[string][]byte{
				".opencode/opencode.jsonc": []byte("{}\n"),
				".opencode/opencode.json":  []byte("{\"model\": \"x\", \"experimental\": {\"subagent_depth\": 3}}\n"),
				OpencodeStageGate:          nil,
			},
			target: ".opencode/opencode.json", out: "{\"model\": \"x\", \"experimental\": {\"subagent_depth\": 3}, \"default_agent\": \"gabo-assistant\"}\n",
			lines: []string{"wrote .opencode/opencode.json: added default_agent", opencodeInPlace},
			steps: []string{opencodeBuildStep},
		},
		{
			name: "already set is unchanged", files: map[string][]byte{"opencode.json": []byte("{\"default_agent\": \"gabo-assistant\", \"experimental\": {\"subagent_depth\": 3}}\n"), OpencodeStageGate: nil},
			lines: []string{"unchanged opencode.json", opencodeInPlace},
			steps: []string{opencodeBuildStep},
		},
		{
			name: "the user's own value is kept", files: map[string][]byte{"opencode.json": []byte("{\"default_agent\": \"gentle-orchestrator\", \"experimental\": {\"subagent_depth\": 3}}\n"), OpencodeStageGate: nil},
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
			res := opencodeRule(Input{Files: files, Version: "0.2.6", Assistants: []string{Opencode}})
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
			again := opencodeRule(Input{Files: files, Version: "0.2.6", Assistants: []string{Opencode}})
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
	gabo := "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"model\": \"x\", // pinned\n  \"default_agent\": \"gabo-assistant\"\n}\n"
	clean := "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"model\": \"x\" // pinned\n}\n"
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
		res := opencodeRule(Input{Files: maps.Clone(files), Assistants: []string{Claude}})
		check(t, res.Writes, res.Section.Lines)
		if !slices.Equal(res.Section.Lines[len(removed):], []string{opencodeMissingNow}) || !slices.Equal(res.Section.Steps, []string{wantOpencodeStep}) {
			t.Errorf("section = %+v", res.Section)
		}
	})
	t.Run("opencode not in runtimes", func(t *testing.T) {
		report, writes := Remove(Set("claude"), Input{Files: maps.Clone(files), Assistants: []string{Claude}})
		if len(report.Sections) != 1 || report.Sections[0].Engine != Opencode || len(report.Sections[0].Steps) != 0 {
			t.Fatalf("sections = %+v", report.Sections)
		}
		check(t, writes, report.Sections[0].Lines)
		if report.Failed() {
			t.Error("Failed() = true")
		}
	})
	t.Run("unparsable file outside the set is skipped", func(t *testing.T) {
		in := Input{Files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant"`)}, Assistants: []string{Claude}}
		if report, writes := Remove(Set("claude"), in); len(writes) != 0 || len(report.Sections) != 0 {
			t.Errorf("writes %q sections %+v", writes, report.Sections)
		}
	})
	t.Run("unparsable file in the set fails", func(t *testing.T) {
		in := Input{Files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant"`), ".opencode/opencode.json": []byte(gabo)}, Assistants: []string{Claude}}
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

// depthOK is an OpenCode config whose experimental.subagent_depth needs no
// change, for tests about other keys.
const depthOK = "{\"experimental\": {\"subagent_depth\": 3}}\n"

// Intent 0015 S8, D6: gabo init neither writes nor requires
// experimental.subagent_depth. With a role on opencode and OpenCode a listed
// assistant engine, the config gains only default_agent; whatever the file
// holds under experimental stays byte-identical, with no line and no refusal
// for it; a second run is byte-identical.
func TestOpencodeRuleLeavesSubagentDepthAlone(t *testing.T) {
	cases := []struct {
		name      string
		assistant string
		files     map[string][]byte
		target    string // the only file written, "" for none
		out       string
		lines     []string
	}{
		{
			name: "no config creates only default_agent", assistant: Opencode, files: map[string][]byte{},
			target: ".opencode/opencode.json", out: "{\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote .opencode/opencode.json: added default_agent"},
		},
		{
			name: "no config and OpenCode not listed creates no file", assistant: Claude, files: map[string][]byte{},
		},
		{
			name: "no config and no assistant creates no file", files: map[string][]byte{},
		},
		{
			name: "a missing depth is not added next to default_agent", assistant: Opencode, files: map[string][]byte{"opencode.jsonc": []byte(jsoncWithComments)},
			target: "opencode.jsonc", out: strings.TrimSuffix(jsoncWithComments, "}\n") + "  \"default_agent\": \"gabo-assistant\",\n}\n",
			lines: []string{"wrote opencode.jsonc: added default_agent"},
		},
		{
			name: "1 is not raised", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte("{\n  \"experimental\": {\"subagent_depth\": 1}\n}\n")},
			target: "opencode.json", out: "{\n  \"experimental\": {\"subagent_depth\": 1},\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote opencode.json: added default_agent"},
		},
		{
			name: "3 stays with the gabo default_agent", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant", "experimental": {"subagent_depth": 3}}`)},
			lines: []string{"unchanged opencode.json"},
		},
		{
			name: "1 stays when OpenCode is not listed", assistant: Claude, files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 1}}`)},
		},
		{
			name: "3 stays when OpenCode is not listed", assistant: Claude, files: map[string][]byte{"opencode.json": []byte(depthOK)},
		},
		{
			name: "a string value gets no line and no refusal", assistant: Claude, files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": "3"}}`)},
		},
		{
			name: "a fraction gets no line and no refusal", assistant: Claude, files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 2.5}}`)},
		},
		{
			name: "null gets no line and no refusal", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": null}}`)},
		},
		{
			name: "a string value stays beside the gabo default_agent", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant", "experimental": {"subagent_depth": "x"}}`)},
			lines: []string{"unchanged opencode.json"},
		},
		{
			name: "experimental not an object gets no refusal", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant", "experimental": true}`)},
			lines: []string{"unchanged opencode.json"},
		},
		{
			name: "experimental twice gets no refusal", assistant: Claude, files: map[string][]byte{"opencode.json": []byte(`{"experimental": {}, "experimental": {"subagent_depth": 1}}`)},
		},
		{
			name: "removal of the gabo default_agent keeps the depth", assistant: Claude, files: map[string][]byte{"opencode.json": []byte("{\n  \"model\": \"x\",\n  \"experimental\": {\"subagent_depth\": 1},\n  \"default_agent\": \"gabo-assistant\"\n}\n")},
			target: "opencode.json", out: "{\n  \"model\": \"x\",\n  \"experimental\": {\"subagent_depth\": 1}\n}\n",
			lines: []string{`removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is claude)`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := maps.Clone(c.files)
			files[OpencodeStageGate] = []byte("x")
			files[OpencodeAgentFile] = nil
			res := opencodeRule(Input{Files: files, Version: "0.2.6", Assistants: []string{c.assistant}})
			want := append(slices.Clone(c.lines), opencodeInPlace)
			if !slices.Equal(res.Section.Lines, want) {
				t.Errorf("lines =\n%q\nwant\n%q", res.Section.Lines, want)
			}
			if len(res.Section.Steps) != 0 {
				t.Errorf("steps = %q, want none", res.Section.Steps)
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
			again := opencodeRule(Input{Files: files, Version: "0.2.6", Assistants: []string{c.assistant}})
			if len(again.Writes) != 0 {
				t.Errorf("second run writes %q, want none", again.Writes)
			}
		})
	}
}

// Intent 0015 S8, D6: through Integrate, the report of the opencode section
// has no experimental.subagent_depth line and an existing value is left as it is.
func TestOpencodeIntegrateReportsNoDepth(t *testing.T) {
	t.Run("no key, OpenCode listed", func(t *testing.T) {
		report, writes := Integrate(Set("claude", "opencode"), Input{Files: map[string][]byte{OpencodeStageGate: nil}, Version: "0.2.6", Assistants: []string{Opencode}})
		if string(writes[".opencode/opencode.json"]) != "{\n  \"default_agent\": \"gabo-assistant\"\n}\n" {
			t.Errorf("writes = %q", writes)
		}
		if out := report.Render(); strings.Contains(out, "subagent_depth") || !strings.Contains(out, "wrote .opencode/opencode.json: added default_agent") {
			t.Errorf("Render() =\n%s", out)
		}
	})
	t.Run("a value of 1 is left as it is", func(t *testing.T) {
		files := map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 1}}`), OpencodeStageGate: nil}
		report, writes := Integrate(Set("opencode"), Input{Files: files, Version: "0.2.6"})
		if len(writes) != 0 || strings.Contains(report.Render(), "subagent_depth") || report.Failed() {
			t.Errorf("writes %q report:\n%s", writes, report.Render())
		}
	})
}

// Intent 0013 S3, D7: OpenCode listed with another engine writes
// default_agent like a single opencode and never removes it; OpenCode not
// listed loses only the gabo value, also when it is outside the engine set,
// and the line names the list.
func TestOpencodeDefaultAgentAssistantList(t *testing.T) {
	gabo := "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"model\": \"x\", // pinned\n  \"default_agent\": \"gabo-assistant\"\n}\n"
	clean := "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"model\": \"x\" // pinned\n}\n"
	t.Run("opencode listed adds default_agent", func(t *testing.T) {
		files := map[string][]byte{"opencode.json": []byte(clean), OpencodeStageGate: nil, OpencodeAgentFile: nil}
		in := Input{Files: files, Version: "0.2.6", Assistants: []string{Claude, Opencode}}
		res := opencodeRule(in)
		if got := string(res.Writes["opencode.json"]); got != gabo || len(res.Writes) != 1 {
			t.Fatalf("writes = %q, want only opencode.json =\n%s", res.Writes, gabo)
		}
		if want := []string{"wrote opencode.json: added default_agent", opencodeInPlace}; !slices.Equal(res.Section.Lines, want) {
			t.Errorf("lines = %q, want %q", res.Section.Lines, want)
		}
		files["opencode.json"] = res.Writes["opencode.json"]
		if again := opencodeRule(in); len(again.Writes) != 0 {
			t.Errorf("second run writes %q, want none", again.Writes)
		}
	})
	t.Run("opencode listed keeps the gabo value", func(t *testing.T) {
		files := map[string][]byte{"opencode.json": []byte(gabo), "opencode.jsonc": []byte(gabo), OpencodeStageGate: nil}
		if res := opencodeRule(Input{Files: files, Assistants: []string{Pi, Opencode}}); len(res.Writes) != 0 {
			t.Errorf("writes = %q, want none", res.Writes)
		}
		if report, writes := Remove(Set(Pi, Opencode), Input{Files: files, Assistants: []string{Pi, Opencode}}); len(writes) != 0 || len(report.Sections) != 0 {
			t.Errorf("Remove writes %q sections %+v, want none", writes, report.Sections)
		}
	})
	t.Run("opencode not listed and outside the set is removed", func(t *testing.T) {
		in := Input{Files: map[string][]byte{"opencode.json": []byte(gabo)}, Assistants: []string{Claude, Pi}}
		report, writes := Remove(Set(Claude, Pi), in)
		if got := string(writes["opencode.json"]); got != clean || len(writes) != 1 {
			t.Fatalf("writes = %q, want only opencode.json =\n%s", writes, clean)
		}
		want := []string{`removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is claude, pi)`}
		if len(report.Sections) != 1 || report.Sections[0].Engine != Opencode || !slices.Equal(report.Sections[0].Lines, want) {
			t.Errorf("sections = %+v, want one opencode section %q", report.Sections, want)
		}
	})
	t.Run("opencode not listed and in the set is removed", func(t *testing.T) {
		res := opencodeRule(Input{Files: map[string][]byte{"opencode.json": []byte(gabo), OpencodeStageGate: nil}, Assistants: []string{Claude, Pi}})
		if got := string(res.Writes["opencode.json"]); got != clean {
			t.Fatalf("writes = %q, want opencode.json =\n%s", res.Writes, clean)
		}
		if res.Section.Lines[0] != `removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is claude, pi)` {
			t.Errorf("lines = %q", res.Section.Lines)
		}
	})
}
