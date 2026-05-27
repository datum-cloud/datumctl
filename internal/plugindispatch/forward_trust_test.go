package plugindispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

// writeManifest serializes a Manifest to plugins.json in dir.
func writeManifest(t *testing.T, dir string, m *pluginstore.Manifest) {
	t.Helper()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), data, 0o600); err != nil {
		t.Fatalf("write plugins.json: %v", err)
	}
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

// TestForwardCompletion_untrustedPathPlugin_returnsNil verifies that
// ForwardCompletion silently returns nil for a PATH plugin that has not been
// trusted. The binary must not be exec'd or receive DATUM_CREDENTIALS_HELPER.
func TestForwardCompletion_untrustedPathPlugin_returnsNil(t *testing.T) {
	// Not parallel — manipulates os.Args and environment.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	writeFakeBinary(t, pathDir, "datumctl-myext")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	orig := os.Args
	os.Args = []string{"datumctl", "__complete", "myext", ""}
	t.Cleanup(func() { os.Args = orig })

	// No trust entry in plugins.json — IsTrusted must return false.
	err := ForwardCompletion(managedDir, nil)
	if err != nil {
		t.Errorf("ForwardCompletion untrusted PATH plugin: want nil, got %v", err)
	}
}

// TestForwardHelp_untrustedPathPlugin_returnsError verifies that ForwardHelp
// returns an error (not nil, not os.Exit) when a PATH plugin has not been trusted.
func TestForwardHelp_untrustedPathPlugin_returnsError(t *testing.T) {
	// Not parallel — manipulates os.Args and environment.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	writeFakeBinary(t, pathDir, "datumctl-myext")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	orig := os.Args
	os.Args = []string{"datumctl", "myext", "--help"}
	t.Cleanup(func() { os.Args = orig })

	err := ForwardHelp(managedDir)
	if err == nil {
		t.Fatal("ForwardHelp untrusted PATH plugin: want error, got nil")
	}
	if !strings.Contains(err.Error(), "has not been trusted") {
		t.Errorf("error %q does not contain expected text %q", err.Error(), "has not been trusted")
	}
	if !strings.Contains(err.Error(), "datumctl plugin trust myext") {
		t.Errorf("error %q does not contain actionable trust command", err.Error())
	}
}

// TestForwardHelp_untrustedPathPlugin_shortFlag verifies the same trust block
// fires when the short flag -h is used instead of --help.
func TestForwardHelp_untrustedPathPlugin_shortFlag(t *testing.T) {
	// Not parallel — manipulates os.Args and environment.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	writeFakeBinary(t, pathDir, "datumctl-myext")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	orig := os.Args
	os.Args = []string{"datumctl", "myext", "-h"}
	t.Cleanup(func() { os.Args = orig })

	err := ForwardHelp(managedDir)
	if err == nil {
		t.Fatal("ForwardHelp untrusted PATH plugin (-h): want error, got nil")
	}
	if !strings.Contains(err.Error(), "has not been trusted") {
		t.Errorf("error %q does not contain expected text %q", err.Error(), "has not been trusted")
	}
}

// TestForwardHelp_managedPlugin_noTrustCheck verifies that a binary in the
// managed directory is treated as managed=true by FindPlugin, meaning the trust
// gate in ForwardHelp is never entered. We validate via FindPlugin directly
// rather than calling ForwardHelp end-to-end (which would os.Exit on exec).
func TestForwardHelp_managedPlugin_noTrustCheck(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	writeFakeBinary(t, managedDir, "datumctl-myext")

	_, managed, err := FindPlugin("myext", managedDir)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if !managed {
		t.Error("expected managed=true for binary in managed dir; ForwardHelp would wrongly apply trust gate")
	}
}

// TestForwardCompletion_managedPlugin_noTrustCheck mirrors the above for the
// completion forward path.
func TestForwardCompletion_managedPlugin_noTrustCheck(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	writeFakeBinary(t, managedDir, "datumctl-myext")

	_, managed, err := FindPlugin("myext", managedDir)
	if err != nil {
		t.Fatalf("FindPlugin: %v", err)
	}
	if !managed {
		t.Error("expected managed=true for binary in managed dir; ForwardCompletion would wrongly apply trust gate")
	}
}

