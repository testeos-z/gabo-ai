package review

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

type fakeWorkspace struct {
	created map[string]string
	removed []string
}

func (f *fakeWorkspace) Create(name string, r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	if f.created == nil {
		f.created = map[string]string{}
	}
	f.created[name] = string(b)
	return "/ci/work/" + name, nil
}

func (f *fakeWorkspace) Dir(name string) (string, error) {
	if _, ok := f.created[name]; !ok {
		return "", errors.New("missing workspace")
	}
	return "/ci/work/" + name, nil
}

func (f *fakeWorkspace) Remove(name string) error {
	f.removed = append(f.removed, name)
	return nil
}

func newCI(r *kerneltest.Runner) (*CIUseCase, *fakeWorkspace) {
	ws := &fakeWorkspace{}
	return NewCIUseCase(ws, r, CIConfig{CredentialEnv: map[string]string{RuntimeClaude: "CLAUDE_CODE_OAUTH_TOKEN", RuntimePi: "ANTHROPIC_API_KEY"}}), ws
}

func TestCIRunExecsPlainArgvInWorkspace(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 2}}
	ci, ws := newCI(r)
	if err := ci.Prepare(wsName, strings.NewReader("TAR")); err != nil {
		t.Fatal(err)
	}
	res, err := ci.Run(context.Background(), wsName, []string{"bun", "run", "test; rm -rf /"})
	if err != nil || res.ExitCode != 2 {
		t.Fatalf("Run() = %+v, %v", res, err)
	}
	c := r.Calls[0].Command
	if c.Name != "bun" || !slices.Equal(c.Args, []string{"run", "test; rm -rf /"}) || c.Dir != "/ci/work/"+wsName || len(c.Env) != 0 {
		t.Fatalf("command = %+v", c)
	}
	if ws.created[wsName] != "TAR" {
		t.Fatalf("workspace = %v", ws.created)
	}
}

func TestCIRejectsBadWorkspaceNames(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	ci, ws := newCI(r)
	for _, name := range []string{"", "../x", "pr-1-ABCDEFG", "pr-0-0123456", "/tmp/x", "pr-1-0123456/.."} {
		if err := ci.Prepare(name, strings.NewReader("")); err == nil {
			t.Errorf("Prepare(%q) error = nil", name)
		}
		if _, err := ci.Run(context.Background(), name, []string{"true"}); err == nil {
			t.Errorf("Run(%q) error = nil", name)
		}
		if err := ci.Cleanup(name); err == nil {
			t.Errorf("Cleanup(%q) error = nil", name)
		}
	}
	if len(r.Calls) != 0 || len(ws.created) != 0 || len(ws.removed) != 0 {
		t.Fatal("bad names reached ports")
	}
}

func TestCIEngineUsesReadOnlyArgvAndCredentialFromStdin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		runtime string
		argv    []string
		env     string
	}{
		{RuntimeClaude, []string{"claude", "-p", "--setting-sources", "user", "--strict-mcp-config", "--output-format", "json", "--allowedTools", "Read,Grep,Glob"}, "CLAUDE_CODE_OAUTH_TOKEN=tok"},
		{RuntimePi, []string{"pi", "-p", "--mode", "json", "--no-mcp", "--no-extensions", "--no-context-files", "--no-skills", "--tools", "read,grep,find,ls"}, "ANTHROPIC_API_KEY=tok"},
	}
	for _, tt := range tests {
		r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("{}")}}
		ci, _ := newCI(r)
		if err := ci.Prepare(wsName, strings.NewReader("")); err != nil {
			t.Fatal(err)
		}
		res, err := ci.Engine(context.Background(), wsName, tt.runtime, strings.NewReader("tok\nthe prompt\nline 2"))
		if err != nil || string(res.Stdout) != "{}" {
			t.Fatalf("%s: Engine() = %+v, %v", tt.runtime, res, err)
		}
		c := r.Calls[0]
		if got := append([]string{c.Command.Name}, c.Command.Args...); !slices.Equal(got, tt.argv) {
			t.Errorf("%s: argv = %q", tt.runtime, got)
		}
		if !slices.Equal(c.Command.Env, []string{tt.env}) || c.Stdin != "the prompt\nline 2" || c.Command.Dir != "/ci/work/"+wsName {
			t.Errorf("%s: env %q stdin %q dir %q", tt.runtime, c.Command.Env, c.Stdin, c.Command.Dir)
		}
		for _, a := range c.Command.Args {
			if strings.Contains(a, "tok") {
				t.Errorf("%s: credential in argv", tt.runtime)
			}
		}
	}
}

func TestCIEngineRejectsUnknownRuntimeAndEmptyCredential(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	ci, _ := newCI(r)
	_ = ci.Prepare(wsName, strings.NewReader(""))
	if _, err := ci.Engine(context.Background(), wsName, "opencode", strings.NewReader("tok\np")); err == nil {
		t.Error("unknown runtime accepted")
	}
	if _, err := ci.Engine(context.Background(), wsName, RuntimeClaude, strings.NewReader("\nprompt")); err == nil {
		t.Error("empty credential accepted")
	}
	if len(r.Calls) != 0 {
		t.Fatalf("calls = %d", len(r.Calls))
	}
}

func TestCICleanupRemovesWorkspace(t *testing.T) {
	t.Parallel()
	ci, ws := newCI(&kerneltest.Runner{})
	if err := ci.Cleanup(wsName); err != nil || !slices.Equal(ws.removed, []string{wsName}) {
		t.Fatalf("Cleanup() = %v, removed %v", err, ws.removed)
	}
}
