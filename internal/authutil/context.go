package authutil

import (
	"encoding/json"
	"errors"
	"fmt"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
)

// GetUserKeyForCurrentSession resolves the user key from the active v1beta1
// session. If no session exists in the config file, it performs a one-time
// bootstrap from existing keyring credentials — for users who logged in
// before the v1beta1 config layer existed.
func GetUserKeyForCurrentSession() (string, *datumconfig.Session, error) {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return "", nil, err
	}

	if session := cfg.ActiveSessionEntry(); session != nil && session.UserKey != "" {
		return session.UserKey, session, nil
	}

	session, err := bootstrapSessionFromKeyring(cfg)
	if err != nil {
		return "", nil, err
	}
	return session.UserKey, session, nil
}

// GetUserKey returns just the user key for the active session. Convenience
// wrapper for callers that don't need the session itself.
func GetUserKey() (string, error) {
	key, _, err := GetUserKeyForCurrentSession()
	return key, err
}

// GetUserKeyForSession looks up a session by name and returns its user key.
// Used by the kubectl exec plugin path: `datumctl auth update-kubeconfig`
// writes the session name into the exec args, and `datumctl auth get-token`
// resolves it back here.
func GetUserKeyForSession(sessionName string) (string, error) {
	if sessionName == "" {
		return "", ErrNoActiveUser
	}
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return "", err
	}
	session := cfg.SessionByName(sessionName)
	if session == nil {
		return "", fmt.Errorf("no session named %q — run 'datumctl login' or 'datumctl auth update-kubeconfig'", sessionName)
	}
	if session.UserKey == "" {
		return "", fmt.Errorf("session %q has no user key", sessionName)
	}
	return session.UserKey, nil
}

// bootstrapSessionFromKeyring detects a pre-v1beta1 user — credentials in the
// keyring with no config file — and creates a v1beta1 Session for them so
// subsequent commands "just work." Returns the new session, or an error if no
// usable keyring credentials are present.
//
// Works for both interactive and machine-account credentials, since both
// populate the same StoredCredentials fields that Session consumes.
func bootstrapSessionFromKeyring(cfg *datumconfig.ConfigV1Beta1) (*datumconfig.Session, error) {
	if cfg == nil {
		return nil, ErrNoActiveUser
	}

	candidateKey, err := pickLegacyUserKey()
	if err != nil {
		return nil, err
	}
	if candidateKey == "" {
		return nil, ErrNoActiveUser
	}

	creds, err := GetStoredCredentials(candidateKey)
	if err != nil {
		return nil, err
	}

	apiHostname := creds.APIHostname
	if apiHostname == "" {
		apiHostname, err = DeriveAPIHostname(creds.Hostname)
		if err != nil {
			return nil, err
		}
	}

	userKey, err := normalizeUserKey(candidateKey, creds)
	if err != nil {
		return nil, err
	}

	session := datumconfig.Session{
		Name:      datumconfig.SessionName(creds.UserEmail, apiHostname),
		UserKey:   userKey,
		UserEmail: creds.UserEmail,
		UserName:  creds.UserName,
		Endpoint: datumconfig.Endpoint{
			Server:       datumconfig.CleanBaseServer(datumconfig.EnsureScheme(apiHostname)),
			AuthHostname: creds.Hostname,
		},
	}

	cfg.UpsertSession(session)
	if cfg.ActiveSession == "" {
		cfg.ActiveSession = session.Name
	}
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return nil, err
	}

	return &session, nil
}

// pickLegacyUserKey returns the keyring user key to bootstrap from. Prefers
// the legacy ActiveUserKey; falls back to KnownUsersKey when exactly one user
// is known.
func pickLegacyUserKey() (string, error) {
	legacyKey, err := keyring.Get(ServiceName, ActiveUserKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("failed to read legacy active user: %w", err)
	}
	if legacyKey != "" {
		return legacyKey, nil
	}

	knownUsersJSON, err := keyring.Get(ServiceName, KnownUsersKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return "", fmt.Errorf("failed to read known users: %w", err)
	}
	if knownUsersJSON == "" {
		return "", nil
	}

	var knownUsers []string
	if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
		return "", fmt.Errorf("failed to unmarshal known users: %w", err)
	}
	if len(knownUsers) == 1 {
		return knownUsers[0], nil
	}
	return "", nil
}

// normalizeUserKey ensures the keyring stores the user under the canonical
// "<subject>@<hostname>" key. If the existing key is in an older format, it
// re-stores the credentials under the new key and returns the new key.
func normalizeUserKey(userKey string, creds *StoredCredentials) (string, error) {
	if creds.Subject == "" || creds.Hostname == "" {
		return userKey, nil
	}

	canonical := fmt.Sprintf("%s@%s", creds.Subject, creds.Hostname)
	if canonical == userKey {
		return userKey, nil
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return "", err
	}
	if err := keyring.Set(ServiceName, canonical, string(credsJSON)); err != nil {
		return "", err
	}
	if err := AddKnownUserKey(canonical); err != nil {
		return "", err
	}
	return canonical, nil
}
