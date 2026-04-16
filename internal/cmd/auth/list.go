package auth

import (
	"fmt"
	"os"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List locally authenticated users",
	Long: `Display a table of all Datum Cloud users whose sessions are stored
locally, along with their status.

Columns:
  User       The email address used to log in. Pass this to 'datumctl auth switch'
             or 'datumctl logout' to act on a specific account.
  Endpoint   Shown only when sessions span more than one API endpoint.
  Status     "Active" marks the account whose credentials are used by default
             for all subsequent datumctl commands.`,
	Example: `  # Show all logged-in users
  datumctl auth list

  # Alias
  datumctl auth ls`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	if len(cfg.Sessions) == 0 {
		fmt.Println("No authenticated users. Run 'datumctl login' to get started.")
		return nil
	}

	showEndpoint := cfg.HasMultipleEndpoints()

	var tbl table.Table
	if showEndpoint {
		tbl = table.New("User", "Endpoint", "Status")
	} else {
		tbl = table.New("User", "Status")
	}
	tbl.WithWriter(os.Stdout)

	for _, s := range cfg.Sessions {
		status := ""
		if s.Name == cfg.ActiveSession {
			status = "Active"
		}
		if showEndpoint {
			tbl.AddRow(s.UserEmail, datumconfig.StripScheme(s.Endpoint.Server), status)
		} else {
			tbl.AddRow(s.UserEmail, status)
		}
	}
	tbl.Print()
	return nil
}
