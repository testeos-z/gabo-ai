package kernel

// Fixed install locations of gbai on the server, used by review request and
// the system config layer until T043 and T006 retire them.
const (
	ServerBinaryPath = "/usr/local/bin/gbai"
	ServerConfigPath = "/etc/gbai/gbai.yaml"
)
