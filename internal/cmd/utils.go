package cmd

import (
	"strings"

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

	// Wrap the existing ValidArgsFunction so the "organizations" alias also
	// appears as a completion option alongside the real resource types.
	if inner := cmd.ValidArgsFunction; inner != nil {
		cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			comps, directive := inner(cmd, args, toComplete)
			if len(args) == 0 && strings.HasPrefix("organizations", toComplete) {
				comps = append(comps, "organizations")
			}
			return comps, directive
		}
	}

	return cmd
}
