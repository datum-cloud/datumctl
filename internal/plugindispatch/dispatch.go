// Package plugindispatch handles plugin discovery, environment passthrough,
// process replacement, and shell completion forwarding for datumctl plugins.
package plugindispatch

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/pluginstore"
)

// osExit is a package-level indirection over os.Exit so tests can capture the
// exit code without terminating the test process.
var osExit = os.Exit

// Exec replaces the current process with the plugin binary. Sets DATUM_*
// environment variables before exec. On success this function does not return
// (Unix: process image is replaced; Windows: never reached because the child
// exit code is propagated via osExit).
// Returns an error only if the exec setup fails or the plugin could not be
// launched at all (e.g. binary not found, permission denied).
func Exec(binaryPath string, args []string, factory *client.DatumCloudFactory) error {
	env, err := BuildEnv(factory)
	if err != nil {
		return fmt.Errorf("build plugin environment: %w", err)
	}
	merged := overlayEnv(os.Environ(), env)
	if execErr := execPlatform(binaryPath, args, merged); execErr != nil {
		// On Windows, kubectl's DefaultPluginHandler returns the child's
		// *exec.ExitError rather than calling os.Exit itself. Propagate the
		// exit code so datumctl's exit status matches the plugin's.
		if code, ok := exitCodeFromErr(execErr); ok {
			osExit(code)
		}
		return execErr
	}
	return nil
}

// exitCodeFromErr extracts an exit code from err if err (or any error in its
// chain) implements ExitCode() int — the interface satisfied by *exec.ExitError.
// Using an interface rather than a concrete type makes the helper testable with
// synthetic error values without importing os/exec in the test.
func exitCodeFromErr(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return coder.ExitCode(), true
	}
	return 0, false
}

// PluginAPIVersion is the current plugin API version declared by this host.
const PluginAPIVersion = 1

// pluginPATHPrefixes are the filename prefixes that mark a bare binary on PATH
// as a datumctl plugin. "milo-" identifies portable milo-os platform plugins
// (host-agnostic, e.g. milo-ipam); "datumctl-" identifies datumctl-native
// plugins. A bare/generic name is NEVER resolved from PATH — a generic name is
// trusted only from the integrity-checked managed directory.
var pluginPATHPrefixes = []string{"datumctl-", "milo-"}

// FindPlugin resolves a plugin name to an absolute binary path.
// Managed dir is searched first, then PATH.
// Returns (path, isManaged, error).
//
// Managed (catalog/index-installed) plugins are stored under their GENERIC name
// (e.g. "ipam"), so the managed lookup resolves "<pluginsDir>/<name>" first and
// falls back to a legacy "<pluginsDir>/datumctl-<name>" file for plugins
// installed before generic naming. The PATH fallback deliberately remains
// "datumctl-<name>" ONLY: a generic name is trusted as a plugin solely from the
// integrity-checked managed directory, never from a bare PATH binary.
func FindPlugin(name, pluginsDir string) (path string, managed bool, err error) {
	// Search managed dir first, generic name then legacy datumctl- prefix.
	if pluginsDir != "" {
		if managedPath, ok := findManagedBinary(pluginsDir, name); ok {
			abs, absErr := filepath.Abs(managedPath)
			if absErr != nil {
				return "", false, fmt.Errorf("resolve managed plugin path: %w", absErr)
			}
			return abs, true, nil
		}
	}

	// Fall back to PATH — only a recognized plugin prefix (milo- or datumctl-)
	// marks a bare PATH binary as a datumctl plugin. Generic names are never
	// resolved from PATH.
	for _, prefix := range pluginPATHPrefixes {
		if found, lookErr := exec.LookPath(prefix + name); lookErr == nil {
			return found, false, nil
		}
	}
	return "", false, fmt.Errorf("plugin %q not found in managed directory or PATH", name)
}

// findManagedBinary returns the path to a managed plugin binary, trying the
// generic name first and a legacy datumctl-<name> file second (with a .exe
// variant on Windows). The returned path is reported only when it is a regular
// file, so a directory such as the per-catalog cache cannot shadow a plugin.
func findManagedBinary(pluginsDir, name string) (string, bool) {
	candidates := []string{name}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, name+".exe")
	}
	candidates = append(candidates, "datumctl-"+name)
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "datumctl-"+name+".exe")
	}
	for _, c := range candidates {
		p := filepath.Join(pluginsDir, c)
		if info, statErr := os.Stat(p); statErr == nil && !info.IsDir() {
			return p, true
		}
	}
	return "", false
}

// BuildEnv constructs the plugin ENV overlay as a []string suitable for
// appending to os.Environ(). It sets the six DATUM_* variables.
// Exported for testing.
func BuildEnv(factory *client.DatumCloudFactory) ([]string, error) {
	// Resolve DATUM_CREDENTIALS_HELPER — must be absolute.
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(exePath)
	if err == nil {
		exePath = resolved
	}

	// Resolve scope.
	project, org, _, scopeErr := factory.ConfigFlags.ResolvedScope()
	if scopeErr != nil {
		// Non-fatal: pass empty strings. Plugin can decide whether org/project are required.
		project, org = "", ""
	}

	// Resolve API host.
	apiHost, hostErr := authutil.GetAPIHostname()
	if hostErr != nil {
		apiHost = ""
	}

	// Resolve active session.
	sessionName := ""
	cfg, cfgErr := datumconfig.LoadAuto()
	if cfgErr == nil && cfg != nil {
		sessionName = cfg.ActiveSession
	}

	return []string{
		"DATUM_ORG=" + org,
		"DATUM_PROJECT=" + project,
		"DATUM_API_HOST=" + apiHost,
		fmt.Sprintf("DATUM_PLUGIN_API_VERSION=%d", PluginAPIVersion),
		"DATUM_CREDENTIALS_HELPER=" + exePath,
		"DATUM_SESSION=" + sessionName,
	}, nil
}

