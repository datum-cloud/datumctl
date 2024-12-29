package auth

import "github.com/spf13/cobra"

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Datum Cloud",
	}

	cmd.AddCommand(
		activateAPITokenCmd(),
		getTokenCmd(),
		logoutCmd(),
		updateKubeconfigCmd(),
	)

	return cmd
}
