package pluginstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sigs.k8s.io/yaml"
)

// backwardCompatManifest is an old, headerless catalog manifest as authored
// before the marketplace feature: no catalog-level name/description/owner, just
// a plugin list. It must still parse and produce usable plugins.
const backwardCompatManifest = `apiVersion: plugins.datumctl.dev/v1
kind: PluginList
items:
  - metadata:
      name: dns
    spec:
      shortDescription: Manage Datum Cloud DNS zones
      version: v1.2.3
      platforms:
        - uri: https://example.com/dns_linux_amd64.tar.gz
          sha256: "abc"
`

func TestPluginList_BackwardCompatibleParse(t *testing.T) {
	var list PluginList
	if err := yaml.Unmarshal([]byte(backwardCompatManifest), &list); err != nil {
		t.Fatalf("old manifest must still parse: %v", err)
	}
	if list.Name != "" || list.Description != "" {
		t.Fatalf("headerless manifest should have empty header: %+v", list.HeaderFor())
	}
	if len(list.Items) != 1 || list.Items[0].Name != "dns" {
		t.Fatalf("expected one plugin named dns, got %+v", list.Items)
	}
	if list.Items[0].Spec.Version != "v1.2.3" {
		t.Fatalf("version not parsed: %+v", list.Items[0].Spec)
	}
}

// catalogManifestWithHeader exercises the new optional catalog-level header.
const catalogManifestWithHeader = `name: acme
description: ACME internal Datum tooling
owner: ACME Platform Team
homepage: https://plugins.acme.example
items:
  - metadata:
      name: deploy
    spec:
      shortDescription: ACME guided deploy workflow
      version: v2.1.0
      platforms:
        - uri: https://plugins.acme.example/deploy_linux_amd64.tar.gz
          sha256: "def"
`

func TestRefreshCatalog_localSourceWithHeader(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "index.yaml")
	if err := os.WriteFile(manifestPath, []byte(catalogManifestWithHeader), 0o644); err != nil {
		t.Fatal(err)
	}

	cat := Catalog{Name: "acme", Source: manifestPath, Type: CatalogTypeCustom}
	idx, err := RefreshCatalog(context.Background(), dir, cat)
	if err != nil {
		t.Fatalf("refresh local catalog: %v", err)
	}
	if idx.Header.Name != "acme" || idx.Header.Owner != "ACME Platform Team" {
		t.Fatalf("header not captured: %+v", idx.Header)
	}
	if FindInIndex(idx, "deploy") == nil {
		t.Fatal("deploy plugin not found in refreshed index")
	}

	// Cache must have been written under indexes/<name>/index.json.
	cachePath, _ := CatalogIndexPath(dir, "acme")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("catalog cache not written: %v", err)
	}
}

func TestRefreshCatalog_rejectsNonHTTPSPluginURI(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "index.yaml")
	bad := `items:
  - metadata:
      name: evil
    spec:
      version: v1.0.0
      platforms:
        - uri: http://insecure.example/evil.tar.gz
          sha256: "x"
`
	if err := os.WriteFile(manifestPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	cat := Catalog{Name: "evil", Source: manifestPath, Type: CatalogTypeCustom}
	if _, err := RefreshCatalog(context.Background(), dir, cat); err == nil {
		t.Fatal("expected refresh to reject non-HTTPS plugin URI")
	}
}

func TestRefreshCatalog_degradesToStaleCache(t *testing.T) {
	dir := t.TempDir()

	// Seed a stale cache for "acme".
	stale := &CachedIndex{Plugins: []Plugin{{Spec: PluginSpec{Version: "v1.0.0"}}}}
	stale.Header.Name = "acme"
	// Set a non-zero RefreshedAt so degraded fallback considers it valid.
	stale.RefreshedAt = time.Now()
	if err := SaveCatalogIndex(dir, "acme", stale); err != nil {
		t.Fatal(err)
	}

	// Source points at a missing local file -> fetch fails, stale cache returned.
	cat := Catalog{Name: "acme", Source: filepath.Join(dir, "does-not-exist", "index.yaml"), Type: CatalogTypeCustom}
	idx, err := RefreshCatalog(context.Background(), dir, cat)
	if err == nil {
		t.Fatal("expected an error alongside the degraded cache")
	}
	if idx == nil || len(idx.Plugins) != 1 {
		t.Fatalf("expected stale cache returned, got %+v", idx)
	}
}
