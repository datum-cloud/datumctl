package api

import (
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// fixtureConfig returns a config with an active session AND an active
// project-scoped context. Every session carries a UserKey and an endpoint
// Server so resolution never has to touch the keyring.
func fixtureConfig() *datumconfig.ConfigV1Beta1 {
	return &datumconfig.ConfigV1Beta1{
		APIVersion: datumconfig.V1Beta1APIVersion,
		Kind:       datumconfig.DefaultKind,
		Sessions: []datumconfig.Session{
			{
				Name:      "maya@datum.net@api.datum.net",
				UserKey:   "sub-maya@auth.datum.net",
				UserEmail: "maya@datum.net",
				Endpoint: datumconfig.Endpoint{
					Server:       "https://api.datum.net",
					AuthHostname: "auth.datum.net",
				},
			},
			{
				Name:      "sam@datum.net@api.staging.env.datum.net",
				UserKey:   "sub-sam@auth.staging.env.datum.net",
				UserEmail: "sam@datum.net",
				Endpoint: datumconfig.Endpoint{
					Server:       "https://api.staging.env.datum.net",
					AuthHostname: "auth.staging.env.datum.net",
				},
			},
		},
		Contexts: []datumconfig.DiscoveredContext{
			{
				Name:           "ctx-org/ctx-project",
				Session:        "maya@datum.net@api.datum.net",
				OrganizationID: "ctx-org",
				ProjectID:      "ctx-project",
			},
		},
		CurrentContext: "ctx-org/ctx-project",
		ActiveSession:  "maya@datum.net@api.datum.net",
	}
}

// TestResolveProxyTarget_NoScopeFlagsTargetsEndpointRoot pins the
// no-context-inheritance rule: even with an active project context configured
// and DATUM_* scope variables set, a bare proxy targets the endpoint root.
func TestResolveProxyTarget_NoScopeFlagsTargetsEndpointRoot(t *testing.T) {
	t.Setenv("DATUM_PROJECT", "env-project")
	t.Setenv("DATUM_ORGANIZATION", "env-org")

	target, err := resolveProxyTarget(fixtureConfig(), "", "", "", false)
	if err != nil {
		t.Fatalf("resolveProxyTarget: %v", err)
	}

	if got, want := target.upstream.String(), "https://api.datum.net"; got != want {
		t.Errorf("upstream = %q, want the endpoint root %q (no active-context or env inheritance)", got, want)
	}
	if !strings.Contains(target.scope, "full endpoint") {
		t.Errorf("scope = %q, want it to describe the full endpoint", target.scope)
	}
}

func TestResolveProxyTarget_ScopedFlags(t *testing.T) {
	tests := []struct {
		name         string
		project      string
		organization string
		platformWide bool
		wantUpstream string
		wantScope    string
	}{
		{
			name:         "project scope",
			project:      "my-project",
			wantUpstream: "https://api.datum.net/apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane",
			wantScope:    "project my-project",
		},
		{
			name:         "organization scope",
			organization: "my-org",
			wantUpstream: "https://api.datum.net/apis/resourcemanager.miloapis.com/v1alpha1/organizations/my-org/control-plane",
			wantScope:    "organization my-org",
		},
		{
			name:         "platform-wide scope",
			platformWide: true,
			wantUpstream: "https://api.datum.net",
			wantScope:    "platform-wide",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target, err := resolveProxyTarget(fixtureConfig(), "", tc.project, tc.organization, tc.platformWide)
			if err != nil {
				t.Fatalf("resolveProxyTarget: %v", err)
			}
			if got := target.upstream.String(); got != tc.wantUpstream {
				t.Errorf("upstream = %q, want %q", got, tc.wantUpstream)
			}
			if target.scope != tc.wantScope {
				t.Errorf("scope = %q, want %q", target.scope, tc.wantScope)
			}
		})
	}
}

// TestResolveProxyTarget_ScopeUsesPinnedSessionEndpoint proves scoped mode
// builds the control-plane URL on the pinned session's endpoint, not the
// active session's.
func TestResolveProxyTarget_ScopeUsesPinnedSessionEndpoint(t *testing.T) {
	target, err := resolveProxyTarget(fixtureConfig(), "sam@datum.net@api.staging.env.datum.net", "my-project", "", false)
	if err != nil {
		t.Fatalf("resolveProxyTarget: %v", err)
	}
	want := "https://api.staging.env.datum.net/apis/resourcemanager.miloapis.com/v1alpha1/projects/my-project/control-plane"
	if got := target.upstream.String(); got != want {
		t.Errorf("upstream = %q, want %q", got, want)
	}
}

