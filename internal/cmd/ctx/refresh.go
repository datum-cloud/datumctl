package ctx

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
	customerrors "go.datum.net/datumctl/internal/errors"
)

func runRefresh(cmd *cobra.Command) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	session := cfg.ActiveSessionEntry()
	if session == nil {
		return customerrors.NewUserErrorWithHint(
			"No active session.",
			"Run 'datumctl login' to authenticate.",
		)
	}

	fmt.Fprintln(os.Stderr, "Refreshing contexts...")

	count, err := discovery.RefreshSession(cmd.Context(), cfg, session)
	if err != nil {
		return err
	}

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\u2713 Discovered %d context(s)\n\n", count)
	return nil
}
