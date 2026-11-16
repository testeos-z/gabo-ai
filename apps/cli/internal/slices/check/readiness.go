package check

import (
	"io/fs"
	"strings"

	"github.com/testeos-z/gabo-ai/apps/cli/internal/config"
)

// checkReviewReadiness reports the blocking setup gap of issue #904 that fits
// the check's read-only role: the dependency install step. It only reads.
// The review policy (REVIEW.md on the integration branch) and the
// integration-branch existence stay in gabo-setup: the check runs on an
// arbitrary worktree with no git or network port, so it cannot verify what
// the integration branch holds.
func (u *UseCase) checkReviewReadiness(r *Report, cfg config.Config) {
	locks := []struct{ file, use, install string }{
		{"bun.lock", "bun ", "bun install --frozen-lockfile"},
		{"package-lock.json", "npm ", "npm ci"},
		{"pnpm-lock.yaml", "pnpm ", "pnpm install --frozen-lockfile"},
		{"yarn.lock", "yarn ", "yarn install --frozen-lockfile"},
	}
	for _, l := range locks {
		if _, err := fs.Stat(u.project, l.file); err != nil {
			continue
		}
		usesManager := false
		hasInstall := false
		for _, c := range cfg.Review.Checks {
			lc := strings.ToLower(c)
			if strings.Contains(lc, l.use) {
				usesManager = true
			}
			if strings.Contains(lc, "install") || strings.Contains(lc, "npm ci") {
				hasInstall = true
			}
		}
		if usesManager && !hasInstall {
			r.fail("✗ review.checks has no dependency install step: checks run in a fresh worktree with nothing installed",
				"next: list "+l.install+" as the first review.checks entry, then rerun gabo check")
			return
		}
	}
}
