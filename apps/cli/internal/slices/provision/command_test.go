package provision

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
	cmd := NewCommand(
		func(context.Context) (*UseCase, error) { return uc, nil },
		func() (string, error) { return "/proc/self/exe", nil },
	)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestCommandDryRunPrintsPlan(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	bin := &fakeBinary{}
	out, err := execute(t, NewUseCase(shell, bin, settings()), "--dry-run")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out, "# step: legacy") || len(shell.Calls) != 0 || len(bin.calls) != 0 {
		t.Fatalf("output = %q, calls = %d, reads = %d", out, len(shell.Calls), len(bin.calls))
	}
}

func TestCommandDefaultsToRunningExecutable(t *testing.T) {
	t.Parallel()
	shell := &kerneltest.RemoteShell{}
	bin := &fakeBinary{data: linuxELF(t)}
	out, err := execute(t, NewUseCase(shell, bin, settings()))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(bin.calls) != 1 || bin.calls[0] != "/proc/self/exe" {
		t.Fatalf("ReadBinary calls = %v", bin.calls)
	}
	if !strings.HasSuffix(out, "ok legacy\nok skills\n") {
		t.Fatalf("output = %q", out)
	}
}

func TestCommandRejectsArguments(t *testing.T) {
	t.Parallel()
	if _, err := execute(t, NewUseCase(&kerneltest.RemoteShell{}, &fakeBinary{}, settings()), "extra"); err == nil {
		t.Fatal("Execute() error = nil, want rejection")
	}
}
