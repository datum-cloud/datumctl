// internal/cmd/mcp/mcp.go
package mcp

import (
	"errors"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/client"
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
		Short: "Start a Model Context Protocol (MCP) server for Datum Cloud",
		Long: `Start a local MCP server that exposes Datum Cloud resource management
capabilities to AI agents and MCP-compatible clients (e.g., Claude).

Available tools:
  list_crds, get_crd                  Discover and inspect resource type schemas
  validate_yaml                       Validate manifests via server-side dry run
  create_resource, get_resource,      Generic CRUD for any Datum Cloud resource type
  update_resource, delete_resource,
  list_resources
  change_context                      Switch between organization and project contexts
                                      at runtime

Safety: all write operations default to dry-run mode. Pass dryRun: false
in the tool arguments to apply changes for real.

MCP clients connect over STDIO. Use --port to also expose a local HTTP
debug API on 127.0.0.1:<port> for testing tool calls with curl.

Exactly one of --organization or --project is required.`,
		Example: `  # Start MCP server targeting an organization
  datumctl mcp --organization my-org-id

  # Start MCP server targeting a specific project with a debug HTTP port
  datumctl mcp --project my-project-id --port 8080

  # Start MCP server with a default namespace for resource operations
  datumctl mcp --organization my-org-id --namespace default

  # Claude Desktop config (macOS) — add to mcpServers in claude_desktop_config.json:
  # {
  #   "datum_mcp": {
  #     "command": "/usr/local/bin/datumctl",
  #     "args": ["mcp", "--organization", "my-org-id", "--namespace", "default"]
  #   }
  # }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Exactly one of --organization or --project is required.
			if (organization == "") == (project == "") {
				return errors.New("exactly one of --organization or --project is required")
			}

			// Build *rest.Config from Datum context (no kubeconfig reliance).
			cfg, err := client.RestConfigForContext(cmd.Context(), organization, project)
			if err != nil {
				return err
			}

			cmd.Printf("MCP target: %s (org=%s project=%s)\n", cfg.Host, organization, project)

			// Construct the k8s client (no kubeconfig fallback) and set default namespace.
			k, err := client.NewK8sFromRESTConfig(cfg)
			if err != nil {
				return err
			}
			k.Namespace = namespace

			// Preflight: verify we can reach the API server for this context.
			if err := k.Preflight(cmd.Context()); err != nil {
				return err
			}

			if port > 0 {
				cmd.Printf("[datum-mcp] HTTP debug API will listen on 127.0.0.1:%d\n", port)
			}

			svc := serversvc.NewService(k)
			svc.RunSTDIO(port) // blocks; if --port > 0, also serves HTTP
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Run HTTP debug API on 127.0.0.1:<port> (MCP still uses STDIO)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Default namespace for CRUD/validation (if YAML or tool args omit it)")
	cmd.Flags().StringVar(&organization, "organization", "", "Organization ID to target (mutually exclusive with --project)")
	cmd.Flags().StringVar(&project, "project", "", "Project ID to target (mutually exclusive with --organization)")
	return cmd
}

