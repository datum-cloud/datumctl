package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/keyring"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List locally authenticated users",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runList()
	},
}

func runList() error {
	// Get the list of known user keys
	knownUsers := []string{}
	knownUsersJSON, err := keyring.Get(authutil.ServiceName, authutil.KnownUsersKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			// No users known yet
			fmt.Println("No users have been logged in yet.")
			return nil
		}
		// Other error getting the list
		return fmt.Errorf("failed to get known users list from keyring: %w", err)
	}

	if knownUsersJSON != "" {
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
			return fmt.Errorf("failed to unmarshal known users list: %w", err)
		}
	}

	if len(knownUsers) == 0 {
		fmt.Println("No users have been logged in yet.")
		return nil
	}

	// Get the active user key
	activeUserKey, err := keyring.Get(authutil.ServiceName, authutil.ActiveUserKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		// Don't fail if active user key is missing, just proceed without marking active
		fmt.Printf("Warning: could not determine active user: %v\n", err)
		activeUserKey = ""
	}

	// Initialize table
	tbl := table.New("Name", "Email", "Status").WithWriter(os.Stdout)

	for _, userKey := range knownUsers {
		// Retrieve the stored credentials for this user to get name/email
		credsJSON, err := keyring.Get(authutil.ServiceName, userKey)
		if err != nil {
			// Add row with error message if details retrieval fails
			tbl.AddRow("<Unknown>", userKey, fmt.Sprintf("Error: %v", err))
			continue
		}

		var creds authutil.StoredCredentials
		if err := json.Unmarshal([]byte(credsJSON), &creds); err != nil {
			// Add row with error message if unmarshal fails
			tbl.AddRow("<Unknown>", userKey, fmt.Sprintf("Error parsing: %v", err))
			continue
		}

		// Prepare display values
		displayName := creds.UserName
		if displayName == "" {
			displayName = "<N/A>"
		}
		displayEmail := creds.UserEmail
		if displayEmail == "" {
			displayEmail = "<N/A>"
		}
		status := ""
		if userKey == activeUserKey {
			status = "Active"
		}

		// Add row to table
		tbl.AddRow(displayName, displayEmail, status)
	}

	// Print the table
	tbl.Print()

	return nil
}
