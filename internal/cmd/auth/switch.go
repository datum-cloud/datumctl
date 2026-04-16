package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/picker"
)

var switchCmd = &cobra.Command{
	Use:   "switch [email]",
	Short: "Switch the active user",
	Long: `Switch the active user to a different authenticated session.

If no email is provided, an interactive picker is shown.
If the email exists on multiple endpoints, you will be prompted to choose.
The last-used context for that session is restored.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSwitch,
}

func runSwitch(_ *cobra.Command, args []string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	var sessionName string

	if len(args) == 0 {
		// Interactive picker of all sessions.
		if len(cfg.Sessions) == 0 {
			return customerrors.NewUserErrorWithHint(
				"No authenticated sessions.",
				"Run 'datumctl login' to authenticate.",
			)
		}
		allSessions := make([]*datumconfig.Session, len(cfg.Sessions))
		for i := range cfg.Sessions {
			allSessions[i] = &cfg.Sessions[i]
		}
		sessionName, err = picker.SelectSession(allSessions)
		if err != nil {
			return err
		}
	} else {
		email := args[0]
		sessions := cfg.SessionByEmail(email)
		if len(sessions) == 0 {
			return customerrors.NewUserErrorWithHint(
				fmt.Sprintf("No sessions found for %s.", email),
				"Run 'datumctl auth list' to see authenticated users, or 'datumctl login' to add a new one.",
			)
		}
		if len(sessions) == 1 {
			sessionName = sessions[0].Name
		} else {
			sessionName, err = picker.SelectSession(sessions)
			if err != nil {
				return err
			}
		}
	}

	session := cfg.SessionByName(sessionName)
	if session == nil {
		return fmt.Errorf("session %q not found", sessionName)
	}

	cfg.ActiveSession = sessionName

	// Restore last context for this session.
	if session.LastContext != "" {
		if cfg.ContextByName(session.LastContext) != nil {
			cfg.CurrentContext = session.LastContext
		}
	}

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n\u2713 Switched to %s (%s)\n", session.UserName, session.UserEmail)
	if ctxEntry := cfg.CurrentContextEntry(); ctxEntry != nil {
		fmt.Printf("  Context:  %s\n", datumconfig.FormatWithID(cfg.DisplayRef(ctxEntry), ctxEntry.Ref()))
	}
	if cfg.HasMultipleEndpoints() {
		fmt.Printf("  Endpoint: %s\n", datumconfig.StripScheme(session.Endpoint.Server))
	}

	return nil
}

