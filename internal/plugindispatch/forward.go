package plugindispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/pluginstore"
)

// ForwardPlugin checks whether os.Args represents a plugin invocation and, if so,
// replaces the current process with the plugin binary before cobra can parse (and
// discard) flags that belong to the plugin's own subcommands. This verbatim
// forwarding is what lets "datumctl <plugin> <subcmd> -o wide" work without the
// caller having to insert a "--" separator.
//
// It execs both managed plugins (found in pluginsDir, gated by a SHA256 integrity
// check) and trusted PATH plugins (milo-<name>/datumctl-<name>, gated by the same
// pluginstore.IsTrusted check the help/completion forward paths and the root RunE
// use). An untrusted or unknown PATH binary is left for cobra, where the root RunE
// surfaces the "has not been trusted" guidance — it is never exec'd or handed
// DATUM_* credentials from here.
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
	if err != nil {
		// Not a known plugin in the managed dir or on PATH — let cobra handle it.
		return nil
	}

	if managed {
		// Verify the on-disk binary's SHA256 matches the recorded manifest entry.
		// This guards against the binary being silently replaced after install.
		if verifyErr := VerifyManagedPluginIntegrity(pluginsDir, name, binaryPath); verifyErr != nil {
			return verifyErr
		}
	} else {
		// Unmanaged PATH plugin. Trust must be established BEFORE exec because Exec
		// injects DATUM_CREDENTIALS_HELPER; an untrusted binary must never reach it.
		// Resolve symlinks once so the trust check and the subsequent exec act on
		// the same real path (TOCTOU defense, matching the root RunE and the
		// help/completion forward paths).
		if abs, absErr := filepath.EvalSymlinks(binaryPath); absErr == nil {
			binaryPath = abs
		}
		if !pluginstore.IsTrusted(pluginsDir, name, binaryPath) {
			// Untrusted/unknown binary: fall through to cobra unchanged. The root
			// RunE surfaces the "has not been trusted" guidance; do not exec or
			// inject credentials here.
			return nil
		}
	}

	return Exec(binaryPath, args[1:], factory)
}

// VerifyManagedPluginIntegrity loads plugins.json, finds the entry for name,
// hashes the binary at binaryPath, and compares against InstalledPlugin.SHA256.
//
// It fails closed: a managed binary stored under its GENERIC name (e.g. "ipam")
// is trusted only when plugins.json has a record for it with a non-empty SHA256
// that matches the on-disk bytes. A generic-named binary with no record, an
// empty recorded hash, or an unreadable manifest is REJECTED, so a bare binary
// dropped into the managed directory cannot be exec'd with DATUM_* credentials.
//
// The one documented exception is the legacy "datumctl-<name>" layout: plugins
// installed under the datumctl- prefix before install records existed predate
// the SHA256 bookkeeping, so they are allowed through without a record. A
// recorded legacy binary is still hash-checked when an entry is present.
func VerifyManagedPluginIntegrity(pluginsDir, name, binaryPath string) error {
	legacy := isLegacyManagedName(name, binaryPath)

	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		// Manifest unreadable. Legacy datumctl-<name> installs predate records
		// and must still run; generic-named binaries fail closed.
		if legacy {
			return nil
		}
		return fmt.Errorf("plugin %s has no verifiable install record (cannot read plugins.json); run 'datumctl plugin install %s' to (re)install it", name, name)
	}

	entry, ok := manifest.Plugins[name]
	if !ok || entry == nil || entry.SHA256 == "" {
		// No recorded SHA256.
		if legacy {
			return nil
		}
		return fmt.Errorf("plugin %s is not a recorded managed plugin; run 'datumctl plugin install %s' to install it", name, name)
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

// isLegacyManagedName reports whether binaryPath is the legacy
// "datumctl-<name>" managed layout (with an optional .exe suffix on Windows)
// for the given generic plugin name. Legacy installs predate install records
// and are allowed through VerifyManagedPluginIntegrity without a recorded hash.
func isLegacyManagedName(name, binaryPath string) bool {
	base := filepath.Base(binaryPath)
	if base == "datumctl-"+name {
		return true
	}
	if runtime.GOOS == "windows" && base == "datumctl-"+name+".exe" {
		return true
	}
	return false
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
	} else if VerifyManagedPluginIntegrity(pluginsDir, name, binaryPath) != nil {
		// Defense in depth: a managed plugin is exec'd with DATUM_* credentials
		// during completion. Fail closed for a bare unrecorded binary dropped
		// into the managed dir so it cannot harvest credentials on a tab-press.
		// Silently decline so cobra still handles built-in completion.
		return nil
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
