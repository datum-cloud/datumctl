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

// gitHubScopePrefix marks an allow-list entry that scopes GitHub by owner (and
// optionally repo), e.g. "github.com/acme-corp/*" or
// "github.com/acme-corp/datumctl-plugins".
const gitHubScopePrefix = "github.com/"

// IsAllowed reports whether a catalog with the given name and source may be
// added under the allow-list. When no allow-list is configured, everything is
// allowed.
//
// Allow-list entries are matched by form:
//   - GitHub owner/repo scopes ("github.com/<owner>", "github.com/<owner>/*",
//     "github.com/<owner>/<repo>", or "github.com/*") authorize GitHub catalog
//     sources by owner and optionally repo. A GitHub source — the "owner/repo"
//     shorthand, the "github.com/owner/repo" form, and the
//     raw.githubusercontent.com manifest URL they resolve through — is permitted
//     ONLY by a matching scope entry. A plain "raw.githubusercontent.com" host
//     entry does NOT authorize GitHub repos, so the scoping is meaningful; use
//     "github.com/*" to authorize every GitHub repo.
//   - Host patterns (entries containing a dot or a "*." wildcard) gate the
//     source HOST for NON-GitHub remote catalogs, e.g. "plugins.acme.example"
//     or "*.acme.example".
//   - Bare names (no dot) authorize a catalog NAME, but only for local-path
//     sources. Because the user chooses the local name freely, a bare-name
//     entry is intentionally NOT sufficient to authorize a remote source — that
//     would let a trusted name be pointed at an arbitrary host. Remote sources
//     always require a host-pattern or GitHub-scope match.
func (c *ManagedConfig) IsAllowed(name, source string) bool {
	if !c.Enforced() {
		return true
	}
	host := SourceHost(source) // "" for local-path sources
	ghOwnerRepo, isGitHub := SourceGitHubOwnerRepo(source)

	for _, entry := range c.AllowedIndexes {
		le := strings.ToLower(strings.TrimSpace(entry))
		switch {
		case strings.HasPrefix(le, gitHubScopePrefix):
			// GitHub owner/repo scope: authorizes only matching GitHub sources.
			if isGitHub && gitHubScopeMatches(le[len(gitHubScopePrefix):], ghOwnerRepo) {
				return true
			}
		case isHostPattern(entry):
			// Host patterns authorize non-GitHub remote sources by host. GitHub
			// sources must be scoped by a "github.com/<owner>" entry so a bare
			// raw.githubusercontent.com entry can't green-light every repo.
			if isGitHub {
				continue
			}
			if host != "" && hostMatches(entry, host) {
				return true
			}
		default:
			// Bare-name entry: sufficient only for local sources (no host).
			if entry == name && host == "" {
				return true
			}
		}
	}
	return false
}

// gitHubScopeMatches reports whether a GitHub "owner/repo" satisfies a scope
// (the part of an allow-list entry after "github.com/"). Supported scope forms:
//   - "*"            authorizes every GitHub repo
//   - "<owner>"      authorizes every repo under the owner
//   - "<owner>/*"    authorizes every repo under the owner
//   - "<owner>/<repo>" authorizes exactly that repo
func gitHubScopeMatches(scope, ownerRepo string) bool {
	scope = strings.Trim(strings.ToLower(scope), "/")
	ownerRepo = strings.ToLower(ownerRepo)
	owner, _, _ := strings.Cut(ownerRepo, "/")
	switch {
	case scope == "":
		return false
	case scope == "*":
		return true
	case strings.HasSuffix(scope, "/*"):
		return owner == strings.TrimSuffix(scope, "/*")
	case !strings.Contains(scope, "/"):
		return owner == scope
	default:
		return scope == ownerRepo
	}
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
