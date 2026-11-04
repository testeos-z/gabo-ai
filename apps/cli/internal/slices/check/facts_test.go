package check

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/skills"
)

const piAgent = ".pi/agents/gabo-coder.md"

// factsJSON is a gabo_facts stdin object for the Pi engine; tools nil
// gives an empty list, targets nil omits target_paths.
func factsJSON(t *testing.T, event string, tools, targets []string) string {
	t.Helper()
	if tools == nil {
		tools = []string{}
	}
	m := map[string]any{"gabo_facts": 1, "event": event, "engine": "pi", "tools": tools, "package_version": "0.5.0"}
	if targets != nil {
		m["target_paths"] = targets
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestIsFacts(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]bool{
		`{"gabo_facts":1}`:                 true,
		`{"gabo_facts":2,"event":"x"}`:     true,
		`{"tool_input":{"file_path":"a"}}`: false,
		`{"gabo_facts":1`:                  false,
		`[]`:                               false,
		``:                                 false,
		`{"tool_input":{"gabo_facts":1}}`:  false,
		`{"tool_input":{"file_path":"a"}} {"x":1}`: false,
	} {
		if got := IsFacts([]byte(in)); got != want {
			t.Errorf("IsFacts(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseFacts(t *testing.T) {
	t.Parallel()
	f, err := ParseFacts([]byte(`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":["read","subagent_run"],"package_version":"0.5.0","target_paths":["a.md","b/c.md"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if f.Event != EventToolCall || f.Engine != "pi" || strings.Join(f.Tools, ",") != "read,subagent_run" || f.PackageVersion != "0.5.0" || strings.Join(f.TargetPaths, ",") != "a.md,b/c.md" {
		t.Fatalf("ParseFacts = %+v", f)
	}
}

// Malformed JSON, an unknown schema version, an engine other than pi, an
// unknown event, a tool_call without target_paths and fields of the wrong
// type are usage errors with one line.
func TestParseFactsRejectsBadInput(t *testing.T) {
	t.Parallel()
	for _, in := range []string{
		`{"gabo_facts":1`,
		`{"gabo_facts":1,"event":"session_start","engine":"pi","tools":[]} {}`,
		`{"gabo_facts":2,"event":"session_start","engine":"pi","tools":[]}`,
		`{"gabo_facts":"1","event":"session_start","engine":"pi","tools":[]}`,
		`{"gabo_facts":1.5,"event":"session_start","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"session_start","engine":"claude","tools":[]}`,
		`{"gabo_facts":1,"event":"session_start","tools":[]}`,
		`{"gabo_facts":1,"event":"agent_end","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[],"target_paths":[]}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[],"target_paths":null}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[],"target_paths":[""]}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[],"target_paths":"a.md"}`,
		`{"gabo_facts":1,"event":"session_start","engine":"pi"}`,
		`{"gabo_facts":1,"event":"session_start","engine":"pi","tools":"subagent_run"}`,
		`{"gabo_facts":1,"event":"session_start","engine":"pi","tools":[1]}`,
		`{"gabo_facts":1,"event":"session_start","engine":"pi","tools":[],"package_version":1}`,
	} {
		_, err := ParseFacts([]byte(in))
		if !errors.Is(err, ErrPreWriteUsage) || strings.Contains(err.Error(), "\n") {
			t.Errorf("ParseFacts(%s) = %v, want one ErrPreWriteUsage line", in, err)
		}
	}
}

func TestJudgeFacts(t *testing.T) {
	t.Parallel()
	refusal := FactsResult{Notify: []Notification{{Level: LevelWarning, Text: SubagentRefusal}}}
	blocked := FactsResult{Notify: []Notification{}, Block: &Block{Reason: SubagentRefusal}}
	none := FactsResult{Notify: []Notification{}}
	pi := []string{"read", "write", "edit", "bash"}
	gentle := append([]string{"subagent_run"}, pi...)
	tests := []struct {
		name string
		f    Facts
		want FactsResult
	}{
		{name: "session_start, no subagent tool", f: Facts{Event: EventSessionStart, Engine: "pi", Tools: pi}, want: refusal},
		{name: "session_start, no tools", f: Facts{Event: EventSessionStart, Engine: "pi"}, want: refusal},
		{name: "session_start, subagent_run", f: Facts{Event: EventSessionStart, Engine: "pi", Tools: gentle}, want: none},
		{name: "tool_call on .pi/agents, no subagent tool", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{piAgent}}, want: blocked},
		{name: "tool_call on .pi/agents, absolute", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{preRoot + "/" + piAgent}}, want: blocked},
		{name: "tool_call on .pi/agents via ./ and ..", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{"./docs/../.pi/agents/x.md"}}, want: blocked},
		{name: "tool_call, one of several targets", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{"README.md", piAgent, ".pi/agents/b.md"}}, want: blocked},
		{name: "tool_call on .pi/agents, subagent_run", f: Facts{Event: EventToolCall, Engine: "pi", Tools: gentle, TargetPaths: []string{piAgent}}, want: none},
		{name: "tool_call on another path, no subagent tool", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{"README.md"}}, want: none},
		{name: "tool_call on .pi/skills, no subagent tool", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{".pi/skills/x/SKILL.md", ".pi/agentsx/a.md", "docs/.pi/agents/a.md"}}, want: none},
		{name: "tool_call outside the root, no subagent tool", f: Facts{Event: EventToolCall, Engine: "pi", Tools: pi, TargetPaths: []string{"../other/.pi/agents/a.md", "/elsewhere/.pi/agents/a.md"}}, want: none},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u := NewUseCase(intentWith(""), failingReader{}, goodUser(), Settings{Root: preRoot})
			got := u.JudgeFacts(tt.f)
			if g, w := mustJSON(t, got), mustJSON(t, tt.want); g != w {
				t.Fatalf("JudgeFacts = %s, want %s", g, w)
			}
		})
	}
}

