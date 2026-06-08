package plugin

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/pluginstore"
)

func searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search [query]",
		Short: "Search for available datumctl plugins",
		Long: `Search the curated plugin index for available datumctl plugins.

The index is fetched from the datum-cloud/datumctl-plugins repository and
cached locally. An optional query filters results by name or description.

Run 'datumctl plugin install <name>' to install a plugin listed here.`,
		Example: `  # List all available plugins
  datumctl plugin search

  # Search for DNS-related plugins
  datumctl plugin search dns`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idx, err := pluginstore.LoadIndex()
			if err != nil || pluginstore.IsStale(idx) {
				idx, err = pluginstore.RefreshIndex(cmd.Context())
				if err != nil {
					if idx == nil {
						return indexFetchUserError(err)
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: index refresh failed (%v), showing cached results\n", err)
				}
			}

			query := ""
			if len(args) == 1 {
				query = strings.ToLower(args[0])
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
			for _, p := range idx.Plugins {
				if query != "" && !strings.Contains(p.Name, query) &&
					!strings.Contains(strings.ToLower(p.Spec.ShortDescription), query) {
					continue
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, p.Spec.Version, p.Spec.ShortDescription)
			}
			return w.Flush()
		},
	}
}
