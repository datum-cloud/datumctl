package discovery

import (
	"context"
	"errors"
	"sync"
)

type fakeAPI struct {
	orgs        []DiscoveredOrg
	orgsErr     error
	projects    map[string][]DiscoveredProject
	projectErrs map[string]error

	mu    sync.Mutex
	calls []string
}

func (f *fakeAPI) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, call)
}

func (f *fakeAPI) ListOrgs(context.Context) ([]DiscoveredOrg, error) {
	f.record("ListOrgs")
	return f.orgs, f.orgsErr
}

func (f *fakeAPI) ListProjects(_ context.Context, orgID string) ([]DiscoveredProject, error) {
	f.record("ListProjects " + orgID)
	if err := f.projectErrs[orgID]; err != nil {
		return nil, err
	}
	return append([]DiscoveredProject(nil), f.projects[orgID]...), nil
}

func (f *fakeAPI) GetProject(_ context.Context, orgID, projectID string) (*DiscoveredProject, error) {
	f.record("GetProject " + orgID + "/" + projectID)
	for _, p := range f.projects[orgID] {
		if p.Name == projectID {
			return &p, nil
		}
	}
	return nil, ErrNotFound
}

var errBoom = errors.New("boom")
