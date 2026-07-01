package plugindispatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

// recordManaged writes a plugins.json with a single managed install record for
// name carrying the given SHA256 (which may be empty to model a record with no
// recorded hash).
func recordManaged(t *testing.T, dir, name, sha string) {
	t.Helper()
	writeManifest(t, dir, &pluginstore.Manifest{
		Plugins: map[string]*pluginstore.InstalledPlugin{
			name: {Source: name, Version: "v0.1.0", SHA256: sha},
		},
	})
}

// TestVerifyManagedPluginIntegrity_genericMatchingRecord_accepted is the
// positive control: a generic-named managed binary whose on-disk bytes match
// the recorded SHA256 is accepted.
func TestVerifyManagedPluginIntegrity_genericMatchingRecord_accepted(t *testing.T) {
	managedDir := t.TempDir()
	bin := writeFakeBinary(t, managedDir, "ipam")
	recordManaged(t, managedDir, "ipam", sha256HexFile(t, bin))

	if err := VerifyManagedPluginIntegrity(managedDir, "ipam", bin); err != nil {
		t.Fatalf("matching record should be accepted, got error: %v", err)
	}
}

// TestVerifyManagedPluginIntegrity_hashMismatch_rejected verifies that swapping
// the on-disk bytes after the SHA256 was recorded causes rejection.
func TestVerifyManagedPluginIntegrity_hashMismatch_rejected(t *testing.T) {
	managedDir := t.TempDir()
	bin := writeFakeBinary(t, managedDir, "ipam")
	recordManaged(t, managedDir, "ipam", sha256HexFile(t, bin))

	// Swap the binary's bytes after the hash was recorded.
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho pwned\n"), 0o755); err != nil {
		t.Fatalf("rewrite binary: %v", err)
	}

	err := VerifyManagedPluginIntegrity(managedDir, "ipam", bin)
	if err == nil {
		t.Fatal("hash mismatch must be rejected, got nil")
	}
	if !strings.Contains(err.Error(), "modified since install") {
		t.Errorf("error %q should mention the binary was modified", err.Error())
	}
}

// TestVerifyManagedPluginIntegrity_genericNoRecord_failsClosed verifies the new
// fail-closed behavior: a generic-named binary in the managed dir with NO
// plugins.json record is rejected (it could otherwise be exec'd with
// DATUM_CREDENTIALS_HELPER without any integrity check).
func TestVerifyManagedPluginIntegrity_genericNoRecord_failsClosed(t *testing.T) {
	managedDir := t.TempDir()
	bin := writeFakeBinary(t, managedDir, "ipam")
	// No plugins.json written at all (Load returns an empty manifest).

	err := VerifyManagedPluginIntegrity(managedDir, "ipam", bin)
	if err == nil {
		t.Fatal("unrecorded generic managed binary must fail closed, got nil")
	}
	if !strings.Contains(err.Error(), "ipam") {
		t.Errorf("error %q should name the plugin", err.Error())
	}
}

// TestVerifyManagedPluginIntegrity_genericEmptyHash_failsClosed verifies that a
// record present but carrying an empty SHA256 is treated as unverifiable and
// rejected for a generic-named binary.
func TestVerifyManagedPluginIntegrity_genericEmptyHash_failsClosed(t *testing.T) {
	managedDir := t.TempDir()
	bin := writeFakeBinary(t, managedDir, "ipam")
	recordManaged(t, managedDir, "ipam", "") // empty recorded hash

	if err := VerifyManagedPluginIntegrity(managedDir, "ipam", bin); err == nil {
		t.Fatal("generic managed binary with empty recorded hash must fail closed, got nil")
	}
}

// TestVerifyManagedPluginIntegrity_legacyNoRecord_accepted verifies the
// preserved legacy exception: a datumctl-<name> binary in the managed dir that
// predates install records is still allowed through without a record.
func TestVerifyManagedPluginIntegrity_legacyNoRecord_accepted(t *testing.T) {
	managedDir := t.TempDir()
	legacy := writeFakeBinary(t, managedDir, "datumctl-dns")
	// No plugins.json record for "dns".

	if err := VerifyManagedPluginIntegrity(managedDir, "dns", legacy); err != nil {
		t.Fatalf("legacy datumctl- managed binary without a record must be accepted, got: %v", err)
	}
}

// TestVerifyManagedPluginIntegrity_legacyRecordedHash_stillChecked verifies the
// legacy exception does NOT disable hash verification when a record IS present:
// a recorded datumctl-<name> with mismatched bytes is still rejected.
func TestVerifyManagedPluginIntegrity_legacyRecordedHash_stillChecked(t *testing.T) {
	managedDir := t.TempDir()
	legacy := writeFakeBinary(t, managedDir, "datumctl-dns")
	recordManaged(t, managedDir, "dns", sha256HexFile(t, legacy))

	if err := os.WriteFile(legacy, []byte("#!/bin/sh\necho tampered\n"), 0o755); err != nil {
		t.Fatalf("rewrite legacy binary: %v", err)
	}
	if err := VerifyManagedPluginIntegrity(managedDir, "dns", legacy); err == nil {
		t.Fatal("recorded legacy binary with mismatched bytes must still be rejected")
	}
}

// TestVerifyManagedPluginIntegrity_genericNoRecord_unreadableManifest verifies
// the manifest-unreadable branch also fails closed for a generic name. A
// directory at plugins.json makes Load return an error (not os.IsNotExist).
func TestVerifyManagedPluginIntegrity_genericNoRecord_unreadableManifest(t *testing.T) {
	managedDir := t.TempDir()
	bin := writeFakeBinary(t, managedDir, "ipam")
	// Make plugins.json unreadable as a file by creating a directory in its place.
	if err := os.Mkdir(filepath.Join(managedDir, "plugins.json"), 0o755); err != nil {
		t.Fatalf("mkdir plugins.json: %v", err)
	}

	if err := VerifyManagedPluginIntegrity(managedDir, "ipam", bin); err == nil {
		t.Fatal("generic managed binary must fail closed when the manifest is unreadable")
	}
}
