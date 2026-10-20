// Package adapters holds the driven adapters of the repo slice.
package adapters

import (
	"bytes"
	"context"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/repo"
)

// API implements repo.API with `gh api`, authenticated by the owner's gh
// session. Request bodies travel on stdin, never in argv.
type API struct {
	runner kernel.Runner
}

var _ repo.API = (*API)(nil)

// NewAPI returns a gh-backed API client.
func NewAPI(r kernel.Runner) *API { return &API{runner: r} }

// Call performs one REST request and returns the response body.
func (a *API) Call(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	c := kernel.Command{Name: "gh", Args: []string{"api", "--method", method, path}}
	if body != nil {
		c.Args = append(c.Args, "--input", "-")
		c.Stdin = bytes.NewReader(body)
	}
	res, err := kernel.RunChecked(ctx, a.runner, c)
	if err != nil {
		return nil, err
	}
	return res.Stdout, nil
}
