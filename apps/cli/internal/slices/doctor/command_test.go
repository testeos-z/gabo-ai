package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestCommandFailsWhenAnyCheckFails(t *testing.T) {
	t.Parallel()
	s := settings
	s.Remote = false
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return NewUseCase(fakeLocal{"gh": true}, &kerneltest.Runner{}, &kerneltest.RemoteShell{}, s), nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); !errors.Is(err, ErrFailed) {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "fail  local: railway") || !strings.Contains(out.String(), "fix: install the Railway CLI") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestCommandJSON(t *testing.T) {
	t.Parallel()
	s := settings
	s.Remote = false
	cmd := NewCommand(func(context.Context) (*UseCase, error) {
		return NewUseCase(allTools, &kerneltest.Runner{}, &kerneltest.RemoteShell{}, s), nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var rep Report
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil || len(rep.Checks) != 5 {
		t.Fatalf("json = %s, %v", out.String(), err)
	}
}
