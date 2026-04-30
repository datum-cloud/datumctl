package datumconfig

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigV1Beta1RoundTrip verifies marshal/unmarshal round-trip preserves all fields.
func TestConfigV1Beta1RoundTrip(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	original := NewV1Beta1()
	original.CurrentContext = "org-acme/proj-infra"
	original.ActiveSession = "jane@acme.com@api.datum.net"
	original.Sessions = []Session{
		{
			Name:      "jane@acme.com@api.datum.net",
			UserKey:   "key-abc123",
			UserEmail: "jane@acme.com",
			UserName:  "Jane Doe",
			Endpoint: Endpoint{
				Server:       "https://api.datum.net",
				AuthHostname: "auth.datum.net",
			},
			LastContext: "org-acme/proj-infra",
		},
	}
	original.Contexts = []DiscoveredContext{
		{
			Name:           "org-acme",
			Session:        "jane@acme.com@api.datum.net",
			OrganizationID: "org-acme",
		},
		{
			Name:           "org-acme/proj-infra",
			Session:        "jane@acme.com@api.datum.net",
			OrganizationID: "org-acme",
			ProjectID:      "proj-infra",
			Namespace:      "default",
		},
	}

	if err := SaveV1Beta1ToPath(original, path); err != nil {
		t.Fatalf("SaveV1Beta1ToPath: %v", err)
	}

	loaded, err := LoadV1Beta1FromPath(path)
	if err != nil {
		t.Fatalf("LoadV1Beta1FromPath: %v", err)
	}

	if loaded.APIVersion != V1Beta1APIVersion {
		t.Errorf("APIVersion=%q, want %q", loaded.APIVersion, V1Beta1APIVersion)
	}
	if loaded.Kind != DefaultKind {
		t.Errorf("Kind=%q, want %q", loaded.Kind, DefaultKind)
	}
	if loaded.CurrentContext != "org-acme/proj-infra" {
		t.Errorf("CurrentContext=%q, want %q", loaded.CurrentContext, "org-acme/proj-infra")
	}
	if loaded.ActiveSession != "jane@acme.com@api.datum.net" {
		t.Errorf("ActiveSession=%q, want %q", loaded.ActiveSession, "jane@acme.com@api.datum.net")
	}
	if len(loaded.Sessions) != 1 {
		t.Fatalf("Sessions len=%d, want 1", len(loaded.Sessions))
	}
	s := loaded.Sessions[0]
	if s.Name != "jane@acme.com@api.datum.net" {
		t.Errorf("Session.Name=%q, want %q", s.Name, "jane@acme.com@api.datum.net")
	}
	if s.UserKey != "key-abc123" {
		t.Errorf("Session.UserKey=%q, want %q", s.UserKey, "key-abc123")
	}
	if s.UserEmail != "jane@acme.com" {
		t.Errorf("Session.UserEmail=%q, want %q", s.UserEmail, "jane@acme.com")
	}
	if s.UserName != "Jane Doe" {
		t.Errorf("Session.UserName=%q, want %q", s.UserName, "Jane Doe")
	}
	if s.Endpoint.Server != "https://api.datum.net" {
		t.Errorf("Endpoint.Server=%q, want %q", s.Endpoint.Server, "https://api.datum.net")
	}
	if s.LastContext != "org-acme/proj-infra" {
		t.Errorf("Session.LastContext=%q, want %q", s.LastContext, "org-acme/proj-infra")
	}
	if len(loaded.Contexts) != 2 {
		t.Fatalf("Contexts len=%d, want 2", len(loaded.Contexts))
	}
}

// TestSessionByName verifies lookup by session name.
func TestSessionByName(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Sessions = []Session{
		{Name: "alice@api.datum.net", UserEmail: "alice@example.com"},
		{Name: "bob@api.datum.net", UserEmail: "bob@example.com"},
	}

	got := cfg.SessionByName("alice@api.datum.net")
	if got == nil {
		t.Fatal("SessionByName returned nil for known session")
	}
	if got.UserEmail != "alice@example.com" {
		t.Errorf("UserEmail=%q, want %q", got.UserEmail, "alice@example.com")
	}

	missing := cfg.SessionByName("nobody@api.datum.net")
	if missing != nil {
		t.Errorf("expected nil for unknown session, got %+v", missing)
	}
}

