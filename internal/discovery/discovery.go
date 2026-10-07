package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	resourcemanagerv1alpha1 "go.miloapis.com/milo/pkg/apis/resourcemanager/v1alpha1"
	"golang.org/x/oauth2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.datum.net/datumctl/internal/authutil"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/miloapi"
)

const displayNameAnnotation = "kubernetes.io/display-name"

// maxConcurrentOrgs bounds how many org control planes are queried at once
// when listing projects.
const maxConcurrentOrgs = 8

// ErrNotFound is returned when a project does not exist or the user cannot
// see it.
var ErrNotFound = errors.New("not found")

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

// API is the part of the Datum Cloud API used to find a user's organizations
// and projects.
type API interface {
	ListOrgs(ctx context.Context) ([]DiscoveredOrg, error)
	ListProjects(ctx context.Context, orgID string) ([]DiscoveredProject, error)
	// GetProject returns ErrNotFound when the project does not exist or is
	// not visible to the user.
	GetProject(ctx context.Context, orgID, projectID string) (*DiscoveredProject, error)
}

// ForSession returns an API client authenticated as the given session. It is a
// variable so tests can substitute a fake.
var ForSession = func(ctx context.Context, session *datumconfig.Session) (API, error) {
	tknSrc, err := authutil.GetTokenSourceForUser(ctx, session.UserKey)
	if err != nil {
		return nil, fmt.Errorf("get token source: %w", err)
	}
	userID, err := authutil.GetUserIDFromTokenForUser(session.UserKey)
	if err != nil {
		return nil, fmt.Errorf("get user ID: %w", err)
	}
	return NewAPI(datumconfig.StripScheme(session.Endpoint.Server), tknSrc, userID), nil
}

// NewAPI returns an API client for the given endpoint and user.
func NewAPI(apiHostname string, tokenSource oauth2.TokenSource, userID string) API {
	return &remoteAPI{apiHostname: apiHostname, tokenSource: tokenSource, userID: userID}
}

type remoteAPI struct {
	apiHostname string
	tokenSource oauth2.TokenSource
	userID      string
}

func (a *remoteAPI) ListOrgs(ctx context.Context) ([]DiscoveredOrg, error) {
	c, err := newClient(miloapi.UserControlPlaneURL(a.apiHostname, a.userID), a.tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create user control-plane client: %w", err)
	}

	var memberships resourcemanagerv1alpha1.OrganizationMembershipList
	if err := c.List(ctx, &memberships); err != nil {
		return nil, fmt.Errorf("list organization memberships: %w", err)
	}

	orgs := make([]DiscoveredOrg, 0, len(memberships.Items))
	for _, m := range memberships.Items {
		displayName := m.Status.Organization.DisplayName
		if displayName == "" {
			displayName = m.Spec.OrganizationRef.Name
		}
		orgs = append(orgs, DiscoveredOrg{Name: m.Spec.OrganizationRef.Name, DisplayName: displayName})
	}
	return orgs, nil
}

func (a *remoteAPI) ListProjects(ctx context.Context, orgID string) ([]DiscoveredProject, error) {
	c, err := newClient(miloapi.OrgControlPlaneURL(a.apiHostname, orgID), a.tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create org control-plane client for %s: %w", orgID, err)
	}

	var list resourcemanagerv1alpha1.ProjectList
	if err := c.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("list projects for org %s: %w", orgID, err)
	}

	projects := make([]DiscoveredProject, 0, len(list.Items))
	for i := range list.Items {
		projects = append(projects, toDiscoveredProject(&list.Items[i], orgID))
	}
	return projects, nil
}

