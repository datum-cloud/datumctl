package data

import (
	"context"
	"errors"
	"testing"
)

// mockEventsClient is a minimal ResourceClient stub that only implements ListEvents.
// All other methods panic if called, making unexpected calls visible immediately.
type mockEventsClient struct {
	wantKind      string
	wantName      string
	wantNamespace string

	returnRows []EventRow
	returnErr  error

	gotKind      string
	gotName      string
	gotNamespace string
	called       bool
}

func (m *mockEventsClient) ListResourceTypes(_ context.Context) ([]ResourceType, error) {
	panic("mockEventsClient: ListResourceTypes not implemented")
}
func (m *mockEventsClient) ListResources(_ context.Context, _ ResourceType, _ string) ([]ResourceRow, []string, error) {
	panic("mockEventsClient: ListResources not implemented")
}
func (m *mockEventsClient) DescribeResource(_ context.Context, _ ResourceType, _, _ string) (DescribeResult, error) {
	panic("mockEventsClient: DescribeResource not implemented")
}
func (m *mockEventsClient) DeleteResource(_ context.Context, _ ResourceType, _, _ string) error {
	panic("mockEventsClient: DeleteResource not implemented")
}
func (m *mockEventsClient) IsForbidden(_ error) bool     { return false }
func (m *mockEventsClient) IsNotFound(_ error) bool      { return false }
func (m *mockEventsClient) IsConflict(_ error) bool      { return false }
func (m *mockEventsClient) IsUnauthorized(_ error) bool  { return false }
func (m *mockEventsClient) InvalidateResourceListCache(_ string) {}
func (m *mockEventsClient) ListEvents(_ context.Context, kind, name, ns string) ([]EventRow, error) {
	m.called = true
	m.gotKind = kind
	m.gotName = name
	m.gotNamespace = ns
	return m.returnRows, m.returnErr
}

// TestLoadEventsCmd_SuccessPath verifies that LoadEventsCmd invokes rc.ListEvents and
// returns an EventsLoadedMsg with the events and a nil error on success.
func TestLoadEventsCmd_SuccessPath(t *testing.T) {
	t.Parallel()

	rows := []EventRow{
		{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created resource", Count: 1},
	}
	rc := &mockEventsClient{returnRows: rows, returnErr: nil}

	cmd := LoadEventsCmd(context.Background(), rc, "Pod", "my-pod", "default")
	if cmd == nil {
		t.Fatal("LoadEventsCmd returned nil cmd, want non-nil")
	}

	msg := cmd()
	loaded, ok := msg.(EventsLoadedMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want EventsLoadedMsg", msg)
	}
	if loaded.Err != nil {
		t.Errorf("EventsLoadedMsg.Err = %v, want nil", loaded.Err)
	}
	if len(loaded.Events) != len(rows) {
		t.Errorf("EventsLoadedMsg.Events length = %d, want %d", len(loaded.Events), len(rows))
	}
	if !rc.called {
		t.Error("ListEvents was not called")
	}
}

// TestLoadEventsCmd_ErrorPath verifies that when ListEvents returns an error,
// EventsLoadedMsg.Err is set and Events is nil.
func TestLoadEventsCmd_ErrorPath(t *testing.T) {
	t.Parallel()

	someErr := errors.New("connection refused")
	rc := &mockEventsClient{returnRows: nil, returnErr: someErr}

	cmd := LoadEventsCmd(context.Background(), rc, "Pod", "my-pod", "default")
	msg := cmd()

	loaded, ok := msg.(EventsLoadedMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want EventsLoadedMsg", msg)
	}
	if loaded.Err == nil {
		t.Fatal("EventsLoadedMsg.Err = nil, want error")
	}
	if !errors.Is(loaded.Err, someErr) {
		t.Errorf("EventsLoadedMsg.Err = %v, want %v", loaded.Err, someErr)
	}
	if loaded.Events != nil {
		t.Errorf("EventsLoadedMsg.Events = %v, want nil on error", loaded.Events)
	}
}

// TestLoadEventsCmd_ForwardsArgs verifies that kind, name, and namespace are
// forwarded to ListEvents without mutation.
func TestLoadEventsCmd_ForwardsArgs(t *testing.T) {
	t.Parallel()

	rc := &mockEventsClient{}
	cmd := LoadEventsCmd(context.Background(), rc, "Gateway", "prod-gw", "production")
	cmd()

	if !rc.called {
		t.Fatal("ListEvents was not called")
	}
	if rc.gotKind != "Gateway" {
		t.Errorf("ListEvents kind = %q, want %q", rc.gotKind, "Gateway")
	}
	if rc.gotName != "prod-gw" {
		t.Errorf("ListEvents name = %q, want %q", rc.gotName, "prod-gw")
	}
	if rc.gotNamespace != "production" {
		t.Errorf("ListEvents namespace = %q, want %q", rc.gotNamespace, "production")
	}
}