// ListPluginNames returns completion candidates for installed plugins.
// Each entry is "name\tdescription" (cobra tab-completion format).
// Managed plugins are listed first; PATH plugins follow with duplicates dropped.
// descriptions is an optional map of plugin name → description (from plugins.json).
func ListPluginNames(pluginsDir string, descriptions map[string]string) []string {
	seen := map[string]bool{}
	var names []string

	candidate := func(name, desc string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		if desc != "" {
			names = append(names, name+"\t"+desc)
		} else {
			names = append(names, name)
		}
	}

	// Managed plugins are recorded in plugins.json, not detectable by a filename
	// prefix (they are stored under their generic name), so the install record is
	// the source of truth for managed names.
	if pluginsDir != "" {
		if m, loadErr := pluginstore.Load(pluginsDir); loadErr == nil {
			for n, entry := range m.Plugins {
				desc := descriptions[n]
				if desc == "" && entry != nil && entry.Manifest != nil {
					desc = entry.Manifest.Description
				}
				candidate(n, desc)
			}
		}
	}

	// PATH: walk every directory and collect milo-* and datumctl-* executables.
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			for _, prefix := range pluginPATHPrefixes {
				if n, ok := strings.CutPrefix(e.Name(), prefix); ok {
					candidate(n, descriptions[n])
					break
				}
			}
		}
	}

	return names
}

// IsBuiltIn reports whether name matches a registered Cobra subcommand or a
// reserved internal name. Built-in commands always take precedence over plugins.
func IsBuiltIn(root *cobra.Command, name string) bool {
	// Reserved names that are always built-in.
	reserved := map[string]bool{
		"help":         true,
		"completion":   true,
		"__complete":   true,
		"__completeNoDesc": true,
	}
	if reserved[name] {
		return true
	}

	// Walk the live command tree.
	for _, sub := range root.Commands() {
		if sub.Name() == name {
			return true
		}
		// Check aliases.
		if slices.Contains(sub.Aliases, name) {
			return true
		}
	}
	return false
}

// CheckCompatibilityAtInvocation performs the invocation-time compatibility checks.
// Some checks are warnings (soft) rather than hard errors.
// Returns (warn, nil) for soft warnings, ("", err) for hard blocking errors.
func CheckCompatibilityAtInvocation(m *pluginstore.PluginManifest, currentVersion string, currentAPIVersion int) (warn string, err error) {
	if m == nil {
		return "", nil
	}

	// If host API version is lower than what the plugin was built against, refuse.
	if m.APIVersion > currentAPIVersion {
		return "", fmt.Errorf("plugin was built for API version %d but host only supports API version %d; upgrade datumctl to run this plugin",
			m.APIVersion, currentAPIVersion)
	}

	// If min_api_version is set and host is below it, hard block.
	if m.MinAPIVersion > 0 && currentAPIVersion < m.MinAPIVersion {
		return "", fmt.Errorf("plugin requires API version %d or higher (current: %d)",
			m.MinAPIVersion, currentAPIVersion)
	}

	var warns []string

	// Warn if plugin was built for an older API version (forward compatibility).
	if m.APIVersion > 0 && m.APIVersion < currentAPIVersion {
		warns = append(warns, fmt.Sprintf("plugin was built for API version %d (host is %d); it may not support all current features",
			m.APIVersion, currentAPIVersion))
	}

	// Warn if datumctl version is below min_datumctl_version at invocation time.
	if m.MinDatumctlVersion != "" && semver.IsValid(m.MinDatumctlVersion) {
		if semver.IsValid(currentVersion) && semver.Compare(currentVersion, m.MinDatumctlVersion) < 0 {
			warns = append(warns, fmt.Sprintf("plugin requires datumctl %s or newer (current: %s); some features may not work",
				m.MinDatumctlVersion, currentVersion))
		}
	}

	if len(warns) > 0 {
		return strings.Join(warns, "; "), nil
	}
	return "", nil
}

// overlayEnv merges overlay variables onto base, with overlay values winning.
// Overlay entries must be in "KEY=VALUE" format.
func overlayEnv(base []string, overlay []string) []string {
	// Build a set of keys that appear in the overlay.
	overrideKeys := make(map[string]struct{}, len(overlay))
	for _, kv := range overlay {
		if key, _, ok := strings.Cut(kv, "="); ok {
			overrideKeys[key] = struct{}{}
		}
	}

	// Keep base entries whose keys are not in the overlay.
	result := make([]string, 0, len(base)+len(overlay))
	for _, kv := range base {
		if key, _, ok := strings.Cut(kv, "="); ok {
			if _, dup := overrideKeys[key]; dup {
				continue
			}
		}
		result = append(result, kv)
	}
	result = append(result, overlay...)
	return result
}
