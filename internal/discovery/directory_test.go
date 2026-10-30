package discovery

import "testing"

func resolveRef(d *Directory, query string) string {
	if got := d.Resolve(query); got != nil {
		return got.Ref()
	}
	return ""
}

func TestResolve(t *testing.T) {
	t.Parallel()

	d := &Directory{
		Orgs: []DiscoveredOrg{{Name: "datum"}, {Name: "staging"}, {Name: "acme"}},
		Projects: []DiscoveredProject{
			{Name: "datum-cloud", OrgName: "datum"},
			{Name: "other-proj", OrgName: "datum"},
			{Name: "my-app", OrgName: "staging"},
			{Name: "web-app", OrgName: "acme"},
		},
	}

	tests := []struct {
		query   string
		wantRef string
	}{
		{query: "datum", wantRef: "datum"},
		{query: "datum/datum-cloud", wantRef: "datum/datum-cloud"},
		{query: "acme/web-app", wantRef: "acme/web-app"},
		{query: "staging", wantRef: "staging"},
		{query: "my-app", wantRef: "staging/my-app"},
		{query: "datum-cloud", wantRef: "datum/datum-cloud"},
		{query: "nonexistent", wantRef: ""},
		{query: "foo/bar", wantRef: ""},
	}
	for _, tt := range tests {
		if got := resolveRef(d, tt.query); got != tt.wantRef {
			t.Errorf("Resolve(%q) = %q, want %q", tt.query, got, tt.wantRef)
		}
	}
}

func TestResolve_AmbiguousProjectID(t *testing.T) {
	t.Parallel()

	d := &Directory{Projects: []DiscoveredProject{
		{Name: "shared", OrgName: "org-a"},
		{Name: "shared", OrgName: "org-b"},
	}}

	if got := resolveRef(d, "shared"); got != "" {
		t.Errorf("ambiguous project ID resolved to %q", got)
	}
	if got := resolveRef(d, "org-a/shared"); got != "org-a/shared" {
		t.Errorf("Resolve(org-a/shared) = %q", got)
	}
}

func TestResolve_AmbiguousDisplayName(t *testing.T) {
	t.Parallel()

	d := &Directory{Orgs: []DiscoveredOrg{
		{Name: "org-a", DisplayName: "Production"},
		{Name: "org-b", DisplayName: "Production"},
	}}

	if got := resolveRef(d, "Production"); got != "" {
		t.Errorf("ambiguous org display name resolved to %q", got)
	}
}

func TestResolve_IDWinsOverDisplayName(t *testing.T) {
	t.Parallel()

	d := &Directory{Projects: []DiscoveredProject{
		{Name: "proj-a", DisplayName: "something-else", OrgName: "org-1"},
		{Name: "proj-b", DisplayName: "proj-a", OrgName: "org-1"},
	}}

	if got := resolveRef(d, "proj-a"); got != "org-1/proj-a" {
		t.Errorf("Resolve(proj-a) = %q, want org-1/proj-a", got)
	}
}

func TestResolve_ProjectDisplayNameScopedToOrg(t *testing.T) {
	t.Parallel()

	d := &Directory{Projects: []DiscoveredProject{
		{Name: "proj-a", DisplayName: "shared", OrgName: "org-a"},
		{Name: "proj-b", DisplayName: "shared", OrgName: "org-b"},
	}}

	if got := resolveRef(d, "org-a/shared"); got != "org-a/proj-a" {
		t.Errorf("Resolve(org-a/shared) = %q", got)
	}
	if got := resolveRef(d, "org-b/shared"); got != "org-b/proj-b" {
		t.Errorf("Resolve(org-b/shared) = %q", got)
	}
}

func TestDescribe_FallsBackToIDs(t *testing.T) {
	t.Parallel()

	d := &Directory{Orgs: []DiscoveredOrg{{Name: "org-1", DisplayName: "Acme Corp"}}}
	ctx := d.projectContext("org-1", "proj-1")
	if got := d.Describe(ctx); got != "project proj-1 in Acme Corp (org-1/proj-1)" {
		t.Errorf("Describe = %q", got)
	}
	var none *Directory
	if got := none.Describe(ctx); got != "project proj-1 in org-1 (org-1/proj-1)" {
		t.Errorf("nil Describe = %q", got)
	}
}
