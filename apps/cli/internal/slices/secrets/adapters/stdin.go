// Package adapters holds driven adapters specific to the secrets slice.
package adapters

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const maxSecretBytes = 64 << 10

// StdinReader reads a secret from a terminal without echo, or from piped stdin.
type StdinReader struct {
	in     *os.File
	prompt io.Writer
}

// NewStdinReader reads from in and writes the prompt (terminal only) to prompt.
func NewStdinReader(in *os.File, prompt io.Writer) StdinReader {
	return StdinReader{in: in, prompt: prompt}
}

// ReadSecret returns the value with one trailing newline removed.
func (r StdinReader) ReadSecret(prompt string) ([]byte, error) {
	fd := int(r.in.Fd())
	if term.IsTerminal(fd) {
		if _, err := fmt.Fprint(r.prompt, prompt); err != nil {
			return nil, fmt.Errorf("write prompt: %w", err)
		}
		b, err := term.ReadPassword(fd)
		_, _ = fmt.Fprintln(r.prompt)
		if err != nil {
			return nil, fmt.Errorf("read from terminal: %w", err)
		}
		return b, nil
	}
	b, err := io.ReadAll(io.LimitReader(r.in, maxSecretBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read from stdin: %w", err)
	}
	if len(b) > maxSecretBytes {
		clear(b)
		return nil, errors.New("secret on stdin exceeds 64 KiB")
	}
	b = bytes.TrimSuffix(b, []byte("\n"))
	return bytes.TrimSuffix(b, []byte("\r")), nil
}
