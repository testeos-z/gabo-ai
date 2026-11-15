package initcmd

import (
	"context"
	"io/fs"
)

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

// Updater replaces the running gabo with exactly version want of a newer
// catalog stamp and re-executes init (TRD 9.4). updated is false when nothing
// changed, with lines that say why; when true the binary was replaced and
// err reports a failed restart.
type Updater interface {
	Update(ctx context.Context, want, have string) (lines []string, updated bool, err error)
}

// PiProbe reads whether a Pi subagent package is installed (intent 0016 S1,
// D1). It runs only `pi list` and a PATH lookup and writes nothing.
type PiProbe interface {
	// PiList returns the output of `pi list` run until ctx ends; ok is false
	// when pi is not on PATH, exits non-zero or the deadline passes.
	PiList(ctx context.Context) (out []byte, ok bool)
	// GentleShellOnPath reports whether gentle-shell is on PATH.
	GentleShellOnPath() bool
}
