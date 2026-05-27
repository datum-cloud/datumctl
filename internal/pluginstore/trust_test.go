package pluginstore

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeFakeBinary creates an executable file at dir/name and returns its path.
func writeFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatalf("write fake binary %s: %v", name, err)
	}
	return path
}

// sha256HexFile returns the hex-encoded SHA256 of the file at path.
func sha256HexFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file for sha256: %v", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestIsTrusted_envVarGrants verifies that a name in DATUMCTL_TRUSTED_PLUGINS
// causes IsTrusted to return true without needing plugins.json.
func TestIsTrusted_envVarGrants(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "myplugin,other")

	if !IsTrusted(managedDir, "myplugin", "/irrelevant/path") {
		t.Error("IsTrusted: want true when name is in DATUMCTL_TRUSTED_PLUGINS, got false")
	}
}

// TestIsTrusted_envVarDoesNotGrantOtherNames verifies the env var only grants
// trust to the exact names listed, not to all plugins.
func TestIsTrusted_envVarDoesNotGrantOtherNames(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "myplugin")

	if IsTrusted(managedDir, "otherplugin", "/irrelevant/path") {
		t.Error("IsTrusted: want false for name not in DATUMCTL_TRUSTED_PLUGINS, got true")
	}
}

// TestIsTrusted_manifestEntryWithCorrectHash verifies that a plugins.json trusted
// entry with matching path and correct SHA256 causes IsTrusted to return true.
func TestIsTrusted_manifestEntryWithCorrectHash(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	dir := t.TempDir()
	binPath := writeFakeBinary(t, dir, "datumctl-myplugin")

	resolvedPath, err := filepath.EvalSymlinks(binPath)
	if err == nil {
		binPath = resolvedPath
	}

	digest := sha256HexFile(t, binPath)

	manifest := &Manifest{
		Trusted: map[string]*TrustedEntry{
			"myplugin": {
				Path:      binPath,
				SHA256:    digest,
				TrustedAt: time.Now(),
			},
		},
	}
	if err := Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest: %v", err)
	}

	if !IsTrusted(dir, "myplugin", binPath) {
		t.Error("IsTrusted: want true for valid trust entry, got false")
	}
}

// TestIsTrusted_manifestEntryWrongPath verifies that a mismatch in the trusted
// path causes IsTrusted to return false, even if SHA256 matches.
func TestIsTrusted_manifestEntryWrongPath(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	dir := t.TempDir()
	binPath := writeFakeBinary(t, dir, "datumctl-myplugin")

	resolvedPath, err := filepath.EvalSymlinks(binPath)
	if err == nil {
		binPath = resolvedPath
	}

	digest := sha256HexFile(t, binPath)

	manifest := &Manifest{
		Trusted: map[string]*TrustedEntry{
			"myplugin": {
				Path:   "/completely/different/path",
				SHA256: digest,
			},
		},
	}
	if err := Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest: %v", err)
	}

	if IsTrusted(dir, "myplugin", binPath) {
		t.Error("IsTrusted: want false when trusted path does not match binary path, got true")
	}
}

// TestIsTrusted_manifestEntryWrongHash verifies that a SHA256 mismatch causes
// IsTrusted to return false, blocking a binary that has been replaced since trust
// was granted.
func TestIsTrusted_manifestEntryWrongHash(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	dir := t.TempDir()
	binPath := writeFakeBinary(t, dir, "datumctl-myplugin")

	resolvedPath, err := filepath.EvalSymlinks(binPath)
	if err == nil {
		binPath = resolvedPath
	}

	manifest := &Manifest{
		Trusted: map[string]*TrustedEntry{
			"myplugin": {
				Path:   binPath,
				SHA256: "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899",
			},
		},
	}
	if err := Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest: %v", err)
	}

	if IsTrusted(dir, "myplugin", binPath) {
		t.Error("IsTrusted: want false when SHA256 does not match on-disk binary, got true")
	}
}

// TestIsTrusted_noEntry verifies that a plugin with no trusted entry returns false.
func TestIsTrusted_noEntry(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	dir := t.TempDir()
	// Empty manifest — no trusted entries at all.
	manifest := &Manifest{}
	if err := Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest: %v", err)
	}

	if IsTrusted(dir, "myplugin", "/some/path") {
		t.Error("IsTrusted: want false when no trusted entry exists, got true")
	}
}

// TestIsTrusted_emptyPluginsDir verifies that when pluginsDir is empty only
// the env-var path runs (and returns false when the env var is also absent).
func TestIsTrusted_emptyPluginsDir(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	if IsTrusted("", "myplugin", "/some/path") {
		t.Error("IsTrusted: want false with empty pluginsDir and no env var, got true")
	}
}

// TestIsTrusted_emptySHA256InEntry verifies that a trusted entry with an empty
// SHA256 is rejected (fail-closed), guarding against legacy entries.
func TestIsTrusted_emptySHA256InEntry(t *testing.T) {
	// Not parallel — uses t.Setenv.
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "")

	dir := t.TempDir()
	binPath := writeFakeBinary(t, dir, "datumctl-myplugin")

	manifest := &Manifest{
		Trusted: map[string]*TrustedEntry{
			"myplugin": {
				Path:   binPath,
				SHA256: "", // empty — legacy entry
			},
		},
	}
	if err := Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest: %v", err)
	}

	if IsTrusted(dir, "myplugin", binPath) {
		t.Error("IsTrusted: want false for trusted entry with empty SHA256 (fail-closed), got true")
	}
}
