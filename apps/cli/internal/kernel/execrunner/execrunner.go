// Package execrunner implements kernel.Runner with os/exec.
package execrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
)

// Runner runs local processes.
type Runner struct{}

var _ kernel.Runner = Runner{}

// New returns an os/exec backed runner.
func New() Runner { return Runner{} }

// Run executes cmd and waits for it. A non-zero exit is not an error.
func (Runner) Run(ctx context.Context, cmd kernel.Command) (kernel.Result, error) {
	c := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	switch {
	case cmd.CleanEnv:
		c.Env = append([]string{}, cmd.Env...)
	case len(cmd.Env) > 0:
		c.Env = append(os.Environ(), cmd.Env...)
	}
	c.Dir = cmd.Dir
	c.Stdin = cmd.Stdin
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr

	err := c.Run()
	res := kernel.Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.Exited() {
		res.ExitCode = exitErr.ExitCode()
		return res, nil
	}
	if err != nil {
		return res, fmt.Errorf("run %s: %w", cmd.Name, err)
	}
	return res, nil
}
