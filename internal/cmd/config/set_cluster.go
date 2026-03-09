package config

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/datumconfig"
)

func newSetClusterCmd() *cobra.Command {
	var server string
	var tlsServerName string
	var insecureSkipTLSVerify bool
	var caData string

	cmd := &cobra.Command{
		Use:   "set-cluster NAME",
		Short: "Create or update a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requireArgs(cmd, args, 1)
			if err != nil {
				return err
			}

			cfg, err := datumconfig.Load()
			if err != nil {
				return err
			}

			cluster := datumconfig.Cluster{
				Server:                   server,
				TLSServerName:            tlsServerName,
				InsecureSkipTLSVerify:    insecureSkipTLSVerify,
				CertificateAuthorityData: caData,
			}
			if err := cfg.ValidateCluster(cluster); err != nil {
				return err
			}

			cfg.UpsertCluster(datumconfig.NamedCluster{
				Name:    name,
				Cluster: cluster,
			})

			if err := datumconfig.Save(cfg); err != nil {
				return err
			}

			cmd.Printf("Cluster %q set.\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&server, "server", "", "API server base URL (e.g., https://api.example.com)")
	cmd.Flags().StringVar(&tlsServerName, "tls-server-name", "", "TLS server name override")
	cmd.Flags().BoolVar(&insecureSkipTLSVerify, "insecure-skip-tls-verify", false, "Skip TLS verification")
	cmd.Flags().StringVar(&caData, "certificate-authority-data", "", "Base64-encoded PEM certificate data")
	cmd.MarkFlagRequired("server")

	return cmd
}
