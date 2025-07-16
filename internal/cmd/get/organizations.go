package get

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"

	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/output"
)

func getOrganizationsCmd() *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "organizations",
		Short: "List organizations for the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			c, err := client.NewUserContextualClient(ctx)
			if err != nil {
				return err
			}

			// List organization memberships using the dynamic client
			result := &resourcemanagerv1alpha1.OrganizationMembershipList{}
			if err := c.List(ctx, result); err != nil {
				return fmt.Errorf("failed to list organization memberships: %w", err)
			}

			if len(result.Items) == 0 {
				fmt.Println("You are not a member of any organization.")
				return nil
			}

			// Use the new Kubernetes-native output library
			return output.CLIPrint(os.Stdout, outputFormat, result, func() (output.ColumnFormatter, output.RowFormatterFunc) {
				return output.ColumnFormatter{"DISPLAY NAME", "ORGANIZATION ID"}, func() output.RowFormatter {
					var rows output.RowFormatter
					for _, membership := range result.Items {
						// Extract organization information from the typed membership
						displayName := membership.Status.Organization.DisplayName
						if displayName == "" {
							displayName = "<Unknown>"
						}

						// Get the organization ID from spec.organizationRef.name
						orgID := membership.Spec.OrganizationRef.Name
						if orgID == "" {
							orgID = "<Unknown>"
						}

						rows = append(rows, []any{displayName, orgID})
					}
					return rows
				}
			})
		},
	}

	cmd.Flags().StringVar(&outputFormat, "output", "table", "Specify the output format to use. Supported options: table, json, yaml")

	return cmd
}
