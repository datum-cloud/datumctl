package ctx

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/picker"
)

func useCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use [context]",
		Short: "Switch the active context",
		Long: `Switch the active context to an organization or project.

If no argument is provided, an interactive picker is shown.
Use the format 'org/project' to select a project context, or just 'org' for an org context.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runUse,
	}
}

func runUse(_ *cobra.Command, args []string) error {
	if err := authutil.GuardAmbientMutation(); err != nil {
		return err
	}
	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		return err
	}

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts available. Run 'datumctl login' to get started.")
		return nil
	}

	var resolved *datumconfig.DiscoveredContext

	if len(args) == 1 {
		resolved = cfg.ResolveContext(args[0])
		if resolved == nil {
			return customerrors.NewUserErrorWithHint(
				fmt.Sprintf("Context %q not found.", args[0]),
				"Run 'datumctl ctx' to see available contexts.",
			)
		}
	} else {
		selected, err := picker.SelectContext(cfg.Contexts, cfg)
		if err != nil {
			return err
		}
		// Picker returns context Name directly — use exact lookup.
		resolved = cfg.ContextByName(selected)
		if resolved == nil {
			return fmt.Errorf("selected context not found")
		}
	}

	cfg.CurrentContext = resolved.Name

	if s := cfg.SessionByName(resolved.Session); s != nil {
		s.LastContext = resolved.Name
	}

	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("\n\u2713 Switched to %s\n", cfg.ContextDescription(resolved))
	return nil
}
