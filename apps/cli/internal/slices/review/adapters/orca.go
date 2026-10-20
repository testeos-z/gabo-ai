package adapters

import (
	"context"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// Orca implements review.Orca with the orca CLI.
type Orca struct {
	runner kernel.Runner
}

var _ review.Orca = (*Orca)(nil)

// NewOrca returns an orca-backed adapter.
func NewOrca(r kernel.Runner) *Orca { return &Orca{runner: r} }

// CreateTerminal opens a detached terminal running command and returns orca's JSON output.
func (o *Orca) CreateTerminal(ctx context.Context, title, command string) (string, error) {
	res, err := run(ctx, o.runner, kernel.Command{Name: "orca", Args: []string{"terminal", "create", "--title", title, "--command", command, "--json"}})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(res.Stdout)), nil
}

// Send posts a status message to an orchestration mailbox.
func (o *Orca) Send(ctx context.Context, to, subject string) error {
	_, err := run(ctx, o.runner, kernel.Command{Name: "orca", Args: []string{"orchestration", "send", "--to", to, "--subject", subject, "--type", "status"}})
	return err
}
