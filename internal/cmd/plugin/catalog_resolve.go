package plugin

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/pluginstore"
)

// catalogMatch is a plugin found in a particular catalog during bare-name
// resolution.
type catalogMatch struct {
	catalog pluginstore.Catalog
	plugin  *pluginstore.Plugin
}

// loadOrRefreshCatalog loads a catalog's cached index and refreshes it when
// stale. On a refresh failure it degrades to the cache (with a warning) and
// only returns an error when no cache is available at all.
func loadOrRefreshCatalog(cmd *cobra.Command, pluginsDir string, cat pluginstore.Catalog) (*pluginstore.CachedIndex, error) {
	idx, _ := pluginstore.LoadCatalogIndex(pluginsDir, cat.Name)
	if !pluginstore.IsStale(idx) {
		return idx, nil
	}
	fresh, err := pluginstore.RefreshCatalog(cmd.Context(), pluginsDir, cat)
	if err != nil {
		if fresh == nil {
			return nil, err
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: catalog %q refresh failed (%v), using cached results\n", cat.Name, err)
		return fresh, nil
	}
	return fresh, nil
}

// resolveBareName finds every catalog that contains a plugin with the given
// name. The default catalog is checked first. A catalog that cannot be reached
// is skipped with a warning rather than failing the whole resolution.
func resolveBareName(cmd *cobra.Command, pluginsDir string, reg *pluginstore.Registry, name string) []catalogMatch {
	var matches []catalogMatch
	for i := range reg.Catalogs {
		cat := reg.Catalogs[i]
		idx, err := loadOrRefreshCatalog(cmd, pluginsDir, cat)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping catalog %q: %v\n", cat.Name, err)
			continue
		}
		if p := pluginstore.FindInIndex(idx, name); p != nil {
			matches = append(matches, catalogMatch{catalog: cat, plugin: p})
		}
	}
	return matches
}

// collisionError renders the "available from multiple catalogs" message that
// asks the user to qualify a bare name. Matches are listed default-first.
func collisionError(name string, matches []catalogMatch) error {
	var b strings.Builder
	fmt.Fprintf(&b, "%q is available from multiple catalogs. Choose one:", name)
	for _, m := range matches {
		desc := m.plugin.Spec.ShortDescription
		fmt.Fprintf(&b, "\n  datumctl plugin install %s/%s\t%s\t(%s)",
			m.catalog.Name, name, desc, m.catalog.Trust())
	}
	return fmt.Errorf("%s", b.String())
}

// installBadge returns the trust badge for a plugin installed from the given
// catalog. Only the reserved default catalog is "official".
func installBadge(catalog string) string {
	if catalog == pluginstore.DefaultCatalogName {
		return pluginstore.TrustOfficial
	}
	return pluginstore.TrustThirdParty
}

// installedCatalogLabel returns the INDEX and TRUST columns for an installed
// plugin record, accounting for legacy records that predate the catalog field.
func installedCatalogLabel(entry *pluginstore.InstalledPlugin) (index, trust string) {
	if entry.Catalog != "" {
		return entry.Catalog, installBadge(entry.Catalog)
	}
	// Legacy records: a slash in Source means a direct GitHub install; otherwise
	// it was installed from the curated default catalog.
	if strings.Contains(entry.Source, "/") {
		return "(direct)", pluginstore.TrustThirdParty
	}
	return pluginstore.DefaultCatalogName, pluginstore.TrustOfficial
}
