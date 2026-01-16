package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
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
	Short: "Retrieve access token for active user (raw or K8s format)",
	Long: `Retrieves credentials for the currently active datumctl user.

Default behavior (--output=token) prints the raw access token to stdout.
With --output=client.authentication.k8s.io/v1, prints a K8s ExecCredential JSON object
suitable for use as a kubectl credential plugin.`, // Updated description
	Args: cobra.NoArgs,
	RunE: runGetToken, // Use single function
}

func init() {
	// Add flags for direct execution mode
	getTokenCmd.Flags().StringP("output", "o", outputFormatToken, fmt.Sprintf("Output format. One of: %s|%s", outputFormatToken, outputFormatK8sV1Creds))
}

// runGetToken implements the logic based on the --output flag.
func runGetToken(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	outputFormat, _ := cmd.Flags().GetString("output") // Ignore error, handled by validation

	if outputFormat != outputFormatToken && outputFormat != outputFormatK8sV1Creds {
		// Return error here so Cobra prints usage
		return fmt.Errorf("invalid --output format %q. Must be %s or %s", outputFormat, outputFormatToken, outputFormatK8sV1Creds)
	}

	// Get the token source (which handles refresh and persistence automatically)
	tokenSource, err := authutil.GetTokenSource(ctx)
	if err != nil {
		if errors.Is(err, authutil.ErrNoActiveUser) {
			return err
		}
		return fmt.Errorf("failed to get token source: %w", err)
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
