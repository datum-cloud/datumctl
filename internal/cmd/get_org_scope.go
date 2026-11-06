package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/client"
)

// getFatalHint is the hint for a Forbidden that the running get command
// expects. kubectl get reports its errors through cmdutil.CheckErr, which only
// passes the formatted message on, so the hint is recorded before the command
// runs and attached by FatalHint when the error is printed.
var getFatalHint string

// FatalHint returns guidance to print after a fatal kubectl error message, or
// "" when there is none.
func FatalHint(msg string) string {
	if getFatalHint == "" || !strings.Contains(msg, "(Forbidden)") {
		return ""
	}
	return getFatalHint
}

// orgScopeAllNamespacesHint returns the hint for a get across all namespaces
// at organization scope. The authorizer refuses a namespaced list with no
// namespace in an organization's control plane, whatever the caller's role,
// because an organization's resources live in a single namespace and project
// resources live in each project's own control plane.
func orgScopeAllNamespacesHint(c *cobra.Command, factory *client.DatumCloudFactory, args []string) string {
	allNamespaces, err := c.Flags().GetBool("all-namespaces")
	if err != nil || !allNamespaces {
		return ""
	}
	scope, err := factory.ConfigFlags.ResolvedScopeDetails()
	if err != nil || scope.Organization == "" {
		return ""
	}
	kind := "<kind>"
	if len(args) > 0 {
		kind, _, _ = strings.Cut(args[0], "/")
	}
	return fmt.Sprintf(`Listing across all namespaces is not allowed at organization scope. List a project's resources instead:
  datumctl get %s --project <project> -A
To find project names, run:
  datumctl get projects --organization %s`, kind, scope.Organization)
}
