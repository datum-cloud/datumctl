package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	componentversion "k8s.io/component-base/version"

	plugincmd "go.datum.net/datumctl/internal/cmd/plugin"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

// suggestAndInstallPlugin checks the local plugin index for an exact match on
// name. If found and stdin is a TTY, it prompts the user to install and
// re-exec. Returns nil (allowing the caller to fall through to the normal
// "unknown command" error) when no match is found or the user declines.
// When stdin is not a TTY but a match is found, a one-line hint is printed to
// stderr so the user knows the plugin is available.
func suggestAndInstallPlugin(cmd *cobra.Command, name, pluginsDir string, originalArgs []string, factory *client.DatumCloudFactory) error {
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))

	idx, err := pluginstore.LoadIndex()
	if err != nil || pluginstore.IsStale(idx) {
		fresh, refreshErr := pluginstore.RefreshIndex(cmd.Context())
		if refreshErr == nil {
			idx = fresh
		}
	}

	entry := pluginstore.FindInIndex(idx, name)
	if entry == nil {
		return nil
	}

	if !isTTY {
		// Non-interactive: print a breadcrumb hint so the user knows the plugin
		// exists, then return nil so the normal "unknown command" error fires.
		fmt.Fprintf(cmd.ErrOrStderr(), "hint: %q is an available plugin; install it with: datumctl plugin install %s\n", name, name)
		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\nTo use %q you need to install the %s plugin.\n", name, name)
	if entry.Spec.ShortDescription != "" {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", entry.Spec.ShortDescription)
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "\nWould you like to install it now? [y/N] ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "\nInstalling %s...\n", entry.Name)
	currentVersion := componentversion.Get().GitVersion
	installed, _, installErr := plugincmd.InstallPlugin(cmd.Context(), pluginsDir, entry.Name, "", currentVersion, idx)
	if installErr != nil {
		return fmt.Errorf("install plugin: %w", installErr)
	}

	// Register the installed plugin in plugins.json so plugin list/remove work.
	manifest, loadErr := pluginstore.Load(pluginsDir)
	if loadErr == nil {
		if manifest.Plugins == nil {
			manifest.Plugins = make(map[string]*pluginstore.InstalledPlugin)
		}
		manifest.Plugins[entry.Name] = installed
		_ = pluginstore.Save(pluginsDir, manifest)
	}

	binaryPath, _, findErr := plugindispatch.FindPlugin(name, pluginsDir)
	if findErr != nil {
		return fmt.Errorf("plugin installed but binary not found: %w", findErr)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "Installed %s %s. Running: datumctl %s\n\n",
		entry.Name, installed.Version, strings.Join(append([]string{name}, originalArgs[1:]...), " "))

	// Re-exec the original command via the now-installed plugin.
	return plugindispatch.Exec(binaryPath, originalArgs[1:], factory)
}
