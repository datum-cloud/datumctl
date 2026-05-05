//go:build !windows

package updatecheck

import "syscall"

func reExec(binPath string, argv []string, env []string) error {
	return syscall.Exec(binPath, argv, env)
}
