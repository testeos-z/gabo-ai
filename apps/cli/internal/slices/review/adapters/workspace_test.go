package adapters

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type entry struct {
	name, body, link string
	typ              byte
	mode             int64
}

func tarOf(t *testing.T, entries ...entry) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: e.mode, Size: int64(len(e.body)), Linkname: e.link}
		if e.typ != tar.TypeReg {
			h.Size = 0
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if e.typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func TestWorkspaceExtractsAndReplaces(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "work")
	ws := NewDirWorkspace(root)
	dir, err := ws.Create("pr-1-0123456", tarOf(t,
		entry{name: "src/", typ: tar.TypeDir, mode: 0o755},
		entry{name: "src/a.ts", body: "a", typ: tar.TypeReg, mode: 0o644},
		entry{name: "run.sh", body: "#!/bin/sh", typ: tar.TypeReg, mode: 0o755},
		entry{name: "link", link: "src/a.ts", typ: tar.TypeSymlink},
	))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "link")); err != nil || string(b) != "a" {
		t.Fatalf("link = %q, %v", b, err)
	}
	if info, err := os.Stat(filepath.Join(dir, "run.sh")); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("run.sh mode = %v, %v", info, err)
	}
	if info, err := os.Stat(root); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("root mode = %v, %v", info, err)
	}
	if _, err := ws.Create("pr-1-0123456", tarOf(t, entry{name: "b.ts", body: "b", typ: tar.TypeReg, mode: 0o644})); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "src")); !os.IsNotExist(err) {
		t.Fatalf("old content survived: %v", err)
	}
	if err := ws.Remove("pr-1-0123456"); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Dir("pr-1-0123456"); err == nil {
		t.Fatal("Dir() after Remove succeeded")
	}
	if err := ws.Remove("pr-1-0123456"); err != nil {
		t.Fatalf("Remove() missing = %v", err)
	}
}

func TestWorkspaceRejectsEscapes(t *testing.T) {
	t.Parallel()
	outside := t.TempDir()
	for name, tr := range map[string]*bytes.Buffer{
		"dotdot":   tarOf(t, entry{name: "../x", body: "x", typ: tar.TypeReg, mode: 0o644}),
		"absolute": tarOf(t, entry{name: filepath.Join(outside, "x"), body: "x", typ: tar.TypeReg, mode: 0o644}),
		"via symlink": tarOf(t,
			entry{name: "out", link: outside, typ: tar.TypeSymlink},
			entry{name: "out/x", body: "x", typ: tar.TypeReg, mode: 0o644}),
		"hardlink": tarOf(t, entry{name: "h", link: "/etc/passwd", typ: tar.TypeLink}),
	} {
		ws := NewDirWorkspace(filepath.Join(t.TempDir(), "work"))
		if _, err := ws.Create("pr-1-0123456", tr); err == nil {
			t.Errorf("%s: Create() error = nil", name)
		}
		if _, err := os.Stat(filepath.Join(outside, "x")); err == nil {
			t.Fatalf("%s: wrote outside the workspace", name)
		}
	}
}
