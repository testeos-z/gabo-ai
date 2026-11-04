package skills

import (
	"io/fs"
	"strings"
	"testing"
)

// Intent 0002 S12 and S3 (plan T3): a wave, auto mode included, never starts
// with a `codex` or `opencode` reviewer that has no isolation test pass. The
// cycle runs the user-level reviewer preflight, which reads the pass marker
// (T2), before it claims any task; `claude` and `pi` keep the current start.
func TestCycleRunsReviewerPreflightBeforeClaiming(t *testing.T) {
	t.Parallel()
	b, err := fs.ReadFile(Catalog(), "gabo-cycle/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := string(b)

	const run = "`bash \"${XDG_DATA_HOME:-$HOME/.local/share}/gabo/skills/gabo-reviewer/scripts/reviewer.sh\" preflight`"
	for _, s := range []string{
		// which engines, from which key, in which modes.
		"`runtimes.reviewer`", "is `codex` or `opencode`", "auto mode included",
		// the user-level copy, never a repository copy.
		run, "user-level copy", "never a copy from a repository checkout",
		// before any claim; a failure starts nothing and reports the line.
		"before you claim any task", "start no task", "claim nothing",
		"`PREFLIGHT-FAILED`", "names the missing isolation test",
		// claude and pi start as before.
		"`claude` and `pi` reviewers start as before",
		// the only way forward is the configuration-time isolation test.
		"`gabo-reviewer-config`", "no other way around",
	} {
		if !strings.Contains(doc, s) {
			t.Errorf("gabo-cycle/SKILL.md: missing %q", s)
		}
	}

	// The preflight comes before the step that claims the intent's tasks.
	steps := doc[strings.Index(doc, "## Execution Steps"):]
	pre, claim := strings.Index(steps, run), strings.Index(steps, "Claim each")
	if pre < 0 || claim < 0 || pre > claim {
		t.Errorf("gabo-cycle/SKILL.md: the reviewer preflight (at %d) must come before `Claim each` (at %d) in the Execution Steps", pre, claim)
	}

	// No way around the check: no engine override, no skip, no isolation run
	// or marker write during a wave.
	for _, s := range []string{
		"REVIEWER_ENGINE", "isolation-test.sh", "skip the preflight", "without the preflight",
		"--no-preflight", "reviewer-isolation/", "write the marker",
	} {
		if strings.Contains(doc, s) {
			t.Errorf("gabo-cycle/SKILL.md: must not say %q", s)
		}
	}
}
