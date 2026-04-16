package discovery

import (
	"testing"
	"time"

	"go.datum.net/datumctl/internal/datumconfig"
)

// TestUpdateConfigCache_GeneratesOrgAndProjectContexts verifies that
// UpdateConfigCache creates org-level and project-level contexts using resource
// names (orgID, orgID/projectID).
func TestUpdateConfigCache_GeneratesOrgAndProjectContexts(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	sessionName := "alice@example.com@api.datum.net"

	orgs := []DiscoveredOrg{
		{Name: "org-acme", DisplayName: "acme-corp"},
		{Name: "org-personal", DisplayName: "personal"},
	}
	projects := []DiscoveredProject{
		{Name: "proj-infra", DisplayName: "infra", OrgName: "org-acme"},
		{Name: "proj-web", DisplayName: "web-app", OrgName: "org-acme"},
		{Name: "proj-sandbox", DisplayName: "sandbox", OrgName: "org-personal"},
	}

	UpdateConfigCache(cfg, sessionName, orgs, projects)

	// Expect 2 org contexts + 3 project contexts = 5 total.
	if len(cfg.Contexts) != 5 {
		t.Fatalf("Contexts len=%d, want 5", len(cfg.Contexts))
	}

	// Verify org context uses resource name.
	ctx := cfg.ContextByName("org-acme")
	if ctx == nil {
		t.Fatal("expected org context 'org-acme', not found")
	}
	if ctx.OrganizationID != "org-acme" {
		t.Errorf("org-acme OrganizationID=%q, want %q", ctx.OrganizationID, "org-acme")
	}
	if ctx.ProjectID != "" {
		t.Errorf("org-acme ProjectID=%q, want empty (org context)", ctx.ProjectID)
	}
	if ctx.Session != sessionName {
		t.Errorf("org-acme Session=%q, want %q", ctx.Session, sessionName)
	}

	// Verify project context uses "orgID/projectID" format.
	projCtx := cfg.ContextByName("org-acme/proj-infra")
	if projCtx == nil {
		t.Fatal("expected project context 'org-acme/proj-infra', not found")
	}
	if projCtx.OrganizationID != "org-acme" {
		t.Errorf("org-acme/proj-infra OrganizationID=%q, want %q", projCtx.OrganizationID, "org-acme")
	}
	if projCtx.ProjectID != "proj-infra" {
		t.Errorf("org-acme/proj-infra ProjectID=%q, want %q", projCtx.ProjectID, "proj-infra")
	}
	if projCtx.Namespace != datumconfig.DefaultNamespace {
		t.Errorf("org-acme/proj-infra Namespace=%q, want %q", projCtx.Namespace, datumconfig.DefaultNamespace)
	}

	// Check cross-org project.
	sandboxCtx := cfg.ContextByName("org-personal/proj-sandbox")
	if sandboxCtx == nil {
		t.Error("expected project context 'org-personal/proj-sandbox', not found")
	}
}

// TestUpdateConfigCache_FallsBackToIDWhenNoDisplayName verifies that resource
// names are used regardless of whether display names are present.
func TestUpdateConfigCache_FallsBackToIDWhenNoDisplayName(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	sessionName := "user@api.datum.net"

	orgs := []DiscoveredOrg{
		{Name: "org-123", DisplayName: ""},
	}
	projects := []DiscoveredProject{
		{Name: "proj-456", DisplayName: "", OrgName: "org-123"},
	}

	UpdateConfigCache(cfg, sessionName, orgs, projects)

	orgCtx := cfg.ContextByName("org-123")
	if orgCtx == nil {
		t.Error("expected org context 'org-123', not found")
	}

	projCtx := cfg.ContextByName("org-123/proj-456")
	if projCtx == nil {
		t.Error("expected project context 'org-123/proj-456', not found")
	}
}

