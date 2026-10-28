package selfupdate

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/slices/selfupdate/selfupdatetest"
)

const (
	base   = "https://releases.example/download"
	want   = "9.0.0"
	have   = "1.2.3"
	exe    = "/home/u/bin/gabo"
	asset  = "gabo_9.0.0_linux_amd64.tar.gz"
	prompt = "this repo needs gabo 9.0.0, you have 1.2.3. Update? This affects all your projects."
)

// release is a signed release v9.0.0 holding "new binary" for linux/amd64.
func release(t *testing.T, s selfupdatetest.Signer) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	for k, v := range selfupdatetest.Release(t, s, want, "linux", "amd64", "new binary") {
		files[base+"/"+k] = v
	}
	return files
}

func newSigner(t *testing.T) selfupdatetest.Signer { return selfupdatetest.NewSigner(t) }

func tarball(t *testing.T, files map[string]string) []byte { return selfupdatetest.Tarball(t, files) }

func sha(b []byte) string { return selfupdatetest.SHA256(b) }

type fakeReleases struct {
	files map[string][]byte
	err   error
	got   []string
}

func (f *fakeReleases) Get(_ context.Context, url string) ([]byte, error) {
	f.got = append(f.got, url)
	if f.err != nil {
		return nil, f.err
	}
	b, ok := f.files[url]
	if !ok {
		return nil, fmt.Errorf("GET %s: 404: %w", url, ErrNotFound)
	}
	return b, nil
}

type fakeBinary struct {
	exe        string
	exeErr     error
	writeErr   error
	replaceErr error
	execErr    error
	replaced   map[string]string
	execs      [][]string
}

func (b *fakeBinary) Executable() (string, error) { return b.exe, b.exeErr }
func (b *fakeBinary) Writable(string) error       { return b.writeErr }
func (b *fakeBinary) Replace(path string, data []byte) error {
	if b.replaceErr != nil {
		return b.replaceErr
	}
	if b.replaced == nil {
		b.replaced = map[string]string{}
	}
	b.replaced[path] = string(data)
	return nil
}
func (b *fakeBinary) Exec(path string, args, env []string) error {
	b.execs = append(b.execs, slices.Concat([]string{path}, args, []string{"|"}, env))
	return b.execErr
}

type fakeConsole struct {
	answer string
	err    error
	asked  []string
	said   []string
}

func (c *fakeConsole) Ask(q string) (string, error) {
	c.asked = append(c.asked, q)
	return c.answer, c.err
}
func (c *fakeConsole) Say(line string) { c.said = append(c.said, line) }

type fixture struct {
	rel *fakeReleases
	bin *fakeBinary
	con *fakeConsole
	s   Settings
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	sg := newSigner(t)
	return &fixture{
		rel: &fakeReleases{files: release(t, sg)},
		bin: &fakeBinary{exe: exe},
		con: &fakeConsole{answer: "y"},
		s: Settings{
			BaseURL: base, PublicKey: sg.PublicKey(), GOOS: "linux", GOARCH: "amd64",
			Interactive: true, Args: []string{"gabo", "init"},
		},
	}
}

func (f *fixture) run() ([]string, bool, error) {
	return NewUseCase(f.rel, f.bin, f.con, f.s).Update(context.Background(), want, have)
}

func TestYesReplacesBinaryAndReexecutesInit(t *testing.T) {
	for _, answer := range []string{"y", "Y", "yes", " YES "} {
		t.Run(answer, func(t *testing.T) {
			f := newFixture(t)
			f.con.answer = answer
			lines, updated, err := f.run()
			if err != nil || !updated {
				t.Fatalf("updated=%v err=%v lines=%q", updated, err, lines)
			}
			if !slices.Equal(f.con.asked, []string{prompt}) {
				t.Fatalf("asked %q", f.con.asked)
			}
			if got := f.bin.replaced[exe]; got != "new binary" || len(f.bin.replaced) != 1 {
				t.Fatalf("replaced = %q", f.bin.replaced)
			}
			wantExec := []string{exe, "gabo", "init", "|", "GABO_NO_SELF_UPDATE=1"}
			if len(f.bin.execs) != 1 || !slices.Equal(f.bin.execs[0], wantExec) {
				t.Fatalf("execs = %q", f.bin.execs)
			}
			if !slices.Contains(f.con.said, "gabo updated 1.2.3 → 9.0.0; restarting gabo init") {
				t.Fatalf("said %q", f.con.said)
			}
		})
	}
}

func TestLegacyMinisignSignatureVerifies(t *testing.T) {
	f := newFixture(t)
	sg := newSigner(t)
	f.s.PublicKey = sg.PublicKey()
	sums := f.rel.files[base+"/v9.0.0/checksums.txt"]
	f.rel.files[base+"/v9.0.0/checksums.txt.minisig"] = sg.Sign(sums, "Ed", "legacy")
	if lines, updated, err := f.run(); err != nil || !updated {
		t.Fatalf("updated=%v err=%v lines=%q", updated, err, lines)
	}
}

