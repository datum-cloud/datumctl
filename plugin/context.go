// Package plugin is the Go SDK for datumctl plugins. Plugin authors can import
// this package to get automatic context injection, credential helper access,
// and pre-wired Cobra flags.
//
// This package must never import from go.datum.net/datumctl/internal — it reads
// only environment variables and execs subprocesses so that any Go binary can
// depend on it without pulling in internal dependencies.
package plugin

import (
	"os"
	"strconv"
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
