package selfupdate

import (
	"context"
	"errors"
)

// ErrNotFound marks a release file the server does not have (HTTP 404).
var ErrNotFound = errors.New("not found")

// Releases downloads release files.
type Releases interface {
	// Get returns the body of url, at most MaxDownload bytes; a missing file
	// returns an error matching ErrNotFound.
	Get(ctx context.Context, url string) ([]byte, error)
}

// Binary is the running gabo executable.
type Binary interface {
	// Executable returns the running binary's path with symlinks followed.
	Executable() (string, error)
	// Writable returns nil when the user can create files in dir.
	Writable(dir string) error
	// Replace writes data beside path and renames it over path (atomic on the
	// same filesystem); on error path is unchanged.
	Replace(path string, data []byte) error
	// Exec replaces the current process with path, args (args[0] included)
	// and the current environment plus env; it returns only on failure.
	Exec(path string, args, env []string) error
}

// Console talks to the user at the terminal.
type Console interface {
	// Ask prints the question and returns one answer line.
	Ask(question string) (string, error)
	// Say prints one line.
	Say(line string)
}
