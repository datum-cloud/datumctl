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
	endpointFlag string
	clientIDFlag string
)

// Command returns the top-level "login" command that authenticates and selects
// a context in one step.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to Datum Cloud and select a context",
		Long: `Authenticate with Datum Cloud via OAuth2 PKCE and discover your
organizations and projects. After authentication you will be prompted to
select a default context (org or project) so subsequent commands do not
require --project or --organization flags.`,
		RunE: runLogin,
	}

	cmd.Flags().StringVar(&endpointFlag, "endpoint", "", "API endpoint URL (defaults to https://api.datum.net)")
	cmd.Flags().StringVar(&clientIDFlag, "client-id", "", "Override the OAuth2 Client ID")

	return cmd
}

func runLogin(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Derive auth and API hostnames from the endpoint flag.
	apiHostname := "api.datum.net"
	authHostname := "auth.datum.net"

	if endpointFlag != "" {
		apiHostname = datumconfig.StripScheme(endpointFlag)
		derived, err := deriveAuthHostname(apiHostname)
		if err != nil {
			return err
		}
		authHostname = derived
	}

	clientID, err := authutil.ResolveClientID(clientIDFlag, authHostname)
	if err != nil {
		return err
	}

	// Run the PKCE login flow.
	result, err := authutil.RunPKCELogin(ctx, authHostname, apiHostname, clientID)
	if err != nil {
		return err
	}

	fmt.Printf("\u2713 Authenticated as %s (%s)\n\n", result.UserName, result.UserEmail)

	// Load or create config.
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Create or update session.
	session := authutil.BuildSession(result, authHostname)
	cfg.UpsertSession(session)
	cfg.ActiveSession = session.Name
	sessionName := session.Name

	// Get a token source for API discovery.
	tknSrc, err := authutil.GetTokenSourceForUser(ctx, result.UserKey)
	if err != nil {
		return fmt.Errorf("get token source: %w", err)
	}

	// Discover orgs and projects.
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

	// Print summary.
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

	// Update cache and generate contexts.
	discovery.UpdateConfigCache(cfg, sessionName, orgs, projects)

	// Select a context.
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
	// Update the session's last context.
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
