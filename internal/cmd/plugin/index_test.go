package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

const testCatalogManifest = `name: acme
description: ACME internal Datum tooling
owner: ACME Platform Team
items:
  - metadata:
      name: deploy
    spec:
      shortDescription: ACME guided deploy
      version: v2.1.0
      platforms:
        - uri: https://plugins.acme.example/deploy.tar.gz
          sha256: "abc"
`

// writeLocalCatalog creates a directory with an index.yaml and returns the dir.
func writeLocalCatalog(t *testing.T, manifest string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runIndex executes `index <args...>` against a fresh command tree, with stdin
// wired to the given input. It returns stdout+stderr combined and the error.
func runIndex(t *testing.T, pluginsDir, stdin string, args ...string) (string, error) {
	t.Helper()
	t.Setenv("DATUMCTL_PLUGINS_DIR", pluginsDir)
	cmd := indexCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestIndexAdd_withYesRegistersCatalog(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)

	out, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes")
	if err != nil {
		t.Fatalf("add failed: %v (out=%s)", err, out)
	}
	if !strings.Contains(out, "Added catalog acme") || !strings.Contains(out, pluginstore.TrustThirdParty) {
		t.Fatalf("unexpected output: %s", out)
	}

	reg, err := pluginstore.LoadRegistry(pluginsDir)
	if err != nil {
		t.Fatal(err)
	}
	cat := reg.Find("acme")
	if cat == nil {
		t.Fatal("acme not registered")
	}
	if cat.TrustedAt.IsZero() {
		t.Fatal("trustedAt not recorded")
	}
	// Initial refresh should have captured the header.
	if cat.Description != "ACME internal Datum tooling" {
		t.Fatalf("catalog header not captured: %+v", cat)
	}
}

func TestIndexAdd_trustPromptAccept(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)

	out, err := runIndex(t, pluginsDir, "y\n", "add", "acme", src)
	if err != nil {
		t.Fatalf("add failed: %v (out=%s)", err, out)
	}
	if !strings.Contains(out, "third-party plugin catalog") {
		t.Fatalf("trust prompt not shown: %s", out)
	}
	if reg, _ := pluginstore.LoadRegistry(pluginsDir); reg.Find("acme") == nil {
		t.Fatal("acme should be registered after accepting")
	}
}

func TestIndexAdd_trustPromptDecline(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)

	out, err := runIndex(t, pluginsDir, "n\n", "add", "acme", src)
	if err != nil {
		t.Fatalf("decline should not error: %v", err)
	}
	if !strings.Contains(out, "Aborted") {
		t.Fatalf("expected abort message: %s", out)
	}
	if reg, _ := pluginstore.LoadRegistry(pluginsDir); reg.Find("acme") != nil {
		t.Fatal("acme must NOT be registered after declining")
	}
}

func TestIndexAdd_emptyStdinDeclines(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)

	// No stdin -> EOF -> treated as "no".
	_, err := runIndex(t, pluginsDir, "", "add", "acme", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reg, _ := pluginstore.LoadRegistry(pluginsDir); reg.Find("acme") != nil {
		t.Fatal("acme must NOT be registered when prompt gets no confirmation")
	}
}

func TestIndexAdd_reservedNameRejected(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)
	if _, err := runIndex(t, pluginsDir, "", "add", "default", src, "--yes"); err == nil {
		t.Fatal("expected reserved-name error")
	}
	if _, err := runIndex(t, pluginsDir, "", "add", "official", src, "--yes"); err == nil {
		t.Fatal("expected reserved-name error for 'official'")
	}
}

func TestIndexAdd_duplicateRejected(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)
	if _, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes"); err != nil {
		t.Fatal(err)
	}
	if _, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes"); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestIndexAdd_allowListDenies(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)
	t.Setenv("DATUMCTL_PLUGIN_ALLOWED_INDEXES", "approved-only")
	_, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes")
	if err == nil {
		t.Fatal("expected allow-list denial")
	}
	if !strings.Contains(err.Error(), "allow-list") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIndexAdd_allowListPermitsByName(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)
	t.Setenv("DATUMCTL_PLUGIN_ALLOWED_INDEXES", "acme")
	if _, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes"); err != nil {
		t.Fatalf("acme should be permitted by name: %v", err)
	}
}