// TestContextByName verifies lookup by context name.
func TestContextByName(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Contexts = []DiscoveredContext{
		{Name: "acme-corp", Session: "sess-1", OrganizationID: "org-1"},
		{Name: "acme-corp/web", Session: "sess-1", OrganizationID: "org-1", ProjectID: "proj-web"},
	}

	got := cfg.ContextByName("acme-corp/web")
	if got == nil {
		t.Fatal("ContextByName returned nil for known context")
	}
	if got.ProjectID != "proj-web" {
		t.Errorf("ProjectID=%q, want %q", got.ProjectID, "proj-web")
	}

	missing := cfg.ContextByName("nonexistent")
	if missing != nil {
		t.Errorf("expected nil for unknown context, got %+v", missing)
	}
}

// TestUpsertSession verifies insert and update behavior.
func TestUpsertSession(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()

	// Insert new session.
	cfg.UpsertSession(Session{Name: "sess-1", UserEmail: "user@example.com"})
	if len(cfg.Sessions) != 1 {
		t.Fatalf("Sessions len=%d after insert, want 1", len(cfg.Sessions))
	}

	// Update existing session.
	cfg.UpsertSession(Session{Name: "sess-1", UserEmail: "updated@example.com"})
	if len(cfg.Sessions) != 1 {
		t.Fatalf("Sessions len=%d after update, want 1", len(cfg.Sessions))
	}
	if cfg.Sessions[0].UserEmail != "updated@example.com" {
		t.Errorf("UserEmail after update=%q, want %q", cfg.Sessions[0].UserEmail, "updated@example.com")
	}

	// Insert a second distinct session.
	cfg.UpsertSession(Session{Name: "sess-2", UserEmail: "other@example.com"})
	if len(cfg.Sessions) != 2 {
		t.Fatalf("Sessions len=%d after second insert, want 2", len(cfg.Sessions))
	}
}

// TestUpsertContext verifies insert and update behavior.
func TestUpsertContext(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()

	// Insert new context.
	cfg.UpsertContext(DiscoveredContext{Name: "acme-corp", Session: "sess-1", OrganizationID: "org-1"})
	if len(cfg.Contexts) != 1 {
		t.Fatalf("Contexts len=%d after insert, want 1", len(cfg.Contexts))
	}

	// Update existing context.
	cfg.UpsertContext(DiscoveredContext{Name: "acme-corp", Session: "sess-1", OrganizationID: "org-updated"})
	if len(cfg.Contexts) != 1 {
		t.Fatalf("Contexts len=%d after update, want 1", len(cfg.Contexts))
	}
	if cfg.Contexts[0].OrganizationID != "org-updated" {
		t.Errorf("OrganizationID after update=%q, want %q", cfg.Contexts[0].OrganizationID, "org-updated")
	}

	// Insert a second distinct context.
	cfg.UpsertContext(DiscoveredContext{Name: "acme-corp/web", Session: "sess-1", OrganizationID: "org-1", ProjectID: "proj-web"})
	if len(cfg.Contexts) != 2 {
		t.Fatalf("Contexts len=%d after second insert, want 2", len(cfg.Contexts))
	}
}

// TestRemoveSession verifies that removing a session also removes its contexts
// and clears ActiveSession when it matches.
func TestRemoveSession(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.ActiveSession = "sess-1"
	cfg.Sessions = []Session{
		{Name: "sess-1"},
		{Name: "sess-2"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "acme-corp", Session: "sess-1"},
		{Name: "acme-corp/web", Session: "sess-1"},
		{Name: "other-org", Session: "sess-2"},
	}

	cfg.RemoveSession("sess-1")

	if len(cfg.Sessions) != 1 {
		t.Errorf("Sessions len=%d after remove, want 1", len(cfg.Sessions))
	}
	if cfg.Sessions[0].Name != "sess-2" {
		t.Errorf("remaining session=%q, want %q", cfg.Sessions[0].Name, "sess-2")
	}

	// Both contexts for sess-1 should be removed, the sess-2 one kept.
	if len(cfg.Contexts) != 1 {
		t.Errorf("Contexts len=%d after remove, want 1", len(cfg.Contexts))
	}
	if cfg.Contexts[0].Name != "other-org" {
		t.Errorf("remaining context=%q, want %q", cfg.Contexts[0].Name, "other-org")
	}

	// ActiveSession should be cleared.
	if cfg.ActiveSession != "" {
		t.Errorf("ActiveSession=%q after remove, want empty", cfg.ActiveSession)
	}
}

