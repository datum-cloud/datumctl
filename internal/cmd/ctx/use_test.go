package ctx

import (
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// A context owned by another session must point at a switch command that
// works, including --endpoint when that session's email spans endpoints.
func TestUseHintForContextInOtherSession(t *testing.T) {
	const email = "swells@datum.net"
	prod := datumconfig.SessionName(email, "api.datum.net")
	staging := datumconfig.SessionName(email, "api.staging.env.datum.net")
	solo := datumconfig.SessionName("solo@example.com", "api.datum.net")

	tests := []struct {
		name     string
		active   string
		ref      string
		wantHint string
	}{
		{
			name:     "shared email includes endpoint",
			active:   prod,
			ref:      "org-staging",
			wantHint: "Run 'datumctl auth switch " + email + " --endpoint api.staging.env.datum.net' first, then 'datumctl ctx use org-staging'.",
		},
		{
			name:     "unique email stays bare",
			active:   prod,
			ref:      "org-solo",
			wantHint: "Run 'datumctl auth switch solo@example.com' first, then 'datumctl ctx use org-solo'.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)

			cfg := datumconfig.NewV1Beta1()
			cfg.Sessions = []datumconfig.Session{
				{Name: prod, UserEmail: email, Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}},
				{Name: staging, UserEmail: email, Endpoint: datumconfig.Endpoint{Server: "https://api.staging.env.datum.net"}},
				{Name: solo, UserEmail: "solo@example.com", Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}},
			}
			cfg.Contexts = []datumconfig.DiscoveredContext{
				{Name: datumconfig.QualifiedContextName(prod, "org-prod"), Session: prod, OrganizationID: "org-prod"},
				{Name: datumconfig.QualifiedContextName(staging, "org-staging"), Session: staging, OrganizationID: "org-staging"},
				{Name: datumconfig.QualifiedContextName(solo, "org-solo"), Session: solo, OrganizationID: "org-solo"},
			}
			cfg.ActiveSession = tc.active
			if err := datumconfig.SaveV1Beta1(cfg); err != nil {
				t.Fatalf("save config: %v", err)
			}

			err := runUse(nil, []string{tc.ref})
			userErr, ok := customerrors.IsUserError(err)
			if !ok {
				t.Fatalf("expected *UserError, got %T: %v", err, err)
			}
			if !strings.Contains(userErr.Hint, tc.wantHint) {
				t.Errorf("Hint = %q, want %q", userErr.Hint, tc.wantHint)
			}
		})
	}
}
