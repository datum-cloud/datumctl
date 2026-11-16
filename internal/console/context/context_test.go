package tuictx

import (
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
)

func TestFromConfigNamespace(t *testing.T) {
	tests := []struct {
		name string
		ctx  datumconfig.DiscoveredContext
		want string
	}{
		{
			// A namespaced list with no namespace is refused at organization
			// scope for every role, so org-wide lists the org's own namespace.
			name: "org-wide context uses the organization namespace",
			ctx:  datumconfig.DiscoveredContext{Name: "s/acme", OrganizationID: "acme"},
			want: "organization-acme",
		},
		{
			name: "org-wide context keeps a namespace it already has",
			ctx:  datumconfig.DiscoveredContext{Name: "s/acme", OrganizationID: "acme", Namespace: "team"},
			want: "team",
		},
		{
			name: "project context keeps its namespace",
			ctx:  datumconfig.DiscoveredContext{Name: "s/acme/web", OrganizationID: "acme", ProjectID: "web", Namespace: "default"},
			want: "default",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &datumconfig.ConfigV1Beta1{
				Contexts:       []datumconfig.DiscoveredContext{tt.ctx},
				CurrentContext: tt.ctx.Name,
			}
			if got := FromConfig(cfg).Namespace; got != tt.want {
				t.Errorf("Namespace = %q, want %q", got, tt.want)
			}
		})
	}
}
