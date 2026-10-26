package configcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/config"
)

const minimal = `
server:
  railway:
    project_id: p1
    environment_id: e1
    service_id: s1
repo:
  name: o/r
`

func resolution(t *testing.T) config.Resolution {
	t.Helper()
	cfg, err := config.Parse([]byte(minimal))
	if err != nil {
		t.Fatal(err)
	}
	return config.Resolution{
		Layers: []config.Layer{
			{Name: config.LayerSystem, Path: "/etc/gbai/gbai.yaml"},
			{Name: config.LayerUser, Path: "/home/o/.config/gbai/config.yaml", Found: true},
			{Name: config.LayerProject, Path: "/src/p/gbai.yaml", Found: true},
		},
		Config: cfg,
	}
}

func TestCommandListsLayersThenResolvedConfig(t *testing.T) {
	t.Parallel()
	res := resolution(t)
	var out bytes.Buffer
	cmd := NewCommand(func(context.Context) (config.Resolution, error) { return res, nil })
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"layers (lowest to highest precedence):",
		"system", "/etc/gbai/gbai.yaml", "not found",
		"user", "/home/o/.config/gbai/config.yaml",
		"project", "/src/p/gbai.yaml",
		"resolved config:",
		"project_id: p1",
		"parallelism: 3",
		"pi_credential_env: ANTHROPIC_API_KEY",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q:\n%s", want, got)
		}
	}
	lines := strings.Split(got, "\n")
	for i, name := range []string{"system", "user", "project"} {
		if !strings.HasPrefix(strings.TrimSpace(lines[i+1]), name+" ") {
			t.Errorf("layer line %d = %q, want %s", i+1, lines[i+1], name)
		}
	}
	if !strings.Contains(lines[1], "not found") || strings.Contains(lines[2], "not found") {
		t.Errorf("found markers wrong:\n%s", got)
	}
}

func TestCommandShowsLayersEvenWhenResolutionFails(t *testing.T) {
	t.Parallel()
	res := resolution(t)
	var out bytes.Buffer
	cmd := NewCommand(func(context.Context) (config.Resolution, error) {
		return config.Resolution{Layers: res.Layers}, errors.New("config /src/p/gbai.yaml: unknown field")
	})
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "/src/p/gbai.yaml") {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), "/home/o/.config/gbai/config.yaml") || strings.Contains(out.String(), "resolved config:") {
		t.Fatalf("output:\n%s", out.String())
	}
}

func TestRedactHidesCredentialLookingValues(t *testing.T) {
	t.Parallel()
	in := []byte("review:\n  pi_credential_env: ANTHROPIC_API_KEY\n  api_token: abc\nbot:\n  password: hunter2\n  login: owner\nsecrets:\n  - x\nprivate_key: k\n")
	got, err := Redact(in)
	if err != nil {
		t.Fatalf("Redact() error = %v", err)
	}
	for _, leaked := range []string{"abc", "hunter2", "- x", ": k\n"} {
		if strings.Contains(string(got), leaked) {
			t.Errorf("leaked %q:\n%s", leaked, got)
		}
	}
	for _, kept := range []string{"pi_credential_env: ANTHROPIC_API_KEY", "login: owner", "api_token: '[redacted]'", "password: '[redacted]'"} {
		if !strings.Contains(string(got), kept) {
			t.Errorf("missing %q:\n%s", kept, got)
		}
	}
}