// TestRemoveSessionDoesNotClearOtherActiveSession verifies ActiveSession is
// preserved when removing a different session.
func TestRemoveSessionDoesNotClearOtherActiveSession(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.ActiveSession = "sess-2"
	cfg.Sessions = []Session{
		{Name: "sess-1"},
		{Name: "sess-2"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "ctx-1", Session: "sess-1"},
	}

	cfg.RemoveSession("sess-1")

	if cfg.ActiveSession != "sess-2" {
		t.Errorf("ActiveSession=%q, want %q", cfg.ActiveSession, "sess-2")
	}
}

// TestHasMultipleEndpoints verifies detection of multiple distinct endpoints.
func TestHasMultipleEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sessions []Session
		want     bool
	}{
		{
			name:     "no sessions",
			sessions: nil,
			want:     false,
		},
		{
			name: "single session",
			sessions: []Session{
				{Endpoint: Endpoint{Server: "https://api.datum.net"}},
			},
			want: false,
		},
		{
			name: "two sessions same endpoint",
			sessions: []Session{
				{Endpoint: Endpoint{Server: "https://api.datum.net"}},
				{Endpoint: Endpoint{Server: "https://api.datum.net"}},
			},
			want: false,
		},
		{
			name: "two sessions different endpoints",
			sessions: []Session{
				{Endpoint: Endpoint{Server: "https://api.datum.net"}},
				{Endpoint: Endpoint{Server: "https://api.staging.datum.net"}},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := NewV1Beta1()
			cfg.Sessions = tt.sessions
			if got := cfg.HasMultipleEndpoints(); got != tt.want {
				t.Errorf("HasMultipleEndpoints()=%v, want %v", got, tt.want)
			}
		})
	}
}

// TestActiveSessionEntry verifies fallback logic: explicit ActiveSession first,
// then session derived from current context.
func TestActiveSessionEntry(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Sessions = []Session{
		{Name: "sess-1", UserEmail: "alice@example.com"},
		{Name: "sess-2", UserEmail: "bob@example.com"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "acme-corp", Session: "sess-1"},
	}

	// No active session and no current context — should return nil.
	got := cfg.ActiveSessionEntry()
	if got != nil {
		t.Errorf("expected nil when no ActiveSession and no CurrentContext, got %+v", got)
	}

	// Set CurrentContext only — should fall back to context's session.
	cfg.CurrentContext = "acme-corp"
	got = cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected session from current context, got nil")
	}
	if got.UserEmail != "alice@example.com" {
		t.Errorf("UserEmail=%q, want %q", got.UserEmail, "alice@example.com")
	}

	// Set explicit ActiveSession — should prefer it over context session.
	cfg.ActiveSession = "sess-2"
	got = cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected session from ActiveSession, got nil")
	}
	if got.UserEmail != "bob@example.com" {
		t.Errorf("UserEmail=%q, want %q", got.UserEmail, "bob@example.com")
	}

	// ActiveSession set but nonexistent — falls back to current context.
	cfg.ActiveSession = "nonexistent"
	got = cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected fallback to current context session, got nil")
	}
	if got.UserEmail != "alice@example.com" {
		t.Errorf("fallback UserEmail=%q, want %q", got.UserEmail, "alice@example.com")
	}
}

// TestSessionNameGeneration verifies the canonical session name format.
func TestSessionNameGeneration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		email        string
		apiHostname  string
		wantName     string
	}{
		{
			name:        "plain hostname",
			email:       "user@example.com",
			apiHostname: "api.datum.net",
			wantName:    "user@example.com@api.datum.net",
		},
		{
			name:        "https scheme stripped",
			email:       "user@example.com",
			apiHostname: "https://api.datum.net",
			wantName:    "user@example.com@api.datum.net",
		},
		{
			name:        "http scheme stripped",
			email:       "user@example.com",
			apiHostname: "http://api.staging.datum.net",
			wantName:    "user@example.com@api.staging.datum.net",
		},
		{
			name:        "trailing slash stripped",
			email:       "user@example.com",
			apiHostname: "https://api.datum.net/",
			wantName:    "user@example.com@api.datum.net",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := SessionName(tt.email, tt.apiHostname)
			if got != tt.wantName {
				t.Errorf("SessionName(%q, %q)=%q, want %q", tt.email, tt.apiHostname, got, tt.wantName)
			}
		})
	}
}

