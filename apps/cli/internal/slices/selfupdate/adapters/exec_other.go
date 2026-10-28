//go:build !unix

package adapters

import "errors"

func execProcess(string, []string, []string) error {
	return errors.New("restarting a process is not supported on this platform")
}
