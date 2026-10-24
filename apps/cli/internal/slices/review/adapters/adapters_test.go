package adapters

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/execrunner"
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
	if err := gh.Merge(ctx, 12, sha); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"gh", "api", "--method", "POST", "repos/o/r/statuses/" + sha, "-f", "state=failure", "-f", "context=reviewer/tests", "-f", "description=a; b"},
		{"gh", "pr", "comment", "12", "--repo", "o/r", "--body-file", "-"},
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

func TestGitHubCommentsDecodesEveryPageInOrder(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`[{"author":"owner","body":"a"},{"author":"bob","body":"b\nc"}]` + "\n" + `[{"author":"owner","body":"d"}]` + "\n")}}
	got, err := NewGitHub(r, "o/r").Comments(context.Background(), 12)
	if err != nil {
		t.Fatalf("Comments() error = %v", err)
	}
	if !slices.Equal(got, []review.Comment{{Author: "owner", Body: "a"}, {Author: "bob", Body: "b\nc"}, {Author: "owner", Body: "d"}}) {
		t.Fatalf("Comments() = %q", got)
	}
	want := []string{"gh", "api", "--paginate", "repos/o/r/issues/12/comments", "--jq", "[.[] | {author: .user.login, body: .body}]"}
	if got := argv(r.Calls[0]); !slices.Equal(got, want) {
		t.Fatalf("argv = %q", got)
	}
	r.Result.Stdout = []byte("not json")
	if _, err := NewGitHub(r, "o/r").Comments(context.Background(), 12); err == nil {
		t.Fatal("Comments() on garbage error = nil")
	}
}

