package apiversions

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/apiresources"
	"k8s.io/kubectl/pkg/cmd/util"
)

func CommandApiResources(factory util.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	return apiresources.NewCmdAPIResources(factory, ioStreams)
}
