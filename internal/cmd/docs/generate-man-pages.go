package docs

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"
)

var (
	generateManPagesExample = templates.Examples(i18n.T(`
		# Generate man pages into a temporary directory
		datumctl docs generate-man-pages --output-dir /tmp/datumctl-man

		# Generate man pages into the system man directory
		datumctl docs generate-man-pages --output-dir /usr/local/share/man/man1`))

	generateManPagesLong = templates.LongDesc(i18n.T(`
		Generate a man page for every datumctl command and write them to
		the specified output directory.

		Each command produces one file named after its full command path using
		hyphens as separators (e.g., datumctl-get.1), which follows the same
		convention used by kubectl.

		The output directory must already exist before running this command.
		Install the generated pages somewhere on your MANPATH to use them with
		the man(1) command.
		`))
)

func GenerateManPagesCmd(root *cobra.Command) *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:     "generate-man-pages",
		Short:   "Generate man page reference documentation for all datumctl commands",
		Example: generateManPagesExample,
		Long:    generateManPagesLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			header := &doc.GenManHeader{
				Title:   "DATUMCTL",
				Section: "1",
				Source:  "Datum Cloud",
				Manual:  "Datum Cloud Manual",
			}
			return doc.GenManTreeFromOpts(root, doc.GenManTreeOptions{
				Header:           header,
				Path:             outputDir,
				CommandSeparator: "-",
			})
		},
	}
	cmd.Flags().StringVar(&outputDir, "output-dir", "/tmp/datumctl-man", "Directory to write the generated man pages into")
	return cmd
}
