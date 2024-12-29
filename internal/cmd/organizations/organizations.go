package organizations

import (
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "organizations",
	Short: "Manage organizations",
}

func init() {
	Command.AddCommand(
		listOrgsCommand(),
	)
}