// A write whose symlinks lead under .pi/agents/ is judged at its target.
func TestJudgeFactsThroughASymlink(t *testing.T) {
	t.Parallel()
	links := map[string]string{preRoot + "/notes.md": preRoot + "/.pi/agents/gabo-coder.md"}
	u := NewUseCase(intentWith(""), failingReader{}, goodUser(), Settings{
		Root: preRoot,
		Resolve: func(p string) (string, error) {
			if _, ok := links[p]; ok {
				return "", fs.ErrNotExist
			}
			return p, nil
		},
		Readlink: func(p string) (string, bool, error) {
			t, ok := links[p]
			return t, ok, nil
		},
	})
	got := u.JudgeFacts(Facts{Event: EventToolCall, Engine: "pi", TargetPaths: []string{"notes.md"}})
	if got.Block == nil || got.Block.Reason != SubagentRefusal {
		t.Fatalf("JudgeFacts = %s, want the refusal block", mustJSON(t, got))
	}
}

// A target whose symlinks cannot be resolved counts as under .pi/agents/
// (fail closed); the stage gate fails closed on it too, after the refusal.
// With subagent_run only the stage gate's line remains.
func TestJudgeFactsUnresolvableTargetFailsClosed(t *testing.T) {
	t.Parallel()
	u := NewUseCase(intentWith(""), failingReader{}, goodUser(), Settings{
		Root:    preRoot,
		Resolve: func(p string) (string, error) { return "", fs.ErrPermission },
	})
	gate := "✗ README.md: its symlinks cannot be resolved (permission denied); it counts as not accepted"
	got := u.JudgeFacts(Facts{Event: EventToolCall, Engine: "pi", TargetPaths: []string{"README.md"}})
	if got.Block == nil || got.Block.Reason != SubagentRefusal+"\n"+gate {
		t.Fatalf("JudgeFacts = %s, want the refusal, then the stage gate line", mustJSON(t, got))
	}
	got = u.JudgeFacts(Facts{Event: EventToolCall, Engine: "pi", Tools: SubagentTools, TargetPaths: []string{"README.md"}})
	if got.Block == nil || got.Block.Reason != gate {
		t.Fatalf("with subagent_run: JudgeFacts = %s, want only the stage gate line", mustJSON(t, got))
	}
}

// The subagent tool list is gentle-pi's subagent_run (spec D2).
func TestSubagentTools(t *testing.T) {
	t.Parallel()
	if strings.Join(SubagentTools, ",") != "subagent_run" {
		t.Fatalf("SubagentTools = %q", SubagentTools)
	}
}

// The refusal text equals the quoted block of the embedded catalog's
// gabo-runtime-pi Prerequisite step 4, byte for byte (spec D7).
func TestSubagentRefusalEqualsTheRuntimePiSkill(t *testing.T) {
	t.Parallel()
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
	if got := strings.Join(quoted, "\n"); got != SubagentRefusal {
		t.Fatalf("Prerequisite step 4 quoted block:\n%q\nwant SubagentRefusal:\n%q", got, SubagentRefusal)
	}
}

// runFacts runs `check --pre-write` with a gabo_facts stdin and returns
// stdout, stderr and the exit code.
func runFacts(t *testing.T, factory Factory, stdin string) (string, string, int) {
	t.Helper()
	cmd := NewCommand(factory)
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs([]string{"--pre-write"})
	err := cmd.ExecuteContext(context.Background())
	code := 0
	var ee *ExitError
	if errors.As(err, &ee) {
		code = ee.Code
		if ee.Err != nil {
			errOut.WriteString(ee.Err.Error())
		}
	} else if err != nil {
		t.Fatalf("err = %v", err)
	}
	return out.String(), errOut.String(), code
}

