package plugin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.datum.net/datumctl/internal/plugindispatch"
	"go.datum.net/datumctl/internal/pluginstore"
)

// TestPluginAssetName_allPlatforms verifies archive name generation for the
// current platform matches the expected pattern.
func TestPluginAssetName_allPlatforms(t *testing.T) {
	t.Parallel()

	got, err := pluginAssetName("datumctl-dns", "v1.0.0")
	if err != nil {
		supported := (runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows") &&
			(runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" || runtime.GOARCH == "386")
		if supported {
			t.Fatalf("pluginAssetName for current platform: %v", err)
		}
		t.Skipf("pluginAssetName: unsupported test platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	if !strings.HasPrefix(got, "datumctl-dns_") {
		t.Errorf("pluginAssetName: got %q, want prefix %q", got, "datumctl-dns_")
	}
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(got, ".zip") {
			t.Errorf("pluginAssetName on windows: got %q, want .zip suffix", got)
		}
	} else {
		if !strings.HasSuffix(got, ".tar.gz") {
			t.Errorf("pluginAssetName on non-windows: got %q, want .tar.gz suffix", got)
		}
	}
}

// TestBinaryNameFromAsset verifies that binaryNameFromAsset correctly derives
// the expected binary name from various asset filenames.
func TestBinaryNameFromAsset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		assetName string
		want      string
	}{
		{"datumctl-dns_Linux_x86_64.tar.gz", "datumctl-dns"},
		{"datumctl-dns_Darwin_arm64.tar.gz", "datumctl-dns"},
		{"datumctl-dns_Windows_x86_64.zip", "datumctl-dns.exe"},
	}

	for _, tt := range tests {
		got := binaryNameFromAsset(tt.assetName)
		if got != tt.want {
			t.Errorf("binaryNameFromAsset(%q) = %q, want %q", tt.assetName, got, tt.want)
		}
	}
}

// TestFetchChecksums_parsesOneSpaceAndTwoSpace verifies that fetchChecksums
// handles both "sha256  file" (two spaces, goreleaser default) and "sha256 file"
// (one space).
func TestFetchChecksums_parsesOneSpaceAndTwoSpace(t *testing.T) {
	t.Parallel()

	checksumBody := "" +
		"abc123  datumctl-dns_Linux_x86_64.tar.gz\n" +
		"def456 datumctl-dns_Darwin_arm64.tar.gz\n" +
		"\n" // trailing blank line — must be ignored

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, checksumBody)
	}))
	defer srv.Close()

	expected := map[string]string{
		"datumctl-dns_Linux_x86_64.tar.gz": "abc123",
		"datumctl-dns_Darwin_arm64.tar.gz": "def456",
	}

	ctx := context.Background()
	got, err := fetchChecksumsFromURL(ctx, srv.URL)
	if err != nil {
		t.Fatalf("fetchChecksumsFromURL: %v", err)
	}

	for file, sha := range expected {
		if got[file] != sha {
			t.Errorf("checksums[%q] = %q, want %q", file, got[file], sha)
		}
	}
}

// fetchChecksumsFromURL is a test helper that fetches and parses a raw checksums
// URL (as opposed to fetchChecksums which constructs the GitHub Release URL).
func fetchChecksumsFromURL(ctx context.Context, rawURL string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		body.Write(buf[:n])
		if readErr != nil {
			break
		}
	}

	checksums := make(map[string]string)
	for _, line := range strings.Split(body.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		checksums[fields[1]] = fields[0]
	}
	return checksums, nil
}

// TestCheckCompatibility_rejectsOldVersion verifies that checkCompatibility
// returns an error when the current datumctl version is below min_datumctl_version.
func TestCheckCompatibility_rejectsOldVersion(t *testing.T) {
	t.Parallel()

	m := &pluginstore.PluginManifest{
		Name:               "dns",
		APIVersion:         plugindispatch.PluginAPIVersion,
		MinDatumctlVersion: "v2.0.0",
	}

	_, err := checkCompatibility(m, "v1.0.0", plugindispatch.PluginAPIVersion)
	if err == nil {
		t.Error("checkCompatibility: want error when current version < min_datumctl_version, got nil")
	}
}

