// Package inventory defines the `datumctl inventory` command tree — a
// purpose-built read view over the Datum Cloud physical inventory
// (providers, regions, sites, clusters, nodes).
package inventory

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
)

// Command returns the `datumctl inventory` parent command.
func Command(factory *client.DatumCloudFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Browse the Datum Cloud physical inventory",
		Long: templates.LongDesc(`
			Browse the Datum Cloud physical inventory: providers, regions, sites,
			clusters, and nodes.

			These records describe the real infrastructure Datum Cloud runs on —
			which provider owns a site, which region a site sits in, and which
			nodes are assigned to which cluster. Use the list subcommands to query
			one kind at a time, 'inventory tree' to see the region/site/node
			hierarchy, and 'inventory summary' for fleet-wide counts.

			Inventory lives on the platform root, so these commands default to
			--platform-wide. Pass --organization or --project to override.`),
		Example: templates.Examples(`
			# List every region
			datumctl inventory regions

			# Sites in one region, by provider
			datumctl inventory sites --region us-central-2
			datumctl inventory sites --provider netactuate

			# Nodes at a site or in a cluster
			datumctl inventory nodes --site us-central-2a
			datumctl inventory nodes --cluster my-edge-cluster

			# Region -> site -> node hierarchy
			datumctl inventory tree

			# Fleet-wide counts
			datumctl inventory summary`),
	}

	cmd.PersistentFlags().StringP("output", "o", "table", "Output format. One of: table, json, yaml.")

	cmd.AddCommand(
		newListCmd(factory, providersView),
		newListCmd(factory, regionsView),
		newListCmd(factory, sitesView),
		newListCmd(factory, clustersView),
		newListCmd(factory, nodesView),
		newTreeCmd(factory),
		newSummaryCmd(factory),
	)

	return cmd
}
