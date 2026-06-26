package pluginstore

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// OfficialCatalogName is the reserved name of Datum's curated, official catalog.
// It is always present, trusted with no user action, and the only catalog that
// carries the "official" trust badge.
const OfficialCatalogName = "datum"

// legacyOfficialAlias is the pre-rename name of the official catalog. It is kept
// as a hidden alias so existing config and install records, and the
// "default/<plugin>" addressing form, continue to resolve to the official
// catalog after the rename to "datum".
const legacyOfficialAlias = "default"

// CanonicalCatalogName maps the legacy "default" alias to the official catalog
// name "datum"; every other name passes through unchanged. Apply it wherever a
// user- or record-supplied catalog name is resolved or displayed.
func CanonicalCatalogName(name string) string {
	if name == legacyOfficialAlias {
		return OfficialCatalogName
	}
	return name
}

// Catalog type values recorded in indexes.json and shown in `plugin index list`.
const (
	CatalogTypeOfficial = "official"
	CatalogTypeCustom   = "custom"
)

// Trust badge values shown wherever a plugin or catalog appears.
const (
	TrustOfficial   = "official"
	TrustThirdParty = "third-party"
)

// indexesFileName is the registry of user-registered catalogs.
const indexesFileName = "indexes.json"

// reservedCatalogNames cannot be registered by users; they are reserved so a
// third-party catalog cannot present itself as the official one.
var reservedCatalogNames = map[string]bool{
	OfficialCatalogName: true, // "datum"
	legacyOfficialAlias: true, // "default"
	"official":          true,
}

// reCatalogName matches a valid catalog name. The same character class is used
// for plugin names; restricting it to lowercase letters, digits, and hyphens
// also guarantees a catalog name is a safe single path component for its cache
// directory (no separators, no "..", no leading dot).
var reCatalogName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// reOwnerRepo matches a bare GitHub "owner/repo" shorthand source.
var reOwnerRepo = regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)

// Catalog is one registered plugin catalog (called an "index" in command
// syntax). The default catalog is synthesized at load time; user-registered
// catalogs are persisted in indexes.json.
type Catalog struct {
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Type        string    `json:"type"`
	TrustedAt   time.Time `json:"trusted_at,omitempty"`
	LastUpdated time.Time `json:"last_updated,omitempty"`

	// Cached catalog-level header, copied from the source manifest on refresh so
	// listings can show a friendly identity without re-fetching.
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Homepage    string `json:"homepage,omitempty"`

	// Managed marks a catalog pre-seeded by managed (enterprise) configuration.
	// Managed catalogs are not persisted to indexes.json and cannot be removed
	// by the user. Not serialized.
	Managed bool `json:"-"`
}

// IsOfficial reports whether this is the reserved official catalog.
func (c *Catalog) IsOfficial() bool {
	return c.Type == CatalogTypeOfficial
}

// Trust returns the trust badge string for this catalog: "official" for the
// reserved default catalog, "third-party" for everything else.
func (c *Catalog) Trust() string {
	if c.IsOfficial() {
		return TrustOfficial
	}
	return TrustThirdParty
}

// DefaultCatalog returns the synthesized official catalog. Its source tracks
// IndexURL, so DATUMCTL_PLUGIN_INDEX_URL continues to override it.
func OfficialCatalog() Catalog {
	return Catalog{
		Name:        OfficialCatalogName,
		Source:      IndexURL,
		Type:        CatalogTypeOfficial,
		Description: "Datum-curated plugins",
	}
}

// Registry is the in-memory view of all registered catalogs. It always presents
// the default catalog first, followed by managed pre-seeds, followed by the
// user's own registered catalogs.
type Registry struct {
	Catalogs []Catalog `json:"catalogs"`
}

// Find returns the catalog with the given name, or nil. The legacy "default"
// alias resolves to the official "datum" catalog.
func (r *Registry) Find(name string) *Catalog {
	name = CanonicalCatalogName(name)
	for i := range r.Catalogs {
		if r.Catalogs[i].Name == name {
			return &r.Catalogs[i]
		}
	}
	return nil
}

