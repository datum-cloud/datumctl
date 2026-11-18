package plugin

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

func listCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed datumctl plugins",
		Long: templates.LongDesc(`
			List the plugins you have installed, without running any of them.

			Status column indicators:
			  ok          Plugin is installed and compatible with this datumctl.
			  update      A newer version is available in its catalog.
			  deprecated  Its catalog marks the plugin deprecated.
			  !           Built for a different datumctl version.
			  ?           Version info unavailable.

			Run 'datumctl plugin search' to refresh available plugins from your
			catalogs.`),
		Example: templates.Examples(`
			# List all installed plugins
			datumctl plugin list`),
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

			// Load cached catalog indexes read-only — no network calls. Memoized
			// per catalog so each is read at most once.
			indexCache := map[string]*pluginstore.CachedIndex{}
			catalogIndex := func(catalogName string) *pluginstore.CachedIndex {
				if idx, ok := indexCache[catalogName]; ok {
					return idx
				}
				idx, _ := pluginstore.LoadCatalogIndex(pluginsDir, catalogName)
				indexCache[catalogName] = idx
				return idx
			}

			// Cached indexes only; the registry is a local file. A plugin whose
			// catalog index isn't cached is simply not marked deprecated.
			reg, _ := pluginstore.LoadRegistry(pluginsDir)
			deprecations := map[string]*pluginstore.Deprecation{}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tINDEX\tVERSION\tTRUST\tDESCRIPTION\tSTATUS")
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
				indexLabel, trust := installedCatalogLabel(entry)
				// Catalog lookup only applies to catalog-sourced plugins (the
				// "(direct)" label marks a direct GitHub install with no catalog).
				// The deprecation notice is recorded whatever the status; the
				// STATUS column keeps "!" and "?" ahead of "deprecated".
				if indexLabel != "(direct)" {
					if indexEntry := pluginstore.FindInIndex(catalogIndex(indexLabel), name); indexEntry != nil {
						if indexEntry.Spec.Deprecated != nil {
							deprecations[name] = indexEntry.Spec.Deprecated
						}
						if status == "ok" {
							var update bool
							status, update = listStatus(entry.Version, indexEntry)
							anyUpdates = anyUpdates || update
						}
					}
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", name, indexLabel, entry.Version, trust, description, status)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			names := make([]string, 0, len(deprecations))
			for name := range deprecations {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				writeDeprecationNotice(cmd.ErrOrStderr(), reg, name, deprecations[name])
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

// listStatus returns the STATUS column value for a compatible, catalog-sourced
// plugin, and whether a newer catalog version exists. Deprecated takes
// precedence in the column, but the update flag is still reported because
// 'plugin upgrade' upgrades installed deprecated plugins.
func listStatus(installedVersion string, indexEntry *pluginstore.Plugin) (status string, updateAvailable bool) {
	updateAvailable = isUpdateAvailable(installedVersion, indexEntry.Spec.Version)
	switch {
	case indexEntry.Spec.Deprecated != nil:
		return "deprecated", updateAvailable
	case updateAvailable:
		return "update", true
	}
	return "ok", false
}
