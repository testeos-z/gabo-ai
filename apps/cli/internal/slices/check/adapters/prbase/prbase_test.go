package prbase

import (
	"errors"
	"reflect"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel/kerneltest"
)

// I0028 D4: the base is read with one gh pr view in the project directory, the
// -R repository and the pull request as the command names it.
func TestPRBaseReadsTheBaseBranch(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("main\n")}}
	base, err := GH{Runner: r, Dir: "/p"}.PRBase("o/r", "12")
	if err != nil || base != "main" {
		t.Fatalf("PRBase() = %q, %v; want main", base, err)
	}
	want := []string{"pr", "view", "12", "--json", "baseRefName", "--jq", ".baseRefName", "--repo", "o/r"}
	if len(r.Calls) != 1 || !reflect.DeepEqual(r.Calls[0].Command.Args, want) || r.Calls[0].Command.Name != "gh" || r.Calls[0].Command.Dir != "/p" {
		t.Fatalf("calls = %+v; want one gh %q in /p", r.Calls, want)
	}
}

// I0028 D4: a read that fails, prints nothing or is refused blocks the merge
// by returning an error: no answer is ever a base.
func TestPRBaseFailuresAreErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		r    *kerneltest.Runner
		ref  string
	}{
		{name: "run error", r: &kerneltest.Runner{Err: errors.New("no gh")}},
		{name: "non-zero exit", r: &kerneltest.Runner{Result: kernel.Result{ExitCode: 1, Stderr: []byte("no pull request")}}},
		{name: "empty answer", r: &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("\n")}}},
		{name: "option-like reference", r: &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("main")}}, ref: "--web"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if base, err := (GH{Runner: tt.r}).PRBase("", tt.ref); err == nil {
				t.Fatalf("PRBase() = %q, nil; want an error", base)
			}
			if tt.ref == "--web" && len(tt.r.Calls) != 0 {
				t.Fatalf("an option-like reference ran gh: %+v", tt.r.Calls)
			}
		})
	}
}
