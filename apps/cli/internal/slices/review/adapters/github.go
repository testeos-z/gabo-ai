package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel"
	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

// GitHub implements review.GitHub with the gh CLI authenticated in the
// reviewer's HOME.
type GitHub struct {
	runner kernel.Runner
	repo   string
}

var _ review.GitHub = (*GitHub)(nil)

// NewGitHub returns a gh-backed adapter for repo (owner/name).
func NewGitHub(r kernel.Runner, repo string) *GitHub { return &GitHub{runner: r, repo: repo} }

func (g *GitHub) gh(ctx context.Context, stdin string, args ...string) (kernel.Result, error) {
	c := kernel.Command{Name: "gh", Args: args}
	if stdin != "" {
		c.Stdin = strings.NewReader(stdin)
	}
	return run(ctx, g.runner, c)
}

// PullRequest reads the PR state, head, base and labels.
func (g *GitHub) PullRequest(ctx context.Context, n int) (review.PullRequest, error) {
	res, err := g.gh(ctx, "", "pr", "view", strconv.Itoa(n), "--repo", g.repo, "--json", "state,headRefOid,baseRefName,labels")
	if err != nil {
		return review.PullRequest{}, err
	}
	var doc struct {
		State       string `json:"state"`
		HeadRefOid  string `json:"headRefOid"`
		BaseRefName string `json:"baseRefName"`
		Labels      []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(res.Stdout, &doc); err != nil {
		return review.PullRequest{}, fmt.Errorf("decode gh pr view: %w", err)
	}
	pr := review.PullRequest{Number: n, State: doc.State, HeadSHA: doc.HeadRefOid, BaseBranch: doc.BaseRefName}
	for _, l := range doc.Labels {
		pr.Labels = append(pr.Labels, l.Name)
	}
	return pr, nil
}

// SetStatus posts a commit status on sha.
func (g *GitHub) SetStatus(ctx context.Context, sha string, s review.Status) error {
	_, err := g.gh(ctx, "", "api", "--method", "POST", "repos/"+g.repo+"/statuses/"+sha,
		"-f", "state="+string(s.State), "-f", "context="+s.Context, "-f", "description="+s.Description)
	return err
}

// Comment posts body as one PR comment, streamed over stdin.
func (g *GitHub) Comment(ctx context.Context, n int, body string) error {
	_, err := g.gh(ctx, body, "pr", "comment", strconv.Itoa(n), "--repo", g.repo, "--body-file", "-")
	return err
}

// Comments lists the PR's issue comments with their authors, oldest first.
// Each page prints one JSON array, so the output is decoded as a stream.
func (g *GitHub) Comments(ctx context.Context, n int) ([]review.Comment, error) {
	res, err := g.gh(ctx, "", "api", "--paginate", "repos/"+g.repo+"/issues/"+strconv.Itoa(n)+"/comments",
		"--jq", "[.[] | {author: .user.login, body: .body}]")
	if err != nil {
		return nil, err
	}
	var comments []review.Comment
	dec := json.NewDecoder(bytes.NewReader(res.Stdout))
	for {
		var page []struct {
			Author string `json:"author"`
			Body   string `json:"body"`
		}
		if err := dec.Decode(&page); errors.Is(err, io.EOF) {
			return comments, nil
		} else if err != nil {
			return nil, fmt.Errorf("decode gh api comments: %w", err)
		}
		for _, c := range page {
			comments = append(comments, review.Comment{Author: c.Author, Body: c.Body})
		}
	}
}

// Login returns the login gh is authenticated as, the identity every
// review comment is posted with.
func (g *GitHub) Login(ctx context.Context) (string, error) {
	res, err := g.gh(ctx, "", "api", "user", "--jq", ".login")
	if err != nil {
		return "", err
	}
	login := strings.TrimSpace(string(res.Stdout))
	if login == "" || strings.ContainsAny(login, " \t\r\n") {
		return "", fmt.Errorf("gh api user returned login %q", login)
	}
	return login, nil
}

// Merge merges the PR with a merge commit, only if its head is still sha.
func (g *GitHub) Merge(ctx context.Context, n int, sha string) error {
	_, err := g.gh(ctx, "", "pr", "merge", strconv.Itoa(n), "--repo", g.repo, "--merge", "--match-head-commit", sha)
	return err
}
