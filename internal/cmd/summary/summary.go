// Package summary provides the summary command for displaying user resources.
package summary

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/client"
)

// Options holds the configuration for the summary command.
type Options struct {
	User   string // User identifier (ID, email, or name) - empty means current user
	Output io.Writer
}

// NewCmdSummary creates a new summary command.
func NewCmdSummary() *cobra.Command {
	opts := &Options{
		Output: os.Stdout,
	}

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Display a summary of user's projects and resources",
		Long: `Display a summary of a user's organizations, projects, and resources.

By default, shows the summary for the currently logged-in user.
Use --user to specify a different user by ID, email, or name (partial matches supported).

Examples:
  # Show summary for current user
  datumctl summary

  # Show summary for a specific user by email
  datumctl summary --user john@example.com

  # Show summary for a user by partial name match
  datumctl summary --user smith

  # Show summary for a user by ID
  datumctl summary --user 357322048915119892`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSummary(cmd.Context(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.User, "user", "", "User identifier (ID, email, or name) - supports partial matches")

	return cmd
}

// UserInfo holds user details.
type UserInfo struct {
	ID         string
	Email      string
	GivenName  string
	FamilyName string
	State      string
	Approval   string
}

// OrgInfo holds organization membership details.
type OrgInfo struct {
	Name        string
	DisplayName string
	Type        string
	Namespace   string
}

// ProjectInfo holds project details.
type ProjectInfo struct {
	Name         string
	Organization string
	Ready        bool
}

// ProxyInfo holds HTTP proxy details.
type ProxyInfo struct {
	Name       string
	Project    string
	Hostname   string
	Programmed bool
	Backend    string
}

func runSummary(ctx context.Context, opts *Options) error {
	// Create a platform-wide client for user lookups
	platformClient, err := newPlatformClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create platform client: %w", err)
	}

	// Determine which user to show
	var targetUserID string
	var userInfo *UserInfo

	if opts.User == "" {
		// Use current logged-in user
		targetUserID, err = authutil.GetUserIDFromToken(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}
		userInfo, err = getUserByID(ctx, platformClient, targetUserID)
		if err != nil {
			return fmt.Errorf("failed to get user info: %w", err)
		}
	} else {
		// Look up user by the provided identifier
		userInfo, err = findUser(ctx, platformClient, opts.User)
		if err != nil {
			return err
		}
		targetUserID = userInfo.ID
	}

	// Get organization memberships for the user
	orgs, err := getOrganizationsForUser(ctx, platformClient, targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}

	// Get projects for each organization
	var projects []ProjectInfo
	for _, org := range orgs {
		orgProjects, err := getProjectsForOrganization(ctx, org.Name)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(opts.Output, "Warning: failed to get projects for org %s: %v\n", org.DisplayName, err)
			continue
		}
		projects = append(projects, orgProjects...)
	}

	// Get HTTP proxies for each project
	var proxies []ProxyInfo
	for _, proj := range projects {
		projProxies, err := getProxiesForProject(ctx, proj.Name)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(opts.Output, "Warning: failed to get proxies for project %s: %v\n", proj.Name, err)
			continue
		}
		proxies = append(proxies, projProxies...)
	}

	// Print the summary
	printSummary(opts.Output, userInfo, orgs, projects, proxies)

	return nil
}

// newPlatformClient creates a K8s client for platform-wide operations.
func newPlatformClient(ctx context.Context) (*client.K8sClient, error) {
	tknSrc, err := authutil.GetTokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get token source: %w", err)
	}

	apiHostname, err := authutil.GetAPIHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get API hostname: %w", err)
	}

	cfg := &rest.Config{
		Host: fmt.Sprintf("https://%s", apiHostname),
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{Source: tknSrc, Base: rt}
		},
	}

	return client.NewK8sFromRESTConfig(cfg)
}

// getUserByID fetches user details by ID.
func getUserByID(ctx context.Context, k8sClient *client.K8sClient, userID string) (*UserInfo, error) {
	result, err := k8sClient.Get(ctx, client.GetOptions{
		Kind:       "User",
		APIVersion: "iam.miloapis.com/v1alpha1",
		Name:       userID,
	})
	if err != nil {
		return nil, err
	}

	return extractUserInfo(result), nil
}

