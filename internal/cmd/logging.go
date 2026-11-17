package cmd

import (
	"k8s.io/klog/v2"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// RouteControllerRuntimeLogs sends controller-runtime's logs through klog, so
// they follow the same -v verbosity and the same sink as client-go's (the
// console sends klog to a file). Call it before any command runs.
//
// Without it, controller-runtime writes a "log.SetLogger(...) was never
// called" warning and a goroutine stack to stderr the first time one of its
// clients is created 30 seconds or more after startup. That happens when the
// console's context switcher is first opened late in a session, where the
// trace is drawn over the TUI and looks like a crash, and after a slow
// browser login.
func RouteControllerRuntimeLogs() {
	ctrllog.SetLogger(klog.NewKlogr())
}
