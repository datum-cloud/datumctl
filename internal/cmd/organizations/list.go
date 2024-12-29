package organizations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/keyring"
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
	var hostname string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List organizations for the authenticated user",
		RunE: func(cmd *cobra.Command, _ []string) error {
			token, err := keyring.Get("datumctl", "datumctl")
			if err != nil {
				return fmt.Errorf("failed to get token from keyring: %w", err)
			}

			url := fmt.Sprintf("https://%s/query", hostname)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, strings.NewReader(getAllOrganizationsRequest))
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to make request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("unexpected status code %d from token endpoint", resp.StatusCode)
			}

			var r listResponse
			if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
				return fmt.Errorf("failed to decode JSON response: %w", err)
			}

			fmt.Printf("%-20s\t%-20s\n", "NAME", "RESOURCE ID")

			if len(r.Data.Organizations.Edges) == 0 {
				fmt.Printf("No organizations found")
			} else {
				for _, org := range r.Data.Organizations.Edges {
					fmt.Printf("%-20s\t%-20s\n", org.Node.Name, org.Node.UserEntityID)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&hostname, "hostname", "api.datum.net", "The hostname of the Datum Cloud instance to authenticate with")

	return cmd
}

const getAllOrganizationsRequest = `{
  "operationName": "GetAllOrganizations",
  "query": "query GetAllOrganizations {organizations {edges {node {name userEntityID}}}}"
}`
