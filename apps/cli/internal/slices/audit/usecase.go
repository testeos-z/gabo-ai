// Package audit cross-checks GitHub: every PR merged into the integration
// branch must have been merged by the bot, never by the coder, carry a
// successful reviewer/review status set by the bot on its head commit, and
// hold a clean verdict comment by the bot whose marker signs that head.
package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/testeos-z/gbai-team/apps/cli/internal/kernel/verdict"
)

// ReviewContext is the status context carrying the reviewer verdict.
const ReviewContext = "reviewer/review"

// ErrViolations is returned by the command when the report holds violations.
var ErrViolations = errors.New("audit found violations")

// Settings name the integration branch and the identities. An empty Coder
// skips the coder-merge rule.
type Settings struct {
	Integration string
	Bot         string
	Coder       string
}

// Violation is one broken rule on one PR.
type Violation struct {
	PR     int    `json:"pr"`
	Reason string `json:"reason"`
}

// Report is the audit outcome.
type Report struct {
	Since      time.Time   `json:"since"`
	Checked    int         `json:"checked"`
	Violations []Violation `json:"violations"`
}

// UseCase runs the audit.
type UseCase struct {
	gh GitHub
	s  Settings
}

// NewUseCase wires the use case.
func NewUseCase(gh GitHub, s Settings) *UseCase { return &UseCase{gh: gh, s: s} }

// Audit checks every PR merged into the integration branch since the given time.
func (u *UseCase) Audit(ctx context.Context, since time.Time) (Report, error) {
	prs, err := u.gh.MergedPullRequests(ctx, u.s.Integration, since)
	if err != nil {
		return Report{}, fmt.Errorf("list merged pull requests: %w", err)
	}
	rep := Report{Since: since, Checked: len(prs), Violations: []Violation{}}
	for _, pr := range prs {
		reasons := append(u.check(ctx, pr), u.checkVerdict(ctx, pr)...)
		for _, reason := range reasons {
			rep.Violations = append(rep.Violations, Violation{PR: pr.Number, Reason: reason})
		}
	}
	return rep, nil
}

func (u *UseCase) check(ctx context.Context, pr MergedPR) []string {
	var reasons []string
	switch {
	case u.s.Coder != "" && sameLogin(pr.MergedBy, u.s.Coder):
		reasons = append(reasons, "merged by the coder identity "+pr.MergedBy)
	case !sameLogin(pr.MergedBy, u.s.Bot):
		reasons = append(reasons, fmt.Sprintf("merged by %s, not the bot %s", orNobody(pr.MergedBy), u.s.Bot))
	}
	statuses, err := u.gh.Statuses(ctx, pr.HeadSHA)
	if err != nil {
		return append(reasons, fmt.Sprintf("could not read statuses of head %s: %v", pr.HeadSHA, err))
	}
	for _, st := range statuses {
		if st.Context != ReviewContext {
			continue
		}
		if st.State != "success" {
			reasons = append(reasons, fmt.Sprintf("latest %s on head %s is %s", ReviewContext, pr.HeadSHA, st.State))
		}
		if !sameLogin(st.Creator, u.s.Bot) {
			reasons = append(reasons, fmt.Sprintf("%s on head %s was set by %s, not the bot", ReviewContext, pr.HeadSHA, orNobody(st.Creator)))
		}
		return reasons
	}
	return append(reasons, fmt.Sprintf("no %s status on head %s", ReviewContext, pr.HeadSHA))
}

// checkVerdict requires the newest marker comment posted by the bot to sign
// the merged head and open with the clean verdict line.
func (u *UseCase) checkVerdict(ctx context.Context, pr MergedPR) []string {
	comments, err := u.gh.Comments(ctx, pr.Number)
	if err != nil {
		return []string{fmt.Sprintf("could not read comments of PR %d: %v", pr.Number, err)}
	}
	// GitHub logins are case-insensitive and apps post as "<name>[bot]".
	bot := u.s.Bot
	for i, c := range comments {
		if sameLogin(c.Author, bot) {
			comments[i].Author = bot
		}
	}
	c, signed, ok := verdict.NewestSigned(comments, bot)
	switch {
	case !ok:
		return []string{fmt.Sprintf("no reviewer verdict comment by %s on PR %d", orNobody(bot), pr.Number)}
	case signed != pr.HeadSHA:
		return []string{fmt.Sprintf("newest reviewer verdict comment signs %s, not head %s", signed, pr.HeadSHA)}
	case !verdict.OpensClean(c.Body):
		return []string{fmt.Sprintf("newest reviewer verdict comment on head %s does not start with %q", pr.HeadSHA, verdict.Clean)}
	}
	return nil
}

func sameLogin(a, b string) bool {
	return a != "" && strings.EqualFold(strings.TrimSuffix(a, "[bot]"), strings.TrimSuffix(b, "[bot]"))
}

func orNobody(s string) string {
	if s == "" {
		return "an unknown user"
	}
	return s
}