func TestResolveProxyTarget_PinnedSession(t *testing.T) {
	target, err := resolveProxyTarget(fixtureConfig(), "sam@datum.net@api.staging.env.datum.net", "", "", false)
	if err != nil {
		t.Fatalf("resolveProxyTarget: %v", err)
	}
	if got, want := target.upstream.String(), "https://api.staging.env.datum.net"; got != want {
		t.Errorf("upstream = %q, want %q", got, want)
	}
	if target.session == nil || target.session.UserEmail != "sam@datum.net" {
		t.Errorf("session = %+v, want the pinned staging session", target.session)
	}
	if got, want := target.endpoint.UserKey, "sub-sam@auth.staging.env.datum.net"; got != want {
		t.Errorf("user key = %q, want %q", got, want)
	}
}

func TestResolveProxyTarget_UnknownSessionIsUserError(t *testing.T) {
	_, err := resolveProxyTarget(fixtureConfig(), "nobody@datum.net@api.datum.net", "", "", false)
	if err == nil {
		t.Fatal("expected an error for an unknown --session")
	}
	userErr, ok := customerrors.IsUserError(err)
	if !ok {
		t.Fatalf("error = %v (%T), want a UserError", err, err)
	}
	if !strings.Contains(userErr.Message, "nobody@datum.net@api.datum.net") {
		t.Errorf("message = %q, want it to name the unknown session", userErr.Message)
	}
	if !strings.Contains(userErr.Hint, "datumctl auth list") {
		t.Errorf("hint = %q, want it to point at 'datumctl auth list'", userErr.Hint)
	}
}

func TestResolveProxyTarget_ConflictingScopeFlags(t *testing.T) {
	tests := []struct {
		name         string
		project      string
		organization string
		platformWide bool
	}{
		{name: "project and organization", project: "p", organization: "o"},
		{name: "platform-wide and project", project: "p", platformWide: true},
		{name: "platform-wide and organization", organization: "o", platformWide: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveProxyTarget(fixtureConfig(), "", tc.project, tc.organization, tc.platformWide)
			if err == nil {
				t.Fatal("expected an error for conflicting scope flags")
			}
			if _, ok := customerrors.IsUserError(err); !ok {
				t.Errorf("error = %v (%T), want a UserError", err, err)
			}
		})
	}
}

func TestTLSClientConfig(t *testing.T) {
	t.Run("empty settings keep defaults", func(t *testing.T) {
		cfg, err := tlsClientConfig(client.EndpointTLS{})
		if err != nil {
			t.Fatalf("tlsClientConfig: %v", err)
		}
		if cfg != nil {
			t.Errorf("config = %+v, want nil for default TLS", cfg)
		}
	})

	t.Run("server name and insecure carry over", func(t *testing.T) {
		cfg, err := tlsClientConfig(client.EndpointTLS{
			ServerName:            "internal.example",
			InsecureSkipTLSVerify: true,
		})
		if err != nil {
			t.Fatalf("tlsClientConfig: %v", err)
		}
		if cfg == nil {
			t.Fatal("config = nil, want TLS settings applied")
		}
		if cfg.ServerName != "internal.example" {
			t.Errorf("ServerName = %q, want %q", cfg.ServerName, "internal.example")
		}
		if !cfg.InsecureSkipVerify {
			t.Error("InsecureSkipVerify = false, want true")
		}
	})

	t.Run("invalid CA data errors", func(t *testing.T) {
		_, err := tlsClientConfig(client.EndpointTLS{CAData: []byte("not a pem")})
		if err == nil {
			t.Fatal("expected an error for unparseable CA data")
		}
		userErr, ok := customerrors.IsUserError(err)
		if !ok {
			t.Fatalf("error = %v (%T), want a UserError with actionable guidance", err, err)
		}
		if !strings.Contains(userErr.Hint, "datumctl login") {
			t.Errorf("hint = %q, want it to point at 'datumctl login'", userErr.Hint)
		}
	})
}
