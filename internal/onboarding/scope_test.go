package onboarding

import (
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
)

func TestResolveOrgID(t *testing.T) {
	cfg := &datumconfig.ConfigV1Beta1{
		Cache: datumconfig.ContextCache{
			Projects: []datumconfig.CachedProject{
				{ID: "proj-a", OrgID: "org-a"},
			},
		},
	}
	ctx := &datumconfig.DiscoveredContext{
		OrganizationID: "org-ctx",
		ProjectID:      "proj-ctx",
	}

	tests := []struct {
		name   string
		project, org string
		ctx    *datumconfig.DiscoveredContext
		cfg    *datumconfig.ConfigV1Beta1
		want   string
	}{
		{"explicit org", "", "org-explicit", nil, nil, "org-explicit"},
		{"project from context", "proj-ctx", "", ctx, cfg, "org-ctx"},
		{"project from cache", "proj-a", "", nil, cfg, "org-a"},
		{"org from context", "", "", ctx, cfg, "org-ctx"},
		{"no scope", "", "", nil, cfg, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOrgID(tt.project, tt.org, tt.ctx, tt.cfg)
			if got != tt.want {
				t.Fatalf("ResolveOrgID() = %q, want %q", got, tt.want)
			}
		})
	}
}
