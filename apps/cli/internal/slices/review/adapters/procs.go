//go:build linux

package adapters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// ProcFS implements review.Processes with /proc and pidfds, so a signal can
// never hit a recycled pid.
type ProcFS struct {
	root  string
	grace time.Duration
}

var _ review.Processes = ProcFS{}

// NewProcFS reads the process table under root (normally /proc); grace is the
// single bounded wait between TERM and KILL.
func NewProcFS(root string, grace time.Duration) ProcFS { return ProcFS{root: root, grace: grace} }

// Self describes the current process.
func (ProcFS) Self() review.Process {
	return review.Process{PID: os.Getpid(), PPID: os.Getppid(), UID: os.Getuid()}
}

// List returns every readable process in pid order; vanished ones are skipped.
func (p ProcFS) List() ([]review.Process, error) {
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return nil, err
	}
	var out []review.Process
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		if proc, ok := p.read(pid); ok {
			out = append(out, proc)
		}
	}
	slices.SortFunc(out, func(a, b review.Process) int { return a.PID - b.PID })
	return out, nil
}

func (p ProcFS) read(pid int) (review.Process, bool) {
	dir := filepath.Join(p.root, strconv.Itoa(pid))
	stat, err := os.ReadFile(filepath.Join(dir, "stat"))
	if err != nil {
		return review.Process{}, false
	}
	// The command name may hold spaces and parentheses; fields follow the last ')'.
	i := strings.LastIndexByte(string(stat), ')')
	fields := strings.Fields(string(stat[i+1:]))
	if i < 0 || len(fields) < 2 {
		return review.Process{}, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return review.Process{}, false
	}
	status, err := os.ReadFile(filepath.Join(dir, "status"))
	if err != nil {
		return review.Process{}, false
	}
	for _, line := range strings.Split(string(status), "\n") {
		if rest, ok := strings.CutPrefix(line, "Uid:"); ok {
			f := strings.Fields(rest)
			if len(f) == 0 {
				break
			}
			uid, err := strconv.Atoi(f[0])
			if err != nil {
				break
			}
			// cwd is readable only for our own processes; empty otherwise.
			cwd, _ := os.Readlink(filepath.Join(dir, "cwd"))
			return review.Process{PID: pid, PPID: ppid, UID: uid, Cwd: cwd}, true
		}
	}
	return review.Process{}, false
}

// Terminate sends TERM through pidfds, waits once until every target exited
// or the grace period ends, then KILLs the survivors.
func (p ProcFS) Terminate(ctx context.Context, pids []int) error {
	var errs []error
	var fds []int
	defer func() {
		for _, fd := range fds {
			_ = unix.Close(fd)
		}
	}()
	for _, pid := range pids {
		fd, err := unix.PidfdOpen(pid, 0)
		if errors.Is(err, unix.ESRCH) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("pidfd %d: %w", pid, err))
			continue
		}
		fds = append(fds, fd)
		if err := unix.PidfdSendSignal(fd, unix.SIGTERM, nil, 0); err != nil && !errors.Is(err, unix.ESRCH) {
			errs = append(errs, fmt.Errorf("TERM %d: %w", pid, err))
		}
	}
	pending := waitExit(ctx, fds, time.Now().Add(p.grace))
	for _, fd := range pending {
		if err := unix.PidfdSendSignal(fd, unix.SIGKILL, nil, 0); err != nil && !errors.Is(err, unix.ESRCH) {
			errs = append(errs, fmt.Errorf("KILL: %w", err))
		}
	}
	return errors.Join(errs...)
}

// waitExit blocks on the pidfds until all exited or deadline; it returns the
// fds of processes still alive.
func waitExit(ctx context.Context, fds []int, deadline time.Time) []int {
	pending := append([]int(nil), fds...)
	for len(pending) > 0 && ctx.Err() == nil {
		ms := time.Until(deadline).Milliseconds()
		if ms <= 0 {
			break
		}
		poll := make([]unix.PollFd, len(pending))
		for i, fd := range pending {
			poll[i] = unix.PollFd{Fd: int32(fd), Events: unix.POLLIN}
		}
		n, err := unix.Poll(poll, int(ms))
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil || n == 0 {
			break
		}
		alive := pending[:0]
		for i, fd := range pending {
			if poll[i].Revents == 0 {
				alive = append(alive, fd)
			}
		}
		pending = alive
	}
	return pending
}
