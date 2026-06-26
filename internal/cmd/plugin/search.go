package plugin

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

func searchCmd() *cobra.Command {
	var indexName string
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for available datumctl plugins",
		Long: `Search every registered plugin catalog for available datumctl plugins.

Results span the official datum catalog and any catalogs you have added, with a
column showing which catalog each plugin came from and a trust badge ("official"
for Datum's curated datum catalog, "third-party" for catalogs you added). An
optional query filters by name or description; use --index to scope to one catalog.

Run 'datumctl plugin install <name>' to install a plugin listed here.`,
		Example: `  # List all available plugins across catalogs
  datumctl plugin search

  # Search for DNS-related plugins
  datumctl plugin search dns

  # Scope the search to one catalog
  datumctl plugin search dns --index acme`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pluginsDir, err := resolvePluginsDir(cmd)
			if err != nil {
				return err
			}
			reg, err := pluginstore.LoadRegistry(pluginsDir)
			if err != nil {
				return fmt.Errorf("load catalog registry: %w", err)
			}

			catalogs := reg.Catalogs
			if indexName != "" {
				cat := reg.Find(indexName)
				if cat == nil {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered", indexName))
				}
				catalogs = []pluginstore.Catalog{*cat}
			}

			query := ""
			if len(args) == 1 {
				query = strings.ToLower(args[0])
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tINDEX\tVERSION\tTRUST\tDESCRIPTION")
			var rows int
			for i := range catalogs {
				cat := catalogs[i]
				idx, idxErr := loadOrRefreshCatalog(cmd, pluginsDir, cat)
				if idxErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping catalog %q: %v\n", cat.Name, idxErr)
					continue
				}
				for j := range idx.Plugins {
					p := &idx.Plugins[j]
					if query != "" && !strings.Contains(p.Name, query) &&
						!strings.Contains(strings.ToLower(p.Spec.ShortDescription), query) {
						continue
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
						p.Name, cat.Name, p.Spec.Version, cat.Trust(), p.Spec.ShortDescription)
					rows++
				}
			}
			if err := w.Flush(); err != nil {
				return err
			}
			if rows == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No matching plugins found.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&indexName, "index", "", "Scope the search to a single catalog")
	return cmd
}
