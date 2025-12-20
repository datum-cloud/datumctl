package v2

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
	"golang.org/x/oauth2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/get"
)

func Command(factory *client.MyFactory, ioStreams genericclioptions.IOStreams, projectID *string, organizationID *string) *cobra.Command {
	getCmd := get.NewCmdGet("datumctl", factory, ioStreams)
	getCmd.Run = func(cmd *cobra.Command, args []string) {
		apiHostname, err := authutil.GetAPIHostname()
		if err != nil {
			panic(fmt.Errorf("get API hostname: %w", err))
		}
		restConfig, err := factory.ToRESTConfig()
		if err != nil {
			panic(fmt.Errorf("get API hostname: %w", err))
		}
		tknSrc, err := authutil.GetTokenSource(cmd.Context())
		if err != nil {
			panic(err)
		}
		restConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{Source: tknSrc, Base: rt}
		}
		switch {
		case (projectID == nil || *projectID == "") && (organizationID == nil || *organizationID == ""):
		case (projectID == nil || *projectID == "") && (organizationID != nil || *organizationID != ""):
			factory.RestConfig.Host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/organizations/%s/control-plane",
				apiHostname, *organizationID)
		case (projectID != nil || *projectID != "") && (organizationID == nil || *organizationID == ""):
			factory.RestConfig.Host = fmt.Sprintf("https://%s/apis/resourcemanager.miloapis.com/v1alpha1/projects/%s/control-plane",
				apiHostname, *projectID)
		default:
			panic(fmt.Errorf("exactly one of organizationID or projectID must be provided"))
		}
		cmdInt := get.NewCmdGet("datumctl", factory, ioStreams)
		cmdInt.Run(cmd, args)
	}
	return getCmd
}
