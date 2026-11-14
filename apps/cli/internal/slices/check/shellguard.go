package check

// ShellViolation is one shell-guard rule a shell command breaks (intent 0028
// S4): the rule in words and the route a person or the coder takes instead.
type ShellViolation struct {
	Rule  string
	Route string
}

// ShellViolations returns the shell-guard violations of one command line,
// in command order. coder selects the coder session rules. Stub: RED.
func ShellViolations(command string, coder bool) []ShellViolation {
	return nil
}
