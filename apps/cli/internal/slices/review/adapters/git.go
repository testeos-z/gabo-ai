package adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// Network git calls authenticate through gh, so no credential is ever stored
// in the clone's config.
var credentialFlags = []string{"-c", "credential.helper=", "-c", "credential.helper=!gh auth git-credential"}

// Git implements review.Git under the reviewer's own state dir: a clone
// checked out on the integration branch at <stateDir>/clone and per-review
// worktrees at <stateDir>/worktrees/<name>.
type Git struct {
	runner    kernel.Runner
	repo      string
	stateDir  string
	removeAll func(string) error
	lstat     func(string) (os.FileInfo, error)
}

var _ review.Git = (*Git)(nil)

// NewGit returns a git adapter. removeAll defaults to os.RemoveAll.
func NewGit(r kernel.Runner, repo, stateDir string, removeAll func(string) error) *Git {
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	return &Git{runner: r, repo: repo, stateDir: stateDir, removeAll: removeAll, lstat: os.Lstat}
}

func (g *Git) clone() string { return path.Join(g.stateDir, "clone") }

func (g *Git) worktree(name string) (string, error) {
	if err := review.ValidateWorkspace(name); err != nil {
		return "", err
	}
	return path.Join(g.stateDir, "worktrees", name), nil
}

// exists reports whether p exists; errors other than not-exist are returned.
func (g *Git) exists(p string) (bool, error) {
	_, err := g.lstat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Guard refuses (ErrDirtyClone) a state clone that is not a clean checkout
// of the integration branch: not its own repository, not on the branch, or
// with any tracked, untracked or ignored change. A missing clone is a fresh
// state that Prepare clones.
func (g *Git) Guard(ctx context.Context, integration string) error {
	clone := g.clone()
	ok, err := g.exists(clone)
	if err != nil {
		return fmt.Errorf("stat state clone: %w", err)
	}
	if !ok {
		return nil
	}
	// --git-dir is ".git" only at the top of a non-bare repository; a plain
	// directory inside another repository resolves to that repository.
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", clone, "rev-parse", "--git-dir"}})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 || strings.TrimSpace(string(res.Stdout)) != ".git" {
		return fmt.Errorf("%w: %s is not its own git checkout", review.ErrDirtyClone, clone)
	}
	res, err = g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", clone, "symbolic-ref", "--quiet", "--short", "HEAD"}})
	if err != nil {
		return err
	}
	if branch := strings.TrimSpace(string(res.Stdout)); res.ExitCode != 0 || branch != integration {
		return fmt.Errorf("%w: %s is on %q, not %q", review.ErrDirtyClone, clone, branch, integration)
	}
	status, err := g.git(ctx, "-C", clone, "status", "--porcelain=v1", "--untracked-files=all", "--ignored")
	if err != nil {
		return err
	}
	if changed := strings.TrimSpace(string(status.Stdout)); changed != "" {
		return fmt.Errorf("%w: %s has %d changed paths", review.ErrDirtyClone, clone, len(strings.Split(changed, "\n")))
	}
	return nil
}

func (g *Git) git(ctx context.Context, args ...string) (kernel.Result, error) {
	return run(ctx, g.runner, kernel.Command{Name: "git", Args: args})
}

// Prepare clones the repository on the integration branch when missing,
// fetches the integration branch and the PR head, and resets the checkout
// to origin so the clone stays a clean mirror of the integration branch.
func (g *Git) Prepare(ctx context.Context, n int, integration string) error {
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", g.clone(), "rev-parse", "--git-dir"}})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		args := append(slices.Clone(credentialFlags), "clone", "--branch", integration, "https://github.com/"+g.repo+".git", g.clone())
		if _, err := g.git(ctx, args...); err != nil {
			return fmt.Errorf("clone: %w", err)
		}
	}
	args := append(slices.Clone(credentialFlags), "-C", g.clone(), "fetch", "--no-tags", "--prune", "origin",
		"+refs/heads/"+integration+":refs/remotes/origin/"+integration,
		"+refs/pull/"+strconv.Itoa(n)+"/head:refs/remotes/origin/pr/"+strconv.Itoa(n))
	if _, err := g.git(ctx, args...); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	// The clone is the reviewer's own mirror and Guard has already refused
	// any local change, so it follows origin with a hard reset: a force-pushed
	// (rewritten) integration branch cannot be fast-forwarded and would fail
	// every later review.
	if _, err := g.git(ctx, "-C", g.clone(), "-c", "core.hooksPath=/dev/null", "reset", "--hard", "--quiet", "refs/remotes/origin/"+integration); err != nil {
		return fmt.Errorf("sync %s: %w", integration, err)
	}
	return nil
}

