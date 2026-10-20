package review

import (
	"context"
	"fmt"
	"strings"
)

// trace records port calls across fakes in order.
type trace struct{ calls []string }

func (t *trace) add(format string, args ...any) { t.calls = append(t.calls, fmt.Sprintf(format, args...)) }

func (t *trace) has(prefix string) bool {
	for _, c := range t.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

type fakeGitHub struct {
	t          *trace
	pr         PullRequest
	prErr      error
	statusErr  error
	mergeErr   error
	statuses   []Status
	comments   []string
	approvedAt string
	mergedAt   string
}

func (f *fakeGitHub) PullRequest(_ context.Context, n int) (PullRequest, error) {
	f.t.add("gh.pr %d", n)
	return f.pr, f.prErr
}

func (f *fakeGitHub) SetStatus(_ context.Context, sha string, s Status) error {
	f.t.add("gh.status %s %s %s", sha, s.Context, s.State)
	f.statuses = append(f.statuses, s)
	return f.statusErr
}

func (f *fakeGitHub) Comment(_ context.Context, n int, body string) error {
	f.t.add("gh.comment %d", n)
	f.comments = append(f.comments, body)
	return nil
}

func (f *fakeGitHub) Approve(_ context.Context, n int, sha string) error {
	f.t.add("gh.approve %d %s", n, sha)
	f.approvedAt = sha
	return nil
}

func (f *fakeGitHub) Merge(_ context.Context, n int, sha string) error {
	f.t.add("gh.merge %d %s", n, sha)
	f.mergedAt = sha
	return f.mergeErr
}

type fakeGit struct {
	t           *trace
	prepareErr  error
	addErr      error
	conflict    bool
	mergeErr    error
	snapshot    Snapshot
	snapshotErr error
	removed     []string
}

func (f *fakeGit) Prepare(_ context.Context, n int, integration string) error {
	f.t.add("git.prepare %d %s", n, integration)
	return f.prepareErr
}

func (f *fakeGit) AddWorktree(_ context.Context, name, integration string) error {
	f.t.add("git.add %s %s", name, integration)
	return f.addErr
}

func (f *fakeGit) MergePinned(_ context.Context, name, sha string) (bool, error) {
	f.t.add("git.merge %s %s", name, sha)
	return f.conflict, f.mergeErr
}

func (f *fakeGit) Snapshot(_ context.Context, name, integration, sha string) (Snapshot, error) {
	f.t.add("git.snapshot %s %s %s", name, integration, sha)
	return f.snapshot, f.snapshotErr
}

func (f *fakeGit) RemoveWorktree(_ context.Context, name string) error {
	f.t.add("git.remove %s", name)
	f.removed = append(f.removed, name)
	return nil
}

type fakeChecks struct {
	t          *trace
	prepareErr error
	fail       map[string]bool
	tree       []byte
	cleaned    []string
}

func (f *fakeChecks) Prepare(_ context.Context, ws string, tree []byte) error {
	f.t.add("checks.prepare %s", ws)
	f.tree = tree
	return f.prepareErr
}

func (f *fakeChecks) Run(_ context.Context, ws string, argv []string) (CheckOutcome, error) {
	cmd := strings.Join(argv, " ")
	f.t.add("checks.run %s %s", ws, cmd)
	if f.fail[cmd] {
		return CheckOutcome{Output: []byte("FAIL " + cmd)}, nil
	}
	return CheckOutcome{Passed: true, Output: []byte("ok")}, nil
}

func (f *fakeChecks) Cleanup(_ context.Context, ws string) error {
	f.t.add("checks.cleanup %s", ws)
	f.cleaned = append(f.cleaned, ws)
	return nil
}

type fakeEngine struct {
	t        *trace
	findings []Finding
	err      error
	prompt   string
}

func (f *fakeEngine) Review(_ context.Context, ws, prompt string) ([]Finding, error) {
	f.t.add("engine.review %s", ws)
	f.prompt = prompt
	return f.findings, f.err
}

type fakeOrca struct {
	t        *trace
	err      error
	title    string
	command  string
	sentTo   string
	subjects []string
}

func (f *fakeOrca) CreateTerminal(_ context.Context, title, command string) (string, error) {
	f.t.add("orca.terminal %s", title)
	f.title, f.command = title, command
	return `{"id":"t1"}`, f.err
}

func (f *fakeOrca) Send(_ context.Context, to, subject string) error {
	f.t.add("orca.send %s", to)
	f.sentTo = to
	f.subjects = append(f.subjects, subject)
	return f.err
}
