package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/keyring"
)

func updateKubeconfigCmd() *cobra.Command {
	var kubeconfig, projectName, organizationName string

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

			// --- Get hostname from stored credentials ---
			activeUserKey, err := keyring.Get(authutil.ServiceName, authutil.ActiveUserKey)
			if err != nil {
				if errors.Is(err, keyring.ErrNotFound) {
					return errors.New("no active user found. Please login using 'datumctl auth login'")
				}
				return fmt.Errorf("failed to get active user from keyring: %w", err)
			}

			credsJSON, err := keyring.Get(authutil.ServiceName, activeUserKey)
			if err != nil {
				return fmt.Errorf("failed to get credentials for user '%s' from keyring: %w", activeUserKey, err)
			}

			var storedCreds authutil.StoredCredentials
			if err := json.Unmarshal([]byte(credsJSON), &storedCreds); err != nil {
				return fmt.Errorf("failed to parse stored credentials for user '%s': %w", activeUserKey, err)
			}

			authHostname := storedCreds.Hostname // Get auth hostname from credentials
			if authHostname == "" {
				return fmt.Errorf("hostname not found in stored credentials for user '%s'", activeUserKey)
			}

			// Derive API hostname from auth hostname (replace 'auth.' with 'api.')
			apiHostname := strings.Replace(authHostname, "auth.", "api.", 1)
			if apiHostname == authHostname { // Check if replacement occurred
				// Consider logging a warning or handling cases where 'auth.' prefix isn't present
				fmt.Printf("Warning: Could not derive API hostname from auth hostname '%s'. Using it directly.\n", authHostname)
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

			fmt.Printf("Successfully updated kubeconfig at %s for user %s (API Server: %s)\n", kubeconfigPath, activeUserKey, apiHostname) // Update print message
			return nil
		},
	}

	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
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
