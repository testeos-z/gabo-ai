package check

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
)

func runCommand(t *testing.T, proj fs.FS, args ...string) (string, error) {
	t.Helper()
	s := Settings{Version: "1.2.3", Layers: layers(), Getenv: env("HOME", home), ReviewerDir: reviewerDir}
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return NewUseCase(proj, configFiles{projectFile: validConfig}, goodUser(), s), nil
	})
	// The root command silences cobra's own error and usage output.
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.ExecuteContext(context.Background())
	return out.String(), err
}

// Without docs/product/intents/ the command prints exactly what it printed
// before the intent check and exits 0.
func TestCommandWithoutIntentsIsUnchanged(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if out != PassedLine+"\n" {
		t.Fatalf("stdout = %q, want %q", out, PassedLine+"\n")
	}
}

func TestCommandIntentFailureExitsOne(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, withTree(t, project(), "plan-without-spec"))
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed {
		t.Fatalf("err = %v, want exit %d", err, ExitFailed)
	}
	want := "✗ " + intentsP + "0001-alpha/plan.md: no spec.md; a plan starts only after a human accepts the spec\n" + NextFixIntents + "\n"
	if out != want {
		t.Fatalf("stdout = %q, want %q", out, want)
	}
}

func TestCommandHelpNamesTheIntentCheck(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	line := "  - each " + IntentsDir + "/<NNNN-slug>/ folder: its name, intent.md, an accepted"
	if !strings.Contains(out, line) {
		t.Fatalf("help lacks %q:\n%s", line, out)
	}
}

func runPreWrite(t *testing.T, proj fs.FS, args ...string) (string, error) {
	t.Helper()
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return NewUseCase(proj, failingReader{}, goodUser(), Settings{Root: preRoot}), nil
	})
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(append([]string{}, args...))
	err := cmd.ExecuteContext(context.Background())
	return out.String(), err
}

// The hook mode judges only the given paths: it never reads the config
// (failingReader would fail the run) and prints only the blocked paths.
func TestCommandPreWrite(t *testing.T) {
	t.Parallel()
	out, err := runPreWrite(t, intentWith(unfilledMD), "--pre-write", "README.md", prePlan)
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitFailed || ee.Err != nil {
		t.Fatalf("err = %v, want a silent exit %d", err, ExitFailed)
	}
	if out != notAccepted+"\n" {
		t.Fatalf("stdout = %q, want %q", out, notAccepted+"\n")
	}
	out, err = runPreWrite(t, intentWith(acceptedMD), "--pre-write", prePlan, "README.md")
	if err != nil || out != "" {
		t.Fatalf("accepted: err = %v, stdout = %q, want nil and no output", err, out)
	}
}

func TestCommandPreWriteUsageExitsTwo(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--pre-write"}, {"--pre-write", ""}, {"--bogus", "--pre-write", prePlan}} {
		out, err := runPreWrite(t, intentWith(""), args...)
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitUsage {
			t.Fatalf("%q: err = %v, want exit %d", args, err, ExitUsage)
		}
		if !strings.Contains(err.Error(), "usage: gabo check --pre-write <path>...") || out != "" {
			t.Fatalf("%q: err = %v, stdout = %q", args, err, out)
		}
	}
}

func TestCommandHelpNamesThePreWriteFlag(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--help")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	for _, want := range []string{"--pre-write", "gabo check --pre-write <path>..."} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
}

// Usage errors of the hook mode never depend on the project: the factory
// is not called and the flag may come anywhere.
func TestCommandPreWriteUsageBeforeTheFactory(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--pre-write"}, {"--pre-write", ""}, {"--bogus", "--pre-write", prePlan}, {"--pre-write=x", prePlan}} {
		cmd := NewCommand(func(context.Context) (*UseCase, error) {
			t.Fatalf("%q: factory called", args)
			return nil, nil
		})
		cmd.SilenceUsage, cmd.SilenceErrors = true, true
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitUsage || !strings.Contains(err.Error(), "usage: gabo check --pre-write <path>...") {
			t.Fatalf("%q: err = %v, want the pre-write usage and exit %d", args, err, ExitUsage)
		}
	}
}

// --pre-write=false is not the plain check: it is a usage error, as any
// flag of the plain mode.
func TestCommandPreWriteFalseIsAUsageError(t *testing.T) {
	t.Parallel()
	out, err := runCommand(t, project(), "--pre-write=false")
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != ExitUsage || out != "" || !strings.Contains(err.Error(), "usage: gabo check") {
		t.Fatalf("err = %v, stdout = %q, want exit %d", err, out, ExitUsage)
	}
}

// Every argument after --pre-write is a path, also one that starts with
// "-": a hook passing paths in argv cannot be turned into a usage error
// (which hooks let through) by the path's name.
func TestCommandPreWriteArgumentsAfterTheFlagArePaths(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"--pre-write", "-x/../" + prePlan},
		{"--pre-write", "--help", prePlan},
		{"--pre-write", "--", prePlan},
	} {
		out, err := runPreWrite(t, intentWith(unfilledMD), args...)
		var ee *ExitError
		if !errors.As(err, &ee) || ee.Code != ExitFailed || out != notAccepted+"\n" {
			t.Fatalf("%q: err = %v, stdout = %q, want exit %d and %q", args, err, out, ExitFailed, notAccepted)
		}
	}
}
