package onboarding

import (
	"fmt"
	"net/url"
	"strings"
)

// DerivePortalURL maps an API hostname to the corresponding cloud-portal base URL.
func DerivePortalURL(apiHostname string) (string, error) {
	host := strings.TrimPrefix(apiHostname, "https://")
	host = strings.TrimPrefix(host, "http://")

	switch {
	case strings.HasSuffix(host, ".staging.env.datum.net"):
		return "https://cloud.staging.env.datum.net", nil
	case strings.HasSuffix(host, ".datum.net"):
		return "https://cloud.datum.net", nil
	default:
		return "", fmt.Errorf("portal URL not configured for API hostname %q", apiHostname)
	}
}

// OrgProjectsURL returns the cloud-portal org projects page. The portal redirects
// users into onboarding when the organization is not fully set up.
func OrgProjectsURL(portalBase, orgID string) string {
	return portalBase + "/org/" + url.PathEscape(orgID) + "/projects"
}

// NoOrgsResult returns a Result that directs users without organizations to the
// cloud portal, where they can complete account setup and create an organization.
func NoOrgsResult(apiHostname string) (Result, error) {
	portalBase, err := DerivePortalURL(apiHostname)
	if err != nil {
		return Result{}, err
	}
	return Result{
		State:     NeedsOnboarding,
		PortalURL: portalBase,
		ActionURL: portalBase,
	}, nil
}
