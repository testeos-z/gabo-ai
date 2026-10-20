// Package review runs the reviewer flow: requesting a review from the coder's
// pane, running it inside the reviewer boundary, executing untrusted checks
// as reviewer-ci and notifying the coder's mailbox with one summary line.
package review

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// Verdict is the outcome of one review.
type Verdict string

// Verdicts.
const (
	VerdictClean    Verdict = "clean"
	VerdictBlocking Verdict = "blocking"
	VerdictError    Verdict = "error"
	VerdictSkipped  Verdict = "skipped"
)

// Merge is the merge decision taken after a review.
type Merge string

// Merge decisions.
const (
	MergeMerged     Merge = "merged"
	MergeNeedsHuman Merge = "needs-human-merge"
	MergeSkipped    Merge = "skipped"
	MergeNone       Merge = "none"
)

// Severity of a finding. Anything other than SeverityMinor blocks.
type Severity string

// Severities.
const (
	SeverityBlocking Severity = "blocking"
	SeverityMinor    Severity = "minor"
)

// CloudLabel marks PRs the reviewer may merge on its own.
const CloudLabel = "mode:cloud"

// Finding is one review comment from the engine (or the merge step).
type Finding struct {
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Title    string   `json:"title"`
	Detail   string   `json:"detail"`
}

// CheckResult is the outcome of one check command.
type CheckResult struct {
	Command string
	Passed  bool
	Tail    string
}

// PullRequest is the GitHub state the review relies on.
type PullRequest struct {
	Number     int
	State      string
	HeadSHA    string
	BaseBranch string
	Labels     []string
}

// Evidence is everything the verdict is computed from.
type Evidence struct {
	Infra    error
	Conflict bool
	Checks   []CheckResult
	Findings []Finding
}

// DecideVerdict computes the verdict from local evidence only.
func DecideVerdict(e Evidence) Verdict {
	if e.Infra != nil {
		return VerdictError
	}
	if e.Conflict {
		return VerdictBlocking
	}
	for _, c := range e.Checks {
		if !c.Passed {
			return VerdictBlocking
		}
	}
	for _, f := range e.Findings {
		if f.Severity != SeverityMinor {
			return VerdictBlocking
		}
	}
	return VerdictClean
}

// MergePolicy holds the configuration that gates automatic merges.
type MergePolicy struct {
	AutoMerge      bool
	Integration    string
	HumanOnlyPaths []string
}

// DefaultHumanOnlyPaths lists the paths only a human may merge. Entries
// ending in "/" are root directories; others are file names matched at any depth.
func DefaultHumanOnlyPaths() []string {
	return []string{".claude/", ".github/", ".opencode/", "AGENTS.md", "CLAUDE.md", "docs/policies/", "tools/hook-tests/", "agents/"}
}

// DecideMerge returns merged only for a clean cloud PR into the integration
// branch with auto merge enabled and no human-only path changed.
func DecideMerge(v Verdict, pr PullRequest, files []string, p MergePolicy) Merge {
	switch v {
	case VerdictSkipped:
		return MergeSkipped
	case VerdictClean:
	default:
		return MergeNone
	}
	if !p.AutoMerge || pr.BaseBranch != p.Integration || !hasLabel(pr.Labels, CloudLabel) || len(files) == 0 {
		return MergeNeedsHuman
	}
	for _, f := range files {
		if IsHumanOnly(f, p.HumanOnlyPaths) {
			return MergeNeedsHuman
		}
	}
	return MergeMerged
}

// IsHumanOnly reports whether file is covered by one of the human-only entries.
func IsHumanOnly(file string, entries []string) bool {
	for _, e := range entries {
		if dir, ok := strings.CutSuffix(e, "/"); ok {
			if strings.HasPrefix(file, dir+"/") {
				return true
			}
			continue
		}
		if file == e || !strings.Contains(e, "/") && path.Base(file) == e {
			return true
		}
	}
	return false
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

// Summary is the one line printed by `review run` and relayed to the coder.
type Summary struct {
	PR      int
	SHA     string
	Verdict Verdict
	Merge   Merge
}

var summaryPattern = regexp.MustCompile(`^PR #([1-9][0-9]*) head ([0-9a-f]{40}) verdict=(clean|blocking|error|skipped) merge=(merged|needs-human-merge|skipped|none)$`)

func (s Summary) String() string {
	return fmt.Sprintf("PR #%d head %s verdict=%s merge=%s", s.PR, s.SHA, s.Verdict, s.Merge)
}

// ParseSummary parses a summary line exactly.
func ParseSummary(line string) (Summary, bool) {
	m := summaryPattern.FindStringSubmatch(line)
	if m == nil {
		return Summary{}, false
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return Summary{}, false
	}
	return Summary{PR: n, SHA: m[2], Verdict: Verdict(m[3]), Merge: Merge(m[4])}, true
}

// Input validation errors.
var (
	ErrInvalidPR      = errors.New("pr must be a positive integer")
	ErrInvalidSHA     = errors.New("sha must be 40 lowercase hex characters")
	ErrInvalidMailbox = errors.New("mailbox must be run:<id> or dispatch:<id>")
)

var (
	shaPattern     = regexp.MustCompile(`^[0-9a-f]{40}$`)
	mailboxPattern = regexp.MustCompile(`^(run|dispatch):[A-Za-z0-9._-]+$`)
)

// ValidatePR checks a PR number.
func ValidatePR(n int) error {
	if n <= 0 {
		return fmt.Errorf("%w, got %d", ErrInvalidPR, n)
	}
	return nil
}

// ValidateSHA checks a full commit SHA.
func ValidateSHA(s string) error {
	if !shaPattern.MatchString(s) {
		return fmt.Errorf("%w, got %q", ErrInvalidSHA, s)
	}
	return nil
}

// ValidateMailbox checks an Orca mailbox address.
func ValidateMailbox(m string) error {
	if !mailboxPattern.MatchString(m) {
		return fmt.Errorf("%w, got %q", ErrInvalidMailbox, m)
	}
	return nil
}

// Tail returns the last n lines of out.
func Tail(out []byte, n int) string {
	s := strings.TrimRight(string(out), "\n")
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
