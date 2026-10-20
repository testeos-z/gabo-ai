// Package kerneltest provides hand-written in-memory fakes of the kernel ports.
package kerneltest

import (
	"context"
	"io"
	"strconv"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// RunnerCall records one Runner invocation with its stdin fully read.
type RunnerCall struct {
	Command kernel.Command
	Stdin   string
}

// Runner is a fake kernel.Runner. When Handle is set it decides every
// outcome; otherwise Result or Err is returned for every call.
type Runner struct {
	Result kernel.Result
	Err    error
	Handle func(kernel.Command) (kernel.Result, error)
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
	if r.Handle != nil {
		return r.Handle(cmd)
	}
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

// Terminals is a fake kernel.Terminals over an in-memory terminal list.
// Errors, when set, fail the matching operation.
type Terminals struct {
	Live      []kernel.Terminal
	ListErr   error
	CreateErr error
	CloseErr  error
	Created   []TerminalCreate
	Closed    []string
	next      int
}

// TerminalCreate records one Create call.
type TerminalCreate struct {
	Title, Command string
}

var _ kernel.Terminals = (*Terminals)(nil)

// List returns the live terminals.
func (t *Terminals) List(context.Context) ([]kernel.Terminal, error) {
	if t.ListErr != nil {
		return nil, t.ListErr
	}
	return append([]kernel.Terminal(nil), t.Live...), nil
}

// Create records the call and adds a live terminal.
func (t *Terminals) Create(_ context.Context, title, command string) (string, error) {
	t.Created = append(t.Created, TerminalCreate{Title: title, Command: command})
	if t.CreateErr != nil {
		return "", t.CreateErr
	}
	t.next++
	h := "term_new" + strconv.Itoa(t.next)
	t.Live = append(t.Live, kernel.Terminal{Handle: h, Title: title})
	return `{"handle":"` + h + `"}`, nil
}

// Close records the call and removes the terminal.
func (t *Terminals) Close(_ context.Context, handle string) error {
	t.Closed = append(t.Closed, handle)
	if t.CloseErr != nil {
		return t.CloseErr
	}
	for i, l := range t.Live {
		if l.Handle == handle {
			t.Live = append(t.Live[:i], t.Live[i+1:]...)
			break
		}
	}
	return nil
}
