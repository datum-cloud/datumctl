package inventory

import (
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
)

func newTreeCmd(factory *client.DatumCloudFactory) *cobra.Command {
	var regionFilter string
	cmd := &cobra.Command{
		Use:   "tree",
		Short: "Show the region -> site -> node hierarchy",
		Long: templates.LongDesc(`
			Print the inventory as a topology tree: each region, the sites within
			it, the nodes at each site, and the clusters anchored in the region.

			Use --region to scope the tree to a single region.`),
		Example: templates.Examples(`
			# Full topology tree
			datumctl inventory tree

			# Just one region
			datumctl inventory tree --region us-central-2`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			applyInventoryScope(cmd, factory)
			ctx := cmd.Context()

			regions, err := listResources(ctx, factory, "regions", "")
			if err != nil {
				return err
			}
			sites, err := listResources(ctx, factory, "sites", "")
			if err != nil {
				return err
			}
			clusters, err := listResources(ctx, factory, "clusters", "")
			if err != nil {
				return err
			}
			nodes, err := listResources(ctx, factory, "nodes", "")
			if err != nil {
				return err
			}

			printTree(cmd.OutOrStdout(), regionFilter, regions, sites, clusters, nodes)
			return nil
		},
	}
	cmd.Flags().StringVar(&regionFilter, "region", "", "Limit the tree to a single region")
	return cmd
}

func printTree(w io.Writer, regionFilter string, regions, sites, clusters, nodes *unstructured.UnstructuredList) {
	sitesByRegion := groupBy(sites.Items, func(u unstructured.Unstructured) string { return str(u, "spec", "regionRef", "name") })
	nodesBySite := groupBy(nodes.Items, func(u unstructured.Unstructured) string { return str(u, "spec", "siteRef", "name") })
	clustersByRegion := groupBy(clusters.Items, func(u unstructured.Unstructured) string {
		if r := u.GetLabels()[labelRegion]; r != "" {
			return r
		}
		return none
	})

	names := make([]string, 0, len(regions.Items))
	for _, r := range regions.Items {
		names = append(names, r.GetName())
	}
	sort.Strings(names)

	printed := 0
	for _, region := range names {
		if regionFilter != "" && region != regionFilter {
			continue
		}
		printed++
		fmt.Fprintf(w, "%s\n", region)

		if cls := clustersByRegion[region]; len(cls) > 0 {
			sort.Strings(cls)
			fmt.Fprintf(w, "  clusters: %s\n", join(cls))
		}

		regionSites := sitesByRegion[region]
		sort.Strings(regionSites)
		for _, site := range regionSites {
			fmt.Fprintf(w, "  %s\n", site)
			siteNodes := nodesBySite[site]
			sort.Strings(siteNodes)
			for _, n := range siteNodes {
				fmt.Fprintf(w, "    %s\n", n)
			}
		}
	}

	if printed == 0 {
		fmt.Fprintln(w, "No matching inventory found.")
	}
}

func groupBy(items []unstructured.Unstructured, key func(u unstructured.Unstructured) string) map[string][]string {
	out := map[string][]string{}
	for _, item := range items {
		k := key(item)
		out[k] = append(out[k], item.GetName())
	}
	return out
}

func join(items []string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}
