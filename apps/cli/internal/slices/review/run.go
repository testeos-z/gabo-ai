package review

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"unicode/utf8"
)

const (
	defaultMaxDiffBytes = 200 << 10
	checkTailLines      = 60
	statusTests         = "reviewer/tests"
	statusReview        = "reviewer/review"
)

var (
	//go:embed prompt.md.tmpl
	promptText     string
	promptTemplate = template.Must(template.New("prompt").Parse(promptText))
)

// RunConfig configures the reviewer-side run.
type RunConfig struct {
	Checks       [][]string // argv of each check, run in order until one fails
	Policy       MergePolicy
	MaxDiffBytes int // diff budget in the prompt; 0 means the default
	// KeepWorktree leaves the worktree and the checks workspace in place
	// for debugging; by default both are removed.
	KeepWorktree bool
	// Login, when set, is the owner's configured GitHub login; the gh login
	// the reviewer posts with must match it or the merge gate stays closed.
	Login string
}

// Report is the outcome of a run: the summary line plus diagnostic notes
// meant for stderr.
type Report struct {
	Summary Summary
	Notes   []string
}

func (r *Report) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

// RunUseCase reviews one pinned PR head inside the reviewer boundary.
type RunUseCase struct {
	lock   Lock
	gh     GitHub
	git    Git
	checks Checks
	engine Engine
	cfg    RunConfig
}

// NewRunUseCase wires the run use case with its ports.
func NewRunUseCase(gh GitHub, git Git, checks Checks, engine Engine, cfg RunConfig) *RunUseCase {
	if cfg.MaxDiffBytes <= 0 {
		cfg.MaxDiffBytes = defaultMaxDiffBytes
	}
	return &RunUseCase{gh: gh, git: git, checks: checks, engine: engine, cfg: cfg}
}

// WithLock serializes runs through l; without it runs are not serialized.
func (u *RunUseCase) WithLock(l Lock) *RunUseCase {
	u.lock = l
	return u
}

// WorkspaceName names the worktree and the checks workspace of a review.
func WorkspaceName(number int, sha string) string {
	return fmt.Sprintf("pr-%d-%s", number, sha[:7])
}

// Run reviews PR number at sha. It returns an error for invalid input and
// when it refuses to run because its own state clone is not a clean checkout
// of the integration branch (nothing is read or posted then); every other
// failure is reported as verdict=error in the summary.
func (u *RunUseCase) Run(ctx context.Context, number int, sha string) (Report, error) {
	if err := errors.Join(ValidatePR(number), ValidateSHA(sha)); err != nil {
		return Report{}, err
	}
	r := Report{Summary: Summary{PR: number, SHA: sha, Verdict: VerdictError, Merge: MergeNone}}
	if u.lock != nil {
		release, err := u.lock.Acquire(ctx)
		if err != nil {
			r.note("acquire review lock: %v", err)
			return r, nil
		}
		defer release()
	}
	if err := u.git.Guard(ctx, u.cfg.Policy.Integration); err != nil {
		return Report{}, fmt.Errorf("refusing to review: %w", err)
	}

	pr, err := u.gh.PullRequest(ctx, number)
	if err != nil {
		r.note("read PR: %v", err)
		return r, nil
	}
	if !strings.EqualFold(pr.State, "open") || pr.HeadSHA != sha {
		r.Summary.Verdict, r.Summary.Merge = VerdictSkipped, MergeSkipped
		r.note("PR state %s head %s: nothing to review for %s", pr.State, pr.HeadSHA, sha)
		return r, nil
	}

	ev, files := u.evaluate(ctx, number, sha, &r)
	verdict := DecideVerdict(ev)
	if ev.Infra != nil {
		r.note("review error: %v", ev.Infra)
	}
	post := true
	if ev.Conflict {
		if login, comments, err := u.thread(ctx, number); err != nil {
			r.note("read review thread: %v", err)
		} else if ConflictNoticeFor(comments, login, sha, u.cfg.Policy.Integration) {
			post = false
			r.note("conflict for %s already reported; comment not repeated", sha)
		}
	}
	if err := u.publish(ctx, number, sha, ev, verdict, post); err != nil {
		r.note("publish review: %v", err)
		return r, nil
	}
	r.Summary.Verdict = verdict
	var login string
	var comments []Comment
	if verdict == VerdictClean {
		if login, comments, err = u.thread(ctx, number); err != nil {
			r.note("read review thread: %v", err)
			login, comments = "", nil // an unknown author or thread never opens the merge gate
		}
	}
	r.Summary.Merge = DecideMerge(verdict, pr, files, comments, login, u.cfg.Policy)
	if r.Summary.Merge == MergeMerged {
		if err := u.merge(ctx, number, sha); err != nil {
			r.note("merge: %v", err)
			r.Summary.Merge = MergeNeedsHuman
		}
	}
	return r, nil
}

