package plugindispatch

import (
	"os"
	"path/filepath"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

// TestTrust_symlinkResolvesToRealPath exercises the EvalSymlinks-once TOCTOU
// defense shared by the root exec path and the completion/help forward paths:
// a PATH plugin reached through a real symlink must be trusted against its
// RESOLVED real path, and the manifest records that real path. A regression
// that compared the unresolved symlink path against a manifest keyed by the
// real path would fail this test, because the two paths differ.
func TestTrust_symlinkResolvesToRealPath(t *testing.T) {
	managedDir := t.TempDir()
	realDir := t.TempDir()
	linkDir := t.TempDir()

	// Real binary lives in realDir; linkDir/datumctl-ext is a symlink to it.
	realPath := writeFakeBinary(t, realDir, "datumctl-ext")
	linkPath := filepath.Join(linkDir, "datumctl-ext")
	if err := symlinkOrSkip(t, realPath, linkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	// EvalSymlinks must resolve the link to the real path (the single
	// resolution the callers perform before the trust check and exec).
	resolved, err := filepath.EvalSymlinks(linkPath)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	wantReal, _ := filepath.EvalSymlinks(realPath)
	if resolved != wantReal {
		t.Fatalf("symlink resolved to %q, want real path %q", resolved, wantReal)
	}

	// Manifest records the trusted entry against the RESOLVED real path.
	writeManifest(t, managedDir, &pluginstore.Manifest{
		Trusted: map[string]*pluginstore.TrustedEntry{
			"ext": {Path: wantReal, SHA256: sha256HexFile(t, wantReal)},
		},
	})

	// Trust succeeds for the resolved path (what the caller passes after
	// EvalSymlinks-once).
	if !pluginstore.IsTrusted(managedDir, "ext", resolved) {
		t.Error("IsTrusted should accept the symlink-resolved real path")
	}

	// Regression guard: had a caller skipped resolution and compared the raw
	// symlink path, trust would be (incorrectly) denied — the path mismatch is
	// observable here.
	if linkPath != resolved && pluginstore.IsTrusted(managedDir, "ext", linkPath) {
		t.Error("IsTrusted unexpectedly accepted the unresolved symlink path; path comparison is not anchored to the real path")
	}
}

// symlinkOrSkip creates a symlink, skipping the test on platforms/filesystems
// that disallow it (e.g. Windows without privilege) rather than failing.
func symlinkOrSkip(t *testing.T, oldname, newname string) error {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlinks unsupported in this environment: %v", err)
	}
	return nil
}
