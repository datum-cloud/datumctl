package client

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/miloapi"
)

const (
	ovProd    = "swells@datum.net@api.datum.net"
	ovStaging = "swells@datum.net@api.staging.env.datum.net"
	ovSolo    = "solo@example.com@api.datum.net"
)

func setupFactoryOverrideEnv(t *testing.T) *DatumCloudFactory {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DATUM_PROJECT", "")
	t.Setenv("DATUM_ORGANIZATION", "")
	keyring.MockInit()
	t.Cleanup(datumconfig.ClearSessionOverride)

	for key, subject := range map[string]string{"key-prod": "u-prod", "key-staging": "u-staging", "key-solo": "u-solo"} {
		blob, _ := json.Marshal(authutil.StoredCredentials{
			Hostname: "auth.datum.net",
			Subject:  subject,
			Token:    &oauth2.Token{AccessToken: "tok", Expiry: time.Now().Add(time.Hour)},
		})
		if err := keyring.Set(authutil.ServiceName, key, string(blob)); err != nil {
			t.Fatal(err)
		}
	}

	prodCtx := datumconfig.QualifiedContextName(ovProd, "org-prod")
	stagingCtx := datumconfig.QualifiedContextName(ovStaging, "org-staging/proj-staging")
	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{Name: ovProd, UserKey: "key-prod", UserEmail: "swells@datum.net",
			Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}, LastContext: prodCtx},
		{Name: ovStaging, UserKey: "key-staging", UserEmail: "swells@datum.net",
			Endpoint: datumconfig.Endpoint{Server: "https://api.staging.env.datum.net", TLSServerName: "staging.sni"},
			LastContext: stagingCtx},
		{Name: ovSolo, UserKey: "key-solo", UserEmail: "solo@example.com",
			Endpoint: datumconfig.Endpoint{Server: "https://api.solo.example", TLSServerName: "solo.sni"}},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{
		{Name: prodCtx, Session: ovProd, OrganizationID: "org-prod"},
		{Name: stagingCtx, Session: ovStaging, OrganizationID: "org-staging", ProjectID: "proj-staging"},
	}
	cfg.ActiveSession = ovProd
	cfg.CurrentContext = prodCtx
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatal(err)
	}

	f, err := NewDatumFactory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f.ConfigFlags.SkipOnboardingCheck = true
	return f
}

// Requests go to the overriding session's endpoint with its credentials and
// its last context, while --project and DATUM_ORGANIZATION still pick scope.
func TestToRESTConfig_SessionOverride(t *testing.T) {
	const staging = "https://api.staging.env.datum.net"
	tests := []struct {
		name       string
		override   string
		project    string
		envOrg     string
		wantHost   string
		wantServer string
	}{
		{name: "no override", wantHost: miloapi.OrgControlPlaneURL("https://api.datum.net", "org-prod")},
		{name: "override uses its last context", override: ovStaging,
			wantHost: miloapi.ProjectControlPlaneURL(staging, "proj-staging"), wantServer: "staging.sni"},
		{name: "--project beats the override's context", override: ovStaging, project: "other",
			wantHost: miloapi.ProjectControlPlaneURL(staging, "other"), wantServer: "staging.sni"},
		{name: "DATUM_ORGANIZATION beats the override's context", override: ovStaging, envOrg: "env-org",
			wantHost: miloapi.OrgControlPlaneURL(staging, "env-org"), wantServer: "staging.sni"},
		{name: "override without a context uses its user control plane on its endpoint", override: ovSolo,
			wantHost: miloapi.UserControlPlaneURL("https://api.solo.example", "u-solo"), wantServer: "solo.sni"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := setupFactoryOverrideEnv(t)
			t.Setenv("DATUM_ORGANIZATION", tt.envOrg)
			*f.ConfigFlags.Project = tt.project
			if tt.override != "" {
				datumconfig.SetSessionOverride(tt.override, datumconfig.SessionOverrideFromFlag)
			}
			rc, err := f.ConfigFlags.ToRESTConfig()
			if err != nil {
				t.Fatalf("ToRESTConfig: %v", err)
			}
			if rc.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", rc.Host, tt.wantHost)
			}
			if rc.ServerName != tt.wantServer {
				t.Errorf("ServerName = %q, want %q", rc.ServerName, tt.wantServer)
			}
		})
	}
}

// A DATUM_SESSION that names no session must fail ToRESTConfig with the same
// clear, list-of-choices error a bad --session gives — not silently build a
// REST config for the real active session.
func TestToRESTConfig_StaleEnvSessionOverride(t *testing.T) {
	f := setupFactoryOverrideEnv(t)
	datumconfig.SetPendingSessionOverride("nobody@example.com")

	_, err := f.ConfigFlags.ToRESTConfig()
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := customerrors.IsUserError(err); !ok {
		t.Errorf("error is not a UserError: %v", err)
	}
	for _, want := range []string{"No session matches DATUM_SESSION nobody@example.com.", ovProd, ovStaging, ovSolo} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q:\n%s", want, err)
		}
	}
}

func TestResolveSessionEndpoint_SessionOverride(t *testing.T) {
	setupFactoryOverrideEnv(t)
	datumconfig.SetSessionOverride(ovStaging, datumconfig.SessionOverrideFromEnv)
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		t.Fatal(err)
	}
	session, ep, err := ResolveSessionEndpoint(cfg, "")
	if err != nil {
		t.Fatal(err)
	}
	if session.Name != ovStaging || ep.BaseServer != "https://api.staging.env.datum.net" || ep.UserKey != "key-staging" {
		t.Errorf("got session %q, server %q, key %q; want the staging session", session.Name, ep.BaseServer, ep.UserKey)
	}
}
