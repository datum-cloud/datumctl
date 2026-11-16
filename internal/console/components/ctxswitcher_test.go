package components

import (
	"bytes"
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"

	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/discovery"
)

// Under --session the switcher keeps the stored current context, but requests
// must still follow the switch. The request factory reloads the config from
// disk for each request, so the picked context must resolve there even though
// it is not stored.
func TestCtxSwitcherUnderSessionOverrideReachesRequests(t *testing.T) {
	const (
		active   = "active@example.com@api.datum.net"
		override = "other@example.com@api.datum.net"
	)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Cleanup(datumconfig.ClearSessionOverride)

	activeCtx := datumconfig.QualifiedContextName(active, "org-a")
	cfg := datumconfig.NewV1Beta1()
	cfg.Sessions = []datumconfig.Session{
		{Name: active, UserEmail: "active@example.com"},
		{Name: override, UserEmail: "other@example.com"},
	}
	cfg.Contexts = []datumconfig.DiscoveredContext{{Name: activeCtx, Session: active, OrganizationID: "org-a"}}
	cfg.ActiveSession = active
	cfg.CurrentContext = activeCtx
	if err := datumconfig.SaveV1Beta1(cfg); err != nil {
		t.Fatal(err)
	}
	path, err := datumconfig.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	datumconfig.SetSessionOverride(override, datumconfig.SessionOverrideFromFlag)
	dir := &discovery.Directory{Session: override, Orgs: []discovery.DiscoveredOrg{{Name: "org-b"}}}
	sw := NewCtxSwitcherModel(cfg, dir, 80, 24)
	_, cmd := sw.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter produced no command")
	}
	msg, ok := cmd().(ContextSwitchedMsg)
	if !ok {
		t.Fatalf("enter produced %T, want ContextSwitchedMsg", cmd())
	}
	if msg.Ctx.ActiveCtx == nil || msg.Ctx.ActiveCtx.OrganizationID != "org-b" {
		t.Fatalf("header context = %+v, want org-b", msg.Ctx.ActiveCtx)
	}

	// What the request factory sees: a fresh load from disk.
	fresh, err := datumconfig.LoadAuto()
	if err != nil {
		t.Fatal(err)
	}
	got, err := fresh.CurrentContextEntryE()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.OrganizationID != "org-b" || got.ProjectID != "" {
		t.Errorf("requests use context %+v, want org-wide org-b", got)
	}

	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Errorf("switch under --session changed the stored config:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}
