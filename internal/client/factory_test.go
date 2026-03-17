package client

import (
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestResolveScope_FlagsOverrideContext(t *testing.T) {
	t.Parallel()

	ctxEntry := &datumconfig.NamedContext{
		Name: "ctx",
		Context: datumconfig.Context{
			ProjectID: "ctx-project",
		},
	}

	tests := []struct {
		name           string
		projectFlag    *string
		orgFlag        *string
		platformWide   *bool
		wantProjectID  string
		wantOrgID      string
		wantPlatform   bool
	}{
		{
			name:          "project flag beats context project",
			projectFlag:   stringPtr("flag-project"),
			orgFlag:       stringPtr(""),
			platformWide:  boolPtr(false),
			wantProjectID: "flag-project",
		},
		{
			name:          "organization flag beats context project",
			projectFlag:   stringPtr(""),
			orgFlag:       stringPtr("flag-org"),
			platformWide:  boolPtr(false),
			wantOrgID:     "flag-org",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &CustomConfigFlags{
				Project:      tt.projectFlag,
				Organization: tt.orgFlag,
				PlatformWide: tt.platformWide,
			}

			projectID, organizationID, platformWide, err := c.resolveScope(ctxEntry)
			if err != nil {
				t.Fatalf("resolveScope returned error: %v", err)
			}
			if projectID != tt.wantProjectID {
				t.Fatalf("projectID=%q, want %q", projectID, tt.wantProjectID)
			}
			if organizationID != tt.wantOrgID {
				t.Fatalf("organizationID=%q, want %q", organizationID, tt.wantOrgID)
			}
			if platformWide != tt.wantPlatform {
				t.Fatalf("platformWide=%v, want %v", platformWide, tt.wantPlatform)
			}
		})
	}
}

func TestResolveBaseServer_PrefersFlagOverCluster(t *testing.T) {
	t.Parallel()

	clusterEntry := &datumconfig.NamedCluster{
		Name: "cluster-1",
		Cluster: datumconfig.Cluster{
			Server: "https://cluster.example.com/",
		},
	}

	c := &CustomConfigFlags{
		ConfigFlags: &genericclioptions.ConfigFlags{
			APIServer: stringPtr("https://flag.example.com/"),
		},
	}

	baseServer, err := c.resolveBaseServer("user-key", clusterEntry)
	if err != nil {
		t.Fatalf("resolveBaseServer returned error: %v", err)
	}
	if baseServer != "https://flag.example.com" {
		t.Fatalf("baseServer=%q, want %q", baseServer, "https://flag.example.com")
	}

	c.APIServer = stringPtr("")
	baseServer, err = c.resolveBaseServer("user-key", clusterEntry)
	if err != nil {
		t.Fatalf("resolveBaseServer returned error with cluster fallback: %v", err)
	}
	if baseServer != "https://cluster.example.com" {
		t.Fatalf("baseServer=%q, want %q", baseServer, "https://cluster.example.com")
	}
}

func stringPtr(val string) *string {
	return &val
}

func boolPtr(val bool) *bool {
	return &val
}