func (a *remoteAPI) GetProject(ctx context.Context, orgID, projectID string) (*DiscoveredProject, error) {
	c, err := newClient(miloapi.OrgControlPlaneURL(a.apiHostname, orgID), a.tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create org control-plane client for %s: %w", orgID, err)
	}

	var p resourcemanagerv1alpha1.Project
	if err := c.Get(ctx, client.ObjectKey{Name: projectID}, &p); err != nil {
		if apierrors.IsNotFound(err) || apierrors.IsForbidden(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get project %s/%s: %w", orgID, projectID, err)
	}
	dp := toDiscoveredProject(&p, orgID)
	return &dp, nil
}

func toDiscoveredProject(p *resourcemanagerv1alpha1.Project, orgID string) DiscoveredProject {
	displayName := p.Annotations[displayNameAnnotation]
	if displayName == "" {
		displayName = p.Name
	}
	return DiscoveredProject{Name: p.Name, DisplayName: displayName, OrgName: orgID}
}

func newClient(host string, tokenSource oauth2.TokenSource) (client.Client, error) {
	scheme := runtime.NewScheme()
	if err := resourcemanagerv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("add resourcemanager types to scheme: %w", err)
	}
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

// OrgFailure records an organization whose projects could not be listed.
type OrgFailure struct {
	OrgID string
	Err   error
}

// List fetches every organization the user belongs to and the projects in
// each. Organizations are queried concurrently; an organization whose projects
// cannot be listed is recorded in Failures and the rest are still returned.
// Only a failure to list memberships is returned as an error.
func List(ctx context.Context, api API, sessionName string) (*Directory, error) {
	orgs, err := api.ListOrgs(ctx)
	if err != nil {
		return nil, err
	}
	return listProjects(ctx, api, sessionName, orgs), nil
}

func listProjects(ctx context.Context, api API, sessionName string, orgs []DiscoveredOrg) *Directory {
	dir := &Directory{Session: sessionName, Orgs: orgs}

	results := make([][]DiscoveredProject, len(orgs))
	errs := make([]error, len(orgs))
	sem := make(chan struct{}, maxConcurrentOrgs)
	var wg sync.WaitGroup
	for i, o := range orgs {
		wg.Add(1)
		go func(i int, orgID string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i], errs[i] = api.ListProjects(ctx, orgID)
		}(i, o.Name)
	}
	wg.Wait()

	for i, o := range orgs {
		if errs[i] != nil {
			dir.Failures = append(dir.Failures, OrgFailure{OrgID: o.Name, Err: errs[i]})
			continue
		}
		projects := results[i]
		sort.Slice(projects, func(a, b int) bool { return projects[a].Name < projects[b].Name })
		dir.Projects = append(dir.Projects, projects...)
	}
	return dir
}

// Lookup resolves query to a context in the given session against the live
// API. An "org/project" ref costs a single request when it names a project by
// ID; display names and bare project IDs fall back to listing. Returns
// ErrNotFound when nothing matches. The returned Directory holds whatever
// display names were fetched along the way.
func Lookup(ctx context.Context, api API, sessionName, query string) (*datumconfig.DiscoveredContext, *Directory, error) {
	orgPart, projPart, hasSlash := strings.Cut(query, "/")

	if hasSlash {
		p, err := api.GetProject(ctx, orgPart, projPart)
		if err == nil {
			dir := &Directory{Session: sessionName, Projects: []DiscoveredProject{*p}}
			return dir.projectContext(orgPart, p.Name), dir, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return nil, nil, err
		}
	}

	orgs, err := api.ListOrgs(ctx)
	if err != nil {
		return nil, nil, err
	}

	if !hasSlash {
		for _, o := range orgs {
			if o.Name == query {
				dir := &Directory{Session: sessionName, Orgs: orgs}
				return dir.orgContext(o.Name), dir, nil
			}
		}
	}

	candidates := orgs
	if hasSlash {
		candidates = nil
		ids := (&Directory{Orgs: orgs}).resolveOrgIDs(orgPart)
		ids = appendUnique(ids, orgPart)
		for _, o := range orgs {
			for _, id := range ids {
				if o.Name == id {
					candidates = append(candidates, o)
				}
			}
		}
	}

	dir := listProjects(ctx, api, sessionName, candidates)
	dir.Orgs = orgs
	if resolved := dir.Resolve(query); resolved != nil {
		return resolved, dir, nil
	}
	return nil, dir, ErrNotFound
}
