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

// Git implements review.Git with a private clone at <workDir>/repo and
// worktrees at <workDir>/<name>.
type Git struct {
	runner    kernel.Runner
	repo      string
	workDir   string
	removeAll func(string) error
}

var _ review.Git = (*Git)(nil)

// NewGit returns a git adapter. removeAll defaults to os.RemoveAll.
func NewGit(r kernel.Runner, repo, workDir string, removeAll func(string) error) *Git {
	if removeAll == nil {
		removeAll = os.RemoveAll
	}
	return &Git{runner: r, repo: repo, workDir: workDir, removeAll: removeAll}
}

func (g *Git) clone() string { return path.Join(g.workDir, "repo") }

func (g *Git) worktree(name string) (string, error) {
	if err := review.ValidateWorkspace(name); err != nil {
		return "", err
	}
	return path.Join(g.workDir, name), nil
}

func (g *Git) git(ctx context.Context, args ...string) (kernel.Result, error) {
	return run(ctx, g.runner, kernel.Command{Name: "git", Args: args})
}

// Prepare clones the repository when missing and fetches the integration
// branch and the PR head.
func (g *Git) Prepare(ctx context.Context, n int, integration string) error {
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", g.clone(), "rev-parse", "--git-dir"}})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		args := append(slices.Clone(credentialFlags), "clone", "--no-checkout", "https://github.com/"+g.repo+".git", g.clone())
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
	return nil
}

// AddWorktree creates a detached worktree at origin/<integration>.
func (g *Git) AddWorktree(ctx context.Context, name, integration string) error {
	wt, err := g.worktree(name)
	if err != nil {
		return err
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
