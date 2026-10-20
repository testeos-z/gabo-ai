package execrunner

import (
	"context"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

func TestRunCapturesOutputStdinAndExitCode(t *testing.T) {
	t.Parallel()
	r := New()
	res, err := r.Run(context.Background(), kernel.Command{
		Name:  "sh",
		Args:  []string{"-c", `cat; printf err >&2; printf %s "$EXTRA"; exit 3`},
		Env:   []string{"EXTRA=-x"},
		Stdin: strings.NewReader("in"),
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := string(res.Stdout); got != "in-x" {
		t.Errorf("stdout = %q, want %q", got, "in-x")
	}
	if got := string(res.Stderr); got != "err" {
		t.Errorf("stderr = %q, want %q", got, "err")
	}
	if res.ExitCode != 3 {
		t.Errorf("exit code = %d, want 3", res.ExitCode)
	}
}

func TestRunMissingBinaryIsAnError(t *testing.T) {
	t.Parallel()
	_, err := New().Run(context.Background(), kernel.Command{Name: "gbai-definitely-missing-binary"})
	if err == nil {
		t.Fatal("Run() error = nil, want error for missing binary")
	}
}