// Custom returns the user-registered catalogs (excludes default and managed).
// These are the entries persisted to indexes.json.
func (r *Registry) Custom() []Catalog {
	var out []Catalog
	for _, c := range r.Catalogs {
		if CanonicalCatalogName(c.Name) == OfficialCatalogName || c.Managed {
			continue
		}
		out = append(out, c)
	}
	return out
}

// IndexesPath returns <pluginsDir>/indexes.json.
func IndexesPath(pluginsDir string) string {
	return filepath.Join(pluginsDir, indexesFileName)
}

// CatalogCacheDir returns <pluginsDir>/indexes/<name>/, validating name as a
// safe single path component (defense in depth against path traversal). The
// legacy "default" alias is canonicalized to "datum" so it shares one cache.
func CatalogCacheDir(pluginsDir, name string) (string, error) {
	name = CanonicalCatalogName(name)
	if !reCatalogName.MatchString(name) {
		return "", fmt.Errorf("invalid catalog name %q", name)
	}
	return filepath.Join(pluginsDir, "indexes", name), nil
}

// CatalogIndexPath returns the cached index.json path for a catalog.
func CatalogIndexPath(pluginsDir, name string) (string, error) {
	dir, err := CatalogCacheDir(pluginsDir, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "index.json"), nil
}

// ValidateCatalogName checks that name is well-formed and not reserved. Used by
// `plugin index add`.
func ValidateCatalogName(name string) error {
	if name == "" {
		return fmt.Errorf("catalog name is required")
	}
	if reservedCatalogNames[name] {
		return fmt.Errorf("%q is a reserved catalog name and cannot be registered", name)
	}
	if !reCatalogName.MatchString(name) {
		return fmt.Errorf("invalid catalog name %q: must start with a lowercase letter or digit and contain only lowercase letters, digits, and hyphens", name)
	}
	return nil
}

