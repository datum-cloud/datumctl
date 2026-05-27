package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

// executeTrustCmd runs the trust subcommand against a temporary plugins dir
// with the given PATH and returns the error (if any) and stdout.
func executeTrustCmd(t *testing.T, dir, name, extraPATH string) (string, error) {
	t.Helper()

	cmd := Command(nil)
	cmd.PersistentFlags().Set("plugins-dir", dir) //nolint:errcheck

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"trust", name})

	if extraPATH != "" {
		origPath := os.Getenv("PATH")
		t.Setenv("PATH", extraPATH+string(os.PathListSeparator)+origPath)
	}

	err := cmd.Execute()
	return out.String(), err
}

// executeUntrustCmd runs the untrust subcommand against a temporary plugins dir.
func executeUntrustCmd(t *testing.T, dir, name string) (string, error) {
	t.Helper()

	cmd := Command(nil)
	cmd.PersistentFlags().Set("plugins-dir", dir) //nolint:errcheck

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"untrust", name})

	err := cmd.Execute()
	return out.String(), err
}

// TestTrustCmd_writesRecord verifies that trusting a real executable writes an
// absolute path, a SHA256 hash, and a timestamp to plugins.json.
func TestTrustCmd_writesRecord(t *testing.T) {
	// Not parallel — uses t.Setenv.
	dir := t.TempDir()
	pathDir := t.TempDir()

	// Write a real executable on PATH.
	binPath := filepath.Join(pathDir, "datumctl-mytools")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	_, err := executeTrustCmd(t, dir, "mytools", pathDir)
	if err != nil {
		t.Fatalf("trust cmd: %v", err)
	}

	manifest, err := pluginstore.Load(dir)
	if err != nil {
		t.Fatalf("Load manifest: %v", err)
	}

	entry, ok := manifest.Trusted["mytools"]
	if !ok {
		t.Fatal("trusted entry 'mytools' not found in plugins.json")
	}
	if !filepath.IsAbs(entry.Path) {
		t.Errorf("trusted path %q is not absolute", entry.Path)
	}
	if !strings.Contains(entry.Path, "datumctl-mytools") {
		t.Errorf("trusted path %q does not reference the binary", entry.Path)
	}
	if entry.TrustedAt.IsZero() {
		t.Error("TrustedAt is zero — timestamp not written")
	}
	// H1: SHA256 must be populated so binary replacement is detected at runtime.
	if entry.SHA256 == "" {
		t.Error("SHA256 is empty — hash not recorded at trust time")
	}
	if len(entry.SHA256) != 64 {
		t.Errorf("SHA256 %q has unexpected length %d (want 64 hex chars)", entry.SHA256, len(entry.SHA256))
	}
}

// TestTrustCmd_sha256ChangesWhenBinaryChanges verifies that trusting the same
// name after the binary is replaced stores a different hash. This proves the
// hash reflects the actual file content, not a cached value.
func TestTrustCmd_sha256ChangesWhenBinaryChanges(t *testing.T) {
	// Not parallel — uses t.Setenv.
	dir := t.TempDir()
	pathDir := t.TempDir()

	binPath := filepath.Join(pathDir, "datumctl-myplugin")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho v1\n"), 0o755); err != nil {
		t.Fatalf("write v1 binary: %v", err)
	}

	_, err := executeTrustCmd(t, dir, "myplugin", pathDir)
	if err != nil {
		t.Fatalf("trust v1: %v", err)
	}
	manifest1, _ := pluginstore.Load(dir)
	hash1 := manifest1.Trusted["myplugin"].SHA256

	// Replace the binary with different content.
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho v2\n"), 0o755); err != nil {
		t.Fatalf("write v2 binary: %v", err)
	}

	_, err = executeTrustCmd(t, dir, "myplugin", pathDir)
	if err != nil {
		t.Fatalf("trust v2: %v", err)
	}
	manifest2, _ := pluginstore.Load(dir)
	hash2 := manifest2.Trusted["myplugin"].SHA256

	if hash1 == hash2 {
		t.Errorf("SHA256 did not change after binary replacement: %s", hash1)
	}
}

// TestTrustCmd_managedDirPreferredOverPath verifies that when the binary exists
// in the managed plugins directory, it is used instead of searching PATH.
func TestTrustCmd_managedDirPreferredOverPath(t *testing.T) {
	// Not parallel — uses t.Setenv.
	dir := t.TempDir()
	pathDir := t.TempDir()

	// Write a binary in the managed dir.
	managedBin := filepath.Join(dir, "datumctl-myplugin")
	if err := os.WriteFile(managedBin, []byte("#!/bin/sh\n# managed\n"), 0o755); err != nil {
		t.Fatalf("write managed binary: %v", err)
	}

	// Write a different binary on PATH — should NOT be used.
	pathBin := filepath.Join(pathDir, "datumctl-myplugin")
	if err := os.WriteFile(pathBin, []byte("#!/bin/sh\n# path\n"), 0o755); err != nil {
		t.Fatalf("write path binary: %v", err)
	}

	_, err := executeTrustCmd(t, dir, "myplugin", pathDir)
	if err != nil {
		t.Fatalf("trust cmd: %v", err)
	}

	manifest, err := pluginstore.Load(dir)
	if err != nil {
		t.Fatalf("Load manifest: %v", err)
	}

	entry, ok := manifest.Trusted["myplugin"]
	if !ok {
		t.Fatal("trusted entry not found")
	}
	if !strings.Contains(entry.Path, dir) {
		t.Errorf("trusted path %q should be inside managed dir %q, not PATH", entry.Path, dir)
	}
}

// TestUntrustCmd_removesRecord verifies that untrusting a plugin removes the
// trusted entry from plugins.json.
func TestUntrustCmd_removesRecord(t *testing.T) {
	// Not parallel — uses t.Setenv via executeTrustCmd.
	dir := t.TempDir()
	pathDir := t.TempDir()

	binPath := filepath.Join(pathDir, "datumctl-mytools")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	// First trust it.
	_, err := executeTrustCmd(t, dir, "mytools", pathDir)
	if err != nil {
		t.Fatalf("trust cmd: %v", err)
	}

	// Now untrust it.
	_, err = executeUntrustCmd(t, dir, "mytools")
	if err != nil {
		t.Fatalf("untrust cmd: %v", err)
	}

	manifest, err := pluginstore.Load(dir)
	if err != nil {
		t.Fatalf("Load manifest: %v", err)
	}

	if manifest.Trusted != nil && manifest.Trusted["mytools"] != nil {
		t.Error("trusted entry 'mytools' still present after untrust")
	}
}

// TestTrustCmd_errorWhenBinaryNotFound verifies that trust returns an error when
// the binary is not on PATH and not in the managed dir.
func TestTrustCmd_errorWhenBinaryNotFound(t *testing.T) {
	// Not parallel — uses t.Setenv.
	dir := t.TempDir()

	// Set PATH to an empty dir so the binary cannot be found.
	t.Setenv("PATH", t.TempDir())

	cmd := Command(nil)
	cmd.PersistentFlags().Set("plugins-dir", dir) //nolint:errcheck

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"trust", "nonexistent"})

	err := cmd.Execute()
	if err == nil {
		t.Error("trust nonexistent binary: want error, got nil")
	}
}
