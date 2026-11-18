package plugin

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

// execCmdSplitStreams runs cmd with separate stdout and stderr buffers.
func execCmdSplitStreams(t *testing.T, pluginsDir string, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	t.Setenv("DATUMCTL_PLUGINS_DIR", pluginsDir)
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

// executeListCmd runs the list subcommand with the given plugins.json content
// and returns the captured stdout.
func executeListCmd(t *testing.T, manifest *pluginstore.Manifest) string {
	t.Helper()

	dir := t.TempDir()
	if err := pluginstore.Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest for test: %v", err)
	}

	cmd := Command(nil)                           // factory is not used by listCmd
	cmd.PersistentFlags().Set("plugins-dir", dir) //nolint:errcheck

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute list: %v", err)
	}
	return out.String()
}

// TestListCmd_noPlugins verifies that an empty plugins.json prints the
// "No managed plugins installed" message.
func TestListCmd_noPlugins(t *testing.T) {
	t.Parallel()

	output := executeListCmd(t, &pluginstore.Manifest{})

	if !strings.Contains(output, "No managed plugins installed") {
		t.Errorf("list output %q does not contain 'No managed plugins installed'", output)
	}
}

// TestListCmd_showsDescription verifies that the stored manifest description
// appears in the list output without executing the plugin binary.
func TestListCmd_showsDescription(t *testing.T) {
	t.Parallel()

	manifest := &pluginstore.Manifest{
		Plugins: map[string]*pluginstore.InstalledPlugin{
			"dns": {
				Source:      "github.com/datum-cloud/datumctl-dns",
				Version:     "v1.2.3",
				SHA256:      "abc",
				InstalledAt: time.Now().UTC(),
				Manifest: &pluginstore.PluginManifest{
					Name:        "datumctl-dns",
					Version:     "v1.2.3",
					Description: "Manage DNS zones on Datum Cloud",
					APIVersion:  plugindispatch.PluginAPIVersion,
				},
			},
		},
	}

	output := executeListCmd(t, manifest)

	if !strings.Contains(output, "Manage DNS zones on Datum Cloud") {
		t.Errorf("list output %q does not contain description", output)
	}
	if !strings.Contains(output, "v1.2.3") {
		t.Errorf("list output %q does not contain version", output)
	}
	if !strings.Contains(output, "dns") {
		t.Errorf("list output %q does not contain plugin name", output)
	}
}

// TestListCmd_showsAPIVersionMismatch verifies that when the stored api_version
// does not match the host API version, "!" appears in the output.
func TestListCmd_showsAPIVersionMismatch(t *testing.T) {
	t.Parallel()

	mismatchedAPIVersion := plugindispatch.PluginAPIVersion + 999

	manifest := &pluginstore.Manifest{
		Plugins: map[string]*pluginstore.InstalledPlugin{
			"dns": {
				Source:      "github.com/datum-cloud/datumctl-dns",
				Version:     "v0.1.0",
				InstalledAt: time.Now().UTC(),
				Manifest: &pluginstore.PluginManifest{
					Name:       "datumctl-dns",
					APIVersion: mismatchedAPIVersion,
				},
			},
		},
	}

	output := executeListCmd(t, manifest)

	if !strings.Contains(output, "!") {
		t.Errorf("list output %q does not contain '!' for API version mismatch", output)
	}
}

func listEntryWithManifest(catalog, version string) *pluginstore.InstalledPlugin {
	e := installedEntry(catalog, version)
	e.Manifest = &pluginstore.PluginManifest{
		Name:        "datumctl-ipam",
		Version:     version,
		Description: "IPAM",
		APIVersion:  plugindispatch.PluginAPIVersion,
	}
	return e
}

func TestListCmd_deprecatedStatusAndNotice(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"ipam": listEntryWithManifest("datum", "v0.1.0")})
	// A newer version is available too; deprecated must win over update.
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, deprecatedPlugin("ipam", "v0.2.0"))

	out, stderr, err := execCmdSplitStreams(t, dir, listCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, stderr)
	}
	var row string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "ipam") {
			row = l
		}
	}
	if !strings.HasSuffix(strings.TrimSpace(row), "deprecated") {
		t.Fatalf("want STATUS deprecated in row %q\n%s", row, out)
	}
	if !strings.Contains(stderr, "datumctl plugin install milo-os/ipam") {
		t.Fatalf("want notice with replacement command on stderr:\n%s", stderr)
	}
	if strings.Contains(out, "warning:") || strings.Contains(out, "datumctl plugin install") {
		t.Fatalf("notice must not be on stdout:\n%s", out)
	}
}

func TestListCmd_deprecatedWithAPIMismatchKeepsBangAndNotice(t *testing.T) {
	dir := t.TempDir()
	e := listEntryWithManifest("datum", "v0.1.0")
	e.Manifest.APIVersion = plugindispatch.PluginAPIVersion + 999
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"ipam": e})
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, deprecatedPlugin("ipam", "v0.1.0"))

	out, stderr, err := execCmdSplitStreams(t, dir, listCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, stderr)
	}
	var row string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "ipam") {
			row = l
		}
	}
	if !strings.HasSuffix(strings.TrimSpace(row), "!") {
		t.Fatalf("want STATUS ! in row %q\n%s", row, out)
	}
	if !strings.Contains(stderr, "warning: ipam is deprecated: use the new one") {
		t.Fatalf("want notice on stderr:\n%s", stderr)
	}
}

func TestListCmd_notDeprecatedNoNotice(t *testing.T) {
	dir := t.TempDir()
	seedManifest(t, dir, map[string]*pluginstore.InstalledPlugin{"ipam": listEntryWithManifest("datum", "v0.1.0")})
	seedFreshCache(t, dir, pluginstore.OfficialCatalogName, testPlugin("ipam", "v0.1.0", "IPAM"))

	out, err := execPluginCmd(t, dir, listCmd())
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(out, "deprecated") || strings.Contains(out, "is deprecated") {
		t.Fatalf("want no deprecation output:\n%s", out)
	}
}

// TestListStatus_deprecatedWithUpdate verifies a deprecated plugin with a newer
// catalog version shows "deprecated" yet still reports an available update, so
// the upgrade hint prints.
func TestListStatus_deprecatedWithUpdate(t *testing.T) {
	t.Parallel()

	newer := &pluginstore.Plugin{Spec: pluginstore.PluginSpec{
		Version:    "v1.2.0",
		Deprecated: &pluginstore.Deprecation{},
	}}
	if st, up := listStatus("v1.0.0", newer); st != "deprecated" || !up {
		t.Errorf("listStatus = (%q, %v), want (deprecated, true)", st, up)
	}

	same := &pluginstore.Plugin{Spec: pluginstore.PluginSpec{
		Version:    "v1.0.0",
		Deprecated: &pluginstore.Deprecation{},
	}}
	if st, up := listStatus("v1.0.0", same); st != "deprecated" || up {
		t.Errorf("listStatus = (%q, %v), want (deprecated, false)", st, up)
	}

	plain := &pluginstore.Plugin{Spec: pluginstore.PluginSpec{Version: "v1.2.0"}}
	if st, up := listStatus("v1.0.0", plain); st != "update" || !up {
		t.Errorf("listStatus = (%q, %v), want (update, true)", st, up)
	}
}
