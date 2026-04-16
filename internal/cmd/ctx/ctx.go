package ctx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
)

// Command returns the "ctx" command group. Running "datumctl ctx" without a
// subcommand lists available contexts.
func Command() *cobra.Command {
	var refresh bool

	cmd := &cobra.Command{
		Use:   "ctx",
		Short: "View and switch contexts",
		Long: `List and switch between organizations and projects.

Running 'datumctl ctx' without a subcommand lists all available contexts
for the current user. Use --refresh to update the context cache from the API.`,
		Aliases: []string{"context"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if refresh {
				if err := runRefresh(cmd); err != nil {
					return err
				}
			} else {
				cfg, err := datumconfig.LoadAuto()
				if err == nil && discovery.IsCacheStale(cfg, discovery.DefaultStaleness) {
					fmt.Fprintln(os.Stderr, "Hint: context cache may be stale. Run 'datumctl ctx --refresh' to update.")
				}
			}
			return runList(cmd, args)
		},
	}

	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh the context cache from the API before listing")

	cmd.AddCommand(listCmd())
	cmd.AddCommand(useCmd())

	return cmd
}
