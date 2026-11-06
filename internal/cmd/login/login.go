package login

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/onboarding"
	"go.datum.net/datumctl/internal/picker"
)

var (
	hostname         string
	apiHostnameFlag  string
	clientIDFlag     string
	noBrowser        bool
	credentialsFile  string
	debugCredentials bool
)

// Command returns the top-level "login" command that authenticates and selects
// a context in one step.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Datum Cloud and select a context",
		Long: `Authenticate with Datum Cloud and discover your organizations and projects.
After authentication you will be prompted to select a default context (org
or project) so subsequent commands do not require --project or --organization
flags.

Pass --project or --organization (or set DATUM_PROJECT / DATUM_ORGANIZATION)
to choose the context without the prompt. Without a terminal and without
either, login still signs you in but leaves the context unset; pick one
afterwards with 'datumctl ctx use <org>/<project>'. Coding agents and other
tools that run datumctl without a terminal can use either route.

By default, opens your browser for OAuth2 PKCE authentication. Use
--no-browser on a machine without a browser (SSH, containers) to sign in
with a device code from another device.

CI and automation: pass --credentials with a service account credentials
JSON file to sign in with no browser and no prompts. Create the service
account in the Datum Cloud portal and download its credentials file from
there; human accounts cannot use this path. Set DATUM_PROJECT or
DATUM_ORGANIZATION on later commands to pick the scope without a context.

Login makes the new session active, so it rejects the global --session flag
and ignores DATUM_SESSION.`,
		Annotations: map[string]string{
			datumconfig.SessionOverrideAnnotation: datumconfig.SessionOverrideRejected,
		},
		Example: `  # Log in (opens browser, then picks a context)
  datumctl login

  # Log in straight into a project, with no context prompt
  datumctl login --project my-project
  datumctl login --project my-org/my-project

  # CI / automation: log in as a service account, then scope each command
  datumctl login --credentials ./key.json
  DATUM_PROJECT=my-project datumctl get dnszones

  # Log in without a browser (device code, for SSH sessions and containers)
  datumctl login --no-browser

  # Log in to a staging environment
  datumctl login --hostname auth.staging.env.datum.net
  datumctl login --credentials ./key.json --hostname auth.staging.env.datum.net`,
		RunE: runLogin,
	}

	cmd.Flags().StringVar(&hostname, "hostname", "auth.datum.net", "Hostname of the Datum Cloud authentication server")
	cmd.Flags().StringVar(&apiHostnameFlag, "api-hostname", "", "Hostname of the Datum Cloud API server (derived from auth hostname if omitted)")
	cmd.Flags().StringVar(&clientIDFlag, "client-id", "", "Override the OAuth2 Client ID")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Use the device authorization flow instead of opening a browser")
	cmd.Flags().StringVar(&credentialsFile, "credentials", "", "Path to a service account credentials JSON file (non-interactive login for CI)")
	cmd.Flags().BoolVar(&debugCredentials, "debug", false, "Print JWT claims and token request details (credentials flow only)")

	return cmd
}

