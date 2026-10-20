// Package adapters holds driven adapters specific to the provision slice.
package adapters

import (
	"fmt"
	"io"
	"os"
)

const maxBinaryBytes = 512 << 20

// FileBinary reads the binary from the local filesystem.
type FileBinary struct{}

// ReadBinary returns the content of the regular file at path.
func (FileBinary) ReadBinary(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxBinaryBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxBinaryBytes {
		return nil, fmt.Errorf("%s exceeds %d bytes", path, maxBinaryBytes)
	}
	return b, nil
}
