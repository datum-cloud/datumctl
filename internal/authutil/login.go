package authutil

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/browser"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/keyring"
	"golang.org/x/oauth2"
)

const (
	StagingClientID = "325848904128073754"
	ProdClientID    = "328728232771788043"
	redirectPath    = "/datumctl/auth/callback"
	listenAddr      = "localhost:0"
)

// LoginResult holds the output of a successful PKCE login flow.
type LoginResult struct {
	UserKey     string
	UserEmail   string
	UserName    string
	Subject     string
	APIHostname string
}

// BuildSession constructs a v1beta1 Session from a PKCE login result.
// The caller is responsible for upserting it into the config and saving.
func BuildSession(result *LoginResult, authHostname string) datumconfig.Session {
	apiHostname := result.APIHostname
	return datumconfig.Session{
		Name:      datumconfig.SessionName(result.UserEmail, apiHostname),
		UserKey:   result.UserKey,
		UserEmail: result.UserEmail,
		UserName:  result.UserName,
		Endpoint: datumconfig.Endpoint{
			Server:       datumconfig.CleanBaseServer(datumconfig.EnsureScheme(apiHostname)),
			AuthHostname: authHostname,
		},
	}
}

// ResolveClientID determines the OAuth2 client ID for a given auth hostname.
func ResolveClientID(clientIDFlag, authHostname string) (string, error) {
	if clientIDFlag != "" {
		return clientIDFlag, nil
	}
	if strings.HasSuffix(authHostname, ".staging.env.datum.net") {
		return StagingClientID, nil
	}
	if strings.HasSuffix(authHostname, ".datum.net") {
		return ProdClientID, nil
	}
	return "", fmt.Errorf("client ID not configured for hostname '%s'. Please specify one with the --client-id flag", authHostname)
}

// RunPKCELogin executes the OAuth2 PKCE flow and stores credentials in the
// keyring. It returns the login result on success.
func RunPKCELogin(ctx context.Context, authHostname, apiHostname, clientID string) (*LoginResult, error) {
	fmt.Printf("Opening browser for authentication...\n")

	var finalAPIHostname string
	if apiHostname != "" {
		finalAPIHostname = apiHostname
	} else {
		var err error
		finalAPIHostname, err = DeriveAPIHostname(authHostname)
		if err != nil {
			return nil, fmt.Errorf("failed to derive API hostname from '%s': %w", authHostname, err)
		}
	}

	providerURL := fmt.Sprintf("https://%s", authHostname)
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider at %s: %w", providerURL, err)
	}

	scopes := []string{oidc.ScopeOpenID, "profile", "email", oidc.ScopeOfflineAccess}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}
	defer listener.Close()

	actualListenAddr := listener.Addr().String()

	conf := &oauth2.Config{
		ClientID:    clientID,
		Scopes:      scopes,
		Endpoint:    provider.Endpoint(),
		RedirectURL: fmt.Sprintf("http://%s%s", actualListenAddr, redirectPath),
	}

	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	state, err := generateRandomState(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}

	authURL := conf.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)

	codeChan := make(chan string)
	errChan := make(chan error)
	serverClosed := make(chan struct{})

	server := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		receivedState := r.URL.Query().Get("state")

		if receivedState != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state parameter received")
			return
		}

		if code == "" {
			errMsg := r.URL.Query().Get("error_description")
			errorType := r.URL.Query().Get("error")
			if errMsg == "" {
				if errorType != "" {
					errMsg = fmt.Sprintf("Authorization failed: %s", errorType)
				} else {
					errMsg = "Authorization code not found in callback request."
				}
			}
			http.Error(w, errMsg, http.StatusBadRequest)
			errChan <- fmt.Errorf("%s", errMsg)
			return
		}

		http.Redirect(w, r, "https://www.datum.net/docs/datumctl/cli-reference/#see-also", http.StatusFound)
		codeChan <- code
	})
	server.Handler = mux

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			select {
			case <-ctx.Done():
			default:
				errChan <- fmt.Errorf("failed to start callback server: %w", err)
			}
		}
	}()

	if err := browser.OpenURL(authURL); err != nil {
		fmt.Printf("\nOpen this URL in your browser:\n\n  %s\n\n", authURL)
	} else {
		fmt.Println("\nBrowser opened. Please complete the authentication.")
	}

	fmt.Print("\nWaiting for authentication callback...\n\n")

	var authCode string
	select {
	case code := <-codeChan:
		authCode = code
		go func() {
			_ = server.Shutdown(context.Background())
			close(serverClosed)
		}()
	case err := <-errChan:
		return nil, fmt.Errorf("authentication failed: %w", err)
	case <-ctx.Done():
		go server.Shutdown(context.Background())
		return nil, ctx.Err()
	}

	token, err := conf.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		<-serverClosed
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	<-serverClosed

	idTokenString, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("id_token not found in token response")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: clientID}).Verify(ctx, idTokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims: %w", err)
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("could not extract subject claim from ID token")
	}
	if claims.Email == "" {
		return nil, fmt.Errorf("could not extract email claim from ID token")
	}

	userKey := fmt.Sprintf("%s@%s", claims.Subject, authHostname)

	creds := StoredCredentials{
		Hostname:         authHostname,
		APIHostname:      finalAPIHostname,
		ClientID:         clientID,
		EndpointAuthURL:  provider.Endpoint().AuthURL,
		EndpointTokenURL: provider.Endpoint().TokenURL,
		Scopes:           scopes,
		Token:            token,
		UserName:         claims.Name,
		UserEmail:        claims.Email,
		Subject:          claims.Subject,
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize credentials: %w", err)
	}

	if err := keyring.Set(ServiceName, userKey, string(credsJSON)); err != nil {
		return nil, fmt.Errorf("failed to store credentials in keyring: %w", err)
	}

	if err := AddKnownUserKey(userKey); err != nil {
		fmt.Printf("Warning: Failed to update known users: %v\n", err)
	}

	return &LoginResult{
		UserKey:     userKey,
		UserEmail:   claims.Email,
		UserName:    claims.Name,
		Subject:     claims.Subject,
		APIHostname: finalAPIHostname,
	}, nil
}

func generateCodeVerifier() (string, error) {
	const length = 64
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

func generateRandomState(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
