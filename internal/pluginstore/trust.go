package pluginstore

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// IsTrusted reports whether the plugin identified by name and binaryPath should
// be trusted for execution. It is the shared trust-check used by both the main
// RunE path and the completion/help forward paths.
//
// Trust is granted when either:
//  1. The plugin name appears in the DATUMCTL_TRUSTED_PLUGINS comma-separated
//     environment variable (CI / developer override), OR
//  2. plugins.json contains a trusted entry for name whose stored path matches
//     binaryPath and whose SHA256 matches the current on-disk hash.
//
// binaryPath should already be symlink-resolved (filepath.EvalSymlinks applied)
// before calling this function so that path comparison is stable.
//
// If pluginsDir is empty or plugins.json cannot be read, only the env-var check
// is performed.
func IsTrusted(pluginsDir, name, binaryPath string) bool {
	if isTrustedByEnv(name) {
		return true
	}
	if pluginsDir == "" {
		return false
	}
	manifest, err := Load(pluginsDir)
	if err != nil {
		return false
	}
	return isTrustedByManifest(manifest, name, binaryPath)
}

// isTrustedByEnv checks the DATUMCTL_TRUSTED_PLUGINS comma-separated env var.
func isTrustedByEnv(name string) bool {
	env := os.Getenv("DATUMCTL_TRUSTED_PLUGINS")
	if env == "" {
		return false
	}
	for _, trusted := range strings.Split(env, ",") {
		if strings.TrimSpace(trusted) == name {
			return true
		}
	}
	return false
}

// isTrustedByManifest checks plugins.json trusted entries for the given plugin.
// The resolved binary path must match the stored trusted path, and the SHA256
// hash of the binary on disk must match the hash recorded at trust time.
// binaryPath must already be symlink-resolved (filepath.EvalSymlinks applied)
// so that path comparison is stable and no second symlink resolution occurs.
func isTrustedByManifest(manifest *Manifest, name, binaryPath string) bool {
	if manifest == nil || manifest.Trusted == nil {
		return false
	}
	entry, ok := manifest.Trusted[name]
	if !ok {
		return false
	}
	// Path check: binaryPath is already symlink-resolved by the caller.
	if entry.Path != binaryPath {
		return false
	}
	// Hash check: re-read and re-hash the binary to detect replacement since
	// trust was granted. If SHA256 is empty (legacy entry), fail closed.
	if entry.SHA256 == "" {
		return false
	}
	f, err := os.Open(binaryPath)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	currentDigest := hex.EncodeToString(h.Sum(nil))
	return entry.SHA256 == currentDigest
}
