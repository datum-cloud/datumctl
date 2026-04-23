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
	// Ambient-token mode: there's no local config file to load and the
	// "session" is synthesized from env vars. Short-circuit so whoami shows
	// the ambient identity + any context overrides instead of failing with
	// ErrNoActiveUser.
	if authutil.HasAmbientToken() {
		return runWhoamiAmbient()
	}

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

// runWhoamiAmbient prints the ambient identity (from DATUMCTL_* env vars)
// together with the active DATUM_PROJECT / DATUM_ORGANIZATION override, if
// any. Used when the host process (e.g. the cloud-portal embedded terminal)
// supplies the token directly rather than through the OS keyring.
func runWhoamiAmbient() error {
	creds, _, err := authutil.GetActiveCredentials()
	if err != nil {
		return fmt.Errorf("ambient credentials: %w", err)
	}

	userName := creds.UserName
	userEmail := creds.UserEmail
	switch {
	case userName != "" && userEmail != "":
		fmt.Printf("User:         %s (%s)\n", userName, userEmail)
	case userEmail != "":
		fmt.Printf("User:         %s\n", userEmail)
	case userName != "":
		fmt.Printf("User:         %s\n", userName)
	default:
		fmt.Printf("User:         %s\n", creds.Subject)
	}

	// Context is implicit in ambient mode — whichever of DATUM_PROJECT /
	// DATUM_ORGANIZATION the host set wins. Surface that clearly rather than
	// printing "(none)" like the keyring-backed path does.
	switch {
	case os.Getenv("DATUM_PROJECT") != "":
		fmt.Printf("Context:      project=%s\n", os.Getenv("DATUM_PROJECT"))
	case os.Getenv("DATUM_ORGANIZATION") != "":
		fmt.Printf("Context:      organization=%s\n", os.Getenv("DATUM_ORGANIZATION"))
	default:
		fmt.Println("Context:      (ambient, no project/organization pinned)")
	}

	fmt.Println()
	fmt.Println("Note:         Running in ambient-token mode. Auth and context are")
	fmt.Println("              managed by the host environment and cannot be changed")
	fmt.Println("              from within datumctl.")

	return nil
}

