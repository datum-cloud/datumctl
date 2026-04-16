package authutil

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"

	"go.datum.net/datumctl/internal/keyring"
)

const defaultMachineAccountScope = "openid profile email offline_access"

// RunMachineAccountLogin reads a machine account credentials file, discovers
// the token endpoint via OIDC, mints a JWT, exchanges it for an access token,
// and stores the resulting session in the keyring. Returns a LoginResult so the
// caller can build a v1beta1 Session.
func RunMachineAccountLogin(ctx context.Context, credentialsPath, hostname, apiHostname string, debug bool) (*LoginResult, error) {
	data, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file %q: %w", credentialsPath, err)
	}

	var creds MachineAccountCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file %q: %w", credentialsPath, err)
	}

	if creds.Type != "datum_machine_account" {
		return nil, fmt.Errorf("unsupported credentials type %q: expected \"datum_machine_account\"", creds.Type)
	}

	var missing []string
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
		return nil, fmt.Errorf("credentials file is missing required fields: %s", strings.Join(missing, ", "))
	}

	providerURL := fmt.Sprintf("https://%s", hostname)
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider at %s: %w (pass --hostname to point datumctl at your Datum Cloud auth server)", providerURL, err)
	}
	tokenURI := provider.Endpoint().TokenURL

	scope := creds.Scope
	if scope == "" {
		scope = defaultMachineAccountScope
	}

	finalAPIHostname := apiHostname
	if finalAPIHostname == "" {
		derived, err := DeriveAPIHostname(hostname)
		if err != nil {
			return nil, fmt.Errorf("failed to derive API hostname from auth hostname %q: %w", hostname, err)
		}
		finalAPIHostname = derived
	}

	signedJWT, err := MintJWT(creds.ClientID, creds.PrivateKeyID, creds.PrivateKey, tokenURI)
	if err != nil {
		return nil, fmt.Errorf("failed to mint JWT: %w", err)
	}

	if debug {
		parts := strings.SplitN(signedJWT, ".", 3)
		if len(parts) == 3 {
			hdr, _ := base64.RawURLEncoding.DecodeString(parts[0])
			claims, _ := base64.RawURLEncoding.DecodeString(parts[1])
			fmt.Fprintf(os.Stderr, "\n--- JWT header ---\n%s\n", hdr)
			fmt.Fprintf(os.Stderr, "--- JWT claims ---\n%s\n", claims)
		}
		fmt.Fprintf(os.Stderr, "\n--- Token request ---\nPOST %s\nassertion=%s...\n", tokenURI, signedJWT[:40])
	}

	token, err := ExchangeJWT(ctx, tokenURI, signedJWT, scope)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange JWT for access token: %w", err)
	}

	displayName := creds.ClientEmail
	if displayName == "" {
		displayName = creds.ClientID
	}

	userKey := creds.ClientEmail
	if userKey == "" {
		userKey = creds.ClientID
	}

	keyFilePath, err := WriteMachineAccountKeyFile(userKey, creds.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to write machine account private key to disk: %w", err)
	}

	stored := StoredCredentials{
		Hostname:         hostname,
		APIHostname:      finalAPIHostname,
		ClientID:         creds.ClientID,
		EndpointTokenURL: tokenURI,
		Token:            token,
		UserName:         displayName,
		UserEmail:        creds.ClientEmail,
		Subject:          creds.ClientID,
		CredentialType:   "machine_account",
		MachineAccount: &MachineAccountState{
			ClientEmail:    creds.ClientEmail,
			ClientID:       creds.ClientID,
			PrivateKeyID:   creds.PrivateKeyID,
			PrivateKeyPath: keyFilePath,
			TokenURI:       tokenURI,
			Scope:          scope,
		},
	}

	credsJSON, err := json.Marshal(stored)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize credentials: %w", err)
	}

	if err := keyring.Set(ServiceName, userKey, string(credsJSON)); err != nil {
		if cleanupErr := RemoveMachineAccountKeyFile(userKey); cleanupErr != nil {
			fmt.Printf("Warning: failed to remove machine account key file after keyring error for %s: %v\n", userKey, cleanupErr)
		}
		return nil, fmt.Errorf("failed to store credentials in keyring for %s: %w", userKey, err)
	}

	if err := keyring.Set(ServiceName, ActiveUserKey, userKey); err != nil {
		fmt.Printf("Warning: Failed to set %q as active user in keyring: %v\n", userKey, err)
	}

	if err := AddKnownUserKey(userKey); err != nil {
		fmt.Printf("Warning: Failed to update list of known users: %v\n", err)
	}

	return &LoginResult{
		UserKey:     userKey,
		UserEmail:   creds.ClientEmail,
		UserName:    displayName,
		Subject:     creds.ClientID,
		APIHostname: finalAPIHostname,
	}, nil
}
