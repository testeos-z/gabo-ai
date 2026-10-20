// Package osfs implements kernel.FileSystem on the local disk.
package osfs

import (
	"os"
	"path/filepath"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// FS is the local filesystem.
type FS struct{}

var _ kernel.FileSystem = FS{}

// ReadFile returns the content of name.
func (FS) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }

// WriteFile creates missing parents (0755) and writes data (0644).
func (FS) WriteFile(name string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(name, data, 0o644); err != nil {
		return err
	}
	return os.Chmod(name, 0o644)
}
