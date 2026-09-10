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

	// telemetryGroup is the aggregated APIService group queryapi registers
	// under on the milo core control plane, and telemetryVer the version its
	// routes are served at. These must match the APIService object
	// (o11y config/queryapi-api-registration/apiservice.yaml) exactly: queryapi
	// derives both the paths it serves and the permissions it reviews from
	// them, so a mismatch here is a 404 from the aggregator, not a fallback.
	telemetryGroup = "o11y.miloapis.com"
	telemetryVer   = "v1alpha1"
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

// ProjectLogsAPIPrefix returns the queryapi path prefix for the log signal, to
// append to a project control plane host. Appending it to
// ProjectControlPlaneURL reproduces the aggregator path queryapi's openapi.yaml
// declares as its server:
//
//	/projects/{id}/control-plane/apis/o11y.miloapis.com/v1alpha1/logs
//
// The trailing "logs" segment scopes the request to the log signal; it is also
// the resource queryapi authorizes against, so it is not optional. Loki-shaped
// endpoints hang off this prefix, e.g. .../logs/loki/api/v1/query_range.
func ProjectLogsAPIPrefix() string {
	return fmt.Sprintf("/apis/%s/%s/logs", telemetryGroup, telemetryVer)
}

func normalizeBase(s string) string {
	return datumconfig.CleanBaseServer(datumconfig.EnsureScheme(s))
}
