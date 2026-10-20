package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileLockAcquireReleaseReacquire(t *testing.T) {
	t.Parallel()
	p := filepath.Join(t.TempDir(), "work", ".review.lock")
	l := NewFileLock(p)
	release, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(p); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("lock file %v, %v", info, err)
	}
	release()
	release2, err := l.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	release2()
}
