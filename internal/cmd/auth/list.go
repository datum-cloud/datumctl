package auth

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all authenticated users",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func runList(_ *cobra.Command, _ []string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	if len(cfg.Sessions) == 0 {
		fmt.Println("No authenticated users. Run 'datumctl login' to get started.")
		return nil
	}

	showEndpoint := cfg.HasMultipleEndpoints()

	tw := tabwriter.NewWriter(os.Stdout, 0, 4, 3, ' ', 0)
	if showEndpoint {
		fmt.Fprintln(tw, "  USER\tENDPOINT\tSTATUS")
	} else {
		fmt.Fprintln(tw, "  USER\tSTATUS")
	}

	for _, s := range cfg.Sessions {
		status := ""
		if s.Name == cfg.ActiveSession {
			status = "Active"
		}
		if showEndpoint {
			fmt.Fprintf(tw, "  %s\t%s\t%s\n", s.UserEmail, datumconfig.StripScheme(s.Endpoint.Server), status)
		} else {
			fmt.Fprintf(tw, "  %s\t%s\n", s.UserEmail, status)
		}
	}
	tw.Flush()
	return nil
}
