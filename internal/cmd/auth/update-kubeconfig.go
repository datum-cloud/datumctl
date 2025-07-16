package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"go.datum.net/datumctl/internal/authutil"
)

func updateKubeconfigCmd() *cobra.Command {
	var kubeconfig, projectName, organizationName, hostname string

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

			var apiHostname string
			var activeUserKey string

			// Use override hostname if provided, otherwise get from stored credentials
			if hostname != "" {
				apiHostname = hostname
			} else {
				var err error
				apiHostname, err = authutil.GetAPIHostname()
				if err != nil {
					return fmt.Errorf("failed to get API hostname: %w", err)
				}

				activeUserKey, err = authutil.GetActiveUserKey()
				if err != nil {
					// We only expect an error here if the user is not logged in.
					if errors.Is(err, authutil.ErrNoActiveUser) {
						return errors.New("no active user found. Please login using 'datumctl auth login'")
					}
					// For other errors, provide more context.
					return fmt.Errorf("failed to get active user for kubeconfig message: %w", err)
				}
			}

			// --- Get executable path ---
			executablePath, err := os.Executable()
			if err != nil {
				// Log warning and fallback, or return error? Returning error is safer.
				return fmt.Errorf("failed to determine datumctl executable path: %w", err)
			}
			// --- End Get executable path ---

			var path string
			if projectName != "" {
				path = "/apis/resourcemanager.miloapis.com/v1alpha1/projects/" + projectName + "/control-plane"
			} else {
				path = "/apis/resourcemanager.miloapis.com/v1alpha1/organizations/" + organizationName + "/control-plane"
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
				Server: fmt.Sprintf("https://%s%s", apiHostname, path), // Use derived API hostname
			}

			cfg.Contexts[clusterName] = &api.Context{
				Cluster:  clusterName,
				AuthInfo: "datum-user",
			}
			cfg.CurrentContext = clusterName
			cfg.AuthInfos["datum-user"] = &api.AuthInfo{
				Exec: &api.ExecConfig{
					InstallHint: execPluginInstallHint,
					Command:     executablePath, // Use absolute path
					Args: []string{
						"auth",
						"get-token",
						"--output=client.authentication.k8s.io/v1",
					},
					APIVersion:         "client.authentication.k8s.io/v1",
					ProvideClusterInfo: false,
					InteractiveMode:    "IfAvailable",
				},
			}

			// Save changes back to file
			if err := clientcmd.WriteToFile(*cfg, kubeconfigPath); err != nil {
				return fmt.Errorf("failed to write updated kubeconfig: %v", err)
			}

			// Construct success message
			userInfo := activeUserKey
			if userInfo == "" {
				userInfo = "custom hostname override"
			}
			fmt.Printf("Successfully updated kubeconfig at %s for user %s (API Server: %s)\n", kubeconfigPath, userInfo, apiHostname)
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	cmd.Flags().StringVar(&projectName, "project", "", "Configure kubectl to access a specific project's control plane instead of the core control plane.")
	cmd.Flags().StringVar(&organizationName, "organization", "", "The organization name that is being connected to.")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Override the hostname for the API server")

	cmd.MarkFlagsOneRequired("project", "organization")
	cmd.MarkFlagsMutuallyExclusive("project", "organization")
	return cmd
}

const execPluginInstallHint = `
The datumctl command is required to authenticate to the current cluster. It can
be installed by running the following command.

go install go.datum.net/datumctl@latest
`
