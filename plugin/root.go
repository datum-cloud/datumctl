package plugin

import (
	"github.com/spf13/cobra"
)

// NewRootCmd returns a pre-configured *cobra.Command suitable for use as a plugin's root command.
// It wires --org, --project, and --output flags to the injected DATUM_* context values as defaults,
// so plugin authors do not need to declare these flags manually.
//
// name is the plugin name (e.g. "dns"); short is the one-line description shown in help.
func NewRootCmd(name, short string) *cobra.Command {
	ctx := Context()

	cmd := &cobra.Command{
		Use:   name,
		Short: short,
	}

	cmd.PersistentFlags().String("org", ctx.Org,
		"Datum Cloud organization (defaults to DATUM_ORG injected by datumctl)")
	cmd.PersistentFlags().String("project", ctx.Project,
		"Datum Cloud project (defaults to DATUM_PROJECT injected by datumctl)")
	cmd.PersistentFlags().StringP("output", "o", "table",
		"Output format. One of: table|json|yaml")

	return cmd
}
