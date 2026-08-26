package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.datum.net/datumctl/internal/pluginstore"
)

func seedPluginsManifest(t *testing.T, plugins map[string]*pluginstore.InstalledPlugin) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("DATUMCTL_PLUGINS_DIR", dir)
	data, err := json.MarshalIndent(pluginstore.Manifest{Plugins: plugins}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugins.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPrintInstalledPlugins_listsInstalledAsCommands(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{
		// Catalog "default" is the official catalog under its legacy alias; it must
		// render with the canonical product-facing name "datum".
		"dns":  {Catalog: "default", Version: "v1.2.3", Source: "datum"},
		"ipam": {Catalog: "acme", Version: "v0.4.0", Source: "https://plugins.acme.example/index.yaml"},
		// Legacy record with no catalog and a slashed source -> "direct".
		"costs": {Source: "octo/datumctl-costs", Version: "v2.0.0"},
	})

	var buf bytes.Buffer
	printInstalledPlugins(&buf)
	out := buf.String()

	if !strings.Contains(out, "Installed Plugins") {
		t.Fatalf("expected an Installed Plugins block, got:\n%s", out)
	}
	// Each plugin is shown as a runnable `datumctl <command>` verb.
	for _, cmd := range []string{"datumctl dns", "datumctl ipam", "datumctl costs"} {
		if !strings.Contains(out, cmd) {
			t.Errorf("expected %q in output:\n%s", cmd, out)
		}
	}
	// Provenance is labelled, with the legacy "default" alias canonicalized.
	for _, src := range []string{"(datum)", "(acme)", "(direct)"} {
		if !strings.Contains(out, src) {
			t.Errorf("expected source label %q in output:\n%s", src, out)
		}
	}
	if strings.Contains(out, "(default)") {
		t.Errorf("legacy 'default' catalog should render as 'datum', not 'default':\n%s", out)
	}
}

func TestPrintInstalledPlugins_emptyWhenNoneInstalled(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{})

	var buf bytes.Buffer
	printInstalledPlugins(&buf)

	if buf.Len() != 0 {
		t.Fatalf("expected no output when no plugins are installed, got:\n%s", buf.String())
	}
}

func TestPrintInstalledPlugins_emptyWhenStoreUnreadable(t *testing.T) {
	// Point the store at a path that cannot be a valid plugins dir; the landing
	// must degrade silently rather than emit a partial or erroring block.
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATUMCTL_PLUGINS_DIR", filepath.Join(file, "nested"))

	var buf bytes.Buffer
	printInstalledPlugins(&buf)

	if buf.Len() != 0 {
		t.Fatalf("expected no output when the plugin store is unreadable, got:\n%s", buf.String())
	}
}

func TestLandingPluginSource(t *testing.T) {
	cases := []struct {
		name  string
		entry *pluginstore.InstalledPlugin
		want  string
	}{
		{"nil", nil, ""},
		{"catalog", &pluginstore.InstalledPlugin{Catalog: "acme"}, "acme"},
		{"legacy default alias", &pluginstore.InstalledPlugin{Catalog: "default"}, "datum"},
		{"legacy direct", &pluginstore.InstalledPlugin{Source: "octo/repo"}, "direct"},
		{"legacy official", &pluginstore.InstalledPlugin{Source: "datum"}, pluginstore.OfficialCatalogName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := landingPluginSource(tc.entry); got != tc.want {
				t.Errorf("landingPluginSource = %q, want %q", got, tc.want)
			}
		})
	}
}