func runLogin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	var result *authutil.LoginResult
	var authHostname string

	if credentialsFile != "" {
		r, err := authutil.RunServiceAccountLogin(ctx, credentialsFile, hostname, apiHostnameFlag, debugCredentials)
		if err != nil {
			return err
		}
		result = r
		authHostname = hostname
	} else {
		clientID, err := authutil.ResolveClientID(clientIDFlag, hostname)
		if err != nil {
			return err
		}
		r, err := authutil.RunInteractiveLogin(ctx, hostname, apiHostnameFlag, clientID, noBrowser, false)
		if err != nil {
			return err
		}
		result = r
		authHostname = hostname
	}

	fmt.Printf("\n\u2713 Authenticated as %s (%s)\n\n", result.UserName, result.UserEmail)

	keyring.WarnIfFallbackActive(cmd.ErrOrStderr())

	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	session := authutil.BuildSession(result, authHostname)
	cfg.UpsertSession(session)
	cfg.ActiveSession = session.Name
	sessionName := session.Name

	tknSrc, err := authutil.GetTokenSourceForUser(ctx, result.UserKey)
	if err != nil {
		return fmt.Errorf("get token source: %w", err)
	}

	apiHostname := result.APIHostname

	fmt.Print("Discovering organizations and projects...\n\n")
	dir, err := discovery.List(ctx, discovery.NewAPI(apiHostname, tknSrc, result.Subject), sessionName)
	if err != nil {
		fmt.Printf("Warning: could not discover contexts: %v\n", err)
		fmt.Println("\nYou can set a context manually with 'datumctl ctx use'.")
		if saveErr := datumconfig.SaveV1Beta1(cfg); saveErr != nil {
			return fmt.Errorf("save config: %w", saveErr)
		}
		return nil
	}
	dir.WarnFailures(cmd.ErrOrStderr())

	if len(dir.Orgs) == 0 {
		onboardingResult, err := onboarding.NoOrgsResult(apiHostname)
		if err != nil {
			return fmt.Errorf("derive portal URL: %w", err)
		}
		if saveErr := datumconfig.SaveV1Beta1(cfg); saveErr != nil {
			return fmt.Errorf("save config: %w", saveErr)
		}
		if err := browser.OpenURL(onboardingResult.ActionURL); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nWe couldn't open your browser. Head to %s to create an organization.\n", onboardingResult.ActionURL)
		}
		return onboarding.UserError(onboardingResult)
	}

	fmt.Printf("You have access to %d organization(s):\n\n", len(dir.Orgs))
	for _, o := range dir.Orgs {
		projCount := 0
		for _, p := range dir.Projects {
			if p.OrgName == o.Name {
				projCount++
			}
		}
		fmt.Printf("  %s (%d project(s))\n", o.DisplayName, projCount)
	}
	fmt.Println()

	selected, err := chooseContext(dir, cfg.CurrentContextName(), requestedContext(cmd, os.Getenv), picker.IsTerminal())
	if err != nil {
		// The sign-in itself worked, so keep the session even though no
		// context was chosen.
		if saveErr := saveWithoutContext(cfg, sessionName); saveErr != nil {
			return saveErr
		}
		return err
	}
	if selected == nil {
		if err := saveWithoutContext(cfg, sessionName); err != nil {
			return err
		}
		fmt.Println("Signed in. No context selected: choosing one needs an interactive terminal.")
		fmt.Println("Run 'datumctl ctx use <org>/<project>' to pick one ('datumctl ctx' lists them),")
		fmt.Println("or sign in with 'datumctl login --project <project>' next time.")
		return nil
	}

	onboardingResult, err := onboarding.CheckOrg(
		ctx,
		apiHostname,
		tknSrc,
		result.Subject,
		selected.OrganizationID,
		dir.OrgDisplayName(selected.OrganizationID),
	)
	if err != nil {
		return customerrors.WrapUserErrorWithHint(
			"We couldn't check whether your organization is ready yet.",
			"Try again in a moment, or finish setup in the portal.",
			err,
		)
	}
	if onboardingResult.State != onboarding.Complete {
		if saveErr := datumconfig.SaveV1Beta1(cfg); saveErr != nil {
			return fmt.Errorf("save config: %w", saveErr)
		}
		return onboarding.UserError(onboardingResult)
	}

	cfg.SelectContext(*selected)

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n\u2713 Context set to %s\n", dir.Describe(selected))
	return nil
}

// requestedContext returns the organization or project the caller asked to
// work in: --project or --organization, or, when neither flag is set,
// DATUM_PROJECT and DATUM_ORGANIZATION. With both environment variables set it
// returns "org/project". It returns "" when nothing was requested.
func requestedContext(cmd *cobra.Command, getenv func(string) string) string {
	flagValue := func(name string) string {
		if f := cmd.Flag(name); f != nil {
			return f.Value.String()
		}
		return ""
	}
	project, org := flagValue("project"), flagValue("organization")
	if project == "" && org == "" {
		project, org = getenv("DATUM_PROJECT"), getenv("DATUM_ORGANIZATION")
	}
	switch {
	case org != "" && project != "":
		return org + "/" + project
	case project != "":
		return project
	default:
		return org
	}
}

// chooseContext picks the context login switches to. A requested context must
// match one the user can reach. Otherwise a lone context is chosen
// automatically, and with several the user picks one, if there is a terminal
// to show the picker in. Without one it returns nil and no error, so login can
// keep the session and tell the user how to choose a context.
func chooseContext(dir *discovery.Directory, current, requested string, interactive bool) (*datumconfig.DiscoveredContext, error) {
	if requested != "" {
		if c := dir.Resolve(requested); c != nil {
			return c, nil
		}
		return nil, customerrors.NewUserErrorWithHint(
			fmt.Sprintf("Signed in, but no organization or project matches %q.", requested),
			"Run 'datumctl ctx' to list the ones you can use, then 'datumctl ctx use <org>/<project>'.",
		)
	}
	if contexts := dir.Contexts(); len(contexts) == 1 {
		return &contexts[0], nil
	}
	if !interactive {
		return nil, nil
	}
	return picker.SelectContext(dir, current)
}

// saveWithoutContext saves the config after a sign-in that chose no context.
// A current context from another session is cleared, because the current
// context decides which session is active and the new session should be.
func saveWithoutContext(cfg *datumconfig.ConfigV1Beta1, sessionName string) error {
	clearOtherSessionContext(cfg, sessionName)
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// clearOtherSessionContext unsets the current context when it belongs to a
// session other than sessionName. A context of the same session (signing in
// again) is kept.
func clearOtherSessionContext(cfg *datumconfig.ConfigV1Beta1, sessionName string) {
	if cur := cfg.ContextByName(cfg.CurrentContext); cur != nil && cur.Session != sessionName {
		cfg.CurrentContext = ""
	}
}

func deriveAuthHostname(apiHostname string) (string, error) {
	if rest, ok := strings.CutPrefix(apiHostname, "api."); ok {
		return "auth." + rest, nil
	}
	return "", fmt.Errorf("cannot derive auth hostname from '%s'; expected api.* prefix", apiHostname)
}
