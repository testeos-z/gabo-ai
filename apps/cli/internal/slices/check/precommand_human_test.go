package check

import (
	"errors"
	"strings"
	"testing"
)

// humanRoute is the route of a human-only path (S11), as the coder reads it.
const humanRoute = "a human edits this path"

// humanBlock is the message of a coder write to .github/ci.yml under block.
const humanBlock = "✗ human_only_paths: a coder shell command writes the human-only path .github/ci.yml\nnext: " + humanRoute + "\n"

// I0028 S7, S10, S11, S12, S16: a coder shell call that writes a human-only path
// is judged by the human_only_paths lock: its exit, its message and its record at each state.
func TestJudgeShellHumanOnlyPathPerState(t *testing.T) {
	t.Parallel()
	const cmd = "echo x > .github/ci.yml"
	tests := []struct {
		name, cfg, outcome string
		exit               int
		out                string
	}{
		{name: "no key blocks by default", cfg: validConfig, exit: ExitFailed, out: humanBlock, outcome: "blocked"},
		{name: "off runs with nothing recorded", cfg: validConfig + "locks:\n  human_only_paths: off\n", exit: 0},
		{name: "warn runs and records warned", cfg: validConfig + "locks:\n  human_only_paths: warn\n", exit: 0, outcome: "warned"},
		{name: "ask exits 3 and records asked", cfg: validConfig + "locks:\n  human_only_paths: ask\n", exit: ExitAsk, out: humanBlock, outcome: "asked"},
		{name: "ask in auto blocks with the note", cfg: validConfig + "mode: auto\nlocks:\n  human_only_paths: ask\n", exit: ExitFailed,
			out: "✗ human_only_paths: a coder shell command writes the human-only path .github/ci.yml (the state was ask; mode auto blocks instead of prompting)\nnext: " + humanRoute + "\n", outcome: "blocked"},
		{name: "all switch off", cfg: validConfig + "locks:\n  enabled: false\n", exit: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			v, err := judgeShell(t, configFiles{projectFile: tt.cfg}, "claude", "a1", cmd, out)
			if err != nil {
				t.Fatalf("JudgeShell() error = %v", err)
			}
			if v.Exit != tt.exit || v.Out != tt.out {
				t.Fatalf("verdict = %+v, want exit %d and out %q", v, tt.exit, tt.out)
			}
			if tt.outcome == "" {
				if len(out.lines) != 0 {
					t.Fatalf("records = %q, want none", out.lines)
				}
				return
			}
			if len(out.lines) != 1 {
				t.Fatalf("records = %q, want one", out.lines)
			}
			for _, want := range []string{`"outcome":"` + tt.outcome + `"`, `"lock":"human_only_paths"`, `"task":"I0028/T4"`, `"engine":"claude"`} {
				if !strings.Contains(out.lines[0], want) {
					t.Fatalf("record %q lacks %s", out.lines[0], want)
				}
			}
			if tt.outcome == "asked" && !strings.Contains(out.lines[0], `"answer":"pending"`) {
				t.Fatalf("asked record %q lacks answer pending", out.lines[0])
			}
		})
	}
}

// I0028 S13: a write outside the human-only paths runs with no output and no
// record, and a write to gabo.yaml is blocked with its name in the message.
func TestJudgeShellHumanOnlyPathBoundary(t *testing.T) {
	t.Parallel()
	out := &fakeOutcomes{}
	v, err := judgeShell(t, configFiles{projectFile: validConfig}, "claude", "a1", "echo x > apps/cli/out.txt", out)
	if err != nil || v.Exit != 0 || v.Out != "" || len(out.lines) != 0 {
		t.Fatalf("apps write: verdict = %+v, err = %v, records = %q; want exit 0, no output, no record", v, err, out.lines)
	}
	v, err = judgeShell(t, configFiles{projectFile: validConfig}, "claude", "a1", "echo x > gabo.yaml", &fakeOutcomes{})
	want := "✗ human_only_paths: a coder shell command writes the human-only path gabo.yaml\nnext: " + humanRoute + "\n"
	if err != nil || v.Exit != ExitFailed || v.Out != want {
		t.Fatalf("gabo.yaml: verdict = %+v, err = %v; want exit %d and out %q", v, err, ExitFailed, want)
	}
}

// I0028 S10, S13: each lock judges its own rule: the switch of one lock leaves
// the other judging, and a call that breaks both prints both messages, shell_guard first.
func TestJudgeShellLocksJudgeTheirOwnRules(t *testing.T) {
	t.Parallel()
	shellBlock := "✗ shell_guard: " + pushMainRule + "\nnext: " + pushMainRoute + "\n"
	tests := []struct {
		name, cfg, command string
		exit               int
		out                string
		locks              []string
	}{
		{name: "shell_guard off leaves the human path lock", cfg: validConfig + "locks:\n  shell_guard: off\n",
			command: "echo x > .github/ci.yml", exit: ExitFailed, out: humanBlock, locks: []string{"human_only_paths"}},
		{name: "human_only_paths off leaves the shell guard", cfg: validConfig + "locks:\n  human_only_paths: off\n",
			command: "git push origin main", exit: ExitFailed, out: shellBlock, locks: []string{"shell_guard"}},
		{name: "both rules break one call", cfg: validConfig, command: "git push origin main > .github/ci.yml",
			exit: ExitFailed, out: shellBlock + humanBlock, locks: []string{"shell_guard", "human_only_paths"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out := &fakeOutcomes{}
			v, err := judgeShell(t, configFiles{projectFile: tt.cfg}, "claude", "a1", tt.command, out)
			if err != nil || v.Exit != tt.exit || v.Out != tt.out {
				t.Fatalf("verdict = %+v, err = %v; want exit %d and out %q", v, err, tt.exit, tt.out)
			}
			if len(out.lines) != len(tt.locks) {
				t.Fatalf("records = %q, want one per lock %q", out.lines, tt.locks)
			}
			for i, lock := range tt.locks {
				if !strings.Contains(out.lines[i], `"lock":"`+lock+`"`) {
					t.Fatalf("record %d = %q, want lock %s", i, out.lines[i], lock)
				}
			}
		})
	}
}

// I0028 S7: the human-only rule judges a coder session only; the main session
// of claude is allowed to write the same path.
func TestJudgeShellHumanOnlyPathMainSessionAllowed(t *testing.T) {
	t.Parallel()
	v, err := judgeShell(t, configFiles{projectFile: validConfig}, "claude", "", "echo x > .github/ci.yml", &fakeOutcomes{})
	if err != nil || v.Exit != 0 || v.Out != "" {
		t.Fatalf("main session: verdict = %+v, err = %v; want exit 0 and no output", v, err)
	}
}

// I0028 S11: the CLI prints the human-only message as its one ✗ line and next: line.
func TestCommandPreCommandBlocksHumanOnlyPath(t *testing.T) {
	t.Parallel()
	call := `{"agent_id":"a1","tool_name":"Bash","tool_input":{"command":"echo x > .github/ci.yml"}}`
	out, _, err := runPreCommand(t, configFiles{projectFile: validConfig}, call, "--pre-command")
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed || ee.Err != nil {
		t.Fatalf("err = %v, want a silent exit %d", err, ExitFailed)
	}
	if out != humanBlock {
		t.Fatalf("stdout = %q, want %q", out, humanBlock)
	}
}
