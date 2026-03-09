package datumconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}
	if cfg.APIVersion == "" || cfg.Kind == "" {
		t.Fatalf("expected defaults, got apiVersion=%q kind=%q", cfg.APIVersion, cfg.Kind)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	cfg := New()
	cfg.UpsertCluster(NamedCluster{
		Name: "primary",
		Cluster: Cluster{
			Server: "https://api.example.com",
		},
	})
	cfg.UpsertContext(NamedContext{
		Name: "ctx",
		Context: Context{
			Cluster:        "primary",
			Namespace:      "default",
			ProjectID:      "proj",
			OrganizationID: "",
		},
	})
	cfg.CurrentContext = "ctx"

	if err := SaveToPath(cfg, path); err != nil {
		t.Fatalf("SaveToPath: %v", err)
	}

	loaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}
	if loaded.CurrentContext != "ctx" {
		t.Fatalf("expected current-context ctx, got %q", loaded.CurrentContext)
	}
	if _, ok := loaded.ClusterByName("primary"); !ok {
		t.Fatalf("expected cluster to be loaded")
	}
	if _, ok := loaded.ContextByName("ctx"); !ok {
		t.Fatalf("expected context to be loaded")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}
