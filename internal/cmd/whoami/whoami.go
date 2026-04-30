package whoami

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
)

// Command returns the top-level "whoami" command.
func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current user and context",
		Args:  cobra.NoArgs,
		RunE:  runWhoami,
	}
}

func runWhoami(_ *cobra.Command, _ []string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	session := cfg.ActiveSessionEntry()
	if session == nil {
		return authutil.ErrNoActiveUser
	}

	// Get user info from stored credentials for freshest data.
	creds, err := authutil.GetStoredCredentials(session.UserKey)
	if err != nil {
		return fmt.Errorf("get credentials: %w", err)
	}

	userName := creds.UserName
	if userName == "" {
		userName = session.UserName
	}
	userEmail := creds.UserEmail
	if userEmail == "" {
		userEmail = session.UserEmail
	}

	fmt.Printf("User:         %s (%s)\n", userName, userEmail)

	// Show endpoint only when multiple endpoints are in use.
	if cfg.HasMultipleEndpoints() {
		fmt.Printf("Endpoint:     %s\n", datumconfig.StripScheme(session.Endpoint.Server))
	}

	ctxEntry := cfg.CurrentContextEntry()
	if ctxEntry != nil {
		fmt.Printf("Context:      %s\n", ctxEntry.Ref())

		fmt.Printf("Organization: %s\n", datumconfig.FormatWithID(
			cfg.OrgDisplayName(ctxEntry.OrganizationID), ctxEntry.OrganizationID))

		if ctxEntry.ProjectID != "" {
			fmt.Printf("Project:      %s\n", datumconfig.FormatWithID(
				cfg.ProjectDisplayName(ctxEntry.ProjectID), ctxEntry.ProjectID))
		}
	} else {
		fmt.Println("Context:      (none)")
		fmt.Println("  Run 'datumctl ctx use' to select a context.")
	}

	// Surface env-var overrides — these silently override the active context.
	if v := os.Getenv("DATUM_PROJECT"); v != "" {
		fmt.Printf("\nOverride:     DATUM_PROJECT=%s (overrides context project)\n", v)
	}
	if v := os.Getenv("DATUM_ORGANIZATION"); v != "" {
		fmt.Printf("\nOverride:     DATUM_ORGANIZATION=%s (overrides context organization)\n", v)
	}

	return nil
}

