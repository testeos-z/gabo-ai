package secrets

// SecretReader reads a secret value from the operator without echoing it.
// Implementations must never take the value from argv or the environment.
type SecretReader interface {
	ReadSecret(prompt string) ([]byte, error)
}
