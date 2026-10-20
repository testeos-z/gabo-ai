//go:build linux

package adapters

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

func writeProc(t *testing.T, root, pid, stat, uid string) {
	t.Helper()
	dir := filepath.Join(root, pid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "stat"), []byte(stat), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte("Name:\tx\nUid:\t"+uid+"\t"+uid+"\t"+uid+"\t"+uid+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestProcFSListParsesStatAndStatus(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeProc(t, root, "1", "1 (init) S 0 1 1", "0")
	writeProc(t, root, "42", "42 (evil) name) (x) S 7 42 42", "2001")
	if err := os.MkdirAll(filepath.Join(root, "self"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "99"), 0o755); err != nil { // vanished process
		t.Fatal(err)
	}
	got, err := NewProcFS(root, time.Second).List()
	if err != nil {
		t.Fatal(err)
	}
	want := []review.Process{{PID: 1, PPID: 0, UID: 0}, {PID: 42, PPID: 7, UID: 2001}}
	if !slices.Equal(got, want) {
		t.Fatalf("List() = %+v", got)
	}
}

func TestProcFSTerminateKillsAProcessIgnoringTERM(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("sh", "-c", `trap "" TERM; echo ready; exec sleep 60`)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if line, err := bufio.NewReader(stdout).ReadString('\n'); err != nil || line != "ready\n" {
		t.Fatalf("child not ready: %q, %v", line, err)
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	p := NewProcFS("/proc", 200*time.Millisecond)
	start := time.Now()
	if err := p.Terminate(context.Background(), []int{cmd.Process.Pid, 1 << 22}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		t.Fatal("process survived Terminate")
	}
	// TERM is ignored, so only the KILL after the grace period ends it.
	if took := time.Since(start); took < 150*time.Millisecond || took > 3*time.Second {
		t.Fatalf("terminated after %v, want about the 200ms grace", took)
	}
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); !ok || ws.Signal() != syscall.SIGKILL {
		t.Fatalf("state = %v", cmd.ProcessState)
	}
}

func TestProcFSSelf(t *testing.T) {
	t.Parallel()
	s := NewProcFS("/proc", time.Second).Self()
	if s.PID != os.Getpid() || s.PPID != os.Getppid() || s.UID != os.Getuid() {
		t.Fatalf("Self() = %+v", s)
	}
}

func TestProcFSTerminateReturnsEarlyWhenTERMSuffices(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	start := time.Now()
	if err := NewProcFS("/proc", 10*time.Second).Terminate(context.Background(), []int{cmd.Process.Pid}); err != nil {
		t.Fatal(err)
	}
	<-exited
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("waited the whole grace: %v", took)
	}
	if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); !ok || ws.Signal() != syscall.SIGTERM {
		t.Fatalf("state = %v", cmd.ProcessState)
	}
}
