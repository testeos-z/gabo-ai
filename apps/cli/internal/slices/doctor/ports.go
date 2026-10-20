package doctor

// Local finds executables on this machine.
type Local interface {
	LookPath(name string) (string, error)
}
