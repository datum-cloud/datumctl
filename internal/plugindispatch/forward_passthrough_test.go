package plugindispatch

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

// TestForwardPlugin_trustedPathPlugin_forwardsArgs is the regression test for
// issue #240: a trusted PATH plugin (datumctl-<name> / milo-<name>) must receive
// its full argument list verbatim, exactly like a managed plugin, so that flags
// belonging to the plugin's own subcommands (e.g. "-o wide") are not rejected by
// cobra as unknown datumctl flags. Before the fix, ForwardPlugin returned early
// for unmanaged plugins and the args fell through to cobra, which required an
// undocumented "--" separator.
//
// execPlatform is overridden so the plugin is not actually exec'd (which would
// replace the test process); we capture the binary path and args instead.
func TestForwardPlugin_trustedPathPlugin_forwardsArgs(t *testing.T) {
	// Not parallel — mutates os.Args, execPlatform, and environment.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	binaryPath := writeFakeBinary(t, pathDir, "datumctl-search")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	// Trust the plugin: record its symlink-resolved path + SHA256 in plugins.json,
	// which is exactly what `datumctl plugin trust` writes.
	resolved, err := filepath.EvalSymlinks(binaryPath)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	writeManifest(t, managedDir, &pluginstore.Manifest{
		Trusted: map[string]*pluginstore.TrustedEntry{
			"search": {Path: resolved, SHA256: sha256HexFile(t, resolved)},
		},
	})

	var gotBinary string
	var gotArgs []string
	execCalled := false
	origExec := execPlatform
	execPlatform = func(binary string, args []string, _ []string) error {
		execCalled = true
		gotBinary = binary
		gotArgs = args
		return nil
	}
	t.Cleanup(func() { execPlatform = origExec })

	orig := os.Args
	os.Args = []string{"datumctl", "search", "kinds", "-o", "wide"}
	t.Cleanup(func() { os.Args = orig })

	factory := buildMinimalFactory(t)
	root := buildMinimalCobraTree()

	if err := ForwardPlugin(managedDir, root, factory); err != nil {
		t.Fatalf("ForwardPlugin: unexpected error: %v", err)
	}
	if !execCalled {
		t.Fatal("trusted PATH plugin was not exec'd; args fall through to cobra and -o is rejected")
	}
	if gotBinary != resolved {
		t.Errorf("exec binary = %q, want symlink-resolved trusted path %q", gotBinary, resolved)
	}
	wantArgs := []string{"kinds", "-o", "wide"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Errorf("forwarded args = %v, want %v (verbatim, no -- separator)", gotArgs, wantArgs)
	}
}

// TestForwardPlugin_untrustedPathPlugin_notExeced verifies the security invariant:
// an unmanaged PATH binary that has NOT been trusted is never exec'd and never
// handed DATUM_* credentials by ForwardPlugin. It falls through (returns nil) so
// cobra's root RunE can surface the "has not been trusted" guidance unchanged.
func TestForwardPlugin_untrustedPathPlugin_notExeced(t *testing.T) {
	// Not parallel — mutates os.Args, execPlatform, and environment.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	writeFakeBinary(t, pathDir, "datumctl-search")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	// No plugins.json trust entry and no env-var override — IsTrusted is false.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	execCalled := false
	origExec := execPlatform
	execPlatform = func(_ string, _ []string, _ []string) error {
		execCalled = true
		return nil
	}
	t.Cleanup(func() { execPlatform = origExec })

	orig := os.Args
	os.Args = []string{"datumctl", "search", "kinds"}
	t.Cleanup(func() { os.Args = orig })

	root := buildMinimalCobraTree()

	// factory is nil: this path must return before Exec, so BuildEnv is never
	// reached. A non-nil return or an exec here would be a security regression.
	if err := ForwardPlugin(managedDir, root, nil); err != nil {
		t.Fatalf("ForwardPlugin untrusted PATH plugin: want nil (fall through), got %v", err)
	}
	if execCalled {
		t.Fatal("untrusted PATH binary was exec'd; it must fall through to cobra without credential injection")
	}
}

// TestForwardPlugin_managedPlugin_forwardsArgs verifies the managed path is
// untouched by the unmanaged-trust refactor: a recorded managed plugin whose
// SHA256 matches is still exec'd with its args forwarded verbatim.
func TestForwardPlugin_managedPlugin_forwardsArgs(t *testing.T) {
	// Not parallel — mutates os.Args, execPlatform, and environment.
	managedDir := t.TempDir()
	// Isolate PATH so only the managed binary resolves the plugin name.
	t.Setenv("PATH", t.TempDir())

	bin := writeFakeBinary(t, managedDir, "ipam")
	recordManaged(t, managedDir, "ipam", sha256HexFile(t, bin))

	var gotBinary string
	var gotArgs []string
	execCalled := false
	origExec := execPlatform
	execPlatform = func(binary string, args []string, _ []string) error {
		execCalled = true
		gotBinary = binary
		gotArgs = args
		return nil
	}
	t.Cleanup(func() { execPlatform = origExec })

	orig := os.Args
	os.Args = []string{"datumctl", "ipam", "allocate", "-o", "json"}
	t.Cleanup(func() { os.Args = orig })

	factory := buildMinimalFactory(t)
	root := buildMinimalCobraTree()

	if err := ForwardPlugin(managedDir, root, factory); err != nil {
		t.Fatalf("ForwardPlugin managed plugin: unexpected error: %v", err)
	}
	if !execCalled {
		t.Fatal("managed plugin was not exec'd")
	}
	wantBinary, _ := filepath.Abs(bin)
	if gotBinary != wantBinary {
		t.Errorf("exec binary = %q, want %q", gotBinary, wantBinary)
	}
	wantArgs := []string{"allocate", "-o", "json"}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Errorf("forwarded args = %v, want %v", gotArgs, wantArgs)
	}
}

// TestForwardPlugin_managedTamperedBinary_returnsError verifies the managed
// integrity check still fails closed inside ForwardPlugin: a managed binary whose
// bytes changed after the SHA256 was recorded is rejected (returned as an error)
// rather than exec'd.
func TestForwardPlugin_managedTamperedBinary_returnsError(t *testing.T) {
	// Not parallel — mutates os.Args, execPlatform, and environment.
	managedDir := t.TempDir()
	t.Setenv("PATH", t.TempDir())

	bin := writeFakeBinary(t, managedDir, "ipam")
	recordManaged(t, managedDir, "ipam", sha256HexFile(t, bin))
	// Swap the bytes after recording the hash.
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho pwned\n"), 0o755); err != nil {
		t.Fatalf("rewrite binary: %v", err)
	}

	execCalled := false
	origExec := execPlatform
	execPlatform = func(_ string, _ []string, _ []string) error {
		execCalled = true
		return nil
	}
	t.Cleanup(func() { execPlatform = origExec })

	orig := os.Args
	os.Args = []string{"datumctl", "ipam", "allocate"}
	t.Cleanup(func() { os.Args = orig })

	root := buildMinimalCobraTree()

	err := ForwardPlugin(managedDir, root, buildMinimalFactory(t))
	if err == nil {
		t.Fatal("tampered managed binary must return an error, got nil")
	}
	if execCalled {
		t.Fatal("tampered managed binary must not be exec'd")
	}
}
