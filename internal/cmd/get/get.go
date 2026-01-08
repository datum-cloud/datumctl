package get

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/client"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/get"
)

func Command(factory *client.DatumCloudFactory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	preRunFunc := func(cmd *cobra.Command, args []string) error {
		if args[0] == "organizations" || args[0] == "organization" {
			args[0] = "organizationmemberships"
			cmd.Flag("all-namespaces").Value.Set("true")
		}
		return nil
	}
	getCmd := get.NewCmdGet("datumctl", factory, ioStreams)
	getCmd.PreRunE = preRunFunc
	return getCmd
}
