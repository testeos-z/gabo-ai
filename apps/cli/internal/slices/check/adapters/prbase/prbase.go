// Package prbase is the driven adapter of the check slice's pull-request port
// (intent 0028 D4): it reads the base branch of a pull request through the gh
// CLI, on the local host.
package prbase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/kernel"
)

// readTimeout bounds one host read, so a hung host blocks the merge instead of
// the hook waiting forever (the call then counts as unreadable).
const readTimeout = 30 * time.Second

// GH reads pull requests with `gh pr view` in Dir (the project directory, whose
// git remote names the repository).
type GH struct {
	Runner kernel.Runner
	Dir    string
}

// PRBase returns the base branch of the pull request ref in repo: ref is a
// number, a URL or a branch ("" is the current branch's pull request) and repo
// the -R value ("" is the current repository). A failed command, an empty
// answer or a value that gh would read as an option is an error.
func (g GH) PRBase(repo, ref string) (string, error) {
	if strings.HasPrefix(repo, "-") || strings.HasPrefix(ref, "-") {
		return "", errors.New("pull request reference starts with -")
	}
	args := []string{"pr", "view"}
	if ref != "" {
		args = append(args, ref)
	}
	args = append(args, "--json", "baseRefName", "--jq", ".baseRefName")
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()
	res, err := g.Runner.Run(ctx, kernel.Command{Name: "gh", Args: args, Dir: g.Dir})
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("gh pr view exited %d: %s", res.ExitCode, strings.TrimSpace(string(res.Stderr)))
	}
	base := strings.TrimSpace(string(res.Stdout))
	if base == "" {
		return "", errors.New("gh pr view printed no base branch")
	}
	return base, nil
}
