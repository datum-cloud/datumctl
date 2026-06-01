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

// CachedIndex is the on-disk cache of the remote plugin index.
type CachedIndex struct {
	RefreshedAt time.Time `json:"refreshed_at"`
	Plugins     []Plugin  `json:"plugins"`
}

// PluginList is the wrapper document type for the remote index.yaml.
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	Items           []Plugin `json:"items"`
}
