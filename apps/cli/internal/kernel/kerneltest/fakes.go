// Package kerneltest provides hand-written in-memory fakes of the kernel ports.
package kerneltest

import (
	"context"
	"io"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// RunnerCall records one Runner invocation with its stdin fully read.
type RunnerCall struct {
	Command kernel.Command
	Stdin   string
}

// Runner is a fake kernel.Runner returning Result or Err for every call.
type Runner struct {
	Result kernel.Result
	Err    error
	Calls  []RunnerCall
}

var _ kernel.Runner = (*Runner)(nil)

// Run records the call and returns the configured outcome.
func (r *Runner) Run(_ context.Context, cmd kernel.Command) (kernel.Result, error) {
	call := RunnerCall{Command: cmd}
	if cmd.Stdin != nil {
		b, err := io.ReadAll(cmd.Stdin)
		if err != nil {
			return kernel.Result{}, err
		}
		call.Stdin = string(b)
	}
	r.Calls = append(r.Calls, call)
	return r.Result, r.Err
}

// ShellCall records one RemoteShell invocation with its stdin fully read.
type ShellCall struct {
	Script kernel.Script
	Stdin  string
}

// RemoteShell is a fake kernel.RemoteShell. Results are returned in order;
// once exhausted, the zero Result (exit code 0) is returned.
type RemoteShell struct {
	Results []kernel.Result
	Err     error
	Calls   []ShellCall
}

var _ kernel.RemoteShell = (*RemoteShell)(nil)

// Exec records the call and returns the next configured outcome.
func (s *RemoteShell) Exec(_ context.Context, sc kernel.Script) (kernel.Result, error) {
	call := ShellCall{Script: sc}
	if sc.Stdin != nil {
		b, err := io.ReadAll(sc.Stdin)
		if err != nil {
			return kernel.Result{}, err
		}
		call.Stdin = string(b)
	}
	s.Calls = append(s.Calls, call)
	if s.Err != nil {
		return kernel.Result{}, s.Err
	}
	var res kernel.Result
	if len(s.Results) > 0 {
		res, s.Results = s.Results[0], s.Results[1:]
	}
	return res, nil
}
