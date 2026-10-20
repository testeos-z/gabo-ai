package adapters

import (
	"bytes"
	"context"
	"fmt"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// CIClient implements review.Checks and review.Engine on the reviewer side by
// calling the root-owned review-ci wrapper as reviewer-ci through sudo.
type CIClient struct {
	runner     kernel.Runner
	user       string
	runtime    string
	credential func(runtime string) ([]byte, error)
}

var (
	_ review.Checks = (*CIClient)(nil)
	_ review.Engine = (*CIClient)(nil)
)

// NewCIClient returns the client. credential reads the model credential
// owned by the reviewer; it is only ever sent over stdin.
func NewCIClient(r kernel.Runner, user, runtime string, credential func(string) ([]byte, error)) *CIClient {
	return &CIClient{runner: r, user: user, runtime: runtime, credential: credential}
}

func (c *CIClient) command(args ...string) kernel.Command {
	return kernel.Command{Name: "sudo", Args: append([]string{"-n", "-u", c.user, kernel.ReviewCIPath}, args...)}
}

// Prepare streams the merged tree to a fresh reviewer-ci workspace.
func (c *CIClient) Prepare(ctx context.Context, ws string, tree []byte) error {
	cmd := c.command("prepare", "--dir", ws)
	cmd.Stdin = bytes.NewReader(tree)
	_, err := run(ctx, c.runner, cmd)
	return err
}

// Run executes one check; a non-zero exit is a failed check, not an error.
func (c *CIClient) Run(ctx context.Context, ws string, argv []string) (review.CheckOutcome, error) {
	res, err := c.runner.Run(ctx, c.command(append([]string{"run", "--dir", ws, "--"}, argv...)...))
	if err != nil {
		return review.CheckOutcome{}, err
	}
	return review.CheckOutcome{Passed: res.ExitCode == 0, Output: append(res.Stdout, res.Stderr...)}, nil
}

// Cleanup removes the reviewer-ci workspace.
func (c *CIClient) Cleanup(ctx context.Context, ws string) error {
	_, err := run(ctx, c.runner, c.command("cleanup", "--dir", ws))
	return err
}

// Review runs the engine as reviewer-ci and parses its findings.
func (c *CIClient) Review(ctx context.Context, ws, prompt string) ([]review.Finding, error) {
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
	cmd := c.command("engine", "--dir", ws, "--runtime", c.runtime)
	cmd.Stdin = bytes.NewReader(stdin)
	res, err := c.runner.Run(ctx, cmd)
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("%s engine exited %d", c.runtime, res.ExitCode)
	}
	return ParseFindings(c.runtime, res.Stdout)
}
