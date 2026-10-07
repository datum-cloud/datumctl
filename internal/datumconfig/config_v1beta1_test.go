package datumconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigV1Beta1RoundTrip verifies marshal/unmarshal round-trip preserves all fields.
func TestConfigV1Beta1RoundTrip(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	const sess = "jane@acme.com@api.datum.net"
	const currentCtx = sess + "/org-acme/proj-infra"
	original := NewV1Beta1()
	original.CurrentContext = currentCtx
	original.ActiveSession = sess
	original.Sessions = []Session{
		{
			Name:      sess,
			UserKey:   "key-abc123",
			UserEmail: "jane@acme.com",
			UserName:  "Jane Doe",
			Endpoint: Endpoint{
				Server:       "https://api.datum.net",
				AuthHostname: "auth.datum.net",
			},
			LastContext: currentCtx,
		},
	}
	original.Contexts = []DiscoveredContext{
		{
			Name:           sess + "/org-acme",
			Session:        sess,
			OrganizationID: "org-acme",
		},
		{
			Name:           currentCtx,
			Session:        sess,
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
	if loaded.CurrentContext != currentCtx {
		t.Errorf("CurrentContext=%q, want %q", loaded.CurrentContext, currentCtx)
	}
	if loaded.ActiveSession != sess {
		t.Errorf("ActiveSession=%q, want %q", loaded.ActiveSession, sess)
	}
	if len(loaded.Sessions) != 1 {
		t.Fatalf("Sessions len=%d, want 1", len(loaded.Sessions))
	}
	s := loaded.Sessions[0]
	if s.Name != sess {
		t.Errorf("Session.Name=%q, want %q", s.Name, sess)
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
	if s.LastContext != currentCtx {
		t.Errorf("Session.LastContext=%q, want %q", s.LastContext, currentCtx)
	}
	// Only the selected context survives a save; the unselected org context
	// is dropped.
	if len(loaded.Contexts) != 1 || loaded.Contexts[0].Name != currentCtx {
		t.Fatalf("Contexts=%+v, want only %q", loaded.Contexts, currentCtx)
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

// TestActiveSessionEntry verifies the single-source-of-truth precedence: the
// current context's session is authoritative, and the stored ActiveSession is
// only a fallback for when no current context resolves.
func TestActiveSessionEntry(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Sessions = []Session{
		{Name: "sess-1", UserEmail: "alice@example.com"},
		{Name: "sess-2", UserEmail: "bob@example.com"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: "sess-1/acme-corp", Session: "sess-1", OrganizationID: "acme-corp"},
	}

	// No active session and no current context — should return nil.
	got := cfg.ActiveSessionEntry()
	if got != nil {
		t.Errorf("expected nil when no ActiveSession and no CurrentContext, got %+v", got)
	}

	// ActiveSession only (no current context) — falls back to ActiveSession.
	cfg.ActiveSession = "sess-2"
	got = cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected fallback to ActiveSession, got nil")
	}
	if got.UserEmail != "bob@example.com" {
		t.Errorf("fallback UserEmail=%q, want %q", got.UserEmail, "bob@example.com")
	}

	// Set a current context — its session wins over a divergent ActiveSession.
	// This is the #248 fix: whoami must not report a different environment than
	// the one the current context routes requests to.
	cfg.CurrentContext = "sess-1/acme-corp"
	got = cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected session from current context, got nil")
	}
	if got.UserEmail != "alice@example.com" {
		t.Errorf("current context's session must win: UserEmail=%q, want %q", got.UserEmail, "alice@example.com")
	}

	// Current context points at a missing session — fall back to ActiveSession.
	cfg.CurrentContext = "sess-1/gone"
	cfg.Contexts = append(cfg.Contexts, DiscoveredContext{Name: "sess-1/gone", Session: "removed-session"})
	got = cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected fallback to ActiveSession when context session missing, got nil")
	}
	if got.UserEmail != "bob@example.com" {
		t.Errorf("fallback UserEmail=%q, want %q", got.UserEmail, "bob@example.com")
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

// TestActiveSessionEntry_Issue248 reproduces the production incident: a user is
// logged into staging (ActiveSession = staging) but has selected a production
// context. whoami reads ActiveSessionEntry, and it must report production —
// the same environment requests route to — not the stale ActiveSession.
func TestActiveSessionEntry_Issue248(t *testing.T) {
	t.Parallel()

	const staging = "user@example.com@api.staging.datum.net"
	const prod = "user@example.com@api.datum.net"

	cfg := NewV1Beta1()
	cfg.Sessions = []Session{
		{Name: staging, UserEmail: "user@example.com", Endpoint: Endpoint{Server: "https://api.staging.datum.net"}},
		{Name: prod, UserEmail: "user@example.com", Endpoint: Endpoint{Server: "https://api.datum.net"}},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: QualifiedContextName(prod, "datum/datum-cloud"), Session: prod, OrganizationID: "datum", ProjectID: "datum-cloud"},
	}
	// The bug state: ActiveSession still names staging, but the current context
	// belongs to prod (as it would after `ctx use` a prod context).
	cfg.ActiveSession = staging
	cfg.CurrentContext = QualifiedContextName(prod, "datum/datum-cloud")

	got := cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("ActiveSessionEntry returned nil")
	}
	if got.Name != prod {
		t.Errorf("ActiveSessionEntry = %q, want %q (must follow the current context, not the stale ActiveSession)", got.Name, prod)
	}
}

// TestActiveSessionEntry_FallbackWhenNoContext verifies a freshly-logged-in
// session with no current context falls back to ActiveSession.
func TestActiveSessionEntry_FallbackWhenNoContext(t *testing.T) {
	t.Parallel()

	cfg := NewV1Beta1()
	cfg.Sessions = []Session{{Name: "fresh@api.datum.net", UserEmail: "fresh@example.com"}}
	cfg.ActiveSession = "fresh@api.datum.net"
	// No CurrentContext set (as right after login before picking one).

	got := cfg.ActiveSessionEntry()
	if got == nil {
		t.Fatal("expected fallback to ActiveSession, got nil")
	}
	if got.UserEmail != "fresh@example.com" {
		t.Errorf("UserEmail=%q, want %q", got.UserEmail, "fresh@example.com")
	}
}

// TestMigrateSessionScoping verifies that a config written before session
// qualification is upgraded on load: context names become session-qualified,
// the current-context and last-context pointers follow, and contexts stay
// intact.
func TestMigrateSessionScoping(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	path := filepath.Join(tmp, "config")

	// A pre-change config: bare-ref context names, un-sessioned cache.
	legacy := `apiVersion: datumctl.config.datum.net/v1beta1
kind: DatumctlConfig
current-context: datum/datum-cloud
active-session: user@example.com@api.datum.net
sessions:
- name: user@example.com@api.datum.net
  user-key: k1
  user-email: user@example.com
  endpoint:
    server: https://api.datum.net
    auth-hostname: auth.datum.net
  last-context: datum/datum-cloud
contexts:
- name: datum
  session: user@example.com@api.datum.net
  organization-id: datum
- name: datum/datum-cloud
  session: user@example.com@api.datum.net
  organization-id: datum
  project-id: datum-cloud
  namespace: default
cache:
  organizations:
  - id: datum
    display-name: Datum Technology
  projects:
  - id: datum-cloud
    display-name: Datum Cloud
    org-id: datum
`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadV1Beta1FromPath(path)
	if err != nil {
		t.Fatalf("LoadV1Beta1FromPath: %v", err)
	}

	const sess = "user@example.com@api.datum.net"
	wantCurrent := QualifiedContextName(sess, "datum/datum-cloud")

	// Names upgraded to the qualified form.
	if cfg.CurrentContext != wantCurrent {
		t.Errorf("CurrentContext=%q, want %q", cfg.CurrentContext, wantCurrent)
	}
	if got := cfg.SessionByName(sess); got == nil || got.LastContext != wantCurrent {
		t.Errorf("LastContext not migrated: %+v", got)
	}
	// Contexts intact and addressable by their new qualified names.
	if len(cfg.Contexts) != 2 {
		t.Fatalf("Contexts len=%d, want 2", len(cfg.Contexts))
	}
	if cfg.ContextByName(wantCurrent) == nil {
		t.Error("qualified project context not found after migration")
	}
	if cfg.CurrentContextEntry() == nil {
		t.Error("current context does not resolve after migration")
	}

	// Migration is idempotent: running it again changes nothing.
	before := cfg.CurrentContext
	cfg.migrateSessionScoping()
	if cfg.CurrentContext != before {
		t.Errorf("migration not idempotent: CurrentContext changed to %q", cfg.CurrentContext)
	}
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name != cfg.Contexts[i].QualifiedName() {
			t.Errorf("context %d not stable under re-migration: %q", i, cfg.Contexts[i].Name)
		}
	}
}

// TestLoadLegacyCacheConfig verifies that a config written by a release that
// stored every discovered context and a display-name cache still loads, keeps
// the selection, and drops the cache and unselected contexts on save.
func TestLoadLegacyCacheConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config")
	const sess = "user@example.com@api.datum.net"
	const current = sess + "/datum/datum-cloud"
	legacy := `apiVersion: datumctl.config.datum.net/v1beta1
kind: DatumctlConfig
current-context: ` + current + `
active-session: ` + sess + `
sessions:
- name: ` + sess + `
  user-key: k1
  user-email: user@example.com
  endpoint:
    server: https://api.datum.net
    auth-hostname: auth.datum.net
  last-context: ` + current + `
contexts:
- name: ` + sess + `/datum
  session: ` + sess + `
  organization-id: datum
- name: ` + current + `
  session: ` + sess + `
  organization-id: datum
  project-id: datum-cloud
  namespace: custom
cache:
  organizations:
  - id: datum
    display-name: Datum Technology
    session: ` + sess + `
  projects:
  - id: datum-cloud
    display-name: Datum Cloud
    org-id: datum
    session: ` + sess + `
  last-refreshed: "2026-01-02T03:04:05Z"
`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	cfg, err := LoadV1Beta1FromPath(path)
	if err != nil {
		t.Fatalf("LoadV1Beta1FromPath: %v", err)
	}
	got := cfg.CurrentContextEntry()
	if got == nil || got.ProjectID != "datum-cloud" || got.Namespace != "custom" {
		t.Fatalf("CurrentContextEntry = %+v, want datum/datum-cloud in namespace custom", got)
	}

	if err := SaveV1Beta1ToPath(cfg, path); err != nil {
		t.Fatalf("SaveV1Beta1ToPath: %v", err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	for _, gone := range []string{"cache:", "last-refreshed", "display-name", "name: " + sess + "/datum\n"} {
		if strings.Contains(string(saved), gone) {
			t.Errorf("saved config still contains %q:\n%s", gone, saved)
		}
	}
	if !strings.Contains(string(saved), "current-context: "+current) {
		t.Errorf("saved config lost the current context:\n%s", saved)
	}
}

// TestSelectContext verifies that selecting a context records it as current,
// as its session's last context, and makes its session active.
func TestSelectContext(t *testing.T) {
	t.Parallel()

	const sess = "user@example.com@api.datum.net"
	cfg := NewV1Beta1()
	cfg.Sessions = []Session{{Name: sess}}
	ctx := DiscoveredContext{Session: sess, OrganizationID: "datum", ProjectID: "new-proj"}
	ctx.Name = ctx.QualifiedName()

	cfg.SelectContext(ctx)

	if cfg.CurrentContext != ctx.Name || cfg.ActiveSession != sess {
		t.Errorf("CurrentContext=%q ActiveSession=%q", cfg.CurrentContext, cfg.ActiveSession)
	}
	if cfg.SessionByName(sess).LastContext != ctx.Name {
		t.Errorf("LastContext=%q, want %q", cfg.SessionByName(sess).LastContext, ctx.Name)
	}
	if cfg.CurrentContextEntry() == nil {
		t.Error("current context entry not stored")
	}
}