// TestContextsForSession verifies that only contexts belonging to the given
// session are returned.
func TestContextsForSession(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Contexts = []DiscoveredContext{
		{Name: "acme-corp", Session: "sess-1"},
		{Name: "acme-corp/web", Session: "sess-1"},
		{Name: "other-org", Session: "sess-2"},
	}

	result := cfg.ContextsForSession("sess-1")
	if len(result) != 2 {
		t.Fatalf("ContextsForSession(sess-1) len=%d, want 2", len(result))
	}
	for _, ctx := range result {
		if ctx.Session != "sess-1" {
			t.Errorf("unexpected session %q in results for sess-1", ctx.Session)
		}
	}

	result2 := cfg.ContextsForSession("sess-2")
	if len(result2) != 1 {
		t.Fatalf("ContextsForSession(sess-2) len=%d, want 1", len(result2))
	}
	if result2[0].Name != "other-org" {
		t.Errorf("context name=%q, want %q", result2[0].Name, "other-org")
	}

	empty := cfg.ContextsForSession("nonexistent")
	if len(empty) != 0 {
		t.Errorf("expected empty slice for nonexistent session, got %d entries", len(empty))
	}
}

// TestLoadV1Beta1FromPath_MissingFile verifies that a missing file returns a
// fresh default config (not an error).
func TestLoadV1Beta1FromPath_MissingFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	cfg, err := LoadV1Beta1FromPath(path)
	if err != nil {
		t.Fatalf("LoadV1Beta1FromPath: %v", err)
	}
	if cfg.APIVersion != V1Beta1APIVersion {
		t.Errorf("APIVersion=%q, want %q", cfg.APIVersion, V1Beta1APIVersion)
	}
	if cfg.Kind != DefaultKind {
		t.Errorf("Kind=%q, want %q", cfg.Kind, DefaultKind)
	}
}

// TestLoadV1Beta1FromPath_EmptyFile verifies that an empty file returns a
// fresh default config.
func TestLoadV1Beta1FromPath_EmptyFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadV1Beta1FromPath(path)
	if err != nil {
		t.Fatalf("LoadV1Beta1FromPath: %v", err)
	}
	if cfg.APIVersion != V1Beta1APIVersion {
		t.Errorf("APIVersion=%q, want %q", cfg.APIVersion, V1Beta1APIVersion)
	}
}

// TestRef verifies the Ref() helper on DiscoveredContext.
func TestRef(t *testing.T) {
	t.Parallel()

	orgCtx := DiscoveredContext{OrganizationID: "datum", ProjectID: ""}
	if got := orgCtx.Ref(); got != "datum" {
		t.Errorf("org Ref()=%q, want %q", got, "datum")
	}

	projCtx := DiscoveredContext{OrganizationID: "datum", ProjectID: "datum-cloud"}
	if got := projCtx.Ref(); got != "datum/datum-cloud" {
		t.Errorf("project Ref()=%q, want %q", got, "datum/datum-cloud")
	}
}

// TestResolveContext verifies all six matching strategies.
func TestResolveContext(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Contexts = []DiscoveredContext{
		{Name: "datum", OrganizationID: "datum"},
		{Name: "datum/datum-cloud", OrganizationID: "datum", ProjectID: "datum-cloud"},
		{Name: "datum/other-proj", OrganizationID: "datum", ProjectID: "other-proj"},
		{Name: "staging", OrganizationID: "staging"},
		{Name: "staging/my-app", OrganizationID: "staging", ProjectID: "my-app"},
		// A context with a legacy display-name-style name but correct IDs.
		{Name: "Acme Corp/Web App", OrganizationID: "acme", ProjectID: "web-app"},
	}

	tests := []struct {
		name    string
		query   string
		wantRef string // empty means nil expected
	}{
		// 1. Exact name match.
		{name: "exact org name", query: "datum", wantRef: "datum"},
		{name: "exact project name", query: "datum/datum-cloud", wantRef: "datum/datum-cloud"},
		{name: "exact legacy name", query: "Acme Corp/Web App", wantRef: "acme/web-app"},

		// 2. orgID/projectID match (when name differs).
		{name: "orgID/projectID for legacy context", query: "acme/web-app", wantRef: "acme/web-app"},

		// 3. orgID-only match for org contexts.
		{name: "orgID only", query: "staging", wantRef: "staging"},

		// 4. projectID-only match (unambiguous).
		{name: "unique projectID", query: "my-app", wantRef: "staging/my-app"},
		{name: "unique projectID web-app", query: "web-app", wantRef: "acme/web-app"},

		// Ambiguous projectID — appears in zero project contexts with that exact ID.
		// (datum-cloud is unique, so it resolves)
		{name: "unique projectID datum-cloud", query: "datum-cloud", wantRef: "datum/datum-cloud"},

		// No match.
		{name: "no match", query: "nonexistent", wantRef: ""},
		{name: "no match with slash", query: "foo/bar", wantRef: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cfg.ResolveContext(tt.query)
			if tt.wantRef == "" {
				if got != nil {
					t.Errorf("ResolveContext(%q) = %q, want nil", tt.query, got.Ref())
				}
				return
			}
			if got == nil {
				t.Fatalf("ResolveContext(%q) = nil, want %q", tt.query, tt.wantRef)
			}
			if got.Ref() != tt.wantRef {
				t.Errorf("ResolveContext(%q).Ref() = %q, want %q", tt.query, got.Ref(), tt.wantRef)
			}
		})
	}
}

