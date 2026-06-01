package pluginstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	pluginsDirEnvVar = "DATUMCTL_PLUGINS_DIR"
	manifestFileName = "plugins.json"
)

// PluginsDir returns the resolved managed plugins directory.
// Respects (in order): explicit override arg, DATUMCTL_PLUGINS_DIR env var, default.
// Default is ~/.datumctl/plugins/, consistent with ~/.datumctl/config and credentials.
func PluginsDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if env := os.Getenv(pluginsDirEnvVar); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".datumctl", "plugins"), nil
}

// ManifestPath returns <pluginsDir>/plugins.json.
func ManifestPath(pluginsDir string) string {
	return filepath.Join(pluginsDir, manifestFileName)
}

// Load reads plugins.json from pluginsDir. Returns an empty Manifest if not found.
func Load(pluginsDir string) (*Manifest, error) {
	path := ManifestPath(pluginsDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Manifest{}, nil
		}
		return nil, fmt.Errorf("read plugins manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse plugins manifest: %w", err)
	}
	return &m, nil
}

// Save writes plugins.json atomically (write to tmp, rename).
// It creates the pluginsDir if it does not exist.
func Save(pluginsDir string, m *Manifest) error {
	if err := os.MkdirAll(pluginsDir, 0o700); err != nil {
		return fmt.Errorf("create plugins directory: %w", err)
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal plugins manifest: %w", err)
	}
	data = append(data, '\n')

	path := ManifestPath(pluginsDir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write plugins manifest tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace plugins manifest: %w", err)
	}
	return nil
}