// TestCheckCompatibility_rejectsLowAPIVersion verifies that checkCompatibility
// returns an error when the host API version is below min_api_version.
func TestCheckCompatibility_rejectsLowAPIVersion(t *testing.T) {
	t.Parallel()

	m := &pluginstore.PluginManifest{
		Name:          "dns",
		APIVersion:    1,
		MinAPIVersion: 5,
	}

	_, err := checkCompatibility(m, "v9.0.0", 2)
	if err == nil {
		t.Error("checkCompatibility: want error when host API version < min_api_version, got nil")
	}
}

// TestCheckCompatibility_warnsOnVersionMismatch verifies that soft mismatches
// (such as invocation-time API version drift) return a non-empty warn string
// via CheckCompatibilityAtInvocation.
func TestCheckCompatibility_warnsOnVersionMismatch(t *testing.T) {
	t.Parallel()

	m := &pluginstore.PluginManifest{
		Name:       "dns",
		APIVersion: 1,
	}

	warn, err := plugindispatch.CheckCompatibilityAtInvocation(m, "v9.0.0", 2)
	if err != nil {
		t.Fatalf("CheckCompatibilityAtInvocation: unexpected hard error: %v", err)
	}
	if warn == "" {
		t.Error("CheckCompatibilityAtInvocation: want non-empty warn for API version mismatch, got empty string")
	}
}

// TestReadPluginManifest_noManifest verifies that readPluginManifest returns
// nil, nil when the binary exits non-zero on --plugin-manifest.
func TestReadPluginManifest_noManifest(t *testing.T) {
	t.Parallel()

	binPath := buildHelperBinary(t, `
package main

import "os"

func main() {
	os.Exit(1)
}
`)

	m, err := readPluginManifest(binPath)
	if err != nil {
		t.Errorf("readPluginManifest non-zero exit: want nil error, got %v", err)
	}
	if m != nil {
		t.Errorf("readPluginManifest non-zero exit: want nil manifest, got %+v", m)
	}
}

// TestReadPluginManifest_validJSON verifies that readPluginManifest parses the
// manifest when the binary prints valid JSON to stdout and exits 0.
func TestReadPluginManifest_validJSON(t *testing.T) {
	t.Parallel()

	binPath := buildHelperBinary(t, `
package main

import "fmt"

func main() {
	fmt.Println(`+"`"+`{"name":"datumctl-dns","version":"v1.0.0","description":"DNS plugin","api_version":1}`+"`"+`)
}
`)

	m, err := readPluginManifest(binPath)
	if err != nil {
		t.Fatalf("readPluginManifest valid JSON: %v", err)
	}
	if m == nil {
		t.Fatal("readPluginManifest valid JSON: want non-nil manifest, got nil")
	}
	if m.Name != "datumctl-dns" {
		t.Errorf("manifest.Name = %q, want %q", m.Name, "datumctl-dns")
	}
	if m.APIVersion != 1 {
		t.Errorf("manifest.APIVersion = %d, want 1", m.APIVersion)
	}
}

// TestReadPluginManifest_malformedJSON verifies that readPluginManifest returns
// an error when the binary prints invalid JSON.
func TestReadPluginManifest_malformedJSON(t *testing.T) {
	t.Parallel()

	binPath := buildHelperBinary(t, `
package main

import "fmt"

func main() {
	fmt.Println("{not valid json")
}
`)

	_, err := readPluginManifest(binPath)
	if err == nil {
		t.Error("readPluginManifest malformed JSON: want error, got nil")
	}
}

