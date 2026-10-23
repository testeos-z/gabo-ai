package audit

import (
	"context"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/verdict"
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

// GitHub reads merge, status and comment history.
type GitHub interface {
	MergedPullRequests(ctx context.Context, base string, since time.Time) ([]MergedPR, error)
	// Statuses lists the statuses of sha, newest first.
	Statuses(ctx context.Context, sha string) ([]CommitStatus, error)
	// Comments lists the PR's issue comments with their authors, oldest first.
	Comments(ctx context.Context, number int) ([]verdict.Comment, error)
}