// TestUpdateConfigCache_ReplacesExistingSessionContexts verifies that old
// contexts for the session are removed and replaced by new ones.
func TestUpdateConfigCache_ReplacesExistingSessionContexts(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	sessionName := "user@api.datum.net"

	// Seed with a stale context for this session and one for a different session.
	cfg.UpsertContext(datumconfig.DiscoveredContext{
		Name:    "stale-ctx",
		Session: sessionName,
	})
	cfg.UpsertContext(datumconfig.DiscoveredContext{
		Name:    "other-session-ctx",
		Session: "other-session",
	})

	orgs := []DiscoveredOrg{
		{Name: "org-new", DisplayName: "new-org"},
	}
	UpdateConfigCache(cfg, sessionName, orgs, nil)

	// "stale-ctx" should be gone; "other-session-ctx" should remain.
	if cfg.ContextByName("stale-ctx") != nil {
		t.Error("stale-ctx should have been removed")
	}
	if cfg.ContextByName("other-session-ctx") == nil {
		t.Error("other-session-ctx should have been preserved")
	}

	// New context uses resource name, not display name.
	if cfg.ContextByName("org-new") == nil {
		t.Error("new org context 'org-new' should be present")
	}
}

// TestUpdateConfigCache_UpdatesCache verifies that the cache metadata
// (organizations, projects, LastRefreshed) is populated.
func TestUpdateConfigCache_UpdatesCache(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	sessionName := "user@api.datum.net"

	orgs := []DiscoveredOrg{
		{Name: "org-1", DisplayName: "Org One"},
	}
	projects := []DiscoveredProject{
		{Name: "proj-1", DisplayName: "Project One", OrgName: "org-1"},
	}

	UpdateConfigCache(cfg, sessionName, orgs, projects)

	if len(cfg.Cache.Organizations) != 1 {
		t.Fatalf("Cache.Organizations len=%d, want 1", len(cfg.Cache.Organizations))
	}
	if cfg.Cache.Organizations[0].ID != "org-1" {
		t.Errorf("CachedOrg.ID=%q, want %q", cfg.Cache.Organizations[0].ID, "org-1")
	}
	if cfg.Cache.Organizations[0].DisplayName != "Org One" {
		t.Errorf("CachedOrg.DisplayName=%q, want %q", cfg.Cache.Organizations[0].DisplayName, "Org One")
	}

	if len(cfg.Cache.Projects) != 1 {
		t.Fatalf("Cache.Projects len=%d, want 1", len(cfg.Cache.Projects))
	}
	if cfg.Cache.Projects[0].ID != "proj-1" {
		t.Errorf("CachedProject.ID=%q, want %q", cfg.Cache.Projects[0].ID, "proj-1")
	}
	if cfg.Cache.Projects[0].OrgID != "org-1" {
		t.Errorf("CachedProject.OrgID=%q, want %q", cfg.Cache.Projects[0].OrgID, "org-1")
	}

	if cfg.Cache.LastRefreshed == nil {
		t.Error("Cache.LastRefreshed should be set")
	}
}

// TestUpdateConfigCache_EmptyOrgsAndProjects verifies that calling with no
// orgs/projects clears all contexts for the session.
func TestUpdateConfigCache_EmptyOrgsAndProjects(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	sessionName := "user@api.datum.net"

	cfg.UpsertContext(datumconfig.DiscoveredContext{
		Name:    "old-ctx",
		Session: sessionName,
	})

	UpdateConfigCache(cfg, sessionName, nil, nil)

	if cfg.ContextByName("old-ctx") != nil {
		t.Error("old-ctx should have been removed when updated with empty orgs/projects")
	}
	if len(cfg.Contexts) != 0 {
		t.Errorf("Contexts len=%d, want 0", len(cfg.Contexts))
	}
}

