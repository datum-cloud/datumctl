package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"

	"go.datum.net/datumctl/internal/client"
)

// WrapDeleteCommand applies WrapResourceCommand and, before the deletion runs,
// tells the user on stderr which project or organization it will act on. When
// no --project or --organization flag is given the scope quietly comes from
// the active context, and a destructive command should not keep that quiet.
func WrapDeleteCommand(cmd *cobra.Command, factory *client.DatumCloudFactory) *cobra.Command {
	cmd = WrapResourceCommand(cmd)

	origRun := cmd.Run
	origRunE := cmd.RunE
	cmd.Run = nil
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if msg := deleteScopeNotice(c, factory); msg != "" {
			fmt.Fprintln(c.ErrOrStderr(), msg)
		}
		if origRunE != nil {
			return origRunE(c, args)
		}
		if origRun != nil {
			origRun(c, args)
		}
		return nil
	}
	return cmd
}

// deleteScopeNotice resolves the scope for cmd and returns the line to print,
// or "" when there is nothing useful to say. Resolution errors are left for
// the command itself to report, so they are not duplicated here.
func deleteScopeNotice(cmd *cobra.Command, factory *client.DatumCloudFactory) string {
	scope, err := factory.ConfigFlags.ResolvedScopeDetails()
	if err != nil {
		return ""
	}
	dryRun := false
	if cmd.Flags().Lookup("dry-run") != nil {
		strategy, err := cmdutil.GetDryRunStrategy(cmd)
		dryRun = err == nil && strategy != cmdutil.DryRunNone
	}
	return deleteScopeMessage(scope, dryRun)
}

// deleteScopeMessage renders the scope line, for example:
//
//	Deleting in project "my-project" (organization "my-org", from the active context)
//	Deleting in organization "my-org"
//	Dry run in project "my-project" (from DATUM_PROJECT)
func deleteScopeMessage(scope client.Scope, dryRun bool) string {
	verb := "Deleting"
	if dryRun {
		verb = "Dry run"
	}

	var target string
	switch {
	case scope.PlatformWide:
		target = "platform-wide"
	case scope.Project != "":
		target = fmt.Sprintf("in project %q", scope.Project)
	case scope.Organization != "":
		target = fmt.Sprintf("in organization %q", scope.Organization)
	default:
		return ""
	}

	var notes []string
	if scope.ProjectOrganization != "" {
		notes = append(notes, fmt.Sprintf("organization %q", scope.ProjectOrganization))
	}
	switch scope.Source {
	case client.ScopeSourceContext:
		notes = append(notes, "from the active context")
	case client.ScopeSourceEnv:
		if scope.Project != "" {
			notes = append(notes, "from DATUM_PROJECT")
		} else {
			notes = append(notes, "from DATUM_ORGANIZATION")
		}
	}

	msg := verb + " " + target
	if len(notes) > 0 {
		msg += " (" + strings.Join(notes, ", ") + ")"
	}
	return msg
}
