package authutil

import (
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
)

// TestGetUserKeyForCurrentSession_FollowsSwitchNotLegacyKeyring reproduces the
// datumctl/datumctl#292 bug: after `datumctl auth switch` to a different
// account, the legacy keyring `active_user` marker still names whichever
// account logged in last. Callers that read it directly (rather than the
// config's ActiveSession/CurrentContext) keep authenticating as the old
// account. GetUserKeyForCurrentSession must ignore the stale legacy marker
// once real sessions exist in the config.
func TestGetUserKeyForCurrentSession_FollowsSwitchNotLegacyKeyring(t *testing.T) {
	mockKeyring(t)

	const userKeyA = "userA@example.com@auth.datum.net"
	const userKeyB = "userB@example.com@auth.staging.env.datum.net"

	// userA logged in most recently before the switch, so the legacy keyring
	// marker names userA. Real logins always write this, independent of the
	// v1beta1 config file.
	if err := keyring.Set(ServiceName, ActiveUserKey, userKeyA); err != nil {
		t.Fatalf("seed legacy active_user: %v", err)
	}

	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{
			Name:      "userA@example.com@api.datum.net",
			UserKey:   userKeyA,
			UserEmail: "userA@example.com",
			Endpoint:  datumconfig.Endpoint{Server: "https://api.datum.net", AuthHostname: "auth.datum.net"},
		},
		{
			Name:      "userB@example.com@api.staging.env.datum.net",
			UserKey:   userKeyB,
			UserEmail: "userB@example.com",
			Endpoint:  datumconfig.Endpoint{Server: "https://api.staging.env.datum.net", AuthHostname: "auth.staging.env.datum.net"},
		},
	}

	// `auth switch` to userB: it sets ActiveSession and clears CurrentContext
	// (no LastContext for the newly-selected session), exactly as
	// internal/cmd/auth/switch.go does. It never touches the keyring.
	cfg.ActiveSession = "userB@example.com@api.staging.env.datum.net"
	cfg.CurrentContext = ""
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	gotKey, session, err := GetUserKeyForCurrentSession()
	if err != nil {
		t.Fatalf("GetUserKeyForCurrentSession: %v", err)
	}
	if gotKey != userKeyB {
		t.Errorf("user key = %q, want %q (the switched-to account, not the stale legacy keyring marker %q)", gotKey, userKeyB, userKeyA)
	}
	if session == nil {
		t.Fatal("session = nil, want the userB session")
	}
	if session.Name != "userB@example.com@api.staging.env.datum.net" {
		t.Errorf("session.Name = %q, want the userB session", session.Name)
	}
}

// TestGetUserKeyForCurrentSession_CurrentContextWinsOverActiveSession pins the
// same single-source-of-truth precedence #248 established for whoami: when
// the current context names a different session than ActiveSession, the
// context wins, so `auth get-token`, `docs openapi`, and plugin dispatch all
// agree with whoami and the main API client about which account is active.
func TestGetUserKeyForCurrentSession_CurrentContextWinsOverActiveSession(t *testing.T) {
	mockKeyring(t)

	const userKeyProd = "user@example.com@auth.datum.net"
	const userKeyStaging = "user@example.com@auth.staging.env.datum.net"

	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{
			Name:      "prod",
			UserKey:   userKeyProd,
			UserEmail: "user@example.com",
			Endpoint:  datumconfig.Endpoint{Server: "https://api.datum.net", AuthHostname: "auth.datum.net"},
		},
		{
			Name:      "staging",
			UserKey:   userKeyStaging,
			UserEmail: "user@example.com",
			Endpoint:  datumconfig.Endpoint{Server: "https://api.staging.env.datum.net", AuthHostname: "auth.staging.env.datum.net"},
		},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{
		{Name: "prod/prod-ctx", Session: "prod"},
	}
	// ActiveSession still says staging (e.g. from the last `auth switch`), but
	// the user later ran `ctx use` into a prod context without switching
	// sessions explicitly.
	cfg.ActiveSession = "staging"
	cfg.CurrentContext = "prod/prod-ctx"
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	gotKey, session, err := GetUserKeyForCurrentSession()
	if err != nil {
		t.Fatalf("GetUserKeyForCurrentSession: %v", err)
	}
	if gotKey != userKeyProd {
		t.Errorf("user key = %q, want %q (the current context's session)", gotKey, userKeyProd)
	}
	if session == nil || session.Name != "prod" {
		t.Errorf("session = %+v, want the prod session", session)
	}
}
