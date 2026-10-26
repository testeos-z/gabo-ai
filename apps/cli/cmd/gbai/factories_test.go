package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/sessions"
)

// status is read-only: when the owner login cannot be resolved, only the
// verdict lookups degrade to unknown; the snapshot is still taken.
func TestStatusDegradesWhenTheOwnerLoginLookupFails(t *testing.T) {
	t.Parallel()
	cfg, err := config.Load("../../gbai.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Bot.Login = ""
	r := &kerneltest.Runner{Handle: func(c kernel.Command) (kernel.Result, error) {
		args := strings.Join(c.Args, " ")
		switch {
		case c.Name == "gh" && args == "api user --jq .login":
			return kernel.Result{ExitCode: 1, Stderr: []byte("gh: not logged in")}, nil
		case c.Name == "gh" && strings.HasPrefix(args, "pr list"):
			return kernel.Result{Stdout: []byte(`[{"number":7,"title":"x","headRefOid":"aaa","labels":[]}]`)}, nil
		case c.Name == "gh":
			return kernel.Result{Stdout: []byte(`[]`)}, nil
		}
		return kernel.Result{}, nil
	}}
	f := newFactories(deps{runner: r}, func() (config.Config, error) { return cfg, nil })
	uc, err := f.status(context.Background(), "local")
	if err != nil {
		t.Fatalf("status factory error = %v, want a degraded snapshot", err)
	}
	snap := uc.Snapshot(context.Background())
	if len(snap.PullRequests.Items) != 1 {
		t.Fatalf("pull requests = %+v", snap.PullRequests)
	}
	if v := snap.PullRequests.Items[0].Verdict; !strings.HasPrefix(v, "unknown: ") || !strings.Contains(v, "not logged in") {
		t.Fatalf("verdict = %q, want unknown with the login error", v)
	}
	for _, c := range r.Calls {
		if c.Command.Name == "gh" && slices.Contains(c.Command.Args, "comments") {
			t.Fatalf("verdict looked up without an owner login: %q", c.Command.Args)
		}
	}
}

// A local-only config (TD1) loads; every command that reaches the cloud
// server fails up front naming the missing key and the next step.
func TestCloudCommandsNameTheMissingServerKey(t *testing.T) {
	t.Parallel()
	cfg, err := config.Parse([]byte("repo:\n  name: testeos-z/gbai-team\n"))
	if err != nil {
		t.Fatalf("local-only config error = %v", err)
	}
	r := &kerneltest.Runner{}
	f := newFactories(deps{runner: r}, func() (config.Config, error) { return cfg, nil })
	ctx := context.Background()
	for name, build := range map[string]func() error{
		"secrets":        func() error { _, err := f.secrets(ctx); return err },
		"provision":      func() error { _, err := f.provision(ctx); return err },
		"sessions cloud": func() error { _, err := f.sessions(ctx, sessions.ModeCloud); return err },
	} {
		err := build()
		if err == nil || !strings.Contains(err.Error(), "✗ server.railway.project_id missing") ||
			!strings.Contains(err.Error(), "next: configure slots.agent_host") {
			t.Errorf("%s factory error = %v, want the missing key and next step", name, err)
		}
	}
	if _, err := f.sessions(ctx, sessions.ModeLocal); err != nil {
		t.Errorf("sessions local factory error = %v, want no cloud keys needed", err)
	}
	if len(r.Calls) != 0 {
		t.Errorf("commands ran before the config check: %+v", r.Calls)
	}
}
