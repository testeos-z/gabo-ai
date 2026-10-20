package initcmd

import (
	"context"
	"testing"
)

func TestCommandRejectsBadFlagsBeforeFactory(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"--runtime", "codex"}, {"--scope", "global"}} {
		called := false
		cmd := NewCommand(func(context.Context) (*UseCase, error) { called = true; return nil, nil })
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || called {
			t.Fatalf("%v: err %v, factory called %v", args, err, called)
		}
	}
}
