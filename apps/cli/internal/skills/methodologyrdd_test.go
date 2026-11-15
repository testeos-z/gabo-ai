package skills

import (
	"strings"
	"testing"
)

const methodologyAdapter = ".claude/skills/gabo-methodology-gentle-ai/SKILL.md"

// Intent 0027 (plan T1) S1, S3 to S7, S10; D2, D4, D5: the methodology adapter
// carries the coder RDD rules as text the coders read.
func TestMethodologyAdapterCoderRDDRules(t *testing.T) {
	t.Parallel()
	doc := readRepoDoc(t, methodologyAdapter)

	t.Run("assess names the base and committed-only (S3, S7, D4)", func(t *testing.T) {
		requireIn(t, methodologyAdapter, doc, []string{
			"gentle-ai review assess --cwd <worktree> --base-ref <base> --committed-only --json",
			"the branch point for the first commit",
			"the last reviewed commit after it",
		})
		if strings.Contains(doc, "assess --cwd <worktree> --json") {
			t.Error("the tree-only assess form must not stay documented for committed work")
		}
	})

	t.Run("retry once then unavailable, never passive (S6)", func(t *testing.T) {
		requireIn(t, methodologyAdapter, doc, []string{
			"retry the same assess once on the same commit",
			"`unassessable`",
			"treat that commit as due",
			"report the RDD outcome `unavailable` with the error line",
			"never read as low risk and never reported as `passive`",
		})
	})

	t.Run("consent envelope returned unchanged (S4, D5)", func(t *testing.T) {
		requireIn(t, methodologyAdapter, doc, []string{
			"`gentle-ai.review-integration.consent/v3`",
			"never answer, grant, decline or synthesize",
			"return it unchanged",
			"open question",
		})
	})

	t.Run("no mode toggle outside a prohibition line (S4)", func(t *testing.T) {
		for _, l := range strings.Split(doc, "\n") {
			if strings.Contains(l, "review mode enable") || strings.Contains(l, "review mode disable") ||
				strings.Contains(l, "review mode enable|disable") {
				if !strings.Contains(strings.ToLower(l), "never") {
					t.Errorf("mode toggle outside a prohibition line: %q", l)
				}
			}
		}
		requireIn(t, methodologyAdapter, doc, []string{"Never run `gentle-ai review mode enable` or `gentle-ai review mode disable`"})
	})

	t.Run("reviewer is the merge gate (S5)", func(t *testing.T) {
		requireIn(t, methodologyAdapter, doc, []string{
			"the gabo reviewer is the only merge gate",
			"`VERDICT` line",
		})
	})

	t.Run("RED and GREEN or the exception (S1)", func(t *testing.T) {
		requireIn(t, methodologyAdapter, doc, []string{
			"names that test, its RED failure and its GREEN run",
			"states the exception and the checks it ran, and claims no RED or GREEN",
		})
	})

	t.Run("outcome vocabulary (D2, S3)", func(t *testing.T) {
		requireIn(t, methodologyAdapter, doc, []string{
			"`granted`", "`declined`", "`passive`", "`off`", "`unavailable`",
			"`review_due_reason` verbatim",
			"`risk`",
			"mode is `off`, the outcome is `off` and no assess command runs",
		})
	})
}
