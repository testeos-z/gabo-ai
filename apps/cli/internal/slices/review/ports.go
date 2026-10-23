package review

import (
	"context"
	"errors"
	"io"
)

// StatusState is a GitHub commit status state.
type StatusState string

// Commit status states.
const (
	StatusSuccess StatusState = "success"
	StatusFailure StatusState = "failure"
	StatusError   StatusState = "error"
)

// Status is one commit status posted on the reviewed head.
type Status struct {
	Context     string
	State       StatusState
	Description string
}

// GitHub reads the PR and posts the review as the owner's identity. It never
// approves: the verdict lives in the marker comment.
type GitHub interface {
	PullRequest(ctx context.Context, number int) (PullRequest, error)
	SetStatus(ctx context.Context, sha string, s Status) error
	Comment(ctx context.Context, number int, body string) error
	// Comments returns the PR's comments with their authors, oldest first.
	Comments(ctx context.Context, number int) ([]Comment, error)
	// Login returns the login the reviewer posts as; only its marker
	// comments count.
	Login(ctx context.Context) (string, error)
	Merge(ctx context.Context, number int, sha string) error
}

// Snapshot is the local evidence taken from a merged worktree.
type Snapshot struct {
	Files []string // changed files between the merge base and the head
	Diff  string   // unified diff between the merge base and the head
	Tree  []byte   // tar archive of the merged tree
}

// ErrDirtyClone means the reviewer's own state clone is not a clean checkout
// of the integration branch; the run refuses to start.
var ErrDirtyClone = errors.New("reviewer state clone is not a clean checkout of the integration branch")

// Git manages the reviewer's private clone and per-review worktrees.
// Worktrees are addressed by name; RemoveWorktree of a missing one is a no-op.
type Git interface {
	// Guard returns an error wrapping ErrDirtyClone unless the state clone
	// is missing (a fresh state) or a clean checkout of integration.
	Guard(ctx context.Context, integration string) error
	Prepare(ctx context.Context, number int, integration string) error
	AddWorktree(ctx context.Context, name, integration string) error
	// MergePinned merges sha into the worktree without committing.
	MergePinned(ctx context.Context, name, sha string) (conflict bool, err error)
	Snapshot(ctx context.Context, name, integration, sha string) (Snapshot, error)
	RemoveWorktree(ctx context.Context, name string) error
}

// CheckOutcome is the raw result of one check command.
type CheckOutcome struct {
	Passed bool
	Output []byte
}

// Checks runs untrusted commands, without GitHub credentials, in a workspace
// holding a copy of the merged tree. Cleanup of a missing workspace is a no-op.
type Checks interface {
	Prepare(ctx context.Context, workspace string, tree []byte) error
	Run(ctx context.Context, workspace string, argv []string) (CheckOutcome, error)
	Cleanup(ctx context.Context, workspace string) error
}

// ErrUnparseable is returned by an Engine whose output holds no findings document.
var ErrUnparseable = errors.New("engine output has no findings document")

// Engine runs the read-only review model on a prepared workspace.
type Engine interface {
	Review(ctx context.Context, workspace, prompt string) ([]Finding, error)
}

// Orca opens terminals and sends mailbox messages.
type Orca interface {
	CreateTerminal(ctx context.Context, title, command string) (string, error)
	Send(ctx context.Context, to, subject string) error
}

// Workspace stores the checks' copies of merged trees.
type Workspace interface {
	// Create replaces the named workspace with the tar archive read from r.
	Create(name string, r io.Reader) (dir string, err error)
	Dir(name string) (string, error)
	// Home replaces the workspace's private HOME with a fresh empty directory.
	Home(name string) (string, error)
	// Remove deletes the workspace and its HOME.
	Remove(name string) error
}

// Process is one entry of the process table. Cwd is empty when unreadable.
type Process struct {
	PID, PPID, UID int
	Cwd            string
}

// Processes reads the process table and stops processes.
type Processes interface {
	Self() Process
	List() ([]Process, error)
	// Terminate sends TERM, waits once for a bounded grace period, then KILLs
	// the survivors. Processes that are already gone are not an error.
	Terminate(ctx context.Context, pids []int) error
}

// Lock serializes reviews: a review's checks may only kill leftovers when no
// other review is running.
type Lock interface {
	Acquire(ctx context.Context) (release func(), err error)
}
