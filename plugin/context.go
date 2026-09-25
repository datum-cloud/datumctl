// Package plugin is the Go SDK for datumctl plugins. Plugin authors can import
// this package to get automatic context injection, credential helper access,
// and pre-wired Cobra flags.
//
// This package must never import from go.datum.net/datumctl/internal — it reads
// only environment variables and execs subprocesses so that any Go binary can
// depend on it without pulling in internal dependencies.
package plugin

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	resourceManagerGroup = "resourcemanager.miloapis.com"
	miloAPIVersion       = "v1alpha1"
)

// PluginContext holds the context injected by datumctl before exec-replacing a plugin.
type PluginContext struct {
	// Org is the current Datum Cloud organization slug (DATUM_ORG).
	Org string
	// Project is the current Datum Cloud project slug (DATUM_PROJECT). Empty if not set.
	Project string
	// APIHost is the Datum Cloud API base URL (DATUM_API_HOST), e.g. "api.datum.net".
	APIHost string
	// PluginAPIVersion is the integer API version the host declares (DATUM_PLUGIN_API_VERSION).
	PluginAPIVersion int
	// CredentialsHelper is the absolute path to the datumctl binary (DATUM_CREDENTIALS_HELPER).
	CredentialsHelper string
	// Session is the active datumctl session name (DATUM_SESSION). May be empty.
	Session string
}

// Context reads all DATUM_* environment variables and returns a PluginContext.
// It does not validate that required variables are set; callers should check
// PluginContext.Org / PluginContext.Project before making API calls.
func Context() PluginContext {
	apiVer, _ := strconv.Atoi(os.Getenv("DATUM_PLUGIN_API_VERSION"))
	return PluginContext{
		Org:               os.Getenv("DATUM_ORG"),
		Project:           os.Getenv("DATUM_PROJECT"),
		APIHost:           os.Getenv("DATUM_API_HOST"),
		PluginAPIVersion:  apiVer,
		CredentialsHelper: os.Getenv("DATUM_CREDENTIALS_HELPER"),
		Session:           os.Getenv("DATUM_SESSION"),
	}
}

// ControlPlaneURL returns the scoped Datum Cloud control-plane base URL for this
// context. When Project is set it returns the project control plane; otherwise it
// returns the organization control plane. The APIHost is normalized to include an
// https:// scheme if it has none.
func (c PluginContext) ControlPlaneURL() (string, error) {
	if c.APIHost == "" {
		return "", fmt.Errorf("DATUM_API_HOST is not set; is this plugin running via datumctl?")
	}
	base := normalizeAPIHost(c.APIHost)

	switch {
	case c.Project != "":
		return fmt.Sprintf("%s/apis/%s/%s/projects/%s/control-plane",
			base, resourceManagerGroup, miloAPIVersion, c.Project), nil
	case c.Org != "":
		return fmt.Sprintf("%s/apis/%s/%s/organizations/%s/control-plane",
			base, resourceManagerGroup, miloAPIVersion, c.Org), nil
	default:
		return "", fmt.Errorf("no Datum Cloud org or project in context; set DATUM_ORG or DATUM_PROJECT")
	}
}

// ControlPlaneURL is shorthand for Context().ControlPlaneURL().
func ControlPlaneURL() (string, error) {
	return Context().ControlPlaneURL()
}

// normalizeAPIHost adds an https:// scheme if the host has none and strips any
// trailing slashes. This deliberately duplicates internal/datumconfig's
// EnsureScheme and CleanBaseServer: this package must not import internal.
func normalizeAPIHost(host string) string {
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return strings.TrimRight(host, "/")
}
