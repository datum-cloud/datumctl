package auth

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/keyring"
)

func logoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove authentication for Datum Cloud",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := keyring.Delete("datumctl", "datumctl"); err != nil {
				if errors.Is(err, keyring.ErrNotFound) {
					return fmt.Errorf("no API token to remove from keyring")
				} else {
					return fmt.Errorf("failed to delete token from keyring: %w", err)
				}
			}
			fmt.Println("API token removed from keyring")
			return nil
		},
	}

	return cmd
}
