package osfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesParentsAndReadRoundTrips(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "a", "b", "SKILL.md")
	f := FS{}
	if _, err := f.ReadFile(p); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing ReadFile error = %v", err)
	}
	if err := f.WriteFile(p, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	b, err := f.ReadFile(p)
	if err != nil || string(b) != "hi" {
		t.Fatalf("ReadFile = %q, %v", b, err)
	}
	info, err := os.Stat(p)
	if err != nil || info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v, %v", info.Mode(), err)
	}
}
