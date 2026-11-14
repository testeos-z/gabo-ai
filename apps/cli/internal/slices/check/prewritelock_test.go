package check

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// coderWriteJSON is the tool-call JSON of a file-tool write by a subagent
// (its agent_id is set), as the engine's pre-write hook sends it.
func coderWriteJSON(path string) string {
	return `{"agent_id":"a1","tool_input":{"file_path":"` + path + `"}}`
}

// runWriteHook runs the command with cfg as gabo.yaml, engine as the run's
// engine, out as the outcome sink and stdin as the hook's tool-call JSON.
func runWriteHook(t *testing.T, proj fs.FS, cfg config.Reader, engine string, out *fakeOutcomes, stdin string, args ...string) (string, error) {
	t.Helper()
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		s := Settings{Root: preRoot, Layers: layers(), Engine: engine, Task: "I0028/T7", OutcomePath: "/scratch/outcomes.jsonl", Outcomes: out}
		return NewUseCase(proj, cfg, goodUser(), s), nil
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{}, args...))
	err := cmd.ExecuteContext(context.Background())
	return buf.String(), err
}

// I0028 S7 (acceptance): a coder write to a path under review.human_only_paths,
// or to gabo.yaml, exits 1 with the human-only lock's line and its route.
func TestWriteHumanOnlyPathBlockedForCoder(t *testing.T) {
	t.Parallel()
	for _, path := range []string{".github/workflows/x.yml", "gabo.yaml"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			got, err := runWriteHook(t, project(), configFiles{projectFile: validConfig}, "claude", out, coderWriteJSON(path), "--pre-write")
			var exit *ExitError
			if !errors.As(err, &exit) || exit.Code != ExitFailed {
				t.Fatalf("err = %v, want exit %d", err, ExitFailed)
			}
			want := "✗ human_only_paths: a coder write to the human-only path " + path + " is blocked\nnext: " + humanOnlyRoute + "\n"
			if got != want {
				t.Fatalf("output = %q, want %q", got, want)
			}
			if len(out.lines) != 1 || !strings.Contains(out.lines[0], `"outcome":"blocked"`) || !strings.Contains(out.lines[0], `"lock":"human_only_paths"`) {
				t.Fatalf("records = %q, want one blocked human_only_paths record", out.lines)
			}
		})
	}
}

// I0028 S7 (acceptance): a coder write to an ordinary path passes with no
// output and records nothing.
func TestWriteOrdinaryPathPassesForCoder(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	got, err := runWriteHook(t, project(), configFiles{projectFile: validConfig}, "claude", out, coderWriteJSON("apps/cli/main.go"), "--pre-write")
	if err != nil || got != "" || len(out.lines) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q; want nil, no output, no record", err, got, out.lines)
	}
}

// I0028 S7 and S4 (decision 13): the human-only rule judges coder sessions
// only; a main session (no agent id, an engine that gives one) may write it.
func TestWriteHumanOnlyPathPassesInMainSession(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	stdin := `{"tool_input":{"file_path":".github/workflows/x.yml"}}`
	got, err := runWriteHook(t, project(), configFiles{projectFile: validConfig}, "claude", out, stdin, "--pre-write")
	if err != nil || got != "" || len(out.lines) != 0 {
		t.Fatalf("err = %v, output = %q, records = %q; want nil, no output, no record", err, got, out.lines)
	}
}

