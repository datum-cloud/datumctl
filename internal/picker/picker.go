package picker

import (
	"fmt"
	"os"
	"sort"

	"charm.land/huh/v2"
	"golang.org/x/term"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// SelectContext presents an interactive picker for choosing one of the
// directory's contexts. If only one context is available, it is auto-selected.
// currentContext marks the active selection.
func SelectContext(dir *discovery.Directory, currentContext string) (*datumconfig.DiscoveredContext, error) {
	contexts := dir.Contexts()
	if len(contexts) == 0 {
		return nil, customerrors.NewUserErrorWithHint(
			"No contexts available.",
			"Run 'datumctl login' to authenticate and discover your organizations and projects.",
		)
	}

	if len(contexts) == 1 {
		return &contexts[0], nil
	}

	if !isTerminal() {
		return nil, customerrors.NewUserErrorWithHint(
			"Interactive context selection requires a terminal.",
			"Use --project or --organization flags, or set DATUM_PROJECT / DATUM_ORGANIZATION environment variables.",
		)
	}

	options := buildContextOptions(contexts, dir, currentContext)

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
		return nil, fmt.Errorf("context selection: %w", err)
	}

	for i := range contexts {
		if contexts[i].Name == selected {
			return &contexts[i], nil
		}
	}
	return nil, fmt.Errorf("selected context not found")
}

// buildContextOptions groups contexts by org and formats them with visual
// hierarchy: org entries appear as headers, projects are indented beneath.
func buildContextOptions(contexts []datumconfig.DiscoveredContext, dir *discovery.Directory, currentContext string) []huh.Option[string] {
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
			label := datumconfig.FormatWithID(dir.OrgDisplayName(orgID), orgID)
			if currentContext == g.orgCtx.Name {
				label += "  *"
			}
			options = append(options, huh.NewOption(label, g.orgCtx.Name))
		}

		// Project entries, indented under their org.
		for _, p := range g.projects {
			label := "  " + datumconfig.FormatWithID(dir.ProjectDisplayName(p.ProjectID), p.ProjectID)
			if currentContext == p.Name {
				label += "  *"
			}
			options = append(options, huh.NewOption(label, p.Name))
		}
	}

	return options
}

// SelectSession presents an interactive picker for disambiguating between
// sessions that share the same email. The session whose name matches
// activeSessionName is labelled "(active)" and pre-selected so the picker opens
// on the currently active session. Pass "" when no session is active. Returns
// the selected session name.
func SelectSession(sessions []*datumconfig.Session, activeSessionName string) (string, error) {
	if len(sessions) == 0 {
		return "", fmt.Errorf("no sessions to select from")
	}

	if len(sessions) == 1 {
		return sessions[0].Name, nil
	}

	if !isTerminal() {
		return "", customerrors.NewUserErrorWithHint(
			"Multiple sessions match. Interactive selection requires a terminal.",
			"Run 'datumctl auth list' to see sessions, then 'datumctl auth switch <email> --endpoint <host>'.",
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
		if s.Name == activeSessionName {
			label += "  (active)"
		}
		options[i] = huh.NewOption(label, s.Name)
	}

	// Pre-select the active session so the picker opens on it.
	selected := activeSessionName
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

// IsTerminal reports whether stdin is an interactive terminal. It is a
// variable so callers' tests can simulate running with or without a terminal.
var IsTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

func isTerminal() bool {
	return IsTerminal()
}
