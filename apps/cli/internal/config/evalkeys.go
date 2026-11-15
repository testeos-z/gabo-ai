package config

import "fmt"

// Defaults of the eval counts (intent 0030 S2, S4). The pass threshold and the
// cap have no default (S8, S9): unset is a state of its own, never a number.
const (
	DefaultEvalMinTasks    = 2
	DefaultEvalMaxTasks    = 10
	DefaultEvalRunsPerTask = 1
)

// Eval configures the agent eval set (intent 0030). Each key is optional; a
// pointer tells an unset value from a set one, and no value is filled in.
type Eval struct {
	// MinTasks is the fewest accepted tasks a run accepts (S2).
	MinTasks *int `yaml:"min_tasks,omitempty"`
	// MaxTasks is the most accepted tasks in the set (S2).
	MaxTasks *int `yaml:"max_tasks,omitempty"`
	// RunsPerTask is the number of runs per task (S4).
	RunsPerTask *int `yaml:"runs_per_task,omitempty"`
	// PassThreshold is the pass rate, from 0 to 1, below which a complete run
	// posts a notice (S9).
	PassThreshold *float64 `yaml:"pass_threshold,omitempty"`
	// Cap bounds one eval run under the cap rule of intent 0020 (S8).
	Cap EvalCap `yaml:"cap,omitempty"`
}

// EvalCap holds the caps of one eval run (S8). Unset means no cap.
type EvalCap struct {
	TurnsPerTaskRun    *int `yaml:"turns_per_task_run,omitempty"`
	TaskRunsPerEvalRun *int `yaml:"task_runs_per_eval_run,omitempty"`
}

// EvalMinTasks is eval.min_tasks with its default.
func (c Config) EvalMinTasks() int { return intOr(c.Eval.MinTasks, DefaultEvalMinTasks) }

// EvalMaxTasks is eval.max_tasks with its default.
func (c Config) EvalMaxTasks() int { return intOr(c.Eval.MaxTasks, DefaultEvalMaxTasks) }

// EvalRunsPerTask is eval.runs_per_task with its default.
func (c Config) EvalRunsPerTask() int { return intOr(c.Eval.RunsPerTask, DefaultEvalRunsPerTask) }

// EvalPassThreshold is eval.pass_threshold; ok is false when it is not set.
func (c Config) EvalPassThreshold() (float64, bool) {
	if c.Eval.PassThreshold == nil {
		return 0, false
	}
	return *c.Eval.PassThreshold, true
}

// EvalCapTurnsPerTaskRun is eval.cap.turns_per_task_run; ok is false when it
// is not set.
func (c Config) EvalCapTurnsPerTaskRun() (int, bool) { return setInt(c.Eval.Cap.TurnsPerTaskRun) }

// EvalCapTaskRunsPerEvalRun is eval.cap.task_runs_per_eval_run; ok is false
// when it is not set.
func (c Config) EvalCapTaskRunsPerEvalRun() (int, bool) {
	return setInt(c.Eval.Cap.TaskRunsPerEvalRun)
}

func intOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func setInt(p *int) (int, bool) {
	if p == nil {
		return 0, false
	}
	return *p, true
}

// validateEvalKeys checks the eval block: every set count is at least 1, the
// threshold is a number from 0 to 1, and the effective minimum is at most the
// effective maximum (the defaults count, so a maximum of 1 alone is refused).
func (c Config) validateEvalKeys() []error {
	var errs []error
	fail := func(field, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s", field, fmt.Sprintf(format, args...)))
	}
	atLeastOne := func(field string, p *int) {
		if p != nil && *p < 1 {
			fail(field, "must be at least 1, got %d", *p)
		}
	}
	atLeastOne("eval.min_tasks", c.Eval.MinTasks)
	atLeastOne("eval.max_tasks", c.Eval.MaxTasks)
	atLeastOne("eval.runs_per_task", c.Eval.RunsPerTask)
	atLeastOne("eval.cap.turns_per_task_run", c.Eval.Cap.TurnsPerTaskRun)
	atLeastOne("eval.cap.task_runs_per_eval_run", c.Eval.Cap.TaskRunsPerEvalRun)
	if t := c.Eval.PassThreshold; t != nil && !(*t >= 0 && *t <= 1) {
		fail("eval.pass_threshold", "must be from 0 to 1, got %v", *t)
	}
	lo, hi := c.EvalMinTasks(), c.EvalMaxTasks()
	if lo >= 1 && hi >= 1 && lo > hi {
		fail("eval.min_tasks", "must be at most eval.max_tasks (%d), got %d", hi, lo)
	}
	return errs
}
