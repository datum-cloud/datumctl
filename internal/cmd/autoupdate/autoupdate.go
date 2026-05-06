// Package autoupdate provides the "datumctl auto-update" command, which
// triggers a manual upgrade of the datumctl binary and exposes enable/disable
// subcommands that toggle the auto-update preference in the config file.
package autoupdate

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	componentversion "k8s.io/component-base/version"

	"go.datum.net/datumctl/internal/datumconfig"
	customerrors "go.datum.net/datumctl/internal/errors"
	"go.datum.net/datumctl/internal/updatecheck"
)

// Command returns the "auto-update" command tree.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-update",
		Short: "Update datumctl to the latest released version",
		Long: `Download and install the latest released datumctl binary, replacing
the currently running executable.

When auto-update is enabled in the config, datumctl also performs this
upgrade automatically the next time you run any command and a newer release
is available; the original command is then re-run on the new binary.

Use 'datumctl auto-update enable' or 'datumctl auto-update disable' to
toggle the on-startup behaviour. Running 'datumctl auto-update' without
arguments performs an upgrade immediately, regardless of that setting.`,
		Example: `  # Upgrade to the latest release now
  datumctl auto-update

  # Enable automatic upgrades on every invocation
  datumctl auto-update enable

  # Disable automatic upgrades
  datumctl auto-update disable`,
		RunE: runUpdate,
	}
	cmd.AddCommand(enableCmd(), disableCmd(), statusCmd())
	return cmd
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	current := componentversion.Get().GitVersion

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()
	latest, err := updatecheck.FetchLatestVersion(ctx)
	if err != nil {
		return customerrors.NewUserError(fmt.Sprintf("look up latest version: %v", err))
	}
	if latest == "" {
		return customerrors.NewUserError("could not determine the latest released version")
	}
	if latest == current {
		fmt.Fprintf(cmd.OutOrStdout(), "datumctl %s is already the latest version.\n", current)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updating datumctl %s → %s ...\n", current, latest)
	if err := updatecheck.SelfUpdate(cmd.Context(), latest); err != nil {
		return customerrors.NewUserError(fmt.Sprintf("auto-update failed: %v", err))
	}
	fmt.Fprintf(cmd.OutOrStdout(), "datumctl updated to %s.\n", latest)
	return nil
}

func enableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable automatic upgrades when a newer release is detected",
		Long: `Set auto-update to true in the datumctl config file (~/.datumctl/config).

When enabled, datumctl checks for a newer release on every invocation; if
one is found, it is downloaded and installed before the requested command
runs, and the original command is re-run on the freshly installed binary.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setAutoUpdate(cmd, true)
		},
	}
}

func disableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable automatic upgrades; only print a warning when outdated",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setAutoUpdate(cmd, false)
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether automatic upgrades are enabled",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := datumconfig.LoadV1Beta1()
			if err != nil {
				return customerrors.NewUserError(fmt.Sprintf("load config: %v", err))
			}
			state := "disabled"
			if cfg.AutoUpdate {
				state = "enabled"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "auto-update: %s\n", state)
			return nil
		},
	}
}

func setAutoUpdate(cmd *cobra.Command, enabled bool) error {
	cfg, err := datumconfig.LoadV1Beta1()
	if err != nil {
		return customerrors.NewUserError(fmt.Sprintf("load config: %v", err))
	}
	cfg.AutoUpdate = enabled
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		return customerrors.NewUserError(fmt.Sprintf("save config: %v", err))
	}
	state := "disabled"
	if enabled {
		state = "enabled"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "auto-update %s.\n", state)
	return nil
}

