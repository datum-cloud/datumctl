package cmd

import (
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"k8s.io/klog/v2"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// Once routed, controller-runtime logs reach klog's sink (the console's log
// file) rather than staying unset, which is what makes controller-runtime
// print its "SetLogger was never called" stack to stderr.
func TestRouteControllerRuntimeLogsReachesKlog(t *testing.T) {
	var got strings.Builder
	klog.SetLogger(funcr.New(func(prefix, args string) {
		got.WriteString(prefix + " " + args + "\n")
	}, funcr.Options{}))
	t.Cleanup(klog.ClearLogger)

	RouteControllerRuntimeLogs()
	ctrllog.Log.WithName("probe").Info("routed through klog")

	if !strings.Contains(got.String(), "routed through klog") {
		t.Fatalf("controller-runtime log did not reach klog's sink; got %q", got.String())
	}
}
