package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/plugin"
)

var m = plugin.Manifest{
	Name:          "dns",
	Version:       "v0.1.0",
	Description:   "Manage Datum Cloud DNS zones (reference plugin example)",
	APIVersion:    1,
	MinAPIVersion: 1,
}

func main() {
	// ServeManifest handles --plugin-manifest and exits before cobra runs.
	plugin.ServeManifest(m)

	root := plugin.NewRootCmd("dns", "Manage Datum Cloud DNS resources")

	zonesCmd := &cobra.Command{
		Use:   "zones",
		Short: "Manage DNS zones",
	}

	zonesListCmd := &cobra.Command{
		Use:   "list",
		Short: "List DNS zones",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := plugin.Context()

			if ctx.CredentialsHelper == "" {
				return fmt.Errorf("DATUM_CREDENTIALS_HELPER is not set; run this plugin via 'datumctl dns zones list'")
			}

			// Demonstrate token acquisition — the core of the end-to-end demo.
			token, err := plugin.Token()
			if err != nil {
				return fmt.Errorf("failed to get credentials: %w", err)
			}

			// Make a real API call using the injected context and fresh token.
			// In a production plugin this would call the Datum Cloud DNS API.
			apiURL := fmt.Sprintf("https://%s/v1/organizations/%s/projects/%s/dnszones",
				ctx.APIHost, ctx.Org, ctx.Project)

			req, err := http.NewRequestWithContext(cmd.Context(), http.MethodGet, apiURL, nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("API request failed: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API returned %d", resp.StatusCode)
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"DNS zones for org=%s project=%s (status %d)\n",
				ctx.Org, ctx.Project, resp.StatusCode)
			return nil
		},
	}

	zonesCmd.AddCommand(zonesListCmd)
	root.AddCommand(zonesCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
