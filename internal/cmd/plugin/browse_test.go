package plugin

import (
	"strconv"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

func browseEntryFor(catalog pluginstore.Catalog, name, version, short, long, homepage string) browseEntry {
	p := testPlugin(name, version, short)
	p.Spec.Description = long
	p.Spec.Homepage = homepage
	return browseEntry{catalog: catalog, plugin: p}
}

func TestBrowseOptions_labelsAndIndexValues(t *testing.T) {
	entries := []browseEntry{
		browseEntryFor(pluginstore.DefaultCatalog(), "dns", "v1.2.3", "Manage DNS zones", "", ""),
		browseEntryFor(pluginstore.Catalog{Name: "acme", Type: pluginstore.CatalogTypeCustom}, "deploy", "v2.1.0", "ACME deploy", "", ""),
	}
	opts := browseOptions(entries)
	if len(opts) != 2 {
		t.Fatalf("expected 2 options, got %d", len(opts))
	}
	// Values must be the entry indices so selection maps back unambiguously.
	if opts[0].Value != "0" || opts[1].Value != "1" {
		t.Fatalf("option values should be indices, got %q,%q", opts[0].Value, opts[1].Value)
	}
	// Labels carry name, trust badge, and catalog.
	if !strings.Contains(opts[0].Key, "dns") || !strings.Contains(opts[0].Key, "official") {
		t.Fatalf("default option label missing detail: %q", opts[0].Key)
	}
	if !strings.Contains(opts[1].Key, "deploy") || !strings.Contains(opts[1].Key, "third-party") || !strings.Contains(opts[1].Key, "acme") {
		t.Fatalf("acme option label missing detail: %q", opts[1].Key)
	}
	// Every value must parse to a valid index.
	for _, o := range opts {
		if _, err := strconv.Atoi(o.Value); err != nil {
			t.Fatalf("non-numeric option value %q", o.Value)
		}
	}
}

func TestBrowseDetails_prefersLongDescriptionAndShowsHomepage(t *testing.T) {
	e := browseEntryFor(
		pluginstore.Catalog{Name: "acme", Type: pluginstore.CatalogTypeCustom},
		"deploy", "v2.1.0", "short", "the long description", "https://plugins.acme.example/deploy")
	got := browseDetails(e)
	for _, want := range []string{"acme", "v2.1.0", "third-party", "https://plugins.acme.example/deploy", "the long description"} {
		if !strings.Contains(got, want) {
			t.Fatalf("details missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\nshort") {
		t.Fatalf("long description should override short:\n%s", got)
	}
}

func TestBrowse_requiresTerminal(t *testing.T) {
	// Tests run without a TTY, so browse must return the terminal-required error
	// rather than attempting to render the TUI.
	pluginsDir := t.TempDir()
	seedFreshCache(t, pluginsDir, "default", testPlugin("dns", "v1.2.3", "Manage DNS zones"))
	out, err := execPluginCmd(t, pluginsDir, browseCmd())
	if err == nil {
		t.Fatalf("expected terminal-required error, got output: %s", out)
	}
	if !strings.Contains(err.Error(), "terminal") {
		t.Fatalf("unexpected error: %v", err)
	}
}