// I0028 S10, S12: the human-only lock at each state for one violating write.
func TestWriteHumanOnlyPathOneViolatingWritePerState(t *testing.T) {
	t.Parallel()
	const path = ".github/workflows/x.yml"
	block := "✗ human_only_paths: a coder write to the human-only path " + path + " is blocked\nnext: " + humanOnlyRoute + "\n"
	tests := []struct {
		name    string
		cfg     string
		exit    int
		out     string
		outcome string
		answer  bool
	}{
		{name: "default blocks", cfg: validConfig, exit: ExitFailed, out: block, outcome: "blocked"},
		{name: "off passes with nothing recorded", cfg: validConfig + "locks:\n  human_only_paths: off\n", exit: 0},
		{name: "warn passes and records warned", cfg: validConfig + "locks:\n  human_only_paths: warn\n", exit: 0, outcome: "warned"},
		{name: "ask exits 3 and records asked", cfg: validConfig + "locks:\n  human_only_paths: ask\n", exit: ExitAsk, out: block, outcome: "asked", answer: true},
		{name: "ask in auto blocks with the note", cfg: validConfig + "mode: auto\nlocks:\n  human_only_paths: ask\n", exit: ExitFailed,
			out:     "✗ human_only_paths: a coder write to the human-only path " + path + " is blocked (the state was ask; mode auto blocks instead of prompting)\nnext: " + humanOnlyRoute + "\n",
			outcome: "blocked"},
		{name: "all switch off passes", cfg: validConfig + "locks:\n  enabled: false\n", exit: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			got, err := runWriteHook(t, project(), configFiles{projectFile: tt.cfg}, "claude", out, coderWriteJSON(path), "--pre-write")
			if tt.exit == 0 {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
			} else {
				var exit *ExitError
				if !errors.As(err, &exit) || exit.Code != tt.exit {
					t.Fatalf("err = %v, want exit %d", err, tt.exit)
				}
			}
			if got != tt.out {
				t.Fatalf("output = %q, want %q", got, tt.out)
			}
			if tt.outcome == "" {
				if len(out.lines) != 0 {
					t.Fatalf("records = %q, want none", out.lines)
				}
				return
			}
			if len(out.lines) != 1 || !strings.Contains(out.lines[0], `"outcome":"`+tt.outcome+`"`) {
				t.Fatalf("records = %q, want one %s record", out.lines, tt.outcome)
			}
			if tt.answer && !strings.Contains(out.lines[0], `"answer":"pending"`) {
				t.Fatalf("asked record %q lacks answer pending", out.lines[0])
			}
		})
	}
}

// I0028 S7 and S15: a project whose gabo.yaml cannot be read fails closed for
// a coder write, as the shell guard does for a call.
func TestWriteUnreadableConfigFailsClosedForCoder(t *testing.T) {
	t.Parallel()
	got, err := runWriteHook(t, project(), failingReader{}, "claude", &fakeOutcomes{}, coderWriteJSON("apps/cli/main.go"), "--pre-write")
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != ExitFailed {
		t.Fatalf("err = %v, want exit %d", err, ExitFailed)
	}
	if !strings.HasPrefix(got, "✗ human_only_paths: the project config cannot be read (") || !strings.HasSuffix(got, "next: fix the project, its gabo.yaml or the working directory, then retry\n") {
		t.Fatalf("output = %q, want the fail-closed block", got)
	}
}

// I0028 S7 with the intent stage gate (intent 0001 S7): a plan.md whose spec
// is not accepted keeps today's message and exit when the write is a coder's.
func TestWriteStageGateMessageUnchangedForCoder(t *testing.T) {
	t.Parallel()
	got, err := runWriteHook(t, intentWith(unfilledMD), configFiles{projectFile: validConfig}, "claude", &fakeOutcomes{}, coderWriteJSON(prePlan), "--pre-write")
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != ExitFailed {
		t.Fatalf("err = %v, want exit %d", err, ExitFailed)
	}
	if got != notAccepted+"\n" {
		t.Fatalf("output = %q, want %q", got, notAccepted+"\n")
	}
}

