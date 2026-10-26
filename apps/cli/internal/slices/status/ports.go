package status

import "context"

// PullRequest is an open pull request to the integration branch.
type PullRequest struct {
	Number  int
	Title   string
	HeadSHA string
	Labels  []string
}

// Issue is an open issue.
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// GitHub reads repository state.
type GitHub interface {
	OpenPullRequests(ctx context.Context, base string) ([]PullRequest, error)
	// LatestStatus returns the newest state of the status context on sha, or "" when absent.
	LatestStatus(ctx context.Context, sha, context string) (string, error)
	// LastVerdict returns the body of author's newest comment on the PR that
	// carries the reviewer's verdict marker, or "" when absent. In single-user
	// mode the coder comments with the same login, so the marker is what
	// identifies a verdict.
	LastVerdict(ctx context.Context, number int, author string) (string, error)
	NeedsUserIssues(ctx context.Context) ([]Issue, error)
}
