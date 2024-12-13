package auth

import "github.com/spf13/cobra"

var Command = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Datum Cloud",
}

func init() {
	Command.AddCommand(
		getTokenCmd(),
		updateKubeconfigCmd(),
	)
}
