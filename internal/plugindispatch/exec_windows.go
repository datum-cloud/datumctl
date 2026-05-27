//go:build windows

package plugindispatch

import (
	"errors"
	"os"
	"os/exec"
)

// execPlatform runs binaryPath on Windows using exec.Command and os.Exit.
// Windows does not support syscall.Exec.
func execPlatform(binaryPath string, args []string, env []string) error {
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env

	runErr := cmd.Run()
	if runErr == nil {
		os.Exit(0)
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return runErr
}
