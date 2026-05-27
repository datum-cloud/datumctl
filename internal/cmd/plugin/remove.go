package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed plugin",
		Long: `Remove a managed plugin binary and its plugins.json entry.

The binary is deleted from the managed plugins directory. The plugin entry
is removed from plugins.json.`,
		Example: `  # Remove the dns plugin
  datumctl plugin remove dns`,
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

			if _, ok := manifest.Plugins[name]; !ok {
				return customerrors.NewUserError(fmt.Sprintf("plugin %q is not installed", name))
			}

			// Remove the binary.
			binaryName := "datumctl-" + name
			if runtime.GOOS == "windows" {
				binaryName += ".exe"
			}
			binaryPath := filepath.Join(pluginsDir, binaryName)
			if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove plugin binary: %w", err)
			}

			// Remove from manifest.
			delete(manifest.Plugins, name)
			if err := pluginstore.Save(pluginsDir, manifest); err != nil {
				return fmt.Errorf("save plugins manifest: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", name)
			return nil
		},
	}
	cmd.ValidArgsFunction = installedPluginNames
	return cmd
}
