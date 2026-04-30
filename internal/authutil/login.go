package authutil

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// LoginResult holds the output of a successful login flow.
type LoginResult struct {
	UserKey     string
	UserEmail   string
	UserName    string
	Subject     string
	APIHostname string
}

// BuildSession constructs a v1beta1 Session from a login result.
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

// RunInteractiveLogin executes an interactive OAuth2 login flow (PKCE or device) and stores
// credentials in the keyring. It returns the login result on success.
//
// When noBrowser is true the device authorization flow is used, which does not
// require a browser on the local machine. When false the PKCE flow is used and
// the user's default browser is opened automatically.
func RunInteractiveLogin(ctx context.Context, authHostname, apiHostname, clientID string, noBrowser, verbose bool) (*LoginResult, error) {
	fmt.Printf("Starting login process for %s ...\n", authHostname)

	var finalAPIHostname string
	if apiHostname != "" {
		finalAPIHostname = apiHostname
		fmt.Printf("Using specified API hostname: %s\n", finalAPIHostname)
	} else {
		derived, err := DeriveAPIHostname(authHostname)
		if err != nil {
			return nil, fmt.Errorf("failed to derive API hostname from '%s': %w", authHostname, err)
		}
		finalAPIHostname = derived
		fmt.Printf("Derived API hostname: %s\n", finalAPIHostname)
	}

	providerURL := fmt.Sprintf("https://%s", authHostname)
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider at %s: %w", providerURL, err)
	}

	scopes := []string{oidc.ScopeOpenID, "profile", "email", oidc.ScopeOfflineAccess}

	var token *oauth2.Token
	if noBrowser {
		token, err = runDeviceFlow(ctx, providerURL, clientID, scopes)
		if err != nil {
			return nil, err
		}
	} else {
		token, err = runPKCEFlow(ctx, provider, clientID, scopes)
		if err != nil {
			return nil, err
		}
	}

	return completeLogin(ctx, provider, clientID, authHostname, finalAPIHostname, scopes, token, verbose)
}

// runPKCEFlow executes the OAuth2 PKCE authorization code flow and returns the
// resulting token. It starts a local HTTP server to receive the callback.
func runPKCEFlow(ctx context.Context, provider *oidc.Provider, clientID string, scopes []string) (*oauth2.Token, error) {
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
			errChan <- fmt.Errorf("invalid state parameter received (expected %q, got %q)", state, receivedState)
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

	fmt.Println("\nAttempting to open your default browser for authentication...")
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Println("\nCould not open browser automatically.")
		fmt.Println("Please visit this URL manually to authenticate:")
		fmt.Printf("\n%s\n\n", authURL)
		fmt.Println("Tip: in a headless environment (CI, SSH without forwarding, or a container) use")
		fmt.Println("'datumctl auth login --no-browser' — it uses a device-code flow that doesn't need a local browser.")
	} else {
		fmt.Println("Please complete the authentication in your browser.")
	}

	fmt.Println("\nWaiting for authentication callback...")

	var authCode string
	select {
	case code := <-codeChan:
		authCode = code
		go func() {
			if err := server.Shutdown(context.Background()); err != nil {
				// Best-effort shutdown; ignore error.
			}
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

	return token, nil
}

// completeLogin verifies the ID token, stores credentials in the keyring, sets
// the active user, registers the user in the known-users list, and returns a
// LoginResult.
func completeLogin(ctx context.Context, provider *oidc.Provider, clientID, authHostname, finalAPIHostname string, scopes []string, token *oauth2.Token, verbose bool) (*LoginResult, error) {
	idTokenString, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("id_token not found in token response")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: clientID}).Verify(ctx, idTokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to extract claims from ID token: %w", err)
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("could not extract subject (sub) claim from ID token")
	}
	if claims.Email == "" {
		return nil, fmt.Errorf("could not extract email claim from ID token, which is required for user identification")
	}

	fmt.Printf("\nAuthenticated as: %s (%s)\n", claims.Name, claims.Email)

	// Use email as the keyring key for interactive logins, matching the
	// behaviour of the original cmd/auth/login.go implementation.
	userKey := claims.Email

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
		return nil, fmt.Errorf("failed to store credentials in keyring for user %s: %w", userKey, err)
	}

	activeUserSet := false
	if err := keyring.Set(ServiceName, ActiveUserKey, userKey); err != nil {
		fmt.Printf("Warning: Failed to set '%s' as active user in keyring: %v\n", userKey, err)
		fmt.Printf("Credentials for '%s' were stored successfully.\n", userKey)
	} else {
		activeUserSet = true
	}

	if activeUserSet {
		fmt.Println("Authentication successful. Credentials stored and set as active.")
	} else {
		fmt.Println("Authentication successful. Credentials stored.")
	}

	if err := AddKnownUserKey(userKey); err != nil {
		fmt.Printf("Warning: Failed to update list of known users: %v\n", err)
	}

	if verbose {
		var rawClaims map[string]interface{}
		if err := idToken.Claims(&rawClaims); err == nil {
			claimsJSON, err := json.MarshalIndent(rawClaims, "", "  ")
			if err != nil {
				fmt.Printf("Warning: Failed to marshal claims to JSON: %v\n", err)
			} else {
				fmt.Println("\n--- ID Token Claims (Verbose) ---")
				fmt.Println(string(claimsJSON))
				fmt.Println("---------------------------------")
			}
		} else {
			fmt.Printf("Warning: Failed to extract raw claims map: %v\n", err)
		}
	}

	return &LoginResult{
		UserKey:     userKey,
		UserEmail:   claims.Email,
		UserName:    claims.Name,
		Subject:     claims.Subject,
		APIHostname: finalAPIHostname,
	}, nil
}

