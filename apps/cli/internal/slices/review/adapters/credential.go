package adapters

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/testeos-z/gbai-team/apps/cli/internal/slices/review"
)

const maxCredentialBytes = 64 << 10

// credentialFiles maps a runtime to its credential file name in the secrets dir.
var credentialFiles = map[string]string{
	review.RuntimeClaude: "claude-oauth-token",
	review.RuntimePi:     "pi-api-key",
}

// CredentialReader returns a function reading the model credential of a
// runtime from dir. The file must be a regular file.
func CredentialReader(dir string) func(runtime string) ([]byte, error) {
	return func(runtime string) ([]byte, error) {
		base, ok := credentialFiles[runtime]
		if !ok {
			return nil, fmt.Errorf("no credential file for runtime %q", runtime)
		}
		f, err := os.Open(filepath.Join(dir, base))
		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", base)
		}
		b, err := io.ReadAll(io.LimitReader(f, maxCredentialBytes+1))
		if err != nil {
			return nil, err
		}
		if len(b) > maxCredentialBytes {
			clear(b)
			return nil, fmt.Errorf("%s is too large", base)
		}
		return b, nil
	}
}
