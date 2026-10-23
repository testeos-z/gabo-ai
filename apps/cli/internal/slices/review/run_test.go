package review

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const wsName = "pr-12-0123456"

type harness struct {
	t      *trace
	gh     *fakeGitHub
	git    *fakeGit
	checks *fakeChecks
	engine *fakeEngine
	cfg    RunConfig
}

func newHarness() *harness {
	tr := &trace{}
	return &harness{
		t: tr,
		gh: &fakeGitHub{t: tr, pr: PullRequest{
			Number: 12, State: "OPEN", HeadSHA: sha, BaseBranch: "dev", Labels: []string{CloudLabel},
		}},
		git: &fakeGit{t: tr, snapshot: Snapshot{
			Files: []string{"apps/web/a.ts"}, Diff: "diff --git a/apps/web/a.ts b/apps/web/a.ts\n+ok\n", Tree: []byte("TAR"),
		}},
		checks: &fakeChecks{t: tr},
		engine: &fakeEngine{t: tr, findings: []Finding{{Severity: SeverityMinor, File: "apps/web/a.ts", Line: 3, Title: "nit", Detail: "rename"}}},
		cfg: RunConfig{
			Checks: [][]string{{"bun", "install", "--frozen-lockfile"}, {"bun", "run", "test"}},
			Policy: MergePolicy{AutoMerge: true, Integration: "dev", HumanOnlyPaths: DefaultHumanOnlyPaths()},
		},
	}
}

func (h *harness) run(t *testing.T) Report {
	t.Helper()
	uc := NewRunUseCase(h.gh, h.git, h.checks, h.engine, h.cfg)
	r, err := uc.Run(context.Background(), 12, sha)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	return r
}

func (h *harness) assertCleanedUp(t *testing.T) {
	t.Helper()
	if !h.t.has("git.add") {
		return
	}
	if len(h.git.removed) != 1 || h.git.removed[0] != wsName {
		t.Errorf("worktree removed = %v, want [%s]; calls %v", h.git.removed, wsName, h.t.calls)
	}
	if h.t.has("checks.prepare") && (len(h.checks.cleaned) != 1 || h.checks.cleaned[0] != wsName) {
		t.Errorf("workspace cleaned = %v", h.checks.cleaned)
	}
}

func TestRunCleanCloudPRMergesPinnedToSHA(t *testing.T) {
	t.Parallel()
	h := newHarness()
	r := h.run(t)
	if got := r.Summary.String(); got != "PR #12 head "+sha+" verdict=clean merge=merged" {
		t.Fatalf("summary = %q", got)
	}
	if h.gh.mergedAt != sha || h.t.has("gh.approve") {
		t.Fatalf("merged at %q, calls %v; want a pinned merge and no approve", h.gh.mergedAt, h.t.calls)
	}
	if string(h.checks.tree) != "TAR" {
		t.Fatalf("checks got tree %q", h.checks.tree)
	}
	if len(h.gh.comments) != 1 || !strings.HasPrefix(h.gh.comments[0], "No blocking issues found.\n") ||
		!strings.HasSuffix(h.gh.comments[0], "\n"+Marker(sha)+"\n") {
		t.Fatalf("comments = %q", h.gh.comments)
	}
	want := []Status{{Context: "reviewer/tests", State: StatusSuccess}, {Context: "reviewer/review", State: StatusSuccess}}
	for i, s := range h.gh.statuses {
		if s.Context != want[i].Context || s.State != want[i].State {
			t.Errorf("status %d = %+v", i, s)
		}
	}
	order := strings.Join(h.t.calls, "\n")
	for _, pair := range [][2]string{{"git.merge", "checks.prepare"}, {"gh.status", "gh.merge"}, {"gh.comment ", "gh.list-comments"}, {"gh.list-comments", "gh.merge"}} {
		if strings.Index(order, pair[0]) > strings.Index(order, pair[1]) {
			t.Errorf("%s must happen before %s:\n%s", pair[0], pair[1], order)
		}
	}
	if !strings.Contains(h.engine.prompt, "apps/web/a.ts") || !strings.Contains(h.engine.prompt, "+ok") {
		t.Errorf("prompt lacks files or diff:\n%s", h.engine.prompt)
	}
	h.assertCleanedUp(t)
}