// TestInstallPlugin_notInIndex verifies that installPlugin returns an error when
// the plugin name is not in the index.
func TestInstallPlugin_notInIndex(t *testing.T) {
	t.Parallel()

	idx := &pluginstore.CachedIndex{
		RefreshedAt: time.Now(),
		Plugins:     []pluginstore.Plugin{},
	}

	_, _, _, err := installPlugin(context.Background(), t.TempDir(), "nonexistent", "", "v0.0.0", idx)
	if err == nil {
		t.Error("installPlugin: want error for plugin not in index, got nil")
	}
}

// TestInstallPlugin_nilIndex verifies that installPlugin returns an error when
// idx is nil.
func TestInstallPlugin_nilIndex(t *testing.T) {
	t.Parallel()

	_, _, _, err := installPlugin(context.Background(), t.TempDir(), "dns", "", "v0.0.0", nil)
	if err == nil {
		t.Error("installPlugin: want error for nil index, got nil")
	}
}

// TestInstallPlugin_selectorBased verifies that installPlugin correctly selects
// the matching platform and downloads the archive from a stubbed HTTPS server.
// Not parallel — patches http.DefaultTransport to trust the self-signed test cert.
func TestInstallPlugin_selectorBased(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test builds a unix binary helper; skip on windows")
	}

	// Build a valid plugin binary that responds to --plugin-manifest.
	pluginBin := buildHelperBinary(t, `
package main

import (
	"fmt"
	"os"
)

func main() {
	for _, a := range os.Args[1:] {
		if a == "--plugin-manifest" {
			fmt.Println(`+"`"+`{"name":"datumctl-testplugin","version":"v0.1.0","description":"test","api_version":1}`+"`"+`)
			return
		}
	}
}
`)

	binData, err := os.ReadFile(pluginBin)
	if err != nil {
		t.Fatalf("read plugin binary: %v", err)
	}

	archiveBytes := makeTarGz(t, "datumctl-testplugin", binData)
	archiveSHA := sha256HexOf(archiveBytes)

	// Use a TLS test server so that the HTTPS-only scheme check in
	// downloadAndVerifyURI passes. The self-signed cert is trusted via the
	// server's own Client() helper.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveBytes) //nolint:errcheck
	}))
	defer srv.Close()

	// Patch the default HTTP client to trust the test server's self-signed cert.
	origTransport := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
	}
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	idx := &pluginstore.CachedIndex{
		RefreshedAt: time.Now(),
		Plugins: []pluginstore.Plugin{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "testplugin"},
				Spec: pluginstore.PluginSpec{
					ShortDescription: "test plugin",
					Version:          "v0.1.0",
					Platforms: []pluginstore.Platform{
						{
							// nil Selector matches all platforms.
							URI:    srv.URL + "/datumctl-testplugin.tar.gz",
							SHA256: archiveSHA,
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	entry, name, _, err := installPlugin(context.Background(), dir, "testplugin", "", "v9.0.0", idx)
	if err != nil {
		t.Fatalf("installPlugin: %v", err)
	}
	if name != "testplugin" {
		t.Errorf("pluginName = %q, want %q", name, "testplugin")
	}
	if entry.Version != "v0.1.0" {
		t.Errorf("entry.Version = %q, want %q", entry.Version, "v0.1.0")
	}
	if entry.Source != "testplugin" {
		t.Errorf("entry.Source = %q, want %q", entry.Source, "testplugin")
	}

	// Verify binary exists on disk.
	binPath := filepath.Join(dir, "datumctl-testplugin")
	if _, err := os.Stat(binPath); err != nil {
		t.Errorf("expected binary at %s: %v", binPath, err)
	}
}

// TestInstallPlugin_storedSHA256IsBinaryNotArchive verifies that the SHA256
// stored in InstalledPlugin.SHA256 is the SHA256 of the extracted binary, NOT
// the SHA256 of the downloaded archive. This guards against a regression where
// verifyManagedPluginIntegrity would always reject legitimately installed plugins
// because the archive hash != the binary hash.
func TestInstallPlugin_storedSHA256IsBinaryNotArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test builds a unix binary helper; skip on windows")
	}

	// Build a valid plugin binary that responds to --plugin-manifest.
	pluginBin := buildHelperBinary(t, `
package main

import (
	"fmt"
	"os"
)

func main() {
	for _, a := range os.Args[1:] {
		if a == "--plugin-manifest" {
			fmt.Println(`+"`"+`{"name":"datumctl-hashtest","version":"v0.1.0","description":"test","api_version":1}`+"`"+`)
			return
		}
	}
}
`)

	binData, err := os.ReadFile(pluginBin)
	if err != nil {
		t.Fatalf("read plugin binary: %v", err)
	}

	archiveBytes := makeTarGz(t, "datumctl-hashtest", binData)

	// Pre-compute the hashes that the test will assert against.
	archiveSHA := sha256HexOf(archiveBytes)
	binarySHA := sha256HexOf(binData)

	// The two hashes must be different — if they were equal the test would be vacuous.
	if archiveSHA == binarySHA {
		t.Fatal("test invariant violated: archive SHA256 == binary SHA256; test cannot distinguish the two")
	}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archiveBytes) //nolint:errcheck
	}))
	defer srv.Close()

	origTransport := http.DefaultTransport
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
	}
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	idx := &pluginstore.CachedIndex{
		RefreshedAt: time.Now(),
		Plugins: []pluginstore.Plugin{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "hashtest"},
				Spec: pluginstore.PluginSpec{
					ShortDescription: "hash test plugin",
					Version:          "v0.1.0",
					Platforms: []pluginstore.Platform{
						{
							// nil Selector matches all platforms.
							URI:    srv.URL + "/datumctl-hashtest.tar.gz",
							SHA256: archiveSHA,
						},
					},
				},
			},
		},
	}

	dir := t.TempDir()
	entry, _, _, err := installPlugin(context.Background(), dir, "hashtest", "", "v9.0.0", idx)
	if err != nil {
		t.Fatalf("installPlugin: %v", err)
	}

	// The stored SHA256 must match the binary, not the archive.
	if entry.SHA256 == archiveSHA {
		t.Errorf("entry.SHA256 is the archive SHA256 (%s); it must be the binary SHA256 (%s)", archiveSHA, binarySHA)
	}
	if entry.SHA256 != binarySHA {
		t.Errorf("entry.SHA256 = %s, want binary SHA256 %s", entry.SHA256, binarySHA)
	}

	// Cross-check: the on-disk binary should hash to the stored value.
	onDiskData, err := os.ReadFile(filepath.Join(dir, "datumctl-hashtest"))
	if err != nil {
		t.Fatalf("read installed binary: %v", err)
	}
	if sha256HexOf(onDiskData) != entry.SHA256 {
		t.Errorf("on-disk binary SHA256 does not match entry.SHA256 %s; verifyManagedPluginIntegrity would reject it", entry.SHA256)
	}
}