func TestGitHubLogin(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte("owner\n")}}
	got, err := NewGitHub(r, "o/r").Login(context.Background())
	if err != nil || got != "owner" {
		t.Fatalf("Login() = %q, %v", got, err)
	}
	if want := []string{"gh", "api", "user", "--jq", ".login"}; !slices.Equal(argv(r.Calls[0]), want) {
		t.Fatalf("argv = %q", argv(r.Calls[0]))
	}
	for name, res := range map[string]kernel.Result{
		"empty login": {Stdout: []byte("\n")},
		"two lines":   {Stdout: []byte("owner\nmallory\n")},
		"gh fails":    {ExitCode: 1, Stderr: []byte("not logged in")},
	} {
		r := &kerneltest.Runner{Result: res}
		if got, err := NewGitHub(r, "o/r").Login(context.Background()); err == nil {
			t.Errorf("%s: Login() = %q, want an error", name, got)
		}
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
	state := "/data/home/.local/state/gbai/reviewer"
	if err := NewGit(r, "o/r", state, nil).Prepare(context.Background(), 12, "dev"); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	calls := joined(r.Calls)
	if len(calls) != 4 || !strings.Contains(calls[1], "clone --branch dev https://github.com/o/r.git "+state+"/clone") {
		t.Fatalf("calls = %q", calls)
	}
	if !strings.HasSuffix(calls[3], "-C "+state+"/clone -c core.hooksPath=/dev/null reset --hard --quiet refs/remotes/origin/dev") {
		t.Fatalf("clone not synced to the integration branch: %q", calls[3])
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
	for _, want := range []string{"/w/worktrees/pr-12-0123456", "--no-ff", "--no-commit", sha, "core.hooksPath=/dev/null"} {
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
	if !slices.Contains(joined(r.Calls), "git -C /w/worktrees/pr-12-0123456 archive --format=tar tree1") {
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
	if !slices.Equal(removed, []string{"/w/worktrees/pr-12-0123456"}) {
		t.Fatalf("removed = %v", removed)
	}
}

// R3-001: a fresh run has no stale worktree; the stale-removal step must not
// run, so a removal that would fail cannot fail the run.
func TestGitAddWorktreeOnAFreshStateSkipsStaleRemoval(t *testing.T) {
	t.Parallel()
	rm := func(string) error { return errors.New("permission denied") }
	r := &kerneltest.Runner{Handle: gitScript{exit: map[string]int{"worktree remove": 128}}.handle}
	state := t.TempDir()
	if err := NewGit(r, "o/r", state, rm).AddWorktree(context.Background(), "pr-12-0123456", "dev"); err != nil {
		t.Fatalf("AddWorktree() on a fresh state error = %v", err)
	}
	calls := joined(r.Calls)
	for _, c := range calls {
		if strings.Contains(c, "worktree remove") {
			t.Fatalf("fresh run tried to remove a worktree: %q", calls)
		}
	}
	if last := calls[len(calls)-1]; !strings.HasSuffix(last, "worktree add --detach "+state+"/worktrees/pr-12-0123456 refs/remotes/origin/dev") {
		t.Fatalf("calls = %q", calls)
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

// fakeCICore records the calls LocalCI makes into the CI use case.
type fakeCICore struct {
	res      kernel.Result
	err      error
	prepared map[string]string
	ran      [][]string
	engine   []string // dir, runtime, stdin
	cleaned  []string
}

func (f *fakeCICore) Prepare(name string, tree io.Reader) error {
	b, _ := io.ReadAll(tree)
	if f.prepared == nil {
		f.prepared = map[string]string{}
	}
	f.prepared[name] = string(b)
	return f.err
}

func (f *fakeCICore) Run(_ context.Context, name string, argv []string) (kernel.Result, error) {
	f.ran = append(f.ran, append([]string{name}, argv...))
	return f.res, f.err
}

func (f *fakeCICore) Engine(_ context.Context, name, runtime string, stdin io.Reader) (kernel.Result, error) {
	b, _ := io.ReadAll(stdin)
	f.engine = []string{name, runtime, string(b)}
	return f.res, f.err
}

func (f *fakeCICore) Cleanup(name string) error {
	f.cleaned = append(f.cleaned, name)
	return f.err
}

func TestLocalCIRunsChecksInProcessWithoutSudo(t *testing.T) {
	t.Parallel()
	core := &fakeCICore{res: kernel.Result{ExitCode: 1, Stdout: []byte("out\n"), Stderr: []byte("err\n")}}
	c := NewLocalCI(core, review.RuntimePi, nil)
	ctx := context.Background()
	if err := c.Prepare(ctx, "pr-12-0123456", []byte("TAR")); err != nil || core.prepared["pr-12-0123456"] != "TAR" {
		t.Fatalf("Prepare() = %v, prepared %v", err, core.prepared)
	}
	out, err := c.Run(ctx, "pr-12-0123456", []string{"bun", "run", "lint"})
	if err != nil || out.Passed || string(out.Output) != "out\nerr\n" {
		t.Fatalf("Run() = %+v, %v", out, err)
	}
	if want := []string{"pr-12-0123456", "bun", "run", "lint"}; len(core.ran) != 1 || !slices.Equal(core.ran[0], want) {
		t.Fatalf("ran = %q", core.ran)
	}
	if err := c.Cleanup(ctx, "pr-12-0123456"); err != nil || !slices.Equal(core.cleaned, []string{"pr-12-0123456"}) {
		t.Fatalf("Cleanup() = %v, cleaned %v", err, core.cleaned)
	}
}

func TestLocalCIReviewSendsCredentialOnStdinOnly(t *testing.T) {
	t.Parallel()
	core := &fakeCICore{res: kernel.Result{Stdout: []byte(`{"type":"result","is_error":false,"result":"{\"findings\":[]}"}`)}}
	cred := func(rt string) ([]byte, error) { return []byte("tok-123\n"), nil }
	findings, err := NewLocalCI(core, review.RuntimeClaude, cred).Review(context.Background(), "pr-12-0123456", "PROMPT")
	if err != nil || len(findings) != 0 {
		t.Fatalf("Review() = %v, %v", findings, err)
	}
	if want := []string{"pr-12-0123456", "claude", "tok-123\nPROMPT"}; !slices.Equal(core.engine, want) {
		t.Fatalf("engine = %q, want %q", core.engine, want)
	}
}

func TestLocalCIReviewFailures(t *testing.T) {
	t.Parallel()
	ok := func(string) ([]byte, error) { return []byte("tok"), nil }
	for name, tt := range map[string]struct {
		res  kernel.Result
		cred func(string) ([]byte, error)
		want error
	}{
		"missing credential":    {kernel.Result{}, func(string) ([]byte, error) { return nil, errors.New("no file") }, nil},
		"multi-line credential": {kernel.Result{}, func(string) ([]byte, error) { return []byte("a\nb"), nil }, nil},
		"engine exit":           {kernel.Result{ExitCode: 1}, ok, nil},
		"garbage":               {kernel.Result{Stdout: []byte("I think it looks fine!")}, ok, review.ErrUnparseable},
	} {
		core := &fakeCICore{res: tt.res}
		_, err := NewLocalCI(core, review.RuntimeClaude, tt.cred).Review(context.Background(), "pr-12-0123456", "p")
		if err == nil || (tt.want != nil && !errors.Is(err, tt.want)) {
			t.Errorf("%s: error = %v", name, err)
		}
	}
}

func TestOrcaArgv(t *testing.T) {
	t.Parallel()
	r := &kerneltest.Runner{Result: kernel.Result{Stdout: []byte(`{"id":"t1"}`)}}
	o := NewOrca(r)
	out, err := o.CreateTerminal(context.Background(), "gbai-review-1-0123456", "gbai x | gbai y")
	if err != nil || out != `{"id":"t1"}` {
		t.Fatalf("CreateTerminal() = %q, %v", out, err)
	}
	if err := o.Send(context.Background(), "run:abc", "PR #1 head x verdict=clean merge=none"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"orca", "terminal", "create", "--title", "gbai-review-1-0123456", "--command", "gbai x | gbai y", "--json"},
		{"orca", "orchestration", "send", "--to", "run:abc", "--subject", "PR #1 head x verdict=clean merge=none", "--type", "status"},
	}
	for i, w := range want {
		if got := argv(r.Calls[i]); !slices.Equal(got, w) {
			t.Errorf("call %d = %q", i, got)
		}
	}
}

// A run with --keep-worktree leaves its worktree behind; a later run for the
// same PR and sha must replace it instead of failing on the existing path.
func TestGitAddWorktreeReplacesAKeptWorktree(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("runs the real git binary")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	work := t.TempDir()
	clone := filepath.Join(work, "clone")
	for _, args := range [][]string{
		{"init", "-q", clone},
		{"-C", clone, "-c", "user.name=t", "-c", "user.email=t@localhost", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "init"},
		{"-C", clone, "update-ref", "refs/remotes/origin/dev", "HEAD"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	g := NewGit(execrunner.New(), "o/r", work, nil)
	ctx := context.Background()
	name := "pr-12-0123456"
	if err := g.AddWorktree(ctx, name, "dev"); err != nil {
		t.Fatalf("first AddWorktree() error = %v", err)
	}
	leftover := filepath.Join(work, "worktrees", name, "leftover")
	if err := os.WriteFile(leftover, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := g.AddWorktree(ctx, name, "dev"); err != nil {
		t.Fatalf("rerun AddWorktree() over a kept worktree error = %v", err)
	}
	if _, err := os.Stat(leftover); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("kept worktree content survived the rerun: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clone, ".git")); err != nil {
		t.Fatalf("clone damaged: %v", err)
	}
}

// gitRepo runs git for test fixtures.
func gitRepo(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// The reviewer refuses to work from a state clone that is not a clean
// checkout of the integration branch, like vulcam's "clean dev" rule.
func TestGitGuardRefusesADirtyOrMovedClone(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("runs the real git binary")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	commit := []string{"-c", "user.name=t", "-c", "user.email=t@localhost", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "init"}
	setup := map[string]func(t *testing.T, state, clone string){
		"fresh state": func(*testing.T, string, string) {},
		"clean":       func(*testing.T, string, string) {},
		"untracked file": func(t *testing.T, _, clone string) {
			if err := os.WriteFile(filepath.Join(clone, "x"), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"ignored file": func(t *testing.T, _, clone string) {
			if err := os.WriteFile(filepath.Join(clone, ".git", "info", "exclude"), []byte("node_modules/\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(clone, "node_modules"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(clone, "node_modules", "y"), []byte("y"), 0o600); err != nil {
				t.Fatal(err)
			}
		},
		"other branch": func(t *testing.T, _, clone string) { gitRepo(t, "-C", clone, "checkout", "-q", "-b", "feature") },
		"detached":     func(t *testing.T, _, clone string) { gitRepo(t, "-C", clone, "checkout", "-q", "--detach") },
		"not a repo inside another repo": func(t *testing.T, state, clone string) {
			gitRepo(t, "init", "-q", "-b", "dev", state)
			if err := os.MkdirAll(clone, 0o700); err != nil {
				t.Fatal(err)
			}
		},
	}
	want := map[string]bool{"fresh state": true, "clean": true}
	for name, prepare := range setup {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			state := t.TempDir()
			clone := filepath.Join(state, "clone")
			if name != "fresh state" && name != "not a repo inside another repo" {
				gitRepo(t, "init", "-q", "-b", "dev", clone)
				gitRepo(t, append([]string{"-C", clone}, commit...)...)
			}
			prepare(t, state, clone)
			err := NewGit(execrunner.New(), "o/r", state, nil).Guard(context.Background(), "dev")
			if want[name] {
				if err != nil {
					t.Fatalf("Guard() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, review.ErrDirtyClone) {
				t.Fatalf("Guard() error = %v, want ErrDirtyClone", err)
			}
		})
	}
}

// A force-pushed (rewritten) integration branch cannot be fast-forwarded;
// the reviewer's own clean mirror must follow it instead of failing every
// later review on the same refusal.
func TestGitPrepareFollowsARewrittenIntegrationBranch(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("runs the real git binary")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	work := t.TempDir()
	origin := filepath.Join(work, "origin.git")
	seed := filepath.Join(work, "seed")
	state := filepath.Join(work, "state")
	commit := []string{"-c", "user.name=t", "-c", "user.email=t@localhost", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m"}
	gitRepo(t, "init", "-q", "--bare", origin)
	gitRepo(t, "init", "-q", "-b", "dev", seed)
	gitRepo(t, append([]string{"-C", seed}, append(commit, "one")...)...)
	gitRepo(t, "-C", seed, "push", "-q", origin, "dev:refs/heads/dev", "dev:refs/pull/12/head")
	gitRepo(t, "clone", "-q", "--branch", "dev", origin, filepath.Join(state, "clone"))

	// Rewrite dev on the origin with an unrelated history.
	gitRepo(t, "-C", seed, "checkout", "-q", "--orphan", "rewritten")
	gitRepo(t, append([]string{"-C", seed}, append(commit, "rewritten")...)...)
	gitRepo(t, "-C", seed, "push", "-q", "--force", origin, "rewritten:refs/heads/dev", "rewritten:refs/pull/12/head")

	g := NewGit(execrunner.New(), "o/r", state, nil)
	if err := g.Prepare(context.Background(), 12, "dev"); err != nil {
		t.Fatalf("Prepare() after a force push error = %v", err)
	}
	head, err := exec.Command("git", "-C", filepath.Join(state, "clone"), "log", "-1", "--format=%s").Output()
	if err != nil || strings.TrimSpace(string(head)) != "rewritten" {
		t.Fatalf("clone HEAD = %q, %v; want the rewritten dev", head, err)
	}
	if err := g.Guard(context.Background(), "dev"); err != nil {
		t.Fatalf("Guard() after following the rewrite = %v", err)
	}
}
