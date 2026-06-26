package plugindispatch

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/pluginstore"
)

// TestFindPlugin_genericManagedName verifies a catalog/managed plugin stored
// under its generic name (e.g. "ipam") is found in the managed dir.
func TestFindPlugin_genericManagedName(t *testing.T) {
	managedDir := t.TempDir()
	writeFakeBinary(t, managedDir, "ipam")

	gotPath, managed, err := FindPlugin("ipam", managedDir)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if !managed {
		t.Error("generic managed binary should be reported as managed")
	}
	if filepath.Base(gotPath) != "ipam" {
		t.Errorf("path %q should resolve to the generic name", gotPath)
	}
}

// TestFindPlugin_genericNameNeverFromPath is the SECURITY boundary: a bare
// generic binary on PATH must NOT be treated as a plugin; only the
// datumctl-<name> prefix marks a PATH binary as a datumctl plugin.
func TestFindPlugin_genericNameNeverFromPath(t *testing.T) {
	emptyManaged := t.TempDir()
	pathDir := t.TempDir()
	t.Setenv("PATH", pathDir)

	// A bare "ipam" on PATH is not a plugin.
	writeFakeBinary(t, pathDir, "ipam")
	if _, _, err := FindPlugin("ipam", emptyManaged); err == nil {
		t.Fatal("a bare generic binary on PATH must NOT be resolved as a plugin")
	}

	// But datumctl-ipam on PATH is.
	writeFakeBinary(t, pathDir, "datumctl-ipam")
	gotPath, managed, err := FindPlugin("ipam", emptyManaged)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if managed {
		t.Error("PATH binary should be reported unmanaged")
	}
	if !strings.Contains(gotPath, "datumctl-ipam") {
		t.Errorf("path %q should be the datumctl-prefixed PATH binary", gotPath)
	}
}

// TestFindPlugin_legacyManagedNameStillFound verifies back-compat: a plugin
// installed before generic naming (datumctl-<name> in the managed dir) is still
// found and reported as managed.
func TestFindPlugin_legacyManagedNameStillFound(t *testing.T) {
	managedDir := t.TempDir()
	writeFakeBinary(t, managedDir, "datumctl-dns")

	gotPath, managed, err := FindPlugin("dns", managedDir)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if !managed {
		t.Error("legacy managed binary should be reported as managed")
	}
	if filepath.Base(gotPath) != "datumctl-dns" {
		t.Errorf("path %q should resolve to the legacy datumctl- name", gotPath)
	}
}

// TestListPluginNames_fromInstallRecord verifies managed plugin names are
// derived from plugins.json, not a filename-prefix scan of the managed dir
// (generic names cannot be prefix-detected).
func TestListPluginNames_fromInstallRecord(t *testing.T) {
	managedDir := t.TempDir()
	// Isolate PATH so it contributes no datumctl-* candidates.
	t.Setenv("PATH", t.TempDir())

	// Record two managed plugins under generic names.
	m := &pluginstore.Manifest{Plugins: map[string]*pluginstore.InstalledPlugin{
		"ipam": {Source: "ipam", Catalog: "default", Version: "v0.1.0", InstalledAt: time.Now().UTC()},
		"dns":  {Source: "dns", Catalog: "default", Version: "v1.2.3", InstalledAt: time.Now().UTC()},
	}}
	if err := pluginstore.Save(managedDir, m); err != nil {
		t.Fatal(err)
	}
	// A stray non-recorded file must NOT be surfaced.
	writeFakeBinary(t, managedDir, "datumctl-stray")

	names := ListPluginNames(managedDir, map[string]string{"ipam": "IPAM service"})

	joined := strings.Join(names, "\n")
	if !strings.Contains(joined, "ipam") || !strings.Contains(joined, "dns") {
		t.Fatalf("expected ipam and dns from the install record, got: %q", names)
	}
	if !strings.Contains(joined, "IPAM service") {
		t.Errorf("description from the passed map should be used: %q", names)
	}
	if strings.Contains(joined, "stray") {
		t.Errorf("non-recorded files must not be listed: %q", names)
	}
}