// assertUnchanged: nothing replaced, nothing executed, not updated, and the
// lines name the reason.
func assertUnchanged(t *testing.T, f *fixture, reason string) {
	t.Helper()
	lines, updated, err := f.run()
	if updated || err != nil {
		t.Fatalf("updated=%v err=%v", updated, err)
	}
	if len(f.bin.replaced) != 0 || len(f.bin.execs) != 0 {
		t.Fatalf("changed the binary: replaced=%q execs=%q", f.bin.replaced, f.bin.execs)
	}
	if got := strings.Join(lines, "\n"); !strings.Contains(got, reason) {
		t.Fatalf("lines = %q, want %q", got, reason)
	}
}

func TestNoOrEmptyAnswerChangesNothing(t *testing.T) {
	for _, tc := range []struct {
		answer string
		err    error
	}{{answer: "n"}, {answer: "no"}, {answer: ""}, {answer: "maybe"}, {err: io.EOF}} {
		t.Run(fmt.Sprintf("%q %v", tc.answer, tc.err), func(t *testing.T) {
			f := newFixture(t)
			f.con.answer, f.con.err = tc.answer, tc.err
			assertUnchanged(t, f, "not updated")
			if len(f.rel.got) != 0 {
				t.Fatalf("downloaded %q after no", f.rel.got)
			}
		})
	}
}

// Every refusal before the question asks nothing and downloads nothing.
func TestRefusedBeforeAskingChangesNothing(t *testing.T) {
	for _, tc := range []struct {
		name, reason string
		edit         func(*fixture)
	}{
		{"CI set", "self-update skipped: CI is set", func(f *fixture) { f.s.CI = true }},
		{"opt-out", "self-update skipped: GABO_NO_SELF_UPDATE is set", func(f *fixture) { f.s.OptOut = true }},
		{"non-TTY", "self-update skipped: stdin is not a terminal", func(f *fixture) { f.s.Interactive = false }},
		{"no pinned key", "self-update unavailable: this gabo build pins no release signing key", func(f *fixture) { f.s.PublicKey = "" }},
		{"malformed pinned key", "self-update unavailable: pinned release signing key", func(f *fixture) { f.s.PublicKey = "not base64!" }},
		{"no executable", "self-update unavailable: cannot locate the running gabo", func(f *fixture) { f.bin.exeErr = errors.New("boom") }},
		{"homebrew", "installed by a package manager at /opt/homebrew/bin/gabo: upgrade it to 9.0.0 with that package manager", func(f *fixture) { f.bin.exe = "/opt/homebrew/bin/gabo" }},
		{"cellar", "installed by a package manager at /usr/local/Cellar/gabo/1.2.3/bin/gabo", func(f *fixture) { f.bin.exe = "/usr/local/Cellar/gabo/1.2.3/bin/gabo" }},
		{"linuxbrew", "installed by a package manager", func(f *fixture) { f.bin.exe = "/home/linuxbrew/.linuxbrew/bin/gabo" }},
		{"nix", "installed by a package manager", func(f *fixture) { f.bin.exe = "/nix/store/abc-gabo/bin/gabo" }},
		{"usr bin", "installed by a package manager", func(f *fixture) { f.bin.exe = "/usr/bin/gabo" }},
		{"usr sbin", "installed by a package manager", func(f *fixture) { f.bin.exe = "/usr/sbin/gabo" }},
		{"snap", "installed by a package manager", func(f *fixture) { f.bin.exe = "/snap/gabo/1/bin/gabo" }},
		{"unwritable", "self-update unavailable: cannot write /home/u/bin: permission denied", func(f *fixture) { f.bin.writeErr = errors.New("permission denied") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.edit(f)
			assertUnchanged(t, f, tc.reason)
			if len(f.con.asked) != 0 || len(f.rel.got) != 0 {
				t.Fatalf("asked %q, downloaded %q", f.con.asked, f.rel.got)
			}
		})
	}
}

