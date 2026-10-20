package adapters

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/kerneltest"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

const sha = "0123456789abcdef0123456789abcdef01234567"

func argv(c kerneltest.RunnerCall) []string {
	return append([]string{c.Command.Name}, c.Command.Args...)
}

func joined(calls []kerneltest.RunnerCall) []string {
	out := make([]string, 0, len(calls))
	for _, c := range calls {
		out = append(out, strings.Join(argv(c), " "))
	}
	return out
}

func TestGitHubPullRequest(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`{"state":"OPEN","headRefOid":"` + sha + `","baseRefName":"dev","labels":[{"name":"mode:cloud"},{"name":"bug"}]}`)}}
	pr, err := NewGitHub(r, "cappy-lab/vulcam").PullRequest(context.Background(), 12)
	if err != nil {
		t.Fatalf("PullRequest() error = %v", err)
	}
	if pr.Number != 12 || pr.State != "OPEN" || pr.HeadSHA != sha || pr.BaseBranch != "dev" || !slices.Equal(pr.Labels, []string{"mode:cloud", "bug"}) {
		t.Fatalf("pr = %+v", pr)
	}
	want := []string{"gh", "pr", "view", "12", "--repo", "cappy-lab/vulcam", "--json", "state,headRefOid,baseRefName,labels"}
	if got := argv(r.Calls[0]); !slices.Equal(got, want) {
		t.Fatalf("argv = %q", got)
	}
}

func TestGitHubPostsWithoutInterpolation(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	gh := NewGitHub(r, "o/r")
	ctx := context.Background()
	body := "Blocking issues found.\n$(touch /tmp/pwn) `x`"
	if err := gh.SetStatus(ctx, sha, review.Status{Context: "reviewer/tests", State: review.StatusFailure, Description: "a; b"}); err != nil {
		t.Fatal(err)
	}
	if err := gh.Comment(ctx, 12, body); err != nil {
		t.Fatal(err)
	}
	if err := gh.Approve(ctx, 12, sha); err != nil {
		t.Fatal(err)
	}
	if err := gh.Merge(ctx, 12, sha); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"gh", "api", "--method", "POST", "repos/o/r/statuses/" + sha, "-f", "state=failure", "-f", "context=reviewer/tests", "-f", "description=a; b"},
		{"gh", "pr", "comment", "12", "--repo", "o/r", "--body-file", "-"},
		{"gh", "pr", "review", "12", "--repo", "o/r", "--approve", "--body", "Approved by the gbai reviewer at " + sha + "."},
		{"gh", "pr", "merge", "12", "--repo", "o/r", "--merge", "--match-head-commit", sha},
	}
	for i, w := range want {
		if got := argv(r.Calls[i]); !slices.Equal(got, w) {
			t.Errorf("call %d argv = %q, want %q", i, got, w)
		}
	}
	if r.Calls[1].Stdin != body {
		t.Errorf("comment stdin = %q", r.Calls[1].Stdin)
	}
}

func TestGitHubNonZeroExitIsError(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 1, Stderr: []byte("HTTP 409")}}
	if err := NewGitHub(r, "o/r").Merge(context.Background(), 1, sha); err == nil || !strings.Contains(err.Error(), "HTTP 409") {
		t.Fatalf("Merge() error = %v", err)
	}
}

type gitScript struct {
	exit map[string]int    // command substring -> exit code
	out  map[string]string // command substring -> stdout
}

func (g gitScript) handle(c kernel.Command) (kernel.Result, error) {
	line := strings.Join(append([]string{c.Name}, c.Args...), " ")
	res := kernel.Result{}
	for k, v := range g.exit {
		if strings.Contains(line, k) {
			res.ExitCode = v
		}
	}
	for k, v := range g.out {
		if strings.Contains(line, k) {
			res.Stdout = []byte(v)
		}
	}
	return res, nil
}

