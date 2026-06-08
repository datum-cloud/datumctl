package plugin

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/pluginstore"
)

// indexFetchUserError converts a RefreshIndex failure into a user-facing error,
// attaching actionable guidance when the cause is recognizable (e.g. a GitHub
// token in the environment being rejected by the index host).
func indexFetchUserError(err error) error {
	msg := "could not fetch the plugin index: " + err.Error()
	var fe *pluginstore.IndexFetchError
	if errors.As(err, &fe) {
		if hint := fe.Hint(); hint != "" {
			return customerrors.NewUserErrorWithHint(msg, hint)
		}
	}
	return customerrors.NewUserError(msg)
}

const (
	pluginDownloadTimeout = 60 * time.Second
	manifestReadTimeout   = 5 * time.Second
)

// reGitHubSegment matches valid GitHub owner and repo name characters.
// GitHub allows letters, digits, hyphens, underscores, and dots.
var reGitHubSegment = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// rePluginName matches a valid kebab-case plugin name as installed on disk
// (e.g. "dns", "my-plugin").  Must start with a letter or digit and contain
// only lowercase letters, digits, and hyphens.
var rePluginName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// pluginAssetName returns the archive filename for the current OS/arch.
// Follows the same naming convention as updatecheck.archiveName() (capitalised OS, goarch-style arch).
// Example: "datumctl-dns_Linux_x86_64.tar.gz"
func pluginAssetName(pluginName, version string) (string, error) {
	var osName string
	switch runtime.GOOS {
	case "linux":
		osName = "Linux"
	case "darwin":
		osName = "Darwin"
	case "windows":
		osName = "Windows"
	default:
		return "", fmt.Errorf("unsupported OS %q", runtime.GOOS)
	}

	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "x86_64"
	case "386":
		arch = "i386"
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("unsupported architecture %q", runtime.GOARCH)
	}

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s.%s", pluginName, osName, arch, ext), nil
}

// fetchLatestTag resolves the latest GitHub Release tag via redirect.
// It issues a HEAD request to releases/latest and parses the tag from the final URL.
// FetchLatestTag resolves the latest release tag for a GitHub owner/repo.
// Exported so callers outside this package can pass it to pluginstore.RefreshIndex.
func FetchLatestTag(ctx context.Context, owner, repo string) (string, error) {
	return fetchLatestTag(ctx, owner, repo)
}

func fetchLatestTag(ctx context.Context, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://github.com/%s/%s/releases/latest", owner, repo)

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects automatically; net/http handles this.
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "datumctl-plugin-install")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release tag: %w", err)
	}
	defer resp.Body.Close()

	// After following redirect, the final URL has the tag at the end.
	finalURL := resp.Request.URL.String()
	tag := filepath.Base(finalURL)
	if tag == "" || tag == "." || tag == "/" {
		return "", fmt.Errorf("could not parse release tag from URL %q", finalURL)
	}
	if !strings.HasPrefix(tag, "v") {
		return "", fmt.Errorf("unexpected release tag format %q (expected vX.Y.Z)", tag)
	}
	return tag, nil
}

// fetchChecksums downloads and parses checksums.txt from a GitHub Release.
// Returns a map of filename → sha256hex.
func fetchChecksums(ctx context.Context, owner, repo, tag string) (map[string]string, error) {
	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/checksums.txt", owner, repo, tag)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "datumctl-plugin-install")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download checksums.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("couldn't verify %s/%s@%s — the release doesn't include a checksums file.\nThe plugin author needs to add checksums.txt to the GitHub Release before it can be installed safely", owner, repo, tag)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read checksums.txt: %w", err)
	}

	checksums := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// goreleaser format: "<sha256hex>  <filename>" (two spaces) or "<sha256hex> <filename>" (one space)
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		checksums[fields[1]] = fields[0]
	}
	return checksums, nil
}

// downloadAndVerify downloads the release archive, verifies its SHA256 against
// the checksums map, and extracts the plugin binary.
// Returns the binary bytes and the sha256hex of the archive.
func downloadAndVerify(ctx context.Context, owner, repo, tag, assetName string, checksums map[string]string) (binaryBytes []byte, archiveSHA256 string, err error) {
	expectedSHA, ok := checksums[assetName]
	if !ok {
		return nil, "", fmt.Errorf("no checksum found for %q in checksums.txt", assetName)
	}

	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repo, tag, assetName)
	httpClient := &http.Client{Timeout: pluginDownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "datumctl-plugin-install")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("download %s: %w", assetName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download %s: HTTP %d", assetName, resp.StatusCode)
	}

	archiveBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read archive: %w", err)
	}

	// Verify archive SHA256.
	sum := sha256.Sum256(archiveBytes)
	gotSHA := hex.EncodeToString(sum[:])
	if !strings.EqualFold(gotSHA, expectedSHA) {
		return nil, "", fmt.Errorf("SHA256 mismatch for %s: expected %s, got %s", assetName, expectedSHA, gotSHA)
	}

	// Extract the binary from the archive.
	// The binary name inside the archive is the plugin name without the archive extension.
	// e.g. for "datumctl-dns_Linux_x86_64.tar.gz", the binary is "datumctl-dns"
	pluginBinName := binaryNameFromAsset(assetName)
	var extracted []byte
	if strings.HasSuffix(assetName, ".tar.gz") {
		extracted, err = extractBinaryFromTarGz(bytes.NewReader(archiveBytes), pluginBinName)
	} else if strings.HasSuffix(assetName, ".zip") {
		extracted, err = extractBinaryFromZip(archiveBytes, pluginBinName)
	} else {
		return nil, "", fmt.Errorf("unsupported archive format: %s", assetName)
	}
	if err != nil {
		return nil, "", fmt.Errorf("extract binary from archive: %w", err)
	}

	return extracted, gotSHA, nil
}


