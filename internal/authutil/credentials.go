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
}

// GetActiveCredentials retrieves the StoredCredentials for the currently active user.
func GetActiveCredentials() (*StoredCredentials, string, error) {
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
	creds, userKey, err := GetActiveCredentials()
	if err != nil {
		return nil, err
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

	if creds.Subject == "" {
		return "", errors.New("subject (user ID) not found in stored credentials")
	}

	return creds.Subject, nil
}

// GetActiveUserKey retrieves the key for the currently active user (e.g., email@example.com).
func GetActiveUserKey() (string, error) {
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
	creds, _, err := GetActiveCredentials()
	if err != nil {
		return "", err
	}

	// If API hostname is explicitly stored, use it
	if creds.APIHostname != "" {
		return creds.APIHostname, nil
	}

	// Fall back to deriving from auth hostname
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
