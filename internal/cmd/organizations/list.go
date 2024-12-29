package organizations

import (
	"fmt"

	resourcemanagerv1alpha "buf.build/gen/go/datum-cloud/datum-os/protocolbuffers/go/datum/os/resourcemanager/v1alpha"
	"buf.build/go/protoyaml"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"go.datum.net/datumctl/internal/keyring"
	"go.datum.net/datumctl/internal/resourcemanager"
)

type listResponse struct {
	Data struct {
		Organizations struct {
			Edges []struct {
				Node struct {
					Name         string `json:"name"`
					UserEntityID string `json:"userEntityID"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"organizations"`
	} `json:"data"`
}

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

			// TODO: We should look at abstracting the formatting here into a library
			//       that can be used by multiple commands needing to offer multiple
			//       output formats from a command.
			switch outputFormat {
			case "yaml":
				marshaller := &protoyaml.MarshalOptions{
					Indent: 2,
				}
				output, err := marshaller.Marshal(listOrgs)
				if err != nil {
					return fmt.Errorf("failed to list organizations: %w", err)
				}
				fmt.Print(string(output))
			case "json":
				output, err := protojson.Marshal(listOrgs)
				if err != nil {
					return fmt.Errorf("failed to list organizations: %w", err)
				}
				fmt.Print(string(output))
			case "table":
				orgTable := table.New("DISPLAY NAME", "RESOURCE ID")
				if len(listOrgs.Organizations) == 0 {
					fmt.Printf("No organizations found")
				} else {
					for _, org := range listOrgs.Organizations {
						orgTable.AddRow(org.DisplayName, org.OrganizationId)
					}
				}
				orgTable.Print()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "api.datum.net", "The hostname of the Datum Cloud instance to authenticate with")
	cmd.Flags().StringVar(&outputFormat, "output", "table", "Specify the output format to use. Supported options: table, json, yaml")

	return cmd
}
