package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/keyring"
)

// runMachineAccountLogin handles the --credentials flag path for `datumctl auth login`.
// It reads a machine account credentials file, mints a JWT, exchanges it for an
// initial access token, and stores the resulting session in the keyring.
func runMachineAccountLogin(ctx context.Context, credentialsPath string, debug bool) error {
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

	// Validate all required fields are present.
	missing := []string{}
	if creds.TokenURI == "" {
		missing = append(missing, "token_uri")
	}
	if creds.ClientEmail == "" {
		missing = append(missing, "client_email")
	}
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
		return fmt.Errorf("credentials file is missing required fields: %v", missing)
	}

	// Mint the initial JWT assertion.
	signedJWT, err := authutil.MintJWT(creds.ClientID, creds.PrivateKeyID, creds.PrivateKey, creds.TokenURI)
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
		fmt.Fprintf(os.Stderr, "\n--- Token request ---\nPOST %s\nassertion=%s...\n", creds.TokenURI, signedJWT[:40])
	}

	// Exchange for an access token.
	token, err := authutil.ExchangeJWT(ctx, creds.TokenURI, signedJWT, creds.Scope)
	if err != nil {
		return fmt.Errorf("failed to exchange JWT for access token: %w", err)
	}

	// Derive auth hostname from token_uri (e.g. "auth.datum.net").
	tokenURIParsed, err := url.Parse(creds.TokenURI)
	if err != nil {
		return fmt.Errorf("failed to parse token_uri %q: %w", creds.TokenURI, err)
	}
	authHostname := tokenURIParsed.Host

	// Derive api hostname from api_endpoint (e.g. "api.datum.net").
	var apiHostname string
	if creds.APIEndpoint != "" {
		apiEndpointParsed, err := url.Parse(creds.APIEndpoint)
		if err != nil {
			return fmt.Errorf("failed to parse api_endpoint %q: %w", creds.APIEndpoint, err)
		}
		apiHostname = apiEndpointParsed.Host
	}

	stored := authutil.StoredCredentials{
		Hostname:         authHostname,
		APIHostname:      apiHostname,
		ClientID:         creds.ClientID,
		EndpointTokenURL: creds.TokenURI,
		Token:            token,
		UserName:         creds.ClientEmail,
		UserEmail:        creds.ClientEmail,
		Subject:          creds.ClientID,
		CredentialType:   "machine_account",
		MachineAccount: &authutil.MachineAccountState{
			ClientEmail:  creds.ClientEmail,
			ClientID:     creds.ClientID,
			PrivateKeyID: creds.PrivateKeyID,
			PrivateKey:   creds.PrivateKey,
			TokenURI:     creds.TokenURI,
			Scope:        creds.Scope,
		},
	}

	userKey := creds.ClientEmail

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

	fmt.Printf("Authenticated as machine account: %s\n", creds.ClientEmail)
	return nil
}
