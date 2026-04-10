package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/keyring"
)

// defaultMachineAccountScope is used when the credentials file does not
// specify a scope. The file's scope field is still honored for backward
// compatibility; new credentials files should omit it.
const defaultMachineAccountScope = "openid profile email offline_access"

// runMachineAccountLogin handles the --credentials flag path for `datumctl auth login`.
// It reads a machine account credentials file, discovers the token endpoint via OIDC
// well-known config, mints a JWT, exchanges it for an initial access token, and stores
// the resulting session in the keyring.
//
// hostname is the auth server hostname (e.g., "auth.datum.net"), taken from the --hostname
// flag. apiHostname is the API server hostname; when empty, it is derived from hostname
// using authutil.DeriveAPIHostname.
func runMachineAccountLogin(ctx context.Context, credentialsPath, hostname, apiHostname string, debug bool) error {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return fmt.Errorf("failed to read credentials file %q: %w", credentialsPath, err)
	}

	var creds authutil.MachineAccountCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("failed to parse credentials file %q: %w", credentialsPath, err)
	}

	// Validate type field.
	if creds.Type != "datum_machine_account" {
		return fmt.Errorf("unsupported credentials type %q: expected \"datum_machine_account\"", creds.Type)
	}

	// Validate only the fields that cannot be discovered or derived.
	missing := []string{}
	if creds.ClientID == "" {
		missing = append(missing, "client_id")
	}
	if creds.PrivateKeyID == "" {
		missing = append(missing, "private_key_id")
	}
	if creds.PrivateKey == "" {
		missing = append(missing, "private_key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("credentials file is missing required fields: %s", strings.Join(missing, ", "))
	}

	// Discover the token endpoint from the OIDC provider's well-known config.
	// This mirrors the pattern used by the interactive login flow in login.go.
	providerURL := fmt.Sprintf("https://%s", hostname)
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return fmt.Errorf("failed to discover OIDC provider at %s: %w (pass --hostname to point datumctl at your Datum Cloud auth server)", providerURL, err)
	}
	tokenURI := provider.Endpoint().TokenURL

	// Resolve the scope to use. Honor the file's scope for backward compatibility;
	// otherwise fall back to the default that mirrors the interactive login flow.
	scope := creds.Scope
	if scope == "" {
		scope = defaultMachineAccountScope
	}

	// Resolve the API hostname. Use the flag value when provided; otherwise derive
	// it from the auth hostname using the same logic as the interactive login flow.
	finalAPIHostname := apiHostname
	if finalAPIHostname == "" {
		derived, err := authutil.DeriveAPIHostname(hostname)
		if err != nil {
			return fmt.Errorf("failed to derive API hostname from auth hostname %q: %w", hostname, err)
		}
		finalAPIHostname = derived
	}

	// Mint the initial JWT assertion using the discovered token URI.
	signedJWT, err := authutil.MintJWT(creds.ClientID, creds.PrivateKeyID, creds.PrivateKey, tokenURI)
	if err != nil {
		return fmt.Errorf("failed to mint JWT: %w", err)
	}

	if debug {
		// Print JWT parts so the caller can inspect claims at jwt.io
		parts := strings.SplitN(signedJWT, ".", 3)
		if len(parts) == 3 {
			hdr, _ := base64.RawURLEncoding.DecodeString(parts[0])
			claims, _ := base64.RawURLEncoding.DecodeString(parts[1])
			fmt.Fprintf(os.Stderr, "\n--- JWT header ---\n%s\n", hdr)
			fmt.Fprintf(os.Stderr, "--- JWT claims ---\n%s\n", claims)
		}
		fmt.Fprintf(os.Stderr, "\n--- Token request ---\nPOST %s\nassertion=%s...\n", tokenURI, signedJWT[:40])
	}

	// Exchange for an access token using the discovered token URI.
	token, err := authutil.ExchangeJWT(ctx, tokenURI, signedJWT, scope)
	if err != nil {
		return fmt.Errorf("failed to exchange JWT for access token: %w", err)
	}

	// Determine the display name. Prefer client_email if present; fall back to client_id.
	displayName := creds.ClientEmail
	if displayName == "" {
		displayName = creds.ClientID
	}

	// Use client_email as the keyring key when available; fall back to client_id.
	userKey := creds.ClientEmail
	if userKey == "" {
		userKey = creds.ClientID
	}

	// Write the PEM private key to disk to keep the keyring blob small.
	// On macOS the Keychain has a per-item size limit (~4 KB); embedding the
	// PEM (~2.5 KB) alongside the access token pushes the blob over the limit.
	keyFilePath, err := authutil.WriteMachineAccountKeyFile(userKey, creds.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to write machine account private key to disk: %w", err)
	}

	stored := authutil.StoredCredentials{
		Hostname:         hostname,
		APIHostname:      finalAPIHostname,
		ClientID:         creds.ClientID,
		EndpointTokenURL: tokenURI,
		Token:            token,
		UserName:         displayName,
		UserEmail:        creds.ClientEmail,
		Subject:          creds.ClientID,
		CredentialType:   "machine_account",
		MachineAccount: &authutil.MachineAccountState{
			ClientEmail:  creds.ClientEmail,
			ClientID:     creds.ClientID,
			PrivateKeyID: creds.PrivateKeyID,
			// PrivateKey is intentionally left empty; the key lives on disk at
			// PrivateKeyPath so the keyring blob stays under the macOS size limit.
			PrivateKeyPath: keyFilePath,
			// Store the discovered token URI and resolved scope so that the
			// machineAccountTokenSource can refresh tokens without re-reading
			// the credentials file.
			TokenURI: tokenURI,
			Scope:    scope,
		},
	}

	credsJSON, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	if err := keyring.Set(authutil.ServiceName, userKey, string(credsJSON)); err != nil {
		return fmt.Errorf("failed to store credentials in keyring for %s: %w", userKey, err)
	}

	if err := keyring.Set(authutil.ServiceName, authutil.ActiveUserKey, userKey); err != nil {
		fmt.Printf("Warning: Failed to set %q as active user in keyring: %v\n", userKey, err)
	}

	if err := addKnownUser(userKey); err != nil {
		fmt.Printf("Warning: Failed to update list of known users: %v\n", err)
	}

	fmt.Printf("Authenticated as machine account: %s\n", displayName)
	return nil
}
