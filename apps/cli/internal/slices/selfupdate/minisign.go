package selfupdate

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/crypto/blake2b"
)

// publicKey is a minisign Ed25519 public key.
type publicKey struct {
	id  [8]byte
	key ed25519.PublicKey
}

// parsePublicKey reads the base64 line of a minisign.pub file, or the whole
// file with its untrusted comment.
func parsePublicKey(s string) (publicKey, error) {
	line := ""
	for _, l := range strings.Split(strings.TrimSpace(s), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "untrusted comment:") {
			line = l
		}
	}
	raw, err := base64.StdEncoding.DecodeString(line)
	if err != nil || len(raw) != 2+8+ed25519.PublicKeySize || string(raw[:2]) != "Ed" {
		return publicKey{}, errors.New("not a minisign Ed25519 public key")
	}
	return publicKey{id: [8]byte(raw[2:10]), key: ed25519.PublicKey(raw[10:])}, nil
}

// verify checks a .minisig file over msg: the signature (algorithm "ED",
// BLAKE2b-512 prehashed, minisign's default, or legacy "Ed") must come from
// this key, and the global signature must cover the trusted comment.
func (k publicKey) verify(msg, minisig []byte) error {
	lines := strings.Split(strings.ReplaceAll(string(minisig), "\r\n", "\n"), "\n")
	const trustedPrefix = "trusted comment: "
	if len(lines) < 4 || !strings.HasPrefix(lines[0], "untrusted comment:") || !strings.HasPrefix(lines[2], trustedPrefix) {
		return errors.New("malformed .minisig file")
	}
	sig, err := base64.StdEncoding.DecodeString(lines[1])
	if err != nil || len(sig) != 2+8+ed25519.SignatureSize {
		return errors.New("malformed signature line")
	}
	global, err := base64.StdEncoding.DecodeString(lines[3])
	if err != nil || len(global) != ed25519.SignatureSize {
		return errors.New("malformed global signature line")
	}
	alg, id, s := string(sig[:2]), sig[2:10], sig[10:]
	if !bytes.Equal(id, k.id[:]) {
		return fmt.Errorf("signed by key %s, not the pinned key %s", hex.EncodeToString(id), hex.EncodeToString(k.id[:]))
	}
	signed := msg
	switch alg {
	case "ED":
		h := blake2b.Sum512(msg)
		signed = h[:]
	case "Ed":
	default:
		return fmt.Errorf("unsupported algorithm %q", alg)
	}
	if !ed25519.Verify(k.key, signed, s) {
		return errors.New("does not verify against the pinned key")
	}
	if !ed25519.Verify(k.key, slices.Concat(s, []byte(lines[2][len(trustedPrefix):])), global) {
		return errors.New("trusted comment does not verify against the pinned key")
	}
	return nil
}
