package plugin

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/pluginstore"
)

func testPlugin(name, version, desc string) pluginstore.Plugin {
	var p pluginstore.Plugin
	p.Name = name
	p.Spec = pluginstore.PluginSpec{ShortDescription: desc, Version: version}
	return p
}

// seedFreshCache writes a non-stale cached index for a catalog so resolution and
// search never hit the network in tests.
func seedFreshCache(t *testing.T, pluginsDir, name string, plugins ...pluginstore.Plugin) {
	t.Helper()
	idx := &pluginstore.CachedIndex{RefreshedAt: time.Now(), Plugins: plugins}
	if err := pluginstore.SaveCatalogIndex(pluginsDir, name, idx); err != nil {
		t.Fatal(err)
	}
}

func execPluginCmd(t *testing.T, pluginsDir string, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()
	t.Setenv("DATUMCTL_PLUGINS_DIR", pluginsDir)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestInstallBadge(t *testing.T) {
	if installBadge(pluginstore.OfficialCatalogName) != pluginstore.TrustOfficial {
		t.Fatal("datum catalog must be official")
	}
	// The legacy "default" alias is also official.
	if installBadge("default") != pluginstore.TrustOfficial {
		t.Fatal("legacy default alias must be official")
	}
	if installBadge("acme") != pluginstore.TrustThirdParty {
		t.Fatal("non-official catalog must be third-party")
	}
}

func TestInstalledCatalogLabel(t *testing.T) {
	cases := []struct {
		name      string
		entry     pluginstore.InstalledPlugin
		wantIndex string
		wantTrust string
	}{
		{"catalog datum", pluginstore.InstalledPlugin{Catalog: "datum"}, "datum", "official"},
		// A record written under the legacy "default" name displays as "datum".
		{"legacy default alias", pluginstore.InstalledPlugin{Catalog: "default"}, "datum", "official"},
		{"catalog acme", pluginstore.InstalledPlugin{Catalog: "acme"}, "acme", "third-party"},
		{"legacy github direct", pluginstore.InstalledPlugin{Source: "github.com/o/r"}, "(direct)", "third-party"},
		{"legacy curated", pluginstore.InstalledPlugin{Source: "dns"}, "datum", "official"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx, trust := installedCatalogLabel(&tc.entry)
			if idx != tc.wantIndex || trust != tc.wantTrust {
				t.Fatalf("got (%q,%q), want (%q,%q)", idx, trust, tc.wantIndex, tc.wantTrust)
			}
		})
	}
}

func TestResolveBareName_uniqueAndCollision(t *testing.T) {
	pluginsDir := t.TempDir()
	// Register two custom catalogs.
	if err := pluginstore.SaveRegistry(pluginsDir, &pluginstore.Registry{Catalogs: []pluginstore.Catalog{
		{Name: "acme", Source: "https://acme.example/index.yaml", Type: pluginstore.CatalogTypeCustom},
		{Name: "beta", Source: "https://beta.example/index.yaml", Type: pluginstore.CatalogTypeCustom},
	}}); err != nil {
		t.Fatal(err)
	}
	// datum + acme both contain "deploy"; beta contains "solo".
	seedFreshCache(t, pluginsDir, "datum", testPlugin("deploy", "v1.0.0", "Datum guided deploy"))
	seedFreshCache(t, pluginsDir, "acme", testPlugin("deploy", "v2.1.0", "ACME guided deploy"))
	seedFreshCache(t, pluginsDir, "beta", testPlugin("solo", "v0.1.0", "Solo plugin"))

	reg, err := pluginstore.LoadRegistry(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	var errOut bytes.Buffer
	cmd.SetErr(&errOut)

	// Collision: deploy is in datum and acme; the official datum catalog is first.
	matches := resolveBareName(cmd, pluginsDir, reg, "deploy")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for deploy, got %d", len(matches))
	}
	if matches[0].catalog.Name != "datum" {
		t.Fatalf("datum should be first match, got %q", matches[0].catalog.Name)
	}
	collision := collisionError("deploy", matches).Error()
	if !strings.Contains(collision, "multiple catalogs") ||
		!strings.Contains(collision, "datum/deploy") ||
		!strings.Contains(collision, "acme/deploy") ||
		!strings.Contains(collision, "(official)") ||
		!strings.Contains(collision, "(third-party)") {
		t.Fatalf("collision message missing detail: %s", collision)
	}

	// Unique: solo is only in beta.
	solo := resolveBareName(cmd, pluginsDir, reg, "solo")
	if len(solo) != 1 || solo[0].catalog.Name != "beta" {
		t.Fatalf("expected unique beta match for solo, got %+v", solo)
	}

	// Missing: nothing matches.
	if got := resolveBareName(cmd, pluginsDir, reg, "ghost"); len(got) != 0 {
		t.Fatalf("expected no matches for ghost, got %+v", got)
	}
}

func TestSearch_acrossCatalogsAndScope(t *testing.T) {
	pluginsDir := t.TempDir()
	if err := pluginstore.SaveRegistry(pluginsDir, &pluginstore.Registry{Catalogs: []pluginstore.Catalog{
		{Name: "community", Source: "https://community.example/index.yaml", Type: pluginstore.CatalogTypeCustom},
	}}); err != nil {
		t.Fatal(err)
	}
	seedFreshCache(t, pluginsDir, "datum", testPlugin("dns", "v1.2.3", "Manage Datum Cloud DNS zones"))
	seedFreshCache(t, pluginsDir, "community", testPlugin("zonex", "v2.1.0", "Bulk zone import/export"))

	// Unscoped search shows both catalogs with trust badges.
	out, err := execPluginCmd(t, pluginsDir, searchCmd())
	if err != nil {
		t.Fatalf("search failed: %v (out=%s)", err, out)
	}
	for _, want := range []string{"NAME", "INDEX", "TRUST", "dns", "datum", "official", "zonex", "community", "third-party"} {
		if !strings.Contains(out, want) {
			t.Fatalf("search output missing %q:\n%s", want, out)
		}
	}

	// Scoped search shows only the named catalog.
	scoped, err := execPluginCmd(t, pluginsDir, searchCmd(), "--index", "community")
	if err != nil {
		t.Fatalf("scoped search failed: %v", err)
	}
	if !strings.Contains(scoped, "zonex") || strings.Contains(scoped, "dns") {
		t.Fatalf("scoped search should only show community plugins:\n%s", scoped)
	}

	// Query filter (matches zonex description "import/export" only).
	q, err := execPluginCmd(t, pluginsDir, searchCmd(), "import")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(q, "zonex") || strings.Contains(q, "dns") {
		t.Fatalf("query filter failed:\n%s", q)
	}
}

func TestSearch_unknownIndexErrors(t *testing.T) {
	pluginsDir := t.TempDir()
	if _, err := execPluginCmd(t, pluginsDir, searchCmd(), "--index", "nope"); err == nil {
		t.Fatal("expected error for unknown --index")
	}
}
