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
