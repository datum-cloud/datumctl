package get

import (
	"github.com/spf13/cobra"
)

// Command creates the base "get" command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(getOrganizationsCmd())

	return cmd
}
