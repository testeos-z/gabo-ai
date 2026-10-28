// Package selfupdate updates the running gabo to exactly the version of a
// newer catalog stamp, from a minisign-signed release, when an interactive
// `gabo init` user agrees (TRD 9.4, D24).
package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
)

const (
	// ReleaseBaseURL holds one directory per release tag v<version> (TRD 9.4).
	ReleaseBaseURL = "https://github.com/testeos-z/gabo-ai/releases/download"
	// ReleasePublicKey is the minisign public key (the base64 line of
	// minisign.pub) of the gabo release signing key, held by the owner. It is
	// empty until that key exists: a build without a pinned key never
	// self-updates and prints the upgrade command instead.
	ReleasePublicKey = ""
	// OptOutEnv set to any value disables the self-update.
	OptOutEnv = "GABO_NO_SELF_UPDATE"
	// MaxDownload caps every downloaded file and the extracted binary.
	MaxDownload = 256 << 20

	binaryName = "gabo"
	sumsName   = "checksums.txt"
	sigName    = "checksums.txt.minisig"
)

// packageManagerPrefixes are install locations owned by a package manager
// (TRD 9.4): gabo never replaces a binary under them.
var packageManagerPrefixes = []string{"/opt/homebrew/", "/usr/local/Cellar/", "/home/linuxbrew/", "/nix/store/", "/usr/bin/", "/usr/sbin/", "/snap/"}

// Settings describe the platform, the pinned key and the run.
type Settings struct {
	BaseURL   string
	PublicKey string
	GOOS      string
	GOARCH    string
	// Interactive is true when stdin is a terminal.
	Interactive bool
	// CI and OptOut are true when CI or GABO_NO_SELF_UPDATE is set.
	CI     bool
	OptOut bool
	// Args re-execute init (args[0] included).
	Args []string
}

// UseCase runs one self-update decision.
type UseCase struct {
	rel Releases
	bin Binary
	con Console
	s   Settings
}

// NewUseCase wires the use case.
func NewUseCase(rel Releases, bin Binary, con Console, s Settings) *UseCase {
	return &UseCase{rel: rel, bin: bin, con: con, s: s}
}

// Update asks to update gabo from have to want and, on yes, replaces the
// running binary with the verified release binary and re-executes init.
// updated is false when nothing changed; lines then say why. updated is true
// only when the binary was replaced but the restart failed (err) or, in
// tests, when Exec returned.
func (u *UseCase) Update(ctx context.Context, want, have string) (lines []string, updated bool, err error) {
	switch {
	case u.s.CI:
		return []string{"self-update skipped: CI is set"}, false, nil
	case u.s.OptOut:
		return []string{"self-update skipped: " + OptOutEnv + " is set"}, false, nil
	case !u.s.Interactive:
		return []string{"self-update skipped: stdin is not a terminal"}, false, nil
	case strings.TrimSpace(u.s.PublicKey) == "":
		return []string{"self-update unavailable: this gabo build pins no release signing key"}, false, nil
	}
	key, err := parsePublicKey(u.s.PublicKey)
	if err != nil {
		return []string{"self-update unavailable: pinned release signing key: " + err.Error()}, false, nil
	}
	exe, err := u.bin.Executable()
	if err != nil {
		return []string{"self-update unavailable: cannot locate the running gabo: " + err.Error()}, false, nil
	}
	for _, p := range packageManagerPrefixes {
		if strings.HasPrefix(exe, p) {
			return []string{"installed by a package manager at " + exe + ": upgrade it to " + want + " with that package manager"}, false, nil
		}
	}
	if err := u.bin.Writable(filepath.Dir(exe)); err != nil {
		return []string{"self-update unavailable: cannot write " + filepath.Dir(exe) + ": " + err.Error()}, false, nil
	}
	answer, err := u.con.Ask("this repo needs gabo " + want + ", you have " + have + ". Update? This affects all your projects.")
	if a := strings.ToLower(strings.TrimSpace(answer)); err != nil || (a != "y" && a != "yes") {
		return []string{"not updated"}, false, nil
	}
	binary, err := u.fetch(ctx, key, want)
	if err != nil {
		return []string{"self-update failed: " + err.Error() + "; nothing changed"}, false, nil
	}
	if err := u.bin.Replace(exe, binary); err != nil {
		return []string{"self-update failed: replace " + exe + ": " + err.Error() + "; nothing changed"}, false, nil
	}
	u.con.Say("gabo updated " + have + " → " + want + "; restarting gabo init")
	// The restarted init must not offer another update if the release binary
	// reports an older version than its tag.
	if err := u.bin.Exec(exe, u.s.Args, []string{OptOutEnv + "=1"}); err != nil {
		return nil, true, fmt.Errorf("gabo updated to %s but could not restart: %v; rerun gabo init", want, err)
	}
	return nil, true, nil
}