func TestRunSkipsWhenHeadMovedOrClosed(t *testing.T) {
	t.Parallel()
	for name, pr := range map[string]PullRequest{
		"head moved": {Number: 12, State: "OPEN", HeadSHA: strings.Repeat("f", 40), BaseBranch: "dev"},
		"closed":     {Number: 12, State: "CLOSED", HeadSHA: sha, BaseBranch: "dev"},
		"merged":     {Number: 12, State: "MERGED", HeadSHA: sha, BaseBranch: "dev"},
	} {
		h := newHarness()
		h.gh.pr = pr
		r := h.run(t)
		if r.Summary.Verdict != VerdictSkipped || r.Summary.Merge != MergeSkipped {
			t.Errorf("%s: summary = %s", name, r.Summary)
		}
		if len(h.t.calls) != 1 {
			t.Errorf("%s: calls = %v, want only the PR read", name, h.t.calls)
		}
	}
}

func TestRunConflictIsBlockingAndRemovesWorktree(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.git.conflict = true
	r := h.run(t)
	if r.Summary.Verdict != VerdictBlocking || r.Summary.Merge != MergeNone {
		t.Fatalf("summary = %s", r.Summary)
	}
	if h.t.has("checks.") || h.t.has("engine.") || h.t.has("gh.merge") || h.t.has("gh.approve") {
		t.Fatalf("conflict must stop before checks, engine and merge: %v", h.t.calls)
	}
	if len(h.gh.comments) != 1 || !strings.HasPrefix(h.gh.comments[0], "Blocking issues found.\n") ||
		!strings.Contains(h.gh.comments[0], "merge conflict with dev: rebase required") {
		t.Fatalf("comment = %q", h.gh.comments)
	}
	h.assertCleanedUp(t)
}

func TestRunFailedCheckIsBlockingWithLogs(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.checks.fail = map[string]bool{"bun install --frozen-lockfile": true}
	r := h.run(t)
	if r.Summary.Verdict != VerdictBlocking || r.Summary.Merge != MergeNone {
		t.Fatalf("summary = %s", r.Summary)
	}
	if h.t.has("checks.run " + wsName + " bun run test") {
		t.Errorf("checks after the first failure must not run: %v", h.t.calls)
	}
	body := h.gh.comments[0]
	if !strings.Contains(body, "<details>") || !strings.Contains(body, "FAIL bun install --frozen-lockfile") {
		t.Fatalf("comment lacks failed log:\n%s", body)
	}
	if h.gh.statuses[0].State != StatusFailure || h.gh.statuses[1].State != StatusFailure {
		t.Fatalf("statuses = %+v", h.gh.statuses)
	}
	h.assertCleanedUp(t)
}

func TestRunBlockingFindingNeverMerges(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.engine.findings = []Finding{{Severity: SeverityBlocking, File: "a.ts", Line: 1, Title: "SQL injection", Detail: "user input"}}
	r := h.run(t)
	if r.Summary.Verdict != VerdictBlocking || h.t.has("gh.merge") {
		t.Fatalf("summary = %s, calls %v", r.Summary, h.t.calls)
	}
	if !strings.Contains(h.gh.comments[0], "SQL injection") {
		t.Fatalf("comment = %q", h.gh.comments[0])
	}
	h.assertCleanedUp(t)
}

func TestRunEngineGarbageIsError(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.engine.err = ErrUnparseable
	r := h.run(t)
	if r.Summary.Verdict != VerdictError || r.Summary.Merge != MergeNone || h.t.has("gh.merge") {
		t.Fatalf("summary = %s", r.Summary)
	}
	if h.gh.statuses[1].State != StatusError {
		t.Fatalf("review status = %+v", h.gh.statuses[1])
	}
	h.assertCleanedUp(t)
}

func TestRunInfraFailuresAreErrorsAndAlwaysCleanUp(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	for name, mutate := range map[string]func(*harness){
		"prepare":        func(h *harness) { h.git.prepareErr = boom },
		"add worktree":   func(h *harness) { h.git.addErr = boom },
		"merge":          func(h *harness) { h.git.mergeErr = boom },
		"snapshot":       func(h *harness) { h.git.snapshotErr = boom },
		"checks prepare": func(h *harness) { h.checks.prepareErr = boom },
		"engine":         func(h *harness) { h.engine.err = boom },
		"status post":    func(h *harness) { h.gh.statusErr = boom },
	} {
		h := newHarness()
		mutate(h)
		r := h.run(t)
		if r.Summary.Verdict != VerdictError || r.Summary.Merge != MergeNone {
			t.Errorf("%s: summary = %s", name, r.Summary)
		}
		if h.t.has("gh.merge") || h.t.has("gh.approve") {
			t.Errorf("%s: merged on infra failure", name)
		}
		if name == "add worktree" && len(h.git.removed) != 1 {
			t.Errorf("%s: worktree not removed after failed add", name)
		}
		if len(r.Notes) == 0 {
			t.Errorf("%s: no note explaining the error", name)
		}
		h.assertCleanedUp(t)
	}
}

