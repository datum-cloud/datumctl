package main

import (
	"os"

	"go.datum.net/datumctl/internal/cmd"
	"k8s.io/component-base/cli"
	"k8s.io/component-base/logs"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
	"k8s.io/kubectl/pkg/cmd/util"
)

func main() {
	logs.GlogSetter(kubectlcmd.GetLogVerbosity(os.Args))
	cmd := cmd.RootCmd()
	if err := cli.RunNoErrOutput(cmd); err != nil {
		util.CheckErr(err)
	}
}
