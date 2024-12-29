package cmd

import (
	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/cmd/auth"
	"go.datum.net/datumctl/internal/cmd/organizations"
)

var rootCmd = &cobra.Command{
	Use:   "datumctl",
	Short: "A CLI for Datum Cloud",
}

func init() {
	rootCmd.AddCommand(
		auth.Command,
		organizations.Command,
	)
}

func Execute() error {
	return rootCmd.Execute()
}
