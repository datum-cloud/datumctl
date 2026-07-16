package client

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// The fixtures below always carry a UserKey and an endpoint Server so that
// resolution never falls back to the keyring, keeping these tests hermetic.

func configWithSessions(active string, sessions ...datumconfig.Session) *datumconfig.ConfigV1Beta1 {
	return &datumconfig.ConfigV1Beta1{
		APIVersion:    datumconfig.V1Beta1APIVersion,
		Kind:          datumconfig.DefaultKind,
		Sessions:      sessions,
		ActiveSession: active,
	}
}

// TestResolveSessionEndpoint_BaseServer pins the base-server normalization to
// what ToRESTConfig's resolution applies to session.Endpoint.Server:
// EnsureScheme then CleanBaseServer.
func TestResolveSessionEndpoint_BaseServer(t *testing.T) {
	tests := []struct {
		name   string
		server string
		want   string
	}{
		{name: "scheme kept", server: "https://api.datum.net", want: "https://api.datum.net"},
		{name: "scheme added", server: "api.staging.env.datum.net", want: "https://api.staging.env.datum.net"},
		{name: "trailing slash stripped", server: "https://api.datum.net/", want: "https://api.datum.net"},
		{name: "http scheme kept", server: "http://localhost:8080", want: "http://localhost:8080"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configWithSessions("s", datumconfig.Session{
				Name:      "s",
				UserKey:   "user@auth.datum.net",
				UserEmail: "user@datum.net",
				Endpoint:  datumconfig.Endpoint{Server: tc.server},
			})
			session, endpoint, err := ResolveSessionEndpoint(cfg, "")
			if err != nil {
				t.Fatalf("ResolveSessionEndpoint: %v", err)
			}
			if session == nil || session.Name != "s" {
				t.Fatalf("session = %+v, want the active session", session)
			}
			if endpoint.BaseServer != tc.want {
				t.Errorf("BaseServer = %q, want %q", endpoint.BaseServer, tc.want)
			}
			if endpoint.UserKey != "user@auth.datum.net" {
				t.Errorf("UserKey = %q, want the session's user key", endpoint.UserKey)
			}
		})
	}
}

// TestResolveSessionEndpoint_TLS pins the endpoint TLS handling to what
// ToRESTConfig applies: TLSServerName and InsecureSkipTLSVerify carried
// as-is, CertificateAuthorityData base64-decoded.
func TestResolveSessionEndpoint_TLS(t *testing.T) {
	caPEM := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n")

	tests := []struct {
		name        string
		endpoint    datumconfig.Endpoint
		want        EndpointTLS
		wantErr     bool
		wantErrPart string
	}{
		{
			name:     "no TLS settings",
			endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"},
			want:     EndpointTLS{},
		},
		{
			name: "server name and insecure",
			endpoint: datumconfig.Endpoint{
				Server:                "https://api.staging.env.datum.net",
				TLSServerName:         "api.internal",
				InsecureSkipTLSVerify: true,
			},
			want: EndpointTLS{ServerName: "api.internal", InsecureSkipTLSVerify: true},
		},
		{
			name: "certificate authority data decoded",
			endpoint: datumconfig.Endpoint{
				Server:                   "https://api.staging.env.datum.net",
				CertificateAuthorityData: base64.StdEncoding.EncodeToString(caPEM),
			},
			want: EndpointTLS{CAData: caPEM},
		},
		{
			name: "invalid certificate authority data",
			endpoint: datumconfig.Endpoint{
				Server:                   "https://api.staging.env.datum.net",
				CertificateAuthorityData: "not!!!base64",
			},
			wantErr:     true,
			wantErrPart: "tls-session",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configWithSessions("tls-session", datumconfig.Session{
				Name:      "tls-session",
				UserKey:   "user@auth.datum.net",
				UserEmail: "user@datum.net",
				Endpoint:  tc.endpoint,
			})
			_, endpoint, err := ResolveSessionEndpoint(cfg, "")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.Error(), tc.wantErrPart) {
					t.Errorf("error = %q, want it to contain %q", err, tc.wantErrPart)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveSessionEndpoint: %v", err)
			}
			if endpoint.TLS.ServerName != tc.want.ServerName {
				t.Errorf("ServerName = %q, want %q", endpoint.TLS.ServerName, tc.want.ServerName)
			}
			if endpoint.TLS.InsecureSkipTLSVerify != tc.want.InsecureSkipTLSVerify {
				t.Errorf("InsecureSkipTLSVerify = %v, want %v",
					endpoint.TLS.InsecureSkipTLSVerify, tc.want.InsecureSkipTLSVerify)
			}
			if !bytes.Equal(endpoint.TLS.CAData, tc.want.CAData) {
				t.Errorf("CAData = %q, want %q", endpoint.TLS.CAData, tc.want.CAData)
			}
		})
	}
}

