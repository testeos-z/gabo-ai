package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadBinary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "gbai")
	if err := os.WriteFile(file, []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := FileBinary{}.ReadBinary(file)
	if err != nil || string(b) != "bin" {
		t.Fatalf("ReadBinary() = %q, %v", b, err)
	}
	if _, err := (FileBinary{}).ReadBinary(dir); err == nil {
		t.Fatal("ReadBinary(dir) error = nil")
	}
}
