package datumconfig

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	soProd    = "swells@datum.net@api.datum.net"
	soStaging = "swells@datum.net@api.staging.env.datum.net"
	soSolo    = "solo@example.com@api.datum.net"
)

var (
	soProdCtx     = QualifiedContextName(soProd, "org-prod")
	soStagingCtx  = QualifiedContextName(soStaging, "org-staging")
	soStagingProj = QualifiedContextName(soStaging, "org-staging/proj")
)

func overrideTestConfig() *ConfigV1Beta1 {
	cfg := NewV1Beta1()
	cfg.Sessions = []Session{
		{Name: soProd, UserEmail: "swells@datum.net", LastContext: soProdCtx},
		{Name: soStaging, UserEmail: "swells@datum.net", LastContext: soStagingCtx},
		{Name: soSolo, UserEmail: "solo@example.com"},
	}
	cfg.Contexts = []DiscoveredContext{
		{Name: soProdCtx, Session: soProd, OrganizationID: "org-prod"},
		{Name: soStagingCtx, Session: soStaging, OrganizationID: "org-staging"},
		{Name: soStagingProj, Session: soStaging, OrganizationID: "org-staging", ProjectID: "proj"},
	}
	cfg.ActiveSession = soProd
	cfg.CurrentContext = soProdCtx
	return cfg
}

func TestResolveSessionSelector(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		source  SessionOverrideSource
		want    string
		wantErr []string
	}{
		{name: "exact session name", value: soStaging, want: soStaging},
		{name: "surrounding space is ignored", value: "  " + soStaging + " ", want: soStaging},
		{name: "unique email", value: "solo@example.com", want: soSolo},
		{name: "shared email lists each --session", value: "swells@datum.net", source: SessionOverrideFromFlag,
			wantErr: []string{"--session swells@datum.net is ambiguous", "--session " + soProd, "--session " + soStaging}},
		{name: "unknown value lists every session", value: "nobody@example.com", source: SessionOverrideFromEnv,
			wantErr: []string{"No session matches DATUM_SESSION nobody@example.com.", soProd, soStaging, soSolo}},
		{name: "email suffix is not a match", value: "datum.net", source: SessionOverrideFromFlag,
			wantErr: []string{"No session matches --session datum.net."}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := overrideTestConfig().ResolveSessionSelector(tt.value, tt.source)
			if len(tt.wantErr) > 0 {
				if err == nil {
					t.Fatalf("expected an error, got session %q", s.Name)
				}
				for _, w := range tt.wantErr {
					if !strings.Contains(err.Error(), w) {
						t.Errorf("error missing %q:\n%s", w, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if s.Name != tt.want {
				t.Errorf("session = %q, want %q", s.Name, tt.want)
			}
		})
	}

	_, err := NewV1Beta1().ResolveSessionSelector("x", SessionOverrideFromFlag)
	if err == nil || !strings.Contains(err.Error(), "datumctl login") {
		t.Errorf("no sessions: err = %v, want a hint to log in", err)
	}
}

func TestSessionOverrideLookups(t *testing.T) {
	tests := []struct {
		name        string
		override    string
		current     string // stored current-context, when not the default
		pick        string // SetOverrideContext
		wantSession string
		wantContext string
	}{
		{name: "no override", wantSession: soProd, wantContext: soProdCtx},
		{name: "override uses its last context", override: soStaging, wantSession: soStaging, wantContext: soStagingCtx},
		{name: "override without a context has none", override: soSolo, wantSession: soSolo, wantContext: ""},
		{name: "override matching the active session keeps the current context",
			override: soStaging, current: soStagingProj, wantSession: soStaging, wantContext: soStagingProj},
		{name: "context picked in this process wins",
			override: soStaging, pick: soStagingProj, wantSession: soStaging, wantContext: soStagingProj},
		{name: "context picked from another session is ignored",
			override: soStaging, pick: soProdCtx, wantSession: soStaging, wantContext: soStagingCtx},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(ClearSessionOverride)
			cfg := overrideTestConfig()
			if tt.current != "" {
				cfg.CurrentContext = tt.current
			}
			if tt.override != "" {
				SetSessionOverride(tt.override, SessionOverrideFromFlag)
			}
			if tt.pick != "" {
				SetOverrideContext(tt.pick)
			}
			if got := cfg.ActiveSessionEntry(); got == nil || got.Name != tt.wantSession {
				t.Errorf("ActiveSessionEntry = %v, want %q", got, tt.wantSession)
			}
			if got := cfg.ActiveSessionName(); got != tt.wantSession {
				t.Errorf("ActiveSessionName = %q, want %q", got, tt.wantSession)
			}
			if got := cfg.CurrentContextName(); got != tt.wantContext {
				t.Errorf("CurrentContextName = %q, want %q", got, tt.wantContext)
			}
		})
	}
}

// Saving a config while an override is active writes the stored active
// session and current context, never the override.
func TestSessionOverrideIsNeverSaved(t *testing.T) {
	t.Cleanup(ClearSessionOverride)
	path := filepath.Join(t.TempDir(), "config")
	cfg := overrideTestConfig()
	if err := SaveV1Beta1ToPath(cfg, path); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	SetSessionOverride(soStaging, SessionOverrideFromEnv)
	SetOverrideContext(soStagingProj)
	loaded, err := LoadV1Beta1FromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentContextName() != soStagingProj {
		t.Fatalf("override not in effect")
	}
	if err := SaveV1Beta1ToPath(loaded, path); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Errorf("config changed by saving under an override:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
