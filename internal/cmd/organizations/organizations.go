package organizations

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "organizations",
		Short: "Manage organizations",
	}

	cmd.AddCommand(
		listOrgsCommand(),
	)

	return cmd
}
