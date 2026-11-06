package engineconfig

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const (
	claudeWrote     = `wrote .claude/settings.json: added extraKnownMarketplaces."gabo-ai", enabledPlugins."gabo@gabo-ai"`
	claudeUnchanged = "unchanged .claude/settings.json"
)

var wantClaudeSteps = []string{
	"trust this project folder in Claude Code so the gabo-ai marketplace and the gabo plugin apply",
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "claude", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestClaudeRule(t *testing.T) {
	cases := []struct {
		name  string
		in    string // testdata input, "" for a missing file
		write bool   // a golden <name>.golden.json holds the expected bytes
		lines []string
	}{
		{name: "missing", write: true, lines: []string{claudeWrote}},
		{name: "merge", in: "merge.in.json", write: true, lines: []string{claudeWrote}},
		{name: "empty-object", in: "empty-object.in.json", write: true, lines: []string{claudeWrote}},
		{name: "tabs", in: "tabs.in.json", write: true, lines: []string{claudeWrote}},
		{name: "compact", in: "compact.in.json", write: true, lines: []string{claudeWrote}},
		{name: "unchanged", in: "unchanged.in.json", lines: []string{claudeUnchanged}},
		{name: "kept", in: "kept.in.json", lines: []string{
			`kept .claude/settings.json enabledPlugins."gabo@gabo-ai": false (your setting; gabo leaves it)`,
		}},
		{name: "fork", in: "fork.in.json", write: true, lines: []string{
			`wrote .claude/settings.json: added enabledPlugins."gabo@gabo-ai"`,
			`kept .claude/settings.json extraKnownMarketplaces."gabo-ai": {"source":{"source":"github","repo":"me/gabo-ai-fork"}} (your setting; gabo leaves it)`,
		}},
		{name: "invalid", in: "invalid.in.json", lines: []string{
			"✗ .claude/settings.json: invalid JSON (unexpected end of JSON input); fix it and run gabo init again",
		}},
		{name: "not-object", in: "not-object.in.json", lines: []string{
			"✗ .claude/settings.json: enabledPlugins is not a JSON object; fix it and run gabo init again",
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := map[string][]byte{}
			var before []byte
			if c.in != "" {
				before = readTestdata(t, c.in)
				files[ClaudeSettings] = bytes.Clone(before)
			}
			res := claudeRule(Input{Files: files, Version: "0.2.6"})

			if !slices.Equal(res.Section.Lines, c.lines) {
				t.Errorf("lines =\n%q\nwant\n%q", res.Section.Lines, c.lines)
			}
			if !slices.Equal(res.Section.Steps, wantClaudeSteps) {
				t.Errorf("steps = %q, want %q", res.Section.Steps, wantClaudeSteps)
			}
			if c.in != "" && !bytes.Equal(files[ClaudeSettings], before) {
				t.Error("the rule mutated its input bytes")
			}
			if !c.write {
				if len(res.Writes) != 0 {
					t.Errorf("writes = %q, want none", res.Writes)
				}
				return
			}
			if len(res.Writes) != 1 {
				t.Fatalf("writes = %q, want only %s", res.Writes, ClaudeSettings)
			}
			got, ok := res.Writes[ClaudeSettings]
			if !ok {
				t.Fatalf("writes = %q, want %s", res.Writes, ClaudeSettings)
			}
			if want := readTestdata(t, c.name+".golden.json"); !bytes.Equal(got, want) {
				t.Errorf("wrote\n%s\nwant\n%s", got, want)
			}
			// Running again on the output changes nothing.
			again := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: got}})
			if len(again.Writes) != 0 {
				t.Errorf("second run writes %q, want none", again.Writes)
			}
		})
	}
}

