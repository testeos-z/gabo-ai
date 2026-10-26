package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// ciCore is the checks side the reviewer drives; *review.CIUseCase
// implements it.
type ciCore interface {
	Prepare(name string, tree io.Reader) error
	Run(ctx context.Context, name string, argv []string) (kernel.Result, error)
	Engine(ctx context.Context, name, runtime string, stdin io.Reader) (kernel.Result, error)
	Cleanup(name string) error
}

var _ ciCore = (*review.CIUseCase)(nil)

// LocalCI implements review.Checks and review.Engine in single-user mode:
// the checks side runs in the reviewer's own process, as the same OS user,
// with no sudo. Its isolation is the environment the checks use case builds
// from scratch (no GitHub credentials, empty gh config dir).
type LocalCI struct {
	core       ciCore
	runtime    string
	credential func(runtime string) ([]byte, error)
}

var (
	_ review.Checks = (*LocalCI)(nil)
	_ review.Engine = (*LocalCI)(nil)
)

// NewLocalCI returns the adapter. credential reads the model credential; it
// only ever reaches the engine over its stdin.
func NewLocalCI(core ciCore, runtime string, credential func(string) ([]byte, error)) *LocalCI {
	return &LocalCI{core: core, runtime: runtime, credential: credential}
}

// Prepare replaces the workspace with the merged tree.
func (c *LocalCI) Prepare(_ context.Context, ws string, tree []byte) error {
	return c.core.Prepare(ws, bytes.NewReader(tree))
}

// Run executes one check; a non-zero exit is a failed check, not an error.
func (c *LocalCI) Run(ctx context.Context, ws string, argv []string) (review.CheckOutcome, error) {
	res, err := c.core.Run(ctx, ws, argv)
	if err != nil {
		return review.CheckOutcome{}, err
	}
	return review.CheckOutcome{Passed: res.ExitCode == 0, Output: append(res.Stdout, res.Stderr...)}, nil
}

// Cleanup removes the workspace.
func (c *LocalCI) Cleanup(_ context.Context, ws string) error {
	return c.core.Cleanup(ws)
}

// Review runs the engine and parses its findings.
func (c *LocalCI) Review(ctx context.Context, ws, prompt string) ([]review.Finding, error) {
	cred, err := c.credential(c.runtime)
	if err != nil {
		return nil, fmt.Errorf("read %s credential: %w", c.runtime, err)
	}
	defer clear(cred)
	cred = bytes.TrimSpace(cred)
	if len(cred) == 0 || bytes.ContainsAny(cred, "\r\n") {
		return nil, fmt.Errorf("%s credential is empty or multi-line", c.runtime)
	}
	stdin := make([]byte, 0, len(cred)+1+len(prompt))
	stdin = append(append(append(stdin, cred...), '\n'), prompt...)
	defer clear(stdin)
	res, err := c.core.Engine(ctx, ws, c.runtime, bytes.NewReader(stdin))
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("%s engine exited %d", c.runtime, res.ExitCode)
	}
	return ParseFindings(c.runtime, res.Stdout)
}
