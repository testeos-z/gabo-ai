package kernel

// Fixed install locations of gbai on the server, shared by provision and review.
const (
	ServerBinaryPath = "/usr/local/bin/gbai"
	ServerConfigPath = "/etc/gbai/gbai.yaml"
	ReviewRunPath    = "/usr/local/libexec/gbai/review-run"
	ReviewCIPath     = "/usr/local/libexec/gbai/review-ci"
)
