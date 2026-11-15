package check

import (
	"fmt"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// checkEval reports the eval values the project sets (intent 0030 S2, S4, S8,
// S9): one line, the counts marked when they are defaults, and "not set" for a
// threshold or cap the owner has not set. It flags nothing: a config without
// an eval value prints no line, and a bad value already failed the config step.
func (u *UseCase) checkEval(r *Report, cfg config.Config) {
	if cfg.Eval.MinTasks == nil && cfg.Eval.MaxTasks == nil && cfg.Eval.RunsPerTask == nil &&
		cfg.Eval.PassThreshold == nil && cfg.Eval.Cap.TurnsPerTaskRun == nil && cfg.Eval.Cap.TaskRunsPerEvalRun == nil {
		return
	}
	count := func(name string, set bool, v int) string {
		if !set {
			return fmt.Sprintf("%s %d (default)", name, v)
		}
		return fmt.Sprintf("%s %d", name, v)
	}
	parts := []string{
		count("min_tasks", cfg.Eval.MinTasks != nil, cfg.EvalMinTasks()),
		count("max_tasks", cfg.Eval.MaxTasks != nil, cfg.EvalMaxTasks()),
		count("runs_per_task", cfg.Eval.RunsPerTask != nil, cfg.EvalRunsPerTask()),
	}
	if t, ok := cfg.EvalPassThreshold(); ok {
		parts = append(parts, fmt.Sprintf("pass_threshold %v", t))
	} else {
		parts = append(parts, "pass_threshold not set")
	}
	if n, ok := cfg.EvalCapTurnsPerTaskRun(); ok {
		parts = append(parts, fmt.Sprintf("cap turns_per_task_run %d", n))
	} else {
		parts = append(parts, "cap turns_per_task_run not set")
	}
	if n, ok := cfg.EvalCapTaskRunsPerEvalRun(); ok {
		parts = append(parts, fmt.Sprintf("cap task_runs_per_eval_run %d", n))
	} else {
		parts = append(parts, "cap task_runs_per_eval_run not set")
	}
	r.Lines = append(r.Lines, "eval: "+strings.Join(parts, ", "))
}
