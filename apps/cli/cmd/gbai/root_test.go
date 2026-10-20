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
	if !strings.Contains(out.String(), "version") {
		t.Fatalf("help output does not list version command:\n%s", out.String())
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
