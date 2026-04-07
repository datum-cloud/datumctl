// Package authutil provides shared constants and functions for handling authentication
// credentials, including storage in the system keyring and OAuth2 token management.
package authutil

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
	"github.com/google/uuid"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/keyring"
	"golang.org/x/oauth2"
)

// MachineAccountCredentials is the on-disk JSON format downloaded from the Datum Cloud portal.
type MachineAccountCredentials struct {
	Type         string `json:"type"`           // "datum_machine_account"
	APIEndpoint  string `json:"api_endpoint"`   // "https://api.datum.net"
	TokenURI     string `json:"token_uri"`      // "https://auth.datum.net/oauth/v2/token"
	Scope        string `json:"scope"`          // OAuth2 scope string, e.g. "openid profile email urn:zitadel:..."
	ProjectID    string `json:"project_id"`
	ClientEmail  string `json:"client_email"`   // identity e-mail, used as display name
	ClientID     string `json:"client_id"`      // numeric Zitadel user ID (iss / sub)
	PrivateKeyID string `json:"private_key_id"` // kid header
	PrivateKey   string `json:"private_key"`    // PEM-encoded RSA private key
}

// tokenResponse is a minimal struct for parsing token endpoint responses in the
// JWT bearer exchange. It mirrors the fields we care about from deviceTokenResponse
// without creating a circular import with the auth command package.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}

// MintJWT mints a signed RS256 JWT suitable for the jwt-bearer grant.
// Claims: iss=clientID, sub=clientID, aud=issuer (scheme+host of tokenURI),
// kid=privateKeyID, jti=random UUID, iat=now, exp=now+60s.
func MintJWT(clientID, privateKeyID, privateKeyPEM, tokenURI string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block from private key")
	}

	var rsaKey *rsa.PrivateKey
	// Try PKCS#1 first, fall back to PKCS#8.
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		rsaKey = key
	} else {
		key8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse private key (tried PKCS#1 and PKCS#8): %w", err)
		}
		var ok bool
		rsaKey, ok = key8.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("private key is not an RSA key")
		}
	}

	// aud must be the issuer (scheme+host), not the full token endpoint URL.
	u, err := url.Parse(tokenURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse token URI: %w", err)
	}
	issuer := u.Scheme + "://" + u.Host

	jwk := jose.JSONWebKey{Key: rsaKey, KeyID: privateKeyID}

	sig, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.RS256, Key: jwk},
		(&jose.SignerOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT signer: %w", err)
	}

	now := time.Now()
	signed, err := josejwt.Signed(sig).
		Claims(josejwt.Claims{
			Issuer:   clientID,
			Subject:  clientID,
			Audience: josejwt.Audience{issuer},
			IssuedAt: josejwt.NewNumericDate(now),
			Expiry:   josejwt.NewNumericDate(now.Add(60 * time.Second)),
			ID:       uuid.NewString(),
		}).
		Serialize()
	if err != nil {
		return "", fmt.Errorf("failed to serialize JWT: %w", err)
	}

	return signed, nil
}

// ExchangeJWT POSTs a signed JWT to tokenURI using the jwt-bearer grant and
// returns the resulting oauth2.Token. The token will have no RefreshToken.
// If scope is empty, "openid profile email" is used as the default.
func ExchangeJWT(ctx context.Context, tokenURI, signedJWT, scope string) (*oauth2.Token, error) {
	if scope == "" {
		scope = "openid profile email"
	}
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	form.Set("assertion", signedJWT)
	form.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURI, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT bearer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("JWT bearer token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWT bearer response: %w", err)
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return nil, fmt.Errorf("failed to parse JWT bearer response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if tr.Error != "" {
			return nil, fmt.Errorf("JWT bearer exchange failed: %s (%s)", tr.Error, tr.ErrorDesc)
		}
		return nil, fmt.Errorf("JWT bearer exchange failed with status %s", resp.Status)
	}

	token := &oauth2.Token{
		AccessToken: tr.AccessToken,
		TokenType:   tr.TokenType,
	}
	if tr.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}

	return token, nil
}

// machineAccountTokenSource implements oauth2.TokenSource for machine account sessions.
// It re-mints a JWT and re-exchanges it whenever the stored access token has expired,
// since machine account sessions have no refresh token.
type machineAccountTokenSource struct {
	ctx     context.Context
	creds   *StoredCredentials
	userKey string
	mu      sync.Mutex
}

// Token implements oauth2.TokenSource. If the cached token is still valid it is
// returned immediately. Otherwise a new JWT is minted, exchanged for an access
// token, and the updated credentials are persisted to the keyring.
func (m *machineAccountTokenSource) Token() (*oauth2.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.creds.Token != nil && m.creds.Token.Valid() {
		return m.creds.Token, nil
	}

	ma := m.creds.MachineAccount
	signedJWT, err := MintJWT(ma.ClientID, ma.PrivateKeyID, ma.PrivateKey, ma.TokenURI)
	if err != nil {
		return nil, customerrors.WrapUserErrorWithHint(
			"Failed to mint JWT for machine account authentication.",
			"Please re-authenticate using: `datumctl auth login --credentials <file>`",
			err,
		)
	}

	token, err := ExchangeJWT(m.ctx, ma.TokenURI, signedJWT, ma.Scope)
	if err != nil {
		return nil, customerrors.WrapUserErrorWithHint(
			"Failed to exchange JWT for access token.",
			"Please re-authenticate using: `datumctl auth login --credentials <file>`",
			err,
		)
	}

	m.creds.Token = token

	credsJSON, err := json.Marshal(m.creds)
	if err != nil {
		// Return token even if persistence fails — the caller can still proceed.
		return token, fmt.Errorf("failed to marshal updated machine account credentials: %w", err)
	}

	if err := keyring.Set(ServiceName, m.userKey, string(credsJSON)); err != nil {
		return token, fmt.Errorf("failed to persist refreshed machine account token to keyring: %w", err)
	}

	return token, nil
}
