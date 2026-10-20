package adapters

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// FileLock implements review.Lock with an exclusive flock. Acquire blocks in
// the kernel until the lock is free; it does not observe ctx.
type FileLock struct {
	path string
}

var _ review.Lock = FileLock{}

// NewFileLock locks the file at path (created 0600, parent 0700).
func NewFileLock(path string) FileLock { return FileLock{path: path} }

// Acquire takes the lock and returns its release function.
func (l FileLock) Acquire(context.Context) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	return func() {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		_ = f.Close()
	}, nil
}
