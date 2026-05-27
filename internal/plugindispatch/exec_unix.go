//go:build !windows

package plugindispatch

import (
	kubecmd "k8s.io/kubectl/pkg/cmd"
)

// execPlatform replaces the current process with binaryPath using syscall.Exec
// via kubectl's DefaultPluginHandler (which handles both Unix and Windows).
func execPlatform(binaryPath string, args []string, env []string) error {
	handler := kubecmd.NewDefaultPluginHandler([]string{"datumctl"})
	return handler.Execute(binaryPath, args, env)
}
