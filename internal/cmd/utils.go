package cmd

import (
	"github.com/spf13/cobra"
)

func WrapResourceCommand(cmd *cobra.Command) *cobra.Command {
	preRunFunc := func(cmd *cobra.Command, args []string) error {
		// if there are not args we let the underline command to deal with it.
		if len(args) == 0 {
			return nil
		}
		// This mapping helps user during the getting started phase
		if args[0] == "organizations" || args[0] == "organization" {
			args[0] = "organizationmemberships"
			cmd.Flag("all-namespaces").Value.Set("true")
		}
		return nil
	}
	cmd.PreRunE = preRunFunc
	cmd.GroupID = "resource"
	return cmd
}
