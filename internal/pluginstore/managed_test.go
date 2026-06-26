package pluginstore

import "testing"

func TestManagedConfig_IsAllowed(t *testing.T) {
	cases := []struct {
		name    string
		allow   []string
		catName string
		source  string
		want    bool
	}{
		{"empty allow-list permits all", nil, "acme", "https://anything.example/index.yaml", true},
		{"name match", []string{"acme"}, "acme", "https://acme.example/index.yaml", true},
		{"name mismatch denied", []string{"acme"}, "evil", "https://evil.example/index.yaml", false},
		{"exact host match", []string{"plugins.acme.example"}, "x", "https://plugins.acme.example/index.yaml", true},
		{"parent domain match", []string{"acme.example"}, "x", "https://plugins.acme.example/index.yaml", true},
		{"wildcard host match", []string{"*.acme.example"}, "x", "https://plugins.acme.example/index.yaml", true},
		{"host mismatch denied", []string{"acme.example"}, "x", "https://evil.example/index.yaml", false},
		{"local source denied under host allow-list", []string{"acme.example"}, "x", "./local", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mc := &ManagedConfig{AllowedIndexes: tc.allow}
			if got := mc.IsAllowed(tc.catName, tc.source); got != tc.want {
				t.Fatalf("IsAllowed(%q,%q) = %v, want %v", tc.catName, tc.source, got, tc.want)
			}
		})
	}
}

func TestManagedConfig_SeededCatalogs(t *testing.T) {
	mc := &ManagedConfig{Indexes: []ManagedIndex{
		{Name: "acme", Source: "https://acme.example/index.yaml", Description: "ACME"},
		{Name: "default", Source: "https://evil.example/index.yaml"}, // must be ignored
		{Name: "", Source: "https://x.example"},                      // skipped
		{Name: "nosrc"},                                              // skipped
	}}
	got := mc.SeededCatalogs()
	if len(got) != 1 {
		t.Fatalf("expected 1 seeded catalog, got %d: %+v", len(got), got)
	}
	if got[0].Name != "acme" || !got[0].Managed || got[0].Type != CatalogTypeCustom {
		t.Fatalf("unexpected seeded catalog: %+v", got[0])
	}
	if got[0].Trust() != TrustThirdParty {
		t.Fatalf("managed catalog should be third-party, got %q", got[0].Trust())
	}
}

func TestLoadManagedConfig_envAllowList(t *testing.T) {
	t.Setenv(allowedIndexesEnvVar, "acme, *.corp.example ")
	mc, err := LoadManagedConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !mc.Enforced() || len(mc.AllowedIndexes) != 2 {
		t.Fatalf("unexpected allow-list: %+v", mc.AllowedIndexes)
	}
	if !mc.IsAllowed("acme", "https://acme.example/i.yaml") {
		t.Fatal("acme should be allowed by name")
	}
	if !mc.IsAllowed("x", "https://team.corp.example/i.yaml") {
		t.Fatal("corp subdomain should be allowed")
	}
}
