// Package adapters holds the driven adapters of the review slice. Every
// external CLI is invoked through kernel.Runner with argv only: PR data never
// reaches a shell.
package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
)

// run executes a command and turns a non-zero exit into an error.
func run(ctx context.Context, r kernel.Runner, c kernel.Command) (kernel.Result, error) {
	res, err := r.Run(ctx, c)
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s %s exited %d: %s", c.Name, firstArg(c.Args), res.ExitCode, stderrTail(res.Stderr))
	}
	return res, nil
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func stderrTail(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "no stderr"
	}
	if lines := strings.Split(s, "\n"); len(lines) > 10 {
		s = strings.Join(lines[len(lines)-10:], "\n")
	}
	return s
}
