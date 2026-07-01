package pluginstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

const indexStaleTTL = time.Hour

// Read caps for plugin network fetches. SHA256 verification runs over the
// COMPRESSED, attacker-authored bytes, so it offers no protection against an
// unbounded download or a decompression bomb — these caps do.
const (
	// MaxManifestBytes caps catalog manifest and checksum file reads. Manifests
	// and checksums.txt files are small; this is generous while still bounding a
	// hostile or runaway response. (serviceaccount.go uses 1 MiB as precedent;
	// catalogs can list many plugins, so we allow a few MiB.)
	MaxManifestBytes int64 = 5 << 20 // 5 MiB

	// MaxArchiveBytes caps a downloaded plugin release archive. Plugin binaries
	// can be large, but an archive in the hundreds of MiB is not legitimate.
	MaxArchiveBytes int64 = 256 << 20 // 256 MiB

	// MaxDecompressedFileBytes caps a single decompressed archive entry, so a
	// gzip/zip bomb cannot expand a small archive into an unbounded write.
	MaxDecompressedFileBytes int64 = 512 << 20 // 512 MiB
)

// ReadCapped reads from r up to max bytes and returns an error if the source
// would exceed that cap. It reads max+1 bytes so an over-cap body is detected
// rather than silently truncated.
func ReadCapped(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("response exceeds the maximum allowed size of %d bytes", max)
	}
	return data, nil
}

// SafeCheckRedirect re-validates every HTTP redirect hop: the target must be
// HTTPS and must not resolve to a private, loopback, link-local, or otherwise
// non-routable address. It hardens third-party catalog and download fetches
// against TLS-downgrade and SSRF-via-redirect, where only the initial URL would
// otherwise be checked. The default 10-redirect limit is preserved.
func SafeCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	return validateFetchURL(req.URL)
}

// safeHTTPClient returns an *http.Client that enforces SafeCheckRedirect on
// every hop. timeout of 0 leaves the client timeout unset (callers relying on a
// context deadline).
func safeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: SafeCheckRedirect}
}

// validateFetchURL ensures u is HTTPS and does not target a blocked address.
func validateFetchURL(u *url.URL) error {
	if u == nil {
		return fmt.Errorf("missing redirect URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("refusing redirect to non-HTTPS URL %q: only HTTPS is supported", u.Redacted())
	}
	return validateHostNotBlocked(u.Hostname())
}

// validateHostNotBlocked rejects a host that is, or resolves to, a private,
// loopback, link-local, unspecified, or multicast address (SSRF guard). An IP
// literal is checked directly; a hostname is resolved and every returned
// address must be acceptable.
func validateHostNotBlocked(host string) error {
	if host == "" {
		return fmt.Errorf("refusing to connect to a URL with no host")
	}
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		resolved, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("resolve host %q: %w", host, err)
		}
		ips = resolved
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("refusing to connect to %q: it resolves to a private or non-routable address (%s)", host, ip)
		}
	}
	return nil
}

// isBlockedIP reports whether ip is in a range that must never be reached by a
// plugin fetch: loopback (127.0.0.0/8, ::1), RFC 1918 / ULA private
// (10/8, 172.16/12, 192.168/16, fc00::/7), link-local (169.254/16, fe80::/10),
// the unspecified address (0.0.0.0, ::), and multicast.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}

// IndexURL is the location of the default (official) plugin catalog. Override
// with DATUMCTL_PLUGIN_INDEX_URL for testing or custom deployments. This only
// re-points the default catalog; user-registered catalogs are unaffected.
var IndexURL = "https://raw.githubusercontent.com/datum-cloud/datumctl-plugins/main/index.yaml"

func init() {
	if u := os.Getenv("DATUMCTL_PLUGIN_INDEX_URL"); u != "" {
		IndexURL = u
	}
}

// indexURLSchemeError is set at init-time when DATUMCTL_PLUGIN_INDEX_URL is
// present but not HTTPS. RefreshCatalog checks this for the default catalog
// before making any request.
var indexURLSchemeError error

