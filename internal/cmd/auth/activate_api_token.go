package auth

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"go.datum.net/datumctl/internal/datum"
	"go.datum.net/datumctl/internal/keyring"
)

func activateAPITokenCmd() *cobra.Command {
	var hostname string
	var withToken bool

	cmd := &cobra.Command{
		Use:   "activate-api-token",
		Short: "Authenticate to Datum Cloud with an API token and store in keyring",
		RunE: func(cmd *cobra.Command, _ []string) error {

			var token string

			if withToken {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read token from standard input: %w", err)
				}
				token = strings.TrimSpace(string(b))
			} else {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return errors.New("cannot prompt for token without a TTY; use --with-token to read from stdin")
				}

				fmt.Fprint(os.Stderr, "Enter API token: ")
				rawToken, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return fmt.Errorf("failed to read token from prompt: %w", err)
				}

				fmt.Fprintln(os.Stderr, "")
				token = strings.TrimSpace(string(rawToken))
			}

			// Make sure the token is valid
			tokenSource := datum.NewAPITokenSource(token, hostname)
			_, err := tokenSource.Token()
			if err != nil {
				fmt.Printf("failed to verify API token for %s: %s\n", hostname, err)
				os.Exit(1)
			}

			if err := keyring.Set("datumctl", "datumctl", token); err != nil {
				return fmt.Errorf("failed to store token in keyring: %w", err)
			}

			fmt.Println("API token verified and stored in keyring")

			return nil
		},
	}

	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read API token from standard input")
	cmd.Flags().StringVar(&hostname, "hostname", "api.datum.net", "The hostname of the Datum Cloud instance to authenticate with")

	return cmd
}
