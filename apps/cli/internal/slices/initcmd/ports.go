package initcmd

import "io/fs"

// Store reads and writes the project files init owns.
type Store interface {
	// ReadFile returns an error matching fs.ErrNotExist for a missing file.
	ReadFile(name string) ([]byte, error)
	// WriteFile creates missing parent directories and writes data.
	WriteFile(name string, data []byte) error
	// RemoveAll removes dir and everything below it; a missing dir is no error.
	RemoveAll(dir string) error
	// Lstat returns the mode of name without following a final symlink, or an
	// error matching fs.ErrNotExist.
	Lstat(name string) (fs.FileMode, error)
}
