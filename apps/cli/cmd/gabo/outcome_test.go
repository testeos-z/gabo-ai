package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/check"
)

// I0028 S16 through the composition root: the environment the cycle sets
// names the outcome file, the task and the engine of every record; with no
// file named the factory records nothing.
func TestCheckFactoryWiresTheOutcomeFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		env     map[string]string
		wantOut bool
	}{
		{"file named", map[string]string{check.EnvOutcomePath: "", check.EnvTask: "I0028/T3", check.EnvEngine: "claude"}, true},
		{"no file named", map[string]string{check.EnvTask: "I0028/T3", check.EnvEngine: "claude"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			name := filepath.Join(dir, "outcomes.jsonl")
			env := map[string]string{}
			for k, v := range c.env {
				env[k] = v
			}
			if env[check.EnvOutcomePath] == "" && c.wantOut {
				env[check.EnvOutcomePath] = name
			}
			uc, err := checkFactory(nil, "1.2.3", func(k string) string { return env[k] }, func() (string, error) { return dir, nil })(context.Background())
			if err != nil {
				t.Fatalf("factory error = %v", err)
			}
			if err := uc.RecordDecision(config.LockShellGuard, "git push to main is blocked", check.Decide(config.LockBlock, config.ModeManual)); err != nil {
				t.Fatalf("RecordDecision() error = %v", err)
			}
			got, err := os.ReadFile(name)
			if !c.wantOut {
				if err == nil {
					t.Fatalf("outcome file %s exists with %q; want none when no file is named", name, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			for _, want := range []string{`"outcome":"blocked"`, `"lock":"shell_guard"`, `"task":"I0028/T3"`, `"engine":"claude"`, `"rule":"git push to main is blocked"`} {
				if !strings.Contains(string(got), want) {
					t.Fatalf("record %q lacks %s", got, want)
				}
			}
		})
	}
}
