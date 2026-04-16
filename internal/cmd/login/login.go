package login

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
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

By default, opens your browser for OAuth2 PKCE authentication. Use
--no-browser in headless environments (SSH, CI, containers) to authenticate
via a device-code flow that does not need a browser on this machine.

Use --credentials to authenticate as a machine account (non-interactive).`,
		Example: `  # Log in (opens browser, then picks a context)
  datumctl login

  # Log in without a browser (device-code flow for headless/CI)
  datumctl login --no-browser

  # Log in to a staging environment
  datumctl login --hostname auth.staging.env.datum.net

  # Log in with a machine account credentials file
  datumctl login --credentials ./my-key.json --hostname auth.staging.env.datum.net`,
		RunE: runLogin,
	}

	cmd.Flags().StringVar(&hostname, "hostname", "auth.datum.net", "Hostname of the Datum Cloud authentication server")
	cmd.Flags().StringVar(&apiHostnameFlag, "api-hostname", "", "Hostname of the Datum Cloud API server (derived from auth hostname if omitted)")
	cmd.Flags().StringVar(&clientIDFlag, "client-id", "", "Override the OAuth2 Client ID")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Use the device authorization flow instead of opening a browser")
	cmd.Flags().StringVar(&credentialsFile, "credentials", "", "Path to a machine account credentials JSON file")
	cmd.Flags().BoolVar(&debugCredentials, "debug", false, "Print JWT claims and token request details (credentials flow only)")

	return cmd
}

func runLogin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	var result *authutil.LoginResult
	var authHostname string

	if credentialsFile != "" {
		r, err := authutil.RunMachineAccountLogin(ctx, credentialsFile, hostname, apiHostnameFlag, debugCredentials)
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
	orgs, projects, err := discovery.FetchOrgsAndProjects(ctx, apiHostname, tknSrc, result.Subject)
	if err != nil {
		fmt.Printf("Warning: could not discover contexts: %v\n", err)
		fmt.Println("\nYou can set a context manually with 'datumctl ctx use'.")
		if saveErr := datumconfig.SaveV1Beta1(cfg); saveErr != nil {
			return fmt.Errorf("save config: %w", saveErr)
		}
		return nil
	}

	if len(orgs) > 0 {
		fmt.Printf("You have access to %d organization(s):\n\n", len(orgs))
		for _, o := range orgs {
			projCount := 0
			for _, p := range projects {
				if p.OrgName == o.Name {
					projCount++
				}
			}
			fmt.Printf("  %s (%d project(s))\n", o.Name, projCount)
		}
		fmt.Println()
	}

	discovery.UpdateConfigCache(cfg, sessionName, orgs, projects)

	sessionContexts := cfg.ContextsForSession(sessionName)
	if len(sessionContexts) == 0 {
		fmt.Println("\nNo contexts available.")
		if saveErr := datumconfig.SaveV1Beta1(cfg); saveErr != nil {
			return fmt.Errorf("save config: %w", saveErr)
		}
		return nil
	}

	selected, err := picker.SelectContext(sessionContexts, cfg)
	if err != nil {
		return err
	}

	cfg.CurrentContext = selected
	if s := cfg.SessionByName(sessionName); s != nil {
		s.LastContext = selected
	}

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	ctxEntry := cfg.ContextByName(selected)
	if ctxEntry != nil {
		fmt.Printf("\n\u2713 Context set to %s\n", datumconfig.FormatWithID(cfg.DisplayRef(ctxEntry), ctxEntry.Ref()))
	} else {
		fmt.Printf("\n\u2713 Context set to %s\n", selected)
	}
	return nil
}

func deriveAuthHostname(apiHostname string) (string, error) {
	if rest, ok := strings.CutPrefix(apiHostname, "api."); ok {
		return "auth." + rest, nil
	}
	return "", fmt.Errorf("cannot derive auth hostname from '%s'; expected api.* prefix", apiHostname)
}
