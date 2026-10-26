package worktree

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func runClose(t *testing.T, git *fakeGit, args ...string) (string, string, error) {
	t.Helper()
	factory := func(context.Context) (*UseCase, error) {
		return NewUseCase(git, &fakeGitHub{}, fakePaths{}, settings), nil
	}
	var out, errOut bytes.Buffer
	cmd := NewCommand(factory)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestCommandPrintsOneClosedLineAndSucceeds(t *testing.T) {
	t.Parallel()
	out, _, err := runClose(t, approvedGit(), "close", target)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out != "CLOSED "+target+" branch=feat/x reason=in-base\n" {
		t.Fatalf("stdout = %q", out)
	}
}

func TestCommandPrintsOneRefusedLineAndFails(t *testing.T) {
	t.Parallel()
	git := approvedGit()
	git.changed = 3
	out, _, err := runClose(t, git, "close", target)
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("Execute() error = %v, want ErrRefused", err)
	}
	if !strings.HasPrefix(out, "REFUSED "+target+" not clean") || strings.Count(out, "\n") != 1 {
		t.Fatalf("stdout = %q", out)
	}
}

func TestCommandReportsAKeptBranchOnStderr(t *testing.T) {
	t.Parallel()
	git := approvedGit()
	git.deleteErr = errors.New("not fully merged")
	out, errOut, err := runClose(t, git, "close", target)
	if err != nil || strings.Count(out, "\n") != 1 || !strings.Contains(errOut, "not fully merged") {
		t.Fatalf("err = %v, stdout = %q, stderr = %q", err, out, errOut)
	}
}

func TestCommandRejectsForceAndExtraArguments(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"close", "--force", target}, {"close", "-f", target}, {"close"}, {"close", target, "/other"}} {
		git := approvedGit()
		if _, _, err := runClose(t, git, args...); err == nil || len(git.calls) != 0 {
			t.Errorf("%q: error = %v, calls %q", args, err, git.calls)
		}
	}
}
