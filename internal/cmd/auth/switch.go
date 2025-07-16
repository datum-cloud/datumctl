package auth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/keyring"
)

var switchCmd = &cobra.Command{
	Use:   "switch <user-email>",
	Short: "Set the active authenticated user session",
	Long: `Switches the active user context to the specified user email.

The user email must correspond to an existing set of credentials previously
established via 'datumctl auth login'. Use 'datumctl auth list' to see available users.`,
	Args: cobra.ExactArgs(1), // Requires exactly one argument: the user email
	RunE: func(cmd *cobra.Command, args []string) error {
		targetUserKey := args[0]
		return runSwitch(targetUserKey)
	},
}

func runSwitch(targetUserKey string) error {
	// 1. Get the list of known users to validate the target user exists
	knownUsers := []string{}
	knownUsersJSON, err := keyring.Get(authutil.ServiceName, authutil.KnownUsersKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		// Don't fail if list is missing, but we won't be able to validate.
		// Print a warning?
		fmt.Printf("Warning: could not retrieve known users list to validate target: %v\n", err)
	} else if knownUsersJSON != "" {
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
			// Also don't fail, but warn.
			fmt.Printf("Warning: could not parse known users list to validate target: %v\n", err)
		}
	}

	// 2. Validate the target user key exists in the known list (if available)
	found := false
	if len(knownUsers) > 0 {
		for _, key := range knownUsers {
			if key == targetUserKey {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("user '%s' not found in the list of locally authenticated users. Use 'datumctl auth list' to see available users", targetUserKey)
		}
	} else {
		// If known users list wasn't available or parseable, try to get the specific credential as a fallback validation
		_, err := keyring.Get(authutil.ServiceName, targetUserKey)
		if err != nil {
			if errors.Is(err, keyring.ErrNotFound) {
				return fmt.Errorf("credentials for user '%s' not found. Use 'datumctl auth list' to see available users", targetUserKey)
			}
			return fmt.Errorf("failed to check credentials for user '%s': %w", targetUserKey, err)
		}
	}

	// 3. Get current active user (optional, for comparison message)
	currentActiveUser, _ := keyring.Get(authutil.ServiceName, authutil.ActiveUserKey)

	if currentActiveUser == targetUserKey {
		fmt.Printf("User '%s' is already the active user.\n", targetUserKey)
		return nil
	}

	// 4. Set the new active user
	err = keyring.Set(authutil.ServiceName, authutil.ActiveUserKey, targetUserKey)
	if err != nil {
		return fmt.Errorf("failed to set '%s' as active user in keyring: %w", targetUserKey, err)
	}

	fmt.Printf("Switched active user to '%s'\n", targetUserKey)
	return nil
}