// The merge keeps every byte the user wrote: the output starts with the
// input up to the last member and the old members stay in their order.
func TestClaudeRuleKeepsExistingBytesAndOrder(t *testing.T) {
	in := readTestdata(t, "merge.in.json")
	res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: in}})
	got := string(res.Writes[ClaudeSettings])
	keep := string(in)[:strings.Index(string(in), `"other@market": true`)+len(`"other@market": true`)]
	if !strings.HasPrefix(got, keep) {
		t.Errorf("output does not start with the input's bytes:\n%s", got)
	}
	last := -1
	for _, key := range []string{`"permissions"`, `"hooks"`, `"other@market": true`, `"gabo@gabo-ai": true`, `"extraKnownMarketplaces"`} {
		i := strings.Index(got, key)
		if i <= last {
			t.Errorf("%s at %d, want after %d", key, i, last)
		}
		last = i
	}
}

func TestClaudeIsInTheRuleTable(t *testing.T) {
	report, writes := Integrate(Set("claude"), Input{Files: map[string][]byte{}, Version: "0.2.6"})
	if len(report.Sections) != 1 || report.Sections[0].Engine != Claude {
		t.Fatalf("sections = %+v, want one claude section", report.Sections)
	}
	if !slices.Equal(report.Sections[0].Lines, []string{claudeWrote}) {
		t.Errorf("lines = %q", report.Sections[0].Lines)
	}
	var paths []string
	for p := range writes {
		paths = append(paths, p)
	}
	if !slices.Equal(paths, []string{".claude/settings.json"}) {
		t.Errorf("writes %q, want only .claude/settings.json (never settings.local.json or a user-level file)", paths)
	}
	if report.Failed() {
		t.Error("Failed() = true, want false")
	}
	out := report.Render()
	for _, st := range wantClaudeSteps {
		if !strings.Contains(out, "  - claude: "+st+"\n") {
			t.Errorf("Render() lacks step %q:\n%s", st, out)
		}
	}
}

func TestClaudeRuleFailureIsReported(t *testing.T) {
	report, writes := Integrate(Set("claude"), Input{Files: map[string][]byte{ClaudeSettings: []byte("{")}})
	if len(writes) != 0 || !report.Failed() {
		t.Errorf("writes %q, Failed() %v; want none and true", writes, report.Failed())
	}
}

// Whatever the layout, the written file parses, keeps the user's values and
// holds both keys with the S2 values.
func TestClaudeRuleOutputHoldsTheKeys(t *testing.T) {
	inputs := []string{
		`{ }`,
		`{"enabledPlugins": {}}`,
		"{\r\n  \"a\": \"x\\\"}{,\",\r\n  \"enabledPlugins\": {\"o@m\": false}\r\n}\r\n",
		`{"extraKnownMarketplaces": {"other": {"source": {"source": "directory", "path": "../m"}}}, "n": [1, {"b": [2]}], "z": null}`,
	}
	for _, in := range inputs {
		res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: []byte(in)}})
		out := res.Writes[ClaudeSettings]
		var before, after map[string]any
		if err := json.Unmarshal([]byte(in), &before); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(out, &after); err != nil {
			t.Errorf("%q: output is not JSON (%v):\n%s", in, err, out)
			continue
		}
		for k, v := range before {
			if k == "enabledPlugins" || k == "extraKnownMarketplaces" {
				for ck, cv := range v.(map[string]any) {
					if !reflect.DeepEqual(after[k].(map[string]any)[ck], cv) {
						t.Errorf("%q: %s.%s changed", in, k, ck)
					}
				}
				continue
			}
			if !reflect.DeepEqual(after[k], v) {
				t.Errorf("%q: %s changed to %v", in, k, after[k])
			}
		}
		if after["enabledPlugins"].(map[string]any)["gabo@gabo-ai"] != true {
			t.Errorf("%q: enabledPlugins.\"gabo@gabo-ai\" not true:\n%s", in, out)
		}
		want := map[string]any{"source": map[string]any{"source": "github", "repo": "testeos-z/gabo-ai"}}
		if !reflect.DeepEqual(after["extraKnownMarketplaces"].(map[string]any)["gabo-ai"], want) {
			t.Errorf("%q: extraKnownMarketplaces.\"gabo-ai\" wrong:\n%s", in, out)
		}
	}
}

