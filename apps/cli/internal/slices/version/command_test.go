package version

import (
	"bytes"
	"testing"
)

func TestCommandPrintsVersion(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	cmd := NewCommand(NewUseCase("1.2.3"))
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := out.String(), "gbai 1.2.3\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
