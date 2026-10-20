package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

func TestCredentialReader(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pi-api-key"), []byte("k\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	read := CredentialReader(dir)
	if b, err := read(review.RuntimePi); err != nil || string(b) != "k\n" {
		t.Fatalf("read pi = %q, %v", b, err)
	}
	if _, err := read(review.RuntimeClaude); err == nil {
		t.Fatal("missing claude credential returned nil error")
	}
	if _, err := read("opencode"); err == nil {
		t.Fatal("unknown runtime returned nil error")
	}
}
