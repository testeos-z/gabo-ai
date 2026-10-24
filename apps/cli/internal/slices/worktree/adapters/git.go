// Package adapters holds the driven adapters of the worktree slice.
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/worktree"
)

// Git implements worktree.Git with the git CLI.
type Git struct{ runner kernel.Runner }

var _ worktree.Git = (*Git)(nil)

// NewGit returns a git-backed adapter working on the current directory's repository.
func NewGit(r kernel.Runner) *Git { return &Git{runner: r} }

func (g *Git) git(ctx context.Context, args ...string) (kernel.Result, error) {
	return kernel.RunChecked(ctx, g.runner, kernel.Command{Name: "git", Args: args})
}

// safeArg refuses values git could read as an option.
func safeArg(v string) error {
	if v == "" || strings.HasPrefix(v, "-") || strings.ContainsAny(v, "\x00\n") {
		return fmt.Errorf("unsafe git argument %q", v)
	}
	return nil
}

// Worktrees lists the worktrees of the repository the command runs in.
func (g *Git) Worktrees(ctx context.Context) ([]worktree.Worktree, error) {
	return g.worktreesIn(ctx, "")
}

func (g *Git) worktreesIn(ctx context.Context, dir string) ([]worktree.Worktree, error) {
	var args []string
	if dir != "" {
		args = append(args, "-C", dir)
	}
	res, err := g.git(ctx, append(args, "worktree", "list", "--porcelain", "-z")...)
	if err != nil {
		return nil, err
	}
	var out []worktree.Worktree
	for _, field := range strings.Split(string(res.Stdout), "\x00") {
		key, value, _ := strings.Cut(field, " ")
		switch {
		case key == "worktree":
			out = append(out, worktree.Worktree{Path: value})
		case len(out) == 0 || field == "":
		case key == "HEAD":
			out[len(out)-1].Head = value
		case key == "branch":
			out[len(out)-1].Branch = value
		case key == "bare":
			out[len(out)-1].Bare = true
		}
	}
	return out, nil
}

