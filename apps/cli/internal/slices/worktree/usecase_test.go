package worktree

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const (
	mainDir = "/src/repo"
	target  = "/src/repo-worktrees/12-x"
)

type fakeGit struct {
	list       []Worktree
	listErr    error
	top        string
	origin     string
	fetchErr   error
	baseTip    string
	inBase     map[string]bool // "a b" -> ancestor
	own        bool
	changed    int
	ignored    []string
	upstream   bool
	ahead      int
	removeErr  error
	deleteErr  error
	calls      []string
	ignoredErr error
}

func (f *fakeGit) Worktrees(context.Context) ([]Worktree, error) { return f.list, f.listErr }
func (f *fakeGit) Toplevel(context.Context) (string, error)      { return f.top, nil }
func (f *fakeGit) OriginURL(context.Context, string) (string, error) {
	return f.origin, nil
}
func (f *fakeGit) Fetch(_ context.Context, dir, branch string) error {
	f.calls = append(f.calls, "fetch "+dir+" "+branch)
	return f.fetchErr
}
func (f *fakeGit) Rev(context.Context, string, string) (string, error) { return f.baseTip, nil }
func (f *fakeGit) IsAncestor(_ context.Context, _, a, b string) (bool, error) {
	return f.inBase[a+" "+b], nil
}
func (f *fakeGit) OwnCommits(context.Context, string, string) (bool, error) { return f.own, nil }
func (f *fakeGit) Changed(context.Context, string) (int, error)             { return f.changed, nil }
func (f *fakeGit) Ignored(context.Context, string) ([]string, error)        { return f.ignored, f.ignoredErr }
func (f *fakeGit) Unpushed(context.Context, string) (bool, int, error) {
	return f.upstream, f.ahead, nil
}
func (f *fakeGit) RemoveWorktree(_ context.Context, dir, path string) error {
	f.calls = append(f.calls, "remove "+dir+" "+path)
	return f.removeErr
}
func (f *fakeGit) DeleteBranch(_ context.Context, dir, branch string) error {
	f.calls = append(f.calls, "delete "+dir+" "+branch)
	return f.deleteErr
}

type fakeGitHub struct {
	open, merged []PullRequest
	err          error
}

func (f *fakeGitHub) OpenPullRequests(context.Context, string) ([]PullRequest, error) {
	return f.open, f.err
}
func (f *fakeGitHub) MergedPullRequests(context.Context, string, string) ([]PullRequest, error) {
	return f.merged, f.err
}

type fakePaths struct{ symlinks map[string]bool }

func (p fakePaths) Real(path string) (string, error) {
	switch path {
	case mainDir, mainDir + "-worktrees", target, "/src/repo-worktrees/13-y":
		return path, nil
	}
	return "", nil
}
func (p fakePaths) IsSymlink(path string) (bool, error) { return p.symlinks[path], nil }

func approvedGit() *fakeGit {
	return &fakeGit{
		list: []Worktree{
			{Path: mainDir, Head: "b1", Branch: "refs/heads/dev"},
			{Path: target, Head: "h1", Branch: "refs/heads/feat/x"},
		},
		top:      mainDir,
		origin:   "git@github.com:cappy-lab/vulcam.git",
		baseTip:  "b1",
		inBase:   map[string]bool{"h1 refs/remotes/origin/dev": true},
		own:      true,
		upstream: true,
	}
}

var settings = Settings{Integration: "dev", Repo: "cappy-lab/vulcam", Keep: []string{"pr-399"}}

func TestCloseRemovesAnApprovedWorktreeThenDeletesItsBranch(t *testing.T) {
	t.Parallel()
	git := approvedGit()
	res := NewUseCase(git, &fakeGitHub{}, fakePaths{}, settings).Close(context.Background(), target)
	if got := res.Line(); got != "CLOSED "+target+" branch=feat/x reason=in-base" {
		t.Fatalf("line = %q", got)
	}
	want := []string{"fetch " + mainDir + " dev", "remove " + mainDir + " " + target, "delete " + mainDir + " feat/x"}
	if strings.Join(git.calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %q, want %q", git.calls, want)
	}
}

func TestCloseByMergedPullRequestHoldingHead(t *testing.T) {
	t.Parallel()
	git := approvedGit()
	git.inBase = map[string]bool{"h1 p2": true}
	gh := &fakeGitHub{merged: []PullRequest{{Number: 40, HeadSHA: "p1"}, {Number: 41, HeadSHA: "p2"}}}
	res := NewUseCase(git, gh, fakePaths{}, settings).Close(context.Background(), target)
	if got := res.Line(); got != "CLOSED "+target+" branch=feat/x reason=merged-pr" {
		t.Fatalf("line = %q", got)
	}

	// A merged PR that does not hold HEAD (work pushed after the merge) does not approve.
	git = approvedGit()
	git.inBase = nil
	res = NewUseCase(git, gh, fakePaths{}, settings).Close(context.Background(), target)
	if res.Closed || !strings.Contains(res.Reason, "not merged") {
		t.Fatalf("result = %+v", res)
	}
}