func (u *RunUseCase) evaluate(ctx context.Context, number int, sha string, r *Report) (Evidence, []string) {
	integration := u.cfg.Policy.Integration
	name := WorkspaceName(number, sha)
	cleanupCtx := context.WithoutCancel(ctx)

	if err := u.git.Prepare(ctx, number, integration); err != nil {
		return Evidence{Infra: fmt.Errorf("prepare clone: %w", err)}, nil
	}
	if u.cfg.KeepWorktree {
		r.note("keeping worktree and checks workspace %s", name)
	} else {
		defer func() {
			if err := u.git.RemoveWorktree(cleanupCtx, name); err != nil {
				r.note("remove worktree %s: %v", name, err)
			}
		}()
	}
	if err := u.git.AddWorktree(ctx, name, integration); err != nil {
		return Evidence{Infra: fmt.Errorf("add worktree: %w", err)}, nil
	}
	conflict, err := u.git.MergePinned(ctx, name, sha)
	if err != nil {
		return Evidence{Infra: fmt.Errorf("merge %s: %w", sha, err)}, nil
	}
	if conflict {
		return Evidence{Conflict: true, Findings: []Finding{{
			Severity: SeverityBlocking,
			Title:    "merge conflict with " + integration + ": rebase required",
		}}}, nil
	}
	snap, err := u.git.Snapshot(ctx, name, integration, sha)
	if err != nil {
		return Evidence{Infra: fmt.Errorf("snapshot: %w", err)}, nil
	}

	if !u.cfg.KeepWorktree {
		defer func() {
			if err := u.checks.Cleanup(cleanupCtx, name); err != nil {
				r.note("clean workspace %s: %v", name, err)
			}
		}()
	}
	if err := u.checks.Prepare(ctx, name, snap.Tree); err != nil {
		return Evidence{Infra: fmt.Errorf("prepare checks workspace: %w", err)}, snap.Files
	}
	ev := Evidence{}
	for _, argv := range u.cfg.Checks {
		out, err := u.checks.Run(ctx, name, argv)
		if err != nil {
			return Evidence{Infra: fmt.Errorf("run %s: %w", strings.Join(argv, " "), err)}, snap.Files
		}
		ev.Checks = append(ev.Checks, CheckResult{Command: strings.Join(argv, " "), Passed: out.Passed, Tail: Tail(out.Output, checkTailLines)})
		if !out.Passed {
			break
		}
	}

	prompt, err := u.prompt(number, snap)
	if err != nil {
		return Evidence{Infra: err}, snap.Files
	}
	findings, err := u.engine.Review(ctx, name, prompt)
	if err != nil {
		ev.Infra = fmt.Errorf("review engine: %w", err)
		return ev, snap.Files
	}
	ev.Findings = findings
	return ev, snap.Files
}

func (u *RunUseCase) prompt(number int, snap Snapshot) (string, error) {
	diff, truncated := snap.Diff, false
	if len(diff) > u.cfg.MaxDiffBytes {
		diff, truncated = diff[:u.cfg.MaxDiffBytes], true
		for !utf8.ValidString(diff) {
			diff = diff[:len(diff)-1]
		}
	}
	nonce, err := newNonce()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	err = promptTemplate.Execute(&b, map[string]any{
		"Number": number, "Integration": u.cfg.Policy.Integration, "Files": snap.Files,
		"Diff": diff, "Truncated": truncated, "MaxBytes": u.cfg.MaxDiffBytes, "Nonce": nonce,
	})
	if err != nil {
		return "", fmt.Errorf("render prompt: %w", err)
	}
	return b.String(), nil
}

