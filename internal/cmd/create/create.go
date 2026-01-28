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
	return cmd
}
