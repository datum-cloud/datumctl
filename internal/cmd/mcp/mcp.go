// internal/cmd/mcp/mcp.go
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/client-go/rest"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/kube"
	serversvc "go.datum.net/datumctl/internal/mcp"
)

func Command() *cobra.Command {
	var (
		port         int
		namespace    string
		organization string
		project      string
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the Datum MCP server",
		Long: `Start a local MCP server that exposes tools to list CRDs, inspect CRDs,
and validate manifests via server-side dry run. MCP clients (e.g., Claude) connect over STDIO.
Use --port to also expose a local HTTP debug API on 127.0.0.1:<port>.
Select a Datum context with exactly one of --organization or --project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Exactly one of --organization or --project is required.
			if (organization == "") == (project == "") {
				return errors.New("exactly one of --organization or --project is required")
			}

			// Build *rest.Config from Datum context (no kubeconfig reliance).
			cfg, err := restConfigFromFlags(cmd.Context(), organization, project)
			if err != nil {
				return err
			}

			cmd.Printf("MCP target: %s (org=%s project=%s)\n", cfg.Host, organization, project)

			// Use injected rest.Config for all kube ops.
			k := kube.NewWithRESTConfig(cfg)
			k.Namespace = namespace

			svc := serversvc.NewService(k)
			svc.RunSTDIO(port) // blocks; if --port > 0, also serves HTTP
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Run HTTP debug API on 127.0.0.1:<port> (MCP still uses STDIO)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Default namespace for validation (if YAML omits it)")
	cmd.Flags().StringVar(&organization, "organization", "", "Organization ID to target (mutually exclusive with --project)")
	cmd.Flags().StringVar(&project, "project", "", "Project ID to target (mutually exclusive with --organization)")
	return cmd
}

// restConfigFromFlags constructs a client-go *rest.Config using the same auth + host
// pattern as internal/client/user_context.go, but scoped to an org OR a project.
func restConfigFromFlags(ctx context.Context, organizationID, projectID string) (*rest.Config, error) {
	// OIDC token & API hostname from stored credentials
	tknSrc, err := authutil.GetTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token source: %w", err)
	}
	apiHostname, err := authutil.GetAPIHostname()
	if err != nil {
		return nil, fmt.Errorf("get API hostname: %w", err)
	}

	if (organizationID == "") == (projectID == "") {
		return nil, errors.New("exactly one of organizationID or projectID must be provided")
	}

	// Build the control-plane endpoint similar to user_context.go
	var host string
	if organizationID != "" {
		host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
			apiHostname, organizationID)
	} else {
		host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
			apiHostname, projectID)
	}

	return &rest.Config{
		Host: host,
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{Source: tknSrc, Base: rt}
		},
	}, nil
}