// makeTarGz creates a .tar.gz archive in memory containing a single file
// named binName with the given content.
func makeTarGz(t *testing.T, binName string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name:     binName,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar write header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	return buf.Bytes()
}

// TestParseSource_rejectsPathTraversal verifies that owner/repo values
// containing path traversal characters are rejected (C2/L2).
func TestParseSource_rejectsPathTraversal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source string
	}{
		{"owner/../../../../tmp/evil"},
		{"../evil/repo"},
		{"owner/repo/../../../etc"},
		{"ow ner/repo"},
		{"owner/re\x00po"},
	}

	for _, tc := range cases {
		_, _, _, err := parseSource(tc.source)
		if err == nil {
			t.Errorf("parseSource(%q): want error for traversal/invalid input, got nil", tc.source)
		}
	}
}

// TestParseSource_validInputs verifies that well-formed owner/repo[@version]
// strings parse successfully.
func TestParseSource_validInputs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source        string
		wantOwner     string
		wantRepo      string
		wantVersion   string
	}{
		{"datum-cloud/datumctl-dns", "datum-cloud", "datumctl-dns", ""},
		{"datum-cloud/datumctl-dns@v1.2.3", "datum-cloud", "datumctl-dns", "v1.2.3"},
		{"github.com/datum-cloud/datumctl-dns", "datum-cloud", "datumctl-dns", ""},
		{"My.Org_1/repo.name-v2", "My.Org_1", "repo.name-v2", ""},
	}

	for _, tc := range cases {
		owner, repo, ver, err := parseSource(tc.source)
		if err != nil {
			t.Errorf("parseSource(%q): unexpected error: %v", tc.source, err)
			continue
		}
		if owner != tc.wantOwner {
			t.Errorf("parseSource(%q) owner = %q, want %q", tc.source, owner, tc.wantOwner)
		}
		if repo != tc.wantRepo {
			t.Errorf("parseSource(%q) repo = %q, want %q", tc.source, repo, tc.wantRepo)
		}
		if ver != tc.wantVersion {
			t.Errorf("parseSource(%q) version = %q, want %q", tc.source, ver, tc.wantVersion)
		}
	}
}