func TestIndexList_showsDefaultAndAdded(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)
	if _, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes"); err != nil {
		t.Fatal(err)
	}
	out, err := runIndex(t, pluginsDir, "", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "default") || !strings.Contains(out, "official") {
		t.Fatalf("default/official badge missing: %s", out)
	}
	if !strings.Contains(out, "acme") || !strings.Contains(out, "third-party") {
		t.Fatalf("acme/third-party badge missing: %s", out)
	}
	// default must be listed first.
	if i, j := strings.Index(out, "default"), strings.Index(out, "acme"); i < 0 || j < 0 || i > j {
		t.Fatalf("default should be first: %s", out)
	}
}

func TestIndexRemove_defaultRejected(t *testing.T) {
	pluginsDir := t.TempDir()
	if _, err := runIndex(t, pluginsDir, "", "remove", "default"); err == nil {
		t.Fatal("removing default must be rejected")
	}
}

func TestIndexRemove_removesAndCleansCache(t *testing.T) {
	pluginsDir := t.TempDir()
	src := writeLocalCatalog(t, testCatalogManifest)
	if _, err := runIndex(t, pluginsDir, "", "add", "acme", src, "--yes"); err != nil {
		t.Fatal(err)
	}
	cacheDir, _ := pluginstore.CatalogCacheDir(pluginsDir, "acme")
	if _, err := os.Stat(cacheDir); err != nil {
		t.Fatalf("expected cache dir to exist after add: %v", err)
	}

	if _, err := runIndex(t, pluginsDir, "", "remove", "acme"); err != nil {
		t.Fatal(err)
	}
	if reg, _ := pluginstore.LoadRegistry(pluginsDir); reg.Find("acme") != nil {
		t.Fatal("acme should be gone")
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("cache dir should be removed, got err=%v", err)
	}
}

func TestIndexRemove_managedRejected(t *testing.T) {
	pluginsDir := t.TempDir()
	mc := filepath.Join(t.TempDir(), "managed.yaml")
	if err := os.WriteFile(mc, []byte("indexes:\n  - name: corp\n    source: https://corp.example/index.yaml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DATUMCTL_PLUGIN_MANAGED_CONFIG", mc)
	if _, err := runIndex(t, pluginsDir, "", "remove", "corp"); err == nil {
		t.Fatal("removing a managed catalog must be rejected")
	}
}

func TestIndexValidate_goodAndBad(t *testing.T) {
	pluginsDir := t.TempDir()

	goodDir := writeLocalCatalog(t, testCatalogManifest)
	out, err := runIndex(t, pluginsDir, "", "validate", filepath.Join(goodDir, "index.yaml"))
	if err != nil {
		t.Fatalf("valid manifest should pass: %v (out=%s)", err, out)
	}
	if !strings.Contains(out, "OK") {
		t.Fatalf("expected OK: %s", out)
	}

	badDir := writeLocalCatalog(t, `items:
  - metadata:
      name: broken
    spec:
      platforms:
        - uri: http://insecure.example/x.tar.gz
`)
	_, err = runIndex(t, pluginsDir, "", "validate", filepath.Join(badDir, "index.yaml"))
	if err == nil {
		t.Fatal("invalid manifest should fail validation")
	}
	msg := err.Error()
	if !strings.Contains(msg, "HTTPS") || !strings.Contains(msg, "sha256") || !strings.Contains(msg, "version") {
		t.Fatalf("expected problems for https/sha256/version, got: %s", msg)
	}
}

func TestIndexUpdate_missingCatalogErrors(t *testing.T) {
	pluginsDir := t.TempDir()
	if _, err := runIndex(t, pluginsDir, "", "update", "nope"); err == nil {
		t.Fatal("updating an unregistered catalog must error")
	}
}