func init() {
	if IndexURL != "" && !strings.HasPrefix(IndexURL, "https://") {
		indexURLSchemeError = fmt.Errorf(
			"DATUMCTL_PLUGIN_INDEX_URL %q uses a non-HTTPS scheme; only HTTPS index URLs are supported",
			IndexURL,
		)
	}
}

// legacyIndexFileName is the pre-marketplace single-catalog cache file. It is
// still read as a fallback for the default catalog so an upgrade does not lose
// the existing cached index.
const legacyIndexFileName = "plugin-index.json"

// IndexPath returns the legacy default-catalog cache path. Retained for
// backward compatibility.
func IndexPath() (string, error) {
	dir, err := PluginsDir("")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, legacyIndexFileName), nil
}

// LoadCatalogIndex reads the cached index for a named catalog. A missing or
// unparseable cache yields a zero-value CachedIndex (which IsStale reports true
// for) and no error. For the official catalog, older cache locations are read
// when the current per-catalog cache is absent: the pre-rename "default" cache
// directory and the legacy single-catalog plugin-index.json.
func LoadCatalogIndex(pluginsDir, name string) (*CachedIndex, error) {
	name = CanonicalCatalogName(name)
	path, err := CatalogIndexPath(pluginsDir, name)
	if err != nil {
		return nil, err
	}
	data, readErr := os.ReadFile(path)
	if os.IsNotExist(readErr) && name == OfficialCatalogName {
		// Fall back to the pre-rename cache dir, then the legacy single cache.
		for _, legacy := range []string{
			filepath.Join(pluginsDir, "indexes", legacyOfficialAlias, "index.json"),
			filepath.Join(pluginsDir, legacyIndexFileName),
		} {
			if data, readErr = os.ReadFile(legacy); !os.IsNotExist(readErr) {
				break
			}
		}
	}
	if os.IsNotExist(readErr) {
		return &CachedIndex{}, nil
	}
	if readErr != nil {
		return nil, readErr
	}
	var idx CachedIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		// Old format or corrupt — return zero-value so caller knows to refresh.
		return &CachedIndex{}, nil
	}
	return &idx, nil
}

