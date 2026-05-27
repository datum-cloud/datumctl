package plugin

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

// executeListCmd runs the list subcommand with the given plugins.json content
// and returns the captured stdout.
func executeListCmd(t *testing.T, manifest *pluginstore.Manifest) string {
	t.Helper()

	dir := t.TempDir()
	if err := pluginstore.Save(dir, manifest); err != nil {
		t.Fatalf("Save manifest for test: %v", err)
	}

	cmd := Command(nil) // factory is not used by listCmd
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
