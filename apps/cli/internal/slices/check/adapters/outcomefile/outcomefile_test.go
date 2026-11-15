package outcomefile

import (
	"os"
	"path/filepath"
	"testing"
)

// I0028 S16: each call appends its line to the named file, in order, and an
// earlier record is never rewritten.
func TestAppendAddsOneLinePerCall(t *testing.T) {
	t.Parallel()
	name := filepath.Join(t.TempDir(), "outcomes.jsonl")
	var f File
	if err := f.Append(name, []byte("{\"outcome\":\"warned\"}\n")); err != nil {
		t.Fatalf("Append() first error = %v", err)
	}
	if err := f.Append(name, []byte("{\"outcome\":\"blocked\"}\n")); err != nil {
		t.Fatalf("Append() second error = %v", err)
	}
	got, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := "{\"outcome\":\"warned\"}\n{\"outcome\":\"blocked\"}\n"
	if string(got) != want {
		t.Fatalf("file = %q; want %q", got, want)
	}
}

// I0028 S16: a file that cannot be opened is an error, never a silent drop.
func TestAppendReturnsOpenError(t *testing.T) {
	t.Parallel()
	name := filepath.Join(t.TempDir(), "missing-dir", "outcomes.jsonl")
	var f File
	if err := f.Append(name, []byte("{}\n")); err == nil {
		t.Fatal("Append() into a missing directory = nil; want an error")
	}
}
