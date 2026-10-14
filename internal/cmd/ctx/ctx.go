package ctx

import (
	"context"

	"github.com/spf13/cobra"
)

// Command returns the "ctx" command group. Running "datumctl ctx" without a
// subcommand lists available contexts.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ctx",
		Short: "View and switch contexts",
		Long: `List and switch between organizations and projects.

Running 'datumctl ctx' without a subcommand lists the active session's
contexts. Use --all to list every session's contexts grouped by account and
endpoint. Contexts are always fetched live, so new organizations and projects
appear immediately.`,
		Aliases: []string{"context"},
		RunE:    runList,
	}

	cmd.Flags().Bool("refresh", false, "No-op; contexts are always fetched live")
	_ = cmd.Flags().MarkDeprecated("refresh", "contexts are always fetched live")
	cmd.Flags().Bool("all", false, "List contexts from every session, grouped by account and endpoint")

	cmd.AddCommand(listCmd())
	cmd.AddCommand(useCmd())

	return cmd
}

func cmdContext(cmd *cobra.Command) context.Context {
	if cmd != nil && cmd.Context() != nil {
		return cmd.Context()
	}
	return context.Background()
}
