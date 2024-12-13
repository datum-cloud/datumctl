package datum

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

func DefaultTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	var creds *credentialsFile
	var err error

	// Check if the user specified the location through the well-known environment
	// variable.
	if filename := os.Getenv("DATUM_CREDENTIALS"); filename != "" {
		creds, err = readCredentialsFile(ctx, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read credentials file: %w", err)
		}
	}

	// Check if it's in the well-known file location.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	wellKnownFile := path.Join(homeDir, ".config", "datum", "application_credentials.json")
	if _, err := os.Stat(wellKnownFile); err == nil {
		creds, err = readCredentialsFile(ctx, wellKnownFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read application credentials from home directory: %w", err)
		}
	}

	if creds == nil {
		return nil, fmt.Errorf("could not find default application credentials")
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: &wrappedRoundTripper{
			Transport: http.DefaultTransport,
			AdditionalFields: map[string]string{
				"client_id": "jwt-client",
				"audience":  "datum-api",
			},
		},
	})

	var tokenSource oauth2.TokenSource

	if len(creds.PrivateKey) != 0 {
		tokenConfig := jwt.Config{
			PrivateKey: []byte(creds.PrivateKey),
			Email:      creds.ServiceAccountEmail,
			Subject:    creds.ServiceAccountEmail,
			TokenURL:   creds.TokenURL,
			Audience:   creds.TokenURL,
		}
		tokenSource = tokenConfig.TokenSource(ctx)
	} else {
		return nil, fmt.Errorf("invalid application credentials file found")
	}

	return tokenSource, nil
}

func readCredentialsFile(_ context.Context, filename string) (*credentialsFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	credentials := &credentialsFile{}
	return credentials, json.NewDecoder(file).Decode(credentials)
}

type credentialsFile struct {
	TokenURL string `json:"token_url"`

	PrivateKey          string `json:"private_key"`
	ServiceAccountEmail string `json:"service_account_email"`
}
