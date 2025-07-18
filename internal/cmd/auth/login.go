package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc" // OIDC discovery
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil" // Import new authutil package
	"go.datum.net/datumctl/internal/keyring"
)

const (
	stagingClientID = "325848904128073754" // Client ID for staging
	prodClientID    = "328728232771788043" // Client ID for prod
	redirectPath    = "/datumctl/auth/callback"
	// Listen on a random port
	listenAddr = "localhost:0"
)

var (
	hostname     string // Variable to store hostname flag
	apiHostname  string // Variable to store api-hostname flag
	clientIDFlag string // Variable to store client-id flag
	verbose      bool   // Variable to store verbose flag
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
		return runLoginFlow(cmd.Context(), hostname, apiHostname, actualClientID, verbose)
	},
}

func init() {
	// Add the hostname flag
	LoginCmd.Flags().StringVar(&hostname, "hostname", "auth.datum.net", "Hostname of the Datum Cloud authentication server")
	// Add the api-hostname flag
	LoginCmd.Flags().StringVar(&apiHostname, "api-hostname", "", "Hostname of the Datum Cloud API server (if not specified, will be derived from auth hostname)")
	// Add the client-id flag
	LoginCmd.Flags().StringVar(&clientIDFlag, "client-id", "", "Override the OAuth2 Client ID")
	// Add the verbose flag
	LoginCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Print the full ID token claims after successful login")
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
func runLoginFlow(ctx context.Context, authHostname string, apiHostname string, clientID string, verbose bool) error {
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
			errChan <- fmt.Errorf(errMsg)
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
