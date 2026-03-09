package config

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
	"sigs.k8s.io/yaml"
)

func newViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Display the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := datumconfig.Load()
			if err != nil {
				return err
			}
			cfgData, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}
			cmd.Print(string(cfgData))
			return nil
		},
	}

	return cmd
}