func TestCloseDetachedWorktreeDeletesNoBranch(t *testing.T) {
	t.Parallel()
	git := approvedGit()
	git.list[1].Branch = ""
	res := NewUseCase(git, &fakeGitHub{err: errors.New("never asked")}, fakePaths{}, settings).Close(context.Background(), target)
	if got := res.Line(); got != "CLOSED "+target+" branch=detached reason=in-base" {
		t.Fatalf("line = %q", got)
	}
	for _, c := range git.calls {
		if strings.HasPrefix(c, "delete ") {
			t.Fatalf("deleted a branch for a detached worktree: %q", git.calls)
		}
	}
}

func TestCloseKeepsTheWorktreeClosedWhenBranchDeleteRefuses(t *testing.T) {
	t.Parallel()
	git := approvedGit()
	git.deleteErr = errors.New("not fully merged")
	res := NewUseCase(git, &fakeGitHub{}, fakePaths{}, settings).Close(context.Background(), target)
	if !res.Closed || !strings.Contains(res.Warning, "not fully merged") || !strings.HasPrefix(res.Line(), "CLOSED ") {
		t.Fatalf("result = %+v", res)
	}
}

func TestCloseRefusals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		arg   string
		setup func(*fakeGit, *fakeGitHub, *fakePaths)
		want  string
	}{
		{"relative", "repo-worktrees/12-x", nil, "not absolute"},
		{"list fails", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.listErr = errors.New("boom") }, "cannot list the worktrees"},
		{"bare main", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.list[0].Bare = true }, "no main worktree"},
		{"main", mainDir, nil, "main worktree is never closed"},
		{"unregistered", "/src/repo-worktrees/13-y", nil, "not a worktree of this repository"},
		{"symlink", target, func(_ *fakeGit, _ *fakeGitHub, p *fakePaths) { p.symlinks = map[string]bool{target: true} }, "symlink"},
		{"kept", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) {}, ""},
		{"foreign origin", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.origin = "https://github.com/evil/vulcam.git" }, "is not cappy-lab/vulcam"},
		{"fetch fails", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.fetchErr = errors.New("offline") }, "could not fetch origin/dev"},
		{"ignored unreadable", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.ignoredErr = errors.New("x") }, "cannot list the ignored files"},
		{"gh fails", target, func(_ *fakeGit, h *fakeGitHub, _ *fakePaths) { h.err = errors.New("gh down\nline two") }, "could not ask GitHub"},
		{"open pr", target, func(_ *fakeGit, h *fakeGitHub, _ *fakePaths) { h.open = []PullRequest{{Number: 9}} }, "open PR #9"},
		{"dirty", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.changed = 1 }, "not clean"},
		{"secret", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.ignored = []string{"dist/.env"} }, "dist/.env"},
		{"unpushed", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.ahead = 1 }, "1 unpushed"},
		{"never worked", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.own = false }, "never worked"},
		{"not merged", target, func(g *fakeGit, _ *fakeGitHub, _ *fakePaths) { g.inBase = nil }, "not merged"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			git, gh, paths := approvedGit(), &fakeGitHub{}, &fakePaths{}
			s := settings
			if tt.name == "kept" {
				s.Keep = []string{"12-x"}
				tt.want = "keep list"
			}
			if tt.setup != nil {
				tt.setup(git, gh, paths)
			}
			res := NewUseCase(git, gh, paths, s).Close(context.Background(), tt.arg)
			line := res.Line()
			if res.Closed || !strings.HasPrefix(line, "REFUSED "+tt.arg+" ") || !strings.Contains(line, tt.want) || strings.Contains(line, "\n") {
				t.Fatalf("line = %q, want a refusal with %q", line, tt.want)
			}
			for _, c := range git.calls {
				if strings.HasPrefix(c, "remove ") || strings.HasPrefix(c, "delete ") {
					t.Fatalf("refused but changed the repository: %q", git.calls)
				}
			}
		})
	}
}

func TestOriginMatches(t *testing.T) {
	t.Parallel()
	for url, want := range map[string]bool{
		"git@github.com:cappy-lab/vulcam.git":              true,
		"https://github.com/cappy-lab/vulcam":              true,
		"https://github.com/cappy-lab/vulcam.git":          true,
		"ssh://git@github.com/cappy-lab/vulcam.git":        true,
		"https://github.com/cappy-lab/vulcam2.git":         false,
		"https://github.com/x/cappy-lab/vulcam.git":        false,
		"https://evil.com/github.com/cappy-lab/vulcam.git": false,
	} {
		if got := OriginMatches(url, "cappy-lab/vulcam"); got != want {
			t.Errorf("OriginMatches(%q) = %v, want %v", url, got, want)
		}
	}
}
