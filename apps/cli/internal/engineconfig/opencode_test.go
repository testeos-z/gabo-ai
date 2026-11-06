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
			target: ".opencode/opencode.json", out: "{\n  \"experimental\": {\n    \"subagent_depth\": 3\n  },\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote .opencode/opencode.json: experimental.subagent_depth unset → 3; added default_agent", opencodeInPlace},
			steps: []string{opencodeBuildStep},
		},
		{
			name: "jsonc with comments gains only the key", files: map[string][]byte{"opencode.jsonc": []byte(jsoncWithComments)},
			target: "opencode.jsonc",
			out:    strings.TrimSuffix(jsoncWithComments, "}\n") + "  \"experimental\": {\n    \"subagent_depth\": 3\n  },\n  \"default_agent\": \"gabo-assistant\",\n}\n",
			lines:  []string{"wrote opencode.jsonc: experimental.subagent_depth unset → 3; added default_agent", opencodeMissingNow},
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

// Intent 0010 S3: with opencode in runtimes.* the rule makes
// experimental.subagent_depth at least 3 in the S2 file: unset or lower
// becomes 3, reported with the old and new value; 3 or more is left alone;
// anything it cannot merge without loss writes nothing and gets one ✗.
func TestOpencodeRuleSubagentDepth(t *testing.T) {
	stage := []byte("x")
	cases := []struct {
		name      string
		assistant string
		files     map[string][]byte
		target    string // the only file written, "" for none
		out       string
		lines     []string
	}{
		{
			name: "no config creates .opencode/opencode.json", files: map[string][]byte{},
			target: ".opencode/opencode.json", out: "{\n  \"experimental\": {\n    \"subagent_depth\": 3\n  }\n}\n",
			lines: []string{"wrote .opencode/opencode.json: experimental.subagent_depth unset → 3"},
		},
		{
			name: "no config with the opencode assistant holds both keys", assistant: Opencode, files: map[string][]byte{},
			target: ".opencode/opencode.json", out: "{\n  \"experimental\": {\n    \"subagent_depth\": 3\n  },\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote .opencode/opencode.json: experimental.subagent_depth unset → 3; added default_agent"},
		},
		{
			name: "absent key is merged and comments are kept", files: map[string][]byte{"opencode.jsonc": []byte(jsoncWithComments)},
			target: "opencode.jsonc",
			out:    strings.TrimSuffix(jsoncWithComments, "}\n") + "  \"experimental\": {\n    \"subagent_depth\": 3\n  },\n}\n",
			lines:  []string{"wrote opencode.jsonc: experimental.subagent_depth unset → 3"},
		},
		{
			name: "experimental without the key gains only the key", files: map[string][]byte{"opencode.json": []byte("{\n  \"experimental\": {\n    \"other\": true // mine\n  }\n}\n")},
			target: "opencode.json", out: "{\n  \"experimental\": {\n    \"other\": true, // mine\n    \"subagent_depth\": 3\n  }\n}\n",
			lines: []string{"wrote opencode.json: experimental.subagent_depth unset → 3"},
		},
		{
			name: "1 becomes 3 and comments are kept", files: map[string][]byte{"opencode.jsonc": []byte("{\n  // depth\n  \"experimental\": {\"subagent_depth\": 1 /* low */},\n}\n")},
			target: "opencode.jsonc", out: "{\n  // depth\n  \"experimental\": {\"subagent_depth\": 3 /* low */},\n}\n",
			lines: []string{"wrote opencode.jsonc: experimental.subagent_depth 1 → 3"},
		},
		{
			name: "a lower whole number in float form becomes 3", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 2.0}}`)},
			target: "opencode.json", out: `{"experimental": {"subagent_depth": 3}}`,
			lines: []string{"wrote opencode.json: experimental.subagent_depth 2.0 → 3"},
		},
		{
			name: "raised together with default_agent", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte("{\n  \"experimental\": {\"subagent_depth\": 1}\n}\n")},
			target: "opencode.json", out: "{\n  \"experimental\": {\"subagent_depth\": 3},\n  \"default_agent\": \"gabo-assistant\"\n}\n",
			lines: []string{"wrote opencode.json: experimental.subagent_depth 1 → 3; added default_agent"},
		},
		{
			name: "raised next to the user's default_agent", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "mine", "experimental": {"subagent_depth": 0}}`)},
			target: "opencode.json", out: `{"default_agent": "mine", "experimental": {"subagent_depth": 3}}`,
			lines: []string{"wrote opencode.json: experimental.subagent_depth 0 → 3", `kept opencode.json default_agent: "mine" (your setting; gabo leaves it)`},
		},
		{
			name: "raised in the file the gabo default_agent leaves", assistant: Claude, files: map[string][]byte{"opencode.json": []byte("{\n  \"model\": \"x\",\n  \"default_agent\": \"gabo-assistant\"\n}\n")},
			target: "opencode.json", out: "{\n  \"model\": \"x\",\n  \"experimental\": {\n    \"subagent_depth\": 3\n  }\n}\n",
			lines: []string{"wrote opencode.json: experimental.subagent_depth unset → 3", `removed opencode.json default_agent: "gabo-assistant" (runtimes.assistant is claude)`},
		},
		{
			name: "5 leaves the file byte-identical", files: map[string][]byte{"opencode.json": []byte("{\"experimental\": {\"subagent_depth\": 5}} // x\n")},
		},
		{
			name: "3.0 is a whole number of 3", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 3.0}}`)},
		},
		{
			name: "3 with gabo's default_agent is unchanged", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte(`{"default_agent": "gabo-assistant", "experimental": {"subagent_depth": 3}}`)},
			lines: []string{"unchanged opencode.json"},
		},
		{
			name: "only the first existing file is read", files: map[string][]byte{"opencode.json": []byte(depthOK), ".opencode/opencode.json": []byte("{}")},
		},
		{
			name: "experimental not an object writes nothing", assistant: Opencode, files: map[string][]byte{"opencode.json": []byte(`{"experimental": true}`)},
			lines: []string{"✗ opencode.json: experimental is not a JSON object; fix it and run gabo init again"},
		},
		{
			name: "a string value writes nothing", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": "3"}}`)},
			lines: []string{`✗ opencode.json: experimental.subagent_depth is "3", not a whole number; fix it and run gabo init again`},
		},
		{
			name: "a fraction writes nothing", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 2.5}}`)},
			lines: []string{"✗ opencode.json: experimental.subagent_depth is 2.5, not a whole number; fix it and run gabo init again"},
		},
		{
			name: "null writes nothing", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": null}}`)},
			lines: []string{"✗ opencode.json: experimental.subagent_depth is null, not a whole number; fix it and run gabo init again"},
		},
		{
			name: "experimental twice writes nothing", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {}, "experimental": {"subagent_depth": 1}}`)},
			lines: []string{"✗ opencode.json: experimental appears more than once; fix it and run gabo init again"},
		},
		{
			name: "the key twice writes nothing", files: map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 1, "subagent_depth": 4}}`)},
			lines: []string{"✗ opencode.json: experimental.subagent_depth appears more than once; fix it and run gabo init again"},
		},
		{
			name: "an unparsable file writes nothing", files: map[string][]byte{"opencode.json": []byte(`{"experimental": `)},
			lines: []string{"✗ opencode.json: invalid JSONC (unexpected end of JSON input); fix it and run gabo init again"},
		},
		{
			name: "a failure also stops the default_agent removal", assistant: Claude, files: map[string][]byte{
				"opencode.json":           []byte(`{"experimental": [3]}`),
				".opencode/opencode.json": []byte(`{"default_agent": "gabo-assistant"}`),
			},
			lines: []string{"✗ opencode.json: experimental is not a JSON object; fix it and run gabo init again"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := maps.Clone(c.files)
			files[OpencodeStageGate] = stage
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

// Intent 0010 S3, S8: through Integrate, the raise is reported in the
// opencode section and is not a step.
func TestOpencodeDepthThroughIntegrate(t *testing.T) {
	files := map[string][]byte{"opencode.json": []byte(`{"experimental": {"subagent_depth": 1}}`), OpencodeStageGate: nil}
	report, writes := Integrate(Set("claude", "opencode"), Input{Files: files, Version: "0.2.6"})
	if string(writes["opencode.json"]) != `{"experimental": {"subagent_depth": 3}}` {
		t.Errorf("writes = %q", writes)
	}
	out := report.Render()
	if !strings.Contains(out, "  opencode:\n    wrote opencode.json: experimental.subagent_depth 1 → 3\n") || strings.Contains(out, "you still need to:\n  - opencode") {
		t.Errorf("Render() =\n%s", out)
	}
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