// seedCatalogIndex writes a cached catalog index into the plugins dir already
// pointed at by DATUMCTL_PLUGINS_DIR, so the landing reads it without a fetch.
func seedCatalogIndex(t *testing.T, catalog string, plugins ...pluginstore.Plugin) {
	t.Helper()
	dir := os.Getenv("DATUMCTL_PLUGINS_DIR")
	if dir == "" {
		t.Fatal("seedCatalogIndex requires DATUMCTL_PLUGINS_DIR to be set")
	}
	path, err := pluginstore.CatalogIndexPath(dir, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(pluginstore.CachedIndex{
		RefreshedAt: time.Now(),
		Plugins:     plugins,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func catalogPlugin(name, desc string) pluginstore.Plugin {
	return pluginstore.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       pluginstore.PluginSpec{ShortDescription: desc},
	}
}

func TestPrintAvailablePlugins_listsUninstalledOnly(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{
		"dns": {Catalog: pluginstore.OfficialCatalogName, Version: "v1.2.3"},
	})
	seedCatalogIndex(t, pluginstore.OfficialCatalogName,
		catalogPlugin("dns", "Manage DNS zones"),
		catalogPlugin("costs", "Show spend by project"),
	)

	var buf bytes.Buffer
	printAvailablePlugins(&buf, map[string]bool{"dns": true})
	out := buf.String()

	if !strings.Contains(out, "Available plugins") {
		t.Fatalf("expected an Available plugins block, got:\n%s", out)
	}
	if !strings.Contains(out, "costs") || !strings.Contains(out, "Show spend by project") {
		t.Errorf("expected the uninstalled plugin and its description in:\n%s", out)
	}
	// An already-installed plugin belongs to "Installed Plugins", never here.
	if strings.Contains(out, "dns") {
		t.Errorf("installed plugin should not be offered as available:\n%s", out)
	}
	if !strings.Contains(out, "datumctl plugin install <name>") {
		t.Errorf("expected the install instruction in:\n%s", out)
	}
}

func TestPrintAvailablePlugins_capsAndCountsTheRest(t *testing.T) {
	seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{})
	seedCatalogIndex(t, pluginstore.OfficialCatalogName,
		catalogPlugin("alpha", ""),
		catalogPlugin("bravo", ""),
		catalogPlugin("charlie", ""),
		catalogPlugin("delta", ""),
		catalogPlugin("echo", ""),
	)

	var buf bytes.Buffer
	printAvailablePlugins(&buf, nil)
	out := buf.String()

	// Named alphabetically up to the cap...
	for _, name := range []string{"alpha", "bravo", "charlie"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in:\n%s", name, out)
		}
	}
	// ...and the overflow is counted, not silently dropped.
	for _, name := range []string{"delta", "echo"} {
		if strings.Contains(out, name) {
			t.Errorf("%q is past the cap and should not be listed:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "+2 more") {
		t.Errorf("expected the remaining count in:\n%s", out)
	}
}

func TestPrintAvailablePlugins_emptyWhenNothingToOffer(t *testing.T) {
	t.Run("cold cache", func(t *testing.T) {
		seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{})

		var buf bytes.Buffer
		printAvailablePlugins(&buf, nil)
		if buf.Len() != 0 {
			t.Fatalf("expected no output with no cached catalog, got:\n%s", buf.String())
		}
	})

	t.Run("everything already installed", func(t *testing.T) {
		seedPluginsManifest(t, map[string]*pluginstore.InstalledPlugin{
			"dns": {Catalog: pluginstore.OfficialCatalogName},
		})
		seedCatalogIndex(t, pluginstore.OfficialCatalogName, catalogPlugin("dns", "Manage DNS zones"))

		var buf bytes.Buffer
		printAvailablePlugins(&buf, map[string]bool{"dns": true})
		if buf.Len() != 0 {
			t.Fatalf("expected no output when the catalog holds nothing new, got:\n%s", buf.String())
		}
	})
}

func TestElide(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"  padded  ", 10, "padded"},
		{"exactly-10", 10, "exactly-10"},
		{"this one runs long", 10, "this one…"},
		// No room for even one character plus the ellipsis: drop it entirely
		// rather than print a bare "…".
		{"anything", 1, ""},
		{"anything", 0, ""},
	}
	for _, tc := range cases {
		if got := elide(tc.in, tc.max); got != tc.want {
			t.Errorf("elide(%q, %d) = %q, want %q", tc.in, tc.max, got, tc.want)
		}
	}
}