// TestForwardHelp_trustedPathPlugin_isTrustedReturnsTrue verifies that a PATH
// plugin with a valid trust entry in plugins.json passes IsTrusted — the gate
// ForwardHelp uses — so it would proceed to exec rather than return an error.
func TestForwardHelp_trustedPathPlugin_isTrustedReturnsTrue(t *testing.T) {
	// Not parallel — uses t.Setenv.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	binaryPath := writeFakeBinary(t, pathDir, "datumctl-myext")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	// Resolve symlinks so path comparison in IsTrusted is stable.
	resolvedPath, resolveErr := filepath.EvalSymlinks(binaryPath)
	if resolveErr == nil {
		binaryPath = resolvedPath
	}

	digest := sha256HexFile(t, binaryPath)
	writeManifest(t, managedDir, &pluginstore.Manifest{
		Trusted: map[string]*pluginstore.TrustedEntry{
			"myext": {Path: binaryPath, SHA256: digest},
		},
	})

	if !pluginstore.IsTrusted(managedDir, "myext", binaryPath) {
		t.Error("IsTrusted returned false for plugin with valid trust entry; ForwardHelp would block a legitimately-trusted plugin")
	}
}

// TestForwardCompletion_trustedPathPlugin_isTrustedReturnsTrue mirrors the
// above for the completion path.
func TestForwardCompletion_trustedPathPlugin_isTrustedReturnsTrue(t *testing.T) {
	// Not parallel — uses t.Setenv.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	binaryPath := writeFakeBinary(t, pathDir, "datumctl-myext")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	resolvedPath, resolveErr := filepath.EvalSymlinks(binaryPath)
	if resolveErr == nil {
		binaryPath = resolvedPath
	}

	digest := sha256HexFile(t, binaryPath)
	writeManifest(t, managedDir, &pluginstore.Manifest{
		Trusted: map[string]*pluginstore.TrustedEntry{
			"myext": {Path: binaryPath, SHA256: digest},
		},
	})

	if !pluginstore.IsTrusted(managedDir, "myext", binaryPath) {
		t.Error("IsTrusted returned false for plugin with valid trust entry; ForwardCompletion would block a legitimately-trusted plugin")
	}
}

// TestForwardCompletion_envVarTrusted_isTrustedReturnsTrue verifies the
// DATUMCTL_TRUSTED_PLUGINS env-var override causes IsTrusted to return true,
// so a trusted-by-env plugin would pass the ForwardCompletion gate.
func TestForwardCompletion_envVarTrusted_isTrustedReturnsTrue(t *testing.T) {
	// Not parallel — uses t.Setenv.
	managedDir := t.TempDir()
	t.Setenv("DATUMCTL_TRUSTED_PLUGINS", "myext,other")

	// binaryPath doesn't matter for env-var trust — IsTrusted checks env first.
	if !pluginstore.IsTrusted(managedDir, "myext", "/some/path") {
		t.Error("IsTrusted returned false when plugin name is in DATUMCTL_TRUSTED_PLUGINS")
	}
}

// TestForwardHelp_noHelpFlag_returnsNil verifies that ForwardHelp is a no-op
// when neither --help nor -h is present in os.Args.
func TestForwardHelp_noHelpFlag_returnsNil(t *testing.T) {
	// Not parallel — manipulates os.Args.
	pathDir := t.TempDir()
	managedDir := t.TempDir()

	writeFakeBinary(t, pathDir, "datumctl-myext")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+origPath)

	orig := os.Args
	os.Args = []string{"datumctl", "myext", "subcommand"}
	t.Cleanup(func() { os.Args = orig })

	err := ForwardHelp(managedDir)
	if err != nil {
		t.Errorf("ForwardHelp without --help flag: want nil, got %v", err)
	}
}
