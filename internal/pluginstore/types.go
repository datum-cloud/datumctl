package pluginstore

import "time"

// Manifest is the in-memory representation of plugins.json.
type Manifest struct {
	Plugins map[string]*InstalledPlugin `json:"plugins,omitempty"`
	Trusted map[string]*TrustedEntry    `json:"trusted,omitempty"`
}

// InstalledPlugin is one entry in the managed install record.
type InstalledPlugin struct {
	Source string `json:"source"`
	// Catalog is the name of the catalog (index) the plugin was installed from,
	// e.g. "default" or "acme". Empty for direct owner/repo installs and for
	// records written before the marketplace feature (treated as unknown).
	Catalog     string          `json:"catalog,omitempty"`
	Version     string          `json:"version"`
	SHA256      string          `json:"sha256"`
	InstalledAt time.Time       `json:"installed_at"`
	Manifest    *PluginManifest `json:"manifest"`
}

// PluginManifest is the JSON produced by a plugin binary's --plugin-manifest flag.
type PluginManifest struct {
	Name               string `json:"name"`
	Version            string `json:"version"`
	Description        string `json:"description"`
	MinDatumctlVersion string `json:"min_datumctl_version,omitempty"`
	APIVersion         int    `json:"api_version"`
	MinAPIVersion      int    `json:"min_api_version,omitempty"`
}

// TrustedEntry records a trusted PATH-plugin binary path.
type TrustedEntry struct {
	Path      string    `json:"path"`
	SHA256    string    `json:"sha256"`
	TrustedAt time.Time `json:"trusted_at"`
}