// TestResolveContext_AmbiguousProjectID verifies that ambiguous projectID
// returns nil.
func TestResolveContext_AmbiguousProjectID(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Contexts = []DiscoveredContext{
		{Name: "org-a/shared", OrganizationID: "org-a", ProjectID: "shared"},
		{Name: "org-b/shared", OrganizationID: "org-b", ProjectID: "shared"},
	}

	got := cfg.ResolveContext("shared")
	if got != nil {
		t.Errorf("ResolveContext(\"shared\") should return nil for ambiguous match, got %q", got.Ref())
	}

	// But orgID/projectID should still resolve.
	got = cfg.ResolveContext("org-a/shared")
	if got == nil {
		t.Fatal("ResolveContext(\"org-a/shared\") should resolve")
	}
	if got.Ref() != "org-a/shared" {
		t.Errorf("got %q, want %q", got.Ref(), "org-a/shared")
	}
}

// TestResolveContext_DisplayNameMatching verifies display-name resolution.
func TestResolveContext_DisplayNameMatching(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Cache.Organizations = []CachedOrg{
		{ID: "org-acme", DisplayName: "Acme Corp"},
		{ID: "org-datum", DisplayName: "Datum Technology, Inc."},
	}
	cfg.Cache.Projects = []CachedProject{
		{ID: "proj-infra", DisplayName: "Infrastructure", OrgID: "org-acme"},
		{ID: "proj-web", DisplayName: "Web App", OrgID: "org-acme"},
		{ID: "proj-dc", DisplayName: "datum-cloud", OrgID: "org-datum"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "org-acme", OrganizationID: "org-acme"},
		{Name: "org-acme/proj-infra", OrganizationID: "org-acme", ProjectID: "proj-infra"},
		{Name: "org-acme/proj-web", OrganizationID: "org-acme", ProjectID: "proj-web"},
		{Name: "org-datum", OrganizationID: "org-datum"},
		{Name: "org-datum/proj-dc", OrganizationID: "org-datum", ProjectID: "proj-dc"},
	}

	tests := []struct {
		name    string
		query   string
		wantRef string
	}{
		{name: "org display name only", query: "Acme Corp", wantRef: "org-acme"},
		{name: "project display name only", query: "Infrastructure", wantRef: "org-acme/proj-infra"},
		{name: "org/project both display names", query: "Acme Corp/Infrastructure", wantRef: "org-acme/proj-infra"},
		{name: "orgID/project display name", query: "org-acme/Web App", wantRef: "org-acme/proj-web"},
		{name: "org display name/projectID", query: "Acme Corp/proj-web", wantRef: "org-acme/proj-web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cfg.ResolveContext(tt.query)
			if got == nil {
				t.Fatalf("ResolveContext(%q) = nil, want %q", tt.query, tt.wantRef)
			}
			if got.Ref() != tt.wantRef {
				t.Errorf("ResolveContext(%q).Ref() = %q, want %q", tt.query, got.Ref(), tt.wantRef)
			}
		})
	}
}

// TestResolveContext_AmbiguousDisplayName verifies that ambiguous org/project
// display names return nil instead of silently picking the first match.
func TestResolveContext_AmbiguousDisplayName(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Cache.Organizations = []CachedOrg{
		{ID: "org-a", DisplayName: "Production"},
		{ID: "org-b", DisplayName: "Production"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "org-a", OrganizationID: "org-a"},
		{Name: "org-b", OrganizationID: "org-b"},
	}

	got := cfg.ResolveContext("Production")
	if got != nil {
		t.Errorf("ambiguous org display name should return nil, got %q", got.Ref())
	}
}

