package plugin

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/pluginstore"
)

// pluginBinaryPrefixes are the recognized datumctl plugin binary prefixes, in
// preference order. "milo-" identifies portable milo-os platform plugins;
// "datumctl-" identifies datumctl-native plugins.
var pluginBinaryPrefixes = []string{"milo-", "datumctl-"}

// archiveBinaryCandidates returns the in-archive binary names to try when
// auto-detecting a plugin binary, in preference order: the prefixed plugin
// names ("milo-<name>", "datumctl-<name>") first, then the bare "<name>" LAST.
// Trying the bare name last prevents picking up a service binary that shares
// the bare name when an archive bundles both.
func archiveBinaryCandidates(pluginName string) []string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	candidates := make([]string, 0, len(pluginBinaryPrefixes)+1)
	for _, prefix := range pluginBinaryPrefixes {
		candidates = append(candidates, prefix+pluginName+suffix)
	}
	return append(candidates, pluginName+suffix)
}

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
// Columns are padded to a common width so they align when the message is
// rendered as a plain string (UserError does not run it through a tabwriter).
func collisionError(name string, matches []catalogMatch) error {
	cmds := make([]string, len(matches))
	descs := make([]string, len(matches))
	cmdWidth, descWidth := 0, 0
	for i, m := range matches {
		cmds[i] = fmt.Sprintf("datumctl plugin install %s/%s", m.catalog.Name, name)
		descs[i] = m.plugin.Spec.ShortDescription
		if len(cmds[i]) > cmdWidth {
			cmdWidth = len(cmds[i])
		}
		if len(descs[i]) > descWidth {
			descWidth = len(descs[i])
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%q is available from multiple catalogs. Choose one:", name)
	for i, m := range matches {
		fmt.Fprintf(&b, "\n  %-*s  %-*s  (%s)",
			cmdWidth, cmds[i], descWidth, descs[i], m.catalog.Trust())
	}
	return fmt.Errorf("%s", b.String())
}

// installBadge returns the trust badge for a plugin installed from the given
// catalog. Only the reserved official catalog (under its canonical "datum" name
// or the legacy "default" alias) is "official".
func installBadge(catalog string) string {
	if pluginstore.CanonicalCatalogName(catalog) == pluginstore.OfficialCatalogName {
		return pluginstore.TrustOfficial
	}
	return pluginstore.TrustThirdParty
}

// installedCatalogLabel returns the INDEX and TRUST columns for an installed
// plugin record, accounting for legacy records that predate the catalog field
// and the pre-rename "default" catalog name.
func installedCatalogLabel(entry *pluginstore.InstalledPlugin) (index, trust string) {
	if entry.Catalog != "" {
		return pluginstore.CanonicalCatalogName(entry.Catalog), installBadge(entry.Catalog)
	}
	// Legacy records: a slash in Source means a direct GitHub install; otherwise
	// it was installed from the curated official catalog.
	if strings.Contains(entry.Source, "/") {
		return "(direct)", pluginstore.TrustThirdParty
	}
	return pluginstore.OfficialCatalogName, pluginstore.TrustOfficial
}
