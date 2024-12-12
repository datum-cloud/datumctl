package auth

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func updateKubeconfigCmd() *cobra.Command {
	var kubeconfig, baseURL, projectName string

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

			serverURL, err := url.Parse(baseURL)
			if err != nil {
				return fmt.Errorf("failed to parse base URL option: %w", err)
			}

			if projectName != "" {
				serverURL.Path = "/control-plane/v1alpha/projects/" + projectName
			}

			// Load existing config
			cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
			if errors.Is(err, os.ErrNotExist) {
				cfg = api.NewConfig()
			} else if err != nil {
				return fmt.Errorf("unable to load kubeconfig from %s: %v", kubeconfigPath, err)
			}

			clusterName := "datum-cloud"
			if projectName != "" {
				clusterName += "-project-" + projectName
			}

			cfg.Clusters[clusterName] = &api.Cluster{
				Server: serverURL.String(),
			}

			cfg.Contexts[clusterName] = &api.Context{
				Cluster:  clusterName,
				AuthInfo: "datum-cloud-user",
			}
			cfg.CurrentContext = clusterName
			cfg.AuthInfos["datum-cloud-user"] = &api.AuthInfo{
				Exec: &api.ExecConfig{
					InstallHint:        execPluginInstallHint,
					Command:            "datumctl",
					Args:               []string{"auth", "get-token", "--format=client.authentication.k8s.io/v1"},
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
	cmd.Flags().StringVar(&baseURL, "base-url", "https://api.datum.net", "The base URL of the Datum Cloud API")
	cmd.Flags().StringVar(&projectName, "project", "", "Configure kubectl to access a specific project's control plane instead of the core control plane.")
	return cmd
}

const execPluginInstallHint = `
The datumctl command is required to authenticate to the current cluster. It can
be installed by running the following command.

go install go.datum.net/datumctl@latest
`
