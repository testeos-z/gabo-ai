// Package kerneltest provides hand-written in-memory fakes of the kernel ports.
package kerneltest

import (
	"context"
	"io"
	"io/fs"

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

// FS is a fake kernel.FileSystem over an in-memory map of path to content.
type FS struct {
	Files    map[string]string
	WriteErr error
	Writes   []string
}

var _ kernel.FileSystem = (*FS)(nil)

// ReadFile returns the content or an error matching fs.ErrNotExist.
func (f *FS) ReadFile(name string) ([]byte, error) {
	s, ok := f.Files[name]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return []byte(s), nil
}

// WriteFile records the write and stores the content.
func (f *FS) WriteFile(name string, data []byte) error {
	f.Writes = append(f.Writes, name)
	if f.WriteErr != nil {
		return f.WriteErr
	}
	if f.Files == nil {
		f.Files = map[string]string{}
	}
	f.Files[name] = string(data)
	return nil
}
