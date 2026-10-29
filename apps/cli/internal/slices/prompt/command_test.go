package prompt

import (
	"bytes"
	"testing"
)

func TestCommandWritesTheTextVerbatim(t *testing.T) {
	t.Parallel()
	const text = "line one\nline two\n"
	var out bytes.Buffer
	cmd := NewCommand(text)
	cmd.SetOut(&out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil || out.String() != text {
		t.Fatalf("Execute() = %v, output %q", err, out.String())
	}
}
