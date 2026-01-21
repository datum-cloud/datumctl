package docs

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	generateDocExample = templates.Examples(i18n.T(`
		# Generate documentation into a temporary directory
		datumctl generate-cli-docs -o /tmp/commands-doc`))

	generateDocLong = templates.Examples(i18n.T(`
		Generate documentation from each command metadata.
		The directory where you place the documentation (-o) should exists.

		Each command turns into its correspondine markdown file.
		`))
)

func GenerateDocumentationCmd(root *cobra.Command) *cobra.Command {
	var (
		outputDir string
	)
	const fmTemplate = `---
title: "%s"
---
`

	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
	}
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/docs/datumctl/commands/" + strings.ToLower(base) + "/"
	}
	cmd := &cobra.Command{
		Use:     "generate-cli-docs",
		Short:   "Generate markdown documentation",
		Example: generateDocExample,
		Long:    generateDocLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := doc.GenMarkdownTreeCustom(root, outputDir, filePrepender, linkHandler)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outputDir, "output-dir", "/tmp/datumctl-generated-doc", "Directory to use to output the generated documentation")
	return cmd
}
