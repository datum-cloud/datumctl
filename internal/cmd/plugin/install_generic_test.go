package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

// makeTarGzMulti builds a .tar.gz containing several named files.
func makeTarGzMulti(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatalf("tar write: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	return buf.Bytes()
}

// TestInstallPlugin_prefersMiloPrefixOverServiceBinary is the milo- collision
// fix: an archive that bundles BOTH the bare service binary "ipam" and the
// plugin "milo-ipam" must install milo-ipam, never the service binary.
func TestInstallPlugin_prefersMiloPrefixOverServiceBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test builds unix binary helpers; skip on windows")
	}

	// The plugin responds to --plugin-manifest with name "milo-ipam".
	pluginBin := buildHelperBinary(t, `
package main

import (
	"fmt"
	"os"
)

func main() {
	for _, a := range os.Args[1:] {
		if a == "--plugin-manifest" {
			fmt.Println(`+"`"+`{"name":"milo-ipam","version":"v0.1.0","description":"IPAM plugin","api_version":1}`+"`"+`)
			return
		}
	}
}
`)
	pluginBytes, err := os.ReadFile(pluginBin)
	if err != nil {
		t.Fatal(err)
	}
	// The service binary shares the bare name "ipam" and emits no manifest.
	serviceBytes := []byte("#!/bin/sh\necho SERVICE\n")

	// Auto-detect (no files directive) over an archive containing BOTH.
	archiveBytes := makeTarGzMulti(t, map[string][]byte{
		"ipam":      serviceBytes,
		"milo-ipam": pluginBytes,
	})
	url := serveArchive(t, archiveBytes)

	idx := &pluginstore.CachedIndex{
		RefreshedAt: time.Now(),
		Plugins: []pluginstore.Plugin{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "ipam"},
				Spec: pluginstore.PluginSpec{
					ShortDescription: "IPAM plugin",
					Version:          "v0.1.0",
					Platforms: []pluginstore.Platform{
						{URI: url + "/ipam.tar.gz", SHA256: sha256HexOf(archiveBytes)},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	entry, _, _, err := installPlugin(context.Background(), dir, "ipam", "", "v9.0.0", idx)
	if err != nil {
		t.Fatalf("installPlugin: %v", err)
	}

	// Decisive: the plugin (milo-ipam) was extracted, not the service binary.
	if entry.Manifest == nil || entry.Manifest.Name != "milo-ipam" {
		t.Fatalf("expected the milo-ipam plugin to be installed, got manifest %+v", entry.Manifest)
	}
	onDisk, err := os.ReadFile(filepath.Join(dir, "ipam"))
	if err != nil {
		t.Fatal(err)
	}
	if sha256HexOf(onDisk) != sha256HexOf(pluginBytes) {
		t.Fatal("on-disk binary is not the milo-ipam plugin; the service binary was wrongly installed")
	}
}
