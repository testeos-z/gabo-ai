// Package adapters holds the driven adapters of the audit slice.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/audit"
)

// GitHub implements audit.GitHub with read-only gh calls.
type GitHub struct {
	runner kernel.Runner
	repo   string
}

var _ audit.GitHub = (*GitHub)(nil)

// NewGitHub returns a gh-backed adapter for repo (owner/name).
func NewGitHub(r kernel.Runner, repo string) *GitHub { return &GitHub{runner: r, repo: repo} }

func (g *GitHub) ghJSON(ctx context.Context, v any, args ...string) error {
	res, err := kernel.RunChecked(ctx, g.runner, kernel.Command{Name: "gh", Args: args})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(res.Stdout, v); err != nil {
		return fmt.Errorf("decode gh %s: %w", args[0], err)
	}
	return nil
}

// MergedPullRequests lists PRs merged into base on or after since.
func (g *GitHub) MergedPullRequests(ctx context.Context, base string, since time.Time) ([]audit.MergedPR, error) {
	var rows []struct {
		Number   int `json:"number"`
		MergedBy *struct {
			Login string `json:"login"`
		} `json:"mergedBy"`
		HeadRefOid string    `json:"headRefOid"`
		MergedAt   time.Time `json:"mergedAt"`
	}
	if err := g.ghJSON(ctx, &rows, "pr", "list", "--repo", g.repo, "--base", base, "--state", "merged",
		"--search", "merged:>="+since.Format("2006-01-02"),
		"--json", "number,mergedBy,headRefOid,mergedAt", "--limit", "1000"); err != nil {
		return nil, err
	}
	out := make([]audit.MergedPR, 0, len(rows))
	for _, r := range rows {
		pr := audit.MergedPR{Number: r.Number, HeadSHA: r.HeadRefOid, MergedAt: r.MergedAt}
		if r.MergedBy != nil {
			pr.MergedBy = r.MergedBy.Login
		}
		out = append(out, pr)
	}
	return out, nil
}

// Statuses lists the statuses of sha, newest first as GitHub returns them.
func (g *GitHub) Statuses(ctx context.Context, sha string) ([]audit.CommitStatus, error) {
	var rows []struct {
		Context string `json:"context"`
		State   string `json:"state"`
		Creator *struct {
			Login string `json:"login"`
		} `json:"creator"`
	}
	if err := g.ghJSON(ctx, &rows, "api", "repos/"+g.repo+"/commits/"+sha+"/statuses?per_page=100"); err != nil {
		return nil, err
	}
	out := make([]audit.CommitStatus, 0, len(rows))
	for _, r := range rows {
		st := audit.CommitStatus{Context: r.Context, State: r.State}
		if r.Creator != nil {
			st.Creator = r.Creator.Login
		}
		out = append(out, st)
	}
	return out, nil
}
