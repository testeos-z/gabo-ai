package skills

import (
	"io/fs"
	"testing"
)

// Intent 0027 S5, S13, S14, S25 to S27 (plan T13): the reviewer notes, the
// reviewer skill and its README describe the same policy: REVIEW.md read from
// the integration branch, the three passes, the Important-only blocking rule,
// the nit cap, the comment order, the `code read:` line, and that RDD never
// replaces the reviewer.
func TestReviewerDocsNameThePolicy(t *testing.T) {
	t.Parallel()
	docs := []string{
		"gabo-reviewer-notes/SKILL.md",
		"gabo-reviewer/SKILL.md",
		"gabo-reviewer/README.md",
	}
	want := []string{
		"REVIEW.md",
		"integration branch (`dev`)",
		"never from the PR head",
		"review policy: not run: no policy file on `dev`",
		"fails closed",
		"three passes",
		"[bugs]",
		"[security]",
		"[compliance]",
		"Only an Important finding caused by the pull request is blocking",
		"at most five nits",
		"code read:",
		"`code read: pass`",
		"`code read: fail: <finding>`",
		"`code read: not run: <reason>`",
		"the `VERDICT` line, then the checks section, then the code read section",
		"RDD never replaces the reviewer",
	}
	for _, path := range docs {
		b, err := fs.ReadFile(content, path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		requireIn(t, path, string(b), want)
	}
}
