package review

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func executeReview(t *testing.T, f Factories, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd := NewCommand(f)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestRunCommandPrintsExactlyOneSummaryLine(t *testing.T) {
	t.Parallel()
	h := newHarness()
	f := Factories{Run: func(context.Context) (*RunUseCase, error) {
		return NewRunUseCase(h.gh, h.git, h.checks, h.engine, h.cfg), nil
	}}
	out, _, err := executeReview(t, f, "", "run", "head", "12", sha)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out != "PR #12 head "+sha+" verdict=clean merge=merged\n" {
		t.Fatalf("stdout = %q", out)
	}
}

func TestRunCommandRejectsBadArguments(t *testing.T) {
	t.Parallel()
	called := false
	f := Factories{Run: func(context.Context) (*RunUseCase, error) { called = true; return nil, nil }}
	for _, args := range [][]string{{"run", "tail", "12", sha}, {"run", "head", "x", sha}, {"run", "head", "12"}} {
		if _, _, err := executeReview(t, f, "", args...); err == nil {
			t.Errorf("Execute(%q) error = nil", args)
		}
	}
	if called {
		t.Fatal("factory called for bad arguments")
	}
}

func TestRequestAndNotifyCommands(t *testing.T) {
	t.Parallel()
	o := &fakeOrca{t: &trace{}}
	f := Factories{
		Request: func(context.Context) (*RequestUseCase, error) { return NewRequestUseCase(o, "reviewer"), nil },
		Notify:  func(context.Context) (*NotifyUseCase, error) { return NewNotifyUseCase(o), nil },
	}
	out, _, err := executeReview(t, f, "", "request", "--pr", "12", "--sha", sha, "--notify", "run:abc")
	if err != nil || !strings.Contains(out, "gbai-review-12-0123456") {
		t.Fatalf("request = %q, %v", out, err)
	}
	line := "PR #12 head " + sha + " verdict=blocking merge=none"
	out, _, err = executeReview(t, f, "noise\n"+line+"\n", "notify", "--to", "run:abc", "--pr", "12", "--sha", sha)
	if err != nil || out != "noise\n"+line+"\n" || o.subjects[0] != line {
		t.Fatalf("notify out %q subjects %q err %v", out, o.subjects, err)
	}
}

func TestCICommands(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("built\n"), ExitCode: 0}}
	ci, ws := newCI(r)
	f := Factories{CI: func(context.Context) (*CIUseCase, error) { return ci, nil }}
	if _, _, err := executeReview(t, f, "TAR", "ci", "prepare", "--dir", wsName); err != nil || ws.created[wsName] != "TAR" {
		t.Fatalf("prepare err %v, ws %v", err, ws.created)
	}
	out, _, err := executeReview(t, f, "", "ci", "run", "--dir", wsName, "--", "bun", "run", "build", "--flag")
	if err != nil || out != "built\n" {
		t.Fatalf("run = %q, %v", out, err)
	}
	if c := r.Calls[0].Command; c.Name != "bun" || strings.Join(c.Args, " ") != "run build --flag" {
		t.Fatalf("argv = %s %q", c.Name, c.Args)
	}
	r.Result = kernel.Result{ExitCode: 3, Stderr: []byte("boom\n")}
	if _, errOut, err := executeReview(t, f, "", "ci", "run", "--dir", wsName, "--", "false"); err == nil || !strings.HasPrefix(errOut, "boom\n") {
		t.Fatalf("failing run err %v stderr %q", err, errOut)
	}
	if _, _, err := executeReview(t, f, "tok\nprompt", "ci", "engine", "--dir", wsName, "--runtime", "pi"); err == nil {
		t.Fatal("engine exit 3 returned nil")
	}
	if _, _, err := executeReview(t, f, "", "ci", "cleanup", "--dir", wsName); err != nil || len(ws.removed) != 1 {
		t.Fatalf("cleanup err %v removed %v", err, ws.removed)
	}
}
