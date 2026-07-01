package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/pluginstore"
)

// The root command's exec path (RootCmd RunE) gates an unmanaged PATH plugin on
// pluginstore.IsTrusted — the single, shared trust implementation. These tests
// pin that gate's decision so the root path cannot silently regress to
// exec'ing an untrusted binary (which would receive DATUM_CREDENTIALS_HELPER).

func writeTrustTestBinary(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
	return path
}

func sha256Hex(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read for sha256: %v", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeTrustManifest(t *testing.T, dir string, m *pluginstore.Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), data, 0o600); err != nil {
		t.Fatalf("write plugins.json: %v", err)
	}
}

// TestRootTrustGate_untrustedPathPlugin_blocked verifies an unmanaged PATH
// plugin with no trust entry and no env override is NOT trusted, so the root
// exec path blocks it.
func TestRootTrustGate_untrustedPathPlugin_blocked(t *testing.T) {
	pluginsDir := t.TempDir()
	pathDir := t.TempDir()
	bin := writeTrustTestBinary(t, pathDir, "datumctl-ext")
	resolved, _ := filepath.EvalSymlinks(bin)

	// No plugins.json, no DATUMCTL_TRUSTED_PLUGINS.
	if pluginstore.IsTrusted(pluginsDir, "ext", resolved) {
		t.Fatal("untrusted PATH plugin must not be trusted by the root exec gate")
	}
}

// TestRootTrustGate_manifestTrusted_allowed verifies a PATH plugin recorded as
// trusted in plugins.json (matching path + SHA256) passes the gate.
func TestRootTrustGate_manifestTrusted_allowed(t *testing.T) {
	pluginsDir := t.TempDir()
	pathDir := t.TempDir()
	bin := writeTrustTestBinary(t, pathDir, "datumctl-ext")
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil {
		resolved = bin
	}

	writeTrustManifest(t, pluginsDir, &pluginstore.Manifest{
		Trusted: map[string]*pluginstore.TrustedEntry{
			"ext": {Path: resolved, SHA256: sha256Hex(t, resolved), TrustedAt: time.Now().UTC()},
		},
	})

	if !pluginstore.IsTrusted(pluginsDir, "ext", resolved) {
		t.Fatal("PATH plugin with a valid trust entry must pass the root exec gate")
	}
}

// TestRootTrustGate_manifestTrusted_hashMismatch_blocked verifies that a
// trusted-by-path entry whose recorded hash no longer matches the on-disk bytes
// is rejected (the binary was swapped after trust was granted).
func TestRootTrustGate_manifestTrusted_hashMismatch_blocked(t *testing.T) {
	pluginsDir := t.TempDir()
	pathDir := t.TempDir()
	bin := writeTrustTestBinary(t, pathDir, "datumctl-ext")
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil {
		resolved = bin
	}

	writeTrustManifest(t, pluginsDir, &pluginstore.Manifest{
		Trusted: map[string]*pluginstore.TrustedEntry{
			"ext": {Path: resolved, SHA256: sha256Hex(t, resolved), TrustedAt: time.Now().UTC()},
		},
	})

	// Swap the binary's bytes after trust was recorded.
	if err := os.WriteFile(resolved, []byte("#!/bin/sh\necho swapped\n"), 0o755); err != nil {
		t.Fatalf("rewrite binary: %v", err)
	}

	if pluginstore.IsTrusted(pluginsDir, "ext", resolved) {
		t.Fatal("a swapped binary must fail the hash check and be blocked")
	}
}

// TestRootTrustGate_envTrusted_allowed verifies the DATUMCTL_TRUSTED_PLUGINS
// env override is honored by the same shared gate.
func TestRootTrustGate_envTrusted_allowed(t *testing.T) {
	pluginsDir := t.TempDir()
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "ext,other")

	// Path/hash irrelevant when the name is env-trusted.
	if !pluginstore.IsTrusted(pluginsDir, "ext", "/nonexistent/path") {
		t.Fatal("a plugin named in DATUMCTL_TRUSTED_PLUGINS must pass the gate")
	}
}
