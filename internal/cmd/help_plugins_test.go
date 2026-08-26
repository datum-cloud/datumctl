package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/pluginstore"
)

func TestPluginHelpSections_namesInstalledAndAvailable(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{
		"dns": {
			Catalog:  pluginstore.OfficialCatalogName,
			Version:  "v1.2.3",
			Manifest: &pluginstore.PluginManifest{Description: "Manage DNS zones and records"},
		},
	})
	seedCatalogIndex(t, pluginstore.OfficialCatalogName,
		catalogPlugin("dns", "Manage DNS zones and records"),
		catalogPlugin("costs", "Show spend by project"),
	)

	out := pluginHelpSections()

	if !strings.Contains(out, "\n"+helpInstalledTitle+"\n") {
		t.Fatalf("expected a %q section in:\n%s", helpInstalledTitle, out)
	}
	if !strings.Contains(out, "dns") || !strings.Contains(out, "Manage DNS zones and records") {
		t.Errorf("installed plugin and its description missing from:\n%s", out)
	}
	if !strings.Contains(out, "\n"+helpAvailableTitle+"\n") {
		t.Fatalf("expected an %q section in:\n%s", helpAvailableTitle, out)
	}
	if !strings.Contains(out, "costs") {
		t.Errorf("uninstalled catalog plugin missing from:\n%s", out)
	}
	// An installed plugin is listed once, under Plugins — never offered again.
	if available := out[strings.Index(out, helpAvailableTitle):]; strings.Contains(available, "dns") {
		t.Errorf("installed plugin should not appear under %q:\n%s", helpAvailableTitle, out)
	}
	if !strings.Contains(out, "datumctl plugin install <name>") {
		t.Errorf("expected the install instruction in:\n%s", out)
	}
}

func TestPluginHelpSections_describesInstalledPluginWithNoManifest(t *testing.T) {
	// A plugin installed before manifests were recorded still deserves a row
	// with something useful in the description column.
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{
		"costs": {Catalog: "acme", Version: "v2.0.0"},
	})

	out := pluginHelpSections()

	if !strings.Contains(out, "costs") {
		t.Fatalf("expected the installed plugin in:\n%s", out)
	}
	if !strings.Contains(out, "acme catalog") {
		t.Errorf("expected a catalog fallback description in:\n%s", out)
	}
}

func TestPluginHelpSections_capsAvailableList(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{})
	many := make([]pluginstore.Plugin, 0, helpAvailableLimit+2)
	for _, name := range []string{
		"p01", "p02", "p03", "p04", "p05", "p06", "p07", "p08", "p09", "p10", "p11", "p12",
	} {
		many = append(many, catalogPlugin(name, "does a thing"))
	}
	seedCatalogIndex(t, pluginstore.OfficialCatalogName, many...)

	out := pluginHelpSections()

	if strings.Contains(out, "p11") || strings.Contains(out, "p12") {
		t.Errorf("entries past the cap should not be listed:\n%s", out)
	}
	if !strings.Contains(out, "+2 more") {
		t.Errorf("expected the remaining count in:\n%s", out)
	}
}

func TestPluginHelpSections_emptyWhenNoPlugins(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{})

	if out := pluginHelpSections(); out != "" {
		t.Fatalf("expected no sections with no plugins anywhere, got:\n%s", out)
	}
}

func TestInstallPluginsHelpSections_rendersInUsage(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{
		"dns": {
			Catalog:  pluginstore.OfficialCatalogName,
			Manifest: &pluginstore.PluginManifest{Description: "Manage DNS zones and records"},
		},
	})

	root := &cobra.Command{Use: "datumctl"}
	root.AddCommand(&cobra.Command{Use: "get", Short: "List resources", Run: func(*cobra.Command, []string) {}})
	root.Flags().Bool("example", false, "an example flag")
	installPluginsHelpSections(root)

	usage := root.UsageString()
	if !strings.Contains(usage, helpInstalledTitle) || !strings.Contains(usage, "dns") {
		t.Fatalf("plugin section missing from usage output:\n%s", usage)
	}
	// The section belongs between the commands and the flags, where a reader
	// looks for things to run.
	if got, want := strings.Index(usage, helpInstalledTitle), strings.Index(usage, "\nFlags:"); got > want {
		t.Errorf("plugin section should precede the flag list:\n%s", usage)
	}
	if !strings.Contains(usage, "List resources") {
		t.Errorf("splicing dropped Cobra's own command listing:\n%s", usage)
	}
}

func TestInstallPluginsHelpSections_leavesUnknownTemplateAlone(t *testing.T) {
	const custom = "a template with no flags anchor\n"

	root := &cobra.Command{Use: "datumctl"}
	root.SetUsageTemplate(custom)
	installPluginsHelpSections(root)

	if got := root.UsageTemplate(); got != custom {
		t.Fatalf("template without the anchor must be left untouched, got:\n%s", got)
	}
}

func TestRpad(t *testing.T) {
	if got := rpad("dns", 6); got != "dns   " {
		t.Errorf("rpad(%q, 6) = %q", "dns", got)
	}
	// Longer than the column: never truncate a command name.
	if got := rpad("a-very-long-plugin-name", 6); got != "a-very-long-plugin-name" {
		t.Errorf("rpad should not truncate, got %q", got)
	}
}
