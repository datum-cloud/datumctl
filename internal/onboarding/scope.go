package onboarding

import (
	"go.datum.net/datumctl/internal/datumconfig"
)

// ResolveOrgID determines which organization a command targets from explicit
// scope (--project, --organization, env vars) and the active context.
// Returns empty when no org-scoped target can be resolved.
func ResolveOrgID(
	projectID, organizationID string,
	ctxEntry *datumconfig.DiscoveredContext,
) string {
	if organizationID != "" {
		return organizationID
	}
	if projectID != "" {
		if ctxEntry != nil && ctxEntry.ProjectID == projectID && ctxEntry.OrganizationID != "" {
			return ctxEntry.OrganizationID
		}
		return ""
	}
	if ctxEntry != nil && ctxEntry.OrganizationID != "" {
		return ctxEntry.OrganizationID
	}
	return ""
}

// ResolveEffectiveOrgID resolves the organization for the current session using
// env overrides (DATUM_PROJECT, DATUM_ORGANIZATION) and the active context.
func ResolveEffectiveOrgID(cfg *datumconfig.ConfigV1Beta1, envProject, envOrganization string) string {
	if envOrganization != "" {
		return envOrganization
	}
	var ctxEntry *datumconfig.DiscoveredContext
	if cfg != nil {
		ctxEntry = cfg.CurrentContextEntry()
	}
	if envProject != "" {
		return ResolveOrgID(envProject, "", ctxEntry)
	}
	return ResolveOrgID("", "", ctxEntry)
}