// I0028 S7: a write through a symlink whose target is human-only is judged at
// its target, as the stage gate judges a plan through a link.
func TestWriteHumanOnlyPathJudgedThroughASymlink(t *testing.T) {
	t.Parallel()
	links := map[string]string{"/proj/notes.yml": "/proj/.github/x.yml"}
	exists := []string{"/proj", "/proj/.github", "/proj/.github/x.yml"}
	s := Settings{Root: preRoot, Layers: layers(), Engine: "claude", Resolve: fakeResolve(links, exists...), Readlink: fakeReadlink(links, exists...)}
	v := NewUseCase(project(), configFiles{projectFile: validConfig}, goodUser(), s).JudgeWrites([]string{"notes.yml"}, "a1", true)
	want := "✗ human_only_paths: a coder write to the human-only path .github/x.yml is blocked\nnext: " + humanOnlyRoute + "\n"
	if v.Exit != ExitFailed || v.Out != want {
		t.Fatalf("verdict = %+v, want exit %d and %q", v, ExitFailed, want)
	}
}

// I0028 S7: a Codex apply_patch that writes one human-only file and one
// ordinary file is blocked for the human-only file alone.
func TestWriteApplyPatchJudgesEachFile(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	cmdJSON := `{"agent_id":"a1","tool_input":{"command":"*** Begin Patch\n*** Update File: .github/x.yml\n*** Update File: apps/cli/main.go\n*** End Patch"}}`
	got, err := runWriteHook(t, project(), configFiles{projectFile: validConfig}, "codex", out, cmdJSON, "--pre-write")
	var exit *ExitError
	if !errors.As(err, &exit) || exit.Code != ExitFailed {
		t.Fatalf("err = %v, want exit %d", err, ExitFailed)
	}
	if !strings.Contains(got, "a coder write to the human-only path .github/x.yml is blocked") || strings.Contains(got, "apps/cli/main.go") {
		t.Fatalf("output = %q, want the human-only file alone", got)
	}
	if len(out.lines) != 1 {
		t.Fatalf("records = %q, want one", out.lines)
	}
}

// I0028 S7: the Pi facts tool_call judges a coder's human-only write too,
// and an ask (no prompt in this mode) counts as blocked.
func TestJudgeFactsBlocksAHumanOnlyWrite(t *testing.T) {
	t.Parallel()
	s := Settings{Root: preRoot, Layers: layers(), Engine: "pi"}
	f := Facts{Event: EventToolCall, Engine: "pi", Tools: []string{"subagent_run"}, TargetPaths: []string{".github/x.yml"}}
	r := NewUseCase(project(), configFiles{projectFile: validConfig + "locks:\n  human_only_paths: ask\n"}, goodUser(), s).JudgeFacts(f)
	want := "✗ human_only_paths: a coder write to the human-only path .github/x.yml is blocked (the engine cannot show its prompt here; it counts as blocked)\nnext: " + humanOnlyRoute
	if r.Block == nil || r.Block.Reason != want {
		t.Fatalf("block = %+v, want reason %q", r.Block, want)
	}
}

// I0028 S7: the match of a path against review.human_only_paths follows the
// reviewer's rule: a directory entry (trailing /) by prefix, a file entry by
// its exact path or as a trailing path element; gabo.yaml always matches.
func TestHumanOnlyPathMatch(t *testing.T) {
	t.Parallel()
	entries := []string{".github/", "AGENTS.md", "docs/policies/"}
	tests := []struct {
		rel  string
		fold bool
		want bool
	}{
		{".github/workflows/x.yml", false, true},
		{".github-extra/x", false, false},
		{"AGENTS.md", false, true},
		{"sub/AGENTS.md", false, true},
		{"AGENTS.md.bak", false, false},
		{"docs/policies/a.md", false, true},
		{"docs/policy.md", false, false},
		{"gabo.yaml", false, true},
		{"apps/cli/main.go", false, false},
		{"agents.md", false, false},
		{"agents.md", true, true},
		{".GITHUB/x.yml", true, true},
		{".GITHUB/x.yml", false, false},
	}
	for _, tt := range tests {
		if got := humanOnlyPath(tt.rel, entries, tt.fold); got != tt.want {
			t.Errorf("humanOnlyPath(%q, fold=%v) = %v, want %v", tt.rel, tt.fold, got, tt.want)
		}
	}
}
