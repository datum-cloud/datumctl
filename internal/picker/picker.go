package picker

import (
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// SelectContext presents an interactive picker for choosing a context.
// If only one context is available, it is auto-selected. Returns the context name.
func SelectContext(contexts []datumconfig.DiscoveredContext, cfg *datumconfig.ConfigV1Beta1) (string, error) {
	if len(contexts) == 0 {
		return "", customerrors.NewUserErrorWithHint(
			"No contexts available.",
			"Run 'datumctl login' to authenticate and discover your organizations and projects.",
		)
	}

	if len(contexts) == 1 {
		return contexts[0].Name, nil
	}

	if !isTerminal() {
		return "", customerrors.NewUserErrorWithHint(
			"Interactive context selection requires a terminal.",
			"Use --project or --organization flags, or set DATUM_PROJECT / DATUM_ORGANIZATION environment variables.",
		)
	}

	options := buildContextOptions(contexts, cfg)

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a context to work in").
				Options(options...).
				Value(&selected).
				Filtering(true),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("context selection: %w", err)
	}

	return selected, nil
}

// buildContextOptions groups contexts by org and formats them with visual
// hierarchy: org entries appear as headers, projects are indented beneath.
func buildContextOptions(contexts []datumconfig.DiscoveredContext, cfg *datumconfig.ConfigV1Beta1) []huh.Option[string] {
	// Separate orgs and projects, group projects by org ID.
	type orgGroup struct {
		orgCtx   *datumconfig.DiscoveredContext
		projects []datumconfig.DiscoveredContext
	}

	groups := make(map[string]*orgGroup)
	var orgOrder []string

	for i := range contexts {
		ctx := &contexts[i]
		if ctx.ProjectID == "" {
			// Org-level context.
			if _, ok := groups[ctx.OrganizationID]; !ok {
				groups[ctx.OrganizationID] = &orgGroup{}
				orgOrder = append(orgOrder, ctx.OrganizationID)
			}
			groups[ctx.OrganizationID].orgCtx = ctx
		}
	}

	for i := range contexts {
		ctx := contexts[i]
		if ctx.ProjectID != "" {
			orgID := ctx.OrganizationID
			if _, ok := groups[orgID]; !ok {
				groups[orgID] = &orgGroup{}
				orgOrder = append(orgOrder, orgID)
			}
			groups[orgID].projects = append(groups[orgID].projects, ctx)
		}
	}

	// Sort projects within each group by name.
	for _, g := range groups {
		sort.Slice(g.projects, func(i, j int) bool {
			return g.projects[i].Name < g.projects[j].Name
		})
	}

	// Build options with visual grouping.
	var options []huh.Option[string]
	for _, orgID := range orgOrder {
		g := groups[orgID]

		// Org entry — show display name with resource name when they differ.
		if g.orgCtx != nil {
			label := datumconfig.FormatWithID(cfg.OrgDisplayName(orgID), orgID)
			if cfg.CurrentContext == g.orgCtx.Name {
				label += "  *"
			}
			options = append(options, huh.NewOption(label, g.orgCtx.Name))
		}

		// Project entries, indented under their org.
		for _, p := range g.projects {
			label := "  " + datumconfig.FormatWithID(cfg.ProjectDisplayName(p.ProjectID), p.ProjectID)
			if cfg.CurrentContext == p.Name {
				label += "  *"
			}
			options = append(options, huh.NewOption(label, p.Name))
		}
	}

	return options
}

// SelectSession presents an interactive picker for disambiguating between
// sessions that share the same email. Returns the session name.
func SelectSession(sessions []*datumconfig.Session) (string, error) {
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions to select from")
	}

	if len(sessions) == 1 {
		return sessions[0].Name, nil
	}

	if !isTerminal() {
		return "", customerrors.NewUserErrorWithHint(
			"Multiple sessions found for this email. Interactive selection requires a terminal.",
			"Run 'datumctl auth list' to see sessions and identify the email + endpoint to use.",
		)
	}

	// Only show endpoint when sessions span multiple endpoints.
	showEndpoint := false
	if len(sessions) > 1 {
		first := sessions[0].Endpoint.Server
		for _, s := range sessions[1:] {
			if s.Endpoint.Server != first {
				showEndpoint = true
				break
			}
		}
	}

	options := make([]huh.Option[string], len(sessions))
	for i, s := range sessions {
		label := s.UserEmail
		if showEndpoint {
			label = fmt.Sprintf("%s  (%s)", s.UserEmail, datumconfig.StripScheme(s.Endpoint.Server))
		}
		options[i] = huh.NewOption(label, s.Name)
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which login session?").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", fmt.Errorf("session selection: %w", err)
	}

	return selected, nil
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