// findUser searches for a user by ID, email, or name (with partial matching).
func findUser(ctx context.Context, k8sClient *client.K8sClient, query string) (*UserInfo, error) {
	// List all users
	result, err := k8sClient.List(ctx, client.ListOptions{
		Kind:       "User",
		APIVersion: "iam.miloapis.com/v1alpha1",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	queryLower := strings.ToLower(query)
	var matches []*UserInfo

	for _, item := range result.Items {
		info := extractUserInfo(&item)

		// Check for exact ID match first
		if info.ID == query {
			return info, nil
		}

		// Check for partial matches on email, given name, or family name
		emailLower := strings.ToLower(info.Email)
		givenLower := strings.ToLower(info.GivenName)
		familyLower := strings.ToLower(info.FamilyName)
		fullNameLower := strings.ToLower(info.GivenName + " " + info.FamilyName)

		if strings.Contains(emailLower, queryLower) ||
			strings.Contains(givenLower, queryLower) ||
			strings.Contains(familyLower, queryLower) ||
			strings.Contains(fullNameLower, queryLower) {
			matches = append(matches, info)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no user found matching %q", query)
	}

	if len(matches) > 1 {
		// Print all matches to help the user be more specific
		var matchDetails []string
		for _, m := range matches {
			matchDetails = append(matchDetails, fmt.Sprintf("  - %s %s (%s) [ID: %s]", m.GivenName, m.FamilyName, m.Email, m.ID))
		}
		return nil, fmt.Errorf("multiple users found matching %q:\n%s\nPlease be more specific or use the user ID", query, strings.Join(matchDetails, "\n"))
	}

	return matches[0], nil
}

// extractUserInfo extracts user info from an unstructured object.
func extractUserInfo(u *unstructured.Unstructured) *UserInfo {
	spec, _, _ := unstructured.NestedMap(u.Object, "spec")
	status, _, _ := unstructured.NestedMap(u.Object, "status")

	email, _ := spec["email"].(string)
	givenName, _ := spec["givenName"].(string)
	familyName, _ := spec["familyName"].(string)
	state, _ := status["state"].(string)
	approval, _ := status["registrationApproval"].(string)

	return &UserInfo{
		ID:         u.GetName(),
		Email:      email,
		GivenName:  givenName,
		FamilyName: familyName,
		State:      state,
		Approval:   approval,
	}
}

// getOrganizationsForUser fetches organization memberships for a user.
func getOrganizationsForUser(ctx context.Context, k8sClient *client.K8sClient, userID string) ([]OrgInfo, error) {
	result, err := k8sClient.List(ctx, client.ListOptions{
		Kind:       "OrganizationMembership",
		APIVersion: "resourcemanager.miloapis.com/v1alpha1",
	})
	if err != nil {
		return nil, err
	}

	var orgs []OrgInfo
	memberName := fmt.Sprintf("membership-%s", userID)

	for _, item := range result.Items {
		// Check if this membership is for our user
		if item.GetName() != memberName {
			continue
		}

		status, _, _ := unstructured.NestedMap(item.Object, "status")
		orgInfo, _, _ := unstructured.NestedMap(status, "organization")

		displayName, _ := orgInfo["displayName"].(string)
		orgType, _ := orgInfo["type"].(string)

		spec, _, _ := unstructured.NestedMap(item.Object, "spec")
		orgRef, _, _ := unstructured.NestedMap(spec, "organizationRef")
		orgName, _ := orgRef["name"].(string)

		orgs = append(orgs, OrgInfo{
			Name:        orgName,
			DisplayName: displayName,
			Type:        orgType,
			Namespace:   item.GetNamespace(),
		})
	}

	return orgs, nil
}

// getProjectsForOrganization fetches projects for an organization.
func getProjectsForOrganization(ctx context.Context, orgName string) ([]ProjectInfo, error) {
	orgClient, err := client.NewForOrg(ctx, orgName, "default")
	if err != nil {
		return nil, err
	}

	result, err := orgClient.List(ctx, client.ListOptions{
		Kind:       "Project",
		APIVersion: "resourcemanager.miloapis.com/v1alpha1",
	})
	if err != nil {
		return nil, err
	}

	var projects []ProjectInfo
	for _, item := range result.Items {
		conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
		ready := false
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if cond["type"] == "Ready" && cond["status"] == "True" {
				ready = true
				break
			}
		}

		projects = append(projects, ProjectInfo{
			Name:         item.GetName(),
			Organization: orgName,
			Ready:        ready,
		})
	}

	return projects, nil
}

// getProxiesForProject fetches HTTP proxies for a project.
func getProxiesForProject(ctx context.Context, projectName string) ([]ProxyInfo, error) {
	projClient, err := client.NewForProject(ctx, projectName, "default")
	if err != nil {
		return nil, err
	}

	result, err := projClient.List(ctx, client.ListOptions{
		Kind:       "HTTPProxy",
		APIVersion: "networking.datumapis.com/v1alpha",
	})
	if err != nil {
		return nil, err
	}

	var proxies []ProxyInfo
	for _, item := range result.Items {
		// Get hostname from status
		hostnames, _, _ := unstructured.NestedStringSlice(item.Object, "status", "hostnames")
		hostname := ""
		if len(hostnames) > 0 {
			hostname = hostnames[0]
		}

		// Check if programmed
		conditions, _, _ := unstructured.NestedSlice(item.Object, "status", "conditions")
		programmed := false
		for _, c := range conditions {
			cond, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			if cond["type"] == "Programmed" && cond["status"] == "True" {
				programmed = true
				break
			}
		}

		// Get backend endpoint
		backend := ""
		rules, _, _ := unstructured.NestedSlice(item.Object, "spec", "rules")
		if len(rules) > 0 {
			rule, ok := rules[0].(map[string]interface{})
			if ok {
				backends, _, _ := unstructured.NestedSlice(rule, "backends")
				if len(backends) > 0 {
					be, ok := backends[0].(map[string]interface{})
					if ok {
						backend, _ = be["endpoint"].(string)
					}
				}
			}
		}

		proxies = append(proxies, ProxyInfo{
			Name:       item.GetName(),
			Project:    projectName,
			Hostname:   hostname,
			Programmed: programmed,
			Backend:    backend,
		})
	}

	return proxies, nil
}

// printSummary prints the formatted summary output.
func printSummary(w io.Writer, user *UserInfo, orgs []OrgInfo, projects []ProjectInfo, proxies []ProxyInfo) {
	// User section
	fmt.Fprintln(w, "## User")
	fmt.Fprintf(w, "Name:     %s %s\n", user.GivenName, user.FamilyName)
	fmt.Fprintf(w, "ID:       %s\n", user.ID)
	fmt.Fprintf(w, "Email:    %s\n", user.Email)
	fmt.Fprintf(w, "State:    %s\n", user.State)
	fmt.Fprintf(w, "Approval: %s\n", user.Approval)
	fmt.Fprintln(w)

	// Organizations section
	fmt.Fprintf(w, "## Organizations (%d)\n", len(orgs))
	if len(orgs) == 0 {
		fmt.Fprintln(w, "No organizations found.")
	} else {
		tbl := table.New("NAME", "DISPLAY NAME", "TYPE").WithWriter(w)
		for _, org := range orgs {
			tbl.AddRow(org.Name, org.DisplayName, org.Type)
		}
		tbl.Print()
	}
	fmt.Fprintln(w)

	// Projects section
	fmt.Fprintf(w, "## Projects (%d)\n", len(projects))
	if len(projects) == 0 {
		fmt.Fprintln(w, "No projects found.")
	} else {
		tbl := table.New("NAME", "ORGANIZATION", "READY").WithWriter(w)
		for _, proj := range projects {
			ready := "False"
			if proj.Ready {
				ready = "True"
			}
			tbl.AddRow(proj.Name, proj.Organization, ready)
		}
		tbl.Print()
	}
	fmt.Fprintln(w)

	// HTTP Proxies section
	fmt.Fprintf(w, "## HTTP Proxies (%d)\n", len(proxies))
	if len(proxies) == 0 {
		fmt.Fprintln(w, "No HTTP proxies found.")
	} else {
		tbl := table.New("NAME", "PROJECT", "HOSTNAME", "PROGRAMMED", "BACKEND").WithWriter(w)
		for _, proxy := range proxies {
			programmed := "False"
			if proxy.Programmed {
				programmed = "True"
			}
			tbl.AddRow(proxy.Name, proxy.Project, proxy.Hostname, programmed, proxy.Backend)
		}
		tbl.Print()
	}
}
