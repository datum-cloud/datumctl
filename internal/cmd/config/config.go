package config

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage datumctl contexts and clusters",
	}

	cmd.AddCommand(newGetContextsCmd())
	cmd.AddCommand(newUseContextCmd())
	cmd.AddCommand(newSetContextCmd())
	cmd.AddCommand(newSetClusterCmd())
	cmd.AddCommand(newViewCmd())

	return cmd
}

func requireArgs(cmd *cobra.Command, args []string, count int) (string, error) {
	if len(args) < count {
		return "", fmt.Errorf("requires %d argument(s)", count)
	}
	return args[0], nil
}