func TestGitPrepareClonesWhenMissingThenFetchesPinnedRefs(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: gitScript{exit: map[string]int{"rev-parse --git-dir": 128}}.handle}
	if err := NewGit(r, "o/r", "/data/reviewer/work", nil).Prepare(context.Background(), 12, "dev"); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	calls := joined(r.Calls)
	if len(calls) != 3 || !strings.Contains(calls[1], "clone --no-checkout https://github.com/o/r.git /data/reviewer/work/repo") {
		t.Fatalf("calls = %q", calls)
	}
	fetch := argv(r.Calls[2])
	for _, want := range []string{"+refs/heads/dev:refs/remotes/origin/dev", "+refs/pull/12/head:refs/remotes/origin/pr/12", "credential.helper=!gh auth git-credential"} {
		if !slices.Contains(fetch, want) {
			t.Errorf("fetch argv %q lacks %q", fetch, want)
		}
	}
}

func TestGitPrepareReusesExistingClone(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Handle: gitScript{}.handle}
	if err := NewGit(r, "o/r", "/w", nil).Prepare(context.Background(), 12, "dev"); err != nil {
		t.Fatal(err)
	}
	for _, c := range joined(r.Calls) {
		if strings.Contains(c, " clone ") {
			t.Fatalf("cloned again: %q", c)
		}
	}
}

func TestGitMergePinnedDetectsConflicts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		script       gitScript
		wantConflict bool
		wantErr      bool
	}{
		{"clean", gitScript{}, false, false},
		{"conflict", gitScript{exit: map[string]int{"merge --no-ff": 1}, out: map[string]string{"--diff-filter=U": "a.ts\n"}}, true, false},
		{"other failure", gitScript{exit: map[string]int{"merge --no-ff": 128}}, false, true},
		{"sha missing", gitScript{exit: map[string]int{"cat-file": 1}}, false, true},
	}
	for _, tt := range tests {
		r := &kerneltest.Runner{Handle: tt.script.handle}
		conflict, err := NewGit(r, "o/r", "/w", nil).MergePinned(context.Background(), "pr-12-0123456", sha)
		if conflict != tt.wantConflict || (err != nil) != tt.wantErr {
			t.Errorf("%s: conflict=%v err=%v", tt.name, conflict, err)
		}
	}
	r := &kerneltest.Runner{Handle: gitScript{}.handle}
	_, _ = NewGit(r, "o/r", "/w", nil).MergePinned(context.Background(), "pr-12-0123456", sha)
	merge := argv(r.Calls[len(r.Calls)-1])
	for _, want := range []string{"/w/pr-12-0123456", "--no-ff", "--no-commit", sha, "core.hooksPath=/dev/null"} {
		if !slices.Contains(merge, want) {
			t.Errorf("merge argv %q lacks %q", merge, want)
		}
	}
}

func TestGitSnapshot(t *testing.T) {
	t.Parallel()
	base := strings.Repeat("b", 40)
	r := &kerneltest.Runner{Handle: gitScript{out: map[string]string{
		"merge-base": base + "\n", "--name-only": "a.ts\x00dir/b c.md\x00", "diff --no-color": "DIFF", "write-tree": "tree1\n", "archive": "TAR",
	}}.handle}
	snap, err := NewGit(r, "o/r", "/w", nil).Snapshot(context.Background(), "pr-12-0123456", "dev", sha)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if !slices.Equal(snap.Files, []string{"a.ts", "dir/b c.md"}) || snap.Diff != "DIFF" || string(snap.Tree) != "TAR" {
		t.Fatalf("snapshot = %+v", snap)
	}
	if !slices.Contains(joined(r.Calls), "git -C /w/pr-12-0123456 archive --format=tar tree1") {
		t.Fatalf("calls = %q", joined(r.Calls))
	}
}

func TestGitRemoveWorktreeFallsBackToRemoveAll(t *testing.T) {
	t.Parallel()
	var removed []string
	rm := func(p string) error { removed = append(removed, p); return nil }
	r := &kerneltest.Runner{Handle: gitScript{exit: map[string]int{"worktree remove": 128}}.handle}
	if err := NewGit(r, "o/r", "/w", rm).RemoveWorktree(context.Background(), "pr-12-0123456"); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(removed, []string{"/w/pr-12-0123456"}) {
		t.Fatalf("removed = %v", removed)
	}
}

func TestGitRejectsUnsafeNames(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{}
	g := NewGit(r, "o/r", "/w", nil)
	if err := g.AddWorktree(context.Background(), "../repo", "dev"); err == nil {
		t.Error("AddWorktree accepted traversal")
	}
	if err := g.RemoveWorktree(context.Background(), "repo"); err == nil {
		t.Error("RemoveWorktree accepted the clone")
	}
	if len(r.Calls) != 0 {
		t.Fatalf("calls = %q", joined(r.Calls))
	}
}

