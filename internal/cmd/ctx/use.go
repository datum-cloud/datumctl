package ctx

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/picker"
)

func useCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [context]",
		Short: "Switch the active context",
		Long: `Switch the active context to an organization or project.

If no argument is provided, an interactive picker is shown.
Use the format 'org/project' to select a project context, or just 'org' for an org context.

Switching context can change the active session, so this command rejects the
global --session flag and ignores DATUM_SESSION. To use another session's
context for one command, pass --session to that command instead.`,
		Annotations: map[string]string{
			datumconfig.SessionOverrideAnnotation: datumconfig.SessionOverrideRejected,
		},
		Args: cobra.MaximumNArgs(1),
		RunE: runUse,
	}
}

func runUse(cmd *cobra.Command, args []string) error {
	ctx := cmdContext(cmd)

	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	// Contexts are addressed relative to the active session, since the same
	// org/project ref can exist in more than one environment.
	session := cfg.ActiveSessionEntry()
	if session == nil {
		fmt.Println("No contexts available. Run 'datumctl login' to get started.")
		return nil
	}

	api, err := discovery.ForSession(ctx, session)
	if err != nil {
		return err
	}

	var resolved *datumconfig.DiscoveredContext
	var dir *discovery.Directory

	if len(args) == 1 {
		resolved, dir, err = discovery.Lookup(ctx, api, session.Name, args[0])
		if errors.Is(err, discovery.ErrNotFound) {
			// The ref may belong to a different environment's session — point the
			// user at that session rather than a bare "not found".
			if owner := findContextOwner(ctx, cfg, args[0], session.Name); owner != nil {
				return customerrors.NewUserErrorWithHint(
					fmt.Sprintf("Context %q belongs to the session for %s, which is not active.", args[0], owner.UserEmail),
					fmt.Sprintf("Run 'datumctl auth switch %s' first, then 'datumctl ctx use %s'.", cfg.SwitchArgs(owner), args[0]),
				)
			}
			return customerrors.NewUserErrorWithHint(
				fmt.Sprintf("Context %q not found.", args[0]),
				"Check the organization and project IDs, or run 'datumctl ctx' to see available contexts.",
			)
		}
		if err != nil {
			return fmt.Errorf("look up context %q: %w", args[0], err)
		}
	} else {
		dir, err = discovery.List(ctx, api, session.Name)
		if err != nil {
			return fmt.Errorf("list contexts: %w", err)
		}
		dir.WarnFailures(os.Stderr)
		resolved, err = picker.SelectContext(dir, cfg.CurrentContextName())
		if err != nil {
			return err
		}
	}

	cfg.SelectContext(*resolved)

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n✓ Switched to %s\n", dir.Describe(resolved))
	return nil
}

// findContextOwner returns a signed-in session other than excludeSession in
// which query resolves, or nil.
func findContextOwner(ctx context.Context, cfg *datumconfig.ConfigV1Beta1, query, excludeSession string) *datumconfig.Session {
	for i := range cfg.Sessions {
		s := &cfg.Sessions[i]
		if s.Name == excludeSession {
			continue
		}
		api, err := discovery.ForSession(ctx, s)
		if err != nil {
			continue
		}
		if found, _, err := discovery.Lookup(ctx, api, s.Name, query); err == nil && found != nil {
			return s
		}
	}
	return nil
}