// AddWorktree creates a detached worktree at origin/<integration>. A worktree
// of the same name left by a --keep-worktree run (or a crash) is removed
// first, so a rerun for the same PR and sha does not collide with it; on a
// fresh state nothing is removed. The name is validated and resolved under
// the reviewer's own worktrees dir. A registration whose directory is gone
// is cleared by the prune.
func (g *Git) AddWorktree(ctx context.Context, name, integration string) error {
	wt, err := g.worktree(name)
	if err != nil {
		return err
	}
	stale, err := g.exists(wt)
	if err != nil {
		return fmt.Errorf("stat worktree %s: %w", name, err)
	}
	if stale {
		if err := g.RemoveWorktree(ctx, name); err != nil {
			return fmt.Errorf("remove stale worktree %s: %w", name, err)
		}
	}
	if _, err := g.git(ctx, "-C", g.clone(), "worktree", "prune"); err != nil {
		return err
	}
	_, err = g.git(ctx, "-C", g.clone(), "worktree", "add", "--detach", wt, "refs/remotes/origin/"+integration)
	return err
}

// MergePinned merges sha with --no-ff --no-commit; unmerged paths mean a conflict.
func (g *Git) MergePinned(ctx context.Context, name, sha string) (bool, error) {
	wt, err := g.worktree(name)
	if err != nil {
		return false, err
	}
	if _, err := g.git(ctx, "-C", wt, "cat-file", "-e", sha+"^{commit}"); err != nil {
		return false, fmt.Errorf("pinned head not fetched: %w", err)
	}
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{
		"-C", wt, "-c", "core.hooksPath=/dev/null", "-c", "user.name=gbai reviewer", "-c", "user.email=gbai-reviewer@localhost",
		"merge", "--no-ff", "--no-commit", "--no-edit", sha,
	}})
	if err != nil {
		return false, err
	}
	if res.ExitCode == 0 {
		return false, nil
	}
	unmerged, err := g.git(ctx, "-C", wt, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false, err
	}
	if len(bytes.TrimSpace(unmerged.Stdout)) > 0 {
		return true, nil
	}
	return false, fmt.Errorf("merge exited %d: %s", res.ExitCode, stderrTail(res.Stderr))
}

// Snapshot collects the changed files and diff since the merge base, and a
// tar archive of the merged tree.
func (g *Git) Snapshot(ctx context.Context, name, integration, sha string) (review.Snapshot, error) {
	wt, err := g.worktree(name)
	if err != nil {
		return review.Snapshot{}, err
	}
	res, err := g.git(ctx, "-C", wt, "merge-base", "refs/remotes/origin/"+integration, sha)
	if err != nil {
		return review.Snapshot{}, err
	}
	base := strings.TrimSpace(string(res.Stdout))
	files, err := g.git(ctx, "-C", wt, "diff", "--name-only", "-z", "--no-renames", base, sha)
	if err != nil {
		return review.Snapshot{}, err
	}
	diff, err := g.git(ctx, "-C", wt, "diff", "--no-color", "--no-ext-diff", "--no-textconv", base, sha)
	if err != nil {
		return review.Snapshot{}, err
	}
	tree, err := g.git(ctx, "-C", wt, "write-tree")
	if err != nil {
		return review.Snapshot{}, err
	}
	archive, err := g.git(ctx, "-C", wt, "archive", "--format=tar", strings.TrimSpace(string(tree.Stdout)))
	if err != nil {
		return review.Snapshot{}, err
	}
	snap := review.Snapshot{Diff: string(diff.Stdout), Tree: archive.Stdout}
	for _, f := range strings.Split(string(files.Stdout), "\x00") {
		if f != "" {
			snap.Files = append(snap.Files, f)
		}
	}
	return snap, nil
}

// RemoveWorktree removes the worktree; it falls back to deleting the
// directory when git no longer knows it.
func (g *Git) RemoveWorktree(ctx context.Context, name string) error {
	wt, err := g.worktree(name)
	if err != nil {
		return err
	}
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", g.clone(), "worktree", "remove", "--force", wt}})
	if err == nil && res.ExitCode == 0 {
		return nil
	}
	if rmErr := g.removeAll(wt); rmErr != nil {
		return errors.Join(err, rmErr)
	}
	_, _ = g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", g.clone(), "worktree", "prune"}})
	return nil
}
