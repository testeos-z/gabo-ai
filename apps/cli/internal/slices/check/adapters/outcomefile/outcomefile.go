// Package outcomefile is the driven adapter of the check slice's outcome
// port (intent 0028 S16): it appends the records of a run to the file the
// cycle names, on the local disk.
package outcomefile

import "os"

// File appends to files on the local disk.
type File struct{}

// Append opens name for appending (created with mode 0600 when missing),
// writes line and closes the file. An open or write error is returned, so a
// record is never dropped silently.
func (File) Append(name string, line []byte) error {
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(line); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
