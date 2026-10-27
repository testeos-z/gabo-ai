package initcmd

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

var files = []File{{Path: "gabo-setup/SKILL.md", Data: []byte("setup")}, {Path: "gabo-coder/SKILL.md", Data: []byte("coder")}}

func newUC(fsys *kerneltest.FS) *UseCase {
	return NewUseCase(fsys, files, Settings{Home: "/home/u", ProjectDir: "/work/p"})
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
		{"claude user", []Runtime{RuntimeClaude}, ScopeUser, []string{"/home/u/.claude/skills/gabo-setup/SKILL.md", "/home/u/.claude/skills/gabo-coder/SKILL.md"}},
		{"pi user", []Runtime{RuntimePi}, ScopeUser, []string{"/home/u/.pi/agent/skills/gabo-setup/SKILL.md", "/home/u/.pi/agent/skills/gabo-coder/SKILL.md"}},
		{"both project", []Runtime{RuntimeClaude, RuntimePi}, ScopeProject, []string{
			"/work/p/.claude/skills/gabo-setup/SKILL.md", "/work/p/.claude/skills/gabo-coder/SKILL.md",
			"/work/p/.pi/skills/gabo-setup/SKILL.md", "/work/p/.pi/skills/gabo-coder/SKILL.md",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fsys := &kerneltest.FS{}
			if _, err := newUC(fsys).Run(context.Background(), Input{Runtimes: tt.runtimes, Scope: tt.scope}); err != nil {
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
	target := "/home/u/.claude/skills/gabo-setup/SKILL.md"
	same := "/home/u/.claude/skills/gabo-coder/SKILL.md"
	fsys := &kerneltest.FS{Files: map[string]string{target: "edited by human", same: "coder"}}
	out, err := newUC(fsys).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser})
	if err != nil {
		t.Fatal(err)
	}
	if len(fsys.Writes) != 0 || fsys.Files[target] != "edited by human" {
		t.Fatalf("writes = %v", fsys.Writes)
	}
	if !strings.Contains(joined(out), "skipped "+target+" (differs; use --force)") || !strings.Contains(joined(out), "unchanged "+same) {
		t.Fatalf("output:\n%s", joined(out))
	}

	out, err = newUC(fsys).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude}, Scope: ScopeUser, Force: true})
	if err != nil || fsys.Files[target] != "setup" || !slices.Equal(fsys.Writes, []string{target}) {
		t.Fatalf("force: %v, writes %v, content %q", err, fsys.Writes, fsys.Files[target])
	}
	if !strings.Contains(joined(out), "overwrote "+target) {
		t.Fatalf("output:\n%s", joined(out))
	}
}

// gbai no longer ships an MCP server (T042, D18): init installs skills only
// and never mentions or registers one.
func TestInstallsSkillsOnlyWithoutMCPRegistration(t *testing.T) {
	t.Parallel()
	out, err := newUC(&kerneltest.FS{}).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude, RuntimePi}, Scope: ScopeUser})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Lines) != 2*len(files) {
		t.Fatalf("lines = %d, want one per installed file:\n%s", len(out.Lines), joined(out))
	}
	if strings.Contains(strings.ToLower(joined(out)), "mcp") {
		t.Fatalf("output mentions MCP:\n%s", joined(out))
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	t.Parallel()
	fsys := &kerneltest.FS{}
	out, err := newUC(fsys).Run(context.Background(), Input{Runtimes: []Runtime{RuntimeClaude, RuntimePi}, Scope: ScopeUser, DryRun: true})
	if err != nil || len(fsys.Writes) != 0 {
		t.Fatalf("err %v writes %v", err, fsys.Writes)
	}
	if want := "would install /home/u/.claude/skills/gabo-setup/SKILL.md"; !strings.Contains(joined(out), want) {
		t.Fatalf("missing %q:\n%s", want, joined(out))
	}
}

func TestParseFlags(t *testing.T) {
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
}
