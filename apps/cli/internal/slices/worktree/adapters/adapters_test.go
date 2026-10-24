package adapters

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/execrunner"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/worktree"
)

func line(c kerneltest.RunnerCall) string {
	return strings.Join(append([]string{c.Command.Name}, c.Command.Args...), " ")
}

func TestGitRemoveWorktreeNeverForcesThenPrunes(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	if err := NewGit(r).RemoveWorktree(context.Background(), "/src/repo", "/src/repo-worktrees/12-x"); err != nil {
		t.Fatal(err)
	}
	if len(r.Calls) != 2 || line(r.Calls[0]) != "git -C /src/repo worktree remove /src/repo-worktrees/12-x" ||
		line(r.Calls[1]) != "git -C /src/repo worktree prune" {
		t.Fatalf("calls = %d, first %q", len(r.Calls), line(r.Calls[0]))
	}
	for _, c := range r.Calls {
		for _, a := range c.Command.Args {
			if a == "--force" || a == "-f" {
				t.Fatalf("forced: %q", line(c))
			}
		}
	}
}

func TestGitRemoveWorktreeFailureSkipsPrune(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 128, Stderr: []byte("contains modified or untracked files")}}
	if err := NewGit(r).RemoveWorktree(context.Background(), "/src/repo", "/w"); err == nil || len(r.Calls) != 1 {
		t.Fatalf("error = %v, calls %d", err, len(r.Calls))
	}
}

func TestGitDeleteBranchUsesSafeDelete(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	if err := NewGit(r).DeleteBranch(context.Background(), "/src/repo", "feat/x"); err != nil {
		t.Fatal(err)
	}
	if got := line(r.Calls[0]); got != "git -C /src/repo branch -d feat/x" {
		t.Fatalf("call = %q", got)
	}
	if err := NewGit(r).DeleteBranch(context.Background(), "/src/repo", "-D"); err == nil {
		t.Fatal("an option-like branch name was accepted")
	}
}

func TestGitWorktreesParsesPorcelain(t *testing.T) {
	t.Parallel()
	out := "worktree /src/repo\x00HEAD aaa\x00branch refs/heads/dev\x00\x00" +
		"worktree /src/repo-worktrees/12 x\x00HEAD bbb\x00branch refs/heads/feat/x\x00\x00" +
		"worktree /src/repo-worktrees/13\x00HEAD ccc\x00detached\x00\x00"
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(out)}}
	got, err := NewGit(r).Worktrees(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []worktree.Worktree{
		{Path: "/src/repo", Head: "aaa", Branch: "refs/heads/dev"},
		{Path: "/src/repo-worktrees/12 x", Head: "bbb", Branch: "refs/heads/feat/x"},
		{Path: "/src/repo-worktrees/13", Head: "ccc"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("worktrees = %+v", got)
	}
	if c := line(r.Calls[0]); c != "git worktree list --porcelain -z" {
		t.Fatalf("call = %q", c)
	}
}

func TestGitHubArgv(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`[{"number":41,"headRefOid":"abc"}]`)}}
	gh := NewGitHub(r, "cappy-lab/vulcam")
	merged, err := gh.MergedPullRequests(context.Background(), "feat/x", "dev")
	if err != nil || len(merged) != 1 || merged[0] != (worktree.PullRequest{Number: 41, HeadSHA: "abc"}) {
		t.Fatalf("merged = %+v, %v", merged, err)
	}
	if _, err := gh.OpenPullRequests(context.Background(), "feat/x"); err != nil {
		t.Fatal(err)
	}
	if got := line(r.Calls[0]); got != "gh pr list --repo cappy-lab/vulcam --head feat/x --state merged --base dev --json number,headRefOid --limit 100" {
		t.Fatalf("merged call = %q", got)
	}
	if got := line(r.Calls[1]); got != "gh pr list --repo cappy-lab/vulcam --head feat/x --state open --json number,headRefOid --limit 100" {
		t.Fatalf("open call = %q", got)
	}
	r = &kerneltest.Runner{Result: kernel.Result{ExitCode: 1, Stderr: []byte("HTTP 502")}}
	if _, err := NewGitHub(r, "o/r").OpenPullRequests(context.Background(), "b"); err == nil {
		t.Fatal("gh failure not reported")
	}
}