func extractBinaryFromTarGz(r io.Reader, binName string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gunzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("binary %q not found in archive", binName)
		}
		if err != nil {
			return nil, fmt.Errorf("read tar archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != binName {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", binName, err)
		}
		return data, nil
	}
}

func extractBinaryFromZip(archiveBytes []byte, binName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}
	for _, f := range zr.File {
		if filepath.Base(f.Name) != binName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s in zip: %w", binName, err)
		}
		data, copyErr := io.ReadAll(rc)
		rc.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("extract %s: %w", binName, copyErr)
		}
		return data, nil
	}
	return nil, fmt.Errorf("binary %q not found in archive", binName)
}

// binaryNameFromAsset extracts the expected binary name from an asset filename.
// "datumctl-dns_Linux_x86_64.tar.gz" → "datumctl-dns"
// "datumctl-dns_Windows_x86_64.zip"  → "datumctl-dns.exe"
func binaryNameFromAsset(assetName string) string {
	name := strings.TrimSuffix(assetName, ".tar.gz")
	name = strings.TrimSuffix(name, ".zip")
	// The plugin name is everything before the first underscore.
	if i := strings.Index(name, "_"); i >= 0 {
		name = name[:i]
	}
	if strings.Contains(assetName, "_Windows_") {
		name += ".exe"
	}
	return name
}

// downloadAndVerifyURI downloads the archive at uri, verifies its SHA256 against
// sha256hex, and returns the raw archive bytes. This is the index-based download
// path; it is distinct from downloadAndVerify (which takes GitHub asset names).
// Defense-in-depth: rejects non-HTTPS URIs (H3).
func downloadAndVerifyURI(ctx context.Context, uri, sha256hex string) ([]byte, error) {
	if err := requireHTTPS(uri); err != nil {
		return nil, err
	}

	httpClient := &http.Client{Timeout: pluginDownloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "datumctl-plugin-install")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download archive: HTTP %d", resp.StatusCode)
	}

	archiveBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}

	sum := sha256.Sum256(archiveBytes)
	gotSHA := hex.EncodeToString(sum[:])
	if !strings.EqualFold(gotSHA, sha256hex) {
		return nil, fmt.Errorf("SHA256 mismatch: expected %s, got %s", sha256hex, gotSHA)
	}

	return archiveBytes, nil
}

// readPluginManifest writes binaryPath to a temp file (if needed), runs it with
// --plugin-manifest, and parses the JSON output. Returns nil, nil if the binary
// exits non-zero or produces no valid JSON.
func readPluginManifest(binaryPath string) (*pluginstore.PluginManifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), manifestReadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--plugin-manifest")
	out, err := cmd.Output()
	if err != nil {
		// Non-zero exit — treat as "no manifest".
		return nil, nil
	}
	if len(out) == 0 {
		return nil, nil
	}

	var m pluginstore.PluginManifest
	if err := json.Unmarshal(out, &m); err != nil {
		return nil, fmt.Errorf("parse plugin manifest: %w", err)
	}
	return &m, nil
}

// readPluginManifestFromBytes writes binary bytes to a temp file, runs it with
// --plugin-manifest, and returns the parsed manifest.
func readPluginManifestFromBytes(binaryData []byte) (*pluginstore.PluginManifest, error) {
	tmp, err := os.CreateTemp("", "datumctl-plugin-verify-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file for manifest check: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(binaryData); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("write temp binary: %w", err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return nil, fmt.Errorf("chmod temp binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return nil, fmt.Errorf("close temp binary: %w", err)
	}

	return readPluginManifest(tmp.Name())
}

// checkCompatibility validates a manifest against the running datumctl version and API version.
// Returns (warn, nil) for soft warnings, ("", err) for hard errors.
// At install time, hard errors prevent installation.
// At invocation time, callers use warn for non-blocking messages and err for blocking.
func checkCompatibility(m *pluginstore.PluginManifest, currentVersion string, currentAPIVersion int) (warn string, err error) {
	if m == nil {
		return "", nil
	}

	// Check min_datumctl_version.
	if m.MinDatumctlVersion != "" && semver.IsValid(m.MinDatumctlVersion) {
		if semver.IsValid(currentVersion) && semver.Compare(currentVersion, m.MinDatumctlVersion) < 0 {
			return "", fmt.Errorf("plugin requires datumctl %s or newer (current: %s); upgrade datumctl first",
				m.MinDatumctlVersion, currentVersion)
		}
	}

	// Check min_api_version.
	if m.MinAPIVersion > 0 && currentAPIVersion < m.MinAPIVersion {
		return "", fmt.Errorf("plugin requires API version %d or higher (current host API version: %d); upgrade datumctl to use this plugin",
			m.MinAPIVersion, currentAPIVersion)
	}

	return "", nil
}

