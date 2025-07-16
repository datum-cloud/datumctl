package cmd

import (
	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/cmd/auth"
	"go.datum.net/datumctl/internal/cmd/get"
)

var (
	rootCmd = &cobra.Command{
		Use:   "datumctl",
		Short: "A CLI for interacting with the Datum platform",
	}
)

func init() {
	rootCmd.AddCommand(auth.Command())
	rootCmd.AddCommand(get.Command())
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}