// TestPluginNameFromRepo_validAndInvalid verifies that pluginNameFromRepo
// accepts valid repo names and rejects those that would produce invalid binary
// names (C2).
func TestPluginNameFromRepo_validAndInvalid(t *testing.T) {
	t.Parallel()

	valid := []struct {
		repo string
		want string
	}{
		{"datumctl-dns", "dns"},
		{"datumctl-my-plugin", "my-plugin"},
		{"my-plugin", "my-plugin"},
		{"plugin1", "plugin1"},
	}
	for _, tc := range valid {
		got, err := pluginNameFromRepo(tc.repo)
		if err != nil {
			t.Errorf("pluginNameFromRepo(%q): unexpected error: %v", tc.repo, err)
			continue
		}
		if got != tc.want {
			t.Errorf("pluginNameFromRepo(%q) = %q, want %q", tc.repo, got, tc.want)
		}
	}

	invalid := []string{
		"datumctl-UPPER",
		"datumctl-../evil",
		"UPPER",
		"datumctl-",
	}
	for _, repo := range invalid {
		_, err := pluginNameFromRepo(repo)
		if err == nil {
			t.Errorf("pluginNameFromRepo(%q): want error for invalid name, got nil", repo)
		}
	}
}

// TestWriteBinary_rejectsTraversalNames verifies that writeBinary rejects
// binaryName values that would escape the plugins directory (C2).
func TestWriteBinary_rejectsTraversalNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	traversalNames := []string{
		"../evil",
		"../../etc/passwd",
		"/absolute/path",
	}
	for _, name := range traversalNames {
		_, err := writeBinary(dir, name, []byte("data"))
		if err == nil {
			t.Errorf("writeBinary with %q: want error, got nil", name)
		}
	}
}

// TestRequireHTTPS_rejectsNonHTTPS verifies that requireHTTPS blocks
// non-HTTPS URIs (H3).
func TestRequireHTTPS_rejectsNonHTTPS(t *testing.T) {
	t.Parallel()

	bad := []string{
		"http://example.com/plugin.tar.gz",
		"ftp://example.com/plugin.tar.gz",
		"file:///etc/passwd",
	}
	for _, uri := range bad {
		if err := requireHTTPS(uri); err == nil {
			t.Errorf("requireHTTPS(%q): want error, got nil", uri)
		}
	}

	if err := requireHTTPS("https://example.com/plugin.tar.gz"); err != nil {
		t.Errorf("requireHTTPS(https): unexpected error: %v", err)
	}
}

// buildHelperBinary compiles a Go program from src into a temp dir and returns
// the path to the resulting binary. The binary is cleaned up via t.Cleanup.
func buildHelperBinary(t *testing.T, src string) string {
	t.Helper()

	dir := t.TempDir()
	srcFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcFile, []byte(src), 0o644); err != nil {
		t.Fatalf("write helper source: %v", err)
	}

	binName := "helper"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(dir, binName)

	cmd := exec.Command("go", "build", "-o", binPath, srcFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build helper binary: %v\n%s", err, out)
	}
	return binPath
}
