package sessions

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
)

func TestCommandStartPassesModeAndRole(t *testing.T) {
	t.Parallel()
	terms := &kerneltest.Terminals{}
	var gotMode Mode
	cmd := NewCommand(func(_ context.Context, m Mode) (*UseCase, error) {
		gotMode = m
		return NewUseCase(terms, settings), nil
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"start", "--role", "assistant", "--mode", "local"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if gotMode != ModeLocal || len(terms.Created) != 1 || terms.Created[0].Title != "gbai-assistant" {
		t.Fatalf("mode = %s created = %+v", gotMode, terms.Created)
	}
	if !strings.Contains(out.String(), "gbai-assistant: started") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCommandRejectsBadRoleBeforeFactory(t *testing.T) {
	t.Parallel()
	called := false
	cmd := NewCommand(func(context.Context, Mode) (*UseCase, error) { called = true; return nil, nil })
	cmd.SetArgs([]string{"stop", "--role", "boss"})
	if err := cmd.Execute(); err == nil || called {
		t.Fatalf("Execute() error = %v called = %v", err, called)
	}
}
