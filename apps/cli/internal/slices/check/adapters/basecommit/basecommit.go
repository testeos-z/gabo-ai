// Package basecommit is the driven adapter of the check slice's base-commit
// port (intent 0028 D6): it reads, through git, whether a path is tracked at
// the merge base of HEAD and a ref.
package basecommit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
)

// readTimeout bounds one git read, so a hung read counts as unreadable instead
// of blocking the hook.
const readTimeout = 30 * time.Second

// Git reads the base commit with git in Dir (the project directory, whose git
// top level the paths are relative to).
type Git struct {
	Runner kernel.Runner
	Dir    string
}

// Tracked reports whether rel is tracked at the merge base of HEAD and ref: the
// commit the branch forked from when ref is its integration branch. A ref that
// does not resolve, a missing merge base or a failed read is an error; the
// caller then counts rel as not tracked. A ref or path that git would read as an
// option is refused before any command runs.
func (g Git) Tracked(ref, rel string) (bool, error) {
	if strings.HasPrefix(ref, "-") || strings.HasPrefix(rel, "-") {
		return false, errors.New("ref or path starts with -")
	}
	base, err := g.run("merge-base", "HEAD", ref)
	if err != nil {
		return false, err
	}
	commit := strings.TrimSpace(string(base))
	if commit == "" {
		return false, errors.New("git merge-base printed no commit")
	}
	tree, err := g.run("ls-tree", "--name-only", "-z", commit, "--", rel)
	if err != nil {
		return false, err
	}
	for _, entry := range strings.Split(string(tree), "\x00") {
		if entry == rel {
			return true, nil
		}
	}
	return false, nil
}

// run executes git with args in Dir and returns its stdout; a non-zero exit is
// an error that carries git's stderr.
func (g Git) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()
	res, err := g.Runner.Run(ctx, kernel.Command{Name: "git", Args: args, Dir: g.Dir})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("git %s exited %d: %s", args[0], res.ExitCode, strings.TrimSpace(string(res.Stderr)))
	}
	return res.Stdout, nil
}
