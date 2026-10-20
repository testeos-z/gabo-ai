package adapters

import (
	"io"
	"os"
	"strings"
	"testing"
)

func pipeWith(t *testing.T, content string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	go func() {
		_, _ = io.WriteString(w, content)
		_ = w.Close()
	}()
	return r
}

func TestReadSecretFromPipeTrimsTrailingNewline(t *testing.T) {
	t.Parallel()
	var prompt strings.Builder
	got, err := NewStdinReader(pipeWith(t, "tok en\r\n"), &prompt).ReadSecret("Value: ")
	if err != nil {
		t.Fatalf("ReadSecret() error = %v", err)
	}
	if string(got) != "tok en" {
		t.Fatalf("value = %q, want %q", got, "tok en")
	}
	if prompt.Len() != 0 {
		t.Fatalf("prompt written for piped stdin: %q", prompt.String())
	}
}

func TestReadSecretRejectsOversizedInput(t *testing.T) {
	t.Parallel()
	_, err := NewStdinReader(pipeWith(t, strings.Repeat("a", maxSecretBytes+1)), io.Discard).ReadSecret("")
	if err == nil {
		t.Fatal("ReadSecret() error = nil, want size error")
	}
}
