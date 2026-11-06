package plugin

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

func TestPlanCatalogInstall(t *testing.T) {
	t.Parallel()

	idx := &pluginstore.CachedIndex{Plugins: []pluginstore.Plugin{
		testPlugin("dns", "v1.2.0", "DNS"),
		testPlugin("compute", "v2.0.0", "Compute"),
		testPlugin("alb", "v0.5.0", "ALB"),
		testPlugin("search", "v0.1.0", "Search"),
		testPlugin("ipam", "v0.3.0", "IPAM"),
	}}
	manifest := &pluginstore.Manifest{Plugins: map[string]*pluginstore.InstalledPlugin{
		// Up to date from the official catalog.
		"dns": {Source: "dns", Catalog: "datum", Version: "v1.2.0"},
		// Behind the catalog: upgrade.
		"compute": {Source: "compute", Catalog: "datum", Version: "v1.9.0"},
		// Same name from a third-party catalog: leave it.
		"alb": {Source: "alb", Catalog: "acme", Version: "v9.0.0"},
		// Direct GitHub install: leave it.
		"search": {Source: "github.com/someone/datumctl-search", Version: "v0.0.1"},
		// "ipam" is not installed: install.
	}}

	got := planCatalogInstall("datum", idx, manifest)

	if want := []string{"compute", "ipam"}; !reflect.DeepEqual(got.install, want) {
		t.Errorf("install = %v, want %v", got.install, want)
	}
	if want := []string{"dns"}; !reflect.DeepEqual(got.upToDate, want) {
		t.Errorf("upToDate = %v, want %v", got.upToDate, want)
	}
	wantSkipped := []skippedPlugin{
		{name: "alb", reason: "already installed from acme"},
		{name: "search", reason: "already installed from (direct)"},
	}
	if !reflect.DeepEqual(got.skipped, wantSkipped) {
		t.Errorf("skipped = %+v, want %+v", got.skipped, wantSkipped)
	}
}

func TestPlanCatalogInstall_legacyRecordsCountAsOfficial(t *testing.T) {
	t.Parallel()

	idx := &pluginstore.CachedIndex{Plugins: []pluginstore.Plugin{testPlugin("dns", "v1.0.0", "DNS")}}
	manifest := &pluginstore.Manifest{Plugins: map[string]*pluginstore.InstalledPlugin{
		// Recorded before the catalog field existed.
		"dns": {Source: "dns", Version: "v1.0.0"},
	}}

	got := planCatalogInstall("datum", idx, manifest)
	if len(got.install) != 0 || len(got.skipped) != 0 || !reflect.DeepEqual(got.upToDate, []string{"dns"}) {
		t.Fatalf("legacy official record should be up to date, got %+v", got)
	}
}

func TestInstallAllAvailable_upToDateNeedsNoDownload(t *testing.T) {
	pluginsDir := t.TempDir()
	seedFreshCache(t, pluginsDir, "datum", testPlugin("dns", "v1.2.0", "DNS"))
	if err := pluginstore.Save(pluginsDir, &pluginstore.Manifest{Plugins: map[string]*pluginstore.InstalledPlugin{
		"dns": {Source: "dns", Catalog: "datum", Version: "v1.2.0"},
	}}); err != nil {
		t.Fatal(err)
	}

	out, err := execPluginCmd(t, pluginsDir, installCmd(nil), "--all-available")
	if err != nil {
		t.Fatalf("install --all-available failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "dns v1.2.0 is up to date") {
		t.Fatalf("expected an up-to-date line, got:\n%s", out)
	}
}

func TestInstallAllAvailable_emptyCatalog(t *testing.T) {
	pluginsDir := t.TempDir()
	seedFreshCache(t, pluginsDir, "datum")

	out, err := execPluginCmd(t, pluginsDir, installCmd(nil), "--all-available")
	if err != nil {
		t.Fatalf("install --all-available failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Catalog datum has no plugins.") {
		t.Fatalf("expected an empty-catalog message, got:\n%s", out)
	}
}

func TestInstallAllAvailable_flagErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "plugin name with --all-available", args: []string{"--all-available", "dns"}, want: "does not take a plugin name"},
		{name: "--index without --all-available", args: []string{"--index", "datum", "dns"}, want: "--index only applies with --all-available"},
		{name: "unregistered catalog", args: []string{"--all-available", "--index", "nope"}, want: `catalog "nope" is not registered`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := execPluginCmd(t, t.TempDir(), installCmd(nil), tt.args...)
			if err == nil {
				t.Fatalf("expected an error, got output:\n%s", out)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not mention %q", err.Error(), tt.want)
			}
		})
	}
}

func TestSearch_outputFormats(t *testing.T) {
	pluginsDir := t.TempDir()
	seedFreshCache(t, pluginsDir, "datum",
		testPlugin("dns", "v1.2.3", "Manage DNS"),
		testPlugin("compute", "v2.0.0", "Run workloads"),
	)

	jsonOut, err := execPluginCmd(t, pluginsDir, searchCmd(), "--index", "datum", "-o", "json")
	if err != nil {
		t.Fatalf("search -o json failed: %v", err)
	}
	var results []searchResult
	if err := json.Unmarshal([]byte(jsonOut), &results); err != nil {
		t.Fatalf("search -o json is not a JSON array: %v\n%s", err, jsonOut)
	}
	want := []searchResult{
		{Name: "dns", Index: "datum", Version: "v1.2.3", Trust: pluginstore.TrustOfficial, Description: "Manage DNS"},
		{Name: "compute", Index: "datum", Version: "v2.0.0", Trust: pluginstore.TrustOfficial, Description: "Run workloads"},
	}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("search -o json = %+v, want %+v", results, want)
	}

	nameOut, err := execPluginCmd(t, pluginsDir, searchCmd(), "--index", "datum", "-o", "name")
	if err != nil {
		t.Fatalf("search -o name failed: %v", err)
	}
	if nameOut != "dns\ncompute\n" {
		t.Fatalf("search -o name = %q", nameOut)
	}

	emptyOut, err := execPluginCmd(t, pluginsDir, searchCmd(), "nothing-matches", "-o", "json")
	if err != nil {
		t.Fatalf("search -o json with no matches failed: %v", err)
	}
	if strings.TrimSpace(emptyOut) != "[]" {
		t.Fatalf("no matches should print [], got %q", emptyOut)
	}

	if _, err := execPluginCmd(t, pluginsDir, searchCmd(), "-o", "xml"); err == nil {
		t.Fatal("expected an error for -o xml")
	}
}
