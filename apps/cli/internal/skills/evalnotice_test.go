package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Intent 0030 S9, D6, PD4 and PD5 (plan T7): a complete run below the threshold
// posts one notice on its PR, and a complete run at or above it on a PR that
// carries a notice posts the cleared marker. The markers are pinned on both
// sides, the script and the skill text, so the merge hold (T8) and the runner
// cannot drift apart. The fixture tests (eval-run.test.sh) check the behavior.

// The marker lines the runner posts (D6, PD5). The reviewer's own marker is
// never one of them.
var evalNoticeMarkers = []string{
	"<!-- gabo-eval-notice sha=",
	"<!-- gabo-eval-cleared sha=",
}

func TestEvalNoticeScriptPostsTheMarkers(t *testing.T) {
	t.Parallel()
	script, err := os.ReadFile(filepath.Join(evalRunnerSkill, "scripts", "eval-run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range evalNoticeMarkers {
		if !strings.Contains(string(script), marker) {
			t.Errorf("eval-run.sh: the notice marker %q is missing", marker)
		}
	}
	if strings.Contains(string(script), "gbai-reviewer-bot") {
		t.Error("eval-run.sh names the reviewer's marker; a notice must never carry it (D6)")
	}
}

func TestEvalRunnerSkillStatesTheNotice(t *testing.T) {
	t.Parallel()
	b, err := os.ReadFile(filepath.Join(evalRunnerSkill, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	requireIn(t, "gabo-eval-runner/SKILL.md", string(b), []string{
		"## Notice on the PR (intent 0030 S9, D6, PD4, PD5)",
		"`--pr <n>`",
		"`<!-- gabo-eval-notice sha=<head> -->`",
		"`<!-- gabo-eval-cleared sha=<head> -->`",
		"`threshold: not set`",
		"`notice: posted`",
		"`notice: already posted for head`",
		"`notice: none (paused run",
		"never on a paused run",
		"the only write",
		"`notice: none (baseline: none",
		"no accepted baseline",
		"`gh api user`",
		"only comments by the authenticated login",
		"Conservative rule",
		"never posted on a head that carries its own notice",
		"only clears an earlier head's notice",
	})
}
