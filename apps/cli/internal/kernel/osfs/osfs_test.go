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

func TestRemoveAllDeletesTreeAndIgnoresMissing(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "skills")
	f := FS{}
	if err := f.WriteFile(filepath.Join(dir, "a", "SKILL.md"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := f.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("dir still there: %v", err)
	}
	if err := f.RemoveAll(dir); err != nil {
		t.Fatalf("missing dir: %v", err)
	}
}

func TestStatFollowsLinksAndReadLinkReturnsTheRawTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f := FS{}
	if err := f.WriteFile(filepath.Join(dir, "real", "s.sh"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink("real", link); err != nil {
		t.Fatal(err)
	}
	if m, err := f.Stat(filepath.Join(link, "s.sh")); err != nil || !m.IsRegular() {
		t.Fatalf("Stat through link = %v, %v", m, err)
	}
	if m, err := f.Lstat(link); err != nil || m&fs.ModeSymlink == 0 {
		t.Fatalf("Lstat = %v, %v", m, err)
	}
	if got, err := f.ReadLink(link); err != nil || got != "real" {
		t.Fatalf("ReadLink = %q, %v", got, err)
	}
	if _, err := f.Stat(filepath.Join(dir, "none")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing Stat error = %v", err)
	}
}
