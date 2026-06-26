// Package plugin implements the 'datumctl plugin' subcommand group for
// installing, listing, upgrading, removing, and trusting datumctl plugins.
package plugin

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/pluginstore"
)

// Command returns the root 'plugin' command with all subcommands registered.
func Command(factory *client.DatumCloudFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage datumctl plugins",
		Long: `Manage datumctl plugins — install, list, upgrade, remove, and trust plugins.

Plugins are independent binaries named 'datumctl-<name>' that extend the CLI
with additional commands. Run them as 'datumctl <name>'.

Managed plugins are installed via 'datumctl plugin install' and recorded in
plugins.json (~/.datumctl/plugins/plugins.json by default).
Datumctl automatically verifies their SHA256 at install time.

Unmanaged plugins (binaries found on PATH but not installed via this command)
are blocked from running until explicitly trusted. Use 'datumctl plugin trust'
to allow an unmanaged plugin, or 'datumctl plugin install' to manage it.`,
		Example: `  # Install the DNS plugin
  datumctl plugin install datum-cloud/datumctl-dns

  # List all installed plugins
  datumctl plugin list

  # Upgrade the dns plugin
  datumctl plugin upgrade dns

  # Remove the dns plugin
  datumctl plugin remove dns

  # Trust an unmanaged plugin on PATH
  datumctl plugin trust dns`,
	}

	cmd.PersistentFlags().String("plugins-dir", "",
		"Override the managed plugins directory (default: ~/.datumctl/plugins/)")

	cmd.AddCommand(
		installCmd(factory),
		listCmd(),
		searchCmd(),
		browseCmd(),
		indexCmd(),
		upgradeCmd(factory),
		removeCmd(),
		trustCmd(),
		untrustCmd(),
	)
	return cmd
}

// installedPluginNames returns completion candidates from plugins.json.
func installedPluginNames(cmd *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	pluginsDir, err := resolvePluginsDir(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(manifest.Plugins))
	for name := range manifest.Plugins {
		names = append(names, name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// resolvePluginsDir reads --plugins-dir from the nearest ancestor command that
// has a "plugins-dir" persistent flag, then falls back to pluginstore.PluginsDir.
func resolvePluginsDir(cmd *cobra.Command) (string, error) {
	override := ""
	// Walk up the command tree to find the persistent flag.
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.PersistentFlags().Lookup("plugins-dir"); f != nil {
			override = f.Value.String()
			break
		}
	}
	dir, err := pluginstore.PluginsDir(override)
	if err != nil {
		return "", fmt.Errorf("resolve plugins directory: %w", err)
	}
	return dir, nil
}
