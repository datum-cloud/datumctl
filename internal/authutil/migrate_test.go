package authutil

import (
	"encoding/json"
	"testing"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
	"golang.org/x/oauth2"
)

func TestUserKeyFor(t *testing.T) {
	cases := []struct {
		identity, host, want string
	}{
		{"swells@datum.net", "auth.datum.net", "swells@datum.net@auth.datum.net"},
		{"swells@datum.net", "auth.staging.env.datum.net", "swells@datum.net@auth.staging.env.datum.net"},
		{"swells@datum.net", "", "swells@datum.net"}, // no host to qualify with
		{"", "auth.datum.net", ""}, // no identity
	}
	for _, c := range cases {
		if got := userKeyFor(c.identity, c.host); got != c.want {
			t.Errorf("userKeyFor(%q, %q) = %q, want %q", c.identity, c.host, got, c.want)
		}
	}
}

// mockKeyring points HOME at an empty temp dir before installing the mock
// keyring provider. Without the HOME isolation, a real fallback credentials
// file at ~/.datumctl/credentials.json would silently hijack the mock (the
// wrapper prefers an existing on-disk store), making these tests read — and
// write — the developer's real credential store.
func mockKeyring(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	keyring.MockInit()
}

// storeCreds writes a StoredCredentials blob to the keyring under key.
func storeCreds(t *testing.T, key, authHostname string) {
	t.Helper()
	blob, err := json.Marshal(StoredCredentials{
		Hostname:  authHostname,
		UserEmail: "swells@datum.net",
		Subject:   "sub-" + authHostname,
		Token:     &oauth2.Token{AccessToken: "tok-" + authHostname},
	})
	if err != nil {
		t.Fatalf("marshal creds: %v", err)
	}
	if err := keyring.Set(ServiceName, key, string(blob)); err != nil {
		t.Fatalf("seed keyring: %v", err)
	}
}

// collidingConfig returns a config with a staging and production session that
// share the legacy bare-email user key, mirroring the real-world bug.
func collidingConfig() *datumconfig.ConfigV1Beta1 {
	return &datumconfig.ConfigV1Beta1{
		Sessions: []datumconfig.Session{
			{
				Name:      "swells@datum.net@api.staging.env.datum.net",
				UserKey:   "swells@datum.net",
				UserEmail: "swells@datum.net",
				Endpoint:  datumconfig.Endpoint{AuthHostname: "auth.staging.env.datum.net"},
			},
			{
				Name:      "swells@datum.net@api.datum.net",
				UserKey:   "swells@datum.net",
				UserEmail: "swells@datum.net",
				Endpoint:  datumconfig.Endpoint{AuthHostname: "auth.datum.net"},
			},
		},
	}
}

func TestMigrateUserKeys_CollisionRecoversMatchingEnvironment(t *testing.T) {
	mockKeyring(t)

	// The single shared blob currently holds the staging token (last written).
	storeCreds(t, "swells@datum.net", "auth.staging.env.datum.net")

	cfg := collidingConfig()
	changed, err := MigrateUserKeys(cfg)
	if err != nil {
		t.Fatalf("MigrateUserKeys: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}

	stagingKey := "swells@datum.net@auth.staging.env.datum.net"
	prodKey := "swells@datum.net@auth.datum.net"

	if got := cfg.Sessions[0].UserKey; got != stagingKey {
		t.Errorf("staging session UserKey = %q, want %q", got, stagingKey)
	}
	if got := cfg.Sessions[1].UserKey; got != prodKey {
		t.Errorf("prod session UserKey = %q, want %q", got, prodKey)
	}

	// The surviving (staging) token is re-homed under the new key...
	if _, err := GetStoredCredentials(stagingKey); err != nil {
		t.Errorf("expected staging creds under new key, got error: %v", err)
	}
	// ...while the clobbered (production) session has no token and must re-login.
	if _, err := keyring.Get(ServiceName, prodKey); err == nil {
		t.Error("expected no production credentials under new key")
	}
}

func TestMigrateUserKeys_Idempotent(t *testing.T) {
	mockKeyring(t)
	storeCreds(t, "swells@datum.net", "auth.staging.env.datum.net")

	cfg := collidingConfig()
	if _, err := MigrateUserKeys(cfg); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	changed, err := MigrateUserKeys(cfg)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if changed {
		t.Error("second migration should be a no-op")
	}
}

func TestMigrateUserKeys_AlreadyQualifiedNoop(t *testing.T) {
	mockKeyring(t)

	cfg := &datumconfig.ConfigV1Beta1{
		Sessions: []datumconfig.Session{{
			Name:      "swells@datum.net@api.datum.net",
			UserKey:   "swells@datum.net@auth.datum.net",
			UserEmail: "swells@datum.net",
			Endpoint:  datumconfig.Endpoint{AuthHostname: "auth.datum.net"},
		}},
	}

	changed, err := MigrateUserKeys(cfg)
	if err != nil {
		t.Fatalf("MigrateUserKeys: %v", err)
	}
	if changed {
		t.Error("already-qualified session should not be migrated")
	}
}
