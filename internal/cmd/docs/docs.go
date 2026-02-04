package docs

import "github.com/spf13/cobra"

// Command returns the docs parent command with all subcommands.
func Command(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Documentation and API exploration commands",
		Long:  `Commands for exploring and browsing API documentation.`,
	}
	cmd.AddCommand(OpenAPICmd())

	genDoc := GenerateDocumentationCmd(root)
	cmd.AddCommand(genDoc)

	return cmd
}
