package config

import (
	"os"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
)

func newGetContextsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get-contexts",
		Short:   "List contexts",
		Aliases: []string{"contexts", "gc"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := datumconfig.Load()
			if err != nil {
				return err
			}

			if len(cfg.Contexts) == 0 {
				cmd.Println("No contexts found.")
				return nil
			}

			tbl := table.New("Name", "Cluster", "User", "Namespace", "Project", "Organization", "Current").WithWriter(os.Stdout)
			for _, ctx := range cfg.Contexts {
				namespace := ctx.Context.Namespace
				if namespace == "" {
					namespace = datumconfig.DefaultNamespace
				}
				current := ""
				if ctx.Name == cfg.CurrentContext {
					current = "*"
				}
				tbl.AddRow(ctx.Name, ctx.Context.Cluster, ctx.Context.User, namespace, ctx.Context.ProjectID, ctx.Context.OrganizationID, current)
			}

			tbl.Print()
			return nil
		},
	}

	return cmd
}
