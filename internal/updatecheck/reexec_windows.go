//go:build windows

package updatecheck

import (
	"errors"
	"os"
	"os/exec"
)

func reExec(binPath string, argv []string, env []string) error {
	args := []string{}
	if len(argv) > 1 {
		args = argv[1:]
	}
	c := exec.Command(binPath, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Env = env
	err := c.Run()
	if err == nil {
		os.Exit(0)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	return err
}
