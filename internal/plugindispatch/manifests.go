package plugindispatch

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.datum.net/datumctl/internal/pluginstore"
)

const manifestProbeTimeout = 3 * time.Second

// LoadMissingManifests scans pluginsDir for managed plugin binaries that have
// no manifest entry in plugins.json, probes each one with --plugin-manifest,
// and writes the results back to plugins.json.
//
// This is called lazily (e.g. during tab completion) so plugins placed manually
// into the managed directory get their descriptions populated on first use without
// requiring a full `datumctl plugin install` run.
func LoadMissingManifests(pluginsDir string) {
	if pluginsDir == "" {
		return
	}

	manifest, err := pluginstore.Load(pluginsDir)
	if err != nil {
		manifest = &pluginstore.Manifest{Plugins: map[string]*pluginstore.InstalledPlugin{}}
	}
	if manifest.Plugins == nil {
		manifest.Plugins = map[string]*pluginstore.InstalledPlugin{}
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return
	}

	updated := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name, ok := strings.CutPrefix(e.Name(), "datumctl-")
		if !ok || name == "" {
			continue
		}

		entry := manifest.Plugins[name]
		if entry != nil && entry.Manifest != nil {
			continue // already have manifest
		}

		binaryPath := filepath.Join(pluginsDir, e.Name())
		m := probeManifest(binaryPath)
		if m == nil {
			continue
		}

		if entry == nil {
			entry = &pluginstore.InstalledPlugin{}
		}
		entry.Manifest = m
		if entry.Version == "" && m.Version != "" {
			entry.Version = m.Version
		}
		manifest.Plugins[name] = entry
		updated = true
	}

	if updated {
		_ = pluginstore.Save(pluginsDir, manifest)
	}
}

func probeManifest(binaryPath string) *pluginstore.PluginManifest {
	ctx, cancel := context.WithTimeout(context.Background(), manifestProbeTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, binaryPath, "--plugin-manifest").Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	var m pluginstore.PluginManifest
	if err := json.Unmarshal(out, &m); err != nil {
		return nil
	}
	return &m
}
