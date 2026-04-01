// internal/cmd/create/create.go
package create

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	kubeCreate "k8s.io/kubectl/pkg/cmd/create"
	"k8s.io/kubectl/pkg/cmd/util"
)

func NewCmdCreate(f util.Factory, ioStreams genericiooptions.IOStreams) *cobra.Command {
	cmd := kubeCreate.NewCmdCreate(f, ioStreams)
	// create registeres sucommands like secrets, pod an so on. We do not need them since everything is a CRD
	for _, subCmd := range cmd.Commands() {
		cmd.RemoveCommand(subCmd)
	}
	cmd.Short = "Create a Datum Cloud resource from a file or stdin"
	cmd.Long = `Create a new Datum Cloud resource by providing a manifest in YAML or JSON
format, either from a file or piped through stdin.

datumctl create accepts Datum Cloud resource manifests — not Kubernetes
built-in resources. Use 'datumctl apply' for idempotent creation or updates.

Resource manifests must specify the correct apiVersion and kind for the
Datum Cloud resource type. Use 'datumctl explain <type>' to see the schema
for a resource type and 'datumctl api-resources' to list available types.`
	cmd.Example = `  # Create a project from a manifest file
  datumctl create -f ./project.yaml --organization <org-id>

  # Create a resource from stdin
  cat dnszone.yaml | datumctl create -f - --project <project-id>

  # Validate the resource without creating it
  datumctl create -f ./project.yaml --organization <org-id> --dry-run=server`
	return cmd
}
