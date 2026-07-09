package authutil

import (
	"encoding/json"
	"errors"
	"strings"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
)

// userKeyFor returns the keyring key that identifies a stored credential.
// Credentials are keyed by identity *and* auth hostname so that logging in with
// the same identity (e.g. email) in two environments — production and staging —
// does not collapse onto a single keyring entry where the second login silently
// overwrites the first.
func userKeyFor(identity, authHostname string) string {
	if identity == "" || authHostname == "" {
		return identity
	}
	return identity + "@" + authHostname
}

// EnsureUserKeysMigrated runs MigrateUserKeys on cfg and persists the config when
// anything changed. It is safe to call on every auth resolution: when there is
// nothing to migrate it performs no keyring writes and no disk writes.
func EnsureUserKeysMigrated(cfg *datumconfig.ConfigV1Beta1) error {
	changed, err := MigrateUserKeys(cfg)
	if err != nil {
		return err
	}
	if changed {
		return datumconfig.SaveV1Beta1(cfg)
	}
	return nil
}

// MigrateUserKeys upgrades legacy sessions whose keyring key was scoped only by
// identity (an email address, historically) to the environment-scoped scheme
// "<identity>@<auth-hostname>".
//
// Before this scheme, two logins with the same email in different environments
// shared one keyring entry, so the second login silently clobbered the first —
// and switching between the sessions presented a token minted for the wrong
// environment, producing spurious "unauthorized" errors.
//
// For each legacy session it computes the new key and, when the single stored
// blob was minted for that session's auth hostname, copies the blob to the new
// key. When two sessions collided only the environment whose token currently
// occupies the blob is recovered; the other session is left without a stored
// token and must be re-authenticated with `datumctl login`.
//
// The old keyring entries are left untouched — they become inert once no session
// references them — so the migration never destroys a token it could not first
// copy. It reports whether cfg was modified; callers persist cfg when true.
func MigrateUserKeys(cfg *datumconfig.ConfigV1Beta1) (bool, error) {
	if cfg == nil {
		return false, nil
	}

	changed := false
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		authHost := s.Endpoint.AuthHostname
		if authHost == "" || s.UserKey == "" {
			continue
		}
		if strings.HasSuffix(s.UserKey, "@"+authHost) {
			continue // already environment-scoped
		}

		identity := s.UserEmail
		if identity == "" {
			identity = s.UserKey
		}
		newKey := userKeyFor(identity, authHost)
		if newKey == s.UserKey {
			continue
		}

		if err := rehomeCredential(s.UserKey, newKey, authHost); err != nil {
			return changed, err
		}
		s.UserKey = newKey
		changed = true
	}

	return changed, nil
}

// rehomeCredential copies the credential stored under oldKey to newKey when the
// stored blob was minted for authHost and newKey does not already hold a
// credential. A missing, unparseable, or mismatched blob is not an error: the
// session simply starts out needing a fresh login.
func rehomeCredential(oldKey, newKey, authHost string) error {
	raw, err := keyring.Get(ServiceName, oldKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	if raw == "" {
		return nil
	}

	var creds StoredCredentials
	if err := json.Unmarshal([]byte(raw), &creds); err != nil {
		// Unparseable legacy blob: don't block migration, just skip the copy.
		return nil
	}

	// Only claim the blob when it belongs to this session's environment. With a
	// cross-environment collision exactly one session matches; the others need a
	// fresh login. Blobs with no recorded hostname are assumed to match.
	if creds.Hostname != "" && creds.Hostname != authHost {
		return nil
	}

	if existing, err := keyring.Get(ServiceName, newKey); err == nil && existing != "" {
		return nil // already migrated
	} else if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}

	if err := keyring.Set(ServiceName, newKey, raw); err != nil {
		return err
	}
	return AddKnownUserKey(newKey)
}
