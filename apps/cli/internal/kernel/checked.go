package kernel

import (
	"context"
	"fmt"
	"strings"
)

// RunChecked runs c and turns a non-zero exit code into an error carrying
// the tail of stderr.
func RunChecked(ctx context.Context, r Runner, c Command) (Result, error) {
	res, err := r.Run(ctx, c)
	if err != nil {
		return res, err
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s %s exited %d: %s", c.Name, firstArg(c.Args), res.ExitCode, StderrTail(res.Stderr))
	}
	return res, nil
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// StderrTail returns at most the last 10 lines of b, trimmed.
func StderrTail(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "no stderr"
	}
	if lines := strings.Split(s, "\n"); len(lines) > 10 {
		s = strings.Join(lines[len(lines)-10:], "\n")
	}
	return s
}
