package logout

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
)

// Command returns the top-level "logout" command.
func Command() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "logout [email]",
		Short: "Remove local authentication credentials",
		Long: `Remove local authentication credentials.

Specify an email to log out sessions for that user.
Use --all to log out all users.`,
		Args: func(cmd *cobra.Command, args []string) error {
			allFlag, _ := cmd.Flags().GetBool("all")
			if allFlag && len(args) > 0 {
				return errors.New("cannot specify an email when using --all")
			}
			if !allFlag && len(args) != 1 {
				return errors.New("specify an email or use --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				return logoutAll()
			}
			return logoutByEmail(args[0])
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Log out all authenticated users")
	return cmd
}

func logoutByEmail(email string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	sessions := cfg.SessionByEmail(email)
	if len(sessions) == 0 {
		return customerrors.NewUserErrorWithHint(
			fmt.Sprintf("No sessions found for %s.", email),
			"Run 'datumctl auth list' to see authenticated users.",
		)
	}

	for _, s := range sessions {
		deleteKeyringEntry(s.UserKey)
	}

	cfg.RemoveSessionsByEmail(email)

	// If current context is gone, clear it.
	if cfg.CurrentContext != "" && cfg.ContextByName(cfg.CurrentContext) == nil {
		cfg.CurrentContext = ""
	}

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\u2713 Logged out %s.\n", email)
	return nil
}

func logoutAll() error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	for _, s := range cfg.Sessions {
		deleteKeyringEntry(s.UserKey)
	}

	cfg.Sessions = nil
	cfg.Contexts = nil
	cfg.CurrentContext = ""
	cfg.ActiveSession = ""

	// Also clean up legacy keyring entries.
	cleanupLegacyKeyring()

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println("\u2713 Logged out all users.")
	return nil
}

func deleteKeyringEntry(userKey string) {
	if err := keyring.Delete(authutil.ServiceName, userKey); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete credentials for %s: %v\n", userKey, err)
	}
}

func cleanupLegacyKeyring() {
	// Clean up known_users list and active_user key.
	knownUsersJSON, err := keyring.Get(authutil.ServiceName, authutil.KnownUsersKey)
	if err == nil && knownUsersJSON != "" {
		var knownUsers []string
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err == nil {
			for _, uk := range knownUsers {
				deleteKeyringEntry(uk)
			}
		}
	}
	_ = keyring.Delete(authutil.ServiceName, authutil.KnownUsersKey)
	_ = keyring.Delete(authutil.ServiceName, authutil.ActiveUserKey)
}
