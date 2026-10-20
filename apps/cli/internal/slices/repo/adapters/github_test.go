package adapters

import (
	"context"
	"slices"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestAPICallSendsBodyOnStdin(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`{"id":1}`)}}
	out, err := NewAPI(r).Call(context.Background(), "POST", "repos/o/r/rulesets", []byte(`{"name":"x"}`))
	if err != nil || string(out) != `{"id":1}` {
		t.Fatalf("Call() = %s, %v", out, err)
	}
	want := []string{"api", "--method", "POST", "repos/o/r/rulesets", "--input", "-"}
	if got := r.Calls[0].Command.Args; r.Calls[0].Command.Name != "gh" || !slices.Equal(got, want) {
		t.Fatalf("argv = %v", got)
	}
	if r.Calls[0].Stdin != `{"name":"x"}` {
		t.Fatalf("stdin = %q", r.Calls[0].Stdin)
	}
}

func TestAPICallWithoutBody(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`[]`)}}
	if _, err := NewAPI(r).Call(context.Background(), "GET", "repos/o/r/labels", nil); err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	want := []string{"api", "--method", "GET", "repos/o/r/labels"}
	if got := r.Calls[0].Command.Args; !slices.Equal(got, want) || r.Calls[0].Command.Stdin != nil {
		t.Fatalf("argv = %v stdin = %v", got, r.Calls[0].Command.Stdin)
	}
}

func TestAPICallFailsOnNonZeroExit(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 1, Stderr: []byte("HTTP 404")}}
	if _, err := NewAPI(r).Call(context.Background(), "GET", "x", nil); err == nil {
		t.Fatal("Call() error = nil")
	}
}
