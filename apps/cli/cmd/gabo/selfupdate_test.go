package main

import (
	"bytes"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate/adapters"
	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate/selfupdatetest"
)

const upgradeCmd = "go install github.com/testeos-z/gabo-ai/apps/cli/cmd/gabo@v9.0.0"

// tempBinary is the real filesystem adapter with the running executable
// replaced by a temporary file and the process restart recorded, so no test
// touches the installed gabo.
type tempBinary struct {
	adapters.OSBinary
	exe   string
	execs *[][]string
}

func (b tempBinary) Executable() (string, error) { return b.exe, nil }

func (b tempBinary) Exec(path string, args, env []string) error {
	*b.execs = append(*b.execs, slices.Concat([]string{path}, args, env))
	return nil
}

// updateCase is one `gabo init` run with a stamp of 9.0.0 on gabo 1.2.3
// against a fake release server.
type updateCase struct {
	t       *testing.T
	project string
	exe     string
	files   map[string][]byte
	key     string
	answer  string
	tty     bool
	env     map[string]string
	execs   [][]string
}

func newUpdateCase(t *testing.T) *updateCase {
	t.Helper()
	signer := selfupdatetest.NewSigner(t)
	project := t.TempDir()
	cfg := []byte("repo:\n  name: o/r\nruntimes:\n  coder: opencode\n  reviewer: opencode\n  assistant: opencode\n")
	if err := os.WriteFile(filepath.Join(project, "gabo.yaml"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := runInit(t, project, "1.2.3"); err != nil {
		t.Fatalf("restore: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(project, ".gabo", "VERSION"), []byte("9.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(t.TempDir(), "bin", "gabo")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return &updateCase{
		t: t, project: project, exe: exe, answer: "y\n", tty: true,
		files: selfupdatetest.Release(t, signer, "9.0.0", runtime.GOOS, runtime.GOARCH, "new binary"),
		key:   signer.PublicKey(),
		env:   map[string]string{},
	}
}

func (c *updateCase) asset() string {
	return "v9.0.0/gabo_9.0.0_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
}

// run executes gabo init and returns stdout and the error.
func (c *updateCase) run() (string, error) {
	t := c.t
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := c.files[strings.TrimPrefix(r.URL.Path, "/download/")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Chdir(c.project)
	var out bytes.Buffer
	root := newRootCommand(deps{version: "1.2.3", update: updateDeps{
		baseURL:     srv.URL + "/download",
		publicKey:   c.key,
		interactive: c.tty,
		in:          strings.NewReader(c.answer),
		out:         &out,
		args:        []string{"gabo", "init"},
		binary:      tempBinary{exe: c.exe, execs: &c.execs},
		releases:    adapters.HTTPReleases{Client: srv.Client()},
		getenv:      func(k string) string { return c.env[k] },
	}})
	root.SetOut(&out)
	root.SetArgs([]string{"init"})
	err := root.Execute()
	return out.String(), err
}

func TestInitSelfUpdateYesReplacesBinaryAndReexecutes(t *testing.T) {
	c := newUpdateCase(t)
	before := snapshot(t, c.project)
	out, err := c.run()
	if err != nil {
		t.Fatalf("err = %v\n%s", err, out)
	}
	for _, want := range []string{
		"this repo needs gabo 9.0.0, you have 1.2.3. Update? This affects all your projects.",
		"gabo updated 1.2.3 → 9.0.0; restarting gabo init",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
	if got, _ := os.ReadFile(c.exe); string(got) != "new binary" {
		t.Fatalf("binary = %q", got)
	}
	if entries, _ := os.ReadDir(filepath.Dir(c.exe)); len(entries) != 1 {
		t.Fatalf("left files beside the binary: %v", entries)
	}
	want := []string{c.exe, "gabo", "init", "GABO_NO_SELF_UPDATE=1"}
	if len(c.execs) != 1 || !slices.Equal(c.execs[0], want) {
		t.Fatalf("execs = %q", c.execs)
	}
	// The restarted init does the catalog work; this run changed nothing.
	if after := snapshot(t, c.project); !maps.Equal(before, after) {
		t.Fatal("project changed before the restart")
	}
}

// assertNothingChanged: binary and project unchanged, no restart, the upgrade
// command and exit 1, and stdout names the reason.
func (c *updateCase) assertNothingChanged(reason string) {
	t := c.t
	t.Helper()
	before := snapshot(t, c.project)
	out, err := c.run()
	if err == nil || !strings.Contains(err.Error(), upgradeCmd) {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if exitCode(&bytes.Buffer{}, err) != 1 {
		t.Fatalf("exit code != 1 for %v", err)
	}
	if !strings.Contains(out, reason) {
		t.Fatalf("output lacks %q:\n%s", reason, out)
	}
	if got, _ := os.ReadFile(c.exe); string(got) != "old binary" {
		t.Fatalf("binary = %q", got)
	}
	if entries, _ := os.ReadDir(filepath.Dir(c.exe)); len(entries) != 1 {
		t.Fatalf("left files beside the binary: %v", entries)
	}
	if len(c.execs) != 0 {
		t.Fatalf("restarted: %q", c.execs)
	}
	if after := snapshot(t, c.project); !maps.Equal(before, after) {
		t.Fatal("project changed")
	}
}

func TestInitSelfUpdateNoAnswerChangesNothing(t *testing.T) {
	for _, answer := range []string{"n\n", "\n", ""} {
		c := newUpdateCase(t)
		c.answer = answer
		c.assertNothingChanged("not updated")
	}
}

func TestInitSelfUpdateBadSignatureChangesNothing(t *testing.T) {
	c := newUpdateCase(t)
	other := selfupdatetest.NewSigner(t)
	c.files["v9.0.0/checksums.txt.minisig"] = other.Sign(c.files["v9.0.0/checksums.txt"], "ED", "forged")
	c.assertNothingChanged("self-update failed: checksums.txt signature")
}

func TestInitSelfUpdateWrongChecksumChangesNothing(t *testing.T) {
	c := newUpdateCase(t)
	c.files[c.asset()] = selfupdatetest.Tarball(t, map[string]string{"gabo": "evil binary"})
	c.assertNothingChanged("SHA-256 does not match checksums.txt")
}

func TestInitSelfUpdateMissingAssetChangesNothing(t *testing.T) {
	c := newUpdateCase(t)
	delete(c.files, c.asset())
	c.assertNothingChanged("self-update failed: release v9.0.0 has no gabo_9.0.0_")
}

func TestInitSelfUpdateNoNetworkChangesNothing(t *testing.T) {
	c := newUpdateCase(t)
	c.files = map[string][]byte{}
	c.assertNothingChanged("self-update failed: release v9.0.0 has no checksums.txt")
}

func TestInitSelfUpdateUnwritablePathChangesNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	c := newUpdateCase(t)
	dir := filepath.Dir(c.exe)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	c.assertNothingChanged("self-update unavailable: cannot write " + dir)
}

func TestInitSelfUpdatePackageManagerPathChangesNothing(t *testing.T) {
	c := newUpdateCase(t)
	c.exe = "/opt/homebrew/bin/gabo"
	before := snapshot(t, c.project)
	out, err := c.run()
	if err == nil || !strings.Contains(err.Error(), upgradeCmd) {
		t.Fatalf("err = %v\n%s", err, out)
	}
	if want := "installed by a package manager at /opt/homebrew/bin/gabo: upgrade it to 9.0.0 with that package manager"; !strings.Contains(out, want) {
		t.Fatalf("output lacks %q:\n%s", want, out)
	}
	if strings.Contains(out, "Update?") || len(c.execs) != 0 {
		t.Fatalf("asked or restarted:\n%s", out)
	}
	if after := snapshot(t, c.project); !maps.Equal(before, after) {
		t.Fatal("project changed")
	}
}

func TestInitSelfUpdateOptOutsNeverAsk(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		edit         func(*updateCase)
	}{
		{"non-TTY", "self-update skipped: stdin is not a terminal", func(c *updateCase) { c.tty = false }},
		{"CI", "self-update skipped: CI is set", func(c *updateCase) { c.env["CI"] = "true" }},
		{"GABO_NO_SELF_UPDATE", "self-update skipped: GABO_NO_SELF_UPDATE is set", func(c *updateCase) { c.env["GABO_NO_SELF_UPDATE"] = "1" }},
		{"no pinned key", "self-update unavailable: this gabo build pins no release signing key", func(c *updateCase) { c.key = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newUpdateCase(t)
			tc.edit(c)
			c.assertNothingChanged(tc.reason)
			out, _ := c.run()
			if strings.Contains(out, "Update?") {
				t.Fatalf("asked:\n%s", out)
			}
		})
	}
}
