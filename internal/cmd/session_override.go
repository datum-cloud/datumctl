package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
)

// sessionFlag is the global flag that runs one command as a named session.
const sessionFlag = "session"

// applySessionOverride installs the session named by --session, or else by
// DATUM_SESSION, as this process's session. Every lookup of the active session
// and current context goes through datumconfig, so installing it here reaches
// every command without touching the config file.
//
// An explicit --session value is validated immediately: a wrong value fails
// this command right away, same as before. DATUM_SESSION is recorded without
// validating it — datumconfig resolves it lazily, the first time something
// consults the active session, so a stale export left behind by a previous
// logout (or inherited from a plugin) does not break commands that never read
// the session, such as version, plugin list, or completion.
//
// Commands annotated with datumconfig.SessionOverrideAnnotation change the
// active session themselves: they reject --session and ignore DATUM_SESSION.
// A command that defines its own local --session flag (auth get-token) handles
// that flag itself; DATUM_SESSION still applies when the flag is not given.
func applySessionOverride(cmd *cobra.Command) error {
	datumconfig.ClearSessionOverride()

	flag := cmd.Flags().Lookup(sessionFlag)
	flagSet := flag != nil && flag.Changed

	if rejectsSessionOverride(cmd) {
		if flagSet {
			return customerrors.NewUserErrorWithHint(
				fmt.Sprintf("'%s' changes the active session, so it does not accept --session.", cmd.CommandPath()),
				"Run it without --session. It also ignores DATUM_SESSION.",
			)
		}
		return nil
	}

	ownsFlag := flag != nil && flag != cmd.Root().PersistentFlags().Lookup(sessionFlag)
	if flagSet {
		if ownsFlag {
			return nil
		}
		_, err := datumconfig.ApplySessionOverride(flag.Value.String(), datumconfig.SessionOverrideFromFlag)
		return err
	}

	if v := os.Getenv(datumconfig.SessionEnvVar); v != "" {
		datumconfig.SetPendingSessionOverride(v)
	}
	return nil
}

// rejectsSessionOverride reports whether cmd, or a parent of it, is annotated
// as changing the active session.
func rejectsSessionOverride(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Annotations[datumconfig.SessionOverrideAnnotation] == datumconfig.SessionOverrideRejected {
			return true
		}
	}
	return false
}
