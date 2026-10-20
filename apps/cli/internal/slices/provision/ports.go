package provision

// BinarySource reads the local gbai binary that will be installed on the server.
type BinarySource interface {
	ReadBinary(path string) ([]byte, error)
}
