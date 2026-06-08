package pluginstore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

const indexStaleTTL = time.Hour

// IndexURL is the location of the remote plugin index. Override with
// DATUMCTL_PLUGIN_INDEX_URL for testing or custom deployments.
var IndexURL = "https://raw.githubusercontent.com/datum-cloud/datumctl-plugins/main/index.yaml"

func init() {
	if u := os.Getenv("DATUMCTL_PLUGIN_INDEX_URL"); u != "" {
		IndexURL = u
	}
}

// indexURLSchemeError is set at init-time when DATUMCTL_PLUGIN_INDEX_URL is
// present but not HTTPS. RefreshIndex checks this before making any request.
var indexURLSchemeError error

func init() {
	if IndexURL != "" && !strings.HasPrefix(IndexURL, "https://") {
		indexURLSchemeError = fmt.Errorf(
			"DATUMCTL_PLUGIN_INDEX_URL %q uses a non-HTTPS scheme; only HTTPS index URLs are supported",
			IndexURL,
		)
	}
}

// IndexPath returns the path to the local plugin index file.
func IndexPath() (string, error) {
	dir, err := PluginsDir("")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "plugin-index.json"), nil
}

// LoadIndex reads the cached index from disk. If the file does not exist or
// cannot be parsed (e.g. old format), it returns a zero-value CachedIndex
// (which IsStale returns true for) and no error.
func LoadIndex() (*CachedIndex, error) {
	path, err := IndexPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &CachedIndex{}, nil
	}
	if err != nil {
		return nil, err
	}
	var idx CachedIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		// Old format or corrupt — return zero-value so caller knows to refresh.
		return &CachedIndex{}, nil
	}
	return &idx, nil
}

// SaveIndex writes the index to disk atomically.
func SaveIndex(idx *CachedIndex) error {
	path, err := IndexPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
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

// RefreshIndex fetches IndexURL, parses the PluginList, saves, and returns the
// result.
//
// Three-case return contract:
//   - (non-nil, nil)       — success
//   - (non-nil, non-nil)   — fetch failed but stale cache exists on disk
//   - (nil, non-nil)       — fetch failed and no cache available
func RefreshIndex(ctx context.Context) (*CachedIndex, error) {
	// H3: reject non-HTTPS index URL before any network request.
	if indexURLSchemeError != nil {
		return nil, indexURLSchemeError
	}

	httpCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, IndexURL, nil)
	if err != nil {
		return degradedFallback(fmt.Errorf("build index request: %w", err))
	}

	// Attach GitHub token if available.
	token, tokenSource := githubTokenWithSource()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("User-Agent", "datumctl-plugin-index")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return degradedFallback(fmt.Errorf("fetch plugin index: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return degradedFallback(&IndexFetchError{
			URL:         IndexURL,
			StatusCode:  resp.StatusCode,
			Status:      resp.Status,
			TokenSource: tokenSource,
		})
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return degradedFallback(fmt.Errorf("read index response: %w", err))
	}

	var list PluginList
	if err := yaml.Unmarshal(raw, &list); err != nil {
		return degradedFallback(fmt.Errorf("parse plugin index: %w", err))
	}

	// H3: reject the entire index if any platform URI uses a non-HTTPS scheme.
	if err := validateIndexURIs(list.Items); err != nil {
		return degradedFallback(fmt.Errorf("invalid plugin index: %w", err))
	}

	idx := &CachedIndex{
		RefreshedAt: time.Now(),
		Plugins:     list.Items,
	}
	_ = SaveIndex(idx)
	return idx, nil
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

// degradedFallback tries to load a stale cache and returns it alongside the
// original error. If no cache exists, returns (nil, err).
func degradedFallback(origErr error) (*CachedIndex, error) {
	cached, loadErr := LoadIndex()
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

// IndexFetchError is returned by RefreshIndex when the index host responds with
// a non-OK HTTP status. It carries enough context for the command layer to
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
