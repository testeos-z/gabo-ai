package config

import (
	"strings"
	"testing"
)

// I0030 S2, S4, S8, S9: the eval block loads with defaults only where the spec
// gives one (min 2, max 10, runs 1). The pass threshold and the cap have no
// default: unset is a distinct loaded state, never a number.
func TestEvalDefaultsWhenBlockAbsent(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := cfg.EvalMinTasks(); got != 2 {
		t.Errorf("EvalMinTasks() = %d, want the default 2", got)
	}
	if got := cfg.EvalMaxTasks(); got != 10 {
		t.Errorf("EvalMaxTasks() = %d, want the default 10", got)
	}
	if got := cfg.EvalRunsPerTask(); got != 1 {
		t.Errorf("EvalRunsPerTask() = %d, want the default 1", got)
	}
	if v, ok := cfg.EvalPassThreshold(); ok {
		t.Errorf("EvalPassThreshold() = %v, want not set", v)
	}
	if v, ok := cfg.EvalCapTurnsPerTaskRun(); ok {
		t.Errorf("EvalCapTurnsPerTaskRun() = %d, want not set", v)
	}
	if v, ok := cfg.EvalCapTaskRunsPerEvalRun(); ok {
		t.Errorf("EvalCapTaskRunsPerEvalRun() = %d, want not set", v)
	}
	out, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "eval") {
		t.Errorf("Marshal() added an eval block:\n%s", out)
	}
}

// Every eval key loads; the explicit values replace the defaults.
func TestParseEvalKeys(t *testing.T) {
	t.Parallel()
	doc := minimal + `eval:
  min_tasks: 3
  max_tasks: 12
  runs_per_task: 2
  pass_threshold: 1
  cap:
    turns_per_task_run: 1
    task_runs_per_eval_run: 1
`
	cfg, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.EvalMinTasks() != 3 || cfg.EvalMaxTasks() != 12 || cfg.EvalRunsPerTask() != 2 {
		t.Errorf("counts = %d, %d, %d; want 3, 12, 2", cfg.EvalMinTasks(), cfg.EvalMaxTasks(), cfg.EvalRunsPerTask())
	}
	if v, ok := cfg.EvalPassThreshold(); !ok || v != 1 {
		t.Errorf("EvalPassThreshold() = %v, %v; want 1, true", v, ok)
	}
	if v, ok := cfg.EvalCapTurnsPerTaskRun(); !ok || v != 1 {
		t.Errorf("EvalCapTurnsPerTaskRun() = %d, %v; want 1, true", v, ok)
	}
	if v, ok := cfg.EvalCapTaskRunsPerEvalRun(); !ok || v != 1 {
		t.Errorf("EvalCapTaskRunsPerEvalRun() = %d, %v; want 1, true", v, ok)
	}
}

// A threshold of 0 is set (true), not unset: the boundary is a value like any
// other, and the reader must tell it apart from "not set".
func TestEvalThresholdZeroIsSet(t *testing.T) {
	t.Parallel()
	cfg, err := Parse([]byte(minimal + "eval:\n  pass_threshold: 0\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if v, ok := cfg.EvalPassThreshold(); !ok || v != 0 {
		t.Errorf("EvalPassThreshold() = %v, %v; want 0, true", v, ok)
	}
}

// Each bad value gives one line naming its key, the rule and the value seen.
func TestParseRejectsBadEvalKeys(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, yaml, want string }{
		{"min zero", "min_tasks: 0", "eval.min_tasks: must be at least 1, got 0"},
		{"max zero", "max_tasks: 0", "eval.max_tasks: must be at least 1, got 0"},
		{"runs zero", "runs_per_task: 0", "eval.runs_per_task: must be at least 1, got 0"},
		{"min above max", "min_tasks: 5\n  max_tasks: 4", "eval.min_tasks: must be at most eval.max_tasks (4), got 5"},
		{"max below default min", "max_tasks: 1", "eval.min_tasks: must be at most eval.max_tasks (1), got 2"},
		{"threshold above one", "pass_threshold: 1.5", "eval.pass_threshold: must be from 0 to 1, got 1.5"},
		{"threshold negative", "pass_threshold: -0.1", "eval.pass_threshold: must be from 0 to 1, got -0.1"},
		{"cap turns zero", "cap:\n    turns_per_task_run: 0", "eval.cap.turns_per_task_run: must be at least 1, got 0"},
		{"cap runs zero", "cap:\n    task_runs_per_eval_run: 0", "eval.cap.task_runs_per_eval_run: must be at least 1, got 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(minimal + "eval:\n  " + tt.yaml + "\n"))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse() error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

// A non-number threshold and any unknown eval.* key fail the decode.
func TestParseRejectsMalformedEval(t *testing.T) {
	t.Parallel()
	for name, doc := range map[string]string{
		"threshold not a number": "eval:\n  pass_threshold: high\n",
		"unknown eval key":       "eval:\n  chain: 1\n",
		"unknown cap key":        "eval:\n  cap:\n    turns: 1\n",
		"eval not a mapping":     "eval: 3\n",
	} {
		if _, err := Parse([]byte(minimal + doc)); err == nil {
			t.Errorf("%s: Parse() error = nil", name)
		}
	}
}