// Every failure after yes changes nothing.
func TestFailedUpdateChangesNothing(t *testing.T) {
	sums := base + "/v9.0.0/checksums.txt"
	sig := sums + ".minisig"
	tar := base + "/v9.0.0/" + asset
	for _, tc := range []struct {
		name, reason string
		edit         func(*testing.T, *fixture)
	}{
		{"no network", "self-update failed: download checksums.txt: dial tcp: no route", func(_ *testing.T, f *fixture) { f.rel.err = errors.New("dial tcp: no route") }},
		{"missing asset", "self-update failed: release v9.0.0 has no " + asset, func(_ *testing.T, f *fixture) { delete(f.rel.files, tar) }},
		{"missing checksums", "self-update failed: release v9.0.0 has no checksums.txt", func(_ *testing.T, f *fixture) { delete(f.rel.files, sums) }},
		{"missing signature", "self-update failed: release v9.0.0 has no checksums.txt.minisig", func(_ *testing.T, f *fixture) { delete(f.rel.files, sig) }},
		{"signed by another key", "self-update failed: checksums.txt signature: signed by key", func(t *testing.T, f *fixture) {
			f.rel.files[sig] = newSigner(t).Sign(f.rel.files[sums], "ED", "x")
		}},
		{"same key id, other key", "self-update failed: checksums.txt signature: does not verify", func(t *testing.T, f *fixture) {
			other := newSigner(t)
			other.ID = keyID(t, f.s.PublicKey)
			f.rel.files[sig] = other.Sign(f.rel.files[sums], "ED", "x")
		}},
		{"tampered checksums", "self-update failed: checksums.txt signature: does not verify", func(_ *testing.T, f *fixture) {
			f.rel.files[sums] = append(f.rel.files[sums], "deadbeef  extra\n"...)
		}},
		{"tampered trusted comment", "self-update failed: checksums.txt signature: trusted comment does not verify", func(_ *testing.T, f *fixture) {
			f.rel.files[sig] = bytes.Replace(f.rel.files[sig], []byte("timestamp:1"), []byte("timestamp:2"), 1)
		}},
		{"unknown algorithm", "self-update failed: checksums.txt signature: unsupported algorithm", func(_ *testing.T, f *fixture) {
			f.rel.files[sig] = resign(f.rel.files[sig], "Xx")
		}},
		{"malformed signature", "self-update failed: checksums.txt signature: malformed", func(_ *testing.T, f *fixture) { f.rel.files[sig] = []byte("garbage\n") }},
		{"wrong checksum", "self-update failed: " + asset + " SHA-256 does not match checksums.txt", func(t *testing.T, f *fixture) {
			f.rel.files[tar] = tarball(t, map[string]string{"gabo": "evil binary"})
		}},
		{"no checksum line", "self-update failed: checksums.txt has no line for " + asset, func(t *testing.T, f *fixture) {
			signed(t, f, []byte(sha([]byte("x"))+"  other.tar.gz\n"))
		}},
		{"duplicate checksum line", "self-update failed: checksums.txt lists " + asset + " more than once", func(t *testing.T, f *fixture) {
			h := sha(f.rel.files[tar])
			signed(t, f, []byte(h+"  "+asset+"\n"+h+"  "+asset+"\n"))
		}},
		{"archive without gabo", "self-update failed: " + asset + " holds no gabo binary", func(t *testing.T, f *fixture) {
			archive := tarball(t, map[string]string{"bin/other": "x"})
			f.rel.files[tar] = archive
			signed(t, f, []byte(sha(archive)+"  "+asset+"\n"))
		}},
		{"replace fails", "self-update failed: replace " + exe + ": disk full", func(_ *testing.T, f *fixture) { f.bin.replaceErr = errors.New("disk full") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.edit(t, f)
			assertUnchanged(t, f, tc.reason)
			if !slices.Equal(f.con.asked, []string{prompt}) {
				t.Fatalf("asked %q", f.con.asked)
			}
		})
	}
}

// signed replaces checksums.txt with sums, signed by a fresh pinned key.
func signed(t *testing.T, f *fixture, sums []byte) {
	t.Helper()
	sg := newSigner(t)
	f.s.PublicKey = sg.PublicKey()
	f.rel.files[base+"/v9.0.0/checksums.txt"] = sums
	f.rel.files[base+"/v9.0.0/checksums.txt.minisig"] = sg.Sign(sums, "ED", "x")
}

func keyID(t *testing.T, pub string) [8]byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(pub)
	if err != nil {
		t.Fatal(err)
	}
	return [8]byte(b[2:10])
}

// resign swaps the algorithm bytes of a .minisig signature line.
func resign(sig []byte, alg string) []byte {
	lines := strings.Split(string(sig), "\n")
	raw, _ := base64.StdEncoding.DecodeString(lines[1])
	copy(raw, alg)
	lines[1] = base64.StdEncoding.EncodeToString(raw)
	return []byte(strings.Join(lines, "\n"))
}

// A restart that fails after the replacement reports the update and the error.
func TestRestartFailureReportsUpdated(t *testing.T) {
	f := newFixture(t)
	f.bin.execErr = errors.New("exec unsupported")
	lines, updated, err := f.run()
	if !updated || err == nil || !strings.Contains(err.Error(), "gabo updated to 9.0.0 but could not restart: exec unsupported; rerun gabo init") {
		t.Fatalf("updated=%v err=%v lines=%q", updated, err, lines)
	}
}

func TestDownloadsExactlyTheStampVersionForThisPlatform(t *testing.T) {
	f := newFixture(t)
	f.s.GOOS, f.s.GOARCH = "plan9", "mips"
	_, _, _ = f.run()
	want := []string{base + "/v9.0.0/checksums.txt", base + "/v9.0.0/checksums.txt.minisig", base + "/v9.0.0/gabo_9.0.0_plan9_mips.tar.gz"}
	if !slices.Equal(f.rel.got, want) {
		t.Fatalf("downloaded %q, want %q", f.rel.got, want)
	}
}

func TestPinnedKeyAcceptsMinisignPubFile(t *testing.T) {
	sg := newSigner(t)
	file := "untrusted comment: minisign public key ABC\n" + sg.PublicKey() + "\n"
	if _, err := parsePublicKey(file); err != nil {
		t.Fatal(err)
	}
	if _, err := parsePublicKey(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("short key accepted")
	}
}
