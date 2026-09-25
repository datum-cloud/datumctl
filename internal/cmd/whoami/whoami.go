package whoami

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/onboarding"
)

// Command returns the top-level "whoami" command.
func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the current user and context",
		Long: templates.LongDesc(`
			Show the account and context datumctl commands run as.

			With --session or DATUM_SESSION, shows the account and context that
			session gives, and notes that the active session is unchanged.`),
		Example: templates.Examples(`
			# Show the active account and context
			datumctl whoami

			# Show what a command run as another session would use
			datumctl whoami --session user@example.com@api.staging.env.datum.net`),
		Args: cobra.NoArgs,
		RunE: runWhoami,
	}
}

func runWhoami(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()

	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}
	if err := authutil.EnsureUserKeysMigrated(cfg); err != nil {
		return err
	}

	session := cfg.ActiveSessionEntry()
	if session == nil {
		return authutil.ErrNoActiveUser
	}

	// Prefer stored credentials for the freshest name and email; the session
	// record carries both too, so missing credentials are not fatal here.
	userName, userEmail := session.UserName, session.UserEmail
	if creds, err := authutil.GetStoredCredentials(session.UserKey); err == nil {
		if creds.UserName != "" {
			userName = creds.UserName
		}
		if creds.UserEmail != "" {
			userEmail = creds.UserEmail
		}
	}

	fmt.Fprintf(out, "User:         %s (%s)\n", userName, userEmail)

	overrideName, overrideSource := datumconfig.SessionOverride()
	if overrideName != "" {
		fmt.Fprintf(out, "Session:      %s (from %s; the active session is unchanged)\n", overrideName, overrideSource)
	}

	printOnboardingStatus(cmd.Context(), out, cfg, session)

	// Show endpoint only when multiple endpoints are in use.
	if cfg.HasMultipleEndpoints() {
		fmt.Fprintf(out, "Endpoint:     %s\n", datumconfig.StripScheme(session.Endpoint.Server))
	}

	ctxEntry := cfg.CurrentContextEntry()
	if ctxEntry != nil {
		fmt.Fprintf(out, "Context:      %s\n", ctxEntry.Ref())

		fmt.Fprintf(out, "Organization: %s\n", datumconfig.FormatWithID(
			cfg.OrgDisplayName(ctxEntry.Session, ctxEntry.OrganizationID), ctxEntry.OrganizationID))

		if ctxEntry.ProjectID != "" {
			fmt.Fprintf(out, "Project:      %s\n", datumconfig.FormatWithID(
				cfg.ProjectDisplayName(ctxEntry.Session, ctxEntry.ProjectID), ctxEntry.ProjectID))
		}
	} else {
		fmt.Fprintln(out, "Context:      (none)")
		if overrideName != "" {
			fmt.Fprintln(out, "  Pass --project or --organization to choose a scope for this session.")
		} else {
			fmt.Fprintln(out, "  Run 'datumctl ctx use' to select a context.")
		}
	}

	// Surface env-var overrides — these silently override the active context.
	if v := os.Getenv("DATUM_PROJECT"); v != "" {
		fmt.Fprintf(out, "\nOverride:     DATUM_PROJECT=%s (overrides context project)\n", v)
	}
	if v := os.Getenv("DATUM_ORGANIZATION"); v != "" {
		fmt.Fprintf(out, "\nOverride:     DATUM_ORGANIZATION=%s (overrides context organization)\n", v)
	}

	return nil
}

func printOnboardingStatus(ctx context.Context, out io.Writer, cfg *datumconfig.ConfigV1Beta1, session *datumconfig.Session) {
	orgID := onboarding.ResolveEffectiveOrgID(cfg, os.Getenv("DATUM_PROJECT"), os.Getenv("DATUM_ORGANIZATION"))
	if orgID == "" {
		return
	}

	tknSrc, err := authutil.GetTokenSourceForUser(ctx, session.UserKey)
	if err != nil {
		return
	}
	userID, err := authutil.GetUserIDFromTokenForUser(session.UserKey)
	if err != nil {
		return
	}
	apiHostname, err := authutil.GetAPIHostnameForUser(session.UserKey)
	if err != nil {
		return
	}

	result, err := onboarding.CheckOrg(ctx, apiHostname, tknSrc, userID, orgID, cfg.OrgDisplayName(session.Name, orgID))
	if err != nil {
		fmt.Fprintln(out, "Onboarding:   couldn't check")
		return
	}

	fmt.Fprintf(out, "Onboarding:   %s\n", onboarding.StatusLabel(result))
	if result.State != onboarding.Complete {
		fmt.Fprintf(out, "  Finish setup at %s\n", result.ActionURL)
	}
}