// SaveCatalogIndex writes a catalog's cached index to disk atomically.
func SaveCatalogIndex(pluginsDir, name string, idx *CachedIndex) error {
	path, err := CatalogIndexPath(pluginsDir, name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadIndex reads the cached default-catalog index. Backward-compatible wrapper
// over LoadCatalogIndex for the default catalog.
func LoadIndex() (*CachedIndex, error) {
	dir, err := PluginsDir("")
	if err != nil {
		return nil, err
	}
	return LoadCatalogIndex(dir, OfficialCatalogName)
}

// SaveIndex writes the default-catalog index. Backward-compatible wrapper.
func SaveIndex(idx *CachedIndex) error {
	dir, err := PluginsDir("")
	if err != nil {
		return err
	}
	return SaveCatalogIndex(dir, OfficialCatalogName, idx)
}

// IsStale reports whether the index is missing or older than the TTL.
func IsStale(idx *CachedIndex) bool {
	return idx == nil || idx.RefreshedAt.IsZero() || time.Since(idx.RefreshedAt) > indexStaleTTL
}

// FindInIndex returns the Plugin whose Name (from ObjectMeta) exactly matches
// pluginName, or nil if not found.
func FindInIndex(idx *CachedIndex, pluginName string) *Plugin {
	if idx == nil {
		return nil
	}
	for i := range idx.Plugins {
		if idx.Plugins[i].Name == pluginName {
			return &idx.Plugins[i]
		}
	}
	return nil
}

// RefreshIndex refreshes the default catalog. Backward-compatible wrapper over
// RefreshCatalog.
//
// Three-case return contract:
//   - (non-nil, nil)       — success
//   - (non-nil, non-nil)   — fetch failed but stale cache exists on disk
//   - (nil, non-nil)       — fetch failed and no cache available
func RefreshIndex(ctx context.Context) (*CachedIndex, error) {
	dir, err := PluginsDir("")
	if err != nil {
		return nil, err
	}
	return RefreshCatalog(ctx, dir, OfficialCatalog())
}

// RefreshCatalog fetches a catalog's manifest (from an HTTPS URL, a GitHub
// owner/repo, or a local path), parses it, verifies every download URI is
// HTTPS, caches it under indexes/<name>/index.json, and returns the result.
//
// It follows the same three-case return contract as RefreshIndex: on failure it
// attempts to return a stale cache alongside the error, or (nil, err) when no
// cache exists.
func RefreshCatalog(ctx context.Context, pluginsDir string, cat Catalog) (*CachedIndex, error) {
	// The default catalog respects the init-time DATUMCTL_PLUGIN_INDEX_URL scheme
	// check before any network request.
	if cat.Name == OfficialCatalogName && indexURLSchemeError != nil {
		return nil, indexURLSchemeError
	}

	resolved, err := ResolveCatalogSource(cat.Source)
	if err != nil {
		return degradedCatalogFallback(pluginsDir, cat.Name, err)
	}

	raw, err := fetchCatalogManifest(ctx, resolved)
	if err != nil {
		return degradedCatalogFallback(pluginsDir, cat.Name, err)
	}

	// raw is already capped to MaxManifestBytes by fetchCatalogManifest, which
	// bounds the YAML parser's input (sigs.k8s.io/yaml -> yaml.v2 has no
	// alias-expansion limit, so a small body bound is the cheap guard available).
	var list PluginList
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return degradedCatalogFallback(pluginsDir, cat.Name, fmt.Errorf("parse plugin index: %w", err))
	}

	// Reject the entire index if any platform URI uses a non-HTTPS scheme.
	if err := validateIndexURIs(list.Items); err != nil {
		return degradedCatalogFallback(pluginsDir, cat.Name, fmt.Errorf("invalid plugin index: %w", err))
	}

	idx := &CachedIndex{
		RefreshedAt: time.Now(),
		Header:      list.HeaderFor(),
		Plugins:     list.Items,
	}
	_ = SaveCatalogIndex(pluginsDir, cat.Name, idx)
	return idx, nil
}

// fetchCatalogManifest retrieves raw manifest bytes for a resolved source. Local
// sources are read from disk; remote sources are fetched over HTTPS. For GitHub
// shorthand sources, a 404 on the "main" branch falls back to "master".
func fetchCatalogManifest(ctx context.Context, resolved ResolvedSource) ([]byte, error) {
	if resolved.IsLocal {
		data, err := os.ReadFile(resolved.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("read local catalog manifest: %w", err)
		}
		// Apply the same cap as remote fetches before the bytes reach the YAML
		// parser (which has no alias-expansion limit).
		if int64(len(data)) > MaxManifestBytes {
			return nil, fmt.Errorf("local catalog manifest exceeds the maximum allowed size of %d bytes", MaxManifestBytes)
		}
		return data, nil
	}

	raw, status, err := httpGetManifest(ctx, resolved.FetchURL)
	if err == nil && status == http.StatusOK {
		return raw, nil
	}

	// GitHub branch fallback: main -> master.
	if resolved.GitHubOwnerRepo != "" && (status == http.StatusNotFound || status == 0) {
		masterURL := gitHubRawURL(resolved.GitHubOwnerRepo, "master")
		raw2, status2, err2 := httpGetManifest(ctx, masterURL)
		if err2 == nil && status2 == http.StatusOK {
			return raw2, nil
		}
	}

	if err != nil {
		return nil, err
	}
	var tokenSource string
	if isGitHubHost(resolved.FetchURL) {
		if token, src := githubTokenWithSource(); token != "" {
			tokenSource = src
		}
	}
	return nil, &IndexFetchError{
		URL:         resolved.FetchURL,
		StatusCode:  status,
		Status:      fmt.Sprintf("%d %s", status, http.StatusText(status)),
		TokenSource: tokenSource,
	}
}

// httpGetManifest performs a single HTTPS GET and returns the body and status.
// A GitHub token from the environment is attached only for GitHub-owned hosts,
// so credentials are never sent to third-party catalog hosts.
func httpGetManifest(ctx context.Context, rawURL string) (body []byte, status int, err error) {
	if err := requireHTTPSURL(rawURL); err != nil {
		return nil, 0, err
	}

	httpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build index request: %w", err)
	}

	if isGitHubHost(rawURL) {
		if token, _ := githubTokenWithSource(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	req.Header.Set("User-Agent", "datumctl-plugin-index")

	resp, err := safeHTTPClient(0).Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch plugin index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}

	// Cap the manifest body: it is unverified, attacker-controllable, and feeds a
	// YAML parser that lacks alias-expansion limits.
	raw, err := ReadCapped(resp.Body, MaxManifestBytes)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read index response: %w", err)
	}
	return raw, resp.StatusCode, nil
}

