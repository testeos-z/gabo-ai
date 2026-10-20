package kernel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestRunCheckedTurnsNonZeroExitIntoError(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 2, Stderr: []byte("bad thing\n")}}
	_, err := kernel.RunChecked(context.Background(), r, kernel.Command{Name: "gh", Args: []string{"api", "x"}})
	if err == nil || !strings.Contains(err.Error(), "gh api exited 2: bad thing") {
		t.Fatalf("RunChecked() error = %v", err)
	}
	r = &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("ok")}}
	res, err := kernel.RunChecked(context.Background(), r, kernel.Command{Name: "gh"})
	if err != nil || string(res.Stdout) != "ok" {
		t.Fatalf("RunChecked() = %q, %v", res.Stdout, err)
	}
}
