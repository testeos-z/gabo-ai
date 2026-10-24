package worktree

import "context"

// Worktree is one entry of `git worktree list --porcelain`. Branch is the
// full ref (refs/heads/<name>) or "" when detached.
type Worktree struct {
	Path   string
	Head   string
	Branch string
	Bare   bool
}

// PullRequest is a PR of the branch being closed.
type PullRequest struct {
	Number  int
	HeadSHA string
}

// Git reads and changes the repository the command runs in. dir is the main
// worktree for repository-wide calls and the target worktree otherwise.
type Git interface {
	// Worktrees lists the registered worktrees; the first one is the main.
	Worktrees(ctx context.Context) ([]Worktree, error)
	// Toplevel returns the top of the worktree the command runs in, or "".
	Toplevel(ctx context.Context) (string, error)
	OriginURL(ctx context.Context, dir string) (string, error)
	// Fetch updates refs/remotes/origin/<branch>.
	Fetch(ctx context.Context, dir, branch string) error
	// Rev resolves ref to a commit id.
	Rev(ctx context.Context, dir, ref string) (string, error)
	// IsAncestor reports whether commit a is an ancestor of (or equal to) b;
	// an unknown commit is not an ancestor.
	IsAncestor(ctx context.Context, dir, a, b string) (bool, error)
	// OwnCommits reports whether the branch's reflog holds more than its creation.
	OwnCommits(ctx context.Context, dir, branch string) (bool, error)
	// Changed counts tracked changes and untracked files (ignored excluded).
	Changed(ctx context.Context, dir string) (int, error)
	// Ignored lists the ignored paths removing the worktree would delete:
	// files one by one, including inside ignored folders, except the
	// contents of node_modules, which is listed as the folder itself.
	Ignored(ctx context.Context, dir string) ([]string, error)
	// Unpushed returns whether HEAD has an upstream and how many commits it is ahead of it.
	Unpushed(ctx context.Context, dir string) (hasUpstream bool, ahead int, err error)
	// RemoveWorktree runs `git worktree remove` (never --force) and a prune.
	RemoveWorktree(ctx context.Context, dir, path string) error
	// DeleteBranch deletes a local branch with `git branch -d` (never -D).
	DeleteBranch(ctx context.Context, dir, branch string) error
}

// GitHub reads the branch's pull requests.
type GitHub interface {
	OpenPullRequests(ctx context.Context, branch string) ([]PullRequest, error)
	MergedPullRequests(ctx context.Context, branch, base string) ([]PullRequest, error)
}

// Paths resolves local paths.
type Paths interface {
	// Real returns the absolute symlink-free path, or "" when it does not exist.
	Real(path string) (string, error)
	// IsSymlink reports whether the last component of path is a symlink.
	IsSymlink(path string) (bool, error)
}
