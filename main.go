package main

import (
	"fmt"
	"os"
	"strconv"

	"go.datum.net/datumctl/internal/cmd"
	customerrors "go.datum.net/datumctl/internal/errors"
	"k8s.io/component-base/cli"
	"k8s.io/component-base/logs"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
	"k8s.io/kubectl/pkg/cmd/util"
)

func main() {
	logs.GlogSetter(kubectlcmd.GetLogVerbosity(os.Args))
	cmd := cmd.RootCmd()
	if err := cli.RunNoErrOutput(cmd); err != nil {
		// Check if this is a user-facing error
		if userErr, ok := customerrors.IsUserError(err); ok {
			// Print clean user-friendly error message
			fmt.Fprintf(os.Stderr, "error: %s\n", userErr.Error())

			// Show technical details in verbose mode (v >= 4)
			verbosityStr := kubectlcmd.GetLogVerbosity(os.Args)
			verbosity, _ := strconv.Atoi(verbosityStr)
			if verbosity >= 4 && userErr.Err != nil {
				fmt.Fprintf(os.Stderr, "\nDetails:\n%v\n", userErr.Err)
			}

			os.Exit(1)
		}

		// Fall back to standard kubectl error handling for technical errors
		util.CheckErr(err)
	}
}
