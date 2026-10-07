package discovery

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

const testSession = "alice@example.com@api.datum.net"

func TestList_SkipsOrgsWhoseProjectsFail(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{
		orgs: []DiscoveredOrg{
			{Name: "org-a", DisplayName: "A"},
			{Name: "org-broken", DisplayName: "Broken"},
			{Name: "org-b", DisplayName: "B"},
		},
		projects: map[string][]DiscoveredProject{
			"org-a": {{Name: "proj-2", OrgName: "org-a"}, {Name: "proj-1", OrgName: "org-a"}},
			"org-b": {{Name: "proj-3", OrgName: "org-b"}},
		},
		projectErrs: map[string]error{"org-broken": errBoom},
	}

	dir, err := List(context.Background(), api, testSession)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	var refs []string
	for _, c := range dir.Contexts() {
		refs = append(refs, c.Ref())
		if c.Session != testSession || c.Name != c.QualifiedName() {
			t.Errorf("context %+v not qualified with session", c)
		}
	}
	want := "org-a,org-broken,org-b,org-a/proj-1,org-a/proj-2,org-b/proj-3"
	if got := strings.Join(refs, ","); got != want {
		t.Errorf("contexts = %s, want %s", got, want)
	}

	if len(dir.Failures) != 1 || dir.Failures[0].OrgID != "org-broken" {
		t.Fatalf("Failures = %+v, want org-broken", dir.Failures)
	}
	var buf bytes.Buffer
	dir.WarnFailures(&buf)
	if !strings.Contains(buf.String(), "org-broken") || !strings.Contains(buf.String(), "boom") {
		t.Errorf("warning = %q", buf.String())
	}
}

func TestList_FailsWhenMembershipsFail(t *testing.T) {
	t.Parallel()

	_, err := List(context.Background(), &fakeAPI{orgsErr: errBoom}, testSession)
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

// A project created after any earlier listing resolves with a single direct
// lookup (#308).
func TestLookup_ProjectByIDIsOneCall(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{
		projects: map[string][]DiscoveredProject{
			"datum": {{Name: "project-ll2rm", DisplayName: "test-add-project", OrgName: "datum"}},
		},
	}

	got, dir, err := Lookup(context.Background(), api, testSession, "datum/project-ll2rm")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.Ref() != "datum/project-ll2rm" || got.Session != testSession || got.Namespace == "" {
		t.Errorf("context = %+v", got)
	}
	if desc := dir.Describe(got); desc != "project test-add-project in datum (datum/project-ll2rm)" {
		t.Errorf("Describe = %q", desc)
	}
	if len(api.calls) != 1 {
		t.Errorf("calls = %v, want a single GetProject", api.calls)
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()

	api := &fakeAPI{
		orgs: []DiscoveredOrg{
			{Name: "org-acme", DisplayName: "Acme Corp"},
			{Name: "org-datum", DisplayName: "Datum Technology, Inc."},
		},
		projects: map[string][]DiscoveredProject{
			"org-acme": {
				{Name: "proj-infra", DisplayName: "Infrastructure", OrgName: "org-acme"},
				{Name: "proj-web", DisplayName: "Web App", OrgName: "org-acme"},
			},
			"org-datum": {{Name: "proj-dc", DisplayName: "datum-cloud", OrgName: "org-datum"}},
		},
	}

	tests := []struct {
		query   string
		wantRef string
	}{
		{query: "org-acme", wantRef: "org-acme"},
		{query: "proj-dc", wantRef: "org-datum/proj-dc"},
		{query: "Acme Corp", wantRef: "org-acme"},
		{query: "Infrastructure", wantRef: "org-acme/proj-infra"},
		{query: "Acme Corp/Infrastructure", wantRef: "org-acme/proj-infra"},
		{query: "org-acme/Web App", wantRef: "org-acme/proj-web"},
		{query: "Acme Corp/proj-web", wantRef: "org-acme/proj-web"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, _, err := Lookup(context.Background(), api, testSession, tt.query)
			if err != nil {
				t.Fatalf("Lookup(%q): %v", tt.query, err)
			}
			if got.Ref() != tt.wantRef {
				t.Errorf("Lookup(%q) = %q, want %q", tt.query, got.Ref(), tt.wantRef)
			}
		})
	}

	for _, q := range []string{"nope", "org-acme/nope", "nope/proj-web"} {
		if _, _, err := Lookup(context.Background(), api, testSession, q); !errors.Is(err, ErrNotFound) {
			t.Errorf("Lookup(%q) err = %v, want ErrNotFound", q, err)
		}
	}
}

func TestLookup_PropagatesAPIErrors(t *testing.T) {
	t.Parallel()

	if _, _, err := Lookup(context.Background(), &fakeAPI{orgsErr: errBoom}, testSession, "org-acme"); !errors.Is(err, errBoom) {
		t.Errorf("err = %v, want boom", err)
	}
}
