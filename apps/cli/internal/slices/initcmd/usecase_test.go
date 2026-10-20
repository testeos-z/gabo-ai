package initcmd

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

const bin = "/opt/gbai/bin/gbai"

var files = []File{{Path: "gbai-setup/SKILL.md", Data: []byte("setup")}, {Path: "gbai-coder/SKILL.md", Data: []byte("coder")}}

func newUC(fsys *kerneltest.FS, r *kerneltest.Runner) *UseCase {
	return NewUseCase(fsys, r, files, Settings{Home: "/home/u", ProjectDir: "/work/p", Binary: bin})
}

func joined(out Output) string { return strings.Join(out.Lines, "\n") }

func TestInstallsSkillsPerRuntimeAndScope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		runtimes []Runtime
		scope    Scope
		want     []string
	}{
		{"claude user", []Runtime{RuntimeClaude}, ScopeUser, []string{"/home/u/.claude/skills/gbai-setup/SKILL.md", "/home/u/.claude/skills/gbai-coder/SKILL.md"}},
		{"pi user", []Runtime{RuntimePi}, ScopeUser, []string{"/home/u/.pi/agent/skills/gbai-setup/SKILL.md", "/home/u/.pi/agent/skills/gbai-coder/SKILL.md"}},
		{"both project", []Runtime{RuntimeClaude, RuntimePi}, ScopeProject, []string{
			"/work/p/.claude/skills/gbai-setup/SKILL.md", "/work/p/.claude/skills/gbai-coder/SKILL.md",
			"/work/p/.pi/skills/gbai-setup/SKILL.md", "/work/p/.pi/skills/gbai-coder/SKILL.md",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fsys := &kerneltest.FS{}
			r := &kerneltest.Runner{}
			if _, err := newUC(fsys, r).Run(context.Background(), Input{Runtimes: tt.runtimes, Scope: tt.scope}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(fsys.Writes, tt.want) {
				t.Fatalf("writes = %v\nwant     %v", fsys.Writes, tt.want)
			}
			if fsys.Files[tt.want[0]] != "setup" {
				t.Fatalf("content = %q", fsys.Files[tt.want[0]])
			}
		})
	}
}

func TestNeverOverwritesDifferentContentWithoutForce(t *testing.T) {
	t.Parallel()
	target := "/home/u/.claude/skills/gbai-setup/SKILL.md"
	same := "/home/u/.claude/skills/gbai-coder/SKILL.md"
	fsys := &kerneltest.FS{Files: map[string]string{target: "edited by human", same: "coder"}}
	out, err := newUC(fsys, &kerneltest.Runner{}).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser})
	if err != nil {
		t.Fatal(err)
	}
	if len(fsys.Writes) != 0 || fsys.Files[target] != "edited by human" {
		t.Fatalf("writes = %v", fsys.Writes)
	}
	if !strings.Contains(joined(out), "skipped "+target+" (differs; use --force)") || !strings.Contains(joined(out), "unchanged "+same) {
		t.Fatalf("output:\n%s", joined(out))
	}

	out, err = newUC(fsys, &kerneltest.Runner{}).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser, Force: true})
	if err != nil || fsys.Files[target] != "setup" || !slices.Equal(fsys.Writes, []string{target}) {
		t.Fatalf("force: %v, writes %v, content %q", err, fsys.Writes, fsys.Files[target])
	}
	if !strings.Contains(joined(out), "overwrote "+target) {
		t.Fatalf("output:\n%s", joined(out))
	}
}

func TestRegistersClaudeMCPOnlyWhenMissing(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: func(c kernel.Command) (kernel.Result, error) {
		if slices.Equal(c.Args, []string{"mcp", "get", "gbai"}) {
			return kernel.Result{ExitCode: 1}, nil
		}
		return kernel.Result{}, nil
	}}
	out, err := newUC(&kerneltest.FS{}, r).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Calls) != 2 {
		t.Fatalf("calls = %+v", r.Calls)
	}
	add := r.Calls[1].Command
	if add.Name != "claude" || !slices.Equal(add.Args, []string{"mcp", "add", "--scope", "user", "gbai", "--", bin, "mcp"}) {
		t.Fatalf("add = %+v", add)
	}
	if !strings.Contains(joined(out), "claude: registered MCP server gbai") {
		t.Fatalf("output:\n%s", joined(out))
	}

	present := &kerneltest.Runner{}
	out, err = newUC(&kerneltest.FS{}, present).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser})
	if err != nil || len(present.Calls) != 1 || !strings.Contains(joined(out), "already registered") {
		t.Fatalf("present: %v, calls %+v\n%s", err, present.Calls, joined(out))
	}
}

func TestClaudeAddFailureIsReported(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 1, Stderr: []byte("nope")}}
	if _, err := newUC(&kerneltest.FS{}, r).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser}); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error = %v", err)
	}
}

func TestPiPrintsSnippetAndRunsNothing(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	out, err := newUC(&kerneltest.FS{}, r).Run(context.Background(), Input{Runtimes: []Runtime{RuntimePi}, Scope: ScopeUser})
	if err != nil || len(r.Calls) != 0 {
		t.Fatalf("err %v calls %v", err, r.Calls)
	}
	if !strings.Contains(joined(out), `"gbai": {"command": "`+bin+`", "args": ["mcp"]}`) {
		t.Fatalf("output:\n%s", joined(out))
	}
}

func TestDryRunWritesAndRunsNothing(t *testing.T) {
	t.Parallel()
	fsys := &kerneltest.FS{}
	r := &kerneltest.Runner{}
	out, err := newUC(fsys, r).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude, RuntimePi}, Scope: ScopeUser, DryRun: true})
	if err != nil || len(fsys.Writes) != 0 || len(r.Calls) != 0 {
		t.Fatalf("err %v writes %v calls %v", err, fsys.Writes, r.Calls)
	}
	for _, want := range []string{"would install /home/u/.claude/skills/gbai-setup/SKILL.md", "would run: claude mcp add --scope user gbai -- " + bin + " mcp"} {
		if !strings.Contains(joined(out), want) {
			t.Fatalf("missing %q:\n%s", want, joined(out))
		}
	}
}

func TestParseFlagsAndRelativeBinary(t *testing.T) {
	t.Parallel()
	if rt, err := ParseRuntimes("both"); err != nil || !slices.Equal(rt, []Runtime{RuntimeClaude, RuntimePi}) {
		t.Fatalf("both = %v, %v", rt, err)
	}
	if _, err := ParseRuntimes("codex"); err == nil {
		t.Fatal("codex accepted")
	}
	if _, err := ParseScope("global"); err == nil {
		t.Fatal("global accepted")
	}
	uc := NewUseCase(&kerneltest.FS{}, &kerneltest.Runner{}, files, Settings{Home: "/home/u", ProjectDir: "/p", Binary: "gbai"})
	if _, err := uc.Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser}); err == nil {
		t.Fatal("relative binary accepted")
	}
}
