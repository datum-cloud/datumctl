package plugin

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.datum.net/datumctl/internal/pluginstore"
)

func seedManifest(t *testing.T, pluginsDir string, plugins map[string]*pluginstore.InstalledPlugin) {
	t.Helper()
	if err := pluginstore.Save(pluginsDir, &pluginstore.Manifest{Plugins: plugins}); err != nil {
		t.Fatal(err)
	}
}

func installedEntry(catalog, version string) *pluginstore.InstalledPlugin {
	return &pluginstore.InstalledPlugin{
		Source:      "test",
		Catalog:     catalog,
		Version:     version,
		InstalledAt: time.Now().UTC(),
	}
}

func TestUpgradeCmd_requiresNameOrAll(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"dns": installedEntry("datum", "v1.0.0")})

	_, err := execPluginCmd(t, dir, upgradeCmd(nil))
	if err == nil || !strings.Contains(err.Error(), "--all") {
		t.Fatalf("no args: want error mentioning --all, got %v", err)
	}

	_, err = execPluginCmd(t, dir, upgradeCmd(nil), "--all", "dns")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("--all with name: want combination error, got %v", err)
	}
}

func TestUpgradeCmd_notInstalled(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"dns": installedEntry("datum", "v1.0.0")})

	_, err := execPluginCmd(t, dir, upgradeCmd(nil), "missing")
	if err == nil || !strings.Contains(err.Error(), `plugin "missing" is not installed`) {
		t.Fatalf("want not-installed error, got %v", err)
	}
}

func TestUpgradeCmd_alreadyUpToDate(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"dns": installedEntry("datum", "v1.0.0")})
	// Fresh cache so no network refresh happens; same version as installed.
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, testPlugin("dns", "v1.0.0", "DNS"))

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "dns")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "dns is already up to date (v1.0.0)") {
		t.Fatalf("want up-to-date report, got:\n%s", out)
	}
}

func TestUpgradeAll_noPlugins(t *testing.T) {
	dir := t.TempDir()

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "No managed plugins installed.") {
		t.Fatalf("want empty-manifest message, got:\n%s", out)
	}
}

// TestUpgradeAll_reportsEachAndContinuesOnFailure drives `upgrade --all` over
// three installed plugins: one with a newer version in its catalog, one already
// current, and one that has vanished from the catalog. Every plugin must be
// reported, the failure must not stop the others, and only the upgraded
// plugin's record may change.
func TestUpgradeAll_reportsEachAndContinuesOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test builds a unix binary helper; skip on windows")
	}

	pluginBin := buildHelperBinary(t, `
package main

import (
	"fmt"
	"os"
)

func main() {
	for _, a := range os.Args[1:] {
		if a == "--plugin-manifest" {
			fmt.Println(`+"`"+`{"name":"alpha","version":"v0.2.0","description":"Alpha","api_version":1}`+"`"+`)
			return
		}
	}
}
`)
	binData, err := os.ReadFile(pluginBin)
	if err != nil {
		t.Fatal(err)
	}
	archiveBytes := makeTarGz(t, "alpha", binData)
	url := serveArchive(t, archiveBytes)

	alpha := pluginstore.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "alpha"},
		Spec: pluginstore.PluginSpec{
			ShortDescription: "Alpha",
			Version:          "v0.2.0",
			Platforms: []pluginstore.Platform{
				{URI: url + "/alpha.tar.gz", SHA256: sha256HexOf(archiveBytes)},
			},
		},
	}

	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{
		"alpha": installedEntry("datum", "v0.1.0"),
		"beta":  installedEntry("datum", "v0.1.0"),
		"gamma": installedEntry("datum", "v0.1.0"),
	})
	// "gamma" is deliberately absent from the catalog so its upgrade fails.
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, alpha, testPlugin("beta", "v0.1.0", "Beta"))

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "--all")
	if err == nil {
		t.Fatalf("want an error for the failed plugin, got none; output:\n%s", out)
	}
	if !strings.Contains(err.Error(), "some plugins failed to upgrade") || !strings.Contains(err.Error(), "gamma") {
		t.Fatalf("error should list the failed plugin, got: %v", err)
	}
	if strings.Contains(err.Error(), "alpha") || strings.Contains(err.Error(), "beta") {
		t.Fatalf("error must only list failed plugins, got: %v", err)
	}

	for _, want := range []string{
		"Upgraded alpha from v0.1.0 to v0.2.0",
		"beta is already up to date (v0.1.0)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	manifest, err := pluginstore.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := manifest.Plugins["alpha"]; got.Version != "v0.2.0" || got.Catalog != pluginstore.OfficialCatalogName {
		t.Errorf("alpha record not updated: %+v", got)
	}
	if got := manifest.Plugins["alpha"].Manifest; got == nil || got.Name != "alpha" {
		t.Errorf("alpha manifest not read from new binary: %+v", got)
	}
	if got := manifest.Plugins["beta"]; got.Version != "v0.1.0" {
		t.Errorf("beta record should be unchanged: %+v", got)
	}
	if got := manifest.Plugins["gamma"]; got.Version != "v0.1.0" {
		t.Errorf("gamma record should be unchanged after failure: %+v", got)
	}
}
