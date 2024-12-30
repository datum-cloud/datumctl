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

			outTableHeaders, outTableData := getListOrganizationsTableOutputData(listOrgs)
			output.CLIPrint(os.Stdout, outputFormat, listOrgs, outTableHeaders, outTableData)

			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "api.datum.net", "The hostname of the Datum Cloud instance to authenticate with")
	cmd.Flags().StringVar(&outputFormat, "output", "table", "Specify the output format to use. Supported options: table, json, yaml")

	return cmd
}

func getListOrganizationsTableOutputData(listOrgs *resourcemanagerv1alpha.ListOrganizationsResponse) ([]any, [][]any) {
	headers := []any{"DISPLAY NAME", "RESOURCE ID"}
	var rowData [][]any
	for _, org := range listOrgs.Organizations {
		rowData = append(rowData, []any{org.DisplayName, org.OrganizationId})
	}
	return headers, rowData
}