// TestResolveContext_IDWinsOverDisplayName verifies that resource IDs always
// take precedence over display names, even when both could match.
func TestResolveContext_IDWinsOverDisplayName(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Cache.Projects = []CachedProject{
		// Project B's DISPLAY NAME collides with project A's ID.
		{ID: "proj-a", DisplayName: "something-else", OrgID: "org-1"},
		{ID: "proj-b", DisplayName: "proj-a", OrgID: "org-1"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "org-1/proj-a", OrganizationID: "org-1", ProjectID: "proj-a"},
		{Name: "org-1/proj-b", OrganizationID: "org-1", ProjectID: "proj-b"},
	}

	// "proj-a" should match proj-a by ID, not proj-b by display name.
	got := cfg.ResolveContext("proj-a")
	if got == nil {
		t.Fatal("ResolveContext(\"proj-a\") = nil, want proj-a")
	}
	if got.ProjectID != "proj-a" {
		t.Errorf("ResolveContext(\"proj-a\") = %q, want proj-a (ID should win over display name)", got.ProjectID)
	}
}

// TestResolveContext_ProjectDisplayNameScopedToOrg verifies that a query like
// "someorg/projname" doesn't match a project with that display name in a
// different org.
func TestResolveContext_ProjectDisplayNameScopedToOrg(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Cache.Projects = []CachedProject{
		{ID: "proj-a", DisplayName: "shared", OrgID: "org-a"},
		{ID: "proj-b", DisplayName: "shared", OrgID: "org-b"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "org-a/proj-a", OrganizationID: "org-a", ProjectID: "proj-a"},
		{Name: "org-b/proj-b", OrganizationID: "org-b", ProjectID: "proj-b"},
	}

	// "org-a/shared" should resolve to proj-a, not proj-b.
	got := cfg.ResolveContext("org-a/shared")
	if got == nil {
		t.Fatal("ResolveContext(\"org-a/shared\") = nil, want proj-a")
	}
	if got.ProjectID != "proj-a" {
		t.Errorf("got %q, want proj-a (display-name resolution must be org-scoped)", got.ProjectID)
	}

	// And "org-b/shared" should resolve to proj-b.
	got = cfg.ResolveContext("org-b/shared")
	if got == nil {
		t.Fatal("ResolveContext(\"org-b/shared\") = nil, want proj-b")
	}
	if got.ProjectID != "proj-b" {
		t.Errorf("got %q, want proj-b", got.ProjectID)
	}
}

// TestFormatWithID verifies the FormatWithID helper.
func TestFormatWithID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		displayName string
		resourceID  string
		want        string
	}{
		{name: "display differs", displayName: "Acme Corp", resourceID: "org-acme", want: "Acme Corp (org-acme)"},
		{name: "display matches ID", displayName: "org-acme", resourceID: "org-acme", want: "org-acme"},
		{name: "empty display", displayName: "", resourceID: "org-acme", want: "org-acme"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FormatWithID(tt.displayName, tt.resourceID); got != tt.want {
				t.Errorf("FormatWithID(%q, %q) = %q, want %q", tt.displayName, tt.resourceID, got, tt.want)
			}
		})
	}
}

// TestDisplayRef verifies the DisplayRef helper.
func TestDisplayRef(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Cache.Organizations = []CachedOrg{
		{ID: "org-1", DisplayName: "Acme Corp"},
	}
	cfg.Cache.Projects = []CachedProject{
		{ID: "proj-1", DisplayName: "Infra", OrgID: "org-1"},
	}

	orgCtx := &DiscoveredContext{OrganizationID: "org-1"}
	if got := cfg.DisplayRef(orgCtx); got != "Acme Corp" {
		t.Errorf("org DisplayRef = %q, want %q", got, "Acme Corp")
	}

	projCtx := &DiscoveredContext{OrganizationID: "org-1", ProjectID: "proj-1"}
	if got := cfg.DisplayRef(projCtx); got != "Acme Corp/Infra" {
		t.Errorf("project DisplayRef = %q, want %q", got, "Acme Corp/Infra")
	}

	// Missing display names — fall back to IDs.
	orphan := &DiscoveredContext{OrganizationID: "unknown-org", ProjectID: "unknown-proj"}
	if got := cfg.DisplayRef(orphan); got != "unknown-org/unknown-proj" {
		t.Errorf("orphan DisplayRef = %q, want IDs", got)
	}
}
