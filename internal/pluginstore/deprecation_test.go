package pluginstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const deprecatedCatalogYAML = `apiVersion: datumctl.dev/v1alpha1
kind: PluginList
items:
  - apiVersion: datumctl.dev/v1alpha1
    kind: Plugin
    metadata:
      name: ipam
    spec:
      shortDescription: IPAM
      version: v1.0.0
      deprecated:
        message: "moved"
        replacement:
          catalog: milo-os/cli-plugins
          name: ipam
      platforms: []
`

func TestParseCatalog_deprecatedBlock(t *testing.T) {
	var list PluginList
	if err := yaml.Unmarshal([]byte(deprecatedCatalogYAML), &list); err != nil {
		t.Fatal(err)
	}
	check := func(t *testing.T, items []Plugin) {
		t.Helper()
		if len(items) != 1 {
			t.Fatalf("items = %d, want 1", len(items))
		}
		d := items[0].Spec.Deprecated
		if d == nil || d.Message != "moved" || d.Replacement == nil ||
			d.Replacement.Catalog != "milo-os/cli-plugins" || d.Replacement.Name != "ipam" {
			t.Fatalf("deprecated = %+v", d)
		}
	}
	check(t, list.Items)

	dir := t.TempDir()
	if err := SaveCatalogIndex(dir, "acme", &CachedIndex{Plugins: list.Items}); err != nil {
		t.Fatal(err)
	}
	got, err := LoadCatalogIndex(dir, "acme")
	if err != nil {
		t.Fatal(err)
	}
	check(t, got.Plugins)
}

func TestParseCatalog_noDeprecatedBlock(t *testing.T) {
	var list PluginList
	raw := "items:\n  - metadata:\n      name: ipam\n    spec:\n      version: v1.0.0\n"
	if err := yaml.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatal(err)
	}
	if list.Items[0].Spec.Deprecated != nil {
		t.Fatalf("Deprecated = %+v, want nil", list.Items[0].Spec.Deprecated)
	}
}

func TestPluginReplacement_PluginName(t *testing.T) {
	if got := (&PluginReplacement{}).PluginName("ipam"); got != "ipam" {
		t.Errorf("got %q", got)
	}
	if got := (&PluginReplacement{Name: "ipam2"}).PluginName("ipam"); got != "ipam2" {
		t.Errorf("got %q", got)
	}
}

func TestSameCatalogSource(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"milo-os/cli-plugins", "github.com/milo-os/cli-plugins", true},
		{"milo-os/cli-plugins", "https://raw.githubusercontent.com/milo-os/cli-plugins/main/index.yaml", true},
		{"milo-os/cli-plugins", "https://RAW.GITHUBUSERCONTENT.COM/milo-os/cli-plugins/main/index.yaml", true},
		{"milo-os/cli-plugins", "datum-cloud/datumctl-plugins", false},
		{"http://x", "milo-os/cli-plugins", false},
		{"milo-os/cli-plugins", "http://x", false},
		{"http://x", "http://x", false},
		{"milo-os/cli-plugins", "https://raw.githubusercontent.com/milo-os/cli-plugins/master/index.yaml", true},
		{"milo-os/cli-plugins", "Milo-OS/CLI-Plugins", true},
		{"https://example.com/cat/index.yaml", "https://EXAMPLE.com/cat/index.yaml", true},
		{"milo-os/cli-plugins", "milo-os/other-plugins", false},
	}
	for _, c := range cases {
		if got := SameCatalogSource(c.a, c.b); got != c.want {
			t.Errorf("SameCatalogSource(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestLintCatalog_deprecated(t *testing.T) {
	item := func(d *Deprecation) *PluginList {
		return &PluginList{Items: []Plugin{{
			ObjectMeta: metav1.ObjectMeta{Name: "ipam"},
			Spec: PluginSpec{
				Version: "v1.0.0",
				Platforms: []Platform{{
					URI: "https://example.com/a.tar.gz", SHA256: "abc",
				}},
				Deprecated: d,
			},
		}}}
	}
	problems := strings.Join(LintCatalog(item(&Deprecation{})), "\n")
	for _, want := range []string{"ipam: deprecated has no message", "ipam: deprecated has no replacement.catalog"} {
		if !strings.Contains(problems, want) {
			t.Errorf("missing problem %q in %q", want, problems)
		}
	}
	full := &Deprecation{Message: "moved", Replacement: &PluginReplacement{Catalog: "milo-os/cli-plugins"}}
	if p := LintCatalog(item(full)); len(p) != 0 {
		t.Errorf("full block: problems = %v", p)
	}
	if p := LintCatalog(item(nil)); len(p) != 0 {
		t.Errorf("no block: problems = %v", p)
	}
}

func TestPluginReplacement_PluginName_nil(t *testing.T) {
	var r *PluginReplacement
	if got := r.PluginName("ipam"); got != "ipam" {
		t.Errorf("got %q", got)
	}
}

func TestSameCatalogSource_local(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.yaml"), []byte("items: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !SameCatalogSource(dir, dir+"/index.yaml") {
		t.Error("directory and its index.yaml should match")
	}
	if SameCatalogSource(dir, "milo-os/cli-plugins") {
		t.Error("local path must not match a remote source")
	}
}
