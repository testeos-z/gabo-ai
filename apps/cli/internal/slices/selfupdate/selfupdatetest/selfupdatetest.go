// Package selfupdatetest builds minisign keys, signatures and release
// archives for self-update tests, in minisign's file formats.
package selfupdatetest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"maps"
	"slices"
	"testing"

	"golang.org/x/crypto/blake2b"
)

// Signer is a minisign Ed25519 key pair.
type Signer struct {
	ID   [8]byte
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

// NewSigner generates a key pair with a random key id.
func NewSigner(t testing.TB) Signer {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s := Signer{priv: priv, pub: pub}
	if _, err := rand.Read(s.ID[:]); err != nil {
		t.Fatal(err)
	}
	return s
}

// PublicKey is the base64 line of minisign.pub.
func (s Signer) PublicKey() string {
	return base64.StdEncoding.EncodeToString(slices.Concat([]byte("Ed"), s.ID[:], s.pub))
}

// Sign returns a .minisig file over msg: alg "ED" signs the BLAKE2b-512 hash
// (minisign's default), "Ed" the message itself (legacy).
func (s Signer) Sign(msg []byte, alg, trusted string) []byte {
	signed := msg
	if alg == "ED" {
		h := blake2b.Sum512(msg)
		signed = h[:]
	}
	sig := ed25519.Sign(s.priv, signed)
	global := ed25519.Sign(s.priv, slices.Concat(sig, []byte(trusted)))
	return []byte("untrusted comment: signature from minisign secret key\n" +
		base64.StdEncoding.EncodeToString(slices.Concat([]byte(alg), s.ID[:], sig)) + "\n" +
		"trusted comment: " + trusted + "\n" +
		base64.StdEncoding.EncodeToString(global) + "\n")
}

// Tarball packs files (name → content) as a gzipped tar release archive.
func Tarball(t testing.TB, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(files[name])), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// SHA256 is the lowercase hex SHA-256 of b, as in checksums.txt.
func SHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Release returns the files of a signed release v<version> whose asset for
// goos/goarch holds binary, keyed by path below the release base URL
// ("v<version>/<name>").
func Release(t testing.TB, s Signer, version, goos, goarch, binary string) map[string][]byte {
	t.Helper()
	asset := "gabo_" + version + "_" + goos + "_" + goarch + ".tar.gz"
	archive := Tarball(t, map[string]string{"gabo": binary, "README.md": "readme"})
	sums := []byte(SHA256([]byte("other")) + "  gabo_" + version + "_plan9_mips.tar.gz\n" + SHA256(archive) + "  " + asset + "\n")
	dir := "v" + version + "/"
	return map[string][]byte{
		dir + asset:                                    archive,
		dir + "checksums.txt":                          sums,
		dir + "checksums.txt.minisig":                  s.Sign(sums, "ED", "timestamp:1 file:checksums.txt"),
		dir + "gabo_" + version + "_plan9_mips.tar.gz": []byte("other"),
	}
}
