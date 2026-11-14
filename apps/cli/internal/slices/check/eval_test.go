package check

import (
	"strings"
	"testing"
)

// I0030 S2, S4, S8, S9: gabo check reports the eval values the project sets
// (defaults marked) and flags nothing for eval when the keys are absent.
func TestEvalValuesReported(t *testing.T) {
	t.Parallel()
	doc := validConfig + "eval:\n  min_tasks: 3\n  pass_threshold: 1\n"
	r := run(t, project(), configFiles{projectFile: doc}, Settings{})
	if r.Failed {
		t.Fatalf("Failed = true, want false; output:\n%s", joined(r))
	}
	want := "eval: min_tasks 3, max_tasks 10 (default), runs_per_task 1 (default), pass_threshold 1, cap turns_per_task_run not set, cap task_runs_per_eval_run not set"
	wantLines(t, r, want, PassedLine)
}

// Without an eval block the output is the same as before: no eval line and
// nothing flagged.
func TestEvalAbsentPrintsNothing(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig}, Settings{})
	wantPassed(t, r, PassedLine)
	for _, l := range r.Lines {
		if strings.Contains(l, "eval") {
			t.Errorf("line %q, want no eval line when the block is absent", l)
		}
	}
}

// Threshold and cap unset are not flagged: the block may hold only counts.
func TestEvalUnsetThresholdAndCapAreNotFlagged(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig + "eval:\n  max_tasks: 10\n"}, Settings{})
	wantPassed(t, r, PassedLine)
	for _, l := range r.Lines {
		if strings.HasPrefix(l, "!") || strings.HasPrefix(l, "✗") {
			t.Errorf("line %q, want nothing flagged for unset threshold and cap", l)
		}
	}
}

// A bad eval value fails the check and names its key (the layer's schema
// check runs first, so the line reads as its minimum rule).
func TestEvalBadValueFailsNamingKey(t *testing.T) {
	t.Parallel()
	r := run(t, project(), configFiles{projectFile: validConfig + "eval:\n  runs_per_task: 0\n"}, Settings{})
	if !r.Failed || !strings.Contains(joined(r), "✗ "+projectFile+": eval.runs_per_task: minimum: got 0, want 1") {
		t.Fatalf("output:\n%s", joined(r))
	}
}
