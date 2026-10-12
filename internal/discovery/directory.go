package discovery

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"go.datum.net/datumctl/internal/datumconfig"
)

// Directory is a live listing of the organizations and projects one session
// can reach. It is fetched on demand and never persisted.
type Directory struct {
	Session  string
	Orgs     []DiscoveredOrg
	Projects []DiscoveredProject
	// Failures lists organizations whose projects could not be listed.
	Failures []OrgFailure
}

// Contexts returns an org context for every organization followed by a project
// context for every project, in listing order.
func (d *Directory) Contexts() []datumconfig.DiscoveredContext {
	contexts := make([]datumconfig.DiscoveredContext, 0, len(d.Orgs)+len(d.Projects))
	for _, o := range d.Orgs {
		contexts = append(contexts, *d.orgContext(o.Name))
	}
	for _, p := range d.Projects {
		contexts = append(contexts, *d.projectContext(p.OrgName, p.Name))
	}
	return contexts
}

func (d *Directory) orgContext(orgID string) *datumconfig.DiscoveredContext {
	ctx := &datumconfig.DiscoveredContext{Session: d.Session, OrganizationID: orgID}
	ctx.Name = ctx.QualifiedName()
	return ctx
}

func (d *Directory) projectContext(orgID, projectID string) *datumconfig.DiscoveredContext {
	ctx := &datumconfig.DiscoveredContext{
		Session:        d.Session,
		OrganizationID: orgID,
		ProjectID:      projectID,
		Namespace:      datumconfig.DefaultNamespace,
	}
	ctx.Name = ctx.QualifiedName()
	return ctx
}

// WarnFailures prints one line per organization whose projects could not be
// listed.
func (d *Directory) WarnFailures(w io.Writer) {
	for _, f := range d.Failures {
		fmt.Fprintf(w, "Warning: could not list projects for organization %s: %v\n", f.OrgID, f.Err)
	}
}

// OrgDisplayName returns the display name for an org, or the ID if unknown.
func (d *Directory) OrgDisplayName(orgID string) string {
	if d != nil {
		for _, o := range d.Orgs {
			if o.Name == orgID && o.DisplayName != "" {
				return o.DisplayName
			}
		}
	}
	return orgID
}

// ProjectDisplayName returns the display name for a project, or the ID if
// unknown.
func (d *Directory) ProjectDisplayName(projectID string) string {
	if d != nil {
		for _, p := range d.Projects {
			if p.Name == projectID && p.DisplayName != "" {
				return p.DisplayName
			}
		}
	}
	return projectID
}

// Describe returns a human-friendly description of the context type and name.
//
// Examples:
//
//	org Datum Technology, Inc (datum)
//	project Datum Cloud in Datum Technology, Inc (datum/datum-cloud)
func (d *Directory) Describe(ctx *datumconfig.DiscoveredContext) string {
	orgName := d.OrgDisplayName(ctx.OrganizationID)
	if ctx.ProjectID == "" {
		return fmt.Sprintf("org %s (%s)", orgName, ctx.OrganizationID)
	}
	return fmt.Sprintf("project %s in %s (%s)", d.ProjectDisplayName(ctx.ProjectID), orgName, ctx.Ref())
}

// Resolve finds a context by flexible matching. Resource IDs always take
// precedence over display names. It tries, in order:
//
//  1. orgID/projectID match (for "org/project" queries)
//  2. orgID-only match for org contexts
//  3. projectID-only match if unambiguous
//  4. Display-name match on org + project (scoped together, only if unambiguous)
//  5. Display-name-only org or project match (unambiguous)
//
// Returns nil if no match, or if a display-name match is ambiguous.
func (d *Directory) Resolve(query string) *datumconfig.DiscoveredContext {
	orgPart, projPart, hasSlash := strings.Cut(query, "/")

	if hasSlash {
		for _, p := range d.Projects {
			if p.OrgName == orgPart && p.Name == projPart {
				return d.projectContext(p.OrgName, p.Name)
			}
		}

		resolvedOrgIDs := d.resolveOrgIDs(orgPart)
		if len(resolvedOrgIDs) == 0 {
			resolvedOrgIDs = []string{orgPart}
		}

		var match *DiscoveredProject
		for _, orgID := range resolvedOrgIDs {
			for i := range d.Projects {
				p := &d.Projects[i]
				if p.OrgName != orgID {
					continue
				}
				if p.Name != projPart && (p.DisplayName != projPart || p.DisplayName == p.Name) {
					continue
				}
				if match != nil && match != p {
					return nil
				}
				match = p
			}
		}
		if match == nil {
			return nil
		}
		return d.projectContext(match.OrgName, match.Name)
	}

	for _, o := range d.Orgs {
		if o.Name == query {
			return d.orgContext(o.Name)
		}
	}

	var idMatch *DiscoveredProject
	for i := range d.Projects {
		if d.Projects[i].Name == query {
			if idMatch != nil {
				return nil
			}
			idMatch = &d.Projects[i]
		}
	}
	if idMatch != nil {
		return d.projectContext(idMatch.OrgName, idMatch.Name)
	}

	resolvedOrgIDs := d.resolveOrgIDs(query)
	if len(resolvedOrgIDs) == 1 {
		return d.orgContext(resolvedOrgIDs[0])
	} else if len(resolvedOrgIDs) > 1 {
		return nil
	}

	var nameMatch *DiscoveredProject
	for i := range d.Projects {
		p := &d.Projects[i]
		if p.DisplayName == query && p.DisplayName != p.Name {
			if nameMatch != nil {
				return nil
			}
			nameMatch = p
		}
	}
	if nameMatch != nil {
		return d.projectContext(nameMatch.OrgName, nameMatch.Name)
	}
	return nil
}

// resolveOrgIDs returns all org IDs whose display name matches.
func (d *Directory) resolveOrgIDs(displayName string) []string {
	var ids []string
	for _, o := range d.Orgs {
		if o.DisplayName == displayName && o.DisplayName != o.Name {
			ids = append(ids, o.Name)
		}
	}
	return ids
}

func appendUnique(s []string, v string) []string {
	if slices.Contains(s, v) {
		return s
	}
	return append(s, v)
}