// ---- Device flow ----

type deviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

type deviceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

type oauthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func runDeviceFlow(ctx context.Context, providerURL string, clientID string, scopes []string) (*oauth2.Token, error) {
	base := strings.TrimRight(providerURL, "/")
	deviceEndpoint := base + "/oauth/v2/device_authorization"
	tokenURL := base + "/oauth/v2/token"

	deviceResp, err := requestDeviceAuthorization(ctx, deviceEndpoint, clientID, scopes)
	if err != nil {
		return nil, err
	}

	// Force the v2 UI login path regardless of what the server returns.
	verificationURI := base + "/ui/v2/login/device"
	verificationURIComplete := verificationURI
	if deviceResp.UserCode != "" {
		verificationURIComplete += "?user_code=" + url.QueryEscape(deviceResp.UserCode)
	}

	fmt.Println("\nTo authenticate, visit:")
	fmt.Printf("\n%s\n\n", verificationURIComplete)
	if deviceResp.UserCode != "" {
		fmt.Printf("And enter code: %s\n\n", deviceResp.UserCode)
	}

	fmt.Println("Waiting for authorization...")

	return pollDeviceToken(ctx, tokenURL, clientID, deviceResp.DeviceCode, deviceResp.Interval, deviceResp.ExpiresIn)
}

func requestDeviceAuthorization(ctx context.Context, endpoint string, clientID string, scopes []string) (*deviceAuthorizationResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", strings.Join(scopes, " "))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device authorization request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device authorization request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read device authorization response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oauthErr oauthErrorResponse
		_ = json.Unmarshal(body, &oauthErr)
		if oauthErr.Error != "" {
			return nil, fmt.Errorf("device authorization failed: %s (%s)", oauthErr.Error, oauthErr.ErrorDescription)
		}
		return nil, fmt.Errorf("device authorization failed with status %s", resp.Status)
	}

	var deviceResp deviceAuthorizationResponse
	if err := json.Unmarshal(body, &deviceResp); err != nil {
		return nil, fmt.Errorf("failed to parse device authorization response: %w", err)
	}
	if deviceResp.DeviceCode == "" {
		return nil, fmt.Errorf("device authorization response missing device_code")
	}

	return &deviceResp, nil
}

func pollDeviceToken(ctx context.Context, tokenURL string, clientID string, deviceCode string, intervalSeconds int64, expiresIn int64) (*oauth2.Token, error) {
	interval := time.Duration(intervalSeconds) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}

	var deadline time.Time
	if expiresIn > 0 {
		deadline = time.Now().Add(time.Duration(expiresIn) * time.Second)
	}

	for {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return nil, fmt.Errorf("device authorization expired before completion")
		}

		token, errType, err := requestDeviceToken(ctx, tokenURL, clientID, deviceCode)
		if err != nil {
			return nil, err
		}
		switch errType {
		case "":
			return token, nil
		case "authorization_pending":
			// Keep polling.
		case "slow_down":
			interval += 5 * time.Second
		case "access_denied":
			return nil, fmt.Errorf("device authorization denied by user")
		case "expired_token":
			return nil, fmt.Errorf("device authorization expired")
		default:
			return nil, fmt.Errorf("device authorization failed: %s", errType)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func requestDeviceToken(ctx context.Context, tokenURL string, clientID string, deviceCode string) (*oauth2.Token, string, error) {
	form := url.Values{}
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
	form.Set("device_code", deviceCode)
	form.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create device token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("device token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read device token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var oauthErr oauthErrorResponse
		_ = json.Unmarshal(body, &oauthErr)
		if oauthErr.Error != "" {
			return nil, oauthErr.Error, nil
		}
		return nil, "", fmt.Errorf("device token request failed with status %s", resp.Status)
	}

	var tokenResp deviceTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, "", fmt.Errorf("failed to parse device token response: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
	}
	if tokenResp.ExpiresIn > 0 {
		token.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}
	token = token.WithExtra(map[string]any{
		"id_token": tokenResp.IDToken,
		"scope":    tokenResp.Scope,
	})

	return token, "", nil
}

// ---- Crypto helpers ----

func generateCodeVerifier() (string, error) {
	const length = 64
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
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
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
