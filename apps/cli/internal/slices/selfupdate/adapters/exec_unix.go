//go:build unix

package adapters

import "syscall"

func execProcess(path string, args, env []string) error {
	return syscall.Exec(path, args, env)
}