func TestResolveSessionEndpoint_NamedSessionWinsOverActive(t *testing.T) {
	cfg := configWithSessions("active",
		datumconfig.Session{
			Name:      "active",
			UserKey:   "active@auth.datum.net",
			UserEmail: "active@datum.net",
			Endpoint:  datumconfig.Endpoint{Server: "https://api.datum.net"},
		},
		datumconfig.Session{
			Name:      "pinned",
			UserKey:   "pinned@auth.staging.env.datum.net",
			UserEmail: "pinned@datum.net",
			Endpoint:  datumconfig.Endpoint{Server: "https://api.staging.env.datum.net"},
		},
	)

	session, endpoint, err := ResolveSessionEndpoint(cfg, "pinned")
	if err != nil {
		t.Fatalf("ResolveSessionEndpoint: %v", err)
	}
	if session.Name != "pinned" {
		t.Errorf("session = %q, want the pinned session", session.Name)
	}
	if endpoint.BaseServer != "https://api.staging.env.datum.net" {
		t.Errorf("BaseServer = %q, want the pinned session's endpoint", endpoint.BaseServer)
	}
	if endpoint.UserKey != "pinned@auth.staging.env.datum.net" {
		t.Errorf("UserKey = %q, want the pinned session's user key", endpoint.UserKey)
	}
}

func TestResolveSessionEndpoint_UnknownNameIsUserError(t *testing.T) {
	cfg := configWithSessions("active", datumconfig.Session{
		Name:      "active",
		UserKey:   "active@auth.datum.net",
		UserEmail: "active@datum.net",
		Endpoint:  datumconfig.Endpoint{Server: "https://api.datum.net"},
	})

	_, _, err := ResolveSessionEndpoint(cfg, "missing")
	if err == nil {
		t.Fatal("expected an error for an unknown session name")
	}
	userErr, ok := customerrors.IsUserError(err)
	if !ok {
		t.Fatalf("error = %v (%T), want a UserError", err, err)
	}
	if !strings.Contains(userErr.Hint, "datumctl auth list") {
		t.Errorf("hint = %q, want it to point at 'datumctl auth list'", userErr.Hint)
	}
}

// TestResolveSessionEndpoint_ActiveSessionFromContext pins ActiveSessionEntry
// semantics: with no explicit active session, the session referenced by the
// current context is used.
func TestResolveSessionEndpoint_ActiveSessionFromContext(t *testing.T) {
	cfg := configWithSessions("", datumconfig.Session{
		Name:      "ctx-session",
		UserKey:   "user@auth.datum.net",
		UserEmail: "user@datum.net",
		Endpoint:  datumconfig.Endpoint{Server: "https://api.datum.net"},
	})
	cfg.Contexts = []datumconfig.DiscoveredContext{{
		Name:           "org-a",
		Session:        "ctx-session",
		OrganizationID: "org-a",
	}}
	cfg.CurrentContext = "org-a"

	session, endpoint, err := ResolveSessionEndpoint(cfg, "")
	if err != nil {
		t.Fatalf("ResolveSessionEndpoint: %v", err)
	}
	if session == nil || session.Name != "ctx-session" {
		t.Fatalf("session = %+v, want the context's session", session)
	}
	if endpoint.BaseServer != "https://api.datum.net" {
		t.Errorf("BaseServer = %q, want %q", endpoint.BaseServer, "https://api.datum.net")
	}
}