// writeBinary writes binaryData to pluginsDir/binaryName with executable permissions.
// Defense-in-depth: rejects binaryName if it contains path separators or is not
// a local (non-traversal) filename component (C2). The check is applied to the
// original binaryName before any path manipulation.
func writeBinary(pluginsDir, binaryName string, binaryData []byte) (string, error) {
	// Reject names that would escape the plugins directory.
	// filepath.IsLocal rejects absolute paths, "..", and names with path separators.
	// Comparing binaryName to its Base additionally rejects names that contain
	// directory components (e.g. "a/b") which IsLocal would otherwise allow.
	if !filepath.IsLocal(binaryName) || binaryName != filepath.Base(binaryName) {
		return "", fmt.Errorf("plugin binary name %q is invalid: must be a simple filename with no path separators", binaryName)
	}

	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return "", fmt.Errorf("create plugins directory: %w", err)
	}

	destPath := filepath.Join(pluginsDir, binaryName)
	tmp := destPath + ".tmp"
	if err := os.WriteFile(tmp, binaryData, 0o755); err != nil {
		return "", fmt.Errorf("write plugin binary: %w", err)
	}
	if err := os.Rename(tmp, destPath); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("install plugin binary: %w", err)
	}
	return destPath, nil
}

// requireHTTPS returns an error if rawURL does not use the https scheme.
// Used as a defense-in-depth guard before issuing HTTP requests (H3).
func requireHTTPS(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URI %q: %w", rawURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("insecure URI %q: only HTTPS download URIs are supported", rawURL)
	}
	return nil
}

// parseSource parses "owner/repo[@version]" into owner, repo, version.
// version is empty if not specified.
// owner and repo are validated against ^[a-zA-Z0-9._-]+$ to prevent path
// traversal (C2/L2).
func parseSource(source string) (owner, repo, version string, err error) {
	// Strip "github.com/" prefix if present.
	source = strings.TrimPrefix(source, "github.com/")

	// Split off @version suffix.
	sourcePart, versionPart, _ := strings.Cut(source, "@")
	version = versionPart

	parts := strings.SplitN(sourcePart, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", fmt.Errorf("invalid plugin source %q: expected owner/repo[@version]", source)
	}

	owner, repo = parts[0], parts[1]

	// Validate charset before any URL or path construction.
	if !reGitHubSegment.MatchString(owner) {
		return "", "", "", fmt.Errorf("invalid plugin source: owner %q contains disallowed characters (only letters, digits, hyphens, underscores, and dots are allowed)", owner)
	}
	if !reGitHubSegment.MatchString(repo) {
		return "", "", "", fmt.Errorf("invalid plugin source: repo %q contains disallowed characters (only letters, digits, hyphens, underscores, and dots are allowed)", repo)
	}

	return owner, repo, version, nil
}

// pluginNameFromRepo derives the plugin name from the repo name and validates it.
// "datumctl-dns" → "dns", "datum-dns" → "datum-dns" (no stripping)
// Returns an error if the resulting name does not match ^[a-z0-9][a-z0-9-]*$ (C2).
func pluginNameFromRepo(repoName string) (string, error) {
	name := strings.TrimPrefix(repoName, "datumctl-")
	if !rePluginName.MatchString(name) {
		return "", fmt.Errorf("plugin name %q derived from repo %q is invalid: must start with a lowercase letter or digit and contain only lowercase letters, digits, and hyphens", name, repoName)
	}
	return name, nil
}

// isUpdateAvailable returns true when latest is a valid semver tag newer than installed.
func isUpdateAvailable(installed, latest string) bool {
	if latest == "" || !semver.IsValid(installed) || !semver.IsValid(latest) {
		return false
	}
	return semver.Compare(installed, latest) < 0
}

// InstallPlugin performs the full install flow for a single named plugin from
// the curated index. pluginName must match a Plugin.Name entry in idx.
// The returned string is the plugin's short name. The binary path is not
// exposed through this exported wrapper; callers that need L1 cleanup should
// use installPlugin directly.
func InstallPlugin(ctx context.Context, pluginsDir, pluginName, version, currentVersion string, idx *pluginstore.CachedIndex) (*pluginstore.InstalledPlugin, string, error) {
	entry, name, _, err := installPlugin(ctx, pluginsDir, pluginName, version, currentVersion, idx)
	return entry, name, err
}

// sha256HexOf returns the lowercase hex-encoded SHA256 digest of b.
func sha256HexOf(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// newBytesReader wraps b in a *bytes.Reader for callers that need an io.Reader.
func newBytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

