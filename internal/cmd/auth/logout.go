package auth

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/keyring"
)

var logoutAll bool // Flag variable for --all

// logoutCmd removes local authentication credentials for a specified user or all users.
var logoutCmd = &cobra.Command{
	Use:   "logout [user]",
	Short: "Remove local authentication credentials for a specified user or all users",
	Long: `Remove local authentication credentials.

Specify a user in the format 'email@hostname' to log out only that user.
Use 'datumctl auth list' to see available users.
Use the --all flag to log out all known users.`, // Updated Long description
	Args: func(cmd *cobra.Command, args []string) error {
		// Custom args validation
		all, _ := cmd.Flags().GetBool("all")
		if all && len(args) > 0 {
			return errors.New("cannot specify a user argument when using the --all flag")
		}
		if !all && len(args) != 1 {
			return errors.New("must specify exactly one user (email@hostname) or use the --all flag")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if logoutAll {
			return logoutAllUsers()
		} else {
			// Args validation ensures len(args) == 1 here
			userKeyToLogout := args[0]
			return logoutSingleUser(userKeyToLogout) // Renamed function
		}
	},
}

func init() {
	// Add the --all flag
	logoutCmd.Flags().BoolVar(&logoutAll, "all", false, "Log out all authenticated users")
}

// logoutSingleUser handles logging out a specific user (previously runLogout)
func logoutSingleUser(userKeyToLogout string) error {
	fmt.Printf("Logging out user: %s\n", userKeyToLogout)

	// 1. Get known users list
	knownUsers := []string{}
	knownUsersJSON, err := keyring.Get(authutil.ServiceName, authutil.KnownUsersKey)

	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("failed to get known users list from keyring: %w", err)
	}

	if err == nil && knownUsersJSON != "" {
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
			return fmt.Errorf("failed to unmarshal known users list: %w", err)
		}
	}

	// 2. Check if the user exists in the list and prepare updated list
	found := false
	updatedKnownUsers := []string{}
	for _, key := range knownUsers {
		if key == userKeyToLogout {
			found = true
		} else {
			updatedKnownUsers = append(updatedKnownUsers, key)
		}
	}

	if !found {
		fmt.Printf("User '%s' not found in locally stored credentials.\n", userKeyToLogout)
		if err := keyring.Delete(authutil.ServiceName, userKeyToLogout); err != nil && !errors.Is(err, keyring.ErrNotFound) {
			fmt.Printf("Warning: attempt to delete potential stray key for %s failed: %v\n", userKeyToLogout, err)
		}
		return nil
	}

	// 3. Delete the user's specific credential entry
	err = keyring.Delete(authutil.ServiceName, userKeyToLogout)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		fmt.Printf("Warning: failed to delete credentials for user '%s' from keyring: %v\n", userKeyToLogout, err)
	}

	// 4. Update and save the known users list
	updatedJSON, err := json.Marshal(updatedKnownUsers)
	if err != nil {
		return fmt.Errorf("failed to marshal updated known users list: %w", err)
	}

	err = keyring.Set(authutil.ServiceName, authutil.KnownUsersKey, string(updatedJSON))
	if err != nil {
		return fmt.Errorf("failed to store updated known users list: %w", err)
	}

	// 5. Check if the logged-out user was the active user
	activeUserKey, err := keyring.Get(authutil.ServiceName, authutil.ActiveUserKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		fmt.Printf("Warning: could not determine active user: %v\n", err)
	}

	if activeUserKey == userKeyToLogout {
		fmt.Println("Logging out the active user. Clearing active user setting.")
		err = keyring.Delete(authutil.ServiceName, authutil.ActiveUserKey)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			fmt.Printf("Warning: failed to clear active user setting from keyring: %v\n", err)
		}
	}

	fmt.Printf("Successfully logged out user '%s'.\n", userKeyToLogout)
	return nil
}

// logoutAllUsers handles logging out all known users
func logoutAllUsers() error {
	fmt.Println("Logging out all users...")

	// 1. Get known users list
	knownUsers := []string{}
	knownUsersJSON, err := keyring.Get(authutil.ServiceName, authutil.KnownUsersKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			fmt.Println("No users found in keyring to log out.")
			return nil // Nothing to do
		}
		return fmt.Errorf("failed to get known users list from keyring: %w", err)
	}

	if knownUsersJSON != "" {
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
			return fmt.Errorf("failed to unmarshal known users list: %w", err)
		}
	}

	if len(knownUsers) == 0 {
		fmt.Println("No users found in keyring to log out.")
		return nil // Nothing to do
	}

	// 2. Delete each user's specific credential entry
	fmt.Printf("Found %d user(s) to log out.\n", len(knownUsers))
	logoutErrors := false
	for _, userKey := range knownUsers {
		err = keyring.Delete(authutil.ServiceName, userKey)
		if err != nil && !errors.Is(err, keyring.ErrNotFound) {
			fmt.Printf("Warning: failed to delete credentials for user '%s' from keyring: %v\n", userKey, err)
			logoutErrors = true // Mark that at least one error occurred
		}
	}

	// 3. Delete the known users list itself
	err = keyring.Delete(authutil.ServiceName, authutil.KnownUsersKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		fmt.Printf("Warning: failed to delete known users list from keyring: %v\n", err)
		logoutErrors = true
	}

	// 4. Delete the active user setting
	err = keyring.Delete(authutil.ServiceName, authutil.ActiveUserKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		fmt.Printf("Warning: failed to delete active user setting from keyring: %v\n", err)
		logoutErrors = true
	}

	if logoutErrors {
		fmt.Println("Completed logout for all users, but encountered some errors (see warnings above).")
	} else {
		fmt.Println("Successfully logged out all users.")
	}

	return nil
}