// requireHTTPSURL returns an error if rawURL is not HTTPS.
func requireHTTPSURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid catalog URL %q: %w", rawURL, err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("insecure catalog URL %q: only HTTPS is supported", rawURL)
	}
	return nil
}

// isGitHubHost reports whether rawURL points at a GitHub-owned host, where it is
// safe to attach a GitHub token.
func isGitHubHost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "github.com" ||
		host == "raw.githubusercontent.com" ||
		strings.HasSuffix(host, ".githubusercontent.com")
}

// validateIndexURIs checks that every platform URI in the plugin list uses
// the https:// scheme. Returns an error describing the first offending URI.
func validateIndexURIs(plugins []Plugin) error {
	for i := range plugins {
		for j := range plugins[i].Spec.Platforms {
			uri := plugins[i].Spec.Platforms[j].URI
			if uri == "" {
				continue
			}
			u, parseErr := url.Parse(uri)
			if parseErr != nil {
				return fmt.Errorf("plugin %q platform %d has an invalid URI %q: %w", plugins[i].Name, j, uri, parseErr)
			}
			if u.Scheme != "https" {
				return fmt.Errorf("plugin %q platform %d URI %q uses a non-HTTPS scheme; only HTTPS download URIs are supported", plugins[i].Name, j, uri)
			}
		}
	}
	return nil
}

// degradedCatalogFallback tries to load a stale cache for the named catalog and
// returns it alongside the original error. If no cache exists, returns (nil, err).
func degradedCatalogFallback(pluginsDir, name string, origErr error) (*CachedIndex, error) {
	cached, loadErr := LoadCatalogIndex(pluginsDir, name)
	if loadErr != nil || cached == nil || cached.RefreshedAt.IsZero() {
		return nil, origErr
	}
	return cached, origErr
}

// githubTokenWithSource returns a GitHub personal access token from the
// environment along with the name of the variable it came from (empty when no
// token is set).
func githubTokenWithSource() (token, source string) {
	if t := os.Getenv("DATUMCTL_GITHUB_TOKEN"); t != "" {
		return t, "DATUMCTL_GITHUB_TOKEN"
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, "GITHUB_TOKEN"
	}
	return "", ""
}

// IndexFetchError is returned by RefreshCatalog when a catalog host responds
// with a non-OK HTTP status. It carries enough context for the command layer to
// render actionable guidance via Hint.
type IndexFetchError struct {
	URL         string
	StatusCode  int
	Status      string // HTTP status text, e.g. "404 Not Found"
	TokenSource string // env var the Authorization token came from, "" if none
}

func (e *IndexFetchError) Error() string {
	return fmt.Sprintf("the plugin index host returned HTTP %s", e.Status)
}

// Hint returns actionable guidance for resolving the failure, or "" when none
// applies. The common case: a GitHub token in the environment is sent to the
// public index host, which rejects it with a 401/403/404.
func (e *IndexFetchError) Hint() string {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		if e.TokenSource != "" {
			return fmt.Sprintf(
				"A GitHub token from $%s is being sent to the index host, which is the likely cause. "+
					"The public plugin index needs no authentication; unset that variable and retry.",
				e.TokenSource)
		}
	}
	return ""
}
