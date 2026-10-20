package secrets

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func execute(t *testing.T, uc *UseCase, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := NewCommand(func(context.Context) (*UseCase, error) { return uc, nil })
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSetCommandPrintsConfirmationOnly(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	out, err := execute(t, newUC(shell, &fakeReader{value: []byte(secretValue)}), "set", "reviewer-claude")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if out != "reviewer-claude stored for reviewer\n" {
		t.Fatalf("output = %q", out)
	}
}

func TestSetCommandRejectsExtraArgumentsAndValueFlags(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"set", "reviewer-claude", secretValue},
		{"set", "reviewer-claude", "--value", secretValue},
		{"set"},
	} {
		shell := &kerneltest.RemoteShell{}
		r := &fakeReader{value: []byte(secretValue)}
		if _, err := execute(t, newUC(shell, r), args...); err == nil {
			t.Fatalf("Execute(%q) error = nil, want rejection", args)
		}
		if r.calls != 0 || len(shell.Calls) != 0 {
			t.Fatalf("Execute(%q) touched reader or remote", args)
		}
	}
}

func TestSetCommandDryRunPrintsScript(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	out, err := execute(t, newUC(shell, &fakeReader{}), "set", "reviewer-github", "--dry-run")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out, "gh auth login") || !strings.Contains(out, "gh auth status") || len(shell.Calls) != 0 {
		t.Fatalf("output = %q, calls = %d", out, len(shell.Calls))
	}
}
