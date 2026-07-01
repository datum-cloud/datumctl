package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	if !strings.Contains(out, "Your plugins") {
		t.Fatalf("expected a Your plugins block, got:\n%s", out)
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
