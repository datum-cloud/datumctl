package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"go.datum.net/datumctl/internal/cmd"
	customerrors "go.datum.net/datumctl/internal/errors"
	"k8s.io/component-base/cli"
	"k8s.io/component-base/logs"
	kubectlcmd "k8s.io/kubectl/pkg/cmd"
	"k8s.io/kubectl/pkg/cmd/util"
)

func main() {
	logs.GlogSetter(kubectlcmd.GetLogVerbosity(os.Args))
	rootCmd := cmd.RootCmd()

	// Route kubectl's internal fatal errors (from util.CheckErr) through the
	// same Format helper so --error-format applies uniformly. In human mode
	// we print kubectl's preformatted string verbatim to match legacy output;
	// in structured modes we strip the "error: " prefix kubectl may add and
	// re-encode the message inside the JSON/YAML envelope.
	util.BehaviorOnFatal(func(msg string, code int) {
		msg = strings.TrimSuffix(msg, "\n")
		format := formatFor(rootCmd)
		if format == customerrors.FormatHuman {
			fmt.Fprintln(os.Stderr, msg)
			os.Exit(code)
		}
		clean := strings.TrimPrefix(msg, "error: ")
		customerrors.Format(os.Stderr, errors.New(clean), format, verbosity())
		os.Exit(code)
	})

	if err := cli.RunNoErrOutput(rootCmd); err != nil {
		customerrors.Format(os.Stderr, err, formatFor(rootCmd), verbosity())
		os.Exit(1)
	}
}

// formatFor reads --error-format off the parsed root command, falling back to
// human when the flag is absent or unrecognized (e.g. when an error fires
// before flag parsing completes).
func formatFor(rootCmd *cobra.Command) string {
	f := rootCmd.PersistentFlags().Lookup("error-format")
	if f == nil {
		return customerrors.FormatHuman
	}
	switch f.Value.String() {
	case customerrors.FormatJSON:
		return customerrors.FormatJSON
	case customerrors.FormatYAML:
		return customerrors.FormatYAML
	default:
		return customerrors.FormatHuman
	}
}

func verbosity() int {
	v, _ := strconv.Atoi(kubectlcmd.GetLogVerbosity(os.Args))
	return v
}
