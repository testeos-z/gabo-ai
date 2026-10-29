package check

import "io/fs"

// UserFS reads user-level paths outside the project, by absolute OS path,
// without writing. Each error for a missing entry matches fs.ErrNotExist.
type UserFS interface {
	// Lstat returns the mode of name without following a final symlink.
	Lstat(name string) (fs.FileMode, error)
	// Stat returns the mode of name, following every symlink.
	Stat(name string) (fs.FileMode, error)
	// ReadLink returns the raw target of the symlink name.
	ReadLink(name string) (string, error)
}
