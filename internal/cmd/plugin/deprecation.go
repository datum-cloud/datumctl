package plugin

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"

	"go.datum.net/datumctl/internal/pluginstore"
)

// Same character class pluginstore uses for catalog names; plugin names in a
// copy-paste command line must match it.
var (
	reReplacementName   = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	reReplacementSource = regexp.MustCompile(`^[A-Za-z0-9._~:/@+%][A-Za-z0-9._~:/@+%-]*$`)
)

// findCatalogBySource returns the registered catalog whose Source matches
// source, or nil. An enabled match wins over a disabled one, so a source
// registered twice still yields an install command when one copy is enabled.
func findCatalogBySource(reg *pluginstore.Registry, source string) *pluginstore.Catalog {
	if reg == nil {
		return nil
	}
	var disabled *pluginstore.Catalog
	for i := range reg.Catalogs {
		c := &reg.Catalogs[i]
		if !pluginstore.SameCatalogSource(c.Source, source) {
			continue
		}
		if !c.Disabled {
			return c
		}
		if disabled == nil {
			disabled = c
		}
	}
	return disabled
}

// sanitizeCatalogText drops control characters (unicode.IsControl,
// including ESC, \r, and \n) so catalog-supplied text can't move the
// cursor or forge extra lines.
func sanitizeCatalogText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, s)
}

// writeDeprecationNotice writes the notice for a deprecated plugin. It
// writes nothing when d is nil.
func writeDeprecationNotice(w io.Writer, reg *pluginstore.Registry, pluginName string, d *pluginstore.Deprecation) {
	if d == nil {
		return
	}
	pluginName = sanitizeCatalogText(pluginName)
	if msg := sanitizeCatalogText(d.Message); msg != "" {
		fmt.Fprintf(w, "warning: %s is deprecated: %s\n", pluginName, msg)
	} else {
		fmt.Fprintf(w, "warning: %s is deprecated.\n", pluginName)
	}
	r := d.Replacement
	if r == nil {
		return
	}
	source := r.Catalog
	replName := r.PluginName(pluginName)
	if !reReplacementName.MatchString(replName) || !reReplacementSource.MatchString(source) {
		fmt.Fprintf(w, "  The replacement named by this catalog is not valid; contact the catalog's maintainer.\n")
		return
	}
	if _, err := pluginstore.ResolveCatalogSource(source); err != nil {
		fmt.Fprintf(w, "  The replacement named by this catalog is not valid; contact the catalog's maintainer.\n")
		return
	}

	if c := findCatalogBySource(reg, source); c != nil {
		if c.Disabled {
			reason := sanitizeCatalogText(c.DisabledReason)
			if reason == "" {
				reason = "disabled by policy"
			}
			fmt.Fprintf(w, "  The replacement's catalog %q is disabled: %s\n", c.Name, reason)
			return
		}
		fmt.Fprintf(w, "  To switch to the replacement:\n")
		fmt.Fprintf(w, "    datumctl plugin install %s/%s\n", c.Name, replName)
		return
	}

	name := "<name>"
	if owner, ok := pluginstore.SourceGitHubOwnerRepo(source); ok {
		cand, _, _ := strings.Cut(owner, "/")
		if pluginstore.ValidateCatalogName(cand) == nil {
			found := (*pluginstore.Catalog)(nil)
			if reg != nil {
				found = reg.Find(cand)
			}
			if found == nil || pluginstore.SameCatalogSource(found.Source, source) {
				name = cand
			}
		}
	}
	fmt.Fprintf(w, "  To switch to the replacement:\n")
	fmt.Fprintf(w, "    datumctl plugin index add %s %s\n", name, source)
	fmt.Fprintf(w, "    datumctl plugin install %s/%s\n", name, replName)
}
