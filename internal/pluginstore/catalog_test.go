package pluginstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidateCatalogName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "acme", false},
		{"valid hyphen", "acme-internal", false},
		{"valid digit start", "1catalog", false},
		{"reserved datum", "datum", true},
		{"reserved default", "default", true},
		{"reserved official", "official", true},
		{"empty", "", true},
		{"uppercase", "Acme", true},
		{"leading hyphen", "-acme", true},
		{"path traversal", "../evil", true},
		{"slash", "a/b", true},
		{"dot", "a.b", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCatalogName(tc.input)
			if tc.wantErr != (err != nil) {
				t.Fatalf("ValidateCatalogName(%q) err=%v, wantErr=%v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestCatalogIndexPath_rejectsTraversal(t *testing.T) {
	if _, err := CatalogIndexPath("/tmp/plugins", "../../etc"); err == nil {
		t.Fatal("expected error for traversal catalog name")
	}
	p, err := CatalogIndexPath("/tmp/plugins", "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("/tmp/plugins", "indexes", "acme", "index.json")
	if p != want {
		t.Fatalf("got %q, want %q", p, want)
	}
}

func TestResolveCatalogSource(t *testing.T) {
	dir := t.TempDir()
	manifestDir := filepath.Join(dir, "catalog")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestFile := filepath.Join(manifestDir, "index.yaml")
	if err := os.WriteFile(manifestFile, []byte("items: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("https url", func(t *testing.T) {
		got, err := ResolveCatalogSource("https://plugins.acme.example/index.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if got.IsLocal || got.FetchURL != "https://plugins.acme.example/index.yaml" {
			t.Fatalf("unexpected: %+v", got)
		}
	})

	t.Run("http rejected", func(t *testing.T) {
		if _, err := ResolveCatalogSource("http://plugins.acme.example/index.yaml"); err == nil {
			t.Fatal("expected HTTPS-only error")
		}
	})

	t.Run("owner/repo shorthand", func(t *testing.T) {
		got, err := ResolveCatalogSource("priya/datumctl-plugins")
		if err != nil {
			t.Fatal(err)
		}
		if got.GitHubOwnerRepo != "priya/datumctl-plugins" {
			t.Fatalf("unexpected owner/repo: %+v", got)
		}
		want := "https://raw.githubusercontent.com/priya/datumctl-plugins/main/index.yaml"
		if got.FetchURL != want {
			t.Fatalf("got %q, want %q", got.FetchURL, want)
		}
	})

	t.Run("github.com prefix", func(t *testing.T) {
		got, err := ResolveCatalogSource("github.com/priya/datumctl-plugins")
		if err != nil {
			t.Fatal(err)
		}
		if got.GitHubOwnerRepo != "priya/datumctl-plugins" {
			t.Fatalf("unexpected: %+v", got)
		}
	})

	t.Run("local directory appends index.yaml", func(t *testing.T) {
		got, err := ResolveCatalogSource(manifestDir)
		if err != nil {
			t.Fatal(err)
		}
		if !got.IsLocal || got.LocalPath != manifestFile {
			t.Fatalf("unexpected: %+v (want %q)", got, manifestFile)
		}
	})

	t.Run("relative local path", func(t *testing.T) {
		got, err := ResolveCatalogSource("./" + filepath.Base(manifestDir))
		// Relative to cwd; may not exist. Just assert it is treated as local form.
		if err == nil && !got.IsLocal {
			t.Fatalf("expected local treatment, got %+v", got)
		}
	})

	t.Run("unrecognized", func(t *testing.T) {
		if _, err := ResolveCatalogSource("not a valid source!!!"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCanonicalCatalogName_aliasResolvesToDatum(t *testing.T) {
	if OfficialCatalogName != "datum" {
		t.Fatalf("official catalog should be named datum, got %q", OfficialCatalogName)
	}
	if CanonicalCatalogName("default") != "datum" {
		t.Fatal(`"default" must canonicalize to "datum"`)
	}
	if CanonicalCatalogName("datum") != "datum" || CanonicalCatalogName("acme") != "acme" {
		t.Fatal("non-alias names must pass through unchanged")
	}

	dir := t.TempDir()
	reg, err := LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Both the canonical name and the legacy alias resolve to the same official
	// catalog, named "datum".
	byDatum := reg.Find("datum")
	byDefault := reg.Find("default")
	if byDatum == nil || byDefault == nil {
		t.Fatalf("both datum and default must resolve, got %v / %v", byDatum, byDefault)
	}
	if byDatum.Name != "datum" || byDefault.Name != "datum" {
		t.Fatalf("alias should resolve to datum, got %q / %q", byDatum.Name, byDefault.Name)
	}
}

func TestLoadCatalogIndex_defaultAliasSharesDatumCache(t *testing.T) {
	dir := t.TempDir()
	idx := &CachedIndex{RefreshedAt: time.Now(), Plugins: []Plugin{{Spec: PluginSpec{Version: "v1.0.0"}}}}
	// Save under the legacy "default" name; it must land in the datum cache.
	if err := SaveCatalogIndex(dir, "default", idx); err != nil {
		t.Fatal(err)
	}
	cachePath, _ := CatalogIndexPath(dir, "datum")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("default save should write the datum cache: %v", err)
	}
	// Read back under either name.
	for _, name := range []string{"datum", "default"} {
		got, err := LoadCatalogIndex(dir, name)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Plugins) != 1 {
			t.Fatalf("reading %q should return the shared cache, got %+v", name, got)
		}
	}
}

func TestLoadCatalogIndex_readsPreRenameDefaultDir(t *testing.T) {
	dir := t.TempDir()
	// Simulate a pre-rename cache written under indexes/default/index.json by an
	// older datumctl, with no datum cache present.
	legacyDir := filepath.Join(dir, "indexes", "default")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	idx := &CachedIndex{RefreshedAt: time.Now(), Plugins: []Plugin{{Spec: PluginSpec{Version: "v7.7.7"}}}}
	data, _ := json.Marshal(idx)
	if err := os.WriteFile(filepath.Join(legacyDir, "index.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalogIndex(dir, "datum")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Plugins) != 1 || got.Plugins[0].Spec.Version != "v7.7.7" {
		t.Fatalf("pre-rename default cache not read: %+v", got)
	}
}

func TestRegistry_LoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()

	reg, err := LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Default must be present and first.
	if len(reg.Catalogs) != 1 || reg.Catalogs[0].Name != OfficialCatalogName {
		t.Fatalf("expected only default catalog, got %+v", reg.Catalogs)
	}
	if !reg.Catalogs[0].IsOfficial() || reg.Catalogs[0].Trust() != TrustOfficial {
		t.Fatalf("default catalog should be official: %+v", reg.Catalogs[0])
	}

	reg.Catalogs = append(reg.Catalogs, Catalog{
		Name:   "acme",
		Source: "https://plugins.acme.example/index.yaml",
		Type:   CatalogTypeCustom,
	})
	if err := SaveRegistry(dir, reg); err != nil {
		t.Fatal(err)
	}

	// Reload: default synthesized + acme persisted.
	reg2, err := LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	if reg2.Find("acme") == nil {
		t.Fatal("acme not persisted")
	}
	if reg2.Find("default") == nil {
		t.Fatal("default missing after reload")
	}
	if reg2.Catalogs[0].Name != OfficialCatalogName {
		t.Fatalf("default should be first, got %q", reg2.Catalogs[0].Name)
	}
	if got := reg2.Custom(); len(got) != 1 || got[0].Name != "acme" {
		t.Fatalf("Custom() should be [acme], got %+v", got)
	}

	// indexes.json on disk must not contain the default catalog.
	data, err := os.ReadFile(IndexesPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" {
		t.Fatal("empty registry file")
	}
	if strings.Contains(string(data), `"default"`) {
		t.Fatalf("default catalog should not be persisted, file: %s", data)
	}
}

func TestRegistry_ManagedPreseedPrecedence(t *testing.T) {
	dir := t.TempDir()

	// Persist a user "acme" first.
	if err := SaveRegistry(dir, &Registry{Catalogs: []Catalog{
		{Name: "acme", Source: "https://user.example/index.yaml", Type: CatalogTypeCustom},
		{Name: "extra", Source: "https://extra.example/index.yaml", Type: CatalogTypeCustom},
	}}); err != nil {
		t.Fatal(err)
	}

	// Managed config pre-seeds "acme" pointing elsewhere; it must win.
	mc := filepath.Join(dir, "managed.yaml")
	if err := os.WriteFile(mc, []byte(
		"indexes:\n  - name: acme\n    source: https://managed.example/index.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(managedConfigEnvVar, mc)

	reg, err := LoadRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	acme := reg.Find("acme")
	if acme == nil {
		t.Fatal("acme missing")
	}
	if !acme.Managed || acme.Source != "https://managed.example/index.yaml" {
		t.Fatalf("managed acme should win: %+v", acme)
	}
	// The user "extra" still shows.
	if reg.Find("extra") == nil {
		t.Fatal("extra missing")
	}
	// Custom() excludes managed acme but includes extra.
	for _, c := range reg.Custom() {
		if c.Name == "acme" {
			t.Fatal("managed acme should not be in Custom()")
		}
	}
}

func TestCatalogIndex_SaveLoadAndLegacyFallback(t *testing.T) {
	dir := t.TempDir()

	idx := &CachedIndex{
		Header:  CatalogHeader{Name: "acme", Description: "ACME tooling"},
		Plugins: []Plugin{{Spec: PluginSpec{Version: "v1.0.0"}}},
	}
	idx.RefreshedAt = time.Now().UTC()
	if err := SaveCatalogIndex(dir, "acme", idx); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalogIndex(dir, "acme")
	if err != nil {
		t.Fatal(err)
	}
	if got.Header.Description != "ACME tooling" || len(got.Plugins) != 1 {
		t.Fatalf("roundtrip mismatch: %+v", got)
	}

	// Legacy fallback: default catalog reads plugin-index.json when the new
	// per-catalog cache is absent.
	legacy := &CachedIndex{Plugins: []Plugin{{Spec: PluginSpec{Version: "v9.9.9"}}}}
	legacy.RefreshedAt = time.Now().UTC()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(legacy)
	if err := os.WriteFile(filepath.Join(dir, legacyIndexFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	def, err := LoadCatalogIndex(dir, OfficialCatalogName)
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Plugins) != 1 || def.Plugins[0].Spec.Version != "v9.9.9" {
		t.Fatalf("legacy fallback failed: %+v", def)
	}
}
