package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

// searchResult is one plugin in 'datumctl plugin search' output. Its JSON form
// is what '-o json' prints, so scripts and agents can rely on these field names.
type searchResult struct {
	Name        string `json:"name"`
	Index       string `json:"index"`
	Version     string `json:"version"`
	Trust       string `json:"trust"`
	Description string `json:"description"`
}

const (
	searchOutputTable = "table"
	searchOutputJSON  = "json"
	searchOutputName  = "name"
)

func searchCmd() *cobra.Command {
	var indexName, output string
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for available datumctl plugins",
		Long: templates.LongDesc(`
			Search every registered plugin catalog for available datumctl plugins.

			Results span the official datum catalog and any catalogs you have added,
			with a column showing which catalog each plugin came from and a trust badge
			("official" for Datum's curated datum catalog, "third-party" for catalogs
			you added). An optional query filters by name or description; use --index to
			scope to one catalog.

			Run 'datumctl plugin install <name>' to install a plugin listed here.

			Use -o json for machine-readable output, or -o name for one plugin name
			per line.`),
		Example: templates.Examples(`
			# List all available plugins across catalogs
			datumctl plugin search

			# Search for DNS-related plugins
			datumctl plugin search dns

			# Scope the search to one catalog
			datumctl plugin search dns --index acme

			# Machine-readable output
			datumctl plugin search --index datum -o json`),
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

			warnDisabledCatalogs(cmd.ErrOrStderr(), reg)
			catalogs := reg.Active()
			if indexName != "" {
				cat := reg.Find(indexName)
				if cat == nil {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is not registered", indexName))
				}
				if cat.Disabled {
					return customerrors.NewUserError(fmt.Sprintf("catalog %q is disabled: %s", indexName, cat.DisabledReason))
				}
				catalogs = []pluginstore.Catalog{*cat}
			}

			query := ""
			if len(args) == 1 {
				query = strings.ToLower(args[0])
			}

			switch output {
			case searchOutputTable, searchOutputJSON, searchOutputName:
			default:
				return customerrors.NewUserErrorWithHint(
					fmt.Sprintf("invalid value %q for --output", output),
					"Allowed values: table, json, name.",
				)
			}

			results := []searchResult{}
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
					results = append(results, searchResult{
						Name:        p.Name,
						Index:       cat.Name,
						Version:     p.Spec.Version,
						Trust:       cat.Trust(),
						Description: p.Spec.ShortDescription,
					})
				}
			}
			return printSearchResults(cmd.OutOrStdout(), output, results)
		},
	}
	cmd.Flags().StringVar(&indexName, "index", "", "Scope the search to a single catalog")
	cmd.Flags().StringVarP(&output, "output", "o", searchOutputTable, "Output format: table, json or name")
	return cmd
}

// printSearchResults writes search results in the requested format. JSON
// output is always an array, empty when nothing matched.
func printSearchResults(w io.Writer, output string, results []searchResult) error {
	switch output {
	case searchOutputJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	case searchOutputName:
		for _, r := range results {
			fmt.Fprintln(w, r.Name)
		}
		return nil
	}

	if len(results) == 0 {
		fmt.Fprintln(w, "No matching plugins found.")
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tINDEX\tVERSION\tTRUST\tDESCRIPTION")
	for _, r := range results {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", r.Name, r.Index, r.Version, r.Trust, r.Description)
	}
	return tw.Flush()
}
