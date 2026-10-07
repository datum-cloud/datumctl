package onboarding

import (
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
)

func TestResolveOrgID(t *testing.T) {
	ctx := &datumconfig.DiscoveredContext{
		OrganizationID: "org-ctx",
		ProjectID:      "proj-ctx",
	}

	tests := []struct {
		name   string
		project, org string
		ctx    *datumconfig.DiscoveredContext
		want   string
	}{
		{"explicit org", "", "org-explicit", nil, "org-explicit"},
		{"project from context", "proj-ctx", "", ctx, "org-ctx"},
		{"project outside context", "proj-a", "", ctx, ""},
		{"org from context", "", "", ctx, "org-ctx"},
		{"no scope", "", "", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveOrgID(tt.project, tt.org, tt.ctx)
			if got != tt.want {
				t.Fatalf("ResolveOrgID() = %q, want %q", got, tt.want)
			}
		})
	}
}