func preFactory(context.Context) (*UseCase, error) {
	return NewUseCase(intentWith(""), failingReader{}, goodUser(), Settings{Root: preRoot}), nil
}

func TestCommandFacts(t *testing.T) {
	t.Parallel()
	refusal := mustJSON(t, SubagentRefusal)
	tests := []struct {
		name, stdin, out string
		code             int
	}{
		{name: "session_start, no subagent tool", stdin: factsJSON(t, "session_start", []string{"read", "write"}, nil),
			out: `{"notify":[{"level":"warning","text":` + refusal + `}],"block":null}` + "\n"},
		{name: "session_start, subagent_run", stdin: factsJSON(t, "session_start", []string{"read", "subagent_run"}, nil),
			out: `{"notify":[],"block":null}` + "\n"},
		{name: "tool_call on .pi/agents, no subagent tool", stdin: factsJSON(t, "tool_call", []string{"write"}, []string{piAgent}),
			out: `{"notify":[],"block":{"reason":` + refusal + `}}` + "\n", code: ExitFailed},
		{name: "tool_call on .pi/agents, subagent_run", stdin: factsJSON(t, "tool_call", []string{"write", "subagent_run"}, []string{piAgent}),
			out: `{"notify":[],"block":null}` + "\n"},
		{name: "tool_call on another path", stdin: factsJSON(t, "tool_call", []string{"write"}, []string{"README.md"}),
			out: `{"notify":[],"block":null}` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, errOut, code := runFacts(t, preFactory, tt.stdin)
			if code != tt.code || out != tt.out || errOut != "" {
				t.Fatalf("exit %d, stdout %q, stderr %q; want exit %d, stdout %q, no stderr", code, out, errOut, tt.code, tt.out)
			}
		})
	}
}

// A bad gabo_facts object exits 2 with one usage line and no stdout; the
// project is never loaded.
func TestCommandFactsUsageExitsTwo(t *testing.T) {
	t.Parallel()
	for _, stdin := range []string{
		`{"gabo_facts":1,"event":"session_start","engine":"pi","tools":[]`,
		`{"gabo_facts":2,"event":"session_start","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"session_start","engine":"codex","tools":[]}`,
		`{"gabo_facts":1,"event":"agent_end","engine":"pi","tools":[]}`,
		`{"gabo_facts":1,"event":"tool_call","engine":"pi","tools":[]}`,
	} {
		out, errOut, code := runFacts(t, func(context.Context) (*UseCase, error) {
			t.Errorf("%s: the factory was called", stdin)
			return nil, errors.New("unexpected")
		}, stdin)
		if code != ExitUsage || out != "" || !strings.HasPrefix(errOut, "usage: gabo check --pre-write") || strings.Contains(errOut, "\n") {
			t.Fatalf("%s: exit %d, stdout %q, stderr %q; want exit 2 and one usage line", stdin, code, out, errOut)
		}
	}
}

// A project that cannot be loaded blocks a tool_call (fail closed), the
// subagent refusal first; session_start needs no project.
func TestCommandFactsFactoryError(t *testing.T) {
	t.Parallel()
	broken := func(context.Context) (*UseCase, error) { return nil, errors.New("getwd: gone") }
	out, _, code := runFacts(t, broken, factsJSON(t, "tool_call", []string{"write"}, []string{"README.md"}))
	want := `{"notify":[],"block":{"reason":` + mustJSON(t, SubagentRefusal+"\nthe project cannot be read (getwd: gone); the write counts as blocked") + `}}` + "\n"
	if code != ExitFailed || out != want {
		t.Fatalf("tool_call: exit %d, stdout %q; want exit 1, stdout %q", code, out, want)
	}
	out, _, code = runFacts(t, broken, factsJSON(t, "tool_call", []string{"subagent_run"}, []string{"README.md"}))
	want = `{"notify":[],"block":{"reason":"the project cannot be read (getwd: gone); the write counts as blocked"}}` + "\n"
	if code != ExitFailed || out != want {
		t.Fatalf("tool_call with subagent_run: exit %d, stdout %q; want exit 1, stdout %q", code, out, want)
	}
	out, _, code = runFacts(t, broken, factsJSON(t, "session_start", nil, nil))
	want = `{"notify":[{"level":"warning","text":` + mustJSON(t, SubagentRefusal) + `}],"block":null}` + "\n"
	if code != 0 || out != want {
		t.Fatalf("session_start: exit %d, stdout %q; want exit 0, stdout %q", code, out, want)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}
