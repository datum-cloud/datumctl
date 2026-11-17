package plugin

import (
	"bytes"
	"os"
	"runtime"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.datum.net/datumctl/internal/pluginstore"
)

func deprecatedPlugin(name, version string) pluginstore.Plugin {
	p := testPlugin(name, version, name)
	p.Spec.Deprecated = &pluginstore.Deprecation{
		Message: "use the new one",
		Replacement: &pluginstore.PluginReplacement{
			Catalog: "milo-os/cli-plugins",
			Name:    name,
		},
	}
	return p
}

func TestUpgrade_deprecatedUpToDatePrintsNotice(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"ipam": installedEntry("datum", "v0.1.0")})
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, deprecatedPlugin("ipam", "v0.1.0"))

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "ipam")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	for _, want := range []string{"ipam is already up to date", "warning: ipam is deprecated: use the new one"} {
		if !strings.Contains(out, want) {
			t.Fatalf("want %q in output:\n%s", want, out)
		}
	}
}

func TestUpgrade_movedPluginNoNotice(t *testing.T) {
	dir := t.TempDir()
	if err := pluginstore.SaveRegistry(dir, &pluginstore.Registry{Catalogs: []pluginstore.Catalog{
		{Name: "milo-os", Source: "milo-os/cli-plugins", Type: pluginstore.CatalogTypeCustom},
	}}); err != nil {
		t.Fatal(err)
	}
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"ipam": installedEntry("milo-os", "v0.1.0")})
	seedFreshCache(t, dir, "milo-os", testPlugin("ipam", "v0.1.0", "IPAM"))
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, deprecatedPlugin("ipam", "v0.1.0"))

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "ipam")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if !strings.Contains(out, "ipam is already up to date") || strings.Contains(out, "deprecated") {
		t.Fatalf("want up-to-date report and no notice, got:\n%s", out)
	}
}

func TestUpgrade_nonDeprecatedNoNotice(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"dns": installedEntry("datum", "v1.0.0")})
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, testPlugin("dns", "v1.0.0", "DNS"))

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "dns")
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(out, "deprecated") {
		t.Fatalf("want no notice, got:\n%s", out)
	}
}

func TestUpgradeAll_failedUpgradeNoNotice(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{
		"ipam": installedEntry("datum", "v0.1.0"),
		"dns":  installedEntry("datum", "v1.0.0"),
	})
	// ipam is deprecated and has a newer version with no platforms, so its
	// upgrade fails.
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName,
		deprecatedPlugin("ipam", "v0.2.0"), deprecatedPlugin("dns", "v1.0.0"))

	out, err := execPluginCmd(t, dir, upgradeCmd(nil), "--all")
	if err == nil {
		t.Fatalf("want error for failed ipam upgrade, got none:\n%s", out)
	}
	if strings.Contains(out, "warning: ipam is deprecated") {
		t.Fatalf("failed upgrade must not print a notice:\n%s", out)
	}
	if !strings.Contains(out, "warning: dns is deprecated") {
		t.Fatalf("successful plugin should print its notice:\n%s", out)
	}
}

func TestInstallFromCatalog_deprecatedPrintsNotice(t *testing.T) {
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
			Platforms:        []pluginstore.Platform{{URI: url + "/alpha.tar.gz", SHA256: sha256HexOf(archiveBytes)}},
			Deprecated:       &pluginstore.Deprecation{Message: "use beta"},
		},
	}
	dir := t.TempDir()
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, alpha)

	t.Setenv("DATUMCTL_PLUGINS_DIR", dir)
	var stdout, stderr bytes.Buffer
	cmd := installCmd(nil)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"datum/alpha"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v\n%s%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Installed alpha") {
		t.Fatalf("want install report on stdout:\n%s", stdout.String())
	}
	const notice = "warning: alpha is deprecated: use beta"
	if !strings.Contains(stderr.String(), notice) || strings.Contains(stdout.String(), "deprecated") {
		t.Fatalf("notice must be on stderr only; stdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	m, err := pluginstore.Load(dir)
	if err != nil || m.Plugins["alpha"] == nil {
		t.Fatalf("record not saved: %v", err)
	}
}

func TestInstallRestore_deprecatedPrintsNotice(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test builds a unix binary helper; skip on windows")
	}
	build := func(name string) pluginstore.Plugin {
		bin := buildHelperBinary(t, `
package main

import (
	"fmt"
	"os"
)

func main() {
	for _, a := range os.Args[1:] {
		if a == "--plugin-manifest" {
			fmt.Println(`+"`"+`{"name":"`+name+`","version":"v0.2.0","description":"x","api_version":1}`+"`"+`)
			return
		}
	}
}
`)
		data, err := os.ReadFile(bin)
		if err != nil {
			t.Fatal(err)
		}
		archive := makeTarGz(t, name, data)
		url := serveArchive(t, archive)
		return pluginstore.Plugin{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: pluginstore.PluginSpec{
				ShortDescription: name,
				Version:          "v0.2.0",
				Platforms:        []pluginstore.Platform{{URI: url + "/" + name + ".tar.gz", SHA256: sha256HexOf(archive)}},
			},
		}
	}
	alpha := build("alpha")
	alpha.Spec.Deprecated = &pluginstore.Deprecation{Message: "use gamma"}
	beta := build("beta")

	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{
		"alpha": installedEntry("datum", "v0.2.0"),
		"beta":  installedEntry("datum", "v0.2.0"),
	})
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, alpha, beta)

	out, err := execPluginCmd(t, dir, installCmd(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	for _, want := range []string{"Installed alpha", "Installed beta", "warning: alpha is deprecated: use gamma"} {
		if !strings.Contains(out, want) {
			t.Fatalf("want %q in output:\n%s", want, out)
		}
	}
	if strings.Contains(out, "beta is deprecated") {
		t.Fatalf("beta must get no notice:\n%s", out)
	}
}
