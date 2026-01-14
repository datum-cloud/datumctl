package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/cmd/auth"
	"go.datum.net/datumctl/internal/cmd/mcp"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/apiresources"
	"k8s.io/kubectl/pkg/cmd/apply"
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

	ctx := context.Background()
	config, err := client.NewRestConfig(ctx)
	if err != nil {
		panic(err)
	}

	factory, err := client.NewDatumFactory(ctx, config)
	if err != nil {
		panic(err)
	}
	factory.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddGroup(&cobra.Group{ID: "auth", Title: "Authentication"})
	rootCmd.AddGroup(&cobra.Group{ID: "other", Title: "Other Commands"})
	rootCmd.AddGroup(&cobra.Group{ID: "resource", Title: "Resource Management"})

	authCommand := auth.Command()
	authCommand.GroupID = "auth"
	rootCmd.AddCommand(authCommand)

	rootCmd.AddCommand(WrapResourceCommand(get.NewCmdGet("datumctl", factory, ioStreams)))
	rootCmd.AddCommand(WrapResourceCommand(delcmd.NewCmdDelete(factory, ioStreams)))

	createCmd := create.NewCmdCreate(factory, ioStreams)
	createCmd.GroupID = "resource"
	rootCmd.AddCommand(createCmd)

	applyCmd := apply.NewCmdApply("datumctl", factory, ioStreams)
	applyCmd.GroupID = "resource"
	rootCmd.AddCommand(applyCmd)

	rootCmd.AddCommand(WrapResourceCommand(edit.NewCmdEdit(factory, ioStreams)))
	rootCmd.AddCommand(WrapResourceCommand(describe.NewCmdDescribe("datumctl", factory, ioStreams)))

	diffCmd := diff.NewCmdDiff(factory, ioStreams)
	diffCmd.GroupID = "resource"
	rootCmd.AddCommand(diffCmd)

	explainCmd := explain.NewCmdExplain("datumctl", factory, ioStreams)
	explainCmd.GroupID = "other"
	rootCmd.AddCommand(explainCmd)

	apiVersionCmd := apiresources.NewCmdAPIVersions(factory, ioStreams)
	apiVersionCmd.GroupID = "other"
	rootCmd.AddCommand(apiVersionCmd)

	apiResourceCmd := apiresources.NewCmdAPIResources(factory, ioStreams)
	apiResourceCmd.GroupID = "other"
	rootCmd.AddCommand(apiResourceCmd)

	rootCmd.AddCommand(mcp.Command())
	return rootCmd
}
