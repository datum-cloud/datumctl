package pluginstore

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Plugin is the index record for a single datumctl plugin.
// apiVersion/kind fields follow Kubernetes object conventions but are
// used only for YAML parsing — no API server is involved.
type Plugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PluginSpec `json:"spec"`
}

// PluginSpec holds the plugin's release metadata and per-platform download info.
type PluginSpec struct {
	ShortDescription string     `json:"shortDescription"`
	Description      string     `json:"description,omitempty"`
	Homepage         string     `json:"homepage,omitempty"`
	Version          string     `json:"version"`
	Platforms        []Platform `json:"platforms"`
}

// Platform describes a downloadable archive for one OS/arch combination.
// Selector is matched against {"os": GOOS, "arch": GOARCH}.
type Platform struct {
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	URI      string                `json:"uri"`
	SHA256   string                `json:"sha256"`
	Files    []FileOperation       `json:"files,omitempty"`
}

// FileOperation specifies how to copy a file out of the downloaded archive.
type FileOperation struct {
	From string `json:"from"`
	To   string `json:"to,omitempty"`
}

// CachedIndex is the on-disk cache of a remote plugin catalog (index).
type CachedIndex struct {
	RefreshedAt time.Time `json:"refreshed_at"`
	// Header carries the optional catalog-level identity (name/description/owner/
	// homepage) copied from the source manifest, so listings and the browser can
	// show a friendly catalog identity without re-fetching.
	Header  CatalogHeader `json:"header,omitempty"`
	Plugins []Plugin      `json:"plugins"`
}

// CatalogHeader is the optional catalog-level identity block at the top of a
// catalog manifest. Every field is optional; an empty header is valid and keeps
// older, headerless manifests fully backward compatible.
type CatalogHeader struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
}

// PluginList is the wrapper document type for a catalog manifest (index.yaml).
//
// The catalog-level header fields (name/description/owner/homepage) are
// optional. Manifests authored before the marketplace feature omit them
// entirely and still parse: the only required content is the plugin list.
type PluginList struct {
	metav1.TypeMeta `json:",inline"`

	// Optional catalog identity header, surfaced in `plugin index list` and
	// `plugin browse`. Inlined so authors write these at the document root.
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Homepage    string `json:"homepage,omitempty"`

	Items []Plugin `json:"items"`
}

// HeaderFor returns the CatalogHeader derived from this list's identity fields.
func (l *PluginList) HeaderFor() CatalogHeader {
	return CatalogHeader{
		Name:        l.Name,
		Description: l.Description,
		Owner:       l.Owner,
		Homepage:    l.Homepage,
	}
}
