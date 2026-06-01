package plugindispatch

import (
	kubecmd "k8s.io/kubectl/pkg/cmd"
)

// execPlatform delegates plugin execution to kubectl's DefaultPluginHandler,
// which is the shared implementation used by kubectl itself.
//
// On Unix/macOS the handler calls syscall.Exec, which replaces the current
// process image entirely — this function never returns on success.
//
// On Windows syscall.Exec is unavailable, so the handler starts the plugin as
// a child process and returns its exit error when the child exits non-zero.
// The dispatch layer (Exec in dispatch.go) is responsible for propagating that
// exit code via osExit so that datumctl's own exit status matches the plugin's.
//
// This single untagged file replaces the former exec_unix.go / exec_windows.go
// split; kubectl's handler already branches on runtime.GOOS internally and
// compiles correctly on all supported platforms.
var execPlatform = func(binaryPath string, args []string, env []string) error {
	handler := kubecmd.NewDefaultPluginHandler([]string{"datumctl"})
	return handler.Execute(binaryPath, args, env)
}
