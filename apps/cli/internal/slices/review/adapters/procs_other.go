//go:build !linux

package adapters

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

var errNoProcFS = errors.New("process cleanup needs linux /proc and pidfds")

// ProcFS is unavailable outside linux; review ci refuses to run checks.
type ProcFS struct{}

// NewProcFS returns the unavailable implementation.
func NewProcFS(string, time.Duration) ProcFS { return ProcFS{} }

// Self describes the current process.
func (ProcFS) Self() review.Process {
	return review.Process{PID: os.Getpid(), PPID: os.Getppid(), UID: os.Getuid()}
}

// List always fails.
func (ProcFS) List() ([]review.Process, error) { return nil, errNoProcFS }

// Terminate always fails.
func (ProcFS) Terminate(context.Context, []int) error { return errNoProcFS }
