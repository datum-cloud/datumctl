package organizations

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	iamrmv1alpha "buf.build/gen/go/datum-cloud/iam/protocolbuffers/go/datum/resourcemanager/v1alpha"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/output"
	"google.golang.org/protobuf/encoding/protojson"
)

func listOrgsCommand() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organizations for the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			// Get OIDC token source
			tknSrc, err := authutil.GetTokenSource(ctx)
			if err != nil {
				return fmt.Errorf("failed to get token source: %w", err)
			}

			// Use oauth2.NewClient with the token source
			httpClient := oauth2.NewClient(ctx, tknSrc)

			// Get active credentials to find the correct API hostname
			creds, _, err := authutil.GetActiveCredentials()
			if err != nil {
				return fmt.Errorf("failed to get active credentials: %w", err)
			}

			// Derive API hostname from the stored auth hostname
			apiHostname, err := authutil.DeriveAPIHostname(creds.Hostname)
			if err != nil {
				// Error during derivation, return the error directly
				return fmt.Errorf("failed to derive API hostname from stored credentials: %w", err)
			}

			url := fmt.Sprintf("https://%s/datum-os/iam/v1alpha/organizations:search", apiHostname)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader("{}"))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := httpClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to execute request to list organizations: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("failed to list organizations, status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			var searchRespProto iamrmv1alpha.SearchOrganizationsResponse
			// Using UnmarshalOptions to be more resilient, e.g. if API adds new fields not yet in client's proto.
			unmarshalOpts := protojson.UnmarshalOptions{
				DiscardUnknown: true,
			}
			if err := unmarshalOpts.Unmarshal(bodyBytes, &searchRespProto); err != nil {
				return fmt.Errorf("failed to decode organizations list response into protobuf: %w", err)
			}

			if err := output.CLIPrint(os.Stdout, outputFormat, &searchRespProto, func() (output.ColumnFormatter, output.RowFormatterFunc) {
				return output.ColumnFormatter{"DISPLAY NAME", "RESOURCE ID"}, func() output.RowFormatter {
					var rowData output.RowFormatter
					for _, org := range searchRespProto.GetOrganizations() {
						rowData = append(rowData, []any{org.GetDisplayName(), org.GetOrganizationId()})
					}
					return rowData
				}
			}); err != nil {
				return fmt.Errorf("a problem occured while printing organizations list: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&outputFormat, "output", "table", "Specify the output format to use. Supported options: table, json, yaml")

	return cmd
}
