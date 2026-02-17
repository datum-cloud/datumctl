package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	kubectlcmd "k8s.io/kubectl/pkg/cmd"

	"github.com/coreos/go-oidc/v3/oidc" // OIDC discovery
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil" // Import new authutil package
	"go.datum.net/datumctl/internal/keyring"
)

const (
	stagingClientID = "360304563007327549" // Client ID for staging
	prodClientID    = "328728232771788043" // Client ID for prod
	redirectPath    = "/datumctl/auth/callback"
	// Listen on a random port
	listenAddr = "localhost:0"
)

var (
	hostname     string // Variable to store hostname flag
	apiHostname  string // Variable to store api-hostname flag
	clientIDFlag string // Variable to store client-id flag
	noBrowser    bool   // Variable to store no-browser flag
)

var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Datum Cloud via OAuth2 PKCE flow",
	RunE: func(cmd *cobra.Command, args []string) error {
		var actualClientID string
		if clientIDFlag != "" {
			actualClientID = clientIDFlag
		} else if strings.HasSuffix(hostname, ".staging.env.datum.net") {
			actualClientID = stagingClientID
		} else if strings.HasSuffix(hostname, ".datum.net") {
			actualClientID = prodClientID
		} else {
			// Return an error if no client ID could be determined
			return fmt.Errorf("client ID not configured for hostname '%s'. Please specify one with the --client-id flag", hostname)
		}
		return runLoginFlow(cmd.Context(), hostname, apiHostname, actualClientID, noBrowser, (kubectlcmd.GetLogVerbosity(os.Args) != "0"))
	},
}

func init() {
	// Add the hostname flag
	LoginCmd.Flags().StringVar(&hostname, "hostname", "auth.datum.net", "Hostname of the Datum Cloud authentication server")
	// Add the api-hostname flag
	LoginCmd.Flags().StringVar(&apiHostname, "api-hostname", "", "Hostname of the Datum Cloud API server (if not specified, will be derived from auth hostname)")
	// Add the client-id flag
	LoginCmd.Flags().StringVar(&clientIDFlag, "client-id", "", "Override the OAuth2 Client ID")
	// Add the no-browser flag
	LoginCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open a browser; use the device authorization flow")
}

// Generates a random PKCE code verifier
func generateCodeVerifier() (string, error) {
	const length = 64
	randomBytes := make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes), nil
}

