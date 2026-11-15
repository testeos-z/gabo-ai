package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0027 S25 to S27 (plan T7): the review policy lives in REVIEW.md at the
// repository root, with three passes, a definition of Important, a nit cap of
// five and a do-not-report list.
func TestReviewPolicyHasItsSections(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(repoRoot, "REVIEW.md"))
	if err != nil {
		t.Fatalf("REVIEW.md at the repository root: %v", err)
	}
	doc := string(b)
	requireIn(t, "REVIEW.md", doc, []string{
		"## Passes",
		"### Pass 1: bugs",
		"### Pass 2: security",
		"### Pass 3: compliance",
		"spec.md",
		"plan.md",
		"design principles",
		"## What Important means here",
		"Reserve Important for findings that would break behavior, leak data or breach a policy. Style and naming are nits.",
		"## Cap the nits",
		"Report at most five nits per review; summarize the rest as a count.",
		"## Do not report",
		"generated files",
		"review.checks",
	})
	if strings.Count(doc, "tag") == 0 {
		t.Errorf("REVIEW.md must tell the reviewer to tag each finding with its pass")
	}
}
