package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"go.datum.net/datumctl/internal/datum"

	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientauthv1 "k8s.io/client-go/pkg/apis/clientauthentication/v1"
)

func getTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-token",
		Short: "Retrieve access tokens for the Datum Cloud API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			tokenSource, err := datum.DefaultTokenSource(cmd.Context())
			if err != nil {
				return err
			}

			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return err
			} else if !slices.Contains([]string{"token", "client.authentication.k8s.io/v1"}, outputFormat) {
				return fmt.Errorf("invalid `--output` option provided")
			}

			token, err := tokenSource.Token()
			if err != nil {
				return err
			}

			if outputFormat == "token" {
				fmt.Print(token.AccessToken)
			} else if outputFormat == "client.authentication.k8s.io/v1" {
				execToken := clientauthv1.ExecCredential{
					TypeMeta: v1.TypeMeta{
						Kind:       "ExecCredential",
						APIVersion: clientauthv1.SchemeGroupVersion.Identifier(),
					},
					Status: &clientauthv1.ExecCredentialStatus{
						Token: token.AccessToken,
						ExpirationTimestamp: &v1.Time{
							Time: token.Expiry,
						},
					},
				}

				payload, err := json.MarshalIndent(execToken, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal exec credential: %w", err)
				}

				_, err = fmt.Fprintln(os.Stdout, string(payload))
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("output", "token", "Output format of the token. Supports 'token' or 'client.authentication.k8s.io/v1'.")

	return cmd
}