// Generates the PKCE code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// generateRandomState generates a cryptographically random string for CSRF protection.
func generateRandomState(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// runLoginFlow now accepts context, hostname, apiHostname, clientID, and verbose flag
func runLoginFlow(ctx context.Context, authHostname string, apiHostname string, clientID string, noBrowser bool, verbose bool) error {
	fmt.Printf("Starting login process for %s ...\n", authHostname)

	// Determine the final API hostname to use
	var finalAPIHostname string
	if apiHostname != "" {
		// Use the explicitly provided API hostname
		finalAPIHostname = apiHostname
		fmt.Printf("Using specified API hostname: %s\n", finalAPIHostname)
	} else {
		// Derive API hostname from auth hostname
		derivedAPI, err := authutil.DeriveAPIHostname(authHostname)
		if err != nil {
			return fmt.Errorf("failed to derive API hostname from auth hostname '%s': %w", authHostname, err)
		}
		finalAPIHostname = derivedAPI
		fmt.Printf("Derived API hostname: %s\n", finalAPIHostname)
	}

	providerURL := fmt.Sprintf("https://%s", authHostname)
	provider, err := oidc.NewProvider(ctx, providerURL)
	if err != nil {
		return fmt.Errorf("failed to discover OIDC provider at %s: %w", providerURL, err)
	}

	// Define scopes
	scopes := []string{oidc.ScopeOpenID, "profile", "email", oidc.ScopeOfflineAccess}

	if noBrowser {
		token, err := runDeviceFlow(ctx, provider, providerURL, clientID, scopes)
		if err != nil {
			return err
		}
		return completeLogin(ctx, provider, clientID, authHostname, finalAPIHostname, scopes, token, verbose)
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}
	defer listener.Close()

	actualListenAddr := listener.Addr().String()

	conf := &oauth2.Config{
		ClientID:    clientID,
		Scopes:      scopes,
		Endpoint:    provider.Endpoint(),
		RedirectURL: fmt.Sprintf("http://%s%s", actualListenAddr, redirectPath),
	}

	// Generate PKCE parameters
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Generate random state
	state, err := generateRandomState(32)
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Construct the authorization URL
	authURL := conf.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)

	// Channel to receive the authorization code
	codeChan := make(chan string)
	errChan := make(chan error)
	serverClosed := make(chan struct{}) // To signal server shutdown completion

	// Start local server to handle the callback
	server := &http.Server{}
	mux := http.NewServeMux() // Use a mux to avoid conflicts if other handlers exist
	mux.HandleFunc(redirectPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		receivedState := r.URL.Query().Get("state")

		// Validate received state against the original state
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

		// Redirect to documentation site upon success
		http.Redirect(w, r, "https://docs.datum.net", http.StatusFound)

		codeChan <- code // Send code
		// Server shutdown will be initiated by the main goroutine now
	})
	server.Handler = mux

	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			// Don't send error if context is cancelled (which might happen on success)
			select {
			case <-ctx.Done():
				// Expected shutdown due to successful auth or cancellation
			default:
				errChan <- fmt.Errorf("failed to start callback server: %w", err)
			}
		}
	}()

	// Attempt to open browser
	fmt.Println("\nAttempting to open your default browser for authentication...")
	fmt.Printf("\nOpen this URL in your browser: %s\n", authURL)
	err = browser.OpenURL(authURL)
	if err != nil {
		fmt.Println("\nCould not open browser automatically.")
		fmt.Println("Please visit this URL manually to authenticate:")
		fmt.Printf("\n%s\n\n", authURL)
	} else {
		fmt.Println("Please complete the authentication in your browser.")
	}

	fmt.Println("\nWaiting for authentication callback...")

	var authCode string
	select {
	case code := <-codeChan:
		authCode = code
		// Initiate server shutdown *after* receiving the code
		go func() {
			if err := server.Shutdown(context.Background()); err != nil {
				// Log error if needed
			}
			close(serverClosed)
		}()
	case err := <-errChan:
		// Don't wait for serverClosed here if auth already failed
		return fmt.Errorf("authentication failed: %w", err)
	case <-ctx.Done():
		// If context is cancelled, still try to shut down gracefully
		go server.Shutdown(context.Background()) // Best effort
		// Don't necessarily wait for serverClosed here either
		return ctx.Err()
	}

	// Remove the blocking wait before exchange
	// <-serverClosed

	// Exchange code for token (now happens sooner)
	token, err := conf.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		// If exchange fails, wait for server shutdown before returning for cleaner exit
		<-serverClosed
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Wait for server shutdown *after* successful exchange (or failed exchange)
	<-serverClosed

	return completeLogin(ctx, provider, clientID, authHostname, finalAPIHostname, scopes, token, verbose)
}

