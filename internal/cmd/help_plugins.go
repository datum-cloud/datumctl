package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/pluginstore"
)

// Plugins are dispatched dynamically rather than registered as Cobra commands
// (see the root RunE), so they are invisible to Cobra's own command listing.
// The two sections below put them back into `datumctl --help`: what the user
// already has, and what the cached catalogs offer.
const (
	helpInstalledTitle = "Installed Plugins"
	helpAvailableTitle = "Available Plugins"

	// helpAvailableLimit caps the "Available Plugins" section. --help is a
	// reference, so it is far more generous than the landing's cap, but a large
	// third-party catalog must not be able to bury the flag list.
	helpAvailableLimit = 10

	// helpNamePadding matches Cobra's minimum command-name column so plugin rows
	// line up with the built-in commands listed above them.
	helpNamePadding = 13
)

// installPluginsHelpSections rewrites root's usage template so plugin sections
// render between the command groups and the flag list, where a reader already
// looks for things they can run.
//
// It edits Cobra's own template rather than replacing it, so upstream changes to
// the command listing keep flowing through. If the anchor it splices at ever
// disappears, the template is left untouched: a --help without the plugin
// sections beats a --help rebuilt from a stale copy of Cobra's layout.
func installPluginsHelpSections(root *cobra.Command) {
	const anchor = "{{if .HasAvailableLocalFlags}}"

	tmpl := root.UsageTemplate()
	if !strings.Contains(tmpl, anchor) {
		return
	}
	cobra.AddTemplateFunc("datumctlPluginSections", pluginHelpSections)
	root.SetUsageTemplate(strings.Replace(tmpl, anchor, "{{datumctlPluginSections}}"+anchor, 1))
}

// pluginHelpSections renders the plugin sections for the root help, or "" when
// there is nothing to say. Like the landing blocks it is best-effort and
// local-only: it reads the install record and the catalog cache already on
// disk, and never fetches, so --help stays instant and works offline.
func pluginHelpSections() string {
	installed := collectInstalledPlugins()
	installedNames := make(map[string]bool, len(installed))
	for _, p := range installed {
		installedNames[p.name] = true
	}
	available := collectAvailablePlugins(installedNames)

	var b strings.Builder
	if len(installed) > 0 {
		fmt.Fprintf(&b, "\n\n%s", helpInstalledTitle)
		for _, p := range installed {
			fmt.Fprintf(&b, "\n  %s %s", rpad(p.name, helpNamePadding), helpPluginSummary(p.desc, p.source))
		}
	}
	if len(available) > 0 {
		shown := available
		if len(shown) > helpAvailableLimit {
			shown = shown[:helpAvailableLimit]
		}
		fmt.Fprintf(&b, "\n\n%s", helpAvailableTitle)
		for _, p := range shown {
			fmt.Fprintf(&b, "\n  %s %s", rpad(p.name, helpNamePadding), p.desc)
		}
		if remaining := len(available) - len(shown); remaining > 0 {
			fmt.Fprintf(&b, "\n  %s +%d more — run 'datumctl plugin browse'", rpad("", helpNamePadding), remaining)
		}
		fmt.Fprintf(&b, "\n\nInstall a plugin with 'datumctl plugin install <name>'.")
	}
	return strings.TrimRight(b.String(), " ")
}

// helpPluginSummary describes an installed plugin, falling back to naming the
// catalog it came from when the plugin binary reported no description.
func helpPluginSummary(desc, source string) string {
	if desc != "" {
		return desc
	}
	if source != "" {
		return "Installed from the " + source + " catalog"
	}
	return "Installed plugin"
}

// rpad right-pads s to at least width columns, matching Cobra's own command
// listing so plugin rows align with the built-in commands.
func rpad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// installedPlugin is one row of the installed-plugin listings.
type installedPlugin struct {
	name   string
	desc   string
	source string
}

// collectInstalledPlugins returns every managed plugin in the local install
// record, sorted by name. Any error reading the store yields no plugins rather
// than a failure: these listings decorate other output and must never break it.
func collectInstalledPlugins() []installedPlugin {
	dir, err := pluginstore.PluginsDir("")
	if err != nil {
		return nil
	}
	manifest, err := pluginstore.Load(dir)
	if err != nil || manifest == nil {
		return nil
	}

	plugins := make([]installedPlugin, 0, len(manifest.Plugins))
	for name, entry := range manifest.Plugins {
		p := installedPlugin{name: name, source: landingPluginSource(entry)}
		if entry != nil && entry.Manifest != nil {
			p.desc = entry.Manifest.Description
		}
		plugins = append(plugins, p)
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].name < plugins[j].name })
	return plugins
}

// availablePlugin is one row of the available-plugin listings.
type availablePlugin struct {
	name string
	desc string
}

// collectAvailablePlugins returns the plugins the cached catalogs offer that
// are not in installed, sorted by name. It reads only what is already on disk —
// a cold cache yields nothing rather than a fetch, so callers stay instant and
// work offline.
func collectAvailablePlugins(installed map[string]bool) []availablePlugin {
	dir, err := pluginstore.PluginsDir("")
	if err != nil {
		return nil
	}
	reg, err := pluginstore.LoadRegistry(dir)
	if err != nil {
		return nil
	}

	var plugins []availablePlugin
	seen := map[string]bool{}
	for _, cat := range reg.Active() {
		idx, err := pluginstore.LoadCatalogIndex(dir, cat.Name)
		if err != nil || idx == nil {
			continue
		}
		for i := range idx.Plugins {
			p := &idx.Plugins[i]
			// First catalog to offer a name wins, matching the order
			// `plugin install` resolves in, so these listings never advertise a
			// plugin the install would take from a different catalog.
			if p.Name == "" || installed[p.Name] || seen[p.Name] {
				continue
			}
			seen[p.Name] = true
			plugins = append(plugins, availablePlugin{name: p.Name, desc: p.Spec.ShortDescription})
		}
	}
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].name < plugins[j].name })
	return plugins
}
