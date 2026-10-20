package repo

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCommandSetupDryRun(t *testing.T) {
	t.Parallel()
	api := &fakeAPI{}
	cmd := NewCommand(func(context.Context) (*UseCase, error) { return NewUseCase(api, settings), nil })
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"setup", "--dry-run"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(api.calls) != 0 || !strings.Contains(out.String(), "POST repos/cappy-lab/vulcam/rulesets") {
		t.Fatalf("calls = %v output:\n%s", api.calls, out.String())
	}
}
