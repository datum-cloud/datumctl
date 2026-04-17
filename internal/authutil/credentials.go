// Package authutil provides shared constants and functions for handling authentication
// credentials, including storage in the system keyring and OAuth2 token management.
package authutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"golang.org/x/oauth2"
)

// ServiceName is the identifier used for storing credentials in the system keyring.
const ServiceName = "datumctl-auth"

// ActiveUserKey is the key used in the keyring to store the identifier of the currently active user credentials.
const ActiveUserKey = "active_user"

// KnownUsersKey is the key used in the keyring to store a JSON list of known user identifiers (email@hostname).
const KnownUsersKey = "known_users"

// ErrNoActiveUser indicates that no active user is set in the keyring.
var ErrNoActiveUser = customerrors.NewUserErrorWithHint(
	"No active user found.",
	"Please login first using: `datumctl auth login`",
)

// MachineAccountState holds fields needed to re-mint a JWT when the access token expires.
// Only populated when CredentialType == "machine_account".
type MachineAccountState struct {
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key,omitempty"`
	TokenURI     string `json:"token_uri"`
	Scope        string `json:"scope,omitempty"`
	// PrivateKeyPath is the path to an on-disk file containing the PEM-encoded
	// private key. Used when the key is too large to store in the keyring (e.g.
	// on macOS where the Keychain has a per-item size limit). If non-empty, the
	// token source reads the key from this path instead of PrivateKey.
	PrivateKeyPath string `json:"private_key_path,omitempty"`
}

// StoredCredentials holds all necessary information for a single authenticated session.
type StoredCredentials struct {
	Hostname         string        `json:"hostname"`           // The auth server hostname used (e.g., auth.datum.net)
	APIHostname      string        `json:"api_hostname"`       // The API server hostname (e.g., api.datum.net)
	ClientID         string        `json:"client_id"`          // The OAuth2 Client ID used
	EndpointAuthURL  string        `json:"endpoint_auth_url"`  // Discovered OIDC Authorization Endpoint URL
	EndpointTokenURL string        `json:"endpoint_token_url"` // Discovered OIDC Token Endpoint URL
	Scopes           []string      `json:"scopes"`             // Scopes requested/granted
	Token            *oauth2.Token `json:"token"`              // The retrieved OAuth2 token (includes refresh token, expiry)
	UserName         string        `json:"user_name"`          // User's Name (e.g., from 'name' claim)
	UserEmail        string        `json:"user_email"`         // User's Email (e.g., from 'email' claim)
	Subject          string        `json:"subject"`            // User's Subject ID (sub claim from JWT)
	// CredentialType distinguishes how stored credentials should be refreshed.
	// "" or "interactive" → standard oauth2 refresh token path.
	// "machine_account"  → re-mint JWT and re-exchange on expiry.
	CredentialType string               `json:"credential_type,omitempty"`
	MachineAccount *MachineAccountState `json:"machine_account,omitempty"`
}

// GetActiveCredentials retrieves the StoredCredentials for the currently active user.
func GetActiveCredentials() (*StoredCredentials, string, error) {
	if HasAmbientToken() {
		creds, err := ambientCredentials()
		if err != nil {
			return nil, "", err
		}
		return creds, AmbientUserKey, nil
	}

	activeUserKey, err := keyring.Get(ServiceName, ActiveUserKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, "", ErrNoActiveUser
		}
		return nil, "", fmt.Errorf("failed to get active user from keyring: %w", err)
	}

	if activeUserKey == "" {
		return nil, "", ErrNoActiveUser
	}

	creds, err := GetStoredCredentials(activeUserKey)
	if err != nil {
		return nil, activeUserKey, err // Return key even on error for context
	}
	return creds, activeUserKey, nil
}

// GetStoredCredentials retrieves and unmarshals credentials for a specific user key.
func GetStoredCredentials(userKey string) (*StoredCredentials, error) {
	if HasAmbientToken() && userKey == AmbientUserKey {
		return ambientCredentials()
	}

	credsJSON, err := keyring.Get(ServiceName, userKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, fmt.Errorf("credentials for user '%s' not found in keyring", userKey)
		}
		return nil, fmt.Errorf("failed to get credentials for '%s' from keyring: %w", userKey, err)
	}

	if credsJSON == "" {
		return nil, fmt.Errorf("empty credentials found for user '%s' in keyring", userKey)
	}

	var creds StoredCredentials
	if err := json.Unmarshal([]byte(credsJSON), &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials for '%s': %w", userKey, err)
	}

	if creds.Token == nil {
		return nil, fmt.Errorf("stored credentials for '%s' are missing token information", userKey)
	}

	return &creds, nil
}

// persistingTokenSource wraps an oauth2.TokenSource and persists token updates to the keyring.
type persistingTokenSource struct {
	ctx     context.Context
	source  oauth2.TokenSource
	userKey string
	creds   *StoredCredentials
	mu      sync.Mutex
}