func gitIn(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@localhost", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@localhost")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Facts read from a real repository: ignored files are listed one by one
// inside ignored build folders (but not inside node_modules), changes are
// counted, and ancestry, reflog and upstream are read correctly.
func TestGitFactsFromARealRepository(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("runs the real git binary")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	origin := filepath.Join(dir, "origin.git")
	main := filepath.Join(dir, "repo")
	wt := filepath.Join(dir, "repo-worktrees", "12-x")
	gitIn(t, "init", "-q", "--bare", origin)
	gitIn(t, "init", "-q", "-b", "dev", main)
	write(t, filepath.Join(main, ".gitignore"))
	if err := os.WriteFile(filepath.Join(main, ".gitignore"), []byte("dist/\nbuild/\nnode_modules/\n.env*\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, "-C", main, "add", ".gitignore")
	gitIn(t, "-C", main, "commit", "-q", "-m", "init")
	gitIn(t, "-C", main, "remote", "add", "origin", origin)
	gitIn(t, "-C", main, "push", "-q", "-u", "origin", "dev")
	gitIn(t, "-C", main, "worktree", "add", "-q", "-b", "feat/x", wt)

	r := execrunner.New()
	g := NewGit(r)
	ctx := context.Background()

	own, err := g.OwnCommits(ctx, main, "feat/x")
	if err != nil || own {
		t.Fatalf("fresh branch OwnCommits() = %v, %v", own, err)
	}
	if has, _, err := g.Unpushed(ctx, wt); err != nil || has {
		t.Fatalf("Unpushed() without upstream = %v, %v", has, err)
	}
	write(t, filepath.Join(wt, "a.txt"))
	gitIn(t, "-C", wt, "add", "a.txt")
	gitIn(t, "-C", wt, "commit", "-q", "-m", "work")
	if own, err = g.OwnCommits(ctx, main, "feat/x"); err != nil || !own {
		t.Fatalf("worked branch OwnCommits() = %v, %v", own, err)
	}
	gitIn(t, "-C", wt, "push", "-q", "-u", "origin", "feat/x")
	write(t, filepath.Join(wt, "b.txt"))
	gitIn(t, "-C", wt, "add", "b.txt")
	gitIn(t, "-C", wt, "commit", "-q", "-m", "more")
	if has, ahead, err := g.Unpushed(ctx, wt); err != nil || !has || ahead != 1 {
		t.Fatalf("Unpushed() = %v, %d, %v", has, ahead, err)
	}
	head := gitIn(t, "-C", wt, "rev-parse", "HEAD")
	if ok, err := g.IsAncestor(ctx, main, head, "refs/heads/dev"); err != nil || ok {
		t.Fatalf("IsAncestor(head, dev) = %v, %v", ok, err)
	}
	if ok, err := g.IsAncestor(ctx, main, "0123456789abcdef0123456789abcdef01234567", "refs/heads/dev"); err != nil || ok {
		t.Fatalf("IsAncestor(unknown) = %v, %v", ok, err)
	}
	if err := g.Fetch(ctx, main, "dev"); err != nil {
		t.Fatalf("Fetch() = %v", err)
	}
	if tip, err := g.Rev(ctx, main, "refs/remotes/origin/dev"); err != nil || len(tip) != 40 {
		t.Fatalf("Rev() = %q, %v", tip, err)
	}
	if url, err := g.OriginURL(ctx, main); err != nil || url != origin {
		t.Fatalf("OriginURL() = %q, %v", url, err)
	}

	write(t, filepath.Join(wt, "untracked.txt"))
	write(t, filepath.Join(wt, ".env.local"))
	write(t, filepath.Join(wt, "dist", "certs", "server.key"))
	write(t, filepath.Join(wt, "build", "app.js"))
	write(t, filepath.Join(wt, "node_modules", "pkg", "test.pem"))
	if n, err := g.Changed(ctx, wt); err != nil || n != 1 {
		t.Fatalf("Changed() = %d, %v", n, err)
	}
	ignored, err := g.Ignored(ctx, wt)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(ignored)
	want := []string{".env.local", "build/app.js", "dist/certs/server.key", "node_modules/"}
	if !slices.Equal(ignored, want) {
		t.Fatalf("Ignored() = %q, want %q", ignored, want)
	}

	list, err := NewGit(r).worktreesIn(ctx, main)
	if err != nil || len(list) != 2 || list[1].Branch != "refs/heads/feat/x" {
		t.Fatalf("worktrees = %+v, %v", list, err)
	}
}

func TestPathsRealAndSymlink(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(realDir, link); err != nil {
		t.Fatal(err)
	}
	p := Paths{}
	resolved, _ := filepath.EvalSymlinks(realDir)
	if got, err := p.Real(link); err != nil || got != resolved {
		t.Fatalf("Real(link) = %q, %v", got, err)
	}
	if got, err := p.Real(filepath.Join(dir, "missing")); err != nil || got != "" {
		t.Fatalf("Real(missing) = %q, %v", got, err)
	}
	if ok, err := p.IsSymlink(link); err != nil || !ok {
		t.Fatalf("IsSymlink(link) = %v, %v", ok, err)
	}
	if ok, err := p.IsSymlink(realDir); err != nil || ok {
		t.Fatalf("IsSymlink(dir) = %v, %v", ok, err)
	}
}
