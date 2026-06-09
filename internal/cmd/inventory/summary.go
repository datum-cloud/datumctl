package inventory

import (
	"fmt"
	"io"
	"sort"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
)

func newSummaryCmd(factory *client.DatumCloudFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Show fleet-wide inventory counts",
		Long: templates.LongDesc(`
			Print fleet-wide counts: totals per kind, sites and nodes per region,
			and nodes per provider.`),
		Example: templates.Examples(`
			# Fleet-wide counts
			datumctl inventory summary`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			applyInventoryScope(cmd, factory)
			ctx := cmd.Context()

			counts := map[string]int{}
			lists := map[string]*unstructured.UnstructuredList{}
			for _, kind := range []string{"providers", "regions", "sites", "clusters", "nodes"} {
				l, err := listResources(ctx, factory, kind, "")
				if err != nil {
					return err
				}
				lists[kind] = l
				counts[kind] = len(l.Items)
			}

			printSummary(cmd.OutOrStdout(), counts, lists)
			return nil
		},
	}
	return cmd
}

func printSummary(w io.Writer, counts map[string]int, lists map[string]*unstructured.UnstructuredList) {
	fmt.Fprintln(w, "Totals")
	totals := table.New("KIND", "COUNT")
	totals.WithWriter(w)
	for _, kind := range []string{"providers", "regions", "sites", "clusters", "nodes"} {
		totals.AddRow(kind, counts[kind])
	}
	totals.Print()

	sitesPerRegion := tally(lists["sites"].Items, func(u unstructured.Unstructured) string { return str(u, "spec", "regionRef", "name") })
	nodesPerRegion := tally(lists["nodes"].Items, func(u unstructured.Unstructured) string {
		if r := u.GetLabels()[labelRegion]; r != "" {
			return r
		}
		return none
	})
	fmt.Fprintln(w, "\nPer region")
	perRegion := table.New("REGION", "SITES", "NODES")
	perRegion.WithWriter(w)
	for _, region := range union(sitesPerRegion, nodesPerRegion) {
		perRegion.AddRow(region, sitesPerRegion[region], nodesPerRegion[region])
	}
	perRegion.Print()

	sitesPerProvider := tally(lists["sites"].Items, func(u unstructured.Unstructured) string { return str(u, "spec", "providerRef", "name") })
	fmt.Fprintln(w, "\nSites per provider")
	perProvider := table.New("PROVIDER", "SITES")
	perProvider.WithWriter(w)
	for _, provider := range sortedKeys(sitesPerProvider) {
		perProvider.AddRow(provider, sitesPerProvider[provider])
	}
	perProvider.Print()
}

func tally(items []unstructured.Unstructured, key func(u unstructured.Unstructured) string) map[string]int {
	out := map[string]int{}
	for _, item := range items {
		out[key(item)]++
	}
	return out
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func union(a, b map[string]int) []string {
	seen := map[string]bool{}
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
