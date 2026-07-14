package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"

	"go.datum.net/datumctl/internal/authutil"
	datumclient "go.datum.net/datumctl/internal/client"
	"go.datum.net/datumctl/internal/onboarding"
)

const (
	onboardingStatusAnnotation = "datum.net/onboarding-status"
	onboardingReasonAnnotation = "datum.net/onboarding-reason"
	onboardingActionAnnotation = "datum.net/onboarding-action-url"
)

// WrapGetCommand wraps kubectl get so organization listings always use the
// user control plane, skip the onboarding gate, and show per-org onboarding
// status in the default table output for `datumctl get organizations`.
func WrapGetCommand(cmd *cobra.Command, factory *datumclient.DatumCloudFactory, ioStreams genericclioptions.IOStreams) *cobra.Command {
	origRun := cmd.Run
	origRunE := cmd.RunE
	cmd.Run = nil
	cmd.RunE = func(c *cobra.Command, args []string) error {
		if isOrganizationsAlias(args) {
			return runGetOrganizations(c, ioStreams, args)
		}

		if len(args) > 0 && isOrganizationMembershipResource(args[0]) {
			prev := factory.ConfigFlags.ForceUserControlPlane
			factory.ConfigFlags.ForceUserControlPlane = true
			defer func() { factory.ConfigFlags.ForceUserControlPlane = prev }()
			_ = c.Flags().Set("all-namespaces", "true")
		}

		if origRunE != nil {
			return origRunE(c, args)
		}
		if origRun != nil {
			origRun(c, args)
		}
		return nil
	}
	cmd.GroupID = "resource"

	if inner := cmd.ValidArgsFunction; inner != nil {
		cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			comps, directive := inner(cmd, args, toComplete)
			if len(args) == 0 && strings.HasPrefix("organizations", toComplete) {
				comps = append(comps, "organizations")
			}
			return comps, directive
		}
	}

	return cmd
}

func isOrganizationsAlias(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return args[0] == "organizations" || args[0] == "organization"
}

func isOrganizationMembershipResource(name string) bool {
	switch name {
	case "organizationmemberships", "organizationmembership",
		"organizationmemberships.resourcemanager.miloapis.com",
		"organizationmembership.resourcemanager.miloapis.com":
		return true
	default:
		return false
	}
}

func runGetOrganizations(
	cmd *cobra.Command,
	ioStreams genericclioptions.IOStreams,
	args []string,
) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	userClient, err := datumclient.NewUserContextualClient(ctx)
	if err != nil {
		return err
	}

	var memberships resourcemanagerv1alpha1.OrganizationMembershipList
	if err := userClient.List(ctx, &memberships); err != nil {
		return fmt.Errorf("list organization memberships: %w", err)
	}

	if nameFilter := organizationNameFilter(args); nameFilter != "" {
		filtered := memberships.Items[:0]
		for _, m := range memberships.Items {
			if m.Spec.OrganizationRef.Name == nameFilter || m.Name == nameFilter {
				filtered = append(filtered, m)
			}
		}
		memberships.Items = filtered
	}

	userKey, session, err := authutil.GetUserKeyForCurrentSession()
	if err != nil {
		return err
	}
	tknSrc, err := authutil.GetTokenSourceForUser(ctx, userKey)
	if err != nil {
		return err
	}
	userID, err := authutil.GetUserIDFromTokenForUser(userKey)
	if err != nil {
		return err
	}
	apiHostname, err := authutil.GetAPIHostnameForUser(userKey)
	if err != nil {
		if session != nil && session.Endpoint.Server != "" {
			apiHostname = strings.TrimPrefix(strings.TrimPrefix(session.Endpoint.Server, "https://"), "http://")
			apiHostname = strings.TrimSuffix(apiHostname, "/")
		} else {
			return err
		}
	}

	orgRefs := make([]onboarding.OrgRef, 0, len(memberships.Items))
	seen := make(map[string]bool, len(memberships.Items))
	for _, m := range memberships.Items {
		orgID := m.Spec.OrganizationRef.Name
		if orgID == "" || seen[orgID] {
			continue
		}
		seen[orgID] = true
		orgRefs = append(orgRefs, onboarding.OrgRef{
			ID:          orgID,
			DisplayName: m.Status.Organization.DisplayName,
		})
	}

	statuses := onboarding.CheckOrgs(ctx, apiHostname, tknSrc, userID, orgRefs)
	annotateMemberships(&memberships, statuses)

	output, _ := cmd.Flags().GetString("output")
	output = strings.TrimSpace(output)
	switch output {
	case "", "wide":
		return printOrganizationsTable(ioStreams.Out, memberships.Items, statuses, output == "wide")
	case "name":
		for _, m := range memberships.Items {
			fmt.Fprintf(ioStreams.Out, "organization/%s\n", m.Spec.OrganizationRef.Name)
		}
		return nil
	case "yaml":
		return (&printers.YAMLPrinter{}).PrintObj(&memberships, ioStreams.Out)
	case "json":
		return (&printers.JSONPrinter{}).PrintObj(&memberships, ioStreams.Out)
	default:
		return fmt.Errorf("output format %q is not supported for organizations; use table, wide, yaml, json, or name", output)
	}
}