// Token implements oauth2.TokenSource.
// It retrieves a token from the underlying source and persists it to the keyring if refreshed.
func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	currentAccessToken := ""
	if p.creds.Token != nil {
		currentAccessToken = p.creds.Token.AccessToken
	}

	// Get token from the underlying source (may trigger refresh)
	newToken, err := p.source.Token()
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) {
			if retrieveErr.ErrorCode == "invalid_grant" || retrieveErr.ErrorCode == "invalid_request" {
				return nil, customerrors.WrapUserErrorWithHint(
					"Authentication session has expired or refresh token is no longer valid.",
					"Please re-authenticate using: `datumctl auth login`",
					err,
				)
			}
		}
		return nil, err
	}

	// Persist the token if it was refreshed
	if newToken.AccessToken != currentAccessToken {
		p.creds.Token = newToken

		credsJSON, marshalErr := json.Marshal(p.creds)
		if marshalErr != nil {
			return newToken, fmt.Errorf("failed to marshal updated credentials: %w", marshalErr)
		}

		if setErr := keyring.Set(ServiceName, p.userKey, string(credsJSON)); setErr != nil {
			return newToken, fmt.Errorf("failed to persist refreshed token to keyring: %w", setErr)
		}
	}

	return newToken, nil
}

// GetTokenSource creates an oauth2.TokenSource for the active user.
// This source will automatically refresh the token if it's expired and persist updates to the keyring.
func GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if HasAmbientToken() {
		return ambientTokenSource{}, nil
	}
	creds, userKey, err := GetActiveCredentials()
	if err != nil {
		return nil, err
	}
	return tokenSourceFor(ctx, userKey, creds)
}

// GetTokenSourceForUser creates an oauth2.TokenSource for a specific user key.
// Used by multi-user flows (sessions, kubectl exec plugin, MCP).
func GetTokenSourceForUser(ctx context.Context, userKey string) (oauth2.TokenSource, error) {
	if HasAmbientToken() {
		return ambientTokenSource{}, nil
	}
	creds, err := GetStoredCredentials(userKey)
	if err != nil {
		return nil, err
	}
	return tokenSourceFor(ctx, userKey, creds)
}

func tokenSourceFor(ctx context.Context, userKey string, creds *StoredCredentials) (oauth2.TokenSource, error) {
	if creds.CredentialType == "machine_account" {
		if creds.MachineAccount == nil {
			return nil, fmt.Errorf("machine account credentials are missing from stored session")
		}
		return &machineAccountTokenSource{
			ctx:     ctx,
			creds:   creds,
			userKey: userKey,
		}, nil
	}

	// Rebuild the oauth2.Config needed for refreshing
	conf := &oauth2.Config{
		ClientID: creds.ClientID,
		Scopes:   creds.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  creds.EndpointAuthURL,
			TokenURL: creds.EndpointTokenURL,
		},
		// RedirectURL not needed for token refresh
	}

	// Create the base TokenSource with the stored token
	baseSource := conf.TokenSource(ctx, creds.Token)

	// Wrap it with our persisting source
	return &persistingTokenSource{
		ctx:     ctx,
		source:  baseSource,
		userKey: userKey,
		creds:   creds,
	}, nil
}

// GetUserIDFromToken extracts the user ID (sub claim) from the stored credentials.
func GetUserIDFromToken(ctx context.Context) (string, error) {
	creds, _, err := GetActiveCredentials()
	if err != nil {
		return "", err
	}
	return userIDFromCreds(creds)
}

// GetUserIDFromTokenForUser extracts the user ID (sub claim) for a specific user key.
func GetUserIDFromTokenForUser(userKey string) (string, error) {
	creds, err := GetStoredCredentials(userKey)
	if err != nil {
		return "", err
	}
	return userIDFromCreds(creds)
}

func userIDFromCreds(creds *StoredCredentials) (string, error) {
	if creds.Subject == "" {
		return "", errors.New("subject (user ID) not found in stored credentials")
	}
	return creds.Subject, nil
}

// GetActiveUserKey retrieves the key for the currently active user (e.g., email@example.com).
func GetActiveUserKey() (string, error) {
	if HasAmbientToken() {
		return AmbientUserKey, nil
	}

	activeUserKey, err := keyring.Get(ServiceName, ActiveUserKey)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoActiveUser
		}
		return "", fmt.Errorf("failed to get active user from keyring: %w", err)
	}

	if activeUserKey == "" {
		return "", ErrNoActiveUser
	}

	return activeUserKey, nil
}

// GetAPIHostname returns the API hostname from stored credentials.
// If no API hostname is stored, it attempts to derive it from the auth hostname.
func GetAPIHostname() (string, error) {
	if HasAmbientToken() {
		return ambientAPIHostname()
	}
	creds, _, err := GetActiveCredentials()
	if err != nil {
		return "", err
	}
	return apiHostnameFromCreds(creds)
}

// GetAPIHostnameForUser returns the API hostname from stored credentials for
// a specific user key.
func GetAPIHostnameForUser(userKey string) (string, error) {
	if HasAmbientToken() {
		return ambientAPIHostname()
	}
	creds, err := GetStoredCredentials(userKey)
	if err != nil {
		return "", err
	}
	return apiHostnameFromCreds(creds)
}

func apiHostnameFromCreds(creds *StoredCredentials) (string, error) {
	if creds.APIHostname != "" {
		return creds.APIHostname, nil
	}
	return DeriveAPIHostname(creds.Hostname)
}

// DeriveAPIHostname attempts to convert an authentication hostname (e.g., auth.datum.net)
// to its corresponding API hostname (e.g., api.datum.net).
func DeriveAPIHostname(authHostname string) (string, error) {
	if authHostname == "" {
		return "", errors.New("cannot derive API hostname from empty auth hostname")
	}
	// Simple replacement logic for now
	if strings.HasPrefix(authHostname, "auth.") {
		return "api." + strings.TrimPrefix(authHostname, "auth."), nil
	}
	// Add other potential derivation logic here if needed

	// Return an error if no known pattern matches.
	return "", fmt.Errorf("could not derive API hostname from '%s'", authHostname)
}
