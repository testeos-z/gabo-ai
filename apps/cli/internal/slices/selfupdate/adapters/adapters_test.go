package adapters

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate"
)

func TestHTTPReleasesGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte("body"))
		case "/redirect":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/boom":
			http.Error(w, "nope", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	rel := HTTPReleases{Client: srv.Client()}
	for _, p := range []string{"/ok", "/redirect"} {
		if b, err := rel.Get(context.Background(), srv.URL+p); err != nil || string(b) != "body" {
			t.Fatalf("%s: %q %v", p, b, err)
		}
	}
	if _, err := rel.Get(context.Background(), srv.URL+"/missing"); !errors.Is(err, selfupdate.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if _, err := rel.Get(context.Background(), srv.URL+"/boom"); err == nil || errors.Is(err, selfupdate.ErrNotFound) || !strings.Contains(err.Error(), "500") {
		t.Fatalf("500: %v", err)
	}
	srv.Close()
	if _, err := rel.Get(context.Background(), srv.URL+"/ok"); err == nil {
		t.Fatal("no network accepted")
	}
}

func TestHTTPReleasesRefusesOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 11))
	}))
	defer srv.Close()
	if _, err := (HTTPReleases{Client: srv.Client(), Max: 10}).Get(context.Background(), srv.URL); err == nil || !strings.Contains(err.Error(), "larger than 10 bytes") {
		t.Fatalf("err = %v", err)
	}
}

func TestBinaryReplaceWritesBesideAndRenames(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "gabo")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	var b OSBinary
	if err := b.Writable(dir); err != nil {
		t.Fatal(err)
	}
	if err := b.Replace(exe, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(exe)
	if err != nil || string(got) != "new" {
		t.Fatalf("exe = %q %v", got, err)
	}
	if fi, _ := os.Stat(exe); fi.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v", fi.Mode())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Fatalf("left temp files: %v", entries)
	}
}

func TestBinaryUnwritableDirChangesNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "gabo")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	var b OSBinary
	if err := b.Writable(dir); err == nil {
		t.Fatal("unwritable dir reported writable")
	}
	if err := b.Replace(exe, []byte("new")); err == nil {
		t.Fatal("replace in unwritable dir succeeded")
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Fatalf("exe = %q", got)
	}
}

func TestBinaryExecutableFollowsSymlinks(t *testing.T) {
	exe, err := (OSBinary{}).Executable()
	if err != nil || !filepath.IsAbs(exe) {
		t.Fatalf("exe = %q %v", exe, err)
	}
	if real, _ := filepath.EvalSymlinks(exe); real != exe {
		t.Fatalf("exe %q is not resolved (%q)", exe, real)
	}
}

func TestConsoleAskAndSay(t *testing.T) {
	var out bytes.Buffer
	c := Console{In: bufio.NewReader(strings.NewReader("yes\nno\n")), Out: &out}
	if a, err := c.Ask("Update?"); err != nil || a != "yes" {
		t.Fatalf("answer = %q %v", a, err)
	}
	c.Say("done")
	if got := out.String(); got != "Update? [y/N] done\n" {
		t.Fatalf("out = %q", got)
	}
	empty := Console{In: bufio.NewReader(strings.NewReader("")), Out: &out}
	if _, err := empty.Ask("q"); err == nil {
		t.Fatal("EOF without answer accepted")
	}
}

func TestIsTerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "f")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if IsTerminal(f) {
		t.Fatal("regular file is a terminal")
	}
	// /dev/null is a character device but not a terminal (`gabo init </dev/null`).
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if IsTerminal(null) {
		t.Fatal(os.DevNull + " is a terminal")
	}
}
