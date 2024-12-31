package organizations

import (
	"fmt"
	"os"

	resourcemanagerv1alpha "buf.build/gen/go/datum-cloud/datum-os/protocolbuffers/go/datum/os/resourcemanager/v1alpha"
	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/output"
	"go.datum.net/datumctl/internal/resourcemanager"
)

func listOrgsCommand() *cobra.Command {
	var hostname, outputFormat string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organizations for the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, err := keyring.Get("datumctl", "datumctl")
			if err != nil {
				return fmt.Errorf("failed to get token from keyring: %w", err)
			}

			organizationsAPI := &resourcemanager.OrganizationsAPI{
				PAT:      token,
				Hostname: hostname,
			}

			listOrgs, err := organizationsAPI.ListOrganizations(cmd.Context(), &resourcemanagerv1alpha.ListOrganizationsRequest{})
			if err != nil {
				return fmt.Errorf("failed to list organizations: %w", err)
			}

			if err := output.CLIPrint(os.Stdout, outputFormat, listOrgs, func() (output.ColumnFormatter, output.RowFormatterFunc) {
				return output.ColumnFormatter{"DISPLAY NAME", "RESOURCE ID"}, func() output.RowFormatter {
					var rowData output.RowFormatter
					for _, org := range listOrgs.Organizations {
						rowData = append(rowData, []any{org.DisplayName, org.OrganizationId})
					}
					return rowData
				}
			}); err != nil {
				return fmt.Errorf("a problem occured while printing organizations list: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "api.datum.net", "The hostname of the Datum Cloud instance to authenticate with")
	cmd.Flags().StringVar(&outputFormat, "output", "table", "Specify the output format to use. Supported options: table, json, yaml")

	return cmd
}
