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
		// A bare-name entry does NOT authorize a remote source (the name is
		// user-chosen); remote sources require a host-pattern match.
		{"bare name does not authorize remote source", []string{"acme"}, "acme", "https://acme.example/index.yaml", false},
		{"bare name authorizes local source", []string{"acme"}, "acme", "./local-catalog", true},
		{"bare name mismatch denied for local", []string{"acme"}, "evil", "./local-catalog", false},
		{"name mismatch denied", []string{"acme"}, "evil", "https://evil.example/index.yaml", false},
		{"exact host match", []string{"plugins.acme.example"}, "x", "https://plugins.acme.example/index.yaml", true},
		{"parent domain match", []string{"acme.example"}, "x", "https://plugins.acme.example/index.yaml", true},
		{"wildcard host match", []string{"*.acme.example"}, "x", "https://plugins.acme.example/index.yaml", true},
		{"host mismatch denied", []string{"acme.example"}, "x", "https://evil.example/index.yaml", false},
		{"lookalike host denied", []string{"acme.example"}, "x", "https://evil-acme.example/index.yaml", false},
		{"suffix-injection host denied", []string{"acme.example"}, "x", "https://acme.example.evil.com/index.yaml", false},
		{"local source denied under host-only allow-list", []string{"acme.example"}, "x", "./local", false},

		// GitHub owner/repo scoping.
		{"github owner/* permits matching repo shorthand", []string{"github.com/acme-corp/*"}, "x", "acme-corp/datumctl-plugins", true},
		{"github owner/* denies other owner", []string{"github.com/acme-corp/*"}, "x", "evil/repo", false},
		{"github owner (no slash) permits any repo of owner", []string{"github.com/acme-corp"}, "x", "acme-corp/anything", true},
		{"github owner/repo permits exact repo", []string{"github.com/acme-corp/datumctl-plugins"}, "x", "acme-corp/datumctl-plugins", true},
		{"github owner/repo denies other repo of same owner", []string{"github.com/acme-corp/datumctl-plugins"}, "x", "acme-corp/other", false},
		{"github wildcard permits any repo", []string{"github.com/*"}, "x", "anyone/repo", true},
		{"github scope via github.com/owner/repo form", []string{"github.com/acme-corp/*"}, "x", "github.com/acme-corp/repo", true},
		{"github scope via full raw url", []string{"github.com/acme-corp/*"}, "x", "https://raw.githubusercontent.com/acme-corp/repo/main/index.yaml", true},
		// Tightened back-compat: a plain raw.githubusercontent.com host entry no
		// longer green-lights GitHub repos; scoping must be explicit.
		{"plain raw host entry no longer authorizes github", []string{"raw.githubusercontent.com"}, "x", "acme-corp/repo", false},
		// A GitHub scope must not authorize a non-GitHub remote host.
		{"github scope does not authorize other host", []string{"github.com/acme-corp/*"}, "x", "https://acme.example/index.yaml", false},
		// A GitHub source is not authorized by a non-GitHub host pattern.
		{"github source denied under non-github host pattern", []string{"acme.example"}, "x", "acme-corp/repo", false},
		// Anti-repointing intact: a bare name does not authorize a GitHub source.
		{"bare name does not authorize github source", []string{"acme-corp"}, "acme-corp", "acme-corp/repo", false},
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
		{Name: "default", Source: "https://evil.example/index.yaml"},  // reserved, ignored
		{Name: "official", Source: "https://evil.example/index.yaml"}, // reserved, ignored
		{Name: "Bad/Name", Source: "https://evil.example/index.yaml"}, // invalid name, ignored
		{Name: "", Source: "https://x.example"},                       // skipped
		{Name: "nosrc"},                                               // skipped
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

func TestRegistry_ApplyAllowList(t *testing.T) {
	newReg := func() *Registry {
		return &Registry{Catalogs: []Catalog{
			OfficialCatalog(),
			{Name: "corp", Source: "https://corp.example/index.yaml", Type: CatalogTypeCustom, Managed: true},
			{Name: "acme", Source: "https://plugins.acme.example/index.yaml", Type: CatalogTypeCustom},
			{Name: "rogue", Source: "https://rogue.example/index.yaml", Type: CatalogTypeCustom},
		}}
	}

	t.Run("no allow-list disables nothing", func(t *testing.T) {
		reg := newReg()
		if disabled := reg.ApplyAllowList(&ManagedConfig{}); disabled != nil {
			t.Fatalf("expected no disabled catalogs, got %+v", disabled)
		}
		if len(reg.Active()) != len(reg.Catalogs) {
			t.Fatalf("all catalogs should be active when no allow-list is set")
		}
	})

	t.Run("non-permitted catalog disabled, permitted stays active", func(t *testing.T) {
		reg := newReg()
		disabled := reg.ApplyAllowList(&ManagedConfig{AllowedIndexes: []string{"plugins.acme.example"}})
		if len(disabled) != 1 || disabled[0].Name != "rogue" {
			t.Fatalf("expected only 'rogue' disabled, got %+v", disabled)
		}
		if reg.Find("rogue") == nil || !reg.Find("rogue").Disabled {
			t.Fatal("rogue should be marked disabled")
		}
		if reg.Find("rogue").DisabledReason == "" {
			t.Fatal("disabled catalog should carry a reason")
		}
		// Permitted user catalog, the official catalog, and the managed pre-seed
		// all stay active.
		for _, n := range []string{"acme", OfficialCatalogName, "corp"} {
			if c := reg.Find(n); c == nil || c.Disabled {
				t.Fatalf("%q should be active, got %+v", n, c)
			}
		}
		// Active() excludes the disabled catalog.
		for _, c := range reg.Active() {
			if c.Name == "rogue" {
				t.Fatal("Active() must not include a disabled catalog")
			}
		}
		if got := reg.DisabledCatalogs(); len(got) != 1 || got[0].Name != "rogue" {
			t.Fatalf("DisabledCatalogs() = %+v", got)
		}
	})

	t.Run("official and managed never disabled even if unpermitted", func(t *testing.T) {
		reg := newReg()
		// An allow-list that matches none of the sources still leaves the official
		// catalog and the managed pre-seed active (admin-controlled).
		reg.ApplyAllowList(&ManagedConfig{AllowedIndexes: []string{"nothing.example"}})
		if reg.Find(OfficialCatalogName).Disabled {
			t.Fatal("official catalog must never be disabled")
		}
		if reg.Find("corp").Disabled {
			t.Fatal("managed pre-seed must never be disabled")
		}
		if !reg.Find("acme").Disabled || !reg.Find("rogue").Disabled {
			t.Fatal("both user catalogs should be disabled under this allow-list")
		}
	})
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
	if !mc.IsAllowed("acme", "./local-catalog") {
		t.Fatal("acme should be allowed by name for a local source")
	}
	if mc.IsAllowed("acme", "https://acme.example/i.yaml") {
		t.Fatal("a bare-name entry must not authorize a remote source")
	}
	if !mc.IsAllowed("x", "https://team.corp.example/i.yaml") {
		t.Fatal("corp subdomain should be allowed by host wildcard")
	}
}