func completeLogin(ctx context.Context, provider *oidc.Provider, clientID string, authHostname string, finalAPIHostname string, scopes []string, token *oauth2.Token, verbose bool) error {
	// Verify ID token and extract claims
	idTokenString, ok := token.Extra("id_token").(string)
	if !ok {
		return fmt.Errorf("id_token not found in token response")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: clientID}).Verify(ctx, idTokenString) // Use passed-in clientID
	if err != nil {
		return fmt.Errorf("failed to verify ID token: %w", err)
	}

	// Extract claims, including the subject ('sub')
	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return fmt.Errorf("failed to extract claims from ID token: %w", err)
	}

	// Ensure essential claims are present
	if claims.Subject == "" {
		return fmt.Errorf("could not extract subject (sub) claim from ID token")
	}
	if claims.Email == "" {
		return fmt.Errorf("could not extract email claim from ID token, which is required for user identification")
	}

	fmt.Printf("\nAuthenticated as: %s (%s)\n", claims.Name, claims.Email)

	// Use email directly as the key, as it already contains the hostname from the claim
	userKey := claims.Email

	creds := authutil.StoredCredentials{
		Hostname:         authHostname,
		APIHostname:      finalAPIHostname,
		ClientID:         clientID,
		EndpointAuthURL:  provider.Endpoint().AuthURL,
		EndpointTokenURL: provider.Endpoint().TokenURL,
		Scopes:           scopes,
		Token:            token,
		UserName:         claims.Name,    // Store name
		UserEmail:        claims.Email,   // Store email
		Subject:          claims.Subject, // Store subject (sub claim)
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	err = keyring.Set(authutil.ServiceName, userKey, string(credsJSON))
	if err != nil {
		return fmt.Errorf("failed to store credentials in keyring for user %s: %w", userKey, err)
	}

	activeUserKey := "" // Temp variable to check if active user was set
	err = keyring.Set(authutil.ServiceName, authutil.ActiveUserKey, userKey)
	if err != nil {
		fmt.Printf("Warning: Failed to set '%s' as active user in keyring: %v\n", userKey, err)
		fmt.Printf("Credentials for '%s' were stored successfully.\n", userKey)
	} else {
		// fmt.Printf("Credentials stored and set as active for user '%s'.\n", userKey) // Old message
		activeUserKey = userKey // Mark success
	}

	// Update confirmation messages
	if activeUserKey == userKey { // Check if we successfully set the active user
		fmt.Println("Authentication successful. Credentials stored and set as active.")
	} else {
		// This case handles if setting the active user key failed but creds were stored
		fmt.Println("Authentication successful. Credentials stored.")
	}

	// Update the list of known users (using the new key format)
	if err := addKnownUser(userKey); err != nil {
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

	return nil
}

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

func runDeviceFlow(ctx context.Context, provider *oidc.Provider, providerURL string, clientID string, scopes []string) (*oauth2.Token, error) {
	var discovery struct {
		DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	}
	if err := provider.Claims(&discovery); err != nil {
		return nil, fmt.Errorf("failed to read OIDC discovery document: %w", err)
	}

	deviceEndpoint := discovery.DeviceAuthorizationEndpoint
	if deviceEndpoint == "" {
		deviceEndpoint = strings.TrimRight(providerURL, "/") + "/oauth/v2/device_authorization"
	}

	deviceResp, err := requestDeviceAuthorization(ctx, deviceEndpoint, clientID, scopes)
	if err != nil {
		return nil, err
	}

	fmt.Println("\nTo authenticate, visit:")
	if deviceResp.VerificationURIComplete != "" {
		fmt.Printf("\n%s\n\n", deviceResp.VerificationURIComplete)
	} else {
		fmt.Printf("\n%s\n\n", deviceResp.VerificationURI)
	}
	if deviceResp.UserCode != "" {
		fmt.Printf("And enter code: %s\n\n", deviceResp.UserCode)
	}

	fmt.Println("Waiting for authorization...")

	tokenURL := provider.Endpoint().TokenURL
	token, err := pollDeviceToken(ctx, tokenURL, clientID, deviceResp.DeviceCode, deviceResp.Interval, deviceResp.ExpiresIn)
	if err != nil {
		return nil, err
	}

	return token, nil
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
			// Keep polling
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
		ExpiresIn:    tokenResp.ExpiresIn,
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

// addKnownUser adds a userKey (now email@hostname) to the known_users list in the keyring.
func addKnownUser(newUserKey string) error {
	knownUsers := []string{}

	// Get current list
	knownUsersJSON, err := keyring.Get(authutil.ServiceName, authutil.KnownUsersKey)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		// Only return error if it's not ErrNotFound
		return fmt.Errorf("failed to get known users list from keyring: %w", err)
	}

	if err == nil && knownUsersJSON != "" {
		if err := json.Unmarshal([]byte(knownUsersJSON), &knownUsers); err != nil {
			return fmt.Errorf("failed to unmarshal known users list: %w", err)
		}
	}

	// Check if user already exists
	found := false
	for _, key := range knownUsers {
		if key == newUserKey {
			found = true
			break
		}
	}

	// Add if not found
	if !found {
		knownUsers = append(knownUsers, newUserKey)

		// Marshal updated list
		updatedJSON, err := json.Marshal(knownUsers)
		if err != nil {
			return fmt.Errorf("failed to marshal updated known users list: %w", err)
		}

		// Store updated list
		if err := keyring.Set(authutil.ServiceName, authutil.KnownUsersKey, string(updatedJSON)); err != nil {
			return fmt.Errorf("failed to store updated known users list: %w", err)
		}
	}

	return nil
}
