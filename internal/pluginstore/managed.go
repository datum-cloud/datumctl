package pluginstore

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

// Environment variables for managed (enterprise) plugin-catalog configuration.
const (
	// managedConfigEnvVar points at a YAML file describing pre-seeded catalogs
	// and an optional allow-list. Set by managed configuration on locked-down
	// workstations.
	managedConfigEnvVar = "DATUMCTL_PLUGIN_MANAGED_CONFIG"

	// allowedIndexesEnvVar is a convenience allow-list (comma-separated names or
	// host patterns) that supplements any list from the managed config file.
	allowedIndexesEnvVar = "DATUMCTL_PLUGIN_ALLOWED_INDEXES"
)

// ManagedConfig is the optional enterprise configuration for plugin catalogs.
// It lets a platform team pre-seed approved catalogs onto a workstation and, via
// an allow-list, constrain which catalogs a user may add themselves.
type ManagedConfig struct {
	// Indexes are catalogs pre-registered for the user with no manual action.
	Indexes []ManagedIndex `json:"indexes,omitempty"`
	// AllowedIndexes constrains what a user may add. When empty, any catalog may
	// be added. Entries match a catalog by name or by source host (a bare host,
	// a "*.example.com" wildcard, or a parent domain).
	AllowedIndexes []string `json:"allowedIndexes,omitempty"`
}

// ManagedIndex is a single pre-seeded catalog from managed configuration.
type ManagedIndex struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
}

// LoadManagedConfig reads managed configuration from the env-pointed file (if
// any) and merges in the env allow-list. It always returns a non-nil config;
// an unset DATUMCTL_PLUGIN_MANAGED_CONFIG yields an empty one.
func LoadManagedConfig() (*ManagedConfig, error) {
	cfg := &ManagedConfig{}

	if path := os.Getenv(managedConfigEnvVar); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read managed plugin config %q: %w", path, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse managed plugin config %q: %w", path, err)
		}
	}

	if env := os.Getenv(allowedIndexesEnvVar); env != "" {
		for _, entry := range strings.Split(env, ",") {
			if e := strings.TrimSpace(entry); e != "" {
				cfg.AllowedIndexes = append(cfg.AllowedIndexes, e)
			}
		}
	}

	return cfg, nil
}

// SeededCatalogs converts the managed pre-seeded indexes into Catalog records
// marked Managed (so they are never persisted and cannot be user-removed).
// Entries with an empty/missing source, a reserved name, or an otherwise
// invalid catalog name are skipped so a managed config cannot seed a catalog
// that would shadow a reserved name or fail to cache.
func (c *ManagedConfig) SeededCatalogs() []Catalog {
	var out []Catalog
	for _, mi := range c.Indexes {
		if mi.Source == "" || ValidateCatalogName(mi.Name) != nil {
			continue
		}
		out = append(out, Catalog{
			Name:        mi.Name,
			Source:      mi.Source,
			Type:        CatalogTypeCustom,
			Description: mi.Description,
			Owner:       mi.Owner,
			Managed:     true,
		})
	}
	return out
}

// Enforced reports whether an allow-list is in effect.
func (c *ManagedConfig) Enforced() bool {
	return len(c.AllowedIndexes) > 0
}

// IsAllowed reports whether a catalog with the given name and source may be
// added under the allow-list. When no allow-list is configured, everything is
// allowed.
//
// Allow-list entries are matched by form:
//   - Host patterns (entries containing a dot or a "*." wildcard) gate the
//     source HOST. This is how remote catalogs are authorized, e.g.
//     "plugins.acme.example" or "*.acme.example". GitHub owner/repo shorthand
//     sources resolve to the raw.githubusercontent.com host for this match.
//   - Bare names (no dot) authorize a catalog NAME, but only for local-path
//     sources. Because the user chooses the local name freely, a bare-name
//     entry is intentionally NOT sufficient to authorize a remote source — that
//     would let a trusted name be pointed at an arbitrary host. Remote sources
//     always require a host-pattern match.
func (c *ManagedConfig) IsAllowed(name, source string) bool {
	if !c.Enforced() {
		return true
	}
	host := SourceHost(source) // "" for local-path sources
	for _, entry := range c.AllowedIndexes {
		if isHostPattern(entry) {
			if host != "" && hostMatches(entry, host) {
				return true
			}
			continue
		}
		// Bare-name entry: sufficient only for local sources (no host).
		if entry == name && host == "" {
			return true
		}
	}
	return false
}

// isHostPattern reports whether an allow-list entry is a host pattern (rather
// than a bare catalog name). Catalog names never contain a dot (see
// reCatalogName), so any entry with a dot or a wildcard is a host pattern.
func isHostPattern(entry string) bool {
	return strings.HasPrefix(entry, "*.") || strings.Contains(entry, ".")
}

// hostMatches reports whether host satisfies an allow-list host pattern. A
// pattern may be an exact host, a "*.example.com" wildcard, or a parent domain
// that matches any subdomain.
func hostMatches(pattern, host string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	host = strings.ToLower(host)
	switch {
	case pattern == host:
		return true
	case strings.HasPrefix(pattern, "*."):
		return strings.HasSuffix(host, pattern[1:])
	default:
		return strings.HasSuffix(host, "."+pattern)
	}
}