// TestUpdateConfigCache_MultiSession_PreservesOtherSessionCache verifies the
// GC fix: refreshing session-1's discovery must NOT wipe cache entries for orgs
// and projects that only session-2 knows about.
func TestUpdateConfigCache_MultiSession_PreservesOtherSessionCache(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	session1 := "user1@api.datum.net"
	session2 := "user2@api.datum.net"

	// Pre-populate both sessions.
	UpdateConfigCache(cfg, session1, []DiscoveredOrg{
		{Name: "org-prod", DisplayName: "Production"},
	}, []DiscoveredProject{
		{Name: "proj-p1", DisplayName: "Production Project", OrgName: "org-prod"},
	})
	UpdateConfigCache(cfg, session2, []DiscoveredOrg{
		{Name: "org-stg", DisplayName: "Staging"},
	}, []DiscoveredProject{
		{Name: "proj-s1", DisplayName: "Staging Project", OrgName: "org-stg"},
	})

	// Sanity: both sessions' data is present.
	if cfg.OrgDisplayName("org-prod") != "Production" {
		t.Fatal("setup: Production cache missing")
	}
	if cfg.OrgDisplayName("org-stg") != "Staging" {
		t.Fatal("setup: Staging cache missing")
	}

	// Refresh session1 — should not affect session2's cache.
	UpdateConfigCache(cfg, session1, []DiscoveredOrg{
		{Name: "org-prod", DisplayName: "Production"},
	}, []DiscoveredProject{
		{Name: "proj-p1", DisplayName: "Production Project", OrgName: "org-prod"},
	})

	if cfg.OrgDisplayName("org-stg") != "Staging" {
		t.Error("session2's org cache was wiped by session1 refresh")
	}
	if cfg.ProjectDisplayName("proj-s1") != "Staging Project" {
		t.Error("session2's project cache was wiped by session1 refresh")
	}
}

// TestGCCache_RemovesUnreferencedEntries verifies that GCCache removes cache
// entries not referenced by any context.
func TestGCCache_RemovesUnreferencedEntries(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	cfg.Cache.Organizations = []datumconfig.CachedOrg{
		{ID: "org-kept", DisplayName: "Kept"},
		{ID: "org-orphan", DisplayName: "Orphan"},
	}
	cfg.Cache.Projects = []datumconfig.CachedProject{
		{ID: "proj-kept", DisplayName: "Kept Project", OrgID: "org-kept"},
		{ID: "proj-orphan", DisplayName: "Orphan Project", OrgID: "org-orphan"},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{
		{Name: "org-kept", OrganizationID: "org-kept"},
		{Name: "org-kept/proj-kept", OrganizationID: "org-kept", ProjectID: "proj-kept"},
	}

	GCCache(cfg)

	if cfg.OrgDisplayName("org-kept") != "Kept" {
		t.Error("referenced org was removed")
	}
	if len(cfg.Cache.Organizations) != 1 {
		t.Errorf("Cache.Organizations len=%d, want 1", len(cfg.Cache.Organizations))
	}
	if len(cfg.Cache.Projects) != 1 {
		t.Errorf("Cache.Projects len=%d, want 1", len(cfg.Cache.Projects))
	}
}

// TestIsCacheStale verifies cache staleness detection.
func TestIsCacheStale(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()

	// No refresh timestamp — always stale.
	if !IsCacheStale(cfg, time.Hour) {
		t.Error("empty cache should be stale")
	}

	// Recently refreshed — not stale.
	now := time.Now()
	cfg.Cache.LastRefreshed = &now
	if IsCacheStale(cfg, time.Hour) {
		t.Error("recently refreshed cache should not be stale")
	}

	// Old refresh — stale.
	old := now.Add(-2 * time.Hour)
	cfg.Cache.LastRefreshed = &old
	if !IsCacheStale(cfg, time.Hour) {
		t.Error("old cache should be stale")
	}
}

// TestMergeCacheFromDiscovery verifies the merge-only variant preserves
// existing entries and doesn't touch contexts.
func TestMergeCacheFromDiscovery(t *testing.T) {
	t.Parallel()

	cfg := datumconfig.NewV1Beta1()
	cfg.Cache.Organizations = []datumconfig.CachedOrg{
		{ID: "org-existing", DisplayName: "Old Name"},
	}

	MergeCacheFromDiscovery(cfg, []DiscoveredOrg{
		{Name: "org-existing", DisplayName: "New Name"},
		{Name: "org-new", DisplayName: "New Org"},
	}, nil)

	// Existing was updated.
	if cfg.OrgDisplayName("org-existing") != "New Name" {
		t.Errorf("existing org not updated, got %q", cfg.OrgDisplayName("org-existing"))
	}
	// New was added.
	if cfg.OrgDisplayName("org-new") != "New Org" {
		t.Errorf("new org not added")
	}
	// No contexts should have been created.
	if len(cfg.Contexts) != 0 {
		t.Errorf("merge should not touch contexts, got %d", len(cfg.Contexts))
	}
}
