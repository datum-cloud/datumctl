// Package miloapi centralizes URL construction for Milo control-plane API paths.
// All datumctl callers should use these helpers instead of building paths by hand.
package miloapi

import (
	"fmt"

	"go.datum.net/datumctl/internal/datumconfig"
)

const (
	resourceManagerGroup = "resourcemanager.miloapis.com"
	iamGroup             = "iam.miloapis.com"
	apiVersion           = "v1alpha1"
)

// UserControlPlaneURL returns the URL of a user's control plane.
func UserControlPlaneURL(baseServer, userID string) string {
	return fmt.Sprintf("%s/apis/%s/%s/users/%s/control-plane",
		normalizeBase(baseServer), iamGroup, apiVersion, userID)
}

// OrgControlPlaneURL returns the URL of an organization's control plane.
func OrgControlPlaneURL(baseServer, orgID string) string {
	return fmt.Sprintf("%s/apis/%s/%s/organizations/%s/control-plane",
		normalizeBase(baseServer), resourceManagerGroup, apiVersion, orgID)
}

// ProjectControlPlaneURL returns the URL of a project's control plane.
func ProjectControlPlaneURL(baseServer, projectID string) string {
	return fmt.Sprintf("%s/apis/%s/%s/projects/%s/control-plane",
		normalizeBase(baseServer), resourceManagerGroup, apiVersion, projectID)
}

func normalizeBase(s string) string {
	return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(s))
}
