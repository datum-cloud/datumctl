package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func updateKubeconfigCmd() *cobra.Command {
	var kubeconfig, hostname, projectName, organizationName string

	cmd := &cobra.Command{
		Use:   "update-kubeconfig",
		Short: "Update the kubeconfig file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine kubeconfig path
			var kubeconfigPath string
			if kubeconfig != "" {
				kubeconfigPath = kubeconfig
			} else if envKC := os.Getenv("KUBECONFIG"); envKC != "" {
				kubeconfigPath = envKC
			} else {
				kubeconfigPath = clientcmd.RecommendedHomeFile
			}

			var path string
			if projectName != "" {
				path = "/apis/resourcemanager.datumapis.com/v1alpha/projects/" + projectName + "/control-plane"
			} else {
				path = "/apis/resourcemanager.datumapis.com/v1alpha/organizations/" + organizationName + "/control-plane"
			}

			// Load existing config
			cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
			if errors.Is(err, os.ErrNotExist) {
				cfg = api.NewConfig()
			} else if err != nil {
				return fmt.Errorf("unable to load kubeconfig from %s: %v", kubeconfigPath, err)
			}

			clusterName := "datum"
			if projectName != "" {
				clusterName += "-project-" + projectName
			} else {
				clusterName += "-organization-" + organizationName
			}

			cfg.Clusters[clusterName] = &api.Cluster{
				Server: fmt.Sprintf("https://%s%s", hostname, path),
			}

			cfg.Contexts[clusterName] = &api.Context{
				Cluster:  clusterName,
				AuthInfo: "datum-user",
			}
			cfg.CurrentContext = clusterName
			cfg.AuthInfos["datum-user"] = &api.AuthInfo{
				Exec: &api.ExecConfig{
					InstallHint: execPluginInstallHint,
					Command:     "datumctl",
					Args: []string{
						"auth",
						"get-token",
						fmt.Sprintf("--hostname=%s", hostname),
						"--output=client.authentication.k8s.io/v1",
					},
					APIVersion:         "client.authentication.k8s.io/v1",
					ProvideClusterInfo: true,
					InteractiveMode:    "IfAvailable",
				},
			}

			// Save changes back to file
			if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
				return fmt.Errorf("failed to write updated kubeconfig: %v", err)
			}

			fmt.Printf("Successfully updated kubeconfig at %s\n", kubeconfigPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	cmd.Flags().StringVar(&hostname, "hostname", "api.datum.net", "The hostname of the Datum Cloud instance to authenticate with")
	cmd.Flags().StringVar(&projectName, "project", "", "Configure kubectl to access a specific project's control plane instead of the core control plane.")
	cmd.Flags().StringVar(&organizationName, "organization", "", "The organization name that is being connected to.")

	cmd.MarkFlagsOneRequired("project", "organization")
	cmd.MarkFlagsMutuallyExclusive("project", "organization")
	return cmd
}

const execPluginInstallHint = `
The datumctl command is required to authenticate to the current cluster. It can
be installed by running the following command.

go install go.datum.net/datumctl@latest
`
