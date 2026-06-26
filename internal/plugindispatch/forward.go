package plugindispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/pluginstore"
)

// ForwardPlugin checks whether os.Args represents a managed-plugin invocation and,
// if so, replaces the current process with the plugin binary before cobra can
// parse (and discard) flags that belong to the plugin's own subcommands.
//
// It only execs managed plugins (found in pluginsDir). PATH-based plugins reach
// the same destination via the root RunE, where trust-checking also runs.
//
// Must be called after the cobra command tree is fully built so that IsBuiltIn
// can correctly distinguish plugin names from registered subcommands.
// On success this function does not return (process is replaced).
func ForwardPlugin(pluginsDir string, root *cobra.Command, factory *client.DatumCloudFactory) error {
	args := os.Args[1:]
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return nil
	}
	name := args[0]

	if IsBuiltIn(root, name) {
		return nil
	}

	binaryPath, managed, err := FindPlugin(name, pluginsDir)
	if err != nil || !managed {
		return nil
	}

	// Verify the on-disk binary's SHA256 matches the recorded manifest entry.
	// This guards against the binary being silently replaced after install.
	if verifyErr := VerifyManagedPluginIntegrity(pluginsDir, name, binaryPath); verifyErr != nil {
		return verifyErr
	}

	return Exec(binaryPath, args[1:], factory)
}

// VerifyManagedPluginIntegrity loads plugins.json, finds the entry for name,
// hashes the binary at binaryPath, and compares against InstalledPlugin.SHA256.
// Returns nil if the entry has no recorded SHA256 (e.g. manually placed binary).
func VerifyManagedPluginIntegrity(pluginsDir, name, binaryPath string) error {
	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		// Cannot load manifest — proceed without verification rather than blocking.
		return nil
	}

	entry, ok := manifest.Plugins[name]
	if !ok || entry == nil || entry.SHA256 == "" {
		// No recorded SHA256 — nothing to verify.
		return nil
	}

	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("read plugin binary for verification: %w", err)
	}

	sum := sha256.Sum256(data)
	gotHex := hex.EncodeToString(sum[:])

	if !strings.EqualFold(gotHex, entry.SHA256) {
		return fmt.Errorf("plugin %s binary has been modified since install; run 'datumctl plugin install %s' to reinstall", name, name)
	}
	return nil
}

// ForwardCompletion checks whether os.Args represents a __complete call for a plugin.
// If the second argument is "__complete" and the third argument resolves to a known
// plugin name (managed dir or PATH), it runs the plugin with the completion args and
// exits with the child's exit code.
//
// factory is optional; when non-nil the DATUM_* environment variables are injected
// into the plugin process so that completion handlers can authenticate API calls.
// If factory is nil or BuildEnv fails, the plugin is still executed without env
// injection (completion may return empty candidates rather than failing outright).
//
// Returns nil if not applicable (not a completion call, or name is not a plugin).
// Must be called before cobra.Execute().
func ForwardCompletion(pluginsDir string, factory *client.DatumCloudFactory) error {
	// Need at least: datumctl __complete <name> [args...]
	if len(os.Args) < 3 {
		return nil
	}
	if os.Args[1] != "__complete" {
		return nil
	}

	name := os.Args[2]

	// Find the plugin binary.
	binaryPath, managed, err := FindPlugin(name, pluginsDir)
	if err != nil {
		// Not a plugin — let Cobra handle it.
		return nil
	}

	// For PATH-based (unmanaged) plugins, verify trust before forwarding
	// completion. Without this check, any datumctl-* binary on PATH would
	// receive DATUM_CREDENTIALS_HELPER on every shell tab-press.
	if !managed {
		// Resolve symlinks so path comparison in IsTrusted is stable.
		if abs, absErr := filepath.EvalSymlinks(binaryPath); absErr == nil {
			binaryPath = abs
		}
		if !pluginstore.IsTrusted(pluginsDir, name, binaryPath) {
			// Silently decline — do not inject credentials or exec the binary.
			// Return nil so cobra can handle completion for built-in commands.
			return nil
		}
	}

	// Strip the plugin name from the forwarded args so the plugin sees
	// ["__complete", <subargs...>] rather than ["__complete", "compute", <subargs...>].
	// The plugin's own cobra tree has no knowledge of its name as a subcommand.
	pluginArgs := append([]string{"__complete"}, os.Args[3:]...)

	cmd := exec.Command(binaryPath, pluginArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Inject DATUM_* variables so the plugin can authenticate API calls during
	// completion. Non-fatal: if env construction fails we proceed without injection
	// and the plugin will return empty candidates instead of erroring.
	if factory != nil {
		if env, buildErr := BuildEnv(factory); buildErr == nil {
			cmd.Env = overlayEnv(os.Environ(), env)
		}
	}

	runErr := cmd.Run()
	if runErr == nil {
		os.Exit(0)
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	// Unexpected error running the plugin — write to stderr before exiting.
	fmt.Fprintf(os.Stderr, "error: plugin %s completion failed: %v\n", name, runErr)
	os.Exit(1)
	return nil // unreachable
}

// ForwardHelp checks whether os.Args looks like "datumctl <plugin-name> ... --help/-h".
// If so, it execs the plugin with the original args (minus the binary name) so the
// plugin's own help is shown instead of datumctl's root help.
//
// Cobra intercepts --help before RunE fires, so this must be called before
// cobra.Execute(), mirroring the ForwardCompletion pattern.
func ForwardHelp(pluginsDir string) error {
	args := os.Args[1:] // strip "datumctl"
	if len(args) == 0 {
		return nil
	}
	name := args[0]
	if strings.HasPrefix(name, "-") {
		return nil
	}

	hasHelp := false
	for _, a := range args[1:] {
		if a == "--help" || a == "-h" {
			hasHelp = true
			break
		}
	}
	if !hasHelp {
		return nil
	}

	binaryPath, managed, err := FindPlugin(name, pluginsDir)
	if err != nil {
		// Not a known plugin — let Cobra handle it normally.
		return nil
	}

	// For PATH-based (unmanaged) plugins, verify trust before forwarding help.
	// An unmanaged binary that has not been explicitly trusted must not be exec'd
	// by datumctl under any code path.
	if !managed {
		// Resolve symlinks so path comparison in IsTrusted is stable.
		if abs, absErr := filepath.EvalSymlinks(binaryPath); absErr == nil {
			binaryPath = abs
		}
		if !pluginstore.IsTrusted(pluginsDir, name, binaryPath) {
			return fmt.Errorf("'%s' is an unmanaged plugin that has not been trusted; run: datumctl plugin trust %s", filepath.Base(binaryPath), name)
		}
	}

	// Forward the args to the plugin (strip the plugin name, keep the rest).
	cmd := exec.Command(binaryPath, args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	runErr := cmd.Run()
	if runErr == nil {
		os.Exit(0)
	}

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	// Unexpected error running the plugin — write to stderr before exiting.
	fmt.Fprintf(os.Stderr, "error: plugin %s --help failed: %v\n", name, runErr)
	os.Exit(1)
	return nil // unreachable
}
