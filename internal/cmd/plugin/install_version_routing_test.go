package plugin

import (
	"testing"

	"go.datum.net/datumctl/internal/pluginstore"
)

// TestRouteInstallArg pins the install-argument routing, in particular that a
// trailing "@version" is split off BEFORE routing so that catalog-qualified and
// bare-name installs version-pin correctly instead of being misrouted to the
// GitHub path.
func TestRouteInstallArg(t *testing.T) {
	// "acme" is a registered catalog; "owner" is not.
	reg := &pluginstore.Registry{Catalogs: []pluginstore.Catalog{
		{Name: pluginstore.OfficialCatalogName},
		{Name: "acme"},
	}}

	cases := []struct {
		name string
		arg  string
		want installRoute
	}{
		{
			name: "catalog-qualified with version routes to the catalog (not GitHub)",
			arg:  "acme/deploy@v2",
			want: installRoute{kind: routeCatalog, catalog: "acme", name: "deploy", version: "v2"},
		},
		{
			name: "catalog-qualified without version still works",
			arg:  "acme/deploy",
			want: installRoute{kind: routeCatalog, catalog: "acme", name: "deploy"},
		},
		{
			name: "bare name with version resolves across catalogs at that version",
			arg:  "dns@v1.2.3",
			want: installRoute{kind: routeBare, name: "dns", version: "v1.2.3"},
		},
		{
			name: "bare name without version",
			arg:  "dns",
			want: installRoute{kind: routeBare, name: "dns"},
		},
		{
			name: "owner/repo with version still routes to GitHub",
			arg:  "owner/repo@v2",
			want: installRoute{kind: routeGitHub, ghSource: "owner/repo", version: "v2"},
		},
		{
			name: "owner/repo without version routes to GitHub",
			arg:  "owner/repo",
			want: installRoute{kind: routeGitHub, ghSource: "owner/repo"},
		},
		{
			name: "github.com/owner/repo prefix with version routes to GitHub",
			arg:  "github.com/owner/repo@v2",
			want: installRoute{kind: routeGitHub, ghSource: "github.com/owner/repo", version: "v2"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := routeInstallArg(reg, tc.arg)
			if got != tc.want {
				t.Fatalf("routeInstallArg(%q) = %+v, want %+v", tc.arg, got, tc.want)
			}
		})
	}
}
