package docs

import "github.com/spf13/cobra"

// Command returns the docs parent command with all subcommands.
func Command(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Explore API documentation and generate CLI reference docs",
		Long: `The docs group provides tools for discovering and exploring the Datum Cloud
API, as well as generating offline documentation for datumctl itself.

Subcommands:
  openapi               Launch a local Swagger UI to browse OpenAPI specs
                        for any API group available in the current context.
  generate-cli-docs     Generate markdown documentation files for all
                        datumctl commands (used to build the published
                        CLI reference at datum.net/docs).
  generate-man-pages    Generate man page files for all datumctl commands,
                        following the kubectl naming convention (e.g. datumctl-get.1).`,
		Example: `  # Browse platform-wide APIs in a local Swagger UI
  datumctl docs openapi

  # Generate CLI reference docs into a local directory
  datumctl docs generate-cli-docs --output-dir /tmp/datumctl-docs`,
	}
	cmd.AddCommand(OpenAPICmd())

	genDoc := GenerateDocumentationCmd(root)
	cmd.AddCommand(genDoc)

	cmd.AddCommand(GenerateManPagesCmd(root))

	return cmd
}
