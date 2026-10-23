package kernel

// Fixed install locations of gbai on the server, shared by provision and review.
const (
	ServerBinaryPath = "/usr/local/bin/gbai"
	ServerConfigPath = "/etc/gbai/gbai.yaml"
)

// Paths of the retired three-user reviewer model (reviewer and reviewer-ci
// OS users reached through sudo). Provision removes them; doctor checks they
// are gone.
const (
	LegacySudoersPath   = "/etc/sudoers.d/gbai-reviewer"
	LegacyLibexecDir    = "/usr/local/libexec/gbai"
	LegacyReviewRunPath = LegacyLibexecDir + "/review-run"
	LegacyReviewCIPath  = LegacyLibexecDir + "/review-ci"
)

// LegacyReviewerFiles lists the files of the retired three-user model.
func LegacyReviewerFiles() []string {
	return []string{LegacySudoersPath, LegacyReviewRunPath, LegacyReviewCIPath}
}

// SessionTitles are the exact Orca terminal titles of the gbai sessions.
func SessionTitles() []string {
	return []string{"gbai-coder", "gbai-reviewer", "gbai-assistant"}
}
