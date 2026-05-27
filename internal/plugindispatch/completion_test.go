package plugindispatch

import (
	"os"
	"testing"
)

// TestForwardCompletion_noop_whenNotComplete verifies that ForwardCompletion
// returns nil without doing anything when os.Args[1] is not "__complete".
func TestForwardCompletion_noop_whenNotComplete(t *testing.T) {
	t.Parallel()

	// ForwardCompletion reads os.Args but only acts when os.Args[1] == "__complete".
	// We save and restore os.Args to isolate the test.
	orig := os.Args
	os.Args = []string{"datumctl", "get", "resourcegroups"}
	t.Cleanup(func() { os.Args = orig })

	managedDir := t.TempDir()

	err := ForwardCompletion(managedDir, nil)
	if err != nil {
		t.Errorf("ForwardCompletion with non-completion args: want nil, got %v", err)
	}
}

// TestForwardCompletion_noop_whenBuiltin verifies that ForwardCompletion returns
// nil (does not exec or exit) when the third argument is a built-in command name
// — we test by passing a managed dir that contains no plugins, so FindPlugin
// will fail and Cobra is left to handle it.
func TestForwardCompletion_noop_whenBuiltin(t *testing.T) {
	t.Parallel()

	orig := os.Args
	// "__complete" + builtin name "get" — FindPlugin("get", emptyDir) will fail,
	// so ForwardCompletion returns nil without calling os.Exit.
	os.Args = []string{"datumctl", "__complete", "get", ""}
	t.Cleanup(func() { os.Args = orig })

	// Use an empty managed dir so no plugin binary named "datumctl-get" exists.
	managedDir := t.TempDir()

	err := ForwardCompletion(managedDir, nil)
	if err != nil {
		t.Errorf("ForwardCompletion for builtin name: want nil, got %v", err)
	}
}
