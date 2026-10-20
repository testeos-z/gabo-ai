package kernel

// Fixed install locations of gbai on the server, shared by provision and review.
const (
	ServerBinaryPath = "/usr/local/bin/gbai"
	ServerConfigPath = "/etc/gbai/gbai.yaml"
	ReviewRunPath    = "/usr/local/libexec/gbai/review-run"
	ReviewCIPath     = "/usr/local/libexec/gbai/review-ci"
)

// Fixed paths checked by doctor and created by provision.
const SudoersPath = "/etc/sudoers.d/gbai-reviewer"

// SessionTitles are the exact Orca terminal titles of the gbai sessions.
func SessionTitles() []string {
	return []string{"gbai-coder", "gbai-reviewer", "gbai-assistant"}
}
