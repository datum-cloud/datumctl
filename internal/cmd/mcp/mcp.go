// internal/cmd/mcp/mcp.go
package mcp

import (
	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/kube"
	serversvc "go.datum.net/datumctl/internal/mcp"
)

func Command() *cobra.Command {
	var (
		port        int
		kubeContext string
		namespace   string
		kubectlPath string
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the Datum MCP server (kubectl-backed)",
		Long: `Start a local MCP server that exposes tools to list CRDs, inspect CRDs,
and validate manifests via kubectl server-side dry run. MCP clients (e.g., Claude) connect over STDIO.
Use --port to also expose a local HTTP debug API on 127.0.0.1:<port>.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			k := kube.New()
			if kubectlPath != "" {
				k.Path = kubectlPath
			}
			k.Context = kubeContext
			k.Namespace = namespace

			svc := serversvc.NewService(k)
			svc.RunSTDIO(port) // blocks; if --port > 0, also serves HTTP
			return nil
		},
	}

	cmd.Flags().IntVar(&port, "port", 0, "Run HTTP debug API on 127.0.0.1:<port> (MCP still uses STDIO)")
	cmd.Flags().StringVar(&kubeContext, "kube-context", "", "Kube context to use (defaults to current)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Default namespace for validation (if YAML omits it)")
	cmd.Flags().StringVar(&kubectlPath, "kubectl", "kubectl", "Path to kubectl binary")
	return cmd
}
