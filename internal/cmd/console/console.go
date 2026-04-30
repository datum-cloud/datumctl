package console

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/console"
)

func Command(factory *client.DatumCloudFactory) *cobra.Command {
	var readOnly bool

	c := &cobra.Command{
		Use:     "console",

		Short:   "Open the interactive console for browsing Datum Cloud resources",
		Long: `Launch an interactive console for browsing and inspecting Datum Cloud resources.
Navigate resource types in the sidebar, view resources in the table, and inspect
details with 'd'. Press '?' for the full keybind reference.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return console.Run(cmd.Context(), factory, readOnly)
		},
	}

	c.Flags().BoolVar(&readOnly, "read-only", false, "Disable mutation operations (safe for automated testing)")
	return c
}
