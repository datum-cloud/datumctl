package docs

import (
	"bufio"
	"fmt"
	"io"
	"os"
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
		datumctl docs generate-cli-docs --output-dir /tmp/datumctl-docs

		# Generate documentation into the docs output directory
		datumctl docs generate-cli-docs --output-dir ./site/content/cli`))

	generateDocLong = templates.LongDesc(i18n.T(`
		Generate a markdown file for every datumctl command and write them to
		the specified output directory.

		Each command produces one markdown file named after its full command path
		(e.g., datumctl_get.md). Files include front matter compatible with the
		Datum Cloud documentation site.

		The output directory must already exist before running this command.
		This command is primarily used by the Datum Cloud documentation pipeline
		to publish the CLI reference at datum.net/docs/datumctl.
		`))
)

func GenerateDocumentationCmd(root *cobra.Command) *cobra.Command {
	var (
		outputDir string
	)
	const fmTemplate = `---
title: "%s"
sidebar:
  hidden: true
---
`

	filePrepender := func(filename string) string {
		name := filepath.Base(filename)
		base := strings.TrimSuffix(name, path.Ext(name))
		return fmt.Sprintf(fmTemplate, strings.Replace(base, "_", " ", -1))
	}
	linkHandler := func(name string) string {
		base := strings.TrimSuffix(name, path.Ext(name))
		return "/docs/datumctl/command/" + strings.ToLower(base) + "/"
	}
	cmd := &cobra.Command{
		Use:     "generate-cli-docs",
		Short:   "Generate markdown reference documentation for all datumctl commands",
		Example: generateDocExample,
		Long:    generateDocLong,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := doc.GenMarkdownTreeCustom(root, outputDir, filePrepender, linkHandler); err != nil {
				return err
			}
			return downscaleMarkdownHeadersInDir(outputDir)
		},
	}
	cmd.Flags().StringVar(&outputDir, "output-dir", "/tmp/datumctl-generated-doc", "Directory to use to output the generated documentation")
	return cmd
}

func downscaleMarkdownHeadersInDir(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		return downscaleMarkdownHeadersInFile(path)
	})
}

func downscaleMarkdownHeadersInFile(filename string) error {
	isDatumctlDoc := filepath.Base(filename) == "datumctl.md"
	in, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(filename), ".md-transform-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(tmp)
	inFence := false
	var fenceMarker string
	dropNextH2Header := true

	for {
		line, err := reader.ReadString('\n')
		isEOF := err == io.EOF
		if err != nil && !isEOF {
			return err
		}

		trimmed := strings.TrimRight(line, "\r\n")
		fence := strings.TrimLeft(trimmed, " \t")
		if strings.HasPrefix(fence, "```") || strings.HasPrefix(fence, "~~~") {
			marker := fence[:3]
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if fenceMarker == marker {
				inFence = false
				fenceMarker = ""
			}
		} else if !inFence {
			if isDatumctlDoc {
				line = replaceDatumctlTitle(line)
			}
			if dropNextH2Header {
				var dropped bool
				line, dropped = dropFirstH2HeaderLine(line)
				if dropped {
					dropNextH2Header = false
					if isEOF {
						break
					}
					continue
				}
			}
			line = replaceH1WithH3(line)
			line = normalizeSeeAlsoLine(line)
			line = normalizeDatumctlCommandLinks(line)
		}

		if _, werr := writer.WriteString(line); werr != nil {
			return werr
		}

		if isEOF {
			break
		}
	}

	if err := writer.Flush(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filename)
}

func dropFirstH2HeaderLine(line string) (string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "## ") {
		return "", true
	}
	return line, false
}

func replaceH1WithH3(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if len(trimmed) == 0 || trimmed[0] != '#' {
		return line
	}

	hashes := 0
	for hashes < len(trimmed) && trimmed[hashes] == '#' {
		hashes++
	}
	if hashes != 1 {
		return line
	}
	if len(trimmed) <= hashes || trimmed[hashes] != ' ' {
		return line
	}

	prefixLen := len(line) - len(trimmed)
	return line[:prefixLen] + "###" + trimmed[hashes:]
}

func normalizeSeeAlsoLine(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "### SEE ALSO") {
		prefixLen := len(line) - len(trimmed)
		return line[:prefixLen] + "### See also" + trimmed[len("### SEE ALSO"):]
	}
	if strings.Contains(line, "[SEE ALSO](") {
		return strings.ReplaceAll(line, "[SEE ALSO](", "[See also](")
	}
	return line
}

func normalizeDatumctlCommandLinks(line string) string {
	return strings.ReplaceAll(line, "/docs/datumctl/command/datumctl/", "/docs/datumctl/cli-reference/")
}

func replaceDatumctlTitle(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "hidden: true" {
		return strings.Replace(line, "hidden: true", "hidden: false", 1)
	}
	if trimmed == `title: "datumctl"` {
		return strings.Replace(line, `title: "datumctl"`, `title: "CLI command reference"`, 1)
	}
	return line
}