func TestCIClientRunsThroughSudoWrapper(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{ExitCode: 1, Stdout: []byte("out\n"), Stderr: []byte("err\n")}}
	c := NewCIClient(r, "reviewer-ci", review.RuntimePi, nil)
	ctx := context.Background()
	if err := c.Prepare(ctx, "pr-12-0123456", []byte("TAR")); err == nil {
		t.Fatal("Prepare() with failing sudo returned nil")
	}
	out, err := c.Run(ctx, "pr-12-0123456", []string{"bun", "run", "lint"})
	if err != nil || out.Passed || string(out.Output) != "out\nerr\n" {
		t.Fatalf("Run() = %+v, %v", out, err)
	}
	prefix := []string{"sudo", "-n", "-u", "reviewer-ci", kernel.ReviewCIPath}
	want := [][]string{
		append(slices.Clone(prefix), "prepare", "--dir", "pr-12-0123456"),
		append(slices.Clone(prefix), "run", "--dir", "pr-12-0123456", "--", "bun", "run", "lint"),
	}
	for i, w := range want {
		if got := argv(r.Calls[i]); !slices.Equal(got, w) {
			t.Errorf("call %d = %q, want %q", i, got, w)
		}
	}
	if r.Calls[0].Stdin != "TAR" {
		t.Errorf("prepare stdin = %q", r.Calls[0].Stdin)
	}
}

func TestCIClientReviewSendsCredentialOnStdinOnly(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`{"type":"result","is_error":false,"result":"{\"findings\":[]}"}`)}}
	cred := func(rt string) ([]byte, error) { return []byte("tok-123\n"), nil }
	findings, err := NewCIClient(r, "reviewer-ci", review.RuntimeClaude, cred).Review(context.Background(), "pr-12-0123456", "PROMPT")
	if err != nil || len(findings) != 0 {
		t.Fatalf("Review() = %v, %v", findings, err)
	}
	c := r.Calls[0]
	want := []string{"sudo", "-n", "-u", "reviewer-ci", kernel.ReviewCIPath, "engine", "--dir", "pr-12-0123456", "--runtime", "claude"}
	if !slices.Equal(argv(c), want) || c.Stdin != "tok-123\nPROMPT" || len(c.Command.Env) != 0 {
		t.Fatalf("call = %q stdin %q env %q", argv(c), c.Stdin, c.Command.Env)
	}
}

func TestCIClientReviewFailures(t *testing.T) {
	t.Parallel()
	ok := func(string) ([]byte, error) { return []byte("tok"), nil }
	for name, tt := range map[string]struct {
		res  kernel.Result
		cred func(string) ([]byte, error)
		want error
	}{
		"missing credential": {kernel.Result{}, func(string) ([]byte, error) { return nil, errors.New("no file") }, nil},
		"engine exit":        {kernel.Result{ExitCode: 1}, ok, nil},
		"garbage":            {kernel.Result{Stdout: []byte("I think it looks fine!")}, ok, review.ErrUnparseable},
	} {
		r := &kerneltest.Runner{Result: tt.res}
		_, err := NewCIClient(r, "reviewer-ci", review.RuntimeClaude, tt.cred).Review(context.Background(), "pr-12-0123456", "p")
		if err == nil || (tt.want != nil && !errors.Is(err, tt.want)) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func TestOrcaArgv(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`{"id":"t1"}`)}}
	o := NewOrca(r)
	out, err := o.CreateTerminal(context.Background(), "gbai-review-1-0123456", "sudo x | gbai y")
	if err != nil || out != `{"id":"t1"}` {
		t.Fatalf("CreateTerminal() = %q, %v", out, err)
	}
	if err := o.Send(context.Background(), "run:abc", "PR #1 head x verdict=clean merge=none"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"orca", "terminal", "create", "--title", "gbai-review-1-0123456", "--command", "sudo x | gbai y", "--json"},
		{"orca", "orchestration", "send", "--to", "run:abc", "--subject", "PR #1 head x verdict=clean merge=none", "--type", "status"},
	}
	for i, w := range want {
		if got := argv(r.Calls[i]); !slices.Equal(got, w) {
			t.Errorf("call %d = %q", i, got)
		}
	}
}
