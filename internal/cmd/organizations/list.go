package organizations

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/output"
	"go.datum.net/datumctl/internal/resourcemanager"
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

			// Create a new resource manager client
			rmClient := resourcemanager.NewClient(httpClient, apiHostname)

			// List organizations using the client
			searchRespProto, err := rmClient.ListOrganizations(ctx)
			if err != nil {
				return fmt.Errorf("failed to list organizations: %w", err)
			}

			if err := output.CLIPrint(os.Stdout, outputFormat, searchRespProto, func() (output.ColumnFormatter, output.RowFormatterFunc) {
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
