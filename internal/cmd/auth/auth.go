package auth

import (
	"github.com/spf13/cobra"
)

// Command creates the base "auth" command and adds subcommands for login,
// logout, token retrieval, etc.
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
  datumctl auth login

  # Show all logged-in accounts
  datumctl auth list

  # Switch the active account
  datumctl auth switch user@example.com

  # Log out a specific account
  datumctl auth logout user@example.com`,
	}

	cmd.AddCommand(
		getTokenCmd,
		listCmd,
		switchCmd,
		updateKubeconfigCmd(),
	)

	return cmd
}
