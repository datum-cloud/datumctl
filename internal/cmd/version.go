package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/cmd/version"
	"sigs.k8s.io/yaml"

	datumversion "go.datum.net/datumctl/internal/version"
)

// runVersion replaces kubectl's version Run. kubectl reads the client
// version straight from k8s.io/component-base/version, which can't hold
// the pseudo-version datumversion.Get resolves for unreleased builds. It
// also skips kubectl's client/server skew check: datumctl's version and
// the Datum Cloud API server's version are independent, so comparing them
// is meaningless, and the check fails outright on non-semver server
// versions.
func runVersion(f cmdutil.Factory, out io.Writer, cmd *cobra.Command) error {
	clientOnly, _ := cmd.Flags().GetBool("client")
	output, _ := cmd.Flags().GetString("output")
	if output != "" && output != "yaml" && output != "json" {
		return fmt.Errorf("--output must be 'yaml' or 'json'")
	}

	clientInfo := datumversion.Info()
	info := version.Version{ClientVersion: &clientInfo}
	if kv, ok := version.GetKustomizeModVersion(); ok {
		info.KustomizeVersion = kv
	}

	var serverErr error
	if !clientOnly {
		dc, err := f.ToDiscoveryClient()
		// An empty config means no context is set up; print client info only.
		if err != nil && !clientcmd.IsEmptyConfig(err) {
			return err
		}
		if dc != nil {
			dc.Invalidate()
			info.ServerVersion, serverErr = dc.ServerVersion()
		}
	}

	switch output {
	case "":
		fmt.Fprintf(out, "Client Version: %s\n", info.ClientVersion.GitVersion)
		if info.KustomizeVersion != "" {
			fmt.Fprintf(out, "Kustomize Version: %s\n", info.KustomizeVersion)
		}
		if info.ServerVersion != nil {
			fmt.Fprintf(out, "Server Version: %s\n", info.ServerVersion.GitVersion)
		}
	case "yaml":
		b, err := yaml.Marshal(&info)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(b))
	case "json":
		b, err := json.MarshalIndent(&info, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(b))
	}
	return serverErr
}
