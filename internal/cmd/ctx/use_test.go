package ctx

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
	customerrors "go.datum.net/datumctl/internal/errors"
)

type fakeAPI struct {
	orgs        []discovery.DiscoveredOrg
	projects    map[string][]discovery.DiscoveredProject
	projectErrs map[string]error
}

func (f *fakeAPI) ListOrgs(context.Context) ([]discovery.DiscoveredOrg, error) {
	return f.orgs, nil
}

func (f *fakeAPI) ListProjects(_ context.Context, orgID string) ([]discovery.DiscoveredProject, error) {
	if err := f.projectErrs[orgID]; err != nil {
		return nil, err
	}
	return append([]discovery.DiscoveredProject(nil), f.projects[orgID]...), nil
}

func (f *fakeAPI) GetProject(_ context.Context, orgID, projectID string) (*discovery.DiscoveredProject, error) {
	for _, p := range f.projects[orgID] {
		if p.Name == projectID {
			return &p, nil
		}
	}
	return nil, discovery.ErrNotFound
}

// setup writes cfg to a temporary home and routes each session's API calls to
// its fake.
func setup(t *testing.T, cfg *datumconfig.ConfigV1Beta1, apis map[string]*fakeAPI) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	orig := discovery.ForSession
	discovery.ForSession = func(_ context.Context, s *datumconfig.Session) (discovery.API, error) {
		if api, ok := apis[s.Name]; ok {
			return api, nil
		}
		return nil, errors.New("no fake for " + s.Name)
	}
	t.Cleanup(func() { discovery.ForSession = orig })
}

func captureStdio(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	origOut, origErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = origOut, origErr }()
	fn()
	outW.Close()
	errW.Close()
	o, _ := io.ReadAll(outR)
	e, _ := io.ReadAll(errR)
	return string(o), string(e)
}

const prodSession = "felix@example.com@api.datum.net"

func prodConfig() *datumconfig.ConfigV1Beta1 {
	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{Name: prodSession, UserEmail: "felix@example.com", Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}},
	}
	cfg.ActiveSession = prodSession
	return cfg
}

// A project created after the user last listed contexts is usable right away
// (#308).
func TestUseNewProjectNeverListedBefore(t *testing.T) {
	setup(t, prodConfig(), map[string]*fakeAPI{
		prodSession: {projects: map[string][]discovery.DiscoveredProject{
			"datum": {{Name: "project-ll2rm", DisplayName: "test-add-project", OrgName: "datum"}},
		}},
	})

	var err error
	out, _ := captureStdio(t, func() { err = runUse(nil, []string{"datum/project-ll2rm"}) })
	if err != nil {
		t.Fatalf("runUse: %v", err)
	}
	if !strings.Contains(out, "Switched to project test-add-project in datum (datum/project-ll2rm)") {
		t.Errorf("output = %q", out)
	}

	cfg, err := datumconfig.LoadAuto()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	want := datumconfig.QualifiedContextName(prodSession, "datum/project-ll2rm")
	if cfg.CurrentContext != want {
		t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, want)
	}
	if got := cfg.CurrentContextEntry(); got == nil || got.ProjectID != "project-ll2rm" {
		t.Errorf("CurrentContextEntry = %+v", got)
	}
	if got := cfg.SessionByName(prodSession).LastContext; got != want {
		t.Errorf("LastContext = %q, want %q", got, want)
	}
}

func TestUseNotFound(t *testing.T) {
	setup(t, prodConfig(), map[string]*fakeAPI{
		prodSession: {orgs: []discovery.DiscoveredOrg{{Name: "datum"}}},
	})

	err := runUse(nil, []string{"datum/missing"})
	userErr, ok := customerrors.IsUserError(err)
	if !ok {
		t.Fatalf("expected *UserError, got %T: %v", err, err)
	}
	if !strings.Contains(userErr.Message, `"datum/missing" not found`) {
		t.Errorf("Message = %q", userErr.Message)
	}

	cfg, _ := datumconfig.LoadAuto()
	if cfg.CurrentContext != "" {
		t.Errorf("CurrentContext = %q, want unchanged", cfg.CurrentContext)
	}
}

// A context owned by another session must point at a switch command that
// works, including --endpoint when that session's email spans endpoints.
func TestUseHintForContextInOtherSession(t *testing.T) {
	const email = "swells@datum.net"
	prod := datumconfig.SessionName(email, "api.datum.net")
	staging := datumconfig.SessionName(email, "api.staging.env.datum.net")
	solo := datumconfig.SessionName("solo@example.com", "api.datum.net")

	tests := []struct {
		name     string
		active   string
		ref      string
		wantHint string
	}{
		{
			name:     "shared email includes endpoint",
			active:   prod,
			ref:      "org-staging",
			wantHint: "Run 'datumctl auth switch " + email + " --endpoint api.staging.env.datum.net' first, then 'datumctl ctx use org-staging'.",
		},
		{
			name:     "unique email stays bare",
			active:   prod,
			ref:      "org-solo",
			wantHint: "Run 'datumctl auth switch solo@example.com' first, then 'datumctl ctx use org-solo'.",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := datumconfig.NewV1Beta1()
			cfg.Sessions = []datumconfig.Session{
				{Name: prod, UserEmail: email, Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}},
				{Name: staging, UserEmail: email, Endpoint: datumconfig.Endpoint{Server: "https://api.staging.env.datum.net"}},
				{Name: solo, UserEmail: "solo@example.com", Endpoint: datumconfig.Endpoint{Server: "https://api.datum.net"}},
			}
			cfg.ActiveSession = tc.active
			setup(t, cfg, map[string]*fakeAPI{
				prod:    {orgs: []discovery.DiscoveredOrg{{Name: "org-prod"}}},
				staging: {orgs: []discovery.DiscoveredOrg{{Name: "org-staging"}}},
				solo:    {orgs: []discovery.DiscoveredOrg{{Name: "org-solo"}}},
			})

			err := runUse(nil, []string{tc.ref})
			userErr, ok := customerrors.IsUserError(err)
			if !ok {
				t.Fatalf("expected *UserError, got %T: %v", err, err)
			}
			if !strings.Contains(userErr.Hint, tc.wantHint) {
				t.Errorf("Hint = %q, want %q", userErr.Hint, tc.wantHint)
			}
		})
	}
}

// An organization whose projects cannot be listed is skipped with a warning;
// the rest still list.
func TestListToleratesPerOrgFailure(t *testing.T) {
	setup(t, prodConfig(), map[string]*fakeAPI{
		prodSession: {
			orgs: []discovery.DiscoveredOrg{{Name: "datum", DisplayName: "Datum"}, {Name: "broken"}},
			projects: map[string][]discovery.DiscoveredProject{
				"datum": {{Name: "project-ll2rm", DisplayName: "test-add-project", OrgName: "datum"}},
			},
			projectErrs: map[string]error{"broken": errors.New("forbidden")},
		},
	})

	cmd := &cobra.Command{}
	cmd.Flags().Bool("all", false, "")
	var err error
	out, errOut := captureStdio(t, func() { err = runList(cmd, nil) })
	if err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(out, "datum/project-ll2rm") {
		t.Errorf("stdout missing project:\n%s", out)
	}
	if !strings.Contains(errOut, "could not list projects for organization broken") {
		t.Errorf("stderr missing warning:\n%s", errOut)
	}
}
