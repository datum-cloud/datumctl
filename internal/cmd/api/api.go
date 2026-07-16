// Package api implements the 'datumctl api' command group: commands for
// working with the Datum Cloud API directly over HTTP.
package api

import (
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/client"
)

// Command returns the parent 'api' command group.
func Command(factory *client.DatumCloudFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Work with the Datum Cloud API directly",
		Long: templates.LongDesc(`
			Commands for working with the Datum Cloud API directly over HTTP,
			such as running a local authenticated proxy for your own tools.`),
	}
	cmd.AddCommand(proxyCommand(factory))
	return cmd
}
