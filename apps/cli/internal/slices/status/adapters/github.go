// Package adapters holds the driven adapters of the status slice.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/status"
)

// GitHub implements status.GitHub with the gh CLI (read-only calls).
type GitHub struct {
	runner kernel.Runner
	repo   string
}

var _ status.GitHub = (*GitHub)(nil)

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

// OpenPullRequests lists open PRs to base.
func (g *GitHub) OpenPullRequests(ctx context.Context, base string) ([]status.PullRequest, error) {
	var rows []struct {
		Number     int    `json:"number"`
		Title      string `json:"title"`
		HeadRefOid string `json:"headRefOid"`
		Labels     []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := g.ghJSON(ctx, &rows, "pr", "list", "--repo", g.repo, "--base", base, "--state", "open",
		"--json", "number,title,headRefOid,labels", "--limit", "100"); err != nil {
		return nil, err
	}
	out := make([]status.PullRequest, 0, len(rows))
	for _, r := range rows {
		pr := status.PullRequest{Number: r.Number, Title: r.Title, HeadSHA: r.HeadRefOid}
		for _, l := range r.Labels {
			pr.Labels = append(pr.Labels, l.Name)
		}
		out = append(out, pr)
	}
	return out, nil
}

// LatestStatus returns the newest state of context on sha; GitHub lists statuses newest first.
func (g *GitHub) LatestStatus(ctx context.Context, sha, context string) (string, error) {
	var rows []struct {
		Context string `json:"context"`
		State   string `json:"state"`
	}
	if err := g.ghJSON(ctx, &rows, "api", "repos/"+g.repo+"/commits/"+sha+"/statuses?per_page=100"); err != nil {
		return "", err
	}
	for _, r := range rows {
		if r.Context == context {
			return r.State, nil
		}
	}
	return "", nil
}

// LastComment returns the body of author's newest PR comment.
func (g *GitHub) LastComment(ctx context.Context, number int, author string) (string, error) {
	var doc struct {
		Comments []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			Body string `json:"body"`
		} `json:"comments"`
	}
	if err := g.ghJSON(ctx, &doc, "pr", "view", strconv.Itoa(number), "--repo", g.repo, "--json", "comments"); err != nil {
		return "", err
	}
	for i := len(doc.Comments) - 1; i >= 0; i-- {
		if sameLogin(doc.Comments[i].Author.Login, author) {
			return doc.Comments[i].Body, nil
		}
	}
	return "", nil
}

// NeedsUserIssues lists open issues labeled needs-user.
func (g *GitHub) NeedsUserIssues(ctx context.Context) ([]status.Issue, error) {
	var issues []status.Issue
	if err := g.ghJSON(ctx, &issues, "issue", "list", "--repo", g.repo, "--label", "needs-user", "--state", "open",
		"--json", "number,title", "--limit", "100"); err != nil {
		return nil, err
	}
	return issues, nil
}

func sameLogin(a, b string) bool {
	return strings.EqualFold(strings.TrimSuffix(a, "[bot]"), strings.TrimSuffix(b, "[bot]"))
}
