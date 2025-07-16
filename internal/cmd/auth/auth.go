package auth

import "github.com/spf13/cobra"

// Command creates the base "auth" command and adds subcommands for login,
// logout, token retrieval, etc.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Datum Cloud",
	}

	cmd.AddCommand(
		getTokenCmd,
		LoginCmd,
		listCmd,
		logoutCmd,
		switchCmd,
		updateKubeconfigCmd(),
	)

	return cmd
}
