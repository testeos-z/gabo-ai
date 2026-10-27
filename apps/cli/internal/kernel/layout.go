package kernel

// Fixed install locations of gbai on the server, used by review request
// until T043 retires it. The config loader never reads ServerConfigPath.
const (
	ServerBinaryPath = "/usr/local/bin/gbai"
	ServerConfigPath = "/etc/gbai/gbai.yaml"
)
