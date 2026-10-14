package client

import (
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
)

func TestResolvedScopeDetails(t *testing.T) {
	tests := []struct {
		name         string
		override     string
		project      string
		org          string
		platformWide bool
		envProject   string
		envOrg       string
		want         Scope
		wantErr      bool
	}{
		{
			name: "organization context",
			want: Scope{Organization: "org-prod", Source: ScopeSourceContext},
		},
		{
			name:     "project context carries its organization",
			override: ovStaging,
			want:     Scope{Project: "proj-staging", ProjectOrganization: "org-staging", Source: ScopeSourceContext},
		},
		{
			name:     "session without a context has no scope",
			override: ovSolo,
			want:     Scope{Source: ScopeSourceNone},
		},
		{
			name:    "project flag beats the context and knows no organization",
			project: "other",
			want:    Scope{Project: "other", Source: ScopeSourceFlag},
		},
		{
			name: "organization flag",
			org:  "other-org",
			want: Scope{Organization: "other-org", Source: ScopeSourceFlag},
		},
		{
			name:         "platform-wide flag",
			platformWide: true,
			want:         Scope{PlatformWide: true, Source: ScopeSourceFlag},
		},
		{
			name:       "environment beats the context",
			envProject: "env-project",
			want:       Scope{Project: "env-project", Source: ScopeSourceEnv},
		},
		{
			name:       "flag beats the environment",
			project:    "flag-project",
			envProject: "env-project",
			want:       Scope{Project: "flag-project", Source: ScopeSourceFlag},
		},
		{
			name:       "project and organization from the environment is an error",
			envProject: "p",
			envOrg:     "o",
			wantErr:    true,
		},
		{
			name:         "platform-wide with a project is an error",
			platformWide: true,
			project:      "p",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setupFactoryOverrideEnv(t)
			t.Setenv("DATUM_PROJECT", tt.envProject)
			t.Setenv("DATUM_ORGANIZATION", tt.envOrg)
			*f.ConfigFlags.Project = tt.project
			*f.ConfigFlags.Organization = tt.org
			*f.ConfigFlags.PlatformWide = tt.platformWide
			if tt.override != "" {
				datumconfig.SetSessionOverride(tt.override, datumconfig.SessionOverrideFromFlag)
			}

			got, err := f.ConfigFlags.ResolvedScopeDetails()
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("scope = %+v, want %+v", got, tt.want)
			}
		})
	}
}
