package audit

import (
	"context"
	"time"
)

// MergedPR is a pull request merged into the integration branch.
type MergedPR struct {
	Number   int
	MergedBy string
	HeadSHA  string // head commit at merge time
	MergedAt time.Time
}

// CommitStatus is one status posted on a commit.
type CommitStatus struct {
	Context string
	State   string
	Creator string
}

// GitHub reads merge and status history.
type GitHub interface {
	MergedPullRequests(ctx context.Context, base string, since time.Time) ([]MergedPR, error)
	// Statuses lists the statuses of sha, newest first.
	Statuses(ctx context.Context, sha string) ([]CommitStatus, error)
}
