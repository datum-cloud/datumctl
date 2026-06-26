package plugin

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.datum.net/datumctl/internal/pluginstore"
)

// serveArchive starts an HTTPS test server returning archiveBytes and patches
// the default transport to trust it. Returns the server URL.
func serveArchive(t *testing.T, archiveBytes []byte) string {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveBytes) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)

	origTransport := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
	}
	t.Cleanup(func() { http.DefaultTransport = origTransport })
	return srv.URL
}

// TestInstallPlugin_genericArchiveBinary verifies a milo-os style plugin whose
// archive contains a GENERIC binary name (e.g. "ipam", no datumctl- prefix) is
// installed under that generic name in the managed dir.
func TestInstallPlugin_genericArchiveBinary(t *testing.T) {
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
			fmt.Println(`+"`"+`{"name":"ipam","version":"v0.1.0","description":"IPAM service","api_version":1}`+"`"+`)
			return
		}
	}
}
`)
	binData, err := os.ReadFile(pluginBin)
	if err != nil {
		t.Fatal(err)
	}

	// Archive contains a generically named binary "ipam".
	archiveBytes := makeTarGz(t, "ipam", binData)
	url := serveArchive(t, archiveBytes)

	idx := &pluginstore.CachedIndex{
		RefreshedAt: time.Now(),
		Plugins: []pluginstore.Plugin{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "ipam"},
				Spec: pluginstore.PluginSpec{
					ShortDescription: "IPAM service",
					Version:          "v0.1.0",
					Platforms: []pluginstore.Platform{
						{
							URI:    url + "/ipam.tar.gz",
							SHA256: sha256HexOf(archiveBytes),
							// Explicit binary directive, as a milo-os catalog would write.
							Files: []pluginstore.FileOperation{{From: "ipam"}},
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	entry, name, _, err := installPlugin(context.Background(), dir, "ipam", "", "v9.0.0", idx)
	if err != nil {
		t.Fatalf("installPlugin: %v", err)
	}
	if name != "ipam" {
		t.Errorf("name = %q, want ipam", name)
	}
	if entry.Manifest == nil || entry.Manifest.Name != "ipam" {
		t.Errorf("manifest not read from generic binary: %+v", entry.Manifest)
	}

	// Stored under the generic name; NOT datumctl-prefixed.
	if _, err := os.Stat(filepath.Join(dir, "ipam")); err != nil {
		t.Errorf("expected generic binary at <dir>/ipam: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "datumctl-ipam")); err == nil {
		t.Error("binary must not be written under the datumctl- prefix")
	}
}
