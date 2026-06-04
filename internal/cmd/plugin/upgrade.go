package plugin

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	componentversion "k8s.io/component-base/version"

	"go.datum.net/datumctl/internal/client"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

func upgradeCmd(factory *client.DatumCloudFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade <name>",
		Short: "Upgrade an installed plugin to the latest version",
		Long: `Download and install the latest release of a managed plugin.

The plugin must already be recorded in plugins.json. The same install flow
runs as 'datumctl plugin install': SHA256 verification, manifest check, and
compatibility validation.`,
		Example: `  # Upgrade the dns plugin to the latest version
  datumctl plugin upgrade dns`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			manifest, err := pluginstore.Load(pluginsDir)
			if err != nil {
				return fmt.Errorf("load plugins manifest: %w", err)
			}

			entry, ok := manifest.Plugins[name]
			if !ok {
				return customerrors.NewUserError(fmt.Sprintf("plugin %q is not installed; run 'datumctl plugin install' first", name))
			}

			currentVersion := componentversion.Get().GitVersion

			// Load/refresh the index — three-case handling per design.
			idx, refreshErr := pluginstore.RefreshIndex(cmd.Context())
			switch {
			case refreshErr == nil:
				// Success — proceed normally.
			case idx != nil:
				// Degraded: stale cache available.
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: index refresh failed (%v), using cached index\n", refreshErr)
			default:
				// No cache at all.
				return indexFetchUserError(refreshErr)
			}

			var newEntry *pluginstore.InstalledPlugin
			var binaryPath string

			// Third-party source (owner/repo format) — use GitHub release flow.
			if strings.Contains(entry.Source, "/") {
				owner, repo, _, parseErr := parseSource(entry.Source)
				if parseErr != nil {
					return customerrors.NewUserError(fmt.Sprintf("invalid source for plugin %q: %v", name, parseErr))
				}
				// Install latest (no pinned version → fetchLatestTag).
				newEntry, _, binaryPath, err = installPluginFromGitHub(cmd.Context(), pluginsDir, owner, repo, "", currentVersion)
				if err != nil {
					return customerrors.NewUserError(fmt.Sprintf("upgrade plugin %s: %v", name, err))
				}
			} else {
				// Curated index source.
				newEntry, _, binaryPath, err = installPlugin(cmd.Context(), pluginsDir, name, "", currentVersion, idx)
				if err != nil {
					return customerrors.NewUserError(fmt.Sprintf("upgrade plugin %s: %v", name, err))
				}
			}

			manifest.Plugins[name] = newEntry
			if saveErr := pluginstore.Save(pluginsDir, manifest); saveErr != nil {
				// L1: binary is on disk but unrecorded — attempt cleanup.
				if binaryPath != "" {
					if removeErr := os.Remove(binaryPath); removeErr != nil && !os.IsNotExist(removeErr) {
						return fmt.Errorf("save plugins manifest: %w; also failed to remove orphaned binary %s: %v", saveErr, binaryPath, removeErr)
					}
				}
				return fmt.Errorf("save plugins manifest: %w", saveErr)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Upgraded %s to %s\n", name, newEntry.Version)
			return nil
		},
	}
	cmd.ValidArgsFunction = installedPluginNames
	return cmd
}
