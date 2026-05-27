package plugin

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed datumctl plugins",
		Long: `List all managed plugins recorded in plugins.json.

Reads plugins.json only — never execs plugin binaries.

Status column indicators:
  ok      Plugin is installed and API version matches.
  update  A newer version is available in the plugin index.
  !       Stored api_version does not match the host's API version.
  ?       No manifest recorded (plugin did not respond to --plugin-manifest).

Run 'datumctl plugin search' to refresh the plugin index.`,
		Example: `  # List all installed plugins
  datumctl plugin list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}

			manifest, err := pluginstore.Load(pluginsDir)
			if err != nil {
				return fmt.Errorf("load plugins manifest: %w", err)
			}

			if len(manifest.Plugins) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No managed plugins installed.")
				return nil
			}

			// Load the cached index read-only — no network calls.
			idx, _ := pluginstore.LoadIndex()

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION\tSTATUS")
			var anyUpdates bool
			for name, entry := range manifest.Plugins {
				description := ""
				status := "?"
				if entry.Manifest != nil {
					description = entry.Manifest.Description
					if entry.Manifest.APIVersion == plugindispatch.PluginAPIVersion {
						status = "ok"
					} else {
						status = "!"
					}
				}
				if status == "ok" {
					if indexEntry := pluginstore.FindInIndex(idx, name); indexEntry != nil {
						if isUpdateAvailable(entry.Version, indexEntry.Spec.Version) {
							status = "update"
							anyUpdates = true
						}
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, entry.Version, description, status)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if anyUpdates && term.IsTerminal(int(os.Stdout.Fd())) {
				fmt.Fprintln(cmd.ErrOrStderr(),
					"\nRun 'datumctl plugin upgrade' to update plugins with available upgrades.")
			}
			return nil
		},
	}
	return cmd
}