// Toplevel returns the top of the current worktree, or "" outside one.
func (g *Git) Toplevel(ctx context.Context) (string, error) {
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"rev-parse", "--show-toplevel"}})
	if err != nil || res.ExitCode != 0 {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// OriginURL returns the URL of the origin remote.
func (g *Git) OriginURL(ctx context.Context, dir string) (string, error) {
	res, err := g.git(ctx, "-C", dir, "remote", "get-url", "origin")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// Fetch updates refs/remotes/origin/<branch>.
func (g *Git) Fetch(ctx context.Context, dir, branch string) error {
	if err := safeArg(branch); err != nil {
		return err
	}
	_, err := g.git(ctx, "-C", dir, "fetch", "--quiet", "--no-tags", "origin", "+refs/heads/"+branch+":refs/remotes/origin/"+branch)
	return err
}

// Rev resolves ref to a commit id.
func (g *Git) Rev(ctx context.Context, dir, ref string) (string, error) {
	if err := safeArg(ref); err != nil {
		return "", err
	}
	res, err := g.git(ctx, "-C", dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// IsAncestor runs merge-base --is-ancestor: exit 0 yes, 1 no; an unknown
// commit is not an ancestor.
func (g *Git) IsAncestor(ctx context.Context, dir, a, b string) (bool, error) {
	if safeArg(a) != nil || safeArg(b) != nil {
		return false, fmt.Errorf("unsafe commit %q or %q", a, b)
	}
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", dir, "merge-base", "--is-ancestor", a, b}})
	if err != nil {
		return false, err
	}
	switch res.ExitCode {
	case 0:
		return true, nil
	case 1:
		return false, nil
	}
	if bytes.Contains(res.Stderr, []byte("ot a valid commit")) || bytes.Contains(res.Stderr, []byte("no such commit")) {
		return false, nil
	}
	return false, fmt.Errorf("git merge-base exited %d: %s", res.ExitCode, kernel.StderrTail(res.Stderr))
}

// OwnCommits reports whether the branch's reflog holds more than its
// creation entry. A missing reflog counts as no commits of its own.
func (g *Git) OwnCommits(ctx context.Context, dir, branch string) (bool, error) {
	if err := safeArg(branch); err != nil {
		return false, err
	}
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", dir, "reflog", "show", "--format=%H", "refs/heads/" + branch, "--"}})
	if err != nil {
		return false, err
	}
	if res.ExitCode != 0 {
		return false, nil
	}
	return len(strings.Fields(string(res.Stdout))) > 1, nil
}

// Changed counts tracked changes and untracked files.
func (g *Git) Changed(ctx context.Context, dir string) (int, error) {
	res, err := g.git(ctx, "-C", dir, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range strings.Split(string(res.Stdout), "\x00") {
		// A rename entry is followed by its source path, which has no status.
		if len(e) > 3 && e[2] == ' ' {
			n++
		}
	}
	return n, nil
}

// Ignored lists the ignored files removing the worktree would delete. An
// ignored folder is expanded file by file, except node_modules, whose
// package content is listed as the folder itself.
func (g *Git) Ignored(ctx context.Context, dir string) ([]string, error) {
	res, err := g.git(ctx, "-C", dir, "status", "--porcelain=v1", "-z", "--ignored=matching", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range strings.Split(string(res.Stdout), "\x00") {
		p, ok := strings.CutPrefix(e, "!! ")
		if !ok {
			continue
		}
		if !strings.HasSuffix(p, "/") || filepath.Base(p) == "node_modules" {
			out = append(out, p)
			continue
		}
		files, err := g.git(ctx, "-C", dir, "ls-files", "-z", "--others", "--ignored", "--exclude-standard", "--", p)
		if err != nil {
			return nil, err
		}
		for _, f := range strings.Split(string(files.Stdout), "\x00") {
			if f != "" {
				out = append(out, f)
			}
		}
	}
	return out, nil
}

// Unpushed reports whether HEAD has an upstream and how far ahead of it it is.
func (g *Git) Unpushed(ctx context.Context, dir string) (bool, int, error) {
	res, err := g.runner.Run(ctx, kernel.Command{Name: "git", Args: []string{"-C", dir, "rev-parse", "--verify", "--quiet", "--symbolic-full-name", "@{upstream}"}})
	if err != nil {
		return false, 0, err
	}
	if res.ExitCode != 0 || strings.TrimSpace(string(res.Stdout)) == "" {
		return false, 0, nil
	}
	count, err := g.git(ctx, "-C", dir, "rev-list", "--count", "@{upstream}..HEAD")
	if err != nil {
		return true, 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(count.Stdout)))
	if err != nil {
		return true, 0, fmt.Errorf("parse unpushed count: %w", err)
	}
	return true, n, nil
}

// RemoveWorktree runs `git worktree remove` without --force, so git itself
// refuses a worktree that changed after the checks, then prunes.
func (g *Git) RemoveWorktree(ctx context.Context, dir, path string) error {
	if err := safeArg(path); err != nil {
		return err
	}
	if _, err := g.git(ctx, "-C", dir, "worktree", "remove", path); err != nil {
		return err
	}
	_, err := g.git(ctx, "-C", dir, "worktree", "prune")
	return err
}

// DeleteBranch deletes the local branch with `git branch -d`, which refuses
// an unmerged branch; the remote branch is never touched.
func (g *Git) DeleteBranch(ctx context.Context, dir, branch string) error {
	if err := safeArg(branch); err != nil {
		return err
	}
	_, err := g.git(ctx, "-C", dir, "branch", "-d", branch)
	return err
}

// GitHub implements worktree.GitHub with the gh CLI (read-only calls).
type GitHub struct {
	runner kernel.Runner
	repo   string
}

var _ worktree.GitHub = (*GitHub)(nil)

// NewGitHub returns a gh-backed adapter for repo (owner/name).
func NewGitHub(r kernel.Runner, repo string) *GitHub { return &GitHub{runner: r, repo: repo} }

func (g *GitHub) list(ctx context.Context, args ...string) ([]worktree.PullRequest, error) {
	argv := append([]string{"pr", "list", "--repo", g.repo}, args...)
	argv = append(argv, "--json", "number,headRefOid", "--limit", "100")
	res, err := kernel.RunChecked(ctx, g.runner, kernel.Command{Name: "gh", Args: argv})
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Number     int    `json:"number"`
		HeadRefOid string `json:"headRefOid"`
	}
	if err := json.Unmarshal(res.Stdout, &rows); err != nil {
		return nil, fmt.Errorf("decode gh pr list: %w", err)
	}
	out := make([]worktree.PullRequest, 0, len(rows))
	for _, r := range rows {
		out = append(out, worktree.PullRequest{Number: r.Number, HeadSHA: r.HeadRefOid})
	}
	return out, nil
}

// OpenPullRequests lists the open PRs whose head is branch.
func (g *GitHub) OpenPullRequests(ctx context.Context, branch string) ([]worktree.PullRequest, error) {
	return g.list(ctx, "--head", branch, "--state", "open")
}

// MergedPullRequests lists the PRs from branch merged into base.
func (g *GitHub) MergedPullRequests(ctx context.Context, branch, base string) ([]worktree.PullRequest, error) {
	return g.list(ctx, "--head", branch, "--state", "merged", "--base", base)
}

// Paths implements worktree.Paths with the os package.
type Paths struct{}

var _ worktree.Paths = Paths{}

// Real returns the absolute symlink-free path, or "" when it does not exist.
func (Paths) Real(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	real, err := filepath.EvalSymlinks(abs)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return real, err
}

// IsSymlink reports whether the last component of path is a symlink.
func (Paths) IsSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