func organizationNameFilter(args []string) string {
	if len(args) < 2 {
		return ""
	}
	return args[1]
}

func annotateMemberships(
	list *resourcemanagerv1alpha1.OrganizationMembershipList,
	statuses map[string]onboarding.Result,
) {
	for i := range list.Items {
		m := &list.Items[i]
		orgID := m.Spec.OrganizationRef.Name
		result, ok := statuses[orgID]
		if !ok {
			continue
		}
		if m.Annotations == nil {
			m.Annotations = map[string]string{}
		}
		m.Annotations[onboardingStatusAnnotation] = onboarding.ColumnLabel(result)
		if result.Reason != "" {
			m.Annotations[onboardingReasonAnnotation] = result.Reason
		}
		if result.ActionURL != "" && result.State != onboarding.Complete {
			m.Annotations[onboardingActionAnnotation] = result.ActionURL
		}
	}
}

func printOrganizationsTable(
	w io.Writer,
	items []resourcemanagerv1alpha1.OrganizationMembership,
	statuses map[string]onboarding.Result,
	wide bool,
) error {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Spec.OrganizationRef.Name < items[j].Spec.OrganizationRef.Name
	})

	var tbl table.Table
	if wide {
		tbl = table.New("ORGANIZATION", "DISPLAY NAME", "STATUS", "USER", "AGE")
	} else {
		tbl = table.New("ORGANIZATION", "DISPLAY NAME", "STATUS", "AGE")
	}
	tbl.WithWriter(w)

	var incomplete []onboarding.Result
	now := time.Now()
	for _, m := range items {
		orgID := m.Spec.OrganizationRef.Name
		displayName := m.Status.Organization.DisplayName
		if displayName == "" {
			displayName = orgID
		}
		onboardingLabel := "unknown"
		if result, ok := statuses[orgID]; ok {
			onboardingLabel = onboarding.ColumnLabel(result)
			if result.State != onboarding.Complete {
				incomplete = append(incomplete, result)
			}
		}
		age := "<unknown>"
		if !m.CreationTimestamp.IsZero() {
			age = duration.HumanDuration(now.Sub(m.CreationTimestamp.Time))
		}
		if wide {
			user := m.Status.User.Email
			if user == "" {
				user = m.Spec.UserRef.Name
			}
			tbl.AddRow(orgID, displayName, onboardingLabel, user, age)
		} else {
			tbl.AddRow(orgID, displayName, onboardingLabel, age)
		}
	}
	tbl.Print()

	if len(incomplete) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Some organizations still need setup before you can use them with datumctl:")
		seenURLs := map[string]bool{}
		for _, result := range incomplete {
			if seenURLs[result.OrgID] {
				continue
			}
			seenURLs[result.OrgID] = true
			name := result.OrgDisplayName
			if name == "" {
				name = result.OrgID
			}
			fmt.Fprintf(w, "  %s (%s)\n", name, onboarding.ColumnLabel(result))
			if result.ActionURL != "" {
				fmt.Fprintf(w, "    %s\n", result.ActionURL)
			}
		}
	}
	return nil
}