// fetch downloads checksums.txt and its signature, verifies the signature
// against the pinned key, then downloads the asset, checks its SHA-256 and
// returns the gabo binary inside it.
func (u *UseCase) fetch(ctx context.Context, key publicKey, want string) ([]byte, error) {
	asset := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryName, want, u.s.GOOS, u.s.GOARCH)
	get := func(name string) ([]byte, error) {
		b, err := u.rel.Get(ctx, strings.TrimSuffix(u.s.BaseURL, "/")+"/v"+want+"/"+name)
		switch {
		case errors.Is(err, ErrNotFound):
			return nil, fmt.Errorf("release v%s has no %s", want, name)
		case err != nil:
			return nil, fmt.Errorf("download %s: %w", name, err)
		}
		return b, nil
	}
	sums, err := get(sumsName)
	if err != nil {
		return nil, err
	}
	sig, err := get(sigName)
	if err != nil {
		return nil, err
	}
	if err := key.verify(sums, sig); err != nil {
		return nil, fmt.Errorf("%s signature: %w", sumsName, err)
	}
	wantSum, err := checksumFor(sums, asset)
	if err != nil {
		return nil, err
	}
	archive, err := get(asset)
	if err != nil {
		return nil, err
	}
	if sha256.Sum256(archive) != wantSum {
		return nil, fmt.Errorf("%s SHA-256 does not match %s", asset, sumsName)
	}
	return extractBinary(archive, asset)
}

// checksumFor returns the SHA-256 of name from a sha256sum-style file.
func checksumFor(sums []byte, name string) ([32]byte, error) {
	var sum [32]byte
	found := false
	for _, line := range strings.Split(string(sums), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 || strings.TrimPrefix(f[1], "*") != name {
			continue
		}
		if found {
			return sum, fmt.Errorf("%s lists %s more than once", sumsName, name)
		}
		b, err := hex.DecodeString(f[0])
		if err != nil || len(b) != len(sum) {
			return sum, fmt.Errorf("%s has a malformed line for %s", sumsName, name)
		}
		sum, found = [32]byte(b), true
	}
	if !found {
		return sum, fmt.Errorf("%s has no line for %s", sumsName, name)
	}
	return sum, nil
}

// extractBinary returns the regular file "gabo" at the archive root.
func extractBinary(archive []byte, asset string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", asset, err)
	}
	tr := tar.NewReader(gz)
	var out []byte
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", asset, err)
		}
		if path.Clean(h.Name) != binaryName || h.Typeflag != tar.TypeReg {
			continue
		}
		if out != nil {
			return nil, fmt.Errorf("%s holds more than one %s binary", asset, binaryName)
		}
		b, err := io.ReadAll(io.LimitReader(tr, MaxDownload+1))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", asset, err)
		}
		if len(b) > MaxDownload {
			return nil, fmt.Errorf("%s: %s is larger than %d bytes", asset, binaryName, MaxDownload)
		}
		out = b
	}
	if out == nil {
		return nil, fmt.Errorf("%s holds no %s binary", asset, binaryName)
	}
	return out, nil
}