func TestClaudeRuleRefusesNonObjects(t *testing.T) {
	for in, reason := range map[string]string{
		`[]`:                              "the top level is not a JSON object",
		`{"enabledPlugins": null}`:        "enabledPlugins is not a JSON object",
		`{"extraKnownMarketplaces": "x"}`: "extraKnownMarketplaces is not a JSON object",
		``:                                "invalid JSON (unexpected end of JSON input)",
	} {
		res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: []byte(in)}})
		want := []string{"✗ .claude/settings.json: " + reason + "; fix it and run gabo init again"}
		if len(res.Writes) != 0 || !slices.Equal(res.Section.Lines, want) {
			t.Errorf("%q: writes %q lines %q, want none and %q", in, res.Writes, res.Section.Lines, want)
		}
	}
}

// integrationOnly is a settings file holding only the integration keys.
const integrationOnly = `{
  "extraKnownMarketplaces": {
    "gabo-ai": {
      "source": {
        "source": "github",
        "repo": "testeos-z/gabo-ai"
      }
    }
  },
  "enabledPlugins": {
    "gabo@gabo-ai": true
  }
}
`

const buildStep = "ask your agent to build the agents (the main session needs .claude/agents/gabo-assistant.md)"

// S1, S4, S8: the top-level agent key follows runtimes.assistant. Claude
// writes "gabo-assistant", another engine removes exactly that value, any
// other value is the user's; no assistant leaves the key alone.
func TestClaudeRuleAgentKey(t *testing.T) {
	withAgent := strings.TrimSuffix(integrationOnly, "}\n") + "  \"agent\": \"gabo-assistant\"\n}\n"
	withAgent = strings.Replace(withAgent, "    \"gabo@gabo-ai\": true\n  }\n", "    \"gabo@gabo-ai\": true\n  },\n", 1)
	cases := []struct {
		name      string
		assistant string
		in        *string // nil for a missing file
		built     bool    // .claude/agents/gabo-assistant.md exists
		out       string  // expected written bytes, "" for no write
		lines     []string
		steps     []string
	}{
		{
			name: "claude adds agent and keeps every byte", assistant: "claude", in: ptr(integrationOnly),
			out:   withAgent,
			lines: []string{"wrote .claude/settings.json: added agent"},
			steps: append(slices.Clone(wantClaudeSteps), buildStep),
		},
		{
			name: "claude with agent built has no build step", assistant: "claude", in: ptr(integrationOnly), built: true,
			out:   withAgent,
			lines: []string{"wrote .claude/settings.json: added agent"},
			steps: wantClaudeSteps,
		},
		{
			name: "claude already set is unchanged", assistant: "claude", in: ptr(withAgent),
			lines: []string{claudeUnchanged},
			steps: append(slices.Clone(wantClaudeSteps), buildStep),
		},
		{
			name: "claude keeps the user's agent", assistant: "claude", in: ptr(`{"agent": "my-agent", "enabledPlugins": {"gabo@gabo-ai": true}, "extraKnownMarketplaces": {"gabo-ai": {"source": {"source": "github", "repo": "testeos-z/gabo-ai"}}}}`),
			lines: []string{`kept .claude/settings.json agent: "my-agent" (your setting; gabo leaves it)`},
			steps: wantClaudeSteps,
		},
		{
			name: "claude new file holds the integration keys and agent", assistant: "claude",
			out:   withAgent,
			lines: []string{`wrote .claude/settings.json: added extraKnownMarketplaces."gabo-ai", enabledPlugins."gabo@gabo-ai", agent`},
			steps: append(slices.Clone(wantClaudeSteps), buildStep),
		},
		{
			name: "codex removes the gabo value and its comma", assistant: "codex", in: ptr(withAgent),
			out:   integrationOnly,
			lines: []string{`removed .claude/settings.json agent: "gabo-assistant" (runtimes.assistant is codex)`},
			steps: wantClaudeSteps,
		},
		{
			name: "codex removes a first member and the comma after it", assistant: "codex",
			in:    ptr("{\n  \"agent\": \"gabo-assistant\",\n" + strings.TrimPrefix(integrationOnly, "{\n")),
			out:   integrationOnly,
			lines: []string{`removed .claude/settings.json agent: "gabo-assistant" (runtimes.assistant is codex)`},
			steps: wantClaudeSteps,
		},
		{
			name: "codex keeps the user's agent", assistant: "codex",
			in:    ptr(strings.Replace(withAgent, `"gabo-assistant"`, `"my-agent"`, 1)),
			lines: []string{claudeUnchanged},
			steps: wantClaudeSteps,
		},
		{
			name: "codex new file has no agent", assistant: "codex",
			out:   integrationOnly,
			lines: []string{claudeWrote},
			steps: wantClaudeSteps,
		},
		{
			name: "no assistant leaves the gabo value alone", in: ptr(withAgent),
			lines: []string{claudeUnchanged},
			steps: wantClaudeSteps,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files := map[string][]byte{}
			if c.in != nil {
				files[ClaudeSettings] = []byte(*c.in)
			}
			if c.built {
				files[ClaudeAgentFile] = []byte("---\nname: gabo-assistant\n---\n")
			}
			res := claudeRule(Input{Files: files, Version: "0.2.6", Assistant: c.assistant})
			if !slices.Equal(res.Section.Lines, c.lines) {
				t.Errorf("lines =\n%q\nwant\n%q", res.Section.Lines, c.lines)
			}
			if !slices.Equal(res.Section.Steps, c.steps) {
				t.Errorf("steps =\n%q\nwant\n%q", res.Section.Steps, c.steps)
			}
			got, wrote := res.Writes[ClaudeSettings]
			if c.out == "" {
				if len(res.Writes) != 0 {
					t.Fatalf("writes = %q, want none", res.Writes)
				}
				return
			}
			if !wrote || len(res.Writes) != 1 {
				t.Fatalf("writes = %q, want only %s", res.Writes, ClaudeSettings)
			}
			if string(got) != c.out {
				t.Fatalf("wrote\n%s\nwant\n%s", got, c.out)
			}
			// A second run on the output is byte-identical: nothing written.
			files[ClaudeSettings] = got
			if again := claudeRule(Input{Files: files, Assistant: c.assistant}); len(again.Writes) != 0 {
				t.Errorf("second run writes %q, want none", again.Writes)
			}
		})
	}
}

