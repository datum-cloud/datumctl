package cmd

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/client"
	apiresources "go.datum.net/datumctl/internal/cmd/api-resources"
	"go.datum.net/datumctl/internal/cmd/auth"
	"go.datum.net/datumctl/internal/cmd/get"
	"go.datum.net/datumctl/internal/cmd/mcp"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "datumctl",
		Short: "A CLI for interacting with the Datum platform",
	}

	ioStreams := genericclioptions.IOStreams{
		In:     rootCmd.InOrStdin(),
		Out:    rootCmd.OutOrStdout(),
		ErrOut: rootCmd.ErrOrStderr(),
	}

	ctx := rootCmd.Context()
	config, err := client.NewRestConfig(ctx)
	if err != nil {
		panic(err)
	}
	var projectID string
	var organizationID string

	factory, err := client.NewDatumFactory(rootCmd.Context(), config)
	if err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "project id")
	rootCmd.PersistentFlags().StringVar(&organizationID, "organization-id", "", "org id")
	factory.ConfigFlags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(auth.Command())
	rootCmd.AddCommand(get.Command(factory, ioStreams, &projectID, &organizationID))
	rootCmd.AddCommand(apiresources.Command(factory, ioStreams))
	rootCmd.AddCommand(apiresources.CommandApiResources(factory, ioStreams))
	rootCmd.AddCommand(mcp.Command())
	return rootCmd
}
