package discovery

import (
	"context"
	"fmt"
	"net/http"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.datum.net/datumctl/internal/miloapi"
)

const displayNameAnnotation = "kubernetes.io/display-name"

// DiscoveredOrg represents an organization the user has access to.
type DiscoveredOrg struct {
	Name        string // resource name (org ID)
	DisplayName string // human-friendly name
}

// DiscoveredProject represents a project under an organization.
type DiscoveredProject struct {
	Name        string // resource name (project ID)
	DisplayName string // human-friendly name
	OrgName     string // owning organization resource name
}

// FetchOrgsAndProjects discovers all organizations the user belongs to and
// their projects by querying the API.
func FetchOrgsAndProjects(
	ctx context.Context,
	apiHostname string,
	tokenSource oauth2.TokenSource,
	userID string,
) ([]DiscoveredOrg, []DiscoveredProject, error) {
	scheme := runtime.NewScheme()
	if err := resourcemanagerv1alpha1.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("add resourcemanager types to scheme: %w", err)
	}

	// List OrganizationMemberships from the user's IAM control plane.
	userCPHost := miloapi.UserControlPlaneURL(apiHostname, userID)
	userClient, err := newClient(userCPHost, tokenSource, scheme)
	if err != nil {
		return nil, nil, fmt.Errorf("create user control-plane client: %w", err)
	}

	var memberships resourcemanagerv1alpha1.OrganizationMembershipList
	if err := userClient.List(ctx, &memberships); err != nil {
		return nil, nil, fmt.Errorf("list organization memberships: %w", err)
	}

	var orgs []DiscoveredOrg
	var projects []DiscoveredProject

	for _, m := range memberships.Items {
		orgName := m.Spec.OrganizationRef.Name
		displayName := m.Status.Organization.DisplayName
		if displayName == "" {
			displayName = orgName
		}

		orgs = append(orgs, DiscoveredOrg{
			Name:        orgName,
			DisplayName: displayName,
		})

		// List projects in this org's control plane.
		orgCPHost := miloapi.OrgControlPlaneURL(apiHostname, orgName)
		orgClient, err := newClient(orgCPHost, tokenSource, scheme)
		if err != nil {
			return nil, nil, fmt.Errorf("create org control-plane client for %s: %w", orgName, err)
		}

		var projectList resourcemanagerv1alpha1.ProjectList
		if err := orgClient.List(ctx, &projectList); err != nil {
			return nil, nil, fmt.Errorf("list projects for org %s: %w", orgName, err)
		}

		for _, p := range projectList.Items {
			projDisplayName := p.Annotations[displayNameAnnotation]
			if projDisplayName == "" {
				projDisplayName = p.Name
			}
			projects = append(projects, DiscoveredProject{
				Name:        p.Name,
				DisplayName: projDisplayName,
				OrgName:     orgName,
			})
		}
	}

	return orgs, projects, nil
}

func newClient(host string, tokenSource oauth2.TokenSource, scheme *runtime.Scheme) (client.Client, error) {
	cfg := &rest.Config{
		Host:      host,
		UserAgent: "datumctl",
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauth2.Transport{
				Source: tokenSource,
				Base:   rt,
			}
		},
	}
	return client.New(cfg, client.Options{Scheme: scheme})
}
