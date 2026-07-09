package serviceactivation

import (
	"fmt"
	"strings"
)

// Config parameterizes the activation flow for one service. It carries only
// service identity and copy nouns; per-invocation values (project, client, IO
// streams) are supplied separately so a single Config can be shared.
type Config struct {
	// ObjectName is the Service's metadata.name. It is written to
	// spec.serviceRef.name on create (admission rejects the canonical name) and
	// is the object-name fallback when selecting the entitlement.
	ObjectName string

	// CanonicalName is the reverse-DNS service identity (e.g.
	// "compute.datumapis.com"). It is the preferred selection key: dependency-origin
	// entitlements carry it in spec.serviceRef.name, so matching on it avoids
	// mistaking a dependency entitlement for the direct one.
	CanonicalName string

	// DisplayName is the human noun for the service, capitalized for use at the
	// start of a sentence (e.g. "Compute"). Mid-sentence uses are lowercased.
	DisplayName string

	// AccessCommand is the fully-qualified, plugin-local verb root printed in
	// next-step copy (e.g. "datumctl compute access"). Copy references only this
	// verb, never datumctl core commands that may not exist on the running host.
	AccessCommand string

	// SupportURL is an optional pointer shown when the service is unavailable on
	// this platform environment.
	SupportURL string
}

// Validate reports whether the required identity fields are set.
func (c Config) Validate() error {
	if c.ObjectName == "" {
		return fmt.Errorf("serviceactivation: Config.ObjectName is required")
	}
	if c.CanonicalName == "" {
		return fmt.Errorf("serviceactivation: Config.CanonicalName is required")
	}
	if c.DisplayName == "" {
		return fmt.Errorf("serviceactivation: Config.DisplayName is required")
	}
	if c.AccessCommand == "" {
		return fmt.Errorf("serviceactivation: Config.AccessCommand is required")
	}
	return nil
}

// noun returns the display noun for mid-sentence use ("compute").
func (c Config) noun() string { return strings.ToLower(c.DisplayName) }

// requestCommand is the explicit request verb ("datumctl compute access request").
func (c Config) requestCommand() string { return c.AccessCommand + " request" }
