package apiversions

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/apiresources"
	"k8s.io/kubectl/pkg/cmd/util"
)

func Command(factory util.Factory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	return apiresources.NewCmdAPIVersions(factory, ioStreams)
}