// loadPersistedCatalogs reads indexes.json (user-registered catalogs only).
// Returns an empty slice when the file is missing or unparseable.
func loadPersistedCatalogs(pluginsDir string) ([]Catalog, error) {
	data, err := os.ReadFile(IndexesPath(pluginsDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read catalog registry: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse catalog registry: %w", err)
	}
	return reg.Catalogs, nil
}

// LoadRegistry returns the full catalog registry: the synthesized default
// catalog first, then managed pre-seeded catalogs, then the user's own
// registered catalogs. Names are de-duplicated with default and managed entries
// taking precedence over persisted ones, so managed configuration cannot be
// shadowed by a user entry of the same name.
func LoadRegistry(pluginsDir string) (*Registry, error) {
	persisted, err := loadPersistedCatalogs(pluginsDir)
	if err != nil {
		return nil, err
	}

	managed, err := LoadManagedConfig()
	if err != nil {
		return nil, err
	}

	reg := &Registry{}
	seen := map[string]bool{}

	add := func(c Catalog) {
		if seen[c.Name] {
			return
		}
		seen[c.Name] = true
		reg.Catalogs = append(reg.Catalogs, c)
	}

	add(OfficialCatalog())
	for _, c := range managed.SeededCatalogs() {
		add(c)
	}
	for _, c := range persisted {
		if CanonicalCatalogName(c.Name) == OfficialCatalogName {
			// Ignore any stale persisted official catalog (under either "datum" or
			// the legacy "default" name); it is always synthesized.
			continue
		}
		add(c)
	}

	return reg, nil
}

// SaveRegistry persists only the user-registered catalogs (default and managed
// entries are never written to disk).
func SaveRegistry(pluginsDir string, reg *Registry) error {
	out := &Registry{Catalogs: reg.Custom()}
	if err := os.MkdirAll(pluginsDir, 0o700); err != nil {
		return fmt.Errorf("create plugins directory: %w", err)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal catalog registry: %w", err)
	}
	data = append(data, '\n')
	path := IndexesPath(pluginsDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write catalog registry tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace catalog registry: %w", err)
	}
	return nil
}

// ResolvedSource is the outcome of interpreting a catalog source string.
type ResolvedSource struct {
	// IsLocal is true for filesystem sources; LocalPath is then the resolved
	// path to the manifest file.
	IsLocal   bool
	LocalPath string

	// FetchURL is the primary HTTPS manifest URL for remote sources.
	FetchURL string

	// GitHubOwnerRepo is set ("owner/repo") for GitHub shorthand sources, which
	// enables a branch fallback (main -> master) during refresh.
	GitHubOwnerRepo string
}

// ResolveCatalogSource interprets a catalog source string, which may be an
// HTTPS manifest URL, a GitHub "owner/repo" (optionally "github.com/owner/repo")
// shorthand, or a local filesystem path. HTTPS is required for all remote
// sources; only local paths are exempt.
func ResolveCatalogSource(source string) (ResolvedSource, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return ResolvedSource{}, fmt.Errorf("catalog source is required")
	}

	switch {
	case strings.HasPrefix(source, "https://"):
		return ResolvedSource{FetchURL: source}, nil
	case strings.HasPrefix(source, "http://"):
		return ResolvedSource{}, fmt.Errorf("insecure catalog source %q: only HTTPS catalog URLs are supported", source)
	}

	// Explicit local-path forms.
	if isLocalPathSource(source) {
		return resolveLocalSource(source)
	}

	// GitHub shorthand: github.com/owner/repo or owner/repo.
	ghSource := strings.TrimPrefix(source, "github.com/")
	if reOwnerRepo.MatchString(ghSource) {
		return ResolvedSource{
			FetchURL:        gitHubRawURL(ghSource, "main"),
			GitHubOwnerRepo: ghSource,
		}, nil
	}

	// Last resort: treat as a local path if it exists on disk.
	if _, err := os.Stat(source); err == nil {
		return resolveLocalSource(source)
	}

	return ResolvedSource{}, fmt.Errorf("unrecognized catalog source %q: expected an HTTPS URL, a GitHub owner/repo, or a local path", source)
}

// isLocalPathSource reports whether source is written in an explicit local-path
// form (absolute, ./, ../, ~, or a bare ".").
func isLocalPathSource(source string) bool {
	switch {
	case source == ".", source == "..":
		return true
	case strings.HasPrefix(source, "/"),
		strings.HasPrefix(source, "./"),
		strings.HasPrefix(source, "../"),
		strings.HasPrefix(source, "~/"),
		strings.HasPrefix(source, `.\`),
		strings.HasPrefix(source, `..\`):
		return true
	case filepath.IsAbs(source):
		return true
	}
	return false
}

// resolveLocalSource expands a local path and, when it points at a directory,
// appends index.yaml.
func resolveLocalSource(source string) (ResolvedSource, error) {
	path := source
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return ResolvedSource{}, fmt.Errorf("resolve local catalog path %q: %w", source, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return ResolvedSource{}, fmt.Errorf("local catalog path %q is not accessible: %w", source, err)
	}
	if info.IsDir() {
		abs = filepath.Join(abs, "index.yaml")
	}
	return ResolvedSource{IsLocal: true, LocalPath: abs}, nil
}

// gitHubRawURL builds the raw.githubusercontent.com manifest URL for an
// owner/repo at a branch.
func gitHubRawURL(ownerRepo, branch string) string {
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/index.yaml", ownerRepo, branch)
}

// SourceHost returns the hostname of a catalog source for allow-list matching,
// or "" for local sources.
func SourceHost(source string) string {
	resolved, err := ResolveCatalogSource(source)
	if err != nil || resolved.IsLocal {
		return ""
	}
	u, err := url.Parse(resolved.FetchURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
