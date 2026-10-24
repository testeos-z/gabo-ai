package worktree

import (
	"context"
	"fmt"
	"strings"
)

// Settings name the target repository, its integration branch and the
// worktree names (or absolute paths) that are never closed.
type Settings struct {
	Repo        string // owner/name
	Integration string
	Keep        []string
}

// Result is the outcome of Close. Warning reports a branch that could not
// be deleted after its worktree was removed.
type Result struct {
	Closed  bool
	Path    string
	Branch  string
	Reason  string
	Warning string
}

// Line is the single output line: CLOSED <path> branch=<b> reason=<r> or
// REFUSED <path> <reason>.
func (r Result) Line() string {
	if r.Closed {
		branch := r.Branch
		if branch == "" {
			branch = "detached"
		}
		return fmt.Sprintf("CLOSED %s branch=%s reason=%s", r.Path, branch, r.Reason)
	}
	return fmt.Sprintf("REFUSED %s %s", r.Path, r.Reason)
}

// UseCase closes approved worktrees.
type UseCase struct {
	git   Git
	gh    GitHub
	paths Paths
	s     Settings
}

// NewUseCase wires the use case.
func NewUseCase(git Git, gh GitHub, paths Paths, s Settings) *UseCase {
	return &UseCase{git: git, gh: gh, paths: paths, s: s}
}

// refusal builds a refused Result; an error is folded into one line.
func refusal(path, reason string, err error) Result {
	if err != nil {
		reason += ": " + strings.Join(strings.Fields(err.Error()), " ")
	}
	return Result{Path: path, Reason: reason}
}

// Close gathers the facts through the ports, decides, and on approval runs
// `git worktree remove` (never --force), a prune and `git branch -d`.
func (u *UseCase) Close(ctx context.Context, arg string) Result {
	label := strings.TrimRight(arg, "/")
	if label == "" || strings.ContainsAny(arg, "\n\r") {
		return refusal(arg, "the path is empty or has a newline", nil)
	}
	list, err := u.git.Worktrees(ctx)
	if err != nil {
		return refusal(label, "cannot list the worktrees", err)
	}
	if len(list) == 0 || list[0].Bare {
		return refusal(label, "the repository has no main worktree", nil)
	}
	pf, wt, err := u.pathFacts(ctx, label, list)
	if err != nil {
		return refusal(label, "cannot resolve the paths", err)
	}
	if reason := CheckPath(pf); reason != "" {
		return refusal(label, reason, nil)
	}
	main := pf.Main
	branch, ok := strings.CutPrefix(wt.Branch, "refs/heads/")
	if wt.Branch != "" && !ok {
		return refusal(label, "unexpected branch ref "+wt.Branch, nil)
	}
	url, err := u.git.OriginURL(ctx, main)
	if err != nil {
		return refusal(label, "the repository has no origin remote", err)
	}
	if !OriginMatches(url, u.s.Repo) {
		return refusal(label, fmt.Sprintf("origin %s is not %s", url, u.s.Repo), nil)
	}
	if err := u.git.Fetch(ctx, main, u.s.Integration); err != nil {
		return refusal(label, "could not fetch origin/"+u.s.Integration, err)
	}
	sf, refused := u.stateFacts(ctx, label, main, pf.Real, wt.Head, branch)
	if refused != nil {
		return *refused
	}
	d := Decide(sf)
	if !d.Close {
		return refusal(label, d.Reason, nil)
	}
	if err := u.git.RemoveWorktree(ctx, main, wt.Path); err != nil {
		return refusal(label, "git worktree remove failed (left in place)", err)
	}
	res := Result{Closed: true, Path: pf.Real, Branch: branch, Reason: d.Reason}
	if branch != "" {
		if err := u.git.DeleteBranch(ctx, main, branch); err != nil {
			res.Warning = fmt.Sprintf("branch %s kept: %v", branch, err)
		}
	}
	return res
}

func (u *UseCase) pathFacts(ctx context.Context, arg string, list []Worktree) (PathFacts, Worktree, error) {
	f := PathFacts{Arg: arg, Keep: u.s.Keep}
	var err error
	if f.Main, err = u.paths.Real(list[0].Path); err != nil {
		return f, Worktree{}, err
	}
	if f.Main == "" {
		return f, Worktree{}, fmt.Errorf("main worktree %s does not exist", list[0].Path)
	}
	if f.WorktreesDir, err = u.paths.Real(f.Main + "-worktrees"); err != nil {
		return f, Worktree{}, err
	}
	if top, err := u.git.Toplevel(ctx); err == nil && top != "" {
		f.Current, _ = u.paths.Real(top)
	}
	if !strings.HasPrefix(arg, "/") {
		return f, Worktree{}, nil
	}
	if f.Symlink, err = u.paths.IsSymlink(arg); err != nil {
		return f, Worktree{}, err
	}
	if f.Real, err = u.paths.Real(arg); err != nil || f.Real == "" {
		return f, Worktree{}, err
	}
	for _, w := range list[1:] {
		if real, err := u.paths.Real(w.Path); err == nil && real == f.Real && !w.Bare {
			f.Registered = true
			return f, w, nil
		}
	}
	return f, Worktree{}, nil
}

func (u *UseCase) stateFacts(ctx context.Context, label, main, dir, head, branch string) (StateFacts, *Result) {
	fail := func(reason string, err error) (StateFacts, *Result) {
		r := refusal(label, reason, err)
		return StateFacts{}, &r
	}
	base := "refs/remotes/origin/" + u.s.Integration
	f := StateFacts{Integration: u.s.Integration, Branch: branch, Head: head}
	var err error
	if f.BaseTip, err = u.git.Rev(ctx, main, base); err != nil {
		return fail("cannot read origin/"+u.s.Integration, err)
	}
	if f.Changed, err = u.git.Changed(ctx, dir); err != nil {
		return fail("cannot read its status", err)
	}
	if f.Ignored, err = u.git.Ignored(ctx, dir); err != nil {
		return fail("cannot list the ignored files", err)
	}
	if f.HasUpstream, f.Unpushed, err = u.git.Unpushed(ctx, dir); err != nil {
		return fail("cannot compare with its upstream", err)
	}
	if f.InBase, err = u.git.IsAncestor(ctx, main, head, base); err != nil {
		return fail("cannot compare with origin/"+u.s.Integration, err)
	}
	if branch == "" {
		return f, nil
	}
	if f.OwnCommits, err = u.git.OwnCommits(ctx, main, branch); err != nil {
		return fail("cannot read the reflog of "+branch, err)
	}
	open, err := u.gh.OpenPullRequests(ctx, branch)
	if err != nil {
		return fail("could not ask GitHub for open PRs of "+branch, err)
	}
	if len(open) > 0 {
		f.OpenPR = open[0].Number
	}
	merged, err := u.gh.MergedPullRequests(ctx, branch, u.s.Integration)
	if err != nil {
		return fail("could not ask GitHub for merged PRs of "+branch, err)
	}
	for _, pr := range merged {
		if pr.HeadSHA == head {
			f.MergedPR = pr.Number
			break
		}
		if held, err := u.git.IsAncestor(ctx, main, head, pr.HeadSHA); err == nil && held {
			f.MergedPR = pr.Number
			break
		}
	}
	return f, nil
}
