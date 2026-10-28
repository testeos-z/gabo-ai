// Package osfs implements kernel.FileSystem on the local disk.
package osfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
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

// RemoveAll removes dir and everything below it; a missing dir is no error.
func (FS) RemoveAll(dir string) error { return os.RemoveAll(dir) }

// Lstat returns the mode of name without following a final symlink.
func (FS) Lstat(name string) (fs.FileMode, error) {
	info, err := os.Lstat(name)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// Stat returns the mode of name, following every symlink.
func (FS) Stat(name string) (fs.FileMode, error) {
	info, err := os.Stat(name)
	if err != nil {
		return 0, err
	}
	return info.Mode(), nil
}

// ReadLink returns the raw target of the symlink name.
func (FS) ReadLink(name string) (string, error) { return os.Readlink(name) }
