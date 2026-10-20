package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpListsVersion(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, name := range []string{"version", "secrets"} {
		if !strings.Contains(out.String(), name) {
			t.Fatalf("help output does not list %s command:\n%s", name, out.String())
		}
	}
}

func TestRootVersionSubcommand(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := out.String(); got != "gbai 1.2.3\n" {
		t.Fatalf("output = %q", got)
	}
}

func TestSecretsSetFailsWithoutConfigBeforeReadingValue(t *testing.T) {
	t.Parallel()
	root := newRootCommand(deps{version: "1.2.3"})
	root.SetArgs([]string{"--config", t.TempDir() + "/missing.yaml", "secrets", "set", "reviewer-pi"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "config") {
		t.Fatalf("Execute() error = %v, want config error", err)
	}
}
