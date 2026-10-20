// Package adapters holds the driven adapters of the review slice. Every
// external CLI is invoked through kernel.Runner with argv only: PR data never
// reaches a shell.
package adapters

import (
	"context"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// run executes a command and turns a non-zero exit into an error.
func run(ctx context.Context, r kernel.Runner, c kernel.Command) (kernel.Result, error) {
	return kernel.RunChecked(ctx, r, c)
}

func stderrTail(b []byte) string { return kernel.StderrTail(b) }