// newNonce returns 128 random bits in hex. It names the diff delimiters of
// one prompt, so a diff cannot close them: it would have to guess the nonce.
func newNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("diff nonce: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// thread reads the reviewer's own login and the PR comments, oldest first.
func (u *RunUseCase) thread(ctx context.Context, number int) (string, []Comment, error) {
	login, err := u.gh.Login(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("reviewer login: %w", err)
	}
	if u.cfg.Login != "" && !strings.EqualFold(login, u.cfg.Login) {
		return "", nil, fmt.Errorf("gh is logged in as %s, but the configured owner login is %s", login, u.cfg.Login)
	}
	comments, err := u.gh.Comments(ctx, number)
	if err != nil {
		return "", nil, fmt.Errorf("list comments: %w", err)
	}
	return login, comments, nil
}

// publish sets both statuses and, when post is set, the verdict comment.
func (u *RunUseCase) publish(ctx context.Context, number int, sha string, ev Evidence, v Verdict, post bool) error {
	for _, s := range []Status{testsStatus(ev, len(u.cfg.Checks)), reviewStatus(v)} {
		if err := u.gh.SetStatus(ctx, sha, s); err != nil {
			return fmt.Errorf("status %s: %w", s.Context, err)
		}
	}
	if !post {
		return nil
	}
	if err := u.gh.Comment(ctx, number, CommentBody(sha, u.cfg.Policy.Integration, ev, v)); err != nil {
		return fmt.Errorf("comment: %w", err)
	}
	return nil
}

func (u *RunUseCase) merge(ctx context.Context, number int, sha string) error {
	if err := u.gh.Merge(ctx, number, sha); err != nil {
		return fmt.Errorf("merge pinned to %s: %w", sha, err)
	}
	return nil
}

func testsStatus(ev Evidence, configured int) Status {
	s := Status{Context: statusTests}
	switch {
	case ev.Conflict:
		s.State, s.Description = StatusFailure, "merge conflict: checks not run"
	case anyFailed(ev.Checks):
		s.State, s.Description = StatusFailure, "a check failed"
	case len(ev.Checks) == configured:
		s.State, s.Description = StatusSuccess, fmt.Sprintf("%d checks passed", configured)
	default:
		s.State, s.Description = StatusError, "checks could not complete"
	}
	return s
}

func reviewStatus(v Verdict) Status {
	s := Status{Context: statusReview}
	switch v {
	case VerdictClean:
		s.State, s.Description = StatusSuccess, "No blocking issues found."
	case VerdictBlocking:
		s.State, s.Description = StatusFailure, "Blocking issues found."
	default:
		s.State, s.Description = StatusError, "Review could not complete."
	}
	return s
}

func anyFailed(cs []CheckResult) bool {
	for _, c := range cs {
		if !c.Passed {
			return true
		}
	}
	return false
}

// CommentBody renders the single review comment. Its first line is exactly
// CommentClean for a clean verdict and CommentBlocking otherwise, its third
// line names the head (and the conflict, if any); it ends with CommentFooter
// and the Marker of sha.
func CommentBody(sha, integration string, ev Evidence, v Verdict) string {
	var b strings.Builder
	if v == VerdictClean {
		b.WriteString(CommentClean + "\n")
	} else {
		b.WriteString(CommentBlocking + "\n")
	}
	if ev.Conflict {
		b.WriteString("\n" + conflictLine(sha, integration) + "\n")
	} else {
		fmt.Fprintf(&b, "\nReviewed head `%s` merged onto `%s`; verdict `%s`.\n", sha, integration, v)
	}
	if ev.Infra != nil {
		b.WriteString("\nThe review could not complete; the reviewer logged the cause on the server.\n")
	}
	if len(ev.Checks) > 0 {
		b.WriteString("\n### Checks\n\n")
		for _, c := range ev.Checks {
			state := "passed"
			if !c.Passed {
				state = "failed"
			}
			fmt.Fprintf(&b, "- %s: `%s`\n", state, c.Command)
		}
	}
	if len(ev.Findings) > 0 {
		b.WriteString("\n### Findings\n\n")
		for _, f := range ev.Findings {
			loc := f.File
			if f.Line > 0 {
				loc = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			if loc != "" {
				loc = " `" + loc + "`"
			}
			fmt.Fprintf(&b, "- **%s**%s %s", f.Severity, loc, f.Title)
			if f.Detail != "" {
				b.WriteString(": " + f.Detail)
			}
			b.WriteString("\n")
		}
	}
	for _, c := range ev.Checks {
		if c.Passed {
			continue
		}
		fence := strings.Repeat("`", max(3, longestRun(c.Tail, '`')+1))
		fmt.Fprintf(&b, "\n<details><summary>Failed check: <code>%s</code> (last %d lines)</summary>\n\n%s\n%s\n%s\n\n</details>\n",
			template.HTMLEscapeString(c.Command), checkTailLines, fence, c.Tail, fence)
	}
	fmt.Fprintf(&b, "\n%s\n%s\n", CommentFooter, Marker(sha))
	return b.String()
}

func longestRun(s string, r rune) int {
	best, cur := 0, 0
	for _, c := range s {
		if c == r {
			cur++
			best = max(best, cur)
		} else {
			cur = 0
		}
	}
	return best
}
