package pluginstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad_missing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load from nonexistent path: want nil error, got %v", err)
	}
	if m == nil {
		t.Fatal("Load from nonexistent path: want empty Manifest{}, got nil")
	}
	if len(m.Plugins) != 0 || len(m.Trusted) != 0 {
		t.Errorf("Load from nonexistent path: want empty Manifest, got %+v", m)
	}
}

func TestLoad_malformed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := ManifestPath(dir)
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("write malformed file: %v", err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load from malformed JSON: want error, got nil")
	}
}

func TestSave_roundtrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	original := &Manifest{
		Plugins: map[string]*InstalledPlugin{
			"dns": {
				Source:      "github.com/datum-cloud/datumctl-dns",
				Version:     "v1.2.3",
				SHA256:      "abc123",
				InstalledAt: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Manifest: &PluginManifest{
					Name:        "datumctl-dns",
					Version:     "v1.2.3",
					Description: "DNS plugin for datumctl",
					APIVersion:  1,
				},
			},
		},
		Trusted: map[string]*TrustedEntry{
			"mytools": {
				Path:      "/usr/local/bin/datumctl-mytools",
				TrustedAt: time.Date(2024, 2, 1, 8, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := Save(dir, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}

	if loaded.Plugins == nil {
		t.Fatal("roundtrip: Plugins map is nil")
	}
	dns, ok := loaded.Plugins["dns"]
	if !ok {
		t.Fatal("roundtrip: 'dns' plugin entry missing")
	}
	if dns.Source != original.Plugins["dns"].Source {
		t.Errorf("roundtrip Source: got %q, want %q", dns.Source, original.Plugins["dns"].Source)
	}
	if dns.Version != original.Plugins["dns"].Version {
		t.Errorf("roundtrip Version: got %q, want %q", dns.Version, original.Plugins["dns"].Version)
	}
	if dns.SHA256 != original.Plugins["dns"].SHA256 {
		t.Errorf("roundtrip SHA256: got %q, want %q", dns.SHA256, original.Plugins["dns"].SHA256)
	}
	if dns.Manifest == nil {
		t.Fatal("roundtrip: Manifest is nil")
	}
	if dns.Manifest.Description != original.Plugins["dns"].Manifest.Description {
		t.Errorf("roundtrip Manifest.Description: got %q, want %q",
			dns.Manifest.Description, original.Plugins["dns"].Manifest.Description)
	}

	if loaded.Trusted == nil {
		t.Fatal("roundtrip: Trusted map is nil")
	}
	trusted, ok := loaded.Trusted["mytools"]
	if !ok {
		t.Fatal("roundtrip: 'mytools' trusted entry missing")
	}
	if trusted.Path != original.Trusted["mytools"].Path {
		t.Errorf("roundtrip Trusted.Path: got %q, want %q", trusted.Path, original.Trusted["mytools"].Path)
	}
}

func TestSave_atomic(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := ManifestPath(dir)
	tmpPath := manifestPath + ".tmp"

	m := &Manifest{
		Plugins: map[string]*InstalledPlugin{
			"dns": {
				Source:  "github.com/datum-cloud/datumctl-dns",
				Version: "v1.0.0",
				SHA256:  "deadbeef",
			},
		},
	}

	if err := Save(dir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// After a successful save, the .tmp file must not exist (it was renamed away).
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("atomic save: .tmp file still exists after successful Save — want it removed after rename")
	}

	// The final file must exist and be valid JSON.
	_, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after atomic save: %v", err)
	}
}

func TestPluginsDir_default(t *testing.T) {
	// Do not run in parallel — manipulates env.
	t.Setenv(pluginsDirEnvVar, "")

	dir, err := PluginsDir("")
	if err != nil {
		t.Fatalf("PluginsDir default: %v", err)
	}

	want := filepath.Join(".datumctl", "plugins")
	if !strings.HasSuffix(dir, want) {
		t.Errorf("PluginsDir default: got %q, want suffix %q", dir, want)
	}
}

func TestPluginsDir_envVar(t *testing.T) {
	// Do not run in parallel — manipulates env.
	customDir := t.TempDir()
	t.Setenv(pluginsDirEnvVar, customDir)

	dir, err := PluginsDir("")
	if err != nil {
		t.Fatalf("PluginsDir env var: %v", err)
	}

	if dir != customDir {
		t.Errorf("PluginsDir env var: got %q, want %q", dir, customDir)
	}
}

func TestPluginsDir_flagOverride(t *testing.T) {
	// Do not run in parallel — manipulates env.
	envDir := t.TempDir()
	t.Setenv(pluginsDirEnvVar, envDir)

	flagDir := t.TempDir()

	dir, err := PluginsDir(flagDir)
	if err != nil {
		t.Fatalf("PluginsDir flag override: %v", err)
	}

	if dir != flagDir {
		t.Errorf("PluginsDir flag override: got %q, want %q (env was %q)", dir, flagDir, envDir)
	}
}
