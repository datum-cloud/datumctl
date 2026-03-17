package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
)

func newUseContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use-context NAME",
		Short: "Set the current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireArgs(cmd, args, 1)
			if err != nil {
				return err
			}

			cfg, err := datumconfig.Load()
			if err != nil {
				return err
			}

			ctx, ok := cfg.ContextByName(name)
			if !ok {
				return fmt.Errorf("context %q not found", name)
			}

			cfg.CurrentContext = name
			if err := datumconfig.Save(cfg); err != nil {
				return err
			}

			if ctx.Context.Cluster != "" {
				if _, err := authutil.GetActiveUserKeyForCluster(ctx.Context.Cluster); err != nil {
					fmt.Printf("Warning: No credentials found for cluster %q. Run `datumctl auth login` for this cluster.\n", ctx.Context.Cluster)
				}
			}

			cmd.Printf("Switched to context %q.\n", name)
			return nil
		},
	}

	return cmd
}
