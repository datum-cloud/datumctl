package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	customerrors "go.datum.net/datumctl/internal/errors"
)

// movedCommands maps subcommands that used to live under "auth" to their
// current top-level spelling. Login and logout were promoted to top-level
// commands in the context-discovery redesign (#149); users following older
// docs or muscle memory still reach for "datumctl auth login".
var movedCommands = map[string]string{
	"login":  "datumctl login",
	"logout": "datumctl logout",
}

// Command creates the base "auth" command and adds subcommands for session
// management, token retrieval, and kubectl integration.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Datum Cloud authentication credentials",
		Long: `The auth group provides commands to log in to Datum Cloud, manage multiple
user sessions, and retrieve tokens for scripting.

Typical workflow:
  1. Log in:          datumctl login
  2. Verify sessions: datumctl auth list
  3. Switch accounts: datumctl auth switch <email>
  4. Log out:         datumctl logout [email]

Advanced — kubectl integration:
  If you use kubectl and want to point it at a Datum Cloud control plane,
  see 'datumctl auth update-kubeconfig --help'.`,
		Example: `  # Log in to Datum Cloud
  datumctl login

  # Show all logged-in accounts
  datumctl auth list

  # Switch the active account
  datumctl auth switch user@example.com

  # Log out a specific account
  datumctl logout user@example.com`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return unknownSubcommandError(args[0])
		},
	}

	cmd.AddCommand(
		getTokenCmd,
		listCmd,
		switchCmd,
		updateKubeconfigCmd(),
	)

	return cmd
}

// unknownSubcommandError turns an unrecognized "datumctl auth <name>" into a
// hard error instead of a silently-successful help dump. Names that were moved
// to the top level get a hint pointing at the replacement command.
func unknownSubcommandError(name string) error {
	if replacement, moved := movedCommands[name]; moved {
		return &customerrors.UserError{
			Message: fmt.Sprintf("'datumctl auth %s' is now '%s'", name, replacement),
			Hint:    fmt.Sprintf("Login and logout are top-level commands — run '%s'.", replacement),
			Code:    "AUTH_COMMAND_MOVED",
		}
	}
	return &customerrors.UserError{
		Message: fmt.Sprintf("unknown command %q for 'datumctl auth'", name),
		Hint:    "Run 'datumctl auth --help' for available commands.",
		Code:    "UNKNOWN_SUBCOMMAND",
	}
}
