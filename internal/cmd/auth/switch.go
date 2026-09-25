package auth

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/templates"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/picker"
)

func switchCmd() *cobra.Command {
	var endpoint string

	cmd := &cobra.Command{
		Use:   "switch [email | session]",
		Short: "Switch the active Datum Cloud user session",
		Long: templates.LongDesc(`
			Change which locally stored user account is treated as active.

			The active user's credentials are used for all datumctl commands that
			require authentication. Other signed-in accounts stay signed in.

			Pass the email address shown by 'datumctl auth list'. When the same
			email is signed in on more than one endpoint, add --endpoint with the
			endpoint shown by 'datumctl auth list' to pick one. You can also pass
			the full session name (email@endpoint) instead.

			With no argument, or when several sessions match and no --endpoint is
			given, an interactive picker is shown. The picker needs a terminal;
			without one, datumctl prints the command to run for each match.

			Each session remembers the last context you used, so switching users
			also restores the context. To add a new account, run 'datumctl login'.

			This command changes the active session, so it rejects the global
			--session flag and ignores DATUM_SESSION. To run a single command as
			another session without switching, pass --session to that command.`),
		Annotations: map[string]string{
			datumconfig.SessionOverrideAnnotation: datumconfig.SessionOverrideRejected,
		},
		Example: templates.Examples(`
			# Interactive session picker
			datumctl auth switch

			# Switch to a specific account
			datumctl auth switch user@example.com

			# Pick the staging session for an email signed in on two endpoints
			datumctl auth switch user@example.com --endpoint api.staging.env.datum.net

			# Switch by full session name
			datumctl auth switch user@example.com@api.datum.net`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runSwitch(args, endpoint)
		},
	}

	cmd.Flags().StringVar(&endpoint, "endpoint", "",
		"API endpoint of the session to switch to, as shown by 'datumctl auth list' (for example api.datum.net)")

	return cmd
}

// normalizeEndpoint reduces an endpoint to the bare host form that
// 'datumctl auth list' prints, so a scheme, trailing slash, or letter case in
// the user's input does not prevent a match.
func normalizeEndpoint(s string) string {
	return strings.ToLower(datumconfig.StripScheme(datumconfig.CleanBaseServer(strings.TrimSpace(s))))
}

func runSwitch(args []string, endpoint string) error {
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	if len(cfg.Sessions) == 0 {
		return customerrors.NewUserErrorWithHint(
			"No authenticated sessions.",
			"Run 'datumctl login' to authenticate.",
		)
	}

	// Gather the candidate sessions named by the argument: an exact session
	// name wins, then every session for the email, then all sessions.
	var (
		candidates []*datumconfig.Session
		subject    string // who the candidates belong to, for messages
	)
	switch {
	case len(args) == 0:
		for i := range cfg.Sessions {
			candidates = append(candidates, &cfg.Sessions[i])
		}
	case cfg.SessionByName(args[0]) != nil:
		candidates = []*datumconfig.Session{cfg.SessionByName(args[0])}
		subject = args[0]
	default:
		subject = args[0]
		candidates = cfg.SessionByEmail(subject)
		if len(candidates) == 0 {
			return customerrors.NewUserErrorWithHint(
				fmt.Sprintf("No sessions found for %s.", subject),
				"Run 'datumctl auth list' to see authenticated users, or 'datumctl login' to add a new one.",
			)
		}
	}

	if endpoint != "" {
		want := normalizeEndpoint(endpoint)
		var matched []*datumconfig.Session
		var known []string
		for _, s := range candidates {
			host := normalizeEndpoint(s.Endpoint.Server)
			if host == want {
				matched = append(matched, s)
			}
			known = appendUniqueString(known, datumconfig.StripScheme(s.Endpoint.Server))
		}
		if len(matched) == 0 {
			msg := fmt.Sprintf("No session on endpoint %s.", datumconfig.StripScheme(endpoint))
			hint := "Endpoints any account is signed in on: " + strings.Join(known, ", ")
			if subject != "" {
				msg = fmt.Sprintf("No session for %s on endpoint %s.", subject, datumconfig.StripScheme(endpoint))
				hint = "Signed-in endpoints: " + strings.Join(known, ", ")
			}
			return customerrors.NewUserErrorWithHint(msg, hint)
		}
		candidates = matched
	}

	var sessionName string
	switch {
	case len(candidates) == 1:
		sessionName = candidates[0].Name
	case !picker.IsTerminal():
		return ambiguousSessionError(cfg, candidates, subject)
	default:
		sessionName, err = picker.SelectSession(candidates, cfg.ActiveSession)
		if err != nil {
			return err
		}
	}

	session := cfg.SessionByName(sessionName)
	if session == nil {
		return fmt.Errorf("session %q not found", sessionName)
	}

	cfg.ActiveSession = sessionName

	// Repoint the current context at the new session's last context so whoami
	// and requests follow the switch. Clear it when the new session has no
	// usable context, letting the ActiveSession fallback take over rather than
	// leaving a stale context from the previous environment selected.
	if session.LastContext != "" && cfg.ContextByName(session.LastContext) != nil {
		cfg.CurrentContext = session.LastContext
	} else {
		cfg.CurrentContext = ""
	}

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n✓ Switched to %s (%s)\n", session.UserName, session.UserEmail)
	if ctxEntry := cfg.CurrentContextEntry(); ctxEntry != nil {
		fmt.Printf("  Context:  %s\n", datumconfig.FormatWithID(cfg.DisplayRef(ctxEntry), ctxEntry.Ref()))
	}
	if cfg.HasMultipleEndpoints() {
		fmt.Printf("  Endpoint: %s\n", datumconfig.StripScheme(session.Endpoint.Server))
	}

	return nil
}

// ambiguousSessionError explains that several sessions match and the picker
// cannot run, listing a copy-pasteable command for each match.
func ambiguousSessionError(cfg *datumconfig.ConfigV1Beta1, sessions []*datumconfig.Session, subject string) error {
	msg := "Multiple sessions found. Choosing one interactively requires a terminal."
	if subject != "" {
		msg = fmt.Sprintf("%s is signed in on more than one endpoint. Choosing one interactively requires a terminal.", subject)
	}
	var b strings.Builder
	b.WriteString("Run one of:")
	for _, s := range sessions {
		fmt.Fprintf(&b, "\n  datumctl auth switch %s", cfg.SwitchArgs(s))
	}
	return customerrors.NewUserErrorWithHint(msg, b.String())
}

func appendUniqueString(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
