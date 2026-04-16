package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

// Supported output formats
const (
	outputFormatToken      = "token"
	outputFormatK8sV1Creds = "client.authentication.k8s.io/v1"
)

// getTokenCmd retrieves tokens based on the --output flag.
var getTokenCmd = &cobra.Command{
	Use:   "get-token",
	Short: "Print the active user's access token (advanced / kubectl integration)",
	Long: `Print the current access token for the active Datum Cloud user.

Most datumctl users do not need this command — datumctl handles
authentication automatically for all its own commands.

This command exists for two advanced use cases:

  1. kubectl credential plugin: invoked automatically by kubectl after you
     run 'datumctl auth update-kubeconfig'. You do not need to call it
     directly in that case.

  2. Scripting or direct API calls: use --output=token to get a raw bearer
     token to pass to curl or other HTTP clients.

If the stored token is expired, datumctl automatically uses the stored
refresh token to obtain a new one before printing.

Output formats (--output / -o):
  token                         Print the raw access token (default).
  client.authentication.k8s.io/v1  Print a Kubernetes ExecCredential JSON
                                object for kubectl credential plugin use.`,
	Example: `  # Get a raw token for use in a script or direct API call
  datumctl auth get-token

  # Get a Kubernetes ExecCredential JSON object (used by kubectl automatically)
  datumctl auth get-token --output=client.authentication.k8s.io/v1`,
	Args: cobra.NoArgs,
	RunE: runGetToken, // Use single function
}

func init() {
	// Add flags for direct execution mode
	getTokenCmd.Flags().StringP("output", "o", outputFormatToken, fmt.Sprintf("Output format. One of: %s|%s", outputFormatToken, outputFormatK8sV1Creds))
	getTokenCmd.Flags().String("session", "", "Look up a specific session by name (defaults to the active session). Used by the kubectl exec plugin path so each kubeconfig entry pins to its own datumctl session.")
}

// runGetToken implements the logic based on the --output flag.
func runGetToken(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	outputFormat, _ := cmd.Flags().GetString("output") // Ignore error, handled by validation
	sessionName, _ := cmd.Flags().GetString("session")

	if outputFormat != outputFormatToken && outputFormat != outputFormatK8sV1Creds {
		// Return error here so Cobra prints usage
		return fmt.Errorf("invalid --output format %q. Must be %s or %s", outputFormat, outputFormatToken, outputFormatK8sV1Creds)
	}

	var tokenSource oauth2.TokenSource
	if sessionName != "" {
		userKey, err := authutil.GetUserKeyForSession(sessionName)
		if err != nil {
			return err
		}
		tokenSource, err = authutil.GetTokenSourceForUser(ctx, userKey)
		if err != nil {
			return fmt.Errorf("failed to get token source: %w", err)
		}
	} else {
		var err error
		tokenSource, err = authutil.GetTokenSource(ctx)
		if err != nil {
			if errors.Is(err, authutil.ErrNoActiveUser) {
				return errors.New("no active user found in keyring. Please login first using 'datumctl auth login'")
			}
			return fmt.Errorf("failed to get token source: %w", err)
		}
	}

	// Get fresh token (will refresh if needed and persist automatically)
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// --- Output based on requested format ---
	if outputFormat == outputFormatToken {
		// Output raw Access Token
		fmt.Print(newToken.AccessToken)
	} else if outputFormat == outputFormatK8sV1Creds {
		// Output K8s ExecCredential JSON
		outputToken := newToken.AccessToken // Default to AccessToken
		idToken, ok := newToken.Extra("id_token").(string)
		if ok && idToken != "" {
			outputToken = idToken // Prefer ID Token for K8s
		}

		expiry := metav1.Time{Time: newToken.Expiry}
		if newToken.Expiry.IsZero() {
			expiry = metav1.Time{Time: time.Now().Add(5 * time.Minute)}
		}

		responseCred := clientauthv1.ExecCredential{
			TypeMeta: metav1.TypeMeta{
				APIVersion: clientauthv1.SchemeGroupVersion.String(),
				Kind:       "ExecCredential",
			},
			Status: &clientauthv1.ExecCredentialStatus{
				ExpirationTimestamp: &expiry,
				Token:               outputToken,
			},
		}

		responseBytes, err := json.MarshalIndent(responseCred, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal response ExecCredential: %w", err)
		}

		_, err = os.Stdout.Write(responseBytes)
		if err != nil {
			return fmt.Errorf("failed to write ExecCredential JSON to stdout: %w", err)
		}
	}
	// Note: Invalid outputFormat handled at the beginning

	return nil
}