// S1: a file gabo cannot read writes nothing and gets one ✗ line, also with
// the Claude Code assistant.
func TestClaudeRuleAgentRefusesUnreadableFiles(t *testing.T) {
	for in, reason := range map[string]string{
		`{"agent": `:       "invalid JSON (unexpected end of JSON input)",
		`["agent"]`:        "the top level is not a JSON object",
		`"gabo-assistant"`: "the top level is not a JSON object",
	} {
		for _, assistant := range []string{"claude", "codex"} {
			res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: []byte(in)}, Assistant: assistant})
			want := []string{"✗ .claude/settings.json: " + reason + "; fix it and run gabo init again"}
			if len(res.Writes) != 0 || !slices.Equal(res.Section.Lines, want) {
				t.Errorf("%s %q: writes %q lines %q, want none and %q", assistant, in, res.Writes, res.Section.Lines, want)
			}
		}
	}
}

// S1: the added member keeps every earlier byte, in a compact file too.
func TestClaudeRuleAgentKeepsEarlierBytes(t *testing.T) {
	in := `{"permissions": {"allow": []}, "enabledPlugins": {"gabo@gabo-ai": true}, "extraKnownMarketplaces": {"gabo-ai": {"source": {"source": "github", "repo": "testeos-z/gabo-ai"}}}}`
	res := claudeRule(Input{Files: map[string][]byte{ClaudeSettings: []byte(in)}, Assistant: "claude"})
	want := strings.TrimSuffix(in, "}") + `, "agent": "gabo-assistant"}`
	if got := string(res.Writes[ClaudeSettings]); got != want {
		t.Errorf("wrote\n%s\nwant\n%s", got, want)
	}
}

func ptr(s string) *string { return &s }
