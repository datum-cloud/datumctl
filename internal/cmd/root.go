package cmd

import (
	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/cmd/auth"
	"go.datum.net/datumctl/internal/cmd/mcp"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/apiresources"
	"k8s.io/kubectl/pkg/cmd/apply"
	"k8s.io/kubectl/pkg/cmd/clusterinfo"
	"k8s.io/kubectl/pkg/cmd/create"
	delcmd "k8s.io/kubectl/pkg/cmd/delete"
	"k8s.io/kubectl/pkg/cmd/describe"
	"k8s.io/kubectl/pkg/cmd/diff"
	"k8s.io/kubectl/pkg/cmd/edit"
	"k8s.io/kubectl/pkg/cmd/explain"
	"k8s.io/kubectl/pkg/cmd/get"
)

func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "datumctl",
		Short: "A CLI for interacting with the Datum platform",
	}

	ioStreams := genericclioptions.IOStreams{
		In:     rootCmd.InOrStdin(),
		Out:    rootCmd.OutOrStdout(),
		ErrOut: rootCmd.ErrOrStderr(),
	}

	ctx := rootCmd.Context()
	config, err := client.NewRestConfig(ctx)
	if err != nil {
		panic(err)
	}

	factory, err := client.NewDatumFactory(rootCmd.Context(), config)
	if err != nil {
		panic(err)
	}
	factory.AddFlags(rootCmd.PersistentFlags())
	rootCmd.AddCommand(auth.Command())

	rootCmd.AddCommand(WrapResourceCommand(get.NewCmdGet("datumctl", factory, ioStreams)))
	rootCmd.AddCommand(WrapResourceCommand(delcmd.NewCmdDelete(factory, ioStreams)))
	rootCmd.AddCommand(create.NewCmdCreate(factory, ioStreams))
	rootCmd.AddCommand(apply.NewCmdApply("datumctl", factory, ioStreams))
	rootCmd.AddCommand(WrapResourceCommand(edit.NewCmdEdit(factory, ioStreams)))
	rootCmd.AddCommand(WrapResourceCommand(describe.NewCmdDescribe("datumctl", factory, ioStreams)))

	rootCmd.AddCommand(diff.NewCmdDiff(factory, ioStreams))
	rootCmd.AddCommand(explain.NewCmdExplain("datumctl", factory, ioStreams))

	rootCmd.AddCommand(apiresources.NewCmdAPIVersions(factory, ioStreams))
	rootCmd.AddCommand(clusterinfo.NewCmdClusterInfo(factory, ioStreams))
	rootCmd.AddCommand(apiresources.NewCmdAPIResources(factory, ioStreams))

	rootCmd.AddCommand(mcp.Command())
	return rootCmd
}