func TestRunMergeFailureNeedsHuman(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.gh.mergeErr = errors.New("head changed")
	r := h.run(t)
	if r.Summary.Verdict != VerdictClean || r.Summary.Merge != MergeNeedsHuman {
		t.Fatalf("summary = %s", r.Summary)
	}
}

func TestRunMergeDependsOnTheNewestMarkerComment(t *testing.T) {
	t.Parallel()
	staleClean := CommentBody(old, "dev", Evidence{}, VerdictClean)
	for name, mutate := range map[string]func(*harness){
		"list fails":        func(h *harness) { h.gh.listErr = errors.New("HTTP 502") },
		"no marker comment": func(h *harness) { h.gh.listed = []string{"No blocking issues found.\n"} },
		"newest marker is old": func(h *harness) {
			h.gh.listed = []string{CommentBody(sha, "dev", Evidence{}, VerdictClean), staleClean}
		},
		"newest marker blocks": func(h *harness) { h.gh.listed = []string{CommentBody(sha, "dev", Evidence{}, VerdictBlocking)} },
	} {
		h := newHarness()
		mutate(h)
		r := h.run(t)
		if r.Summary.Verdict != VerdictClean || r.Summary.Merge != MergeNeedsHuman || h.t.has("gh.merge") {
			t.Errorf("%s: summary = %s, calls %v", name, r.Summary, h.t.calls)
		}
	}
}

func TestRunNeedsHumanForHumanOnlyPaths(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.git.snapshot.Files = []string{".github/workflows/ci.yml"}
	r := h.run(t)
	if r.Summary.Merge != MergeNeedsHuman || h.t.has("gh.merge") || h.t.has("gh.approve") {
		t.Fatalf("summary = %s calls %v", r.Summary, h.t.calls)
	}
}

func TestRunBoundsDiffInPrompt(t *testing.T) {
	t.Parallel()
	h := newHarness()
	h.cfg.MaxDiffBytes = 100
	h.git.snapshot.Diff = strings.Repeat("y", 1000)
	h.run(t)
	if strings.Contains(h.engine.prompt, strings.Repeat("y", 101)) || !strings.Contains(h.engine.prompt, "diff truncated") {
		t.Fatalf("prompt not bounded:\n%s", h.engine.prompt)
	}
}

func TestRunRejectsInvalidInputBeforeIO(t *testing.T) {
	t.Parallel()
	h := newHarness()
	uc := NewRunUseCase(h.gh, h.git, h.checks, h.engine, h.cfg)
	for _, in := range []struct {
		n   int
		sha string
	}{{0, sha}, {12, "abc"}} {
		if _, err := uc.Run(context.Background(), in.n, in.sha); err == nil {
			t.Errorf("Run(%d, %q) error = nil", in.n, in.sha)
		}
	}
	if len(h.t.calls) != 0 {
		t.Fatalf("calls = %v", h.t.calls)
	}
}

type fakeLock struct {
	t   *trace
	err error
}

func (f *fakeLock) Acquire(context.Context) (func(), error) {
	f.t.add("lock.acquire")
	if f.err != nil {
		return nil, f.err
	}
	return func() { f.t.add("lock.release") }, nil
}

func TestRunIsSerializedByTheLock(t *testing.T) {
	t.Parallel()
	h := newHarness()
	uc := NewRunUseCase(h.gh, h.git, h.checks, h.engine, h.cfg).WithLock(&fakeLock{t: h.t})
	if _, err := uc.Run(context.Background(), 12, sha); err != nil {
		t.Fatal(err)
	}
	if h.t.calls[0] != "lock.acquire" || h.t.calls[1] != "gh.pr 12" || h.t.calls[len(h.t.calls)-1] != "lock.release" {
		t.Fatalf("calls = %v", h.t.calls)
	}
}

func TestRunLockFailureIsAnErrorVerdictWithoutIO(t *testing.T) {
	t.Parallel()
	h := newHarness()
	uc := NewRunUseCase(h.gh, h.git, h.checks, h.engine, h.cfg).WithLock(&fakeLock{t: h.t, err: errors.New("busy")})
	r, err := uc.Run(context.Background(), 12, sha)
	if err != nil || r.Summary.Verdict != VerdictError || len(h.t.calls) != 1 {
		t.Fatalf("report %+v err %v calls %v", r, err, h.t.calls)
	}
}
