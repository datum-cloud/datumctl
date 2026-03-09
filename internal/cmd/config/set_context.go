package config

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
)

func newSetContextCmd() *cobra.Command {
	var cluster string
	var user string
	var namespace string
	var projectID string
	var organizationID string

	cmd := &cobra.Command{
		Use:   "set-context NAME",
		Short: "Create or update a context",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireArgs(cmd, args, 1)
			if err != nil {
				return err
			}

			cfg, err := datumconfig.Load()
			if err != nil {
				return err
			}

			ctx := datumconfig.Context{
				Cluster:        cluster,
				User:           user,
				Namespace:      namespace,
				ProjectID:      projectID,
				OrganizationID: organizationID,
			}
			cfg.EnsureContextDefaults(&ctx)
			if err := cfg.ValidateContext(ctx); err != nil {
				return err
			}

			cfg.UpsertContext(datumconfig.NamedContext{
				Name:    name,
				Context: ctx,
			})

			if err := datumconfig.Save(cfg); err != nil {
				return err
			}

			cmd.Printf("Context %q set.\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&cluster, "cluster", "", "Cluster name")
	cmd.Flags().StringVar(&user, "user", "", "User name (from datumctl config users list)")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Namespace (defaults to 'default')")
	cmd.Flags().StringVar(&projectID, "project", "", "Project ID")
	cmd.Flags().StringVar(&organizationID, "organization", "", "Organization ID")
	cmd.MarkFlagRequired("cluster")
	cmd.MarkFlagsMutuallyExclusive("project", "organization")

	return cmd
}
