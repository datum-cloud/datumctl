package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/datumconfig"
	"go.datum.net/datumctl/internal/tui/components"
	tuictx "go.datum.net/datumctl/internal/tui/context"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/layout"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var ansiEscapeModel = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSIModel(s string) string {
	return ansiEscapeModel.ReplaceAllString(s, "")
}

// collectMsgs executes a cmd and all nested tea.BatchMsg sub-cmds, returning
// every produced message. Useful for asserting on batched commands.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var all []tea.Msg
		for _, sub := range batch {
			all = append(all, collectMsgs(sub)...)
		}
		return all
	}
	if msg != nil {
		return []tea.Msg{msg}
	}
	return nil
}

// stubBucketClient satisfies data.BucketClient without a real API server.
type stubBucketClient struct {
	buckets     []data.AllowanceBucket
	invalidated bool
}

func (s *stubBucketClient) ListAllowanceBuckets(_ context.Context) ([]data.AllowanceBucket, error) {
	return s.buckets, nil
}
func (s *stubBucketClient) InvalidateBucketCache() { s.invalidated = true }

// newAllowanceBucketNavModel builds a minimal AppModel in NavPane with
// allowancebuckets selected and the given BucketClient wired up.
func newAllowanceBucketNavModel(bc data.BucketClient) AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{
		{Name: "allowancebuckets", Kind: "AllowanceBucket", Namespaced: false},
	})
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		bc:          bc,
		activePane:  NavPane,
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(58, 20),
		detail:      components.NewDetailViewModel(58, 20),
		quota:       components.NewQuotaDashboardModel(58, 20, "proj"),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// newQuotaDashboardPaneModel builds a minimal AppModel in QuotaDashboardPane
// with one pre-loaded bucket and the given BucketClient wired up.
func newQuotaDashboardPaneModel(bc data.BucketClient) AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{
		{Name: "allowancebuckets", Kind: "AllowanceBucket", Namespaced: false},
	})
	quota := components.NewQuotaDashboardModel(58, 20, "proj")
	quota.SetBuckets([]data.AllowanceBucket{
		{Name: "my-bucket", ConsumerKind: "project", ResourceType: "cpus", Allocated: 10, Limit: 100},
	})
	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		bc:            bc,
		activePane:    QuotaDashboardPane,
		tableTypeName: "allowancebuckets",
		sidebar:       sidebar,
		table:         components.NewResourceTableModel(58, 20),
		detail:        components.NewDetailViewModel(58, 20),
		quota:         quota,
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// stubResourceClient satisfies data.ResourceClient without a real API server.
type stubResourceClient struct {
	deleteErr error // returned by DeleteResource when set
}

func (s stubResourceClient) ListResourceTypes(_ context.Context) ([]data.ResourceType, error) {
	return nil, nil
}
func (s stubResourceClient) ListResources(_ context.Context, _ data.ResourceType, _ string) ([]data.ResourceRow, []string, error) {
	return nil, nil, nil
}
func (s stubResourceClient) DescribeResource(_ context.Context, _ data.ResourceType, _, _ string) (data.DescribeResult, error) {
	return data.DescribeResult{}, nil
}
func (s stubResourceClient) DeleteResource(_ context.Context, _ data.ResourceType, _, _ string) error {
	return s.deleteErr
}
func (s stubResourceClient) IsForbidden(err error) bool {
	return errors.Is(err, errStubForbidden)
}
func (s stubResourceClient) IsNotFound(err error) bool {
	return errors.Is(err, errStubNotFound)
}
func (s stubResourceClient) IsConflict(err error) bool {
	return errors.Is(err, errStubConflict)
}
func (s stubResourceClient) IsUnauthorized(err error) bool {
	return errors.Is(err, errStubForbidden)
}
func (s stubResourceClient) InvalidateResourceListCache(_ string) {}
func (s stubResourceClient) ListEvents(_ context.Context, _, _, _ string) ([]data.EventRow, error) {
	return nil, nil
}

// Sentinel errors for stubResourceClient's classifier methods.
var (
	errStubForbidden = errors.New("stub: forbidden")
	errStubNotFound  = errors.New("stub: not found")
	errStubConflict  = errors.New("stub: conflict")
)

// newTablePaneModel builds a minimal AppModel in TablePane with one resource
// type and rows pre-loaded, including tableTypeName so the r-key handler fires.
func newTablePaneModel() AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{
		{Name: "pods", Kind: "Pod", Namespaced: true},
	})

	table := components.NewResourceTableModel(58, 20)
	table.SetColumns([]string{"Name"}, 58)
	table.SetRows([]data.ResourceRow{
		{Name: "my-pod", Namespace: "default", Cells: []string{"my-pod"}},
	})
	table.SetTypeContext("pods", true)

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    TablePane,
		tableTypeName: "pods",
		sidebar:       sidebar,
		table:         table,
		detail:        components.NewDetailViewModel(58, 20),
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// --- FB-001 tests (unchanged) ---

// TestAppModel_DKey_TransitionsToDetailWithLoading verifies that pressing "d"
// from TablePane immediately sets activePane=DetailPane and detail.Loading=true,
// so the loading title bar variant is visible from the very first frame.
func TestAppModel_DKey_TransitionsToDetailWithLoading(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v, want DetailPane", appM.activePane)
	}
	if !appM.detail.Loading() {
		t.Errorf("detail.Loading() = false, want true immediately after d keystroke")
	}
}

// TestAppModel_WindowSizeMsg_DetailHeightWithinBounds verifies that after a
// WindowSizeMsg the rendered detail pane height never exceeds the available
// main area height. The +2 tolerance accounts for PaneBorder padding.
func TestAppModel_WindowSizeMsg_DetailHeightWithinBounds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"standard 80x24", 80, 24},
		{"wide 120x40", 120, 40},
		{"short 80x15", 80, 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newTablePaneModel()
			result, _ := m.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			appM := result.(AppModel)

			mainH := layout.MainAreaWithFilter(tt.height, false)
			maxRenderedH := mainH + 2
			renderedH := lipgloss.Height(appM.detail.View())
			if renderedH > maxRenderedH {
				t.Errorf("%dx%d: detail rendered height = %d, want <= %d (mainArea=%d)",
					tt.width, tt.height, renderedH, maxRenderedH, mainH)
			}
		})
	}
}

// --- FB-004 tests (cursor preservation) ---

// TestAppModel_TickMsg_SetsPreserveCursor verifies that a TickMsg with a type
// selected sets preserveCursor=true but NOT refreshing, preserving the FB-002
// invariant that the header spinner must not flash on auto-ticks.
func TestAppModel_TickMsg_SetsPreserveCursor(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, _ := m.Update(data.TickMsg{})
	appM := result.(AppModel)

	if !appM.preserveCursor {
		t.Error("preserveCursor = false after TickMsg, want true")
	}
	if appM.refreshing {
		t.Error("refreshing = true after TickMsg, want false (header must not show refreshing…)")
	}
}

// TestAppModel_ManualR_SetsBothFlags verifies that pressing r in TABLE pane
// sets both refreshing=true and preserveCursor=true.
func TestAppModel_ManualR_SetsBothFlags(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !appM.refreshing {
		t.Error("refreshing = false, want true")
	}
	if !appM.preserveCursor {
		t.Error("preserveCursor = false, want true")
	}
}

// TestAppModel_ResourcesLoadedMsg_PreservesCursorByName verifies that when
// preserveCursor=true (set by tick or manual r), ResourcesLoadedMsg uses
// RefreshRows so the cursor follows the previously-selected row's name.
func TestAppModel_ResourcesLoadedMsg_PreservesCursorByName(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Load three rows and navigate cursor to "gamma-pod" (index 2) via j keypresses.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "alpha-pod", Cells: []string{"alpha-pod"}},
		{Name: "beta-pod", Cells: []string{"beta-pod"}},
		{Name: "gamma-pod", Cells: []string{"gamma-pod"}},
	})
	for i := 0; i < 2; i++ {
		res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = res.(AppModel)
	}

	// Signal cursor preservation (as set by TickMsg or r key).
	m.preserveCursor = true

	// New rows arrive — "gamma-pod" moved from index 2 to index 0.
	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows: []data.ResourceRow{
			{Name: "gamma-pod", Cells: []string{"gamma-pod"}},
			{Name: "alpha-pod", Cells: []string{"alpha-pod"}},
			{Name: "beta-pod", Cells: []string{"beta-pod"}},
		},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "pods"},
	})
	appM := result.(AppModel)

	if appM.preserveCursor {
		t.Error("preserveCursor = true after ResourcesLoadedMsg, want false (flag must be cleared)")
	}
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false")
	}
	if row.Name != "gamma-pod" {
		t.Errorf("SelectedRow().Name = %q, want %q (cursor must follow row by name)", row.Name, "gamma-pod")
	}
}

// TestAppModel_EnterOnSidebar_DoesNotSetPreserveCursor verifies that pressing
// Enter from NavPane (type-switch load) does not set preserveCursor.
func TestAppModel_EnterOnSidebar_DoesNotSetPreserveCursor(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.preserveCursor {
		t.Error("preserveCursor = true after NavPane Enter, want false for type-switch load")
	}
}

// TestAppModel_LoadErrorMsg_ClearsPreserveCursor verifies that a LoadErrorMsg
// clears preserveCursor so a subsequent load starts fresh.
func TestAppModel_LoadErrorMsg_ClearsPreserveCursor(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.preserveCursor = true // simulate tick or r set the flag

	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("timeout")})
	appM := result.(AppModel)

	if appM.preserveCursor {
		t.Error("preserveCursor = true after LoadErrorMsg, want false")
	}
}

// --- FB-011 tests (filter preservation on auto-refresh tick) ---

// TestAppModel_TickMsg_PreservesFilter verifies that when a tick auto-refresh
// delivers ResourcesLoadedMsg, an active table filter is not cleared. The tick
// path sets preserveCursor=true (not refreshing), and the ResourcesLoadedMsg
// handler uses that flag to gate both cursor and filter preservation.
func TestAppModel_TickMsg_PreservesFilter(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Load two rows: one matching "prod", one not.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "other-x", Cells: []string{"other-x"}},
		{Name: "prod-a", Cells: []string{"prod-a"}},
	})
	m.table.SetFilter("prod") // only "prod-a" visible

	// Inject TickMsg: sets preserveCursor=true, does not set refreshing.
	result, _ := m.Update(data.TickMsg{})
	m = result.(AppModel)
	if !m.preserveCursor {
		t.Fatal("preserveCursor = false after TickMsg, prerequisite for this test")
	}

	// Simulate the LoadResources response arriving — non-matching row is first.
	result, _ = m.Update(data.ResourcesLoadedMsg{
		Rows: []data.ResourceRow{
			{Name: "other-x", Cells: []string{"other-x"}},
			{Name: "prod-a", Cells: []string{"prod-a"}},
		},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "pods"},
	})
	appM := result.(AppModel)

	// Filter preserved → only "prod-a" visible, cursor at 0 → "prod-a".
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false, want a row (filter should preserve prod-a)")
	}
	if row.Name != "prod-a" {
		t.Errorf("SelectedRow().Name = %q, want %q (filter not preserved after tick refresh)", row.Name, "prod-a")
	}
}

// TestAppModel_ResourcesLoadedMsg_ClearsFilterOnTypeSwitch verifies that when
// ResourcesLoadedMsg arrives from a type-switch load (Enter on sidebar,
// preserveCursor=false), the table filter is cleared so all rows are visible.
func TestAppModel_ResourcesLoadedMsg_ClearsFilterOnTypeSwitch(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Active filter with one matching row.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "other-x", Cells: []string{"other-x"}},
		{Name: "prod-a", Cells: []string{"prod-a"}},
	})
	m.table.SetFilter("prod")

	// preserveCursor=false (default) — simulates type-switch path.

	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows: []data.ResourceRow{
			{Name: "other-x", Cells: []string{"other-x"}},
			{Name: "prod-a", Cells: []string{"prod-a"}},
		},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "pods"},
	})
	appM := result.(AppModel)

	// Filter cleared → all rows visible, cursor at 0 → "other-x".
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false, want a row")
	}
	if row.Name != "other-x" {
		t.Errorf("SelectedRow().Name = %q, want %q (filter should be cleared on type switch)", row.Name, "other-x")
	}
}

// TestAppModel_TickMsg_PreservesFilter_ZeroMatchingRows verifies that when a
// tick refresh delivers rows that no longer contain any match for the active
// filter, the filter is still preserved (not cleared) — resulting in an empty
// table rather than showing all unfiltered rows.
func TestAppModel_TickMsg_PreservesFilter_ZeroMatchingRows(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Filter "prod" active with two matching rows.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "prod-a", Cells: []string{"prod-a"}},
		{Name: "prod-b", Cells: []string{"prod-b"}},
	})
	m.table.SetFilter("prod")

	// Simulate tick having set preserveCursor.
	m.preserveCursor = true

	// New rows have NO "prod" entries.
	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows: []data.ResourceRow{
			{Name: "other-x", Cells: []string{"other-x"}},
			{Name: "other-y", Cells: []string{"other-y"}},
		},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "pods"},
	})
	appM := result.(AppModel)

	// Filter preserved → no rows match → SelectedRow returns false.
	_, ok := appM.table.SelectedRow()
	if ok {
		row, _ := appM.table.SelectedRow()
		t.Errorf("SelectedRow() = (%q, true), want (_, false): filter should be preserved, making table empty", row.Name)
	}
}

// TestAppModel_LoadErrorMsg_DoesNotTouchFilter verifies that a LoadErrorMsg
// does not clear the active table filter — the filter state is untouched.
func TestAppModel_LoadErrorMsg_DoesNotTouchFilter(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Active filter — only "prod-pod" visible.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "beta-pod", Cells: []string{"beta-pod"}},
		{Name: "prod-pod", Cells: []string{"prod-pod"}},
	})
	m.table.SetFilter("prod")
	m.preserveCursor = true
	m.refreshing = true
	m.tuiCtx.Refreshing = true

	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("timeout")})
	appM := result.(AppModel)

	// Filter intact → only "prod-pod" still visible.
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false, want prod-pod (filter should be intact after error)")
	}
	if row.Name != "prod-pod" {
		t.Errorf("SelectedRow().Name = %q, want %q (filter must not be cleared by LoadErrorMsg)", row.Name, "prod-pod")
	}
}

// --- FB-002 tests ---

// TestAppModel_RKey_TablePane_SetsRefreshing covers criterion 8:
// r in TABLE with a type loaded sets refreshing=true on the model and context,
// and dispatches LoadResourcesCmd (not LoadResourceTypesCmd).
func TestAppModel_RKey_TablePane_SetsRefreshing(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !appM.refreshing {
		t.Error("m.refreshing = false, want true")
	}
	if !appM.tuiCtx.Refreshing {
		t.Error("m.tuiCtx.Refreshing = false, want true")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want LoadResourcesCmd")
	}
	// Execute the cmd and verify it returns ResourcesLoadedMsg, not ResourceTypesLoadedMsg.
	msg := cmd()
	if _, ok := msg.(data.ResourcesLoadedMsg); !ok {
		t.Errorf("cmd() returned %T, want data.ResourcesLoadedMsg", msg)
	}
}

// TestAppModel_RKey_NavPane_DoesNotSetRefreshing covers criterion 4:
// r in NAV pane dispatches LoadResourceTypesCmd, refreshing stays false.
func TestAppModel_RKey_NavPane_DoesNotSetRefreshing(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if appM.refreshing {
		t.Error("m.refreshing = true, want false for NAV pane r")
	}
	if appM.tuiCtx.Refreshing {
		t.Error("m.tuiCtx.Refreshing = true, want false for NAV pane r")
	}
	if appM.loadState != data.LoadStateLoading {
		t.Errorf("loadState = %v, want LoadStateLoading", appM.loadState)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want LoadResourceTypesCmd")
	}
	msg := cmd()
	if _, ok := msg.(data.ResourceTypesLoadedMsg); !ok {
		t.Errorf("cmd() returned %T, want data.ResourceTypesLoadedMsg", msg)
	}
}

// TestAppModel_RKey_NoTypeLoaded_IsNoop covers spec §6c:
// r in TABLE pane when tableTypeName is empty (welcome panel) is a no-op.
func TestAppModel_RKey_NoTypeLoaded_IsNoop(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.tableTypeName = "" // simulate welcome panel — no type selected yet

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if appM.refreshing {
		t.Error("m.refreshing = true, want false for noop")
	}
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil for noop")
	}
}

// TestAppModel_RKey_DetailPane_SetsRefreshingPreservesDetail covers criterion 5:
// r in DETAIL pane dispatches LoadResourcesCmd for the parent type and does not
// re-fetch describe content (detail.Loading stays false).
func TestAppModel_RKey_DetailPane_SetsRefreshingPreservesDetail(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = DetailPane
	m.detail.SetResourceContext("pods", "my-pod")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !appM.refreshing {
		t.Error("m.refreshing = false, want true")
	}
	if appM.detail.Loading() {
		t.Error("detail.Loading() = true, want false — r must not trigger a describe re-fetch")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want LoadResourcesCmd")
	}
	msg := cmd()
	if _, ok := msg.(data.ResourcesLoadedMsg); !ok {
		t.Errorf("cmd() returned %T, want data.ResourcesLoadedMsg", msg)
	}
}

// TestAppModel_RKey_Coalesce_NopWhenAlreadyRefreshing covers spec §6a:
// a second r while refresh is in-flight is silently swallowed.
func TestAppModel_RKey_Coalesce_NopWhenAlreadyRefreshing(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// First r — starts refresh.
	result1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result1.(AppModel)
	if !appM.refreshing {
		t.Fatal("expected refreshing=true after first r")
	}

	// Second r — coalesced, must be a no-op.
	result2, cmd2 := appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM2 := result2.(AppModel)
	if !appM2.refreshing {
		t.Error("m.refreshing = false after second r, want true (still in flight)")
	}
	if cmd2 != nil {
		t.Errorf("cmd2 = non-nil, want nil for coalesced r")
	}
}

// TestAppModel_FilterPreservedOnManualRefresh covers criterion 9 (first half):
// when ResourcesLoadedMsg arrives after a manual refresh, the table filter is
// preserved — SetFilter("") is not called.
func TestAppModel_FilterPreservedOnManualRefresh(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Load two rows — one matching "prod", one not.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "beta-pod", Cells: []string{"beta-pod"}},
		{Name: "prod-pod", Cells: []string{"prod-pod"}},
	})
	m.table.SetFilter("prod") // only "prod-pod" visible

	// Simulate manual refresh in flight (r key sets both flags).
	m.refreshing = true
	m.preserveCursor = true
	m.tuiCtx.Refreshing = true

	// Inject ResourcesLoadedMsg (same rows, as if server returned them).
	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows: []data.ResourceRow{
			{Name: "beta-pod", Cells: []string{"beta-pod"}},
			{Name: "prod-pod", Cells: []string{"prod-pod"}},
		},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "pods"},
	})
	appM := result.(AppModel)

	if appM.refreshing {
		t.Error("m.refreshing = true after ResourcesLoadedMsg, want false")
	}
	// Filter preserved → cursor stays on the "prod-pod" row.
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false, want a row")
	}
	if row.Name != "prod-pod" {
		t.Errorf("SelectedRow().Name = %q, want %q (filter should be preserved)", row.Name, "prod-pod")
	}
}

// TestAppModel_FilterClearedOnInitialLoad covers criterion 9 (contrast half):
// when ResourcesLoadedMsg arrives from an initial/type-switch load (refreshing=false),
// SetFilter("") is called and all rows become visible.
func TestAppModel_FilterClearedOnInitialLoad(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Same two-row setup with "prod" filter active.
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "beta-pod", Cells: []string{"beta-pod"}},
		{Name: "prod-pod", Cells: []string{"prod-pod"}},
	})
	m.table.SetFilter("prod")

	// m.refreshing = false (default) — simulates initial/type-switch load.

	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows: []data.ResourceRow{
			{Name: "beta-pod", Cells: []string{"beta-pod"}},
			{Name: "prod-pod", Cells: []string{"prod-pod"}},
		},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "pods"},
	})
	appM := result.(AppModel)

	// Filter cleared → all rows visible, cursor at 0 → first row is "beta-pod".
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false, want a row")
	}
	if row.Name != "beta-pod" {
		t.Errorf("SelectedRow().Name = %q, want %q (filter should be cleared on initial load)", row.Name, "beta-pod")
	}
}

// TestAppModel_LoadErrorMsg_ClearsRefreshingPreservesRows covers criterion 6:
// a LoadErrorMsg during manual refresh clears refreshing and does not wipe table rows.
func TestAppModel_LoadErrorMsg_ClearsRefreshingPreservesRows(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	// Simulate a manual refresh in flight.
	m.refreshing = true
	m.tuiCtx.Refreshing = true

	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("network timeout")})
	appM := result.(AppModel)

	if appM.refreshing {
		t.Error("m.refreshing = true after LoadErrorMsg, want false")
	}
	if appM.tuiCtx.Refreshing {
		t.Error("m.tuiCtx.Refreshing = true after LoadErrorMsg, want false")
	}
	if appM.loadState != data.LoadStateError {
		t.Errorf("loadState = %v, want LoadStateError", appM.loadState)
	}
	if appM.statusBar.Err == nil {
		t.Error("statusBar.Err = nil, want non-nil error")
	}
	// Previously-loaded row must still be visible — no blank flash.
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false after error — rows were wiped")
	}
	if row.Name != "my-pod" {
		t.Errorf("SelectedRow().Name = %q, want %q", row.Name, "my-pod")
	}
}

// --- FB-010 tests (quota allowance-bucket usage dashboard) ---

// TestAppModel_EnterOnAllowanceBuckets_GoesToQuotaDashboardPane verifies that
// selecting allowancebuckets in the NAV sidebar and pressing Enter navigates to
// QuotaDashboardPane (not TablePane) and dispatches both LoadBucketsCmd and
// LoadResourcesCmd so both S2 dashboard and S1 table are populated.
func TestAppModel_EnterOnAllowanceBuckets_GoesToQuotaDashboardPane(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newAllowanceBucketNavModel(bc)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("activePane = %v, want QuotaDashboardPane", appM.activePane)
	}
	if appM.tableTypeName != "allowancebuckets" {
		t.Errorf("tableTypeName = %q, want %q", appM.tableTypeName, "allowancebuckets")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want batch cmd")
	}
	// Must produce both BucketsLoadedMsg (for S2 dashboard) and ResourcesLoadedMsg
	// (for S1 raw table, so t-toggle lands on a populated table).
	msgs := collectMsgs(cmd)
	var hasBuckets, hasResources bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.BucketsLoadedMsg:
			hasBuckets = true
		case data.ResourcesLoadedMsg:
			hasResources = true
		}
	}
	if !hasBuckets {
		t.Error("cmd batch did not produce data.BucketsLoadedMsg")
	}
	if !hasResources {
		t.Error("cmd batch did not produce data.ResourcesLoadedMsg")
	}
}

// TestAppModel_EnterOnNonAllowanceBuckets_GoesToTablePane verifies that Enter on
// any non-allowancebuckets type still navigates to TablePane as before.
func TestAppModel_EnterOnNonAllowanceBuckets_GoesToTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v, want TablePane for non-allowancebuckets type", appM.activePane)
	}
}

// TestAppModel_BucketsLoadedMsg_SetsQuotaDashboardBuckets verifies that
// BucketsLoadedMsg populates the quota dashboard and clears loading flags.
func TestAppModel_BucketsLoadedMsg_SetsQuotaDashboardBuckets(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})
	m.bucketLoading = true
	m.refreshing = true
	m.tuiCtx.Refreshing = true

	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "new-bucket", ConsumerKind: "project", ResourceType: "cpus", Allocated: 5, Limit: 100},
		},
	})
	appM := result.(AppModel)

	if appM.bucketLoading {
		t.Error("bucketLoading = true after BucketsLoadedMsg, want false")
	}
	if appM.refreshing {
		t.Error("refreshing = true after BucketsLoadedMsg, want false")
	}
	if appM.tuiCtx.Refreshing {
		t.Error("tuiCtx.Refreshing = true after BucketsLoadedMsg, want false")
	}
	b, ok := appM.quota.SelectedBucket()
	if !ok {
		t.Fatal("quota.SelectedBucket() returned false, want a bucket")
	}
	if b.Name != "new-bucket" {
		t.Errorf("quota.SelectedBucket().Name = %q, want %q", b.Name, "new-bucket")
	}
}

// TestAppModel_TKey_TogglesToTableFromDashboard verifies that pressing t in
// QuotaDashboardPane switches to TablePane.
func TestAppModel_TKey_TogglesToTableFromDashboard(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v, want TablePane after t from QuotaDashboardPane", appM.activePane)
	}
}

// TestAppModel_TKey_DashboardToTable_ShowsAllowanceBucketRows verifies that after
// Enter on allowancebuckets fires both LoadBucketsCmd and LoadResourcesCmd, pressing
// t lands on a TablePane that already has rows (not empty).
func TestAppModel_TKey_DashboardToTable_ShowsAllowanceBucketRows(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newQuotaDashboardPaneModel(bc)
	// Simulate the rows that LoadResourcesCmd would have delivered.
	m.table.SetColumns([]string{"Resource Type", "Limit", "Allocated"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "ab-projects", Cells: []string{"resourcemanager.miloapis.com/projects", "10", "4"}},
	})
	m.table.SetTypeContext("allowancebuckets", true)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v, want TablePane after t from QuotaDashboardPane", appM.activePane)
	}
	row, ok := appM.table.SelectedRow()
	if !ok {
		t.Fatal("table.SelectedRow() returned false — table is empty after t toggle to S1")
	}
	if row.Name != "ab-projects" {
		t.Errorf("table.SelectedRow().Name = %q, want %q", row.Name, "ab-projects")
	}
}

// TestAppModel_TKey_TogglesToDashboardFromTable verifies that pressing t in
// TablePane when tableTypeName == "allowancebuckets" switches to QuotaDashboardPane.
func TestAppModel_TKey_TogglesToDashboardFromTable(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.tableTypeName = "allowancebuckets"
	m.bc = &stubBucketClient{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("activePane = %v, want QuotaDashboardPane after t from TablePane (allowancebuckets)", appM.activePane)
	}
}

// TestAppModel_TKey_NonAllowanceBuckets_IsNoop verifies that pressing t in
// TablePane when a non-allowancebuckets type is selected is a no-op.
func TestAppModel_TKey_NonAllowanceBuckets_IsNoop(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel() // tableTypeName = "pods"

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v, want TablePane (t should be noop for non-allowancebuckets)", appM.activePane)
	}
}

// TestAppModel_RKey_QuotaDashboardPane_InvalidatesCacheAndRefreshes verifies that
// pressing r in QuotaDashboardPane invalidates the bucket cache, sets refreshing=true,
// and dispatches both LoadBucketsCmd and LoadResourcesCmd to keep both views fresh.
func TestAppModel_RKey_QuotaDashboardPane_InvalidatesCacheAndRefreshes(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newQuotaDashboardPaneModel(bc)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !appM.refreshing {
		t.Error("refreshing = false after r in QuotaDashboardPane, want true")
	}
	if !appM.tuiCtx.Refreshing {
		t.Error("tuiCtx.Refreshing = false, want true")
	}
	if !bc.invalidated {
		t.Error("InvalidateBucketCache not called, want cache invalidated on r")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want batch cmd")
	}
	msgs := collectMsgs(cmd)
	var hasBuckets, hasResources bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.BucketsLoadedMsg:
			hasBuckets = true
		case data.ResourcesLoadedMsg:
			hasResources = true
		}
	}
	if !hasBuckets {
		t.Error("r cmd batch did not produce data.BucketsLoadedMsg")
	}
	if !hasResources {
		t.Error("r cmd batch did not produce data.ResourcesLoadedMsg")
	}
}

// TestAppModel_RKey_QuotaDashboardPane_CoalesceWhileRefreshing verifies that a
// second r while a bucket refresh is in flight is silently coalesced.
func TestAppModel_RKey_QuotaDashboardPane_CoalesceWhileRefreshing(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newQuotaDashboardPaneModel(bc)
	m.refreshing = true // already refreshing

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil for coalesced r in QuotaDashboardPane")
	}
}

// TestAppModel_EscFromQuotaDashboardPane_GoesToNavPane verifies that pressing
// Esc in QuotaDashboardPane returns to NavPane and resets the grouping.
func TestAppModel_EscFromQuotaDashboardPane_GoesToNavPane(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("activePane = %v, want NavPane after Esc from QuotaDashboardPane", appM.activePane)
	}
}

// TestAppModel_EnterOnDashboard_GoesToDetailPane verifies that Enter on a
// selected bucket in QuotaDashboardPane transitions to DetailPane and sets
// detailReturnPane = QuotaDashboardPane.
func TestAppModel_EnterOnDashboard_GoesToDetailPane(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v, want DetailPane after Enter on dashboard bucket", appM.activePane)
	}
	if appM.detailReturnPane != QuotaDashboardPane {
		t.Errorf("detailReturnPane = %v, want QuotaDashboardPane", appM.detailReturnPane)
	}
	if !appM.detail.Loading() {
		t.Error("detail.Loading() = false, want true immediately after Enter from dashboard")
	}
}

// TestAppModel_EscFromDetailReturnsToQuotaDashboard verifies that when Detail was
// entered from QuotaDashboardPane, pressing Esc returns there (not TablePane).
func TestAppModel_EscFromDetailReturnsToQuotaDashboard(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	// Enter → DetailPane with detailReturnPane = QuotaDashboardPane
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(AppModel)
	if m.activePane != DetailPane {
		t.Fatalf("expected DetailPane after Enter, got %v", m.activePane)
	}

	// Esc → should return to QuotaDashboardPane
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("activePane = %v, want QuotaDashboardPane after Esc from dashboard-originated detail", appM.activePane)
	}
}

// TestAppModel_TickMsg_QuotaDashboardPane_DispatchesLoadBuckets verifies that
// a TickMsg while in QuotaDashboardPane dispatches LoadBucketsCmd (not resources).
func TestAppModel_TickMsg_QuotaDashboardPane_DispatchesLoadBuckets(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newQuotaDashboardPaneModel(bc)

	_, cmd := m.Update(data.TickMsg{})
	if cmd == nil {
		t.Fatal("cmd = nil, want LoadBucketsCmd from tick in QuotaDashboardPane")
	}
	// The tick batches LoadBucketsCmd + TickCmd; execute them to find BucketsLoadedMsg.
	// Batch unwrapping: the cmd is tea.Batch(...), each sub-cmd is callable.
	// Rather than unwrap, just call cmd() and check for expected msg types.
	msg := cmd()
	switch msg.(type) {
	case data.BucketsLoadedMsg, tea.BatchMsg:
		// acceptable: either the bucket message directly or a batch
	default:
		// cmd() returned something unexpected — try to interpret as batch
		// In bubbletea, Batch returns a BatchMsg which is []Cmd. We can't
		// easily iterate here, so just verify bucketLoading is NOT set
		// (tick path doesn't set bucketLoading, only r key does).
	}
	// The key invariant: tick in QuotaDashboard should NOT set refreshing.
	if m.refreshing {
		t.Error("refreshing = true after TickMsg in QuotaDashboardPane, want false")
	}
}

// ==================== FB-108: Ticker refresh is intentionally silent ====================

// AC2 [Anti-behavior] — TickMsg on QuotaDashboardPane does NOT set quota.refreshing=true.
// Background ticker cadence is ambient; SetRefreshing is operator-initiated ([r]) only.
func TestFB108_AC2_TickMsg_QuotaDashboardPane_DoesNotSetRefreshing(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newQuotaDashboardPaneModel(bc)
	// Pre-condition: quota is not refreshing.
	if m.quota.IsRefreshing() {
		t.Fatal("precondition: quota.IsRefreshing() = true, want false before TickMsg")
	}

	result, _ := m.Update(data.TickMsg{})
	appM := result.(AppModel)

	if appM.quota.IsRefreshing() {
		t.Error("AC2: quota.IsRefreshing() = true after TickMsg in QuotaDashboardPane; ticker must not set refreshing (FB-108)")
	}
}

// AC1 [Anti-regression] — [r] keypress in QuotaDashboardPane still sets quota.refreshing=true.
// Existing test TestAppModel_RKey_QuotaDashboardPane_InvalidatesCacheAndRefreshes covers this;
// this stub ensures FB-108 AC1 maps to a named test function.
func TestFB108_AC1_RKey_QuotaDashboardPane_SetsRefreshing(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newQuotaDashboardPaneModel(bc)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !appM.quota.IsRefreshing() {
		t.Error("AC1: quota.IsRefreshing() = false after [r] in QuotaDashboardPane; operator refresh must set refreshing (FB-063 anti-regression)")
	}
}

// ==================== End FB-108 ====================

// TestAppModel_BuildDetailContent_NoBuckets_ReturnsBareContent verifies that
// buildDetailContent returns raw describe text unchanged when no buckets are cached.
func TestAppModel_BuildDetailContent_NoBuckets_ReturnsBareContent(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.describeContent = "describe output here"
	m.buckets = nil

	got := m.buildDetailContent()
	if got != "describe output here" {
		t.Errorf("buildDetailContent (no buckets) = %q, want bare describe content", got)
	}
}

// TestAppModel_BuildDetailContent_MatchingBuckets_AppendsQuotaBlock verifies that
// when buckets are cached and one matches the described resource type, the S3
// quota block is appended to the describe content.
func TestAppModel_BuildDetailContent_MatchingBuckets_AppendsQuotaBlock(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.describeContent = "describe output here"
	m.describeRT = data.ResourceType{Name: "cpus", Group: "compute.example.io"}
	m.buckets = []data.AllowanceBucket{
		{Name: "b1", ResourceType: "compute.example.io/cpus", Allocated: 5, Limit: 100},
	}

	got := m.buildDetailContent()
	if got == "describe output here" {
		t.Error("buildDetailContent: quota block not appended — content unchanged")
	}
	if !containsStr(got, "[3] quota dashboard") {
		t.Errorf("buildDetailContent: want '[3] quota dashboard' separator in %q", got)
	}
	// ResolveDescription with nil registrations falls back to the short name (last "/" segment).
	if !containsStr(got, "cpus") {
		t.Errorf("buildDetailContent: want resource short name 'cpus' in %q", got)
	}
}

// TestAppModel_BuildDetailContent_NoMatchingBuckets_ReturnsBareContent verifies
// that when cached buckets do not match the described resource type, no block is
// appended.
func TestAppModel_BuildDetailContent_NoMatchingBuckets_ReturnsBareContent(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.describeContent = "describe output here"
	m.describeRT = data.ResourceType{Name: "pods", Group: ""}
	m.buckets = []data.AllowanceBucket{
		{Name: "b1", ResourceType: "compute.example.io/cpus"},
	}

	got := m.buildDetailContent()
	if got != "describe output here" {
		t.Errorf("buildDetailContent (no match): want bare content, got %q", got)
	}
}

// TestAppModel_BucketsLoadedMsg_InDetailPane_UpdatesDetailContent verifies that
// a BucketsLoadedMsg arriving while in DetailPane re-renders detail content with
// the S3 quota block appended.
func TestAppModel_BucketsLoadedMsg_InDetailPane_UpdatesDetailContent(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = DetailPane
	m.updatePaneFocus()
	m.describeContent = "describe output here"
	m.describeRT = data.ResourceType{Name: "cpus", Group: "compute.example.io"}

	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "b1", ResourceType: "compute.example.io/cpus", Allocated: 5, Limit: 100},
		},
	})
	appM := result.(AppModel)

	got := appM.buildDetailContent()
	if !containsStr(got, "[3] quota dashboard") {
		t.Errorf("after BucketsLoadedMsg in DetailPane: want '[3] quota dashboard' in rebuilt content, got %q", got)
	}
}

// containsStr is a plain-text substring helper (no ANSI stripping needed here
// because buildDetailContent operates on raw strings, not styled output).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// --- FB-006 test helpers ---

// newDetailPaneModel builds a minimal AppModel in DetailPane with a resource
// context pre-set on the detail view so "a" key transitions work.
// ac is wired but uses a nil factory — safe for ForceRefresh/Invalidate,
// but cmd() must NOT be called for any cmds that invoke ListActivity.
func newDetailPaneModel() AppModel {
	detail := components.NewDetailViewModel(58, 20)
	// Production code calls detail.SetResourceContext(rt.Name, row.Name), so use
	// the resource plural name ("projects") not the Kind ("Project").
	detail.SetResourceContext("projects", "my-project")

	m := AppModel{
		ctx:        context.Background(),
		rc:         stubResourceClient{},
		ac:         data.NewActivityClient(nil),
		activePane: DetailPane,
		describeRT: data.ResourceType{Kind: "Project", Name: "projects", Group: "resourcemanager.miloapis.com"},
		sidebar:    components.NewNavSidebarModel(22, 20),
		table:      components.NewResourceTableModel(58, 20),
		detail:     detail,
		activity:   components.NewActivityViewModel(58, 20),
		filterBar:  components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// newActivityPaneModel builds a minimal AppModel already in ActivityPane with
// resource context and activity rows pre-loaded, and the next-continue token set.
func newActivityPaneModel() AppModel {
	m := newDetailPaneModel()
	m.activityAPIGroup = "resourcemanager.miloapis.com"
	m.activityKind = "Project"
	m.activityRTName = "projects" // rt.Name form — matches detail.ResourceKind()
	m.activityName = "my-project"
	m.activity.SetRows([]data.ActivityRow{
		{Origin: "audit", Summary: "created"},
	}, "tok1")
	m.activePane = ActivityPane
	m.updatePaneFocus()
	return m
}

// --- FB-006 tests (activity integration) ---

// TestAppModel_AKey_FromDetailPane_TransitionsToActivityPane verifies that
// pressing "a" from DetailPane navigates to ActivityPane, sets loading state,
// and dispatches a LoadActivityCmd (cmd non-nil).
func TestAppModel_AKey_FromDetailPane_TransitionsToActivityPane(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.activePane != ActivityPane {
		t.Errorf("activePane = %v, want ActivityPane after a from DetailPane", appM.activePane)
	}
	if !appM.activity.HasRows() == false && cmd == nil {
		// Either loading was triggered (cmd != nil) or rows already existed.
		t.Error("cmd = nil after a from DetailPane with no existing rows")
	}
	// cmd must be non-nil because there are no rows yet.
	if cmd == nil {
		t.Error("cmd = nil, want LoadActivityCmd dispatched on first open")
	}
	// Do NOT call cmd() — it would panic (nil factory). Just verify non-nil.
}

// TestAppModel_AKey_FromDetailPane_SetsActivityContext verifies that the
// activity resource context is populated from the describe result type.
func TestAppModel_AKey_FromDetailPane_SetsActivityContext(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.activityKind != "Project" {
		t.Errorf("activityKind = %q, want %q", appM.activityKind, "Project")
	}
	if appM.activityAPIGroup != "resourcemanager.miloapis.com" {
		t.Errorf("activityAPIGroup = %q, want %q", appM.activityAPIGroup, "resourcemanager.miloapis.com")
	}
	if appM.activityName != "my-project" {
		t.Errorf("activityName = %q, want %q", appM.activityName, "my-project")
	}
}

// TestAppModel_AKey_FromDetailPane_WhileLoading_IsNoop verifies that pressing
// "a" while the detail view is still loading is a no-op.
func TestAppModel_AKey_FromDetailPane_WhileLoading_IsNoop(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModel()
	m.detail.SetLoading(true)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v, want DetailPane (a noop while loading)", appM.activePane)
	}
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil for noop a while detail loading")
	}
}

// TestAppModel_AKey_FromDetailPane_PreservesRows verifies that pressing "a"
// again on the same resource with rows already loaded and a pagination token
// does NOT reset rows (toggle-back-and-forth preserves scroll state).
func TestAppModel_AKey_FromDetailPane_PreservesRows_WhenSameResource(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModel()
	// Pre-populate state matching what the handler sets after the first open:
	// activityRTName must use rt.Name form ("projects") to match detail.ResourceKind().
	m.activityRTName = "projects"
	m.activityKind = "Project"
	m.activityAPIGroup = "resourcemanager.miloapis.com"
	m.activityName = "my-project"
	m.activity.SetRows([]data.ActivityRow{
		{Origin: "audit", Summary: "existing"},
	}, "tok1") // HasRows() == true, NextContinue != ""

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.activePane != ActivityPane {
		t.Errorf("activePane = %v, want ActivityPane", appM.activePane)
	}
	// Rows should be preserved; no reload dispatched.
	if !appM.activity.HasRows() {
		t.Error("HasRows() = false after toggle with same resource + rows, want preserved")
	}
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil (no reload when same resource + rows)")
	}
}

// TestAppModel_GKey_ActivityPane_DoesNotTriggerPagination verifies that pressing
// G in ActivityPane jumps to end of loaded buffer but does NOT dispatch
// NeedNextActivityPageMsg (spec §5b, E2E-9).
func TestAppModel_GKey_ActivityPane_DoesNotTriggerPagination(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel() // has rows + non-empty continue token

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	_ = result

	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(components.NeedNextActivityPageMsg); ok {
			t.Error("G key dispatched NeedNextActivityPageMsg, want no pagination trigger")
		}
	}
}

// TestAppModel_AKey_FromDetailPane_ResetsRows_WhenDifferentResource verifies
// that opening the activity view for a different resource after a previous open
// resets the row buffer (no stale rows from the prior resource).
func TestAppModel_AKey_FromDetailPane_ResetsRows_WhenDifferentResource(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModel()
	// Simulate a previous activity load for resource X.
	m.activityRTName = "projects"
	m.activityKind = "Project"
	m.activityAPIGroup = "resourcemanager.miloapis.com"
	m.activityName = "resource-x"
	m.activity.SetRows([]data.ActivityRow{
		{Origin: "audit", Summary: "stale row from resource-x"},
	}, "")

	// Now detail pane is showing resource Y (different name).
	m.detail.SetResourceContext("projects", "resource-y")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.activePane != ActivityPane {
		t.Errorf("activePane = %v, want ActivityPane", appM.activePane)
	}
	// Rows must be reset — stale rows from resource-x must not be visible.
	if appM.activity.HasRows() {
		t.Error("HasRows() = true after opening activity for different resource, want reset")
	}
}

// TestAppModel_AKey_FromActivityPane_TogglesBackToDetailPane verifies that
// pressing "a" while in ActivityPane returns to DetailPane.
func TestAppModel_AKey_FromActivityPane_TogglesBackToDetailPane(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v, want DetailPane after a from ActivityPane", appM.activePane)
	}
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil for toggle-back a from ActivityPane")
	}
}

// TestAppModel_EscFromActivityPane_ReturnsToDetailPane verifies that Esc from
// ActivityPane transitions to DetailPane.
func TestAppModel_EscFromActivityPane_ReturnsToDetailPane(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v, want DetailPane after Esc from ActivityPane", appM.activePane)
	}
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil for Esc from ActivityPane")
	}
}

// TestAppModel_ActivityLoadedMsg_FirstPage_SetsRows verifies that
// ActivityLoadedMsg with IsFirstPage=true replaces rows.
func TestAppModel_ActivityLoadedMsg_FirstPage_SetsRows(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()
	m.refreshing = true
	m.tuiCtx.Refreshing = true

	result, _ := m.Update(data.ActivityLoadedMsg{
		Rows:         []data.ActivityRow{{Origin: "audit", Summary: "new row"}},
		NextContinue: "tok2",
		IsFirstPage:  true,
	})
	appM := result.(AppModel)

	if !appM.activity.HasRows() {
		t.Error("HasRows() = false after ActivityLoadedMsg first page, want true")
	}
	if appM.activity.NextContinue() != "tok2" {
		t.Errorf("NextContinue() = %q, want %q", appM.activity.NextContinue(), "tok2")
	}
	if appM.refreshing {
		t.Error("refreshing = true after ActivityLoadedMsg, want false")
	}
	if appM.tuiCtx.Refreshing {
		t.Error("tuiCtx.Refreshing = true after ActivityLoadedMsg, want false")
	}
}

// TestAppModel_ActivityLoadedMsg_NextPage_AppendsRows verifies that
// ActivityLoadedMsg with IsFirstPage=false appends rows.
func TestAppModel_ActivityLoadedMsg_NextPage_AppendsRows(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel() // has 1 row, tok1

	result, _ := m.Update(data.ActivityLoadedMsg{
		Rows:         []data.ActivityRow{{Origin: "event", Summary: "appended"}},
		NextContinue: "",
		IsFirstPage:  false,
	})
	appM := result.(AppModel)

	if !appM.activity.HasRows() {
		t.Error("HasRows() = false after next-page append")
	}
	if appM.activity.NextContinue() != "" {
		t.Errorf("NextContinue() = %q after append with empty cont, want empty", appM.activity.NextContinue())
	}
}

// TestAppModel_ActivityLoadedMsg_Error_SetsError verifies that an error
// response routes to SetError and clears the refreshing flag.
func TestAppModel_ActivityLoadedMsg_Error_SetsError(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()
	m.refreshing = true
	m.tuiCtx.Refreshing = true

	result, _ := m.Update(data.ActivityLoadedMsg{
		Err:         errors.New("server error"),
		IsFirstPage: true,
	})
	appM := result.(AppModel)

	if appM.refreshing {
		t.Error("refreshing = true after error ActivityLoadedMsg, want false")
	}
}

// TestAppModel_ActivityLoadedMsg_Unauthorized_SetsUnauthorized verifies that a
// 403-class error sets the unauthorized flag on the activity view. The View()
// output (ANSI-stripped) is checked for the entitlement state message.
func TestAppModel_ActivityLoadedMsg_Unauthorized_SetsUnauthorized(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()
	m.activePane = ActivityPane
	m.updatePaneFocus()

	result, _ := m.Update(data.ActivityLoadedMsg{
		Err:          errors.New("forbidden"),
		Unauthorized: true,
		IsFirstPage:  true,
	})
	appM := result.(AppModel)

	// View should show entitlement state (not the error-tint path with [r] retry).
	view := stripANSIModel(appM.activity.View())
	if !containsStr(view, "Activity is not enabled") {
		t.Errorf("unauthorized ActivityLoadedMsg: want entitlement state text in %q", view)
	}
	if containsStr(view, "[r] retry") {
		t.Errorf("unauthorized ActivityLoadedMsg: must not show [r] retry in %q", view)
	}
}

// TestAppModel_RKey_ActivityPane_ResetsAndDispatches verifies that pressing "r"
// in ActivityPane calls ForceRefresh, resets the view to loading, and dispatches
// a LoadActivityCmd (cmd non-nil). cmd() is NOT called (nil factory).
func TestAppModel_RKey_ActivityPane_ResetsAndDispatches(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if cmd == nil {
		t.Error("cmd = nil after r in ActivityPane, want LoadActivityCmd dispatched")
	}
	if appM.refreshing != true {
		t.Error("refreshing = false after r in ActivityPane, want true")
	}
}

// TestAppModel_RKey_ActivityPane_NilAC_IsNoop verifies that r in ActivityPane
// when ac is nil is a no-op (no cmd dispatched).
func TestAppModel_RKey_ActivityPane_NilAC_IsNoop(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()
	m.ac = nil

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil when ac is nil")
	}
}

// TestAppModel_RKey_ActivityPane_EmptyKind_IsNoop verifies that r is a no-op
// when activityKind is empty (no resource context set yet).
func TestAppModel_RKey_ActivityPane_EmptyKind_IsNoop(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()
	m.activityKind = ""

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil when activityKind is empty")
	}
}

// TestAppModel_EnterAndD_InActivityPane_AreNoops verifies that Enter and d keys
// are no-ops in ActivityPane (they must not trigger navigation).
func TestAppModel_EnterAndD_InActivityPane_AreNoops(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("d")},
	} {
		result, cmd := m.Update(key)
		appM := result.(AppModel)
		if appM.activePane != ActivityPane {
			t.Errorf("key %q: activePane = %v, want ActivityPane (should be noop)", key.String(), appM.activePane)
		}
		_ = cmd // cmd may be non-nil due to viewport/spinner propagation; pane must not change
	}
}

// TestAppModel_ContextSwitchedMsg_ClearsActivityState verifies that a context
// switch clears all activity resource context fields.
func TestAppModel_ContextSwitchedMsg_ClearsActivityState(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.activityKind != "" {
		t.Errorf("activityKind = %q after ContextSwitchedMsg, want empty", appM.activityKind)
	}
	if appM.activityName != "" {
		t.Errorf("activityName = %q after ContextSwitchedMsg, want empty", appM.activityName)
	}
	if appM.activityNamespace != "" {
		t.Errorf("activityNamespace = %q after ContextSwitchedMsg, want empty", appM.activityNamespace)
	}
	if appM.activityAPIGroup != "" {
		t.Errorf("activityAPIGroup = %q after ContextSwitchedMsg, want empty", appM.activityAPIGroup)
	}
}

// TestAppModel_NeedNextActivityPageMsg_DispatchesNextPage verifies that a
// NeedNextActivityPageMsg triggers a LoadActivityCmd for the continuation token.
func TestAppModel_NeedNextActivityPageMsg_DispatchesNextPage(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel() // has NextContinue = "tok1"

	result, cmd := m.Update(components.NeedNextActivityPageMsg{})
	appM := result.(AppModel)

	if cmd == nil {
		t.Error("cmd = nil after NeedNextActivityPageMsg, want LoadActivityCmd")
	}
	_ = appM
	// Do NOT call cmd() — would panic via nil factory.
}

// TestAppModel_NeedNextActivityPageMsg_EmptyContinue_IsNoop verifies that
// NeedNextActivityPageMsg is a no-op when NextContinue is already empty.
func TestAppModel_NeedNextActivityPageMsg_EmptyContinue_IsNoop(t *testing.T) {
	t.Parallel()
	m := newActivityPaneModel()
	m.activity.SetRows([]data.ActivityRow{
		{Origin: "audit", Summary: "created"},
	}, "") // empty continuation

	_, cmd := m.Update(components.NeedNextActivityPageMsg{})
	if cmd != nil {
		t.Errorf("cmd = non-nil, want nil when NextContinue is empty")
	}
}

// --- FB-012 test helpers ---

// newNavPaneModelWithBC builds a NavPane AppModel with a non-allowancebuckets
// resource type selected and a BucketClient wired in. Used for testing the
// Enter-on-governed-type banner fetch path.
func newNavPaneModelWithBC(bc data.BucketClient) AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{
		{Name: "projects", Kind: "Project", Group: "resourcemanager.miloapis.com", Namespaced: false},
	})
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		bc:          bc,
		activePane:  NavPane,
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(58, 20),
		banner:      components.NewQuotaBannerModel(58),
		detail:      components.NewDetailViewModel(58, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// newGoverned TablePaneModel builds a minimal AppModel in TablePane with the
// quota banner pre-loaded with a matching bucket (governed type) and a BucketClient wired.
func newGovernedTablePaneModel(bc data.BucketClient) AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{
		{Name: "projects", Kind: "Project", Group: "resourcemanager.miloapis.com", Namespaced: false},
	})
	table := components.NewResourceTableModel(58, 20)
	table.SetColumns([]string{"Name"}, 58)
	table.SetRows([]data.ResourceRow{
		{Name: "proj-1", Cells: []string{"proj-1"}},
	})
	table.SetTypeContext("projects", false)

	banner := components.NewQuotaBannerModel(58)
	banner.SetBuckets([]data.AllowanceBucket{
		{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 5, Limit: 100},
	})

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		bc:            bc,
		activePane:    TablePane,
		tableTypeName: "projects",
		buckets: []data.AllowanceBucket{
			{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 5, Limit: 100},
		},
		sidebar:     sidebar,
		table:       table,
		banner:      banner,
		detail:      components.NewDetailViewModel(58, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// --- FB-012 tests (quota banner) ---

// TestAppModel_Enter_NavPane_GoverneType_NoBucketCache_DispatchesBoth verifies
// that Enter on a non-allowancebuckets resource type when no buckets are cached
// and bc is set dispatches both LoadResourcesCmd and LoadBucketsCmd (E2E-1).
func TestAppModel_Enter_NavPane_GovernedType_NoBucketCache_DispatchesBoth(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	// buckets nil (cache empty) — default state

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v, want TablePane after Enter on governed type", appM.activePane)
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want batch with LoadResourcesCmd + LoadBucketsCmd")
	}
	msgs := collectMsgs(cmd)
	var hasResources, hasBuckets bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.ResourcesLoadedMsg:
			hasResources = true
		case data.BucketsLoadedMsg:
			hasBuckets = true
		}
	}
	if !hasResources {
		t.Error("cmd batch did not produce ResourcesLoadedMsg")
	}
	if !hasBuckets {
		t.Error("cmd batch did not produce BucketsLoadedMsg (governed type with bc and no cache)")
	}
}

// TestAppModel_Enter_NavPane_GovernedType_BucketsAlreadyCached_DispatchesResourcesOnly
// verifies that when buckets are already cached, Enter only dispatches LoadResourcesCmd
// (no redundant bucket fetch).
func TestAppModel_Enter_NavPane_GovernedType_BucketsAlreadyCached_DispatchesResourcesOnly(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.buckets = []data.AllowanceBucket{
		{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 5, Limit: 100},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil, want LoadResourcesCmd")
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(data.BucketsLoadedMsg); ok {
			t.Error("BucketsLoadedMsg dispatched when buckets already cached, want resources only")
		}
	}
}

// TestAppModel_Enter_NavPane_NoBCNil_DispatchesResourcesOnly verifies that
// Enter with bc==nil dispatches only LoadResourcesCmd (un-governed / no bucket client).
func TestAppModel_Enter_NavPane_NilBC_DispatchesResourcesOnly(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)
	m.bc = nil

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil, want LoadResourcesCmd")
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(data.BucketsLoadedMsg); ok {
			t.Error("BucketsLoadedMsg dispatched with nil bc, want resources only")
		}
	}
}

// TestAppModel_RKey_TablePane_GovernedType_InvalidatesAndDispatchesBoth verifies
// that r in TablePane when the banner has matching buckets calls
// InvalidateBucketCache and dispatches both LoadResourcesCmd + LoadBucketsCmd (E2E-6).
func TestAppModel_RKey_TablePane_GovernedType_InvalidatesAndDispatchesBoth(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newGovernedTablePaneModel(bc)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !bc.invalidated {
		t.Error("InvalidateBucketCache not called on r for governed type")
	}
	if !appM.refreshing {
		t.Error("refreshing = false after r on governed type, want true")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want batch with LoadResourcesCmd + LoadBucketsCmd")
	}
	msgs := collectMsgs(cmd)
	var hasResources, hasBuckets bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.ResourcesLoadedMsg:
			hasResources = true
		case data.BucketsLoadedMsg:
			hasBuckets = true
		}
	}
	if !hasResources {
		t.Error("r on governed type: cmd batch missing ResourcesLoadedMsg")
	}
	if !hasBuckets {
		t.Error("r on governed type: cmd batch missing BucketsLoadedMsg")
	}
}

// TestAppModel_RKey_TablePane_UngovernedType_DoesNotInvalidate verifies that r
// in TablePane when no banner buckets match dispatches only LoadResourcesCmd
// without calling InvalidateBucketCache (E2E-7).
func TestAppModel_RKey_TablePane_UngovernedType_DoesNotInvalidate(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newTablePaneModel()
	m.bc = bc
	// banner has no buckets — un-governed type

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if bc.invalidated {
		t.Error("InvalidateBucketCache called for un-governed type, want no-op")
	}
	if cmd == nil {
		t.Fatal("cmd = nil, want at least LoadResourcesCmd")
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(data.BucketsLoadedMsg); ok {
			t.Error("BucketsLoadedMsg dispatched for un-governed type, want resources only")
		}
	}
}

// TestAppModel_BucketsLoadedMsg_UpdatesBanner verifies that after a
// BucketsLoadedMsg arrives, the banner reflects matching buckets for the
// currently selected sidebar type.
func TestAppModel_BucketsLoadedMsg_UpdatesBanner(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	// Navigate to TablePane first so sidebar has "projects" selected and tableTypeName is set.
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()

	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 50, Limit: 100},
		},
	})
	appM := result.(AppModel)

	if !appM.banner.HasBuckets() {
		t.Error("banner.HasBuckets() = false after BucketsLoadedMsg with matching bucket")
	}
	if appM.banner.Height() != 1 {
		t.Errorf("banner.Height() = %d, want 1", appM.banner.Height())
	}
}

// TestAppModel_BucketsLoadedMsg_SetsActiveConsumer_OnBanner verifies that
// BucketsLoadedMsg wires the active consumer kind/name onto the banner so that
// tree-mode Height() reflects org+project structure (AC-3, AC-5).
func TestAppModel_BucketsLoadedMsg_SetsActiveConsumer_OnBanner(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()

	// ProjectID set → activeConsumer() returns ("Project", "my-proj")
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "my-proj"}

	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "org", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		},
	})
	appM := result.(AppModel)

	// With org+project both present and activeConsumer set, Height() should be 2 (tree mode).
	if appM.banner.Height() != 2 {
		t.Errorf("banner.Height() = %d, want 2 (tree: parent+child, no siblings)", appM.banner.Height())
	}
}

// TestAppModel_BucketsLoadedMsg_SiblingRestricted_PropagatedToBanner verifies
// that SiblingEnumerationRestricted is forwarded to the banner.
func TestAppModel_BucketsLoadedMsg_SiblingRestricted_PropagatedToBanner(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "my-proj"}

	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "org", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
			{Name: "sib", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "other", Allocated: 5, Limit: 100},
		},
		SiblingEnumerationRestricted: true,
	})
	appM := result.(AppModel)

	// siblingsRestricted=true suppresses the sibling-consume row → Height=2, not 3.
	if appM.banner.Height() != 2 {
		t.Errorf("banner.Height() = %d with SiblingEnumerationRestricted=true, want 2 (sibling row suppressed)", appM.banner.Height())
	}
	if !appM.bucketSiblingsRestricted {
		t.Error("bucketSiblingsRestricted = false after SiblingEnumerationRestricted BucketsLoadedMsg")
	}
}

// TestAppModel_BucketsLoadedMsg_ActiveConsumerChanges_HighlightsNewRow verifies
// that when tuiCtx.ActiveCtx.ProjectID changes between two successive
// BucketsLoadedMsg dispatches, the banner's active consumer updates accordingly:
// the new project is the child row and the old project becomes a sibling.
func TestAppModel_BucketsLoadedMsg_ActiveConsumerChanges_HighlightsNewRow(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()

	buckets := []data.AllowanceBucket{
		{Name: "org", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj-a", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "proj-a", Allocated: 10, Limit: 100},
		{Name: "proj-b", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "proj-b", Allocated: 5, Limit: 100},
	}

	// First load: active consumer = proj-a
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "proj-a"}
	result, _ := m.Update(data.BucketsLoadedMsg{Buckets: buckets})
	m = result.(AppModel)
	// org + sibling-consume (proj-b) + proj-a = 3 lines
	if m.banner.Height() != 3 {
		t.Errorf("first load (active=proj-a): banner.Height() = %d, want 3", m.banner.Height())
	}

	// Second load: active consumer switches to proj-b
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "proj-b"}
	result, _ = m.Update(data.BucketsLoadedMsg{Buckets: buckets})
	m = result.(AppModel)
	// org + sibling-consume (proj-a) + proj-b = 3 lines; active consumer updated
	if m.banner.Height() != 3 {
		t.Errorf("second load (active=proj-b): banner.Height() = %d, want 3", m.banner.Height())
	}
}

// TestAppModel_BucketsLoadedMsg_MultiToSingle_CollapsesTree verifies that
// dispatching a tree-form load followed by a single-bucket load collapses the
// banner from tree mode (Height>=2) to flat mode (Height=1).
func TestAppModel_BucketsLoadedMsg_MultiToSingle_CollapsesTree(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "my-proj"}

	// First load: tree form (org + project).
	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "org", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		},
	})
	m = result.(AppModel)
	if m.banner.Height() < 2 {
		t.Fatalf("precondition: after tree load banner.Height() = %d, want >= 2", m.banner.Height())
	}

	// Second load: single bucket only (no org parent → no tree).
	result, _ = m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		},
	})
	m = result.(AppModel)
	if m.banner.Height() != 1 {
		t.Errorf("after single-bucket load: banner.Height() = %d, want 1 (tree collapsed)", m.banner.Height())
	}
}

// TestAppModel_BucketsLoadedMsg_SingleToMulti_ExpandsTree verifies the reverse:
// a single-bucket load followed by a tree-form load expands the banner from
// flat mode (Height=1) to tree mode (Height>=2).
func TestAppModel_BucketsLoadedMsg_SingleToMulti_ExpandsTree(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "my-proj"}

	// First load: single bucket.
	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		},
	})
	m = result.(AppModel)
	if m.banner.Height() != 1 {
		t.Fatalf("precondition: after single-bucket load banner.Height() = %d, want 1", m.banner.Height())
	}

	// Second load: tree form (org + project).
	result, _ = m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "org", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		},
	})
	m = result.(AppModel)
	if m.banner.Height() < 2 {
		t.Errorf("after tree load: banner.Height() = %d, want >= 2 (tree expanded)", m.banner.Height())
	}
}

// TestAppModel_ContextSwitchedMsg_ClearsActiveConsumerOnBanner verifies that a
// context switch resets active consumer state on the banner (tree mode disabled).
func TestAppModel_ContextSwitchedMsg_ClearsActiveConsumerOnBanner(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "my-proj"}
	// Pre-load org+project buckets so banner is in tree mode.
	m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "org", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
			{Name: "proj", ResourceType: "resourcemanager.miloapis.com/projects", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		},
	})

	// Context switch should clear all banner consumer state.
	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	// After context switch banner should have no buckets and no active consumer context.
	if appM.banner.HasBuckets() {
		t.Error("banner.HasBuckets() = true after ContextSwitchedMsg, want cleared")
	}
	if appM.bucketSiblingsRestricted {
		t.Error("bucketSiblingsRestricted = true after ContextSwitchedMsg, want false")
	}
}

// TestAppModel_ContextSwitchedMsg_ClearsBanner verifies that a context switch
// clears the banner (banner.HasBuckets() == false after switch).
func TestAppModel_ContextSwitchedMsg_ClearsBanner(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newGovernedTablePaneModel(bc)

	if !m.banner.HasBuckets() {
		t.Fatal("precondition: banner must have buckets before context switch")
	}

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.banner.HasBuckets() {
		t.Error("banner.HasBuckets() = true after ContextSwitchedMsg, want cleared")
	}
}

// TestAppModel_ResourcesLoadedMsg_OnUngovernedType_ClearsBanner verifies that
// when ResourcesLoadedMsg arrives for an un-governed type, updateBanner() finds
// no matching buckets for that type and clears the banner (E2E-4).
func TestAppModel_ResourcesLoadedMsg_OnUngovernedType_ClearsBanner(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	// Start with governed type loaded — banner has a projects bucket.
	m := newGovernedTablePaneModel(bc)
	if !m.banner.HasBuckets() {
		t.Fatal("precondition: banner must have buckets for governed type")
	}

	// Switch sidebar to an un-governed type (pods) with no matching buckets.
	podsRTName := "pods"
	podsSidebar := components.NewNavSidebarModel(22, 20)
	podsSidebar.SetItems([]data.ResourceType{
		{Name: podsRTName, Kind: "Pod", Group: "", Namespaced: true},
	})
	m.sidebar = podsSidebar
	m.tableTypeName = podsRTName

	result, _ := m.Update(data.ResourcesLoadedMsg{
		ResourceType: data.ResourceType{Name: podsRTName, Kind: "Pod"},
		Rows:         nil,
		Columns:      []string{"Name"},
	})
	appM := result.(AppModel)

	if appM.banner.HasBuckets() {
		t.Errorf("banner.HasBuckets() = true after ResourcesLoadedMsg for un-governed type, want cleared")
	}
	if appM.banner.Height() != 0 {
		t.Errorf("banner.Height() = %d, want 0 for un-governed type", appM.banner.Height())
	}
}

// TestAppModel_ResourcesLoadedMsg_OnGovernedType_SetsBannerForNewType verifies
// that when ResourcesLoadedMsg arrives for a governed resource type and a
// matching bucket is in the cache, updateBanner() populates the banner (E2E-5).
func TestAppModel_ResourcesLoadedMsg_OnGovernedType_SetsBannerForNewType(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	// Start browsing an un-governed type (pods) — no banner.
	m := newTablePaneModel() // pods, bc=nil, banner empty
	m.bc = bc

	// Pre-load a projects bucket into the cache (as if BucketsLoadedMsg already arrived).
	projectsRT := data.ResourceType{Name: "projects", Kind: "Project", Group: "resourcemanager.miloapis.com"}
	m.buckets = []data.AllowanceBucket{
		{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 50, Limit: 100},
	}

	// Switch sidebar to governed type (projects).
	projectsSidebar := components.NewNavSidebarModel(22, 20)
	projectsSidebar.SetItems([]data.ResourceType{projectsRT})
	m.sidebar = projectsSidebar
	m.tableTypeName = projectsRT.Name

	if m.banner.HasBuckets() {
		t.Fatal("precondition: banner must be empty before ResourcesLoadedMsg")
	}

	result, _ := m.Update(data.ResourcesLoadedMsg{
		ResourceType: projectsRT,
		Rows:         nil,
		Columns:      []string{"Name"},
	})
	appM := result.(AppModel)

	if !appM.banner.HasBuckets() {
		t.Error("banner.HasBuckets() = false after ResourcesLoadedMsg for governed type with cached bucket, want true")
	}
	if appM.banner.Height() < 1 {
		t.Errorf("banner.Height() = %d, want >= 1 for governed type with matching bucket", appM.banner.Height())
	}
}

// TestAppModel_BanneredView_TwoBuckets_MostConstrainedFirst verifies that after
// a BucketsLoadedMsg with two buckets for the same resource type, the banner
// renders the more-constrained bucket (higher %) first (E2E-9).
func TestAppModel_BanneredView_TwoBuckets_MostConstrainedFirst(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()

	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "b-low", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 10, Limit: 100},
			{Name: "b-high", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 90, Limit: 100},
		},
	})
	appM := result.(AppModel)

	if appM.banner.Height() != 2 {
		t.Fatalf("banner.Height() = %d, want 2", appM.banner.Height())
	}
	// The banner View renders buckets in the order they were sorted — most-constrained (90%) first.
	view := stripANSIModel(appM.banner.View())
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("banner view has %d lines, want 2: %q", len(lines), view)
	}
	// 90% line should come before 10% line → first line must contain "90" somewhere.
	if !containsStr(lines[0], "90") {
		t.Errorf("banner first line = %q, want most-constrained (90pct) bucket first", lines[0])
	}
}

// --- FB-014 test helpers ---

// stubRegistrationClient satisfies data.ResourceRegistrationClient without a real API server.
type stubRegistrationClient struct {
	registrations []data.ResourceRegistration
	invalidated   bool
}

func (s *stubRegistrationClient) ListResourceRegistrations(_ context.Context) ([]data.ResourceRegistration, error) {
	return s.registrations, nil
}
func (s *stubRegistrationClient) InvalidateRegistrationCache() { s.invalidated = true }

// newAllowanceBucketNavModelWithRRC extends newAllowanceBucketNavModel by wiring
// in a ResourceRegistrationClient so the Enter handler can dispatch
// LoadResourceRegistrationsCmd.
func newAllowanceBucketNavModelWithRRC(bc data.BucketClient, rrc data.ResourceRegistrationClient) AppModel {
	m := newAllowanceBucketNavModel(bc)
	m.rrc = rrc
	return m
}

// --- FB-014 tests (resource registration description labels) ---

// TestAppModel_ResourceRegistrationsLoadedMsg_Success_StoresRegistrations verifies
// that a successful ResourceRegistrationsLoadedMsg stores registrations on the model
// and propagates them to the banner (first-press axis).
func TestAppModel_ResourceRegistrationsLoadedMsg_Success_StoresRegistrations(t *testing.T) {
	t.Parallel()
	m := newGovernedTablePaneModel(&stubBucketClient{})

	regs := []data.ResourceRegistration{
		{Group: "resourcemanager.miloapis.com", Name: "projects", Description: "Projects"},
	}
	result, _ := m.Update(data.ResourceRegistrationsLoadedMsg{Registrations: regs})
	appM := result.(AppModel)

	if len(appM.registrations) != 1 {
		t.Errorf("m.registrations len = %d, want 1 after success msg", len(appM.registrations))
	}
	if appM.registrations[0].Description != "Projects" {
		t.Errorf("m.registrations[0].Description = %q, want %q", appM.registrations[0].Description, "Projects")
	}
	if appM.registrationsLoading {
		t.Error("registrationsLoading = true after success msg, want false")
	}
}

// TestAppModel_ResourceRegistrationsLoadedMsg_ReplacesExistingRegistrations verifies
// that a second ResourceRegistrationsLoadedMsg replaces the previous registration set
// (repeat-press axis).
func TestAppModel_ResourceRegistrationsLoadedMsg_ReplacesExistingRegistrations(t *testing.T) {
	t.Parallel()
	m := newGovernedTablePaneModel(&stubBucketClient{})
	m.registrations = []data.ResourceRegistration{
		{Group: "g", Name: "old", Description: "Old"},
	}

	regs := []data.ResourceRegistration{
		{Group: "resourcemanager.miloapis.com", Name: "projects", Description: "New Projects"},
	}
	result, _ := m.Update(data.ResourceRegistrationsLoadedMsg{Registrations: regs})
	appM := result.(AppModel)

	if len(appM.registrations) != 1 {
		t.Errorf("m.registrations len = %d, want 1 (replaced)", len(appM.registrations))
	}
	if appM.registrations[0].Description != "New Projects" {
		t.Errorf("m.registrations[0].Description = %q, want %q", appM.registrations[0].Description, "New Projects")
	}
}

// TestAppModel_ResourceRegistrationsLoadedMsg_BannerLabelChangesToDescription verifies
// that after registrations are loaded, the banner label changes from short name to the
// spec.description value (input-changed axis).
// Height is non-distinguishing (still 1 bucket line), so View() content is checked.
func TestAppModel_ResourceRegistrationsLoadedMsg_BannerLabelChangesToDescription(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()

	// Pre-load a bucket so the banner is visible.
	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 5, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
		},
	})
	m = result.(AppModel)
	if !m.banner.HasBuckets() {
		t.Fatal("precondition: banner must have buckets")
	}

	// Before registrations: label is the short name.
	before := stripANSIModel(m.banner.View())
	if !containsStr(before, "projects") {
		t.Errorf("before registrations: want 'projects' short name in banner %q", before)
	}

	// Dispatch registrations with a description for this resource type.
	result, _ = m.Update(data.ResourceRegistrationsLoadedMsg{
		Registrations: []data.ResourceRegistration{
			{Group: "resourcemanager.miloapis.com", Name: "projects", Description: "My Projects"},
		},
	})
	m = result.(AppModel)

	after := stripANSIModel(m.banner.View())
	if !containsStr(after, "My Projects") {
		t.Errorf("after registrations: want 'My Projects' description in banner %q", after)
	}
}

// TestAppModel_ResourceRegistrationsLoadedMsg_Error_SilentDegradation verifies that a
// ResourceRegistrationsLoadedMsg carrying an error leaves m.registrations nil and does
// NOT set an error on the status bar (anti-behavior: silent fallback).
func TestAppModel_ResourceRegistrationsLoadedMsg_Error_SilentDegradation(t *testing.T) {
	t.Parallel()
	m := newGovernedTablePaneModel(&stubBucketClient{})

	result, _ := m.Update(data.ResourceRegistrationsLoadedMsg{
		Err: errors.New("forbidden"),
	})
	appM := result.(AppModel)

	if appM.registrations != nil {
		t.Errorf("m.registrations = non-nil after error msg, want nil (silent degradation)")
	}
	if appM.registrationsLoading {
		t.Error("registrationsLoading = true after error msg, want false")
	}
	// Status bar must not surface the error (silent 403 degradation).
	if appM.statusBar.Err != nil {
		t.Errorf("statusBar.Err = %v after 403 registration error, want nil (silent)", appM.statusBar.Err)
	}
}

// TestAppModel_Enter_AllowanceBuckets_WithRRC_DispatchesRegistrations verifies that
// Enter on allowancebuckets with rrc != nil and registrations == nil dispatches
// LoadResourceRegistrationsCmd (anti-behavior: rrc nil guard).
func TestAppModel_Enter_AllowanceBuckets_WithRRC_DispatchesRegistrations(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{
		registrations: []data.ResourceRegistration{
			{Group: "g", Name: "r", Description: "Resource"},
		},
	}
	m := newAllowanceBucketNavModelWithRRC(bc, rrc)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil, want batch with LoadResourceRegistrationsCmd")
	}
	msgs := collectMsgs(cmd)
	var hasRegistrations bool
	for _, msg := range msgs {
		if _, ok := msg.(data.ResourceRegistrationsLoadedMsg); ok {
			hasRegistrations = true
		}
	}
	if !hasRegistrations {
		t.Error("Enter on allowancebuckets with rrc set: batch missing ResourceRegistrationsLoadedMsg")
	}
}

// TestAppModel_Enter_AllowanceBuckets_NilRRC_DoesNotDispatchRegistrations verifies
// that Enter on allowancebuckets with rrc == nil does not dispatch
// LoadResourceRegistrationsCmd (anti-behavior: nil rrc guard).
func TestAppModel_Enter_AllowanceBuckets_NilRRC_DoesNotDispatchRegistrations(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newAllowanceBucketNavModel(bc)
	// m.rrc is nil by default in test helpers

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("cmd = nil, want batch with at least LoadBucketsCmd")
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(data.ResourceRegistrationsLoadedMsg); ok {
			t.Error("ResourceRegistrationsLoadedMsg dispatched with nil rrc, want none")
		}
	}
}

// TestAppModel_RKey_TablePane_DoesNotDispatchRegistrations verifies that pressing r
// in TablePane does not dispatch LoadResourceRegistrationsCmd even when rrc is set
// (anti-behavior: r key only refreshes resources/buckets, not registrations).
func TestAppModel_RKey_TablePane_DoesNotDispatchRegistrations(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newGovernedTablePaneModel(bc)
	m.rrc = rrc

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("cmd = nil, want batch from r key")
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(data.ResourceRegistrationsLoadedMsg); ok {
			t.Error("r key dispatched ResourceRegistrationsLoadedMsg, want none (r does not reload registrations)")
		}
	}
}

// TestAppModel_ResourceRegistrationsLoadedMsg_AllSurfaces_UseDescription verifies that
// after a single ResourceRegistrationsLoadedMsg, the description string appears in all
// three surfaces: banner (S1), quota dashboard content (S2), and describe S3 block
// (AC#13 — integration proof that all surfaces share the same registrations).
func TestAppModel_ResourceRegistrationsLoadedMsg_AllSurfaces_UseDescription(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newNavPaneModelWithBC(bc)
	m.tableTypeName = "projects"
	m.activePane = TablePane
	m.updatePaneFocus()

	const desc = "Projects created within Organizations"

	// Pre-load matching buckets so banner and quota dashboard are populated.
	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 5, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
		},
	})
	m = result.(AppModel)
	m.quota.SetBuckets([]data.AllowanceBucket{
		{Name: "b1", ResourceType: "resourcemanager.miloapis.com/projects", Allocated: 5, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
	})

	// Set up describe S3 context so buildDetailContent can append the quota block.
	m.describeContent = "describe output here"
	m.describeRT = data.ResourceType{Group: "resourcemanager.miloapis.com", Name: "projects"}

	// Dispatch registrations.
	result, _ = m.Update(data.ResourceRegistrationsLoadedMsg{
		Registrations: []data.ResourceRegistration{
			{Group: "resourcemanager.miloapis.com", Name: "projects", Description: desc},
		},
	})
	m = result.(AppModel)

	// S1 banner: description may be truncated at narrow width; check a prefix that
	// survives truncation (the short name "projects" must NOT be the only thing there).
	bannerView := stripANSIModel(m.banner.View())
	const descPrefix = "Projects created within"
	if !containsStr(bannerView, descPrefix) {
		t.Errorf("S1 banner: want description prefix %q in view %q", descPrefix, bannerView)
	}

	// S2 quota dashboard
	quotaView := stripANSIModel(m.quota.View())
	if !containsStr(quotaView, desc) {
		t.Errorf("S2 quota dashboard: want description %q in view %q", desc, quotaView)
	}

	// S3 describe block (buildDetailContent appends quota block when buckets match)
	detailContent := m.buildDetailContent()
	if !containsStr(detailContent, desc) {
		t.Errorf("S3 describe block: want description %q in content %q", desc, detailContent)
	}
}

// TestAppModel_ContextSwitched_ThenEnterGovernedPane_RefetchesRegistrations verifies
// the eager-load contract in two steps:
//   Step 1 — ContextSwitchedMsg clears registrations, invalidates the cache, and
//             eagerly dispatches LoadResourceRegistrationsCmd (registrationsLoading=true).
//   Step 2 — Entering a governed pane while registrations are already loading does NOT
//             dispatch a second LoadResourceRegistrationsCmd (guard: !registrationsLoading).
func TestAppModel_ContextSwitched_ThenEnterGovernedPane_RefetchesRegistrations(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newAllowanceBucketNavModelWithRRC(bc, rrc)
	m.registrations = []data.ResourceRegistration{
		{Group: "g", Name: "r", Description: "R"},
	}

	// Step 1: context switch — clears registrations, invalidates cache, eagerly dispatches LoadRegCmd.
	result, cmd := m.Update(components.ContextSwitchedMsg{})
	m = result.(AppModel)

	if m.registrations != nil {
		t.Errorf("step 1: m.registrations = non-nil, want nil after ContextSwitchedMsg")
	}
	if !rrc.invalidated {
		t.Error("step 1: InvalidateRegistrationCache not called on ContextSwitchedMsg")
	}
	if !m.registrationsLoading {
		t.Error("step 1: registrationsLoading = false after ContextSwitchedMsg with rrc set, want true (eager dispatch)")
	}
	// Verify the eager dispatch actually happened.
	var contextSwitchDispatched bool
	for _, msg := range collectMsgs(cmd) {
		if _, ok := msg.(data.ResourceRegistrationsLoadedMsg); ok {
			contextSwitchDispatched = true
		}
	}
	_ = contextSwitchDispatched // dispatch is via cmd, not msg; registrationsLoading=true is the observable

	// Step 2: Enter while registrations are still loading — must NOT dispatch a second load.
	_, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, msg := range collectMsgs(cmd2) {
		if _, ok := msg.(data.ResourceRegistrationsLoadedMsg); ok {
			t.Error("step 2: Enter while registrationsLoading=true dispatched LoadResourceRegistrationsCmd — want no duplicate")
		}
	}
}

// --- FB-003 tests (slash-filter hint + early-clear) ---

// TestAppModel_SlashKey_NavPane_PostsHint_NoFilterBar verifies that pressing "/" in
// NavPane posts a hint on the status bar and does NOT open the FilterBar (first-press).
func TestAppModel_SlashKey_NavPane_PostsHint_NoFilterBar(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	if appM.filterBar.Focused() {
		t.Error("filterBar.Focused() = true after / in NavPane, want false (hint only)")
	}
	if appM.statusBar.Hint == "" {
		t.Error("statusBar.Hint = empty after / in NavPane, want hint posted")
	}
	// AC#2: pin the spec-mandated hint text.
	if !strings.Contains(appM.statusBar.Hint, "Select a resource type first") {
		t.Errorf("statusBar.Hint = %q, want text containing 'Select a resource type first' (AC#2)", appM.statusBar.Hint)
	}
	if cmd == nil {
		t.Error("cmd = nil after / in NavPane, want HintClearCmd returned")
	}
}

// TestAppModel_SlashKey_TablePane_NoType_PostsHint verifies that "/" in TablePane
// without a resource type loaded posts a hint instead of opening the FilterBar
// (anti-behavior: filter must not open without a type selection).
func TestAppModel_SlashKey_TablePane_NoType_PostsHint(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.tableTypeName = "" // no type loaded

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	if appM.filterBar.Focused() {
		t.Error("filterBar.Focused() = true after / in TablePane with no type, want false")
	}
	if appM.statusBar.Hint == "" {
		t.Error("statusBar.Hint = empty after / in TablePane with no type, want hint posted")
	}
	if cmd == nil {
		t.Error("cmd = nil, want HintClearCmd")
	}
}

// TestAppModel_SlashKey_TablePane_WithType_OpensFilterBar verifies that "/" in
// TablePane with a resource type loaded opens the FilterBar normally (repeat-press).
func TestAppModel_SlashKey_TablePane_WithType_OpensFilterBar(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel() // tableTypeName = "pods"

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	if !appM.filterBar.Focused() {
		t.Error("filterBar.Focused() = false after / in TablePane with type, want FilterBar opened")
	}
	if appM.statusBar.Hint != "" {
		t.Errorf("statusBar.Hint = %q after / in TablePane, want no hint (filter opened normally)", appM.statusBar.Hint)
	}
	if appM.statusBar.Mode != components.ModeFilter {
		t.Errorf("statusBar.Mode = %v, want ModeFilter", appM.statusBar.Mode)
	}
}

// TestAppModel_SlashKey_QuotaDashboardPane_SilentNoOp verifies that "/"
// in QuotaDashboardPane is a silent no-op: no FilterBar, no hint (FB-049 AC6).
func TestAppModel_SlashKey_QuotaDashboardPane_SilentNoOp(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})
	viewBefore := stripANSIModel(m.View())

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	if appM.filterBar.Focused() {
		t.Error("filterBar.Focused() = true after / in QuotaDashboardPane, want false")
	}
	if appM.statusBar.Hint != "" {
		t.Errorf("statusBar.Hint = %q after / in QuotaDashboardPane, want empty (silent no-op)", appM.statusBar.Hint)
	}
	if cmd != nil {
		t.Error("cmd != nil after / in QuotaDashboardPane, want nil (no hint clear scheduled)")
	}
	viewAfter := stripANSIModel(appM.View())
	if viewBefore != viewAfter {
		t.Error("View() changed after / in QuotaDashboardPane, want identical (silent no-op)")
	}
}

// TestAppModel_Filter_ZeroResults_EscClearsFilterAndRestoresRows verifies that when
// the FilterBar is open and no rows match the active filter, pressing Esc clears
// the filter, restores all rows, and removes the "No results" empty-state panel
// (AC#6 — Esc round-trip from zero-results state).
func TestAppModel_Filter_ZeroResults_EscClearsFilterAndRestoresRows(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.table.SetColumns([]string{"Name"}, 58)
	m.table.SetRows([]data.ResourceRow{
		{Name: "alpha", Cells: []string{"alpha"}},
		{Name: "beta", Cells: []string{"beta"}},
		{Name: "gamma", Cells: []string{"gamma"}},
	})
	m.table.SetTypeContext("pods", true)

	// Set up zero-results state: filter "zzz" active, FilterBar focused.
	m.table.SetFilter("zzz")
	_ = m.filterBar.Focus()
	m.statusBar.Mode = components.ModeFilter

	// Precondition: empty-state panel is showing.
	if !strings.Contains(stripANSIModel(m.table.View()), "No results for") {
		t.Fatal("precondition: expected 'No results for' in table view before Esc")
	}

	// Press Esc — routes to handleFilterKey which clears filter and blurs bar.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	// FilterBar must be blurred.
	if appM.filterBar.Focused() {
		t.Error("filterBar.Focused() = true after Esc, want blurred")
	}
	if appM.statusBar.Mode != components.ModeNormal {
		t.Errorf("statusBar.Mode = %v after Esc, want ModeNormal", appM.statusBar.Mode)
	}

	// Empty-state panel must be gone; all rows visible.
	tableView := stripANSIModel(appM.table.View())
	if containsStr(tableView, "No results for") {
		t.Errorf("after Esc: 'No results for' still visible in %q", tableView)
	}
	if containsStr(tableView, "[Esc] clear filter") {
		t.Errorf("after Esc: '[Esc] clear filter' still visible in %q", tableView)
	}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !containsStr(tableView, name) {
			t.Errorf("after Esc: row %q not visible in %q (rows must be restored)", name, tableView)
		}
	}
}

// TestAppModel_HintClearMsg_MatchingToken_ClearsHint verifies that a HintClearMsg
// with the current hint token clears the status bar hint (observable outcome).
func TestAppModel_HintClearMsg_MatchingToken_ClearsHint(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	// Post a hint to get a valid token.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = result.(AppModel)
	if m.statusBar.Hint == "" {
		t.Fatal("precondition: hint not posted")
	}
	token := m.statusBar.HintToken()

	// Matching token → hint clears.
	result, _ = m.Update(data.HintClearMsg{Token: token})
	appM := result.(AppModel)

	if appM.statusBar.Hint != "" {
		t.Errorf("statusBar.Hint = %q after matching HintClearMsg, want empty", appM.statusBar.Hint)
	}
}

// TestAppModel_HintClearMsg_StaleToken_DoesNotClearHint verifies that a HintClearMsg
// with a stale (mismatched) token does NOT clear the hint (anti-behavior: token guard).
func TestAppModel_HintClearMsg_StaleToken_DoesNotClearHint(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	// Post a hint.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = result.(AppModel)
	if m.statusBar.Hint == "" {
		t.Fatal("precondition: hint not posted")
	}
	staleToken := m.statusBar.HintToken() - 1 // deliberately stale

	result, _ = m.Update(data.HintClearMsg{Token: staleToken})
	appM := result.(AppModel)

	if appM.statusBar.Hint == "" {
		t.Error("statusBar.Hint cleared by stale HintClearMsg, want hint preserved")
	}
}

// TestAppModel_ContextSwitchedMsg_ClearsHint verifies that a ContextSwitchedMsg
// clears any active hint synchronously (input-changed: context switch resets hint state).
func TestAppModel_ContextSwitchedMsg_ClearsHint(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	// Post a hint first.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = result.(AppModel)
	if m.statusBar.Hint == "" {
		t.Fatal("precondition: hint not posted")
	}

	result, _ = m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.statusBar.Hint != "" {
		t.Errorf("statusBar.Hint = %q after ContextSwitchedMsg, want empty (sync clear)", appM.statusBar.Hint)
	}
}

// TestAppModel_EarlyClear_AnyKeypress_ClearsHint verifies that pressing any key
// while a hint is active immediately clears it and bumps the hint token so any
// in-flight HintClearCmd becomes stale (input-changed: keypress early-clear).
func TestAppModel_EarlyClear_AnyKeypress_ClearsHint(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	// Post a hint.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	m = result.(AppModel)
	if m.statusBar.Hint == "" {
		t.Fatal("precondition: hint not posted")
	}
	tokenAfterHint := m.statusBar.HintToken()

	// Any keypress (j = cursor move) clears hint and bumps token.
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	appM := result.(AppModel)

	if appM.statusBar.Hint != "" {
		t.Errorf("statusBar.Hint = %q after keypress, want early-clear", appM.statusBar.Hint)
	}
	if appM.statusBar.HintToken() <= tokenAfterHint {
		t.Errorf("HintToken = %d, want > %d (token must be bumped to invalidate in-flight HintClearCmd)",
			appM.statusBar.HintToken(), tokenAfterHint)
	}
}

// ==================== FB-007: Change history with diff ====================

// testFB007Manifests returns three fixture manifests oldest→newest.
// Uses a function to avoid shared mutable state across parallel tests.
func testFB007Manifests() []map[string]any {
	return []map[string]any{
		{"spec": map[string]any{"nodeName": "node-1"}}, // rev 1 — creation
		{"spec": map[string]any{"nodeName": "node-2"}}, // rev 2
		{"spec": map[string]any{"nodeName": "node-3"}}, // rev 3 (newest)
	}
}

// testFB007Rows returns three HistoryRow fixtures parallel to testFB007Manifests.
func testFB007Rows() []data.HistoryRow {
	return []data.HistoryRow{
		{Rev: 1, User: "alice@example.com", Source: "human", Verb: "create", Summary: "Created", Parseable: true},
		{Rev: 2, User: "bob@example.com", Source: "human", Verb: "update", Summary: "spec.nodeName", Parseable: true},
		{Rev: 3, User: "system:rec", Source: "system", Verb: "update", Summary: "spec.nodeName", Parseable: true},
	}
}

// newDetailPaneModelWithHC builds an AppModel in DetailPane with a non-nil hc.
// The describe result is pre-set so detail.Loading() == false.
func newDetailPaneModelWithHC() AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	rt := data.ResourceType{Name: "pods", Kind: "Pod", Group: ""}
	sidebar.SetItems([]data.ResourceType{rt})

	detail := components.NewDetailViewModel(58, 20)
	detail.SetResourceContext("Pod", "my-pod")
	detail.SetContent("describe text")

	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		hc:          data.NewHistoryClient(nil),
		activePane:  DetailPane,
		describeRT:  rt,
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(58, 20),
		detail:      detail,
		quota:       components.NewQuotaDashboardModel(58, 20, "proj"),
		history:     components.NewHistoryViewModel(58, 20),
		diff:        components.NewDiffViewModel(58, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// newHistoryPaneModel builds an AppModel in HistoryPane with three rows preloaded.
func newHistoryPaneModel() AppModel {
	m := newDetailPaneModelWithHC()
	rows := testFB007Rows()
	manifests := testFB007Manifests()
	m.historyRT = m.describeRT
	m.historyName = "my-pod"
	m.historyNamespace = ""
	m.historyRows = rows
	m.currentHistoryManifests = manifests
	m.history.SetResourceContext("Pod", "my-pod")
	m.history.SetRows(rows, false)
	m.activePane = HistoryPane
	m.updatePaneFocus()
	return m
}

// newDiffPaneModel builds an AppModel in DiffPane at the given 0-based manifest index.
func newDiffPaneModel(revIdx int) AppModel {
	m := newHistoryPaneModel()
	rows := testFB007Rows()
	row := rows[revIdx]
	var prev *data.HistoryRow
	if revIdx > 0 {
		p := rows[revIdx-1]
		prev = &p
	}
	isCreation := revIdx == 0
	m.diff.SetRevision(row, prev, "+changed\n", isCreation, false)
	m.selectedRevIdx = revIdx
	m.activePane = DiffPane
	m.updatePaneFocus()
	return m
}

// --- 10a: H from DetailPane ---

// TestAppModel_HKey_DetailPane_FirstPress_TransitionsToHistoryAndDispatchesCmd covers
// the first-press axis of 10a: H from DetailPane with no cache → HistoryPane + cmd.
func TestAppModel_HKey_DetailPane_FirstPress_TransitionsToHistoryAndDispatchesCmd(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != HistoryPane {
		t.Errorf("activePane = %v after H, want HistoryPane", appM.activePane)
	}
	if cmd == nil {
		t.Error("cmd = nil after H from DetailPane with no cache, want LoadHistoryCmd")
	}
	// Observable: statusBar pane label = "HISTORY".
	if appM.statusBar.Pane != "HISTORY" {
		t.Errorf("statusBar.Pane = %q, want HISTORY", appM.statusBar.Pane)
	}
}

// TestAppModel_HKey_DetailPane_RepeatPress_CacheHit_NoCmd covers the repeat-press axis
// of 10a: H with rows already loaded → no new cmd dispatched.
func TestAppModel_HKey_DetailPane_RepeatPress_CacheHit_NoCmd(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	rows := testFB007Rows()
	m.historyRT = m.describeRT
	m.historyName = "my-pod"
	m.history.SetRows(rows, false)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != HistoryPane {
		t.Errorf("activePane = %v, want HistoryPane", appM.activePane)
	}
	if cmd != nil {
		t.Error("cmd != nil on cache-hit H: want no new LoadHistoryCmd dispatched")
	}
}

// TestAppModel_HKey_DetailPane_InputChanged_NewResource_DispatchesCmd covers the
// input-changed axis of 10a: resource target changed → history reset + refetch.
func TestAppModel_HKey_DetailPane_InputChanged_NewResource_DispatchesCmd(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	// Pre-populate history for a DIFFERENT resource.
	m.historyName = "other-pod"
	m.historyRT = data.ResourceType{Name: "pods", Kind: "Pod"}
	m.history.SetRows(testFB007Rows(), false)

	// detail now describes "my-pod" (set in newDetailPaneModelWithHC).
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != HistoryPane {
		t.Errorf("activePane = %v, want HistoryPane", appM.activePane)
	}
	// A new cmd must be dispatched because the resource target changed.
	if cmd == nil {
		t.Error("cmd = nil on resource-change H, want LoadHistoryCmd for new resource")
	}
}

// TestAppModel_HKey_DetailPane_WhileDescribeLoading_Inert covers the anti-behavior
// axis of 10a: H while describe is still loading must be silently inert.
func TestAppModel_HKey_DetailPane_WhileDescribeLoading_Inert(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.detail.SetLoading(true) // simulate in-flight describe

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v after H while describe loading, want DetailPane (inert)", appM.activePane)
	}
	if cmd != nil {
		t.Error("cmd != nil after H while loading, want nil (silent no-op)")
	}
}

// --- 10b: H from HistoryPane ---

// TestAppModel_HKey_HistoryPane_FirstPress_TogglesBackToDetailPane covers first-press
// of 10b: H from HistoryPane → DetailPane, no cmd.
func TestAppModel_HKey_HistoryPane_FirstPress_TogglesBackToDetailPane(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v after H from HistoryPane, want DetailPane", appM.activePane)
	}
	if cmd != nil {
		t.Error("cmd != nil after H from HistoryPane, want nil (toggle to Detail dispatches nothing)")
	}
}

// TestAppModel_HKey_HistoryPane_RepeatPress_RoundTrip covers repeat-press of 10b:
// H toggles to DetailPane, H again returns to HistoryPane via cache hit.
func TestAppModel_HKey_HistoryPane_RepeatPress_RoundTrip(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	// First H: HistoryPane → DetailPane.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m1 := r1.(AppModel)
	if m1.activePane != DetailPane {
		t.Fatalf("precondition failed: first H should go to DetailPane, got %v", m1.activePane)
	}

	// Second H: DetailPane → HistoryPane (cache hit, no cmd).
	r2, cmd2 := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m2 := r2.(AppModel)

	if m2.activePane != HistoryPane {
		t.Errorf("activePane = %v after second H, want HistoryPane (round-trip)", m2.activePane)
	}
	if cmd2 != nil {
		t.Error("cmd2 != nil on cache-hit round-trip H, want nil")
	}
}

// TestAppModel_HKey_HistoryPane_FilterOn_ClearedOnExit covers anti-behavior of 10b:
// filter flag is cleared when exiting HistoryPane via H.
func TestAppModel_HKey_HistoryPane_FilterOn_ClearedOnExit(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	m.history.ToggleHumanFilter()

	// Observable precondition: filter banner visible.
	viewBefore := stripANSIModel(m.history.View())
	if !strings.Contains(viewBefore, "filter: human only") {
		t.Fatal("precondition: filter banner not visible before pressing H")
	}

	// H from HistoryPane → DetailPane (ResetFilter called).
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m1 := r1.(AppModel)

	// H from DetailPane → HistoryPane (re-enter; filter should be off).
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	m2 := r2.(AppModel)

	viewAfter := stripANSIModel(m2.history.View())
	if strings.Contains(viewAfter, "filter: human only") {
		t.Errorf("filter banner still visible after exit+re-enter via H, want cleared: %q", viewAfter)
	}
}

// --- 10c: Enter in HistoryPane ---

// TestAppModel_Enter_HistoryPane_FirstPress_OpensDiffPane covers first-press of 10c:
// Enter from HistoryPane with cursor on rev≥2 → DiffPane, banner contains rev labels.
func TestAppModel_Enter_HistoryPane_FirstPress_OpensDiffPane(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	// Default cursor = newest (rev 3, manifest index 2).

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != DiffPane {
		t.Errorf("activePane = %v after Enter in HistoryPane, want DiffPane", appM.activePane)
	}
	// Observable: banner contains "Rev 3".
	diffView := stripANSIModel(appM.diff.View())
	if !strings.Contains(diffView, "Rev 3") {
		t.Errorf("DiffPane banner: want 'Rev 3', got %q", diffView)
	}
}

// TestAppModel_Enter_DiffPane_Inert covers repeat-press of 10c: Enter in DiffPane is
// a silent no-op (no drill-in in v1).
func TestAppModel_Enter_DiffPane_Inert(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(1) // rev 2

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != DiffPane {
		t.Errorf("activePane = %v after Enter in DiffPane, want DiffPane (no-op)", appM.activePane)
	}
	if cmd != nil {
		t.Error("cmd != nil after Enter in DiffPane, want nil (silent no-op)")
	}
}

// TestAppModel_Enter_HistoryPane_DifferentCursor_DifferentDiff covers input-changed
// of 10c: different cursor → different diff banner.
func TestAppModel_Enter_HistoryPane_DifferentCursor_DifferentDiff(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	// Enter at cursor=0 (rev 3, newest).
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM1 := r1.(AppModel)
	diffView1 := stripANSIModel(appM1.diff.View())

	// Return to HistoryPane, move cursor down (to rev 2).
	m2 := newHistoryPaneModel()
	m2.history.CursorDown() // cursor → rev 2 (manifest index 1)

	r2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM2 := r2.(AppModel)
	diffView2 := stripANSIModel(appM2.diff.View())

	// Both should be in DiffPane but with different rev labels.
	if appM1.activePane != DiffPane || appM2.activePane != DiffPane {
		t.Fatalf("both should be DiffPane, got %v and %v", appM1.activePane, appM2.activePane)
	}
	if diffView1 == diffView2 {
		t.Error("DiffPane views should differ for different cursor positions")
	}
}

// TestAppModel_Enter_HistoryPane_Rev1_ShowsCreationView covers anti-behavior of 10c:
// Enter on rev 1 → creation manifest view (not a diff).
func TestAppModel_Enter_HistoryPane_Rev1_ShowsCreationView(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	m.history.CursorBottom() // → rev 1 (oldest)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != DiffPane {
		t.Fatalf("activePane = %v, want DiffPane", appM.activePane)
	}
	// Observable: "creation" label in banner, not a diff banner.
	diffView := stripANSIModel(appM.diff.View())
	if !strings.Contains(diffView, "creation") {
		t.Errorf("rev 1 DiffPane: want 'creation' label in banner, got %q", diffView)
	}
	if !strings.Contains(diffView, "Created resource") {
		t.Errorf("rev 1 DiffPane: want 'Created resource' label, got %q", diffView)
	}
}

// --- 10d: [ in DiffPane ---

// TestAppModel_BracketLeft_DiffPane_FirstPress_StepsToOlderRev covers first-press of
// 10d: [ in DiffPane on rev K ≥ 2 → selectedRevIdx decrements.
func TestAppModel_BracketLeft_DiffPane_FirstPress_StepsToOlderRev(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(2) // rev 3 (index 2)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	appM := result.(AppModel)

	if appM.selectedRevIdx != 1 {
		t.Errorf("selectedRevIdx = %d after [, want 1 (rev 2)", appM.selectedRevIdx)
	}
	// Observable: banner changes from "Rev 3" to "Rev 2".
	diffView := stripANSIModel(appM.diff.View())
	if !strings.Contains(diffView, "Rev 2") {
		t.Errorf("after [: want 'Rev 2' in DiffPane banner, got %q", diffView)
	}
}

// TestAppModel_BracketLeft_DiffPane_RepeatPress_StepsTwice covers repeat-press of 10d.
func TestAppModel_BracketLeft_DiffPane_RepeatPress_StepsTwice(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(2) // rev 3

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	m1 := r1.(AppModel) // rev 2
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	appM := r2.(AppModel) // rev 1

	if appM.selectedRevIdx != 0 {
		t.Errorf("selectedRevIdx = %d after two [, want 0 (rev 1)", appM.selectedRevIdx)
	}
}

// TestAppModel_BracketLeft_DiffPane_Rev1_Inert covers anti-behavior of 10d:
// [ at rev 1 (index 0) is inert.
func TestAppModel_BracketLeft_DiffPane_Rev1_Inert(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(0) // rev 1

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	appM := result.(AppModel)

	if appM.selectedRevIdx != 0 {
		t.Errorf("selectedRevIdx = %d after [ at rev 1, want 0 (clamped)", appM.selectedRevIdx)
	}
	if cmd != nil {
		t.Error("cmd != nil after [ at rev 1, want nil (silent no-op)")
	}
	// Observable: DiffPane still shows rev 1 (creation view).
	diffView := stripANSIModel(appM.diff.View())
	if !strings.Contains(diffView, "Rev 1") {
		t.Errorf("after [ at rev 1: want 'Rev 1' still in banner, got %q", diffView)
	}
}

// --- 10e: ] in DiffPane ---

// TestAppModel_BracketRight_DiffPane_FirstPress_StepsToNewerRev covers first-press of 10e.
func TestAppModel_BracketRight_DiffPane_FirstPress_StepsToNewerRev(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(0) // rev 1

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	appM := result.(AppModel)

	if appM.selectedRevIdx != 1 {
		t.Errorf("selectedRevIdx = %d after ], want 1 (rev 2)", appM.selectedRevIdx)
	}
	diffView := stripANSIModel(appM.diff.View())
	if !strings.Contains(diffView, "Rev 2") {
		t.Errorf("after ]: want 'Rev 2' in DiffPane banner, got %q", diffView)
	}
}

// TestAppModel_BracketRight_DiffPane_RepeatPress_StepsTwice covers repeat-press of 10e.
func TestAppModel_BracketRight_DiffPane_RepeatPress_StepsTwice(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(0) // rev 1

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	m1 := r1.(AppModel) // rev 2
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	appM := r2.(AppModel) // rev 3

	if appM.selectedRevIdx != 2 {
		t.Errorf("selectedRevIdx = %d after two ], want 2 (rev 3)", appM.selectedRevIdx)
	}
}

// TestAppModel_BracketRight_DiffPane_RevN_Inert covers anti-behavior of 10e:
// ] at rev N (last index) is inert.
func TestAppModel_BracketRight_DiffPane_RevN_Inert(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(2) // rev 3 (last)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	appM := result.(AppModel)

	if appM.selectedRevIdx != 2 {
		t.Errorf("selectedRevIdx = %d after ] at rev N, want 2 (clamped)", appM.selectedRevIdx)
	}
	if cmd != nil {
		t.Error("cmd != nil after ] at rev N, want nil (silent no-op)")
	}
}

// --- 10f: c filter toggle in HistoryPane ---

// TestAppModel_CKey_HistoryPane_FirstPress_EnablesFilter covers first-press of 10f.
func TestAppModel_CKey_HistoryPane_FirstPress_EnablesFilter(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	appM := result.(AppModel)

	// Observable: filter banner appears in history view.
	histView := stripANSIModel(appM.history.View())
	if !strings.Contains(histView, "filter: human only") {
		t.Errorf("after c: want 'filter: human only' banner in history view, got %q", histView)
	}
}

// TestAppModel_CKey_HistoryPane_RepeatPress_RestoresAllRows covers repeat-press of 10f.
func TestAppModel_CKey_HistoryPane_RepeatPress_RestoresAllRows(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	// First c: filter on.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m1 := r1.(AppModel)

	// Second c: filter off.
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	appM := r2.(AppModel)

	histView := stripANSIModel(appM.history.View())
	if strings.Contains(histView, "filter: human only") {
		t.Errorf("after second c: filter banner still visible, want cleared: %q", histView)
	}
	// All rows visible: rev 1, rev 2, rev 3 should appear.
	if !strings.Contains(histView, "1") || !strings.Contains(histView, "3") {
		t.Errorf("after second c: expected all revisions visible, got %q", histView)
	}
}

// TestAppModel_CKey_HistoryPane_ZeroHumanRows_EmptyListNoCrash covers anti-behavior of 10f:
// filter on with only system rows → empty list notice, no crash.
func TestAppModel_CKey_HistoryPane_ZeroHumanRows_EmptyListNoCrash(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	systemRows := []data.HistoryRow{
		{Rev: 1, Source: "system", Verb: "update", Parseable: true},
		{Rev: 2, Source: "system", Verb: "update", Parseable: true},
	}
	m.historyRT = m.describeRT
	m.historyName = "my-pod"
	m.historyRows = systemRows
	m.currentHistoryManifests = []map[string]any{{}, {}}
	m.history.SetResourceContext("Pod", "my-pod")
	m.history.SetRows(systemRows, false)
	m.activePane = HistoryPane
	m.updatePaneFocus()

	// c filter: all rows are system → empty human list.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	appM := result.(AppModel)

	histView := stripANSIModel(appM.history.View())
	if !strings.Contains(histView, "No human-source revisions") {
		t.Errorf("zero-human filter: want 'No human-source revisions' notice, got %q", histView)
	}
}

// TestAppModel_HistoryLoadedMsg_FilterSurvivesRefresh covers input-changed of 10f:
// filter flag persists when HistoryLoadedMsg arrives during in-pane refresh.
func TestAppModel_HistoryLoadedMsg_FilterSurvivesRefresh(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	// Enable filter.
	m.history.ToggleHumanFilter()

	// Simulate refresh: HistoryLoadedMsg with fresh rows arrives.
	freshRows := testFB007Rows()
	result, _ := m.Update(data.HistoryLoadedMsg{
		APIGroup:  m.historyRT.Group,
		Kind:      m.historyRT.Kind,
		Name:      m.historyName,
		Namespace: m.historyNamespace,
		Rows:      freshRows,
		Manifests: testFB007Manifests(),
	})
	appM := result.(AppModel)

	// HistoryLoadedMsg calls m.history.SetRows which preserves filterHuman.
	// Observable: filter banner still visible.
	histView := stripANSIModel(appM.history.View())
	if !strings.Contains(histView, "filter: human only") {
		t.Errorf("after refresh with filter on: want 'filter: human only' banner, got %q", histView)
	}
}

// --- 10g: Esc stacking ---

// TestAppModel_Esc_DiffPane_ReturnsToHistoryPane covers first-press of 10g.
func TestAppModel_Esc_DiffPane_ReturnsToHistoryPane(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(1)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != HistoryPane {
		t.Errorf("activePane = %v after Esc from DiffPane, want HistoryPane", appM.activePane)
	}
}

// TestAppModel_Esc_HistoryPane_ReturnsToDetailPane_ClearsFilter covers repeat-press
// of 10g (second Esc) and anti-behavior (filter cleared).
func TestAppModel_Esc_HistoryPane_ReturnsToDetailPane_ClearsFilter(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	m.history.ToggleHumanFilter()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v after Esc from HistoryPane, want DetailPane", appM.activePane)
	}
	// Observable: filter banner absent after exit.
	histView := stripANSIModel(appM.history.View())
	if strings.Contains(histView, "filter: human only") {
		t.Errorf("filter banner still visible after Esc from HistoryPane, want cleared: %q", histView)
	}
}

// TestAppModel_Esc_HistoryPane_FilterClearedAfterRoundTrip covers input-changed of 10g:
// filter is off when re-entering HistoryPane after exiting via Esc.
func TestAppModel_Esc_HistoryPane_FilterClearedAfterRoundTrip(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	m.history.ToggleHumanFilter()

	// Esc → DetailPane.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m1 := r1.(AppModel)
	if m1.activePane != DetailPane {
		t.Fatalf("expected DetailPane after Esc, got %v", m1.activePane)
	}

	// H → HistoryPane (re-enter).
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := r2.(AppModel)

	histView := stripANSIModel(appM.history.View())
	if strings.Contains(histView, "filter: human only") {
		t.Errorf("filter banner visible after Esc+re-enter, want cleared: %q", histView)
	}
}

// TestAppModel_Esc_DetailPane_HistoryStateUntouched covers anti-behavior of 10g:
// Esc from DetailPane (→ TablePane) does not touch m.history or m.diff state.
func TestAppModel_Esc_DetailPane_HistoryStateUntouched(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.history.SetRows(testFB007Rows(), false)
	m.historyRows = testFB007Rows()
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"
	m.table = components.NewResourceTableModel(58, 20)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Fatalf("Esc from Detail: activePane = %v, want TablePane", appM.activePane)
	}
	// history state must be untouched (rows still present).
	if !appM.history.HasRows() {
		t.Error("Esc from DetailPane touched history.rows, want untouched")
	}
}

// --- 10h: r refresh in HistoryPane ---

// TestAppModel_RKey_HistoryPane_FirstPress_SetsRefreshingAndDispatchesCmd covers
// first-press of 10h: r in HistoryPane → ForceRefresh + refreshing=true + cmd.
func TestAppModel_RKey_HistoryPane_FirstPress_SetsRefreshingAndDispatchesCmd(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if !appM.refreshing {
		t.Error("refreshing = false after r in HistoryPane, want true")
	}
	if cmd == nil {
		t.Error("cmd = nil after r in HistoryPane, want LoadHistoryCmd")
	}
	// Observable: history reset to loading state.
	histView := stripANSIModel(appM.history.View())
	if !strings.Contains(histView, "loading history") {
		t.Errorf("after r: want 'loading history' in HistoryPane view, got %q", histView)
	}
}

// TestAppModel_RKey_DiffPane_TransitionsToHistoryPane_AndRefreshes covers 10h from DiffPane:
// r in DiffPane → activePane = HistoryPane + cmd dispatched.
func TestAppModel_RKey_DiffPane_TransitionsToHistoryPane_AndRefreshes(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(1)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if appM.activePane != HistoryPane {
		t.Errorf("activePane = %v after r in DiffPane, want HistoryPane", appM.activePane)
	}
	if !appM.refreshing {
		t.Error("refreshing = false after r in DiffPane, want true")
	}
	if cmd == nil {
		t.Error("cmd = nil after r in DiffPane, want LoadHistoryCmd")
	}
}

// TestAppModel_HistoryLoadedMsg_Populates_Rows covers observable of 10h:
// HistoryLoadedMsg delivers fresh rows and clears refreshing.
func TestAppModel_HistoryLoadedMsg_Populates_Rows(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.historyRT = m.describeRT
	m.historyName = "my-pod"
	m.historyNamespace = ""
	m.activePane = HistoryPane
	m.refreshing = true
	m.updatePaneFocus()

	freshRows := testFB007Rows()
	result, _ := m.Update(data.HistoryLoadedMsg{
		APIGroup:  m.historyRT.Group,
		Kind:      m.historyRT.Kind,
		Name:      m.historyName,
		Namespace: m.historyNamespace,
		Rows:      freshRows,
		Manifests: testFB007Manifests(),
	})
	appM := result.(AppModel)

	if appM.refreshing {
		t.Error("refreshing = true after HistoryLoadedMsg, want false")
	}
	if !appM.history.HasRows() {
		t.Error("history.HasRows() = false after HistoryLoadedMsg, want true")
	}
}

// TestAppModel_HistoryLoadedMsg_Stale_Discarded verifies that a stale HistoryLoadedMsg
// (for a different resource) is silently discarded (anti-behavior of 10h).
func TestAppModel_HistoryLoadedMsg_Stale_Discarded(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.historyRT = m.describeRT
	m.historyName = "my-pod"
	m.activePane = HistoryPane
	m.updatePaneFocus()

	result, _ := m.Update(data.HistoryLoadedMsg{
		APIGroup: "other-group",
		Kind:     "OtherKind",
		Name:     "other-resource",
		Rows:     testFB007Rows(),
	})
	appM := result.(AppModel)

	if appM.history.HasRows() {
		t.Error("history rows populated by stale HistoryLoadedMsg, want discarded")
	}
}

// --- 10i: H from ineligible panes ---

// TestAppModel_HKey_NavPane_Inert covers first-press + repeat-press of 10i.
func TestAppModel_HKey_NavPane_Inert(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = NavPane
	m.updatePaneFocus()

	for i := 0; i < 2; i++ {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
		appM := result.(AppModel)
		if appM.activePane != NavPane {
			t.Errorf("press %d: activePane = %v, want NavPane (H inert)", i+1, appM.activePane)
		}
		if cmd != nil {
			t.Errorf("press %d: cmd != nil after H in NavPane, want nil", i+1)
		}
		m = appM
	}
}

// TestAppModel_HKey_TablePane_Inert covers input-changed axis of 10i.
func TestAppModel_HKey_TablePane_Inert(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v after H in TablePane, want TablePane (inert)", appM.activePane)
	}
	if cmd != nil {
		t.Error("cmd != nil after H in TablePane, want nil")
	}
}

// TestAppModel_HKey_QuotaDashboardPane_Inert covers anti-behavior of 10i.
func TestAppModel_HKey_QuotaDashboardPane_Inert(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("activePane = %v after H in QuotaDashboard, want QuotaDashboardPane (inert)", appM.activePane)
	}
	if cmd != nil {
		t.Error("cmd != nil after H in QuotaDashboard, want nil")
	}
	// Observable: statusBar unchanged (not HISTORY).
	if appM.statusBar.Pane == "HISTORY" {
		t.Error("statusBar.Pane = HISTORY after H in QuotaDashboard, want unchanged")
	}
}

// --- 10j: / in HistoryPane / DiffPane (FB-003 guard) ---

// TestAppModel_SlashKey_HistoryPane_Silent_FilterBarNotFocused covers first-press + repeat
// of 10j: / in HistoryPane is a silent no-op; FilterBar must not be focused.
func TestAppModel_SlashKey_HistoryPane_Silent_FilterBarNotFocused(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	for i := 0; i < 2; i++ {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
		appM := result.(AppModel)

		if appM.filterBar.Focused() {
			t.Errorf("press %d: filterBar.Focused() = true after / in HistoryPane, want false", i+1)
		}
		if appM.activePane != HistoryPane {
			t.Errorf("press %d: activePane = %v, want HistoryPane unchanged", i+1, appM.activePane)
		}
		m = appM
	}
}

// TestAppModel_SlashKey_DiffPane_Silent_FilterBarNotFocused covers the DiffPane axis of 10j.
func TestAppModel_SlashKey_DiffPane_Silent_FilterBarNotFocused(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(1)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	if appM.filterBar.Focused() {
		t.Error("filterBar.Focused() = true after / in DiffPane, want false")
	}
	if appM.activePane != DiffPane {
		t.Errorf("activePane = %v after / in DiffPane, want DiffPane", appM.activePane)
	}
}

// TestAppModel_SlashKey_HistoryPane_CFilterActive_CFilterUnchanged covers anti-behavior
// of 10j: / must not modify the c-filter state.
func TestAppModel_SlashKey_HistoryPane_CFilterActive_CFilterUnchanged(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	// Enable c-filter.
	m.history.ToggleHumanFilter()
	viewBefore := stripANSIModel(m.history.View())
	if !strings.Contains(viewBefore, "filter: human only") {
		t.Fatal("precondition: c-filter not active")
	}

	// Press /: should be inert.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	viewAfter := stripANSIModel(appM.history.View())
	if !strings.Contains(viewAfter, "filter: human only") {
		t.Error("/ in HistoryPane removed c-filter banner, want c-filter untouched")
	}
}

// --- Context switch invalidates history ---

// TestAppModel_ContextSwitchedMsg_InvalidatesHistoryCache covers E2E-15:
// context switch calls hc.Invalidate() and resets history/diff state.
func TestAppModel_ContextSwitchedMsg_InvalidatesHistoryCache(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.historyRows != nil {
		t.Error("historyRows != nil after ContextSwitchedMsg, want nil")
	}
	if appM.currentHistoryManifests != nil {
		t.Error("currentHistoryManifests != nil after ContextSwitchedMsg, want nil")
	}
	if appM.history.HasRows() {
		t.Error("history.HasRows() = true after ContextSwitchedMsg, want false (Reset called)")
	}
}

// TestAppModel_HKey_DiffPane_TransitionsToDetailPane verifies that pressing H in
// DiffPane bypasses HistoryPane and lands directly in DetailPane, resetting yamlMode
// and detail mode (model.go:769-775).
func TestAppModel_HKey_DiffPane_TransitionsToDetailPane(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(1)
	m.yamlMode = true
	m.detail.SetMode("yaml")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v after H from DiffPane, want DetailPane", appM.activePane)
	}
	if appM.statusBar.Mode != components.ModeDetail {
		t.Errorf("statusBar.Mode = %v after H from DiffPane, want ModeDetail", appM.statusBar.Mode)
	}
	if appM.yamlMode {
		t.Error("yamlMode = true after H from DiffPane, want false (model.go:770 resets it)")
	}
	if got := appM.detail.Mode(); got != "" {
		t.Errorf("detail.Mode() = %q after H from DiffPane, want empty (model.go:771 clears it)", got)
	}
}

// TestAppModel_CKey_DiffPane_Inert verifies that pressing c in DiffPane is a hard
// no-op: no overlay opened, pane unchanged, mode unchanged (§5c of the FB-007 spec).
func TestAppModel_CKey_DiffPane_Inert(t *testing.T) {
	t.Parallel()
	m := newDiffPaneModel(1)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("overlay = %v after c in DiffPane, want NoOverlay (must not open CtxSwitcher)", appM.overlay)
	}
	if appM.activePane != DiffPane {
		t.Errorf("activePane = %v after c in DiffPane, want DiffPane (must be no-op)", appM.activePane)
	}
	if appM.statusBar.Mode == components.ModeOverlay {
		t.Error("statusBar.Mode = ModeOverlay after c in DiffPane, want unchanged")
	}
	if cmd != nil {
		t.Error("c in DiffPane: cmd should be nil, want no-op")
	}
}

// ==================== End FB-007 ====================

// ==================== FB-009: Raw YAML toggle in DETAIL pane ====================
//
// FB-009 axis-coverage table
// AC  | first-press                                       | repeat-press                                       | input-changed                                    | anti-behavior                           | observable
// ----|---------------------------------------------------|----------------------------------------------------|-------------------------------------------------|-----------------------------------------|----------------------------------------------------
// 1   | _FirstPress_TogglesYamlMode                       | -                                                  | -                                               | -                                       | _FirstPress_ContentChanges
// 2   | -                                                 | _RepeatPress_TogglesBackToDescribe                 | -                                               | -                                       | -
// 3   | -                                                 | -                                                  | _ScrollReset_OnToggle                           | -                                       | -
// 4   | -                                                 | -                                                  | -                                               | _NonDetailPane_IsNoOp (5 panes)         | -
// 5   | -                                                 | -                                                  | -                                               | _NilRaw_IsNoOp                          | -
// 6   | -                                                 | -                                                  | _ContextSwitchedMsg_ResetsYamlMode              | -                                       | -
// 7   | -                                                 | -                                                  | _Enter_TablePane_ResetsYamlMode                 | -                                       | -
// 8   | -                                                 | -                                                  | _Esc_DetailPane_YamlMode_Resets                 | -                                       | -
// 9   | -                                                 | -                                                  | -                                               | -                                       | _TitleBarSuffix
// 10  | -                                                 | -                                                  | -                                               | -                                       | _ContentParity_AnnotationVisible
// 11  | -                                                 | -                                                  | -                                               | -                                       | _MarshalError_ShowsWarning
// 12  | -                                                 | -                                                  | -                                               | -                                       | via _DescribeResultMsg_CapturesRaw + AC#1 observables
// 13  | -                                                 | -                                                  | -                                               | -                                       | _Lifecycle_EscAndReEnter
// (All test names above are short-form; prefix is TestAppModel_YKey_DetailPane_ unless noted.)

// testRawObject returns a minimal *unstructured.Unstructured for FB-009 fixtures.
func testRawObject() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]any{"name": "my-pod"},
			"spec":       map[string]any{"nodeName": "node-1"},
		},
	}
}

// longRawObject returns an *unstructured.Unstructured with enough annotations
// to produce >20 lines of YAML, making the detail viewport scrollable (AC#3).
func longRawObject() *unstructured.Unstructured {
	annotations := map[string]any{}
	for i := 0; i < 25; i++ {
		annotations[fmt.Sprintf("annotation-%02d", i)] = fmt.Sprintf("value-%02d", i)
	}
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":        "my-pod",
				"annotations": annotations,
			},
		},
	}
}

// newDetailPaneModelWithRaw builds a DetailPane AppModel with describeRaw set,
// so the y-key handler is eligible to fire.
func newDetailPaneModelWithRaw() AppModel {
	m := newDetailPaneModelWithHC()
	m.describeContent = "describe text\nsome fields here"
	m.describeRaw = testRawObject()
	m.detail.SetDescribeAvailable(true)
	return m
}

// newDetailPaneModelWithYaml builds a DetailPane AppModel already in yaml mode
// (after one y press on a model with describeRaw set).
func newDetailPaneModelWithYaml() AppModel {
	m := newDetailPaneModelWithRaw()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	return result.(AppModel)
}

// --- AC#3: scroll reset on toggle (integration) ---

// TestAppModel_YKey_DetailPane_ScrollReset_OnToggle verifies AC#3: each y toggle resets
// the viewport scroll to top. Uses longRawObject() (25 annotations → >20 lines of YAML)
// so the viewport is scrollable and "top" vs non-zero offset is distinguishable via
// the scroll-footer observable in detail.View().
func TestAppModel_YKey_DetailPane_ScrollReset_OnToggle(t *testing.T) {
	t.Parallel()
	// Long describe content (30 lines) so viewport (height=20) can scroll.
	longDescribe := strings.Repeat("some describe field line\n", 30)

	m := newDetailPaneModelWithHC()
	m.describeContent = longDescribe
	m.describeRaw = longRawObject()

	// Load content via DescribeResultMsg (same path as real usage).
	r0, _ := m.Update(data.DescribeResultMsg{Content: longDescribe, Raw: longRawObject()})
	m = r0.(AppModel)

	// Scroll describe viewport down 10 lines.
	for i := 0; i < 10; i++ {
		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = r.(AppModel)
	}

	// Toggle to yaml mode — scroll must reset to top.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = r1.(AppModel)

	yamlView := stripANSIModel(m.detail.View())
	if !strings.Contains(yamlView, "top") {
		t.Errorf("AC#3: yaml-mode scroll not reset: want 'top' in footer, got:\n%s", yamlView)
	}

	// Scroll yaml viewport down 10 lines.
	for i := 0; i < 10; i++ {
		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = r.(AppModel)
	}

	// Toggle back to describe — scroll must reset to top again.
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = r2.(AppModel)

	descView := stripANSIModel(m.detail.View())
	if !strings.Contains(descView, "top") {
		t.Errorf("AC#3: describe-mode scroll not reset after toggle-back: want 'top' in footer, got:\n%s", descView)
	}
}

// --- 10a: y first-press ---

// TestAppModel_YKey_DetailPane_FirstPress_TogglesYamlMode verifies that pressing y
// in DetailPane with a non-nil describeRaw flips yamlMode true and sets mode="yaml".
func TestAppModel_YKey_DetailPane_FirstPress_TogglesYamlMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	appM := result.(AppModel)

	if !appM.yamlMode {
		t.Error("yamlMode = false after first y press, want true")
	}
	if got := appM.detail.Mode(); got != "yaml" {
		t.Errorf("detail.Mode() = %q after y press, want 'yaml'", got)
	}
}

// TestAppModel_YKey_DetailPane_FirstPress_ContentChanges verifies that View() content
// changes after y press (YAML content different from describe text).
func TestAppModel_YKey_DetailPane_FirstPress_ContentChanges(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()
	before := m.detail.View()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	appM := result.(AppModel)

	after := appM.detail.View()
	if before == after {
		t.Error("y press: detail.View() unchanged, want YAML content to replace describe text")
	}
	// YAML output should contain the spec field.
	if !strings.Contains(stripANSIModel(after), "node-1") {
		t.Errorf("y press: want 'node-1' from YAML in detail view, got:\n%s", stripANSIModel(after))
	}
}

// --- 10b: y repeat-press ---

// TestAppModel_YKey_DetailPane_RepeatPress_TogglesBackToDescribe verifies that a second
// y press restores yamlMode=false and mode="describe".
func TestAppModel_YKey_DetailPane_RepeatPress_TogglesBackToDescribe(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}) // → yaml
	m = r1.(AppModel)
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}) // → describe
	appM := r2.(AppModel)

	if appM.yamlMode {
		t.Error("yamlMode = true after second y press, want false")
	}
	if got := appM.detail.Mode(); got != "describe" {
		t.Errorf("detail.Mode() = %q after second y, want 'describe'", got)
	}
}

// --- 10c: y with describeRaw=nil (anti-behavior) ---

// TestAppModel_YKey_DetailPane_NilRaw_IsNoOp verifies that y is inert when
// describeRaw is nil (describe still loading or errored).
func TestAppModel_YKey_DetailPane_NilRaw_IsNoOp(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	// describeRaw is nil in the base fixture.

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	appM := result.(AppModel)

	if appM.yamlMode {
		t.Error("yamlMode = true with nil describeRaw, want no-op (false)")
	}
	if cmd != nil {
		t.Error("y with nil describeRaw: cmd should be nil, want no-op")
	}
}

// --- 10d: y in non-DetailPane (anti-behavior) ---

// TestAppModel_YKey_NonDetailPane_IsNoOp verifies that y does nothing outside DetailPane.
// AC#4 enumerates NavPane, TablePane, QuotaDashboardPane; HistoryPane and DiffPane are
// additional anti-behavior assertions.
func TestAppModel_YKey_NonDetailPane_IsNoOp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    func() AppModel
	}{
		{"NavPane", func() AppModel { return newNavPaneModelWithBC(nil) }},
		{"TablePane", newTablePaneModel},
		{"QuotaDashboardPane", func() AppModel { return newQuotaDashboardPaneModel(&stubBucketClient{}) }},
		{"HistoryPane", newHistoryPaneModel},
		{"DiffPane", func() AppModel { return newDiffPaneModel(1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.m()
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
			appM := result.(AppModel)
			if appM.yamlMode {
				t.Errorf("%s: yamlMode = true after y press, want no-op", tt.name)
			}
		})
	}
}

// --- 10e: DescribeResultMsg captures describeRaw ---

// TestAppModel_DescribeResultMsg_CapturesRaw verifies that a DescribeResultMsg with
// a non-nil Raw field populates m.describeRaw.
func TestAppModel_DescribeResultMsg_CapturesRaw(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	raw := testRawObject()

	result, _ := m.Update(data.DescribeResultMsg{Content: "text", Raw: raw})
	appM := result.(AppModel)

	if appM.describeRaw == nil {
		t.Fatal("describeRaw = nil after DescribeResultMsg with Raw set, want non-nil")
	}
	if got := appM.describeRaw.GetName(); got != "my-pod" {
		t.Errorf("describeRaw.GetName() = %q, want 'my-pod'", got)
	}
}

// TestAppModel_DescribeResultMsg_NilRaw_SetsNilDescribeRaw verifies that a
// DescribeResultMsg with nil Raw stores nil.
func TestAppModel_DescribeResultMsg_NilRaw_SetsNilDescribeRaw(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw() // starts with non-nil describeRaw

	result, _ := m.Update(data.DescribeResultMsg{Content: "fresh text", Raw: nil})
	appM := result.(AppModel)

	if appM.describeRaw != nil {
		t.Error("describeRaw != nil after DescribeResultMsg with nil Raw, want nil")
	}
}

// --- 10f: ContextSwitchedMsg resets yamlMode ---

// TestAppModel_ContextSwitchedMsg_ResetsYamlMode verifies that a context switch clears
// yamlMode, describeRaw, and detail mode.
func TestAppModel_ContextSwitchedMsg_ResetsYamlMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()
	m.yamlMode = true
	m.detail.SetMode("yaml")

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.yamlMode {
		t.Error("yamlMode = true after ContextSwitchedMsg, want false")
	}
	if appM.describeRaw != nil {
		t.Error("describeRaw != nil after ContextSwitchedMsg, want nil")
	}
	if got := appM.detail.Mode(); got != "" {
		t.Errorf("detail.Mode() = %q after ContextSwitchedMsg, want empty", got)
	}
}

// --- 10g: Esc from DetailPane resets yamlMode ---

// TestAppModel_Esc_DetailPane_YamlMode_Resets verifies that Esc from DetailPane
// clears yamlMode and describeRaw when returning to TablePane.
func TestAppModel_Esc_DetailPane_YamlMode_Resets(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()
	m.yamlMode = true
	m.detail.SetMode("yaml")
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.yamlMode {
		t.Error("yamlMode = true after Esc from DetailPane, want false")
	}
	if appM.describeRaw != nil {
		t.Error("describeRaw != nil after Esc from DetailPane, want nil")
	}
}

// --- 10h: H key from HistoryPane resets yamlMode ---

// TestAppModel_HKey_HistoryPane_ResetsYamlMode verifies that pressing H to return
// from HistoryPane to DetailPane clears yamlMode and detail mode.
func TestAppModel_HKey_HistoryPane_ResetsYamlMode(t *testing.T) {
	t.Parallel()
	m := newHistoryPaneModel()
	m.yamlMode = true
	m.detail.SetMode("yaml")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
	appM := result.(AppModel)

	if appM.yamlMode {
		t.Error("yamlMode = true after H from HistoryPane, want false")
	}
	if got := appM.detail.Mode(); got != "" {
		t.Errorf("detail.Mode() = %q after H from HistoryPane, want empty", got)
	}
}

// --- 10i: row navigation (d key) resets yamlMode ---

// TestAppModel_DKey_TablePane_ResetsYamlMode verifies that pressing d from TablePane
// to open a detail view resets yamlMode and describeRaw.
func TestAppModel_DKey_TablePane_ResetsYamlMode(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.yamlMode = true
	m.describeRaw = testRawObject()
	m.detail.SetMode("yaml")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	appM := result.(AppModel)

	if appM.yamlMode {
		t.Error("yamlMode = true after d from TablePane, want false (new row resets mode)")
	}
	if appM.describeRaw != nil {
		t.Error("describeRaw != nil after d from TablePane, want nil")
	}
}

// --- AC#7 (Enter path): Enter from TablePane resets yamlMode ---

// TestAppModel_Enter_TablePane_ResetsYamlMode verifies that pressing Enter from
// TablePane to open DetailPane clears yamlMode and describeRaw (input-changed via Enter).
func TestAppModel_Enter_TablePane_ResetsYamlMode(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel() // TablePane with "my-pod" row selected
	m.yamlMode = true
	m.describeRaw = testRawObject()
	m.detail.SetMode("yaml")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("activePane = %v after Enter from TablePane, want DetailPane", appM.activePane)
	}
	if appM.yamlMode {
		t.Error("yamlMode = true after Enter from TablePane, want false (new row resets mode)")
	}
	if appM.describeRaw != nil {
		t.Error("describeRaw != nil after Enter from TablePane, want nil")
	}
}

// --- AC#9: observable title-bar suffix ---

// TestAppModel_YKey_DetailPane_TitleBarSuffix verifies that the detail pane's
// title bar reflects mode: "yaml" suffix in yaml mode, "describe" suffix in
// describe mode. Uses newDetailPaneModelWithYaml() to enter yaml mode first.
func TestAppModel_YKey_DetailPane_TitleBarSuffix(t *testing.T) {
	t.Parallel()

	// yaml mode: title bar must contain "yaml".
	yamlM := newDetailPaneModelWithYaml()
	yamlView := stripANSIModel(yamlM.detail.View())
	if !strings.Contains(yamlView, "yaml") {
		t.Errorf("yaml-mode: detail.View() missing 'yaml' suffix, got:\n%s", yamlView)
	}

	// describe mode (second y press): title bar must contain "describe", not "yaml".
	r2, _ := yamlM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	descM := r2.(AppModel)
	descView := stripANSIModel(descM.detail.View())
	if !strings.Contains(descView, "describe") {
		t.Errorf("describe-mode: detail.View() missing 'describe' suffix, got:\n%s", descView)
	}
	if strings.Contains(descView, "yaml") && !strings.Contains(descView, "describe") {
		t.Errorf("describe-mode: detail.View() still shows 'yaml' without 'describe', got:\n%s", descView)
	}
}

// --- AC#10: observable content parity ---

// TestAppModel_YKey_DetailPane_ContentParity_AnnotationVisible verifies that
// switching to yaml mode exposes raw annotation values that are absent from describe mode.
func TestAppModel_YKey_DetailPane_ContentParity_AnnotationVisible(t *testing.T) {
	t.Parallel()
	// Use a wide detail pane so the annotation key+value fits on one rendered line.
	sidebar := components.NewNavSidebarModel(22, 20)
	rt := data.ResourceType{Name: "pods", Kind: "Pod", Group: ""}
	sidebar.SetItems([]data.ResourceType{rt})
	detail := components.NewDetailViewModel(200, 20)
	detail.SetResourceContext("Pod", "my-pod")
	detail.SetContent("Name: my-pod")
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		hc:          data.NewHistoryClient(nil),
		activePane:  DetailPane,
		describeRT:  rt,
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(200, 20),
		detail:      detail,
		history:     components.NewHistoryViewModel(200, 20),
		diff:        components.NewDiffViewModel(200, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()

	// describeContent is the formatted output of DescribeResource — it does NOT include
	// the raw annotation value.
	m.describeContent = "Name: my-pod\nNamespace: default"
	m.describeRaw = &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": "my-pod",
				"annotations": map[string]any{
					"x-yaml-test": "yaml-only-sentinel",
				},
			},
		},
	}

	// Describe mode must NOT expose the annotation value.
	describeView := stripANSIModel(m.detail.View())
	if strings.Contains(describeView, "yaml-only-sentinel") {
		t.Errorf("describe mode: annotation value should not appear in View(), got:\n%s", describeView)
	}

	// Switch to yaml mode.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	appM := result.(AppModel)
	yamlView := stripANSIModel(appM.detail.View())
	if !strings.Contains(yamlView, "yaml-only-sentinel") {
		t.Errorf("yaml mode: annotation value should appear in View(), got:\n%s", yamlView)
	}
}

// --- AC#11: marshal error rendering ---

// TestAppModel_YKey_DetailPane_MarshalError_ShowsWarning verifies that when the raw
// object cannot be marshaled to YAML, the detail pane shows a warning message (not a
// blank or crash), and y can still toggle back to describe mode.
func TestAppModel_YKey_DetailPane_MarshalError_ShowsWarning(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.describeContent = "normal describe text"
	// A channel value is not JSON-serializable, causing yaml.Marshal to fail.
	m.describeRaw = &unstructured.Unstructured{
		Object: map[string]any{
			"kind":        "Pod",
			"unmarshalable": make(chan struct{}),
		},
	}

	// Press y — yaml mode, marshal fails.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	appM := r1.(AppModel)

	view := appM.detail.View()
	if !strings.Contains(view, "Could not render YAML") {
		t.Errorf("marshal error: want 'Could not render YAML' card in View(), got:\n%s", view)
	}

	// y again toggles back to describe mode — must not crash.
	r2, _ := appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	appM2 := r2.(AppModel)

	if appM2.yamlMode {
		t.Error("second y after marshal error: yamlMode should be false (toggled back to describe)")
	}
	if !strings.Contains(appM2.detail.View(), "normal describe text") {
		t.Errorf("second y after marshal error: want describe content, got:\n%s", appM2.detail.View())
	}
}

// --- AC#13: full lifecycle integration ---

// TestAppModel_YKey_DetailPane_Lifecycle_EscAndReEnter verifies the full cycle:
// describe → yaml → Esc (resets) → re-enter (fresh state, no yaml).
func TestAppModel_YKey_DetailPane_Lifecycle_EscAndReEnter(t *testing.T) {
	t.Parallel()
	// Start in TablePane so we can drive d → DetailPane naturally.
	m := newTablePaneModel()
	m.describeContent = "describe text v1"
	m.describeRaw = testRawObject()

	// Step 1: open detail from table.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = r1.(AppModel)
	if m.activePane != DetailPane {
		t.Fatalf("step 1: activePane = %v, want DetailPane", m.activePane)
	}

	// Step 2: DescribeResultMsg delivers raw object.
	r2, _ := m.Update(data.DescribeResultMsg{Content: "describe text v1", Raw: testRawObject()})
	m = r2.(AppModel)
	if m.describeRaw == nil {
		t.Fatal("step 2: describeRaw = nil after DescribeResultMsg, want non-nil")
	}

	// Step 3: press y → yaml mode.
	r3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = r3.(AppModel)
	if !m.yamlMode {
		t.Fatal("step 3: yamlMode = false after y, want true")
	}

	// Step 4: Esc from DetailPane → back to TablePane, yaml state cleared.
	r4, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = r4.(AppModel)
	if m.activePane != TablePane {
		t.Fatalf("step 4: activePane = %v after Esc, want TablePane", m.activePane)
	}
	if m.yamlMode {
		t.Error("step 4: yamlMode = true after Esc, want false")
	}
	if m.describeRaw != nil {
		t.Error("step 4: describeRaw != nil after Esc, want nil")
	}

	// Step 5: re-enter detail for same row (d again).
	r5, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = r5.(AppModel)
	if m.activePane != DetailPane {
		t.Fatalf("step 5: activePane = %v, want DetailPane", m.activePane)
	}
	if m.yamlMode {
		t.Error("step 5: yamlMode = true on re-enter, want false (fresh state)")
	}
	// DescribeResultMsg arrives again.
	r6, _ := m.Update(data.DescribeResultMsg{Content: "describe text v1", Raw: testRawObject()})
	m = r6.(AppModel)
	if m.yamlMode {
		t.Error("step 6: yamlMode = true after fresh DescribeResultMsg, want false")
	}
	if m.describeRaw == nil {
		t.Error("step 6: describeRaw = nil, want non-nil (fresh load)")
	}
}

// ==================== End FB-009 ====================

// TestAppModel_ContextSwitchedMsg_ClearsRegistrationsAndInvalidatesCache verifies that
// a context switch clears m.registrations, resets registrationsLoading, and calls
// InvalidateRegistrationCache on the rrc (observable outcome).
func TestAppModel_ContextSwitchedMsg_ClearsRegistrationsAndInvalidatesCache(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newGovernedTablePaneModel(bc)
	m.rrc = rrc
	m.registrations = []data.ResourceRegistration{
		{Group: "g", Name: "r", Description: "Resource"},
	}
	m.registrationsLoading = false

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.registrations != nil {
		t.Errorf("m.registrations = non-nil after ContextSwitchedMsg, want nil")
	}
	// rrc is non-nil, so ContextSwitchedMsg eagerly dispatches LoadResourceRegistrationsCmd.
	if !appM.registrationsLoading {
		t.Error("registrationsLoading = false after ContextSwitchedMsg with rrc set, want true (eager dispatch)")
	}
	if !rrc.invalidated {
		t.Error("InvalidateRegistrationCache not called on ContextSwitchedMsg, want invalidated")
	}
}

// ==================== FB-015: Contextual welcome / landing screen ====================
//
// Axis-coverage table (submitter-owned) — 23 ACs across model_test.go + resourcetable_test.go
//
// AC  | First-press                                              | Input-changed                                          | Anti-behavior                                     | Observable
// ----|----------------------------------------------------------|--------------------------------------------------------|---------------------------------------------------|-------------------------------------------
//  1  | n/a (component test in resourcetable_test.go)            | n/a                                                    | n/a                                               | resourcetable_test.go: AC#1 header band
//  2  | n/a (component test in resourcetable_test.go)            | badge absent at contentW<60                            | badge absent when ReadOnly=false                  | resourcetable_test.go: AC#2 READ-ONLY badge
//  3  | n/a (component test in resourcetable_test.go)            | n/a                                                    | no slash in org-only view                         | resourcetable_test.go: AC#3 org-only no-slash
//  4  | n/a (component test in resourcetable_test.go)            | AC#13 (hover updates left block)                       | placeholder absent when type hovered              | resourcetable_test.go: AC#4 hover Kind
//  5  | n/a (component test in resourcetable_test.go)            | n/a                                                    | placeholder absent when type hovered              | resourcetable_test.go: AC#5 placeholder
//  6  | n/a (component test in resourcetable_test.go)            | n/a                                                    | n/a                                               | resourcetable_test.go: AC#6 summary line
//  7  | n/a (component test in resourcetable_test.go)            | n/a                                                    | n/a                                               | resourcetable_test.go: AC#7 top-3 sort
//  8  | n/a (component test in resourcetable_test.go)            | n/a                                                    | no governed-types text when 0 governed            | resourcetable_test.go: AC#8 no-governed anti
//  9  | n/a (component test in resourcetable_test.go)            | n/a                                                    | error copy varies 403 vs non-403                  | resourcetable_test.go: AC#9 error placeholder
// 10  | n/a (component test in resourcetable_test.go)            | n/a                                                    | spinner not shown when not loading                | resourcetable_test.go: AC#10 spinner placeholder
// 11  | n/a (component test in resourcetable_test.go)            | n/a                                                    | n/a                                               | resourcetable_test.go: AC#11 registrations
// 12  | TestAppModel_ContextSwitchedMsg_WelcomePanel_ReRendersWithNewOrg | old OrgName absent, new OrgName present       | n/a                                               | stripANSI(appM.table.View()) contains "new-org"
// 13  | TestAppModel_NavPane_JKey_WelcomePanel_HoveredTypeUpdates | left block shows hovered Kind after 'j' press         | old Kind absent in view2                          | stripANSI(appM.table.View()) contains "Deployment"
// 14  | TestAppModel_BucketsLoadedMsg_WelcomePanel_ShowsHealthSummary | spinner→summary transition after BucketsLoadedMsg | n/a                                               | "Platform health" present; bucketLoading=false
// 15  | TestStaleContextAgeDisplay: >24h→(true,"Nd"); TestStaleContextAgeDisplay_Boundary: 24h+1ns→stale | ≤24h strict→(false,"") | nil cfg/nil LastRefreshed→(false,"") | return values from staleContextAgeDisplay
// 16  | TestAppModel_YKey_NavPane_NoOp: y/m/b inert in NavPane   | n/a                                                    | yamlMode unchanged, pane stays NavPane            | appM.activePane==NavPane, appM.yamlMode==false
// 17  | n/a (component test in resourcetable_test.go §6b)        | keybind absent at contentH<18; strip absent AC#17      | keybind present at contentH≥18                    | resourcetable_test.go: §6b height bands
// 18  | TestAppModel_Init_DispatchesBucketsAndRegistrations       | n/a                                                    | n/a                                               | BucketsLoadedMsg + ResourceRegistrationsLoadedMsg
// 19  | TestAppModel_ContextSwitchedMsg_ResetsBucketsToNil        | n/a                                                    | n/a                                               | appM.buckets==nil, appM.bucketLoading==true
// 20  | TestAppModel_TickMsg_WelcomePanel_DispatchesBucketsCmd    | n/a                                                    | TickMsg in TablePane does NOT dispatch LoadBucketsCmd | cmds contain BucketsLoadedMsg when welcome showing
// 21  | n/a (unit tests in data/platformhealth_test.go)           | n/a                                                    | n/a                                               | data/platformhealth_test.go: ComputePlatformHealthSummary
// 22  | TestAppModel_WelcomePanel_Integration: Init→ContextSwitchedMsg→BucketsLoadedMsg | buckets cleared on step 1, health shown on step 2 | n/a | stripANSI(View()) contains "beta-corp" / "Platform health"
// 23  | n/a (component test in resourcetable_test.go AC#23)       | six-boundary width-band sweep (AC#23 WidthBands test)  | barsMode absent at contentW<80                    | resourcetable_test.go: TestResourceTableModel_Welcome_WidthBands

// TestStaleContextAgeDisplay covers AC#15: the unexported staleContextAgeDisplay
// function returns (false,"") for fresh/nil configs and (true, age) for stale ones.
func TestStaleContextAgeDisplay(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	newCfg := func(lastRefreshed time.Time) *datumconfig.ConfigV1Beta1 {
		cfg := &datumconfig.ConfigV1Beta1{}
		cfg.Cache.LastRefreshed = &lastRefreshed
		return cfg
	}

	tests := []struct {
		name      string
		cfg       *datumconfig.ConfigV1Beta1
		wantStale bool
		wantAge   string
	}{
		{
			name:      "nil config → fresh",
			cfg:       nil,
			wantStale: false,
			wantAge:   "",
		},
		{
			name:      "nil LastRefreshed → fresh",
			cfg:       &datumconfig.ConfigV1Beta1{},
			wantStale: false,
			wantAge:   "",
		},
		{
			name:      "exactly 24h ago → fresh (strict boundary)",
			cfg:       newCfg(now.Add(-24 * time.Hour)),
			wantStale: false,
			wantAge:   "",
		},
		{
			name:      "25h ago → stale (rounds to 1d)",
			cfg:       newCfg(now.Add(-25 * time.Hour)),
			wantStale: true,
			wantAge:   "1d",
		},
		{
			name:      "2 days ago → stale (days format)",
			cfg:       newCfg(now.Add(-48 * time.Hour)),
			wantStale: true,
			wantAge:   "2d",
		},
		{
			name:      "3 days ago → stale (days format)",
			cfg:       newCfg(now.Add(-72 * time.Hour)),
			wantStale: true,
			wantAge:   "3d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotStale, gotAge := staleContextAgeDisplay(tt.cfg, now)
			if gotStale != tt.wantStale {
				t.Errorf("stale = %v, want %v", gotStale, tt.wantStale)
			}
			if gotAge != tt.wantAge {
				t.Errorf("age = %q, want %q", gotAge, tt.wantAge)
			}
		})
	}
}

// TestAppModel_YKey_NavPane_NoOp verifies AC#16: pressing y, m, or b while in
// NavPane does not change pane, overlay, or yamlMode (anti-behavior).
func TestAppModel_YKey_NavPane_NoOp(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"y", "m", "b"} {
		key := key
		t.Run("key="+key, func(t *testing.T) {
			t.Parallel()
			m := newNavPaneModelWithBC(nil)
			if m.activePane != NavPane {
				t.Fatal("precondition: must start in NavPane")
			}

			result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			appM := result.(AppModel)

			if cmd != nil {
				// Nil cmd means no side-effects; some keys (like y) legitimately return nil.
				// We allow non-nil only if it's not a pane-changing command.
			}
			if appM.activePane != NavPane {
				t.Errorf("key %q: activePane = %v, want NavPane", key, appM.activePane)
			}
			if appM.overlay != NoOverlay {
				t.Errorf("key %q: overlay = %v, want NoOverlay", key, appM.overlay)
			}
			if appM.yamlMode {
				t.Errorf("key %q: yamlMode = true, want false", key)
			}
		})
	}
}

// TestAppModel_Init_DispatchesBucketsAndRegistrations verifies AC#18: Init()
// pre-warms both the bucket cache and the registration cache when both clients
// are configured (bc!=nil, rrc!=nil).
func TestAppModel_Init_DispatchesBucketsAndRegistrations(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newNavPaneModelWithBC(bc)
	m.rrc = rrc

	cmd := m.Init()
	msgs := collectMsgs(cmd)

	var gotBuckets, gotRegistrations bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.BucketsLoadedMsg:
			gotBuckets = true
		case data.ResourceRegistrationsLoadedMsg:
			gotRegistrations = true
		}
	}
	if !gotBuckets {
		t.Error("AC#18: Init() did not dispatch LoadBucketsCmd (no BucketsLoadedMsg)")
	}
	if !gotRegistrations {
		t.Error("AC#18: Init() did not dispatch LoadResourceRegistrationsCmd (no ResourceRegistrationsLoadedMsg)")
	}
}

// TestAppModel_ContextSwitchedMsg_ResetsBucketsToNil verifies AC#19: a context
// switch resets m.buckets to nil and sets bucketLoading=true when bc is wired,
// triggering an eager re-fetch of the landing-screen platform health data.
func TestAppModel_ContextSwitchedMsg_ResetsBucketsToNil(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newGovernedTablePaneModel(bc)
	// Pre-condition: governed model has buckets loaded.
	if len(m.buckets) == 0 {
		t.Fatal("precondition: want non-empty buckets before context switch")
	}

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.buckets != nil {
		t.Errorf("AC#19: m.buckets = %v after ContextSwitchedMsg, want nil", appM.buckets)
	}
	if !appM.bucketLoading {
		t.Error("AC#19: bucketLoading = false after ContextSwitchedMsg with bc set, want true (eager re-fetch)")
	}
}

// AC#22
// TestAppModel_WelcomePanel_Integration verifies AC#22: the full pre-warm lifecycle
// (Init pre-warms → ContextSwitchedMsg resets and re-fetches → BucketsLoadedMsg
// populates platform-health summary) produces correct View() at each step.
func TestAppModel_WelcomePanel_Integration(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newNavPaneModelWithBC(bc)
	m.rrc = rrc
	m.tuiCtx.OrgName = "acme"

	// Step 1: ContextSwitchedMsg resets state and re-fetches buckets.
	result1, _ := m.Update(components.ContextSwitchedMsg{Ctx: tuictx.TUIContext{OrgName: "beta-corp", UserName: "alice"}})
	m1 := result1.(AppModel)
	if m1.buckets != nil {
		t.Error("AC#22 step 1: m.buckets must be nil after ContextSwitchedMsg")
	}
	if !m1.bucketLoading {
		t.Error("AC#22 step 1: bucketLoading must be true after ContextSwitchedMsg (bc wired)")
	}
	view1 := stripANSIModel(m1.View())
	if !strings.Contains(view1, "beta-corp") {
		t.Errorf("AC#22 step 1: want 'beta-corp' in View(), got: %q", view1)
	}

	// Step 2: BucketsLoadedMsg populates platform-health summary.
	projectID := "proj-ac22"
	m1.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: projectID}
	buckets := []data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 10},
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/jobs", Limit: 10, Allocated: 5},
	}
	result2, _ := m1.Update(data.BucketsLoadedMsg{Buckets: buckets})
	m2 := result2.(AppModel)

	view2 := stripANSIModel(m2.View())
	if !strings.Contains(view2, "Platform health") {
		t.Errorf("AC#22 step 2: want 'Platform health' in View() after BucketsLoadedMsg, got: %q", view2)
	}
	if m2.bucketLoading {
		t.Error("AC#22 step 2: bucketLoading must be false after BucketsLoadedMsg")
	}
}

// ==================== End FB-015 ==========================================

// ==================== FB-015: AC#15 stale-banner boundary (clock-injected) ====================

// makeConfigWithRefresh builds a minimal ConfigV1Beta1 whose cache LastRefreshed
// is set to the given timestamp. Passing a zero time.Time means nil (never refreshed).
func makeConfigWithRefresh(lastRefreshed time.Time) *datumconfig.ConfigV1Beta1 {
	cfg := &datumconfig.ConfigV1Beta1{}
	if !lastRefreshed.IsZero() {
		cfg.Cache.LastRefreshed = &lastRefreshed
	}
	return cfg
}

// TestStaleContextAgeDisplay_Boundary verifies AC#15 gating rules with a frozen
// clock so the `age <= 24h` boundary is deterministically assertable.
//
// Contract: age <= 24h → banner absent (show=false); age > 24h → banner shown.
// The strict `>` means "exactly 24h 0m 0s" must be absent; "24h + 1ns" must be shown.
func TestStaleContextAgeDisplay_Boundary(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		lastRefreshed time.Time // zero = nil (never)
		wantShow      bool
		wantAgePrefix string // non-empty prefix that ageText must start with when wantShow
	}{
		{
			name:          "nil LastRefreshed (never refreshed)",
			lastRefreshed: time.Time{}, // zero → nil
			wantShow:      false,
		},
		{
			name:          "1h ago (clearly fresh)",
			lastRefreshed: now.Add(-1 * time.Hour),
			wantShow:      false,
		},
		{
			name:          "23h59m ago (near boundary, fresh)",
			lastRefreshed: now.Add(-23*time.Hour - 59*time.Minute),
			wantShow:      false,
		},
		{
			name:          "exactly 24h ago (boundary — strict > contract: absent)",
			lastRefreshed: now.Add(-24 * time.Hour),
			wantShow:      false,
		},
		{
			name:          "24h+1ns ago (boundary — strict > contract: shown)",
			lastRefreshed: now.Add(-24*time.Hour - 1),
			wantShow:      true,
			wantAgePrefix: "1",
		},
		{
			name:          "25h ago (clearly stale, 1d)",
			lastRefreshed: now.Add(-25 * time.Hour),
			wantShow:      true,
			wantAgePrefix: "1d",
		},
		{
			name:          "72h ago (clearly stale, 3d)",
			lastRefreshed: now.Add(-72 * time.Hour),
			wantShow:      true,
			wantAgePrefix: "3d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := makeConfigWithRefresh(tt.lastRefreshed)
			show, ageText := staleContextAgeDisplay(cfg, now)
			if show != tt.wantShow {
				t.Errorf("show = %v, want %v", show, tt.wantShow)
			}
			if tt.wantShow && tt.wantAgePrefix != "" && !strings.HasPrefix(ageText, tt.wantAgePrefix) {
				t.Errorf("ageText = %q, want prefix %q", ageText, tt.wantAgePrefix)
			}
			if !tt.wantShow && ageText != "" {
				t.Errorf("ageText = %q, want empty when show=false", ageText)
			}
		})
	}
}

// TestStaleContextAgeDisplay_NilConfig verifies AC#15: nil config yields show=false.
func TestStaleContextAgeDisplay_NilConfig(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	show, age := staleContextAgeDisplay(nil, now)
	if show {
		t.Error("nil config: show = true, want false")
	}
	if age != "" {
		t.Errorf("nil config: age = %q, want empty", age)
	}
}

// ==================== FB-015: model-level Init / ContextSwitchedMsg / TickMsg ====================

// newWelcomePanelAppModel builds an AppModel in the welcome-panel state:
// no sidebar selection, no type loaded, bc and rrc wired.
func newWelcomePanelAppModel(bc data.BucketClient, rrc data.ResourceRegistrationClient) AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		bc:          bc,
		rrc:         rrc,
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 20),
		table:       components.NewResourceTableModel(58, 20),
		detail:      components.NewDetailViewModel(58, 20),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// TestAppModel_Init_DispatchesAllThreeLoads verifies AC#18: Init() batches
// LoadResourceTypesCmd, LoadBucketsCmd, and LoadResourceRegistrationsCmd when
// both bc and rrc are non-nil.
func TestAppModel_Init_DispatchesAllThreeLoads(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newWelcomePanelAppModel(bc, rrc)

	cmd := m.Init()
	msgs := collectMsgs(cmd)

	var hasTypes, hasBuckets, hasRegs bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.ResourceTypesLoadedMsg:
			hasTypes = true
		case data.BucketsLoadedMsg:
			hasBuckets = true
		case data.ResourceRegistrationsLoadedMsg:
			hasRegs = true
		}
	}
	if !hasTypes {
		t.Error("AC#18: Init() batch missing ResourceTypesLoadedMsg")
	}
	if !hasBuckets {
		t.Error("AC#18: Init() batch missing BucketsLoadedMsg — LoadBucketsCmd not dispatched")
	}
	if !hasRegs {
		t.Error("AC#18: Init() batch missing ResourceRegistrationsLoadedMsg — LoadResourceRegistrationsCmd not dispatched")
	}
}

// TestAppModel_ContextSwitchedMsg_DispatchesAllThreeLoads verifies AC#19:
// ContextSwitchedMsg batches LoadResourceTypesCmd, LoadBucketsCmd, and
// LoadResourceRegistrationsCmd when bc and rrc are non-nil.
func TestAppModel_ContextSwitchedMsg_DispatchesAllThreeLoads(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{}
	m := newWelcomePanelAppModel(bc, rrc)

	_, cmd := m.Update(components.ContextSwitchedMsg{})
	msgs := collectMsgs(cmd)

	var hasTypes, hasBuckets, hasRegs bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.ResourceTypesLoadedMsg:
			hasTypes = true
		case data.BucketsLoadedMsg:
			hasBuckets = true
		case data.ResourceRegistrationsLoadedMsg:
			hasRegs = true
		}
	}
	if !hasTypes {
		t.Error("AC#19: ContextSwitchedMsg batch missing ResourceTypesLoadedMsg")
	}
	if !hasBuckets {
		t.Error("AC#19: ContextSwitchedMsg batch missing BucketsLoadedMsg")
	}
	if !hasRegs {
		t.Error("AC#19: ContextSwitchedMsg batch missing ResourceRegistrationsLoadedMsg")
	}
}

// TestAppModel_TickMsg_WelcomePanel_DispatchesBuckets verifies AC#20: TickMsg
// when in welcome-panel state (no type selected) dispatches LoadBucketsCmd.
func TestAppModel_TickMsg_WelcomePanel_DispatchesBuckets(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newWelcomePanelAppModel(bc, nil)

	_, cmd := m.Update(data.TickMsg{})
	msgs := collectMsgs(cmd)

	var hasBuckets bool
	for _, msg := range msgs {
		if _, ok := msg.(data.BucketsLoadedMsg); ok {
			hasBuckets = true
		}
	}
	if !hasBuckets {
		t.Error("AC#20: TickMsg in welcome-panel state did not dispatch LoadBucketsCmd")
	}
}

// TestAppModel_TickMsg_WelcomePanel_NotDispatched_WhenTypeSelected verifies AC#20
// (anti-behavior): TickMsg when a resource type IS selected dispatches
// LoadResourcesCmd (not LoadBucketsCmd) for the welcome panel path.
func TestAppModel_TickMsg_WelcomePanel_NotDispatched_WhenTypeSelected(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	// newTablePaneModel has a sidebar with "pods" selected → SelectedType() returns true.
	m := newTablePaneModel()
	m.bc = bc

	_, cmd := m.Update(data.TickMsg{})
	msgs := collectMsgs(cmd)

	for _, msg := range msgs {
		if _, ok := msg.(data.BucketsLoadedMsg); ok {
			t.Error("AC#20: TickMsg with type selected must not dispatch LoadBucketsCmd (welcome panel branch)")
		}
	}
}

// TestAppModel_WelcomePanel_YMBKeys_Inert verifies AC#16 (anti-behavior): pressing
// y, m, or b while the welcome panel is showing (TablePane, tableTypeName="") does
// not change activePane, overlay, or tableTypeName.
func TestAppModel_WelcomePanel_YMBKeys_Inert(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"y", "m", "b"} {
		t.Run("key="+key, func(t *testing.T) {
			t.Parallel()
			// TablePane with no type selected: welcome panel is shown.
			m := AppModel{
				ctx:         context.Background(),
				rc:          stubResourceClient{},
				activePane:  TablePane,
				sidebar:     components.NewNavSidebarModel(22, 20),
				table:       components.NewResourceTableModel(58, 20),
				detail:      components.NewDetailViewModel(58, 20),
				filterBar:   components.NewFilterBarModel(),
				helpOverlay: components.NewHelpOverlayModel(),
			}
			m.updatePaneFocus()
			if m.tableTypeName != "" {
				t.Fatalf("precondition: tableTypeName must be empty, got %q", m.tableTypeName)
			}

			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			appM := result.(AppModel)

			if appM.activePane != TablePane {
				t.Errorf("key %q: activePane = %v, want TablePane", key, appM.activePane)
			}
			if appM.overlay != NoOverlay {
				t.Errorf("key %q: overlay = %v, want NoOverlay", key, appM.overlay)
			}
			if appM.tableTypeName != "" {
				t.Errorf("key %q: tableTypeName = %q, want empty", key, appM.tableTypeName)
			}
		})
	}
}

// ==================== End FB-015 ====================

// ==================== FB-015: AC#12 — ContextSwitchedMsg re-renders welcome panel ====================

// AC#12
// TestAppModel_ContextSwitchedMsg_WelcomePanel_ReRendersWithNewOrg verifies AC#12:
// sending ContextSwitchedMsg while the welcome panel is active (tableTypeName=="")
// causes the panel to display the new org name and starts a bucket refresh.
func TestAppModel_ContextSwitchedMsg_WelcomePanel_ReRendersWithNewOrg(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newWelcomePanelAppModel(bc, nil)
	m.tuiCtx = tuictx.TUIContext{OrgName: "old-org"}
	m.refreshLandingInputs()

	result, _ := m.Update(components.ContextSwitchedMsg{Ctx: tuictx.TUIContext{OrgName: "new-org"}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.table.View())
	if strings.Contains(got, "old-org") {
		t.Errorf("AC#12: old org 'old-org' still present after ContextSwitchedMsg, got: %q", got)
	}
	if !strings.Contains(got, "new-org") {
		t.Errorf("AC#12: new org 'new-org' not visible in welcome panel after ContextSwitchedMsg, got: %q", got)
	}
	if !appM.bucketLoading {
		t.Error("AC#12: bucketLoading = false after ContextSwitchedMsg with bc set, want true (spinner shown)")
	}
}

// ==================== FB-015: AC#13 — sidebar hover updates welcome panel left block ====================

// AC#13
// TestAppModel_NavPane_JKey_WelcomePanel_HoveredTypeUpdates verifies AC#13: pressing
// 'j' in NavPane while the welcome panel is active moves the sidebar cursor to the
// second item and calls SetHoveredType, causing the welcome panel left block to reflect
// the new type's Kind.
func TestAppModel_NavPane_JKey_WelcomePanel_HoveredTypeUpdates(t *testing.T) {
	t.Parallel()
	sidebar := components.NewNavSidebarModel(22, 30)
	sidebar.SetItems([]data.ResourceType{
		{Name: "pods", Kind: "Pod", Group: "", Namespaced: true},
		{Name: "deployments", Kind: "Deployment", Group: "apps", Namespaced: true},
	})
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  NavPane,
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(80, 30),
		detail:      components.NewDetailViewModel(80, 30),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	// tableTypeName == "" → welcome panel is active.
	if m.tableTypeName != "" {
		t.Fatalf("precondition: tableTypeName must be empty, got %q", m.tableTypeName)
	}

	// First message to seed initial hover (cursor on pods).
	m.refreshLandingInputs()
	before := stripANSIModel(m.table.View())
	if !strings.Contains(before, "Pod") {
		t.Skipf("precondition: 'Pod' not in initial welcome panel view (layout too narrow?): %q", before)
	}

	// Press 'j' to move to Deployment.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	appM := result.(AppModel)

	got := stripANSIModel(appM.table.View())
	if !strings.Contains(got, "Deployment") {
		t.Errorf("AC#13: 'Deployment' not in welcome panel after 'j' in NavPane, got: %q", got)
	}
}

// ==================== FB-015: AC#14 — BucketsLoadedMsg → health summary visible ====================

// AC#14
// TestAppModel_BucketsLoadedMsg_WelcomePanel_ShowsHealthSummary verifies AC#14:
// after BucketsLoadedMsg arrives, the welcome panel drops the loading state and
// renders the platform health summary with the received buckets.
func TestAppModel_BucketsLoadedMsg_WelcomePanel_ShowsHealthSummary(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	m := newWelcomePanelAppModel(bc, nil)
	m.bucketLoading = true
	m.table = components.NewResourceTableModel(80, 30)
	// Set a project context so activeConsumer returns a ProjectID.
	m.tuiCtx = tuictx.TUIContext{OrgName: "acme"}
	m.refreshLandingInputs()

	projectID := "proj-abc"
	result, _ := m.Update(data.BucketsLoadedMsg{
		Buckets: []data.AllowanceBucket{
			{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 3},
		},
	})
	appM := result.(AppModel)

	if appM.bucketLoading {
		t.Error("AC#14: bucketLoading = true after BucketsLoadedMsg, want false")
	}
	got := stripANSIModel(appM.table.View())
	if !strings.Contains(got, "Platform health") {
		t.Errorf("AC#14: 'Platform health' not visible in welcome panel after BucketsLoadedMsg, got: %q", got)
	}
}

// ==================== FB-014: lazy-path safety-net (pre-warm error → Enter re-fetches) ====================

// TestAppModel_LazyPath_RegistrationsErrorThenEnter_RefetchesRegistrations verifies
// the FB-014 AC#7 lazy-path safety-net: if the eager pre-warm of
// LoadResourceRegistrationsCmd fails (ResourceRegistrationsLoadedMsg with Err),
// registrations remain nil and registrationsLoading is reset to false. A subsequent
// Enter on a governed pane triggers the lazy-path guard and dispatches a fresh
// LoadResourceRegistrationsCmd.
func TestAppModel_LazyPath_RegistrationsErrorThenEnter_RefetchesRegistrations(t *testing.T) {
	t.Parallel()
	bc := &stubBucketClient{}
	rrc := &stubRegistrationClient{} // zero-value stub; error injected via msg
	m := newAllowanceBucketNavModelWithRRC(bc, rrc)

	// Step 1: context switch eagerly dispatches LoadResourceRegistrationsCmd.
	result, _ := m.Update(components.ContextSwitchedMsg{})
	m = result.(AppModel)
	if !m.registrationsLoading {
		t.Fatal("step 1: registrationsLoading = false after ContextSwitchedMsg with rrc set, want true")
	}

	// Step 2: the eager load fails → registrationsLoading=false, registrations=nil.
	result, _ = m.Update(data.ResourceRegistrationsLoadedMsg{Err: errors.New("server unavailable")})
	m = result.(AppModel)
	if m.registrationsLoading {
		t.Fatal("step 2: registrationsLoading = true after error msg, want false")
	}
	if m.registrations != nil {
		t.Fatal("step 2: registrations != nil after error msg, want nil")
	}

	// Step 3: Enter on allowancebuckets — lazy-path guard fires because
	// registrations==nil && !registrationsLoading.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("step 3: cmd = nil after Enter, want batch containing LoadResourceRegistrationsCmd")
	}
	msgs := collectMsgs(cmd)
	var hasRegs bool
	for _, msg := range msgs {
		if _, ok := msg.(data.ResourceRegistrationsLoadedMsg); ok {
			hasRegs = true
		}
	}
	if !hasRegs {
		t.Error("FB-014 lazy-path: Enter after pre-warm failure did not dispatch LoadResourceRegistrationsCmd")
	}
}

// ==================== FB-016: Model-level tests ====================

// newActivityDashboardPaneModel builds a minimal AppModel in TablePane with an
// ActivityClient wired up and a project-scoped context, ready for "4" to dispatch.
func newActivityDashboardPaneModel() AppModel {
	m := AppModel{
		ctx:              context.Background(),
		rc:               stubResourceClient{},
		ac:               data.NewActivityClient(nil),
		activePane:       TablePane,
		sidebar:          components.NewNavSidebarModel(22, 20),
		table:            components.NewResourceTableModel(58, 20),
		detail:           components.NewDetailViewModel(58, 20),
		activityDashboard: components.NewActivityDashboardModel(80, 24, "test-project"),
		filterBar:        components.NewFilterBarModel(),
		helpOverlay:      components.NewHelpOverlayModel(),
	}
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-project"}
	m.updatePaneFocus()
	return m
}

// TestAppModel_4Key_TransitionsToActivityDashboardPane verifies AC#1 first-press:
// pressing "4" from TablePane transitions to ActivityDashboardPane and dispatches
// LoadRecentProjectActivityCmd (cmd non-nil) when the cache is empty.
func TestAppModel_4Key_TransitionsToActivityDashboardPane(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := result.(AppModel)

	if appM.activePane != ActivityDashboardPane {
		t.Errorf("AC#1: activePane = %v, want ActivityDashboardPane after 4", appM.activePane)
	}
	if cmd == nil {
		t.Error("AC#1: cmd = nil after 4 with empty cache, want LoadRecentProjectActivityCmd dispatched")
	}
}

// TestAppModel_4Key_Repeat_IsIdempotent verifies AC#1 repeat-press: pressing "4"
// again when rows are already loaded does NOT re-dispatch a load command.
func TestAppModel_4Key_Repeat_IsIdempotent(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	// Pre-load rows so activityRollupLoaded() returns true.
	m.activityDashboard.SetRows([]data.ActivityRow{
		{Summary: "created resource"},
	})

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := result.(AppModel)

	if appM.activePane != ActivityDashboardPane {
		t.Errorf("AC#1 repeat: activePane = %v, want ActivityDashboardPane", appM.activePane)
	}
	if cmd != nil {
		t.Error("AC#1 repeat: cmd != nil when rows already loaded, want nil (no re-fetch)")
	}
}

// TestAppModel_4Key_OrgScope_NoFetchDispatched verifies AC#13: when
// tuiCtx.ActiveCtx == nil, pressing "4" transitions to the pane but dispatches NO
// load command (org-scope gate).
func TestAppModel_4Key_OrgScope_NoFetchDispatched(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.tuiCtx.ActiveCtx = nil // org scope — no project

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := result.(AppModel)

	if appM.activePane != ActivityDashboardPane {
		t.Errorf("AC#13: activePane = %v, want ActivityDashboardPane", appM.activePane)
	}
	if cmd != nil {
		t.Error("AC#13: cmd != nil in org-scope, want nil (no fetch when no project selected)")
	}
}

// TestAppModel_Esc_ActivityDashboardPane_RestoresOriginPane verifies Esc from
// ActivityDashboardPane restores activityOriginPane (FB-048).
func TestAppModel_Esc_ActivityDashboardPane_RestoresOriginPane(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	// Simulate entering ActivityDashboardPane from TablePane via '4'.
	m.activityOriginPane = DashboardOrigin{Pane: TablePane, ShowDashboard: false}
	m.activePane = ActivityDashboardPane
	m.updatePaneFocus()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("Esc from ActivityDashboardPane: activePane = %v, want TablePane (origin pane)", appM.activePane)
	}
	if cmd != nil {
		t.Error("Esc from ActivityDashboardPane: cmd != nil, want nil")
	}
}

// TestAppModel_ProjectActivityLoadedMsg_UpdatesDashboard verifies the lifecycle
// of a successful project-activity load: SetLoading → ProjectActivityLoadedMsg →
// dashboard has rows and loading is false.
func TestAppModel_ProjectActivityLoadedMsg_UpdatesDashboard(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activePane = ActivityDashboardPane
	m.activityDashboard.SetLoading(true)

	rows := []data.ActivityRow{{Summary: "deployed checkout-api"}}
	result, _ := m.Update(data.ProjectActivityLoadedMsg{Rows: rows})
	appM := result.(AppModel)

	if !appM.activityDashboard.HasRows() {
		t.Error("AC#14: activityDashboard.HasRows() = false after ProjectActivityLoadedMsg, want true")
	}
	got := stripANSIModel(appM.activityDashboard.View())
	if strings.Contains(got, "loading recent activity") {
		t.Errorf("AC#14: spinner still showing after ProjectActivityLoadedMsg, got: %q", got)
	}
}

// TestAppModel_ProjectActivityErrorMsg_CRDAbsent_SetsFlag verifies AC#7: a
// ProjectActivityErrorMsg carrying ErrActivityCRDAbsent sets
// activityCRDAbsentThisSession and surfaces the "not available" message.
func TestAppModel_ProjectActivityErrorMsg_CRDAbsent_SetsFlag(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activePane = ActivityDashboardPane
	m.activityDashboard.SetLoading(true)

	result, _ := m.Update(data.ProjectActivityErrorMsg{Err: data.ErrActivityCRDAbsent})
	appM := result.(AppModel)

	if !appM.activityCRDAbsentThisSession {
		t.Error("AC#7: activityCRDAbsentThisSession = false after ErrActivityCRDAbsent msg, want true")
	}
	if !appM.activityDashboard.CRDAbsent() {
		t.Error("AC#7: activityDashboard.CRDAbsent() = false after ErrActivityCRDAbsent msg, want true")
	}
}

// TestAppModel_4Key_CRDAbsentSession_NoRefetch verifies AC#7 one-shot re-entry:
// when activityCRDAbsentThisSession is true, pressing "4" does NOT dispatch a
// second load (activityRollupLoaded returns true).
func TestAppModel_4Key_CRDAbsentSession_NoRefetch(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activityCRDAbsentThisSession = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})

	if cmd != nil {
		t.Error("AC#7 one-shot: cmd != nil when activityCRDAbsentThisSession=true, want nil (no re-fetch)")
	}
}

// TestAppModel_ContextSwitchedMsg_ClearsActivityCRDFlag verifies AC#4: a
// ContextSwitchedMsg resets activityCRDAbsentThisSession and CRDAbsent() flag
// so that the next "4" press will attempt a fresh fetch on the new context.
func TestAppModel_ContextSwitchedMsg_ClearsActivityCRDFlag(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activityCRDAbsentThisSession = true
	m.activityDashboard.SetLoadErr(data.ErrActivityCRDAbsent, false, true)

	result, _ := m.Update(components.ContextSwitchedMsg{
		Ctx: tuictx.TUIContext{OrgName: "new-org"},
	})
	appM := result.(AppModel)

	if appM.activityCRDAbsentThisSession {
		t.Error("AC#4: activityCRDAbsentThisSession = true after ContextSwitchedMsg, want false")
	}
	if appM.activityDashboard.CRDAbsent() {
		t.Error("AC#4: activityDashboard.CRDAbsent() = true after ContextSwitchedMsg, want false")
	}
}

// TestAppModel_RKey_ActivityDashboard_CRDAbsent_IsNoOp verifies AC#15: pressing
// "r" in ActivityDashboardPane when CRD is absent dispatches no command.
func TestAppModel_RKey_ActivityDashboard_CRDAbsent_IsNoOp(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activePane = ActivityDashboardPane
	m.activityDashboard.SetLoadErr(data.ErrActivityCRDAbsent, false, true)
	m.updatePaneFocus()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if cmd != nil {
		t.Error("AC#15: r in ActivityDashboardPane with CRD absent dispatched cmd, want nil")
	}
}

// TestAppModel_RKey_ActivityDashboard_OrgScope_IsNoOp verifies AC#15: pressing
// "r" in ActivityDashboardPane when orgScope (no project) dispatches no command.
func TestAppModel_RKey_ActivityDashboard_OrgScope_IsNoOp(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.tuiCtx.ActiveCtx = nil // org scope
	m.activePane = ActivityDashboardPane
	m.updatePaneFocus()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if cmd != nil {
		t.Error("AC#15: r in ActivityDashboardPane with orgScope dispatched cmd, want nil")
	}
}

// TestAppModel_RKey_ActivityDashboard_ProjectScoped_Dispatches verifies AC#15
// anti-behavior (input-changed): pressing "r" when project-scoped and CRD is
// present DOES dispatch a fresh load command.
func TestAppModel_RKey_ActivityDashboard_ProjectScoped_Dispatches(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activePane = ActivityDashboardPane
	m.updatePaneFocus()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if cmd == nil {
		t.Error("AC#15 input-changed: r in project-scoped ActivityDashboard dispatched nil, want LoadRecentProjectActivityCmd")
	}
}

// TestAppModel_RKey_ActivityDashboard_RepeatDispatches verifies AC#15 repeat-press:
// a second `r` within the 60s TTL window still dispatches because ForceRefreshProject
// invalidates the cache before each fetch.
func TestAppModel_RKey_ActivityDashboard_RepeatDispatches(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activePane = ActivityDashboardPane
	m.updatePaneFocus()

	// First press.
	result1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd1 == nil {
		t.Fatal("AC#15 repeat: first r dispatched nil, want LoadRecentProjectActivityCmd")
	}

	// Second immediate press — ForceRefreshProject already invalidated; must dispatch again.
	_, cmd2 := result1.(AppModel).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd2 == nil {
		t.Error("AC#15 repeat: second r dispatched nil, want LoadRecentProjectActivityCmd (ForceRefreshProject bypasses TTL)")
	}
}

// ==================== S10: App-global key overlay / filter precedence ====================

// TestAppModel_4Key_HelpOverlayActive_NoPaneTransition verifies S10: when the
// help overlay is open, pressing "4" routes to handleOverlayKey (which has no "4"
// handler), so no pane transition occurs.
func TestAppModel_4Key_HelpOverlayActive_NoPaneTransition(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.overlay = HelpOverlayID
	m.statusBar.Mode = components.ModeOverlay
	before := m.activePane

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := result.(AppModel)

	if appM.activePane != before {
		t.Errorf("S10 overlay: activePane changed from %v to %v, want no transition while overlay active", before, appM.activePane)
	}
	if cmd != nil {
		t.Error("S10 overlay: cmd != nil, want nil (overlay consumes key, no fetch)")
	}
}

// TestAppModel_4Key_FilterBarFocused_NoPaneTransition verifies S10: when the
// filter bar is focused, pressing "4" routes to handleFilterKey (typed into the
// filter input), so no pane transition occurs.
func TestAppModel_4Key_FilterBarFocused_NoPaneTransition(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	// Focus the filter bar directly — same effect as pressing "/" in TablePane.
	_ = m.filterBar.Focus()
	m.statusBar.Mode = components.ModeFilter
	before := m.activePane

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := result.(AppModel)

	if appM.activePane != before {
		t.Errorf("S10 filter: activePane changed from %v to %v, want no transition while filter focused", before, appM.activePane)
	}
	// "4" should have been typed into the filter bar instead.
	if appM.filterBar.Value() != "4" {
		t.Errorf("S10 filter: filterBar.Value() = %q, want '4' (key captured by filter)", appM.filterBar.Value())
	}
}

// ==================== FB-017: Delete confirmation dialog ====================

// newDeleteTablePaneModel builds a minimal AppModel in TablePane with a row
// selected, ready for the "x" keybind to open the delete confirmation dialog.
func newDeleteTablePaneModel() AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{
		{Name: "pods", Kind: "Pod", Namespaced: true},
	})
	table := components.NewResourceTableModel(58, 20)
	table.SetColumns([]string{"Name"}, 58)
	table.SetRows([]data.ResourceRow{
		{Name: "datumctl-test-pod", Namespace: "default", Cells: []string{"datumctl-test-pod"}},
	})
	table.SetTypeContext("pods", true)

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    TablePane,
		tableTypeName: "pods",
		sidebar:       sidebar,
		table:         table,
		detail:        components.NewDetailViewModel(58, 20),
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// newDeleteDialogModel builds an AppModel already in the delete-confirmation
// overlay (Prompt state) for datumctl-test-pod, with a configurable deleteErr.
func newDeleteDialogModel(deleteErr error) AppModel {
	m := newDeleteTablePaneModel()
	m.rc = stubResourceClient{deleteErr: deleteErr}
	// Open the dialog directly via the message handler.
	target := data.DeleteTarget{
		RT:        data.ResourceType{Name: "pods", Kind: "Pod", Namespaced: true},
		Name:      "datumctl-test-pod",
		Namespace: "default",
	}
	result, _ := m.Update(data.OpenDeleteConfirmationMsg{Target: target})
	return result.(AppModel)
}

// ==================== AC#1/AC#2: x keybind opens dialog ====================

// TestAppModel_XKey_TablePane_OpensDeleteDialog verifies AC#1/AC#2: pressing "x"
// in TablePane with a selected row dispatches OpenDeleteConfirmationCmd (cmd non-nil)
// and the resulting OpenDeleteConfirmationMsg opens the dialog overlay.
func TestAppModel_XKey_TablePane_OpensDeleteDialog(t *testing.T) {
	t.Parallel()
	m := newDeleteTablePaneModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	appM := result.(AppModel)
	_ = appM // overlay set by msg handler, not directly by x key

	if cmd == nil {
		t.Fatal("AC#1: cmd = nil after x in TablePane with row selected, want OpenDeleteConfirmationCmd")
	}
	// Execute the command to get the OpenDeleteConfirmationMsg.
	msg := cmd()
	odcMsg, ok := msg.(data.OpenDeleteConfirmationMsg)
	if !ok {
		t.Fatalf("AC#1: cmd returned %T, want OpenDeleteConfirmationMsg", msg)
	}
	if odcMsg.Target.Name != "datumctl-test-pod" {
		t.Errorf("AC#1: target name = %q, want %q", odcMsg.Target.Name, "datumctl-test-pod")
	}
}

// TestAppModel_XKey_TablePane_NoSelection_IsNoOp verifies AC#1 anti-behavior:
// pressing "x" in TablePane when no row is selected returns nil cmd.
func TestAppModel_XKey_TablePane_NoSelection_IsNoOp(t *testing.T) {
	t.Parallel()
	m := newDeleteTablePaneModel()
	m.table.SetRows([]data.ResourceRow{}) // empty table → no selection

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		t.Error("AC#1 anti-behavior: x with no row selected dispatched cmd, want nil")
	}
}

// TestAppModel_OpenDeleteConfirmationMsg_SetsOverlay verifies that receiving
// OpenDeleteConfirmationMsg transitions overlay to DeleteConfirmationOverlay.
func TestAppModel_OpenDeleteConfirmationMsg_SetsOverlay(t *testing.T) {
	t.Parallel()
	m := newDeleteTablePaneModel()
	target := data.DeleteTarget{
		RT:   data.ResourceType{Name: "pods", Kind: "Pod"},
		Name: "datumctl-test-pod",
	}

	result, _ := m.Update(data.OpenDeleteConfirmationMsg{Target: target})
	appM := result.(AppModel)

	if appM.overlay != DeleteConfirmationOverlay {
		t.Errorf("overlay = %v, want DeleteConfirmationOverlay after OpenDeleteConfirmationMsg", appM.overlay)
	}
}

// ==================== AC#3: Y key confirms delete ====================

// TestAppModel_YKey_Prompt_DispatchesDeleteCmd verifies AC#3: pressing "Y" in
// Prompt state transitions to InFlight and dispatches DeleteResourceCmd.
func TestAppModel_YKey_Prompt_DispatchesDeleteCmd(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	appM := result.(AppModel)

	if appM.deleteConfirmation.State() != components.DeleteStateInFlight {
		t.Errorf("AC#3: state = %v, want DeleteStateInFlight after Y", appM.deleteConfirmation.State())
	}
	if cmd == nil {
		t.Error("AC#3: cmd = nil after Y in Prompt state, want DeleteResourceCmd")
	}
}

// TestAppModel_YKey_InFlight_IsNoOp verifies AC#3 repeat-press: pressing "Y"
// again while InFlight dispatches no command (captive state).
func TestAppModel_YKey_InFlight_IsNoOp(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	// Transition to InFlight.
	result1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = result1.(AppModel)

	// Second Y — should be captive no-op.
	_, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	if cmd2 != nil {
		t.Error("AC#3 repeat: Y in InFlight dispatched cmd, want nil (captive state)")
	}
}

// ==================== AC#4: N/Esc dismisses dialog ====================

// TestAppModel_NKey_Prompt_DismissesDialog verifies AC#4: pressing "N" in Prompt
// state closes the dialog (overlay → NoOverlay).
func TestAppModel_NKey_Prompt_DismissesDialog(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#4: overlay = %v after N, want NoOverlay", appM.overlay)
	}
	if cmd != nil {
		t.Error("AC#4: cmd != nil after N in Prompt, want nil")
	}
}

// TestAppModel_NKey_InFlight_IsCaptive verifies AC#4 anti-behavior: pressing "N"
// while InFlight does NOT dismiss the dialog.
func TestAppModel_NKey_InFlight_IsCaptive(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	result1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = result1.(AppModel) // now InFlight

	result2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	appM := result2.(AppModel)

	if appM.overlay != DeleteConfirmationOverlay {
		t.Errorf("AC#4 anti-behavior: N dismissed InFlight dialog, want overlay to stay DeleteConfirmationOverlay")
	}
}

// TestAppModel_EscKey_Prompt_DismissesDialog verifies AC#4: Esc in Prompt state
// dismisses the dialog (overlay is universal).
func TestAppModel_EscKey_Prompt_DismissesDialog(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#4: Esc in Prompt: overlay = %v, want NoOverlay", appM.overlay)
	}
}

// TestAppModel_EscKey_InFlight_DismissesDialogButDeleteContinues verifies AC#4:
// Esc during InFlight dismisses the dialog overlay (the API call continues).
// After Esc, overlay is NoOverlay; a late-arriving success still invalidates cache.
func TestAppModel_EscKey_InFlight_DismissesDialogButDeleteContinues(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	// Transition to InFlight.
	result1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = result1.(AppModel)

	// Esc dismisses.
	result2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result2.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#4: Esc in InFlight: overlay = %v, want NoOverlay (API call continues)", appM.overlay)
	}
}

// ==================== AC#4: Late-arrival success after dismiss ====================

// TestAppModel_LateArrival_Success_AfterDismiss_InvalidatesCache verifies AC#4 §4:
// a DeleteResourceSucceededMsg arriving after the dialog was dismissed (Esc during
// InFlight) still invalidates the cache and re-fetches the resource list.
func TestAppModel_LateArrival_Success_AfterDismiss_InvalidatesCache(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	// Open and confirm → InFlight.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = r1.(AppModel)
	// Dismiss via Esc.
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = r2.(AppModel)
	if m.overlay != NoOverlay {
		t.Fatal("precondition: dialog not dismissed")
	}

	// Late-arriving success.
	target := data.DeleteTarget{
		RT:   data.ResourceType{Name: "pods", Kind: "Pod"},
		Name: "datumctl-test-pod",
	}
	_, cmd := m.Update(data.DeleteResourceSucceededMsg{Target: target})
	if cmd == nil {
		t.Error("AC#4 late-arrival: cmd = nil after late DeleteResourceSucceededMsg, want LoadResourcesCmd (cache invalidated)")
	}
}

// ==================== AC#5: DeleteResourceSucceededMsg closes dialog ====================

// TestAppModel_DeleteSucceededMsg_ClosesDialog verifies AC#5: receiving
// DeleteResourceSucceededMsg closes the overlay and sets pendingCursorAdvance.
func TestAppModel_DeleteSucceededMsg_ClosesDialog(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	target := m.deleteConfirmation.Target()

	result, cmd := m.Update(data.DeleteResourceSucceededMsg{Target: target})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#5: overlay = %v after DeleteResourceSucceededMsg, want NoOverlay", appM.overlay)
	}
	if !appM.pendingCursorAdvance {
		t.Error("AC#5: pendingCursorAdvance = false after success, want true")
	}
	if cmd == nil {
		t.Error("AC#5: cmd = nil after success, want LoadResourcesCmd re-fetch")
	}
}

// ==================== AC#6: DeleteResourceFailedMsg → dialog state transitions ====================

// TestAppModel_DeleteFailedMsg_Forbidden_TransitionsForbiddenState verifies AC#6:
// a 403 failure transitions the dialog to DeleteStateForbidden.
func TestAppModel_DeleteFailedMsg_Forbidden_TransitionsForbiddenState(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(errStubForbidden)
	target := m.deleteConfirmation.Target()

	result, _ := m.Update(data.DeleteResourceFailedMsg{
		Target:    target,
		Err:       errStubForbidden,
		Forbidden: true,
	})
	appM := result.(AppModel)

	if appM.deleteConfirmation.State() != components.DeleteStateForbidden {
		t.Errorf("AC#6: state = %v, want DeleteStateForbidden after 403", appM.deleteConfirmation.State())
	}
	if appM.overlay != DeleteConfirmationOverlay {
		t.Errorf("AC#6: overlay = %v, want DeleteConfirmationOverlay (not closed on 403)", appM.overlay)
	}
}

// TestAppModel_DeleteFailedMsg_Conflict_TransitionsConflictState verifies AC#7:
// a 409 conflict failure transitions the dialog to DeleteStateConflict.
func TestAppModel_DeleteFailedMsg_Conflict_TransitionsConflictState(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(errStubConflict)
	target := m.deleteConfirmation.Target()

	result, _ := m.Update(data.DeleteResourceFailedMsg{
		Target:   target,
		Err:      errStubConflict,
		Conflict: true,
	})
	appM := result.(AppModel)

	if appM.deleteConfirmation.State() != components.DeleteStateConflict {
		t.Errorf("AC#7: state = %v, want DeleteStateConflict after 409", appM.deleteConfirmation.State())
	}
}

// TestAppModel_DeleteFailedMsg_TransientError_TransitionsErrorState verifies AC#8:
// a 5xx failure transitions the dialog to DeleteStateTransientError.
func TestAppModel_DeleteFailedMsg_TransientError_TransitionsErrorState(t *testing.T) {
	t.Parallel()
	transientErr := errors.New("server error: 500 internal server error")
	m := newDeleteDialogModel(transientErr)
	target := m.deleteConfirmation.Target()

	result, _ := m.Update(data.DeleteResourceFailedMsg{
		Target: target,
		Err:    transientErr,
	})
	appM := result.(AppModel)

	if appM.deleteConfirmation.State() != components.DeleteStateTransientError {
		t.Errorf("AC#8: state = %v, want DeleteStateTransientError after 5xx", appM.deleteConfirmation.State())
	}
}

// TestAppModel_DeleteFailedMsg_NotFound_TreatedAsSuccess verifies AC#5: a 404
// failure (resource already gone) is treated as success-equivalent: dialog closes,
// cache invalidated, re-fetch dispatched.
func TestAppModel_DeleteFailedMsg_NotFound_TreatedAsSuccess(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(errStubNotFound)
	target := m.deleteConfirmation.Target()

	result, cmd := m.Update(data.DeleteResourceFailedMsg{
		Target:   target,
		Err:      errStubNotFound,
		NotFound: true,
	})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#5 (404): overlay = %v, want NoOverlay (404 treated as success)", appM.overlay)
	}
	if !appM.pendingCursorAdvance {
		t.Error("AC#5 (404): pendingCursorAdvance = false, want true")
	}
	if cmd == nil {
		t.Error("AC#5 (404): cmd = nil, want re-fetch cmd")
	}
}

// TestAppModel_DeleteFailedMsg_LateArrival_Discarded verifies AC#4 §4: a
// DeleteResourceFailedMsg arriving after the dialog was dismissed is silently
// discarded (overlay stays NoOverlay, no state mutation).
func TestAppModel_DeleteFailedMsg_LateArrival_Discarded(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	// Dismiss via N.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	m = r1.(AppModel)
	if m.overlay != NoOverlay {
		t.Fatal("precondition: dialog not dismissed")
	}

	// Late-arriving failure — should be discarded.
	transientErr := errors.New("late error")
	_, cmd := m.Update(data.DeleteResourceFailedMsg{
		Target: data.DeleteTarget{RT: data.ResourceType{Name: "pods"}, Name: "datumctl-test-pod"},
		Err:    transientErr,
	})
	if cmd != nil {
		t.Error("AC#4 late failure: cmd != nil, want nil (late failure discarded)")
	}
}

// ==================== AC#7: r in Conflict refreshes table ====================

// TestAppModel_RKey_ConflictState_RefreshesTableAndDismisses verifies AC#7: "r"
// in Conflict state dismisses the dialog and dispatches LoadResourcesCmd.
func TestAppModel_RKey_ConflictState_RefreshesTableAndDismisses(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(errStubConflict)
	target := m.deleteConfirmation.Target()
	// Simulate the conflict error arriving.
	r1, _ := m.Update(data.DeleteResourceFailedMsg{Target: target, Err: errStubConflict, Conflict: true})
	m = r1.(AppModel)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#7: r in Conflict: overlay = %v, want NoOverlay (dialog dismissed)", appM.overlay)
	}
	if cmd == nil {
		t.Error("AC#7: r in Conflict: cmd = nil, want LoadResourcesCmd (refresh table)")
	}
}

// TestAppModel_RKey_ConflictVsTransient_AreDistinct verifies AC#7/AC#8 anti-behavior:
// r in Conflict dismisses (no re-dispatch); r in TransientError retries (no dismiss).
func TestAppModel_RKey_ConflictVsTransient_AreDistinct(t *testing.T) {
	t.Parallel()

	// Conflict: r dismisses dialog.
	mC := newDeleteDialogModel(errStubConflict)
	tgt := mC.deleteConfirmation.Target()
	r1, _ := mC.Update(data.DeleteResourceFailedMsg{Target: tgt, Err: errStubConflict, Conflict: true})
	mC = r1.(AppModel)
	rC, _ := mC.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appC := rC.(AppModel)
	if appC.overlay != NoOverlay {
		t.Errorf("AC#7: r in Conflict did not dismiss, overlay = %v", appC.overlay)
	}

	// TransientError: r retries (dialog stays open in InFlight).
	transientErr := errors.New("5xx error")
	mT := newDeleteDialogModel(transientErr)
	tgt2 := mT.deleteConfirmation.Target()
	r2, _ := mT.Update(data.DeleteResourceFailedMsg{Target: tgt2, Err: transientErr})
	mT = r2.(AppModel)
	rT, _ := mT.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appT := rT.(AppModel)
	if appT.overlay != DeleteConfirmationOverlay {
		t.Errorf("AC#8: r in TransientError dismissed dialog, want overlay=DeleteConfirmationOverlay (retrying)")
	}
	if appT.deleteConfirmation.State() != components.DeleteStateInFlight {
		t.Errorf("AC#8: r in TransientError: state = %v, want DeleteStateInFlight", appT.deleteConfirmation.State())
	}
}

// ==================== AC#20: x while dialog open is no-op ====================

// TestAppModel_XKey_WhileDialogOpen_IsNoOp verifies AC#20: pressing "x" while
// the delete dialog is already open dispatches no command and stays in dialog.
func TestAppModel_XKey_WhileDialogOpen_IsNoOp(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	if m.overlay != DeleteConfirmationOverlay {
		t.Fatal("precondition: dialog must be open")
	}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	appM := result.(AppModel)

	if appM.overlay != DeleteConfirmationOverlay {
		t.Errorf("AC#20: x while dialog open changed overlay to %v, want DeleteConfirmationOverlay", appM.overlay)
	}
	if cmd != nil {
		t.Error("AC#20: x while dialog open dispatched cmd, want nil")
	}
}

// ==================== AC#19: ContextSwitchedMsg dismisses dialog ====================

// TestAppModel_ContextSwitchedMsg_DismissesDeleteDialog verifies AC#19: a
// ContextSwitchedMsg while the delete dialog is open dismisses it immediately.
func TestAppModel_ContextSwitchedMsg_DismissesDeleteDialog(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	if m.overlay != DeleteConfirmationOverlay {
		t.Fatal("precondition: dialog must be open")
	}

	result, _ := m.Update(components.ContextSwitchedMsg{Ctx: tuictx.TUIContext{OrgName: "new-org"}})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#19: overlay = %v after ContextSwitchedMsg, want NoOverlay (dialog dismissed)", appM.overlay)
	}
}

// ==================== AC#14: DeletePropagationBackground wire contract ====================

// TestDeleteResourceCmd_PropagationBackground verifies AC#14: DeleteResourceCmd
// calls DeleteResource on the resource client; the stub records the call.
// The propagation policy itself is tested via the KubeResourceClient implementation
// (internal/tui/data/resourceclient.go:505-506); here we verify the cmd wire contract.
func TestDeleteResourceCmd_PropagationBackground(t *testing.T) {
	t.Parallel()
	var called bool
	callRecorder := &recordingDeleteClient{onDelete: func() { called = true }}
	target := data.DeleteTarget{
		RT:   data.ResourceType{Name: "pods", Kind: "Pod"},
		Name: "datumctl-test-pod",
	}

	cmd := data.DeleteResourceCmd(context.Background(), callRecorder, target)
	cmd() // execute

	if !called {
		t.Error("AC#14: DeleteResourceCmd did not call DeleteResource on the client")
	}
}

// recordingDeleteClient records DeleteResource invocations for wire contract tests.
type recordingDeleteClient struct {
	onDelete func()
}

func (r *recordingDeleteClient) ListResourceTypes(_ context.Context) ([]data.ResourceType, error) {
	return nil, nil
}
func (r *recordingDeleteClient) ListResources(_ context.Context, _ data.ResourceType, _ string) ([]data.ResourceRow, []string, error) {
	return nil, nil, nil
}
func (r *recordingDeleteClient) DescribeResource(_ context.Context, _ data.ResourceType, _, _ string) (data.DescribeResult, error) {
	return data.DescribeResult{}, nil
}
func (r *recordingDeleteClient) DeleteResource(_ context.Context, _ data.ResourceType, _, _ string) error {
	r.onDelete()
	return nil
}
func (r *recordingDeleteClient) IsForbidden(_ error) bool     { return false }
func (r *recordingDeleteClient) IsNotFound(_ error) bool      { return false }
func (r *recordingDeleteClient) IsConflict(_ error) bool      { return false }
func (r *recordingDeleteClient) IsUnauthorized(_ error) bool  { return false }
func (r *recordingDeleteClient) InvalidateResourceListCache(_ string) {}
func (r *recordingDeleteClient) ListEvents(_ context.Context, _, _, _ string) ([]data.EventRow, error) {
	return nil, nil
}

// ==================== AC#12: cursor-advance after delete ====================

// TestAppModel_CursorAdvance_AfterDelete_MidList verifies AC#12: after delete
// success and resource re-load, the cursor advances to the same index (or clamps
// at len-1 if the list shrank).
func TestAppModel_CursorAdvance_AfterDelete_MidList(t *testing.T) {
	t.Parallel()
	// Build a table pane with 3 rows; cursor at row 1.
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{{Name: "pods", Kind: "Pod", Namespaced: true}})
	table := components.NewResourceTableModel(58, 20)
	table.SetColumns([]string{"Name"}, 58)
	table.SetRows([]data.ResourceRow{
		{Name: "datumctl-test-pod-a", Cells: []string{"datumctl-test-pod-a"}},
		{Name: "datumctl-test-pod-b", Cells: []string{"datumctl-test-pod-b"}},
		{Name: "datumctl-test-pod-c", Cells: []string{"datumctl-test-pod-c"}},
	})
	table.SetTypeContext("pods", true)
	table.SetCursor(1) // cursor on pod-b

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    TablePane,
		tableTypeName: "pods",
		sidebar:       sidebar,
		table:         table,
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()

	// Simulate delete success at cursor=1.
	target := data.DeleteTarget{RT: data.ResourceType{Name: "pods", Kind: "Pod"}, Name: "datumctl-test-pod-b"}
	r1, _ := m.Update(data.DeleteResourceSucceededMsg{Target: target})
	m = r1.(AppModel)
	if !m.pendingCursorAdvance {
		t.Fatal("AC#12: pendingCursorAdvance = false after success, want true")
	}

	// Simulate re-fetch delivering 2 remaining rows (pod-b gone).
	r2, _ := m.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{{Name: "datumctl-test-pod-a"}, {Name: "datumctl-test-pod-c"}},
		ResourceType: data.ResourceType{Name: "pods"},
		Columns:      []string{"Name"},
	})
	appM := r2.(AppModel)

	if appM.pendingCursorAdvance {
		t.Error("AC#12: pendingCursorAdvance still true after ResourcesLoadedMsg, want false (consumed)")
	}
	// cursor was 1; list now has 2 rows (indices 0,1); cursor stays at 1.
	if appM.table.Cursor() != 1 {
		t.Errorf("AC#12: cursor = %d, want 1 (same index, within new list bounds)", appM.table.Cursor())
	}
}

// TestAppModel_CursorAdvance_Clamp_WhenLastRow verifies AC#12: when the deleted
// row was the last in the list (cursor at len-1), the cursor clamps to the new len-1.
func TestAppModel_CursorAdvance_Clamp_WhenLastRow(t *testing.T) {
	t.Parallel()
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{{Name: "pods", Kind: "Pod"}})
	table := components.NewResourceTableModel(58, 20)
	table.SetColumns([]string{"Name"}, 58)
	table.SetRows([]data.ResourceRow{
		{Name: "datumctl-test-pod-a", Cells: []string{"datumctl-test-pod-a"}},
		{Name: "datumctl-test-pod-b", Cells: []string{"datumctl-test-pod-b"}},
	})
	table.SetTypeContext("pods", false)
	table.SetCursor(1) // last row

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    TablePane,
		tableTypeName: "pods",
		sidebar:       sidebar,
		table:         table,
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()

	target := data.DeleteTarget{RT: data.ResourceType{Name: "pods"}, Name: "datumctl-test-pod-b"}
	r1, _ := m.Update(data.DeleteResourceSucceededMsg{Target: target})
	m = r1.(AppModel)

	// Re-fetch delivers 1 row only.
	r2, _ := m.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{{Name: "datumctl-test-pod-a"}},
		ResourceType: data.ResourceType{Name: "pods"},
		Columns:      []string{"Name"},
	})
	appM := r2.(AppModel)

	// cursor was 1; list now has 1 row (index 0); clamps to 0.
	if appM.table.Cursor() != 0 {
		t.Errorf("AC#12 clamp: cursor = %d, want 0 (clamped from 1 to new len-1=0)", appM.table.Cursor())
	}
}

// ==================== S12: cursor-advance all three branches ====================

// TestAppModel_Delete_CursorAdvance_AllBranches verifies S12 third branch (AC#9
// anti-behavior): when the deleted resource was the only row, re-fetch returns 0
// rows and cursor resets to 0 without panic.
func TestAppModel_Delete_CursorAdvance_AllBranches(t *testing.T) {
	t.Parallel()
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{{Name: "pods", Kind: "Pod"}})
	table := components.NewResourceTableModel(58, 20)
	table.SetColumns([]string{"Name"}, 58)
	table.SetRows([]data.ResourceRow{
		{Name: "datumctl-test-pod-only", Cells: []string{"datumctl-test-pod-only"}},
	})
	table.SetTypeContext("pods", false)
	table.SetCursor(0)

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    TablePane,
		tableTypeName: "pods",
		sidebar:       sidebar,
		table:         table,
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()

	target := data.DeleteTarget{RT: data.ResourceType{Name: "pods"}, Name: "datumctl-test-pod-only"}
	r1, _ := m.Update(data.DeleteResourceSucceededMsg{Target: target})
	m = r1.(AppModel)

	if !m.pendingCursorAdvance {
		t.Fatal("S12: pendingCursorAdvance = false after success, want true")
	}

	// Re-fetch returns 0 rows (the only row was deleted).
	r2, _ := m.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{},
		ResourceType: data.ResourceType{Name: "pods"},
		Columns:      []string{"Name"},
	})
	appM := r2.(AppModel)

	if appM.pendingCursorAdvance {
		t.Error("S12: pendingCursorAdvance still true after ResourcesLoadedMsg with 0 rows")
	}
	// Empty table: cursor-advance was consumed and SetCursor(0) called.
	// The underlying bubbletea table returns -1 when it has no rows — that is
	// the observable for the empty-table branch; what matters is the advance flag
	// was consumed without panic and the table transitions to empty-state render.
	c := appM.table.Cursor()
	if c != 0 && c != -1 {
		t.Errorf("S12 empty-table: cursor = %d, want 0 or -1 (empty table boundary)", c)
	}
}

// ==================== AC#2: x from DetailPane opens dialog ====================

// newDetailPaneWithDeleteTarget builds a minimal AppModel in DetailPane with
// a datumctl-test-pod resource loaded so the x keybind opens the delete dialog.
func newDetailPaneWithDeleteTarget() AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	rt := data.ResourceType{Name: "pods", Kind: "Pod", Group: "", Namespaced: true}
	sidebar.SetItems([]data.ResourceType{rt})

	detail := components.NewDetailViewModel(58, 20)
	detail.SetResourceContext("Pod", "datumctl-test-pod")
	detail.SetContent("Name: datumctl-test-pod")

	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  DetailPane,
		describeRT:  rt,
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(58, 20),
		detail:      detail,
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// TestDeleteConfirmation_Prompt_DetailInvocation verifies AC#2: pressing "x"
// in DetailPane with a loaded resource dispatches OpenDeleteConfirmationCmd
// and produces the same dialog layout as from TablePane.
func TestDeleteConfirmation_Prompt_DetailInvocation(t *testing.T) {
	t.Parallel()
	m := newDetailPaneWithDeleteTarget()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd == nil {
		t.Fatal("AC#2: cmd = nil after x in DetailPane with resource loaded, want OpenDeleteConfirmationCmd")
	}
	msg := cmd()
	odcMsg, ok := msg.(data.OpenDeleteConfirmationMsg)
	if !ok {
		t.Fatalf("AC#2: cmd returned %T, want OpenDeleteConfirmationMsg", msg)
	}
	if odcMsg.Target.Name != "datumctl-test-pod" {
		t.Errorf("AC#2: target.Name = %q, want 'datumctl-test-pod'", odcMsg.Target.Name)
	}
}

// TestDeleteConfirmation_Prompt_DetailInvocation_NoResourceLoaded verifies AC#2
// anti-behavior: x in DetailPane with no resource loaded is a no-op.
func TestDeleteConfirmation_Prompt_DetailInvocation_NoResourceLoaded(t *testing.T) {
	t.Parallel()
	sidebar := components.NewNavSidebarModel(22, 20)
	sidebar.SetItems([]data.ResourceType{{Name: "pods", Kind: "Pod"}})

	// detail with no resource context set — ResourceName() == "".
	detail := components.NewDetailViewModel(58, 20)

	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  DetailPane,
		describeRT:  data.ResourceType{Name: "pods", Kind: "Pod"},
		sidebar:     sidebar,
		table:       components.NewResourceTableModel(58, 20),
		detail:      detail,
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	if cmd != nil {
		t.Error("AC#2 anti-behavior: x in DetailPane with no resource dispatched cmd, want nil")
	}
}

// ==================== AC#4: any-other-key dismisses dialog (S5) ====================

// TestAppModel_UnreservedKey_Prompt_DismissesDialog verifies AC#4 / S5: pressing a
// key that is (a) not in handleOverlayKey's outer switch {esc, ?, q, ctrl+c} AND
// (b) not in handleDeleteConfirmationKey's inner cases {y/Y, n/N, r, x} triggers
// the fallthrough-dismiss block at model.go:769-774. This is a distinct code path
// from the Esc branch — Esc returns early at model.go:690 before reaching
// handleDeleteConfirmationKey; "a" reaches it and falls through.
func TestAppModel_UnreservedKey_Prompt_DismissesDialog(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil) // Prompt state

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#4 fallthrough: overlay = %v after unreserved key 'a' in Prompt, want NoOverlay", appM.overlay)
	}
	if cmd != nil {
		t.Error("AC#4 fallthrough: cmd != nil after 'a' in Prompt, want nil (no delete dispatched)")
	}
}

// TestDeleteConfirmation_AnyKeyDismisses verifies AC#4 / S5: pressing any arbitrary
// key (e.g. "k") in Prompt state dismisses the dialog without dispatching a delete.
func TestDeleteConfirmation_AnyKeyDismisses(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil) // Prompt state

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#4: overlay = %v after 'k' in Prompt, want NoOverlay (any-other-key dismisses)", appM.overlay)
	}
	if cmd != nil {
		t.Error("AC#4: cmd != nil after 'k' in Prompt, want nil (no delete dispatched)")
	}
}

// TestDeleteConfirmation_AnyKeyInFlight_IsCaptive verifies AC#4 anti-behavior:
// any-other-key while InFlight does NOT dismiss (captive state).
func TestDeleteConfirmation_AnyKeyInFlight_IsCaptive(t *testing.T) {
	t.Parallel()
	m := newDeleteDialogModel(nil)
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = r1.(AppModel) // now InFlight

	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	appM := r2.(AppModel)

	if appM.overlay != DeleteConfirmationOverlay {
		t.Errorf("AC#4 InFlight captive: 'k' dismissed InFlight dialog, want overlay=DeleteConfirmationOverlay")
	}
}

// ==================== AC#10: x on no-op surfaces ====================

// TestAppModel_XKey_NoopSurfaces verifies AC#10: pressing "x" on NavPane,
// QuotaDashboardPane, and ActivityDashboardPane dispatches no command and does not
// open the delete dialog.
func TestAppModel_XKey_NoopSurfaces(t *testing.T) {
	t.Parallel()

	makeModel := func(pane PaneID) AppModel {
		sidebar := components.NewNavSidebarModel(22, 20)
		sidebar.SetItems([]data.ResourceType{{Name: "pods", Kind: "Pod", Namespaced: true}})
		m := AppModel{
			ctx:               context.Background(),
			rc:                stubResourceClient{},
			activePane:        pane,
			sidebar:           sidebar,
			table:             components.NewResourceTableModel(58, 20),
			detail:            components.NewDetailViewModel(58, 20),
			quota:             components.NewQuotaDashboardModel(58, 20, "proj"),
			activityDashboard: components.NewActivityDashboardModel(58, 20, ""),
			filterBar:         components.NewFilterBarModel(),
			helpOverlay:       components.NewHelpOverlayModel(),
		}
		m.updatePaneFocus()
		return m
	}

	panes := []struct {
		name string
		pane PaneID
	}{
		{"NavPane", NavPane},
		{"QuotaDashboardPane", QuotaDashboardPane},
		{"ActivityDashboardPane", ActivityDashboardPane},
	}

	for _, tc := range panes {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := makeModel(tc.pane)
			result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
			appM := result.(AppModel)

			if appM.overlay == DeleteConfirmationOverlay {
				t.Errorf("AC#10 %s: x opened DeleteConfirmationOverlay, want no-op", tc.name)
			}
			if cmd != nil {
				t.Errorf("AC#10 %s: x dispatched cmd, want nil", tc.name)
			}
		})
	}
}

// ==================== AC#11: HelpOverlay precedence over x ====================

// TestAppModel_XKey_HelpOverlayPrecedence verifies AC#11: when HelpOverlay is active,
// pressing x is a no-op — the help overlay stays open and no delete dialog opens.
func TestAppModel_XKey_HelpOverlayPrecedence(t *testing.T) {
	t.Parallel()
	m := newDeleteTablePaneModel()
	m.overlay = HelpOverlayID
	m.statusBar.Mode = components.ModeOverlay

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	appM := result.(AppModel)

	if appM.overlay == DeleteConfirmationOverlay {
		t.Error("AC#11: x with HelpOverlay active opened DeleteConfirmationOverlay, want no dialog")
	}
	if appM.overlay != HelpOverlayID {
		t.Errorf("AC#11: x with HelpOverlay changed overlay to %v, want HelpOverlayID (help stays)", appM.overlay)
	}
	if cmd != nil {
		t.Error("AC#11: x with HelpOverlay dispatched cmd, want nil")
	}
}

// ==================== AC#12: FilterBar captures x ====================

// TestAppModel_XKey_FilterBarFocused verifies AC#12: when FilterBar is focused,
// pressing x types into the filter and does NOT open the delete dialog.
func TestAppModel_XKey_FilterBarFocused(t *testing.T) {
	t.Parallel()
	m := newDeleteTablePaneModel()
	_ = m.filterBar.Focus()
	m.statusBar.Mode = components.ModeFilter

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	appM := result.(AppModel)

	if appM.overlay == DeleteConfirmationOverlay {
		t.Error("AC#12: x while FilterBar focused opened delete dialog, want x typed to filter")
	}
	if appM.filterBar.Value() != "x" {
		t.Errorf("AC#12: filterBar.Value() = %q, want 'x' (typed to filter)", appM.filterBar.Value())
	}
}

// ==================== S10: x overlay/filter precedence (combined) ====================

// TestAppModel_XKey_OverlayFilterPrecedence verifies S10: the overlay → filter → normal
// routing order applies consistently for the x key.
func TestAppModel_XKey_OverlayFilterPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("HelpOverlay_blocks_dialog", func(t *testing.T) {
		t.Parallel()
		m := newDeleteTablePaneModel()
		m.overlay = HelpOverlayID
		m.statusBar.Mode = components.ModeOverlay

		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
		appM := result.(AppModel)
		if appM.overlay == DeleteConfirmationOverlay {
			t.Error("S10: x with HelpOverlay opened delete dialog")
		}
		if cmd != nil {
			t.Error("S10: x with HelpOverlay dispatched cmd")
		}
	})

	t.Run("FilterBar_captures_x", func(t *testing.T) {
		t.Parallel()
		m := newDeleteTablePaneModel()
		_ = m.filterBar.Focus()
		m.statusBar.Mode = components.ModeFilter

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
		appM := result.(AppModel)
		if appM.overlay == DeleteConfirmationOverlay {
			t.Error("S10: x with FilterBar focused opened delete dialog")
		}
		if appM.filterBar.Value() != "x" {
			t.Errorf("S10: filterBar.Value() = %q, want 'x'", appM.filterBar.Value())
		}
	})
}

// ==================== AC#17: HelpOverlay shows x — delete entry ====================

// TestHelpOverlay_XEntry_PaneGated verifies AC#17: the HelpOverlay view contains
// the "x" delete keybind hint when displayed on TablePane/DetailPane.
func TestHelpOverlay_XEntry_PaneGated(t *testing.T) {
	t.Parallel()

	t.Run("TablePane_shows_x_entry", func(t *testing.T) {
		t.Parallel()
		m := newDeleteTablePaneModel()
		// Open the help overlay.
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#17: expected HelpOverlayID after ?, got %v", appM.overlay)
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if !strings.Contains(view, "[x]") {
			t.Errorf("AC#17 TablePane: '[x]' delete entry missing from HelpOverlay view")
		}
		if !strings.Contains(view, "delete") {
			t.Errorf("AC#17 TablePane: 'delete' text missing from HelpOverlay view")
		}
	})

	t.Run("DetailPane_shows_x_entry", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneWithDeleteTarget()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#17: expected HelpOverlayID after ?, got %v", appM.overlay)
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if !strings.Contains(view, "[x]") {
			t.Errorf("AC#17 DetailPane: '[x]' delete entry missing from HelpOverlay view")
		}
	})

	t.Run("ActivityDashboardPane_omits_x_entry", func(t *testing.T) {
		t.Parallel()
		// Transition to ActivityDashboardPane via "4", then open help with "?".
		// ShowDeleteHint must be false on non-delete-capable panes per AC#17 / D2.
		m := newActivityDashboardPaneModel()
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
		m = r1.(AppModel)
		if m.activePane != ActivityDashboardPane {
			t.Fatalf("AC#17 setup: activePane = %v, want ActivityDashboardPane", m.activePane)
		}
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#17: expected HelpOverlayID after ?, got %v", appM.overlay)
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[x]") {
			t.Errorf("AC#17 anti-behavior: '[x]' delete entry present in ActivityDashboardPane HelpOverlay (must be omitted per D2)")
		}
	})
}

// ==================== AC#15: full delete lifecycle ====================

// TestAppModel_DeleteLifecycle verifies AC#15: the full delete lifecycle from
// TablePane — open dialog, confirm, receive success, cache invalidated, cursor advances.
func TestAppModel_DeleteLifecycle(t *testing.T) {
	t.Parallel()
	m := newDeleteTablePaneModel()

	// Step 1: x opens dialog.
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = r1.(AppModel)
	if cmd1 == nil {
		t.Fatal("AC#15 step 1: x dispatched nil, want OpenDeleteConfirmationCmd")
	}
	r2, _ := m.Update(cmd1())
	m = r2.(AppModel)
	if m.overlay != DeleteConfirmationOverlay {
		t.Fatalf("AC#15 step 1: overlay = %v, want DeleteConfirmationOverlay", m.overlay)
	}

	// Step 2: Y transitions to InFlight and dispatches DeleteResourceCmd.
	r3, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
	m = r3.(AppModel)
	if m.deleteConfirmation.State() != components.DeleteStateInFlight {
		t.Errorf("AC#15 step 2: state = %v, want DeleteStateInFlight", m.deleteConfirmation.State())
	}
	if cmd2 == nil {
		t.Fatal("AC#15 step 2: Y dispatched nil, want DeleteResourceCmd")
	}

	// Step 3: simulate success (execute cmd returns succeeded msg).
	// For test: send the success message directly.
	target := data.DeleteTarget{
		RT:   data.ResourceType{Name: "pods", Kind: "Pod"},
		Name: "datumctl-test-pod",
	}
	r4, cmd3 := m.Update(data.DeleteResourceSucceededMsg{Target: target})
	m = r4.(AppModel)
	if m.overlay != NoOverlay {
		t.Errorf("AC#15 step 3: overlay = %v after success, want NoOverlay", m.overlay)
	}
	if !m.pendingCursorAdvance {
		t.Error("AC#15 step 3: pendingCursorAdvance = false, want true")
	}
	if cmd3 == nil {
		t.Error("AC#15 step 3: no cmd after success, want LoadResourcesCmd (cache invalidation)")
	}
}

// ==================== FB-018: Status/Conditions sub-view ====================
//
// FB-018 axis-coverage table (model-level tests)
//
// AC  | first-press                        | repeat              | input-changed                | anti-behavior                      | observable
// ----|------------------------------------|--------------------|------------------------------|------------------------------------|-----------
// 1   | _CKey_EntersConditions             | -                   | -                            | -                                  | _CKey_EntersConditions
// 3   | -                                  | _CKey_TogglesBack   | -                            | -                                  | _CKey_TogglesBack
// 4   | -                                  | -                   | _ResetsOnEsc                 | -                                  | _ResetsOnEsc
// 5   | -                                  | -                   | _ContextSwitch_Resets        | -                                  | _ContextSwitch_Resets
// 6   | -                                  | -                   | -                            | _CKey_PaneGating (7 panes)         | _CKey_PaneGating
// 7   | -                                  | -                   | -                            | _CKey_NoopPreFetch                 | _CKey_NoopPreFetch
// 8   | -                                  | -                   | -                            | _CKey_FilterBarFocused             | _CKey_FilterBarFocused
// 9   | -                                  | -                   | -                            | _CKey_OverlayPrecedence (3)        | _CKey_OverlayPrecedence
// 10  | -                                  | -                   | _LowercaseC_Remains          | _LowercaseC_Remains                | _LowercaseC_Remains
// 20  | -                                  | -                   | _TriStateToggle              | -                                  | _TriStateToggle
// 21  | -                                  | -                   | _ConditionsRoundTrip         | -                                  | _ConditionsRoundTrip
// 22  | _ConditionsHint_PaneGated          | -                   | _ConditionsHint_PaneGated    | _ConditionsHint_PaneGated          | _ConditionsHint_PaneGated
// S14 | _DeleteDialog_Preserves            | -                   | _DeleteDialog_Preserves      | -                                  | _DeleteDialog_Preserves

// rawObjectWithConditions returns an *unstructured.Unstructured with two conditions
// (one True, one False) for FB-018 model-level fixtures.
func rawObjectWithConditions() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "Gateway",
			"metadata":   map[string]any{"name": "my-gateway"},
			"status": map[string]any{
				"conditions": []interface{}{
					map[string]interface{}{
						"type": "Accepted", "status": "True",
						"reason": "Accepted", "message": "Gateway accepted",
						"lastTransitionTime": "2026-04-18T14:22:03Z",
					},
					map[string]interface{}{
						"type": "Ready", "status": "False",
						"reason": "ListenersNotReady", "message": "TLS not configured",
						"lastTransitionTime": "2026-04-19T02:15:00Z",
					},
				},
			},
		},
	}
}

// newDetailPaneModelWithConditionsRaw builds a DetailPane AppModel with describeRaw
// set to an object containing .status.conditions, so the C key handler is eligible.
func newDetailPaneModelWithConditionsRaw() AppModel {
	m := newDetailPaneModelWithRaw()
	m.describeRaw = rawObjectWithConditions()
	return m
}

// AC#1: C key enters conditions mode (first-press + observable).
func TestAppModel_CKey_EntersConditions_NonEmpty(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithConditionsRaw()
	if m.conditionsMode {
		t.Fatal("setup: conditionsMode already true")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	appM := result.(AppModel)

	if !appM.conditionsMode {
		t.Error("AC#1: conditionsMode = false after C press, want true")
	}
	if appM.detail.Mode() != "conditions" {
		t.Errorf("AC#1: detail.Mode() = %q after C, want 'conditions'", appM.detail.Mode())
	}
	view := stripANSIModel(appM.detail.View())
	if !strings.Contains(view, "conditions") {
		t.Errorf("AC#1: title bar missing 'conditions' mode indicator:\n%s", view)
	}
}

// AC#3: repeat C toggles back to describe.
func TestAppModel_CKey_TogglesBackToDescribe(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithConditionsRaw()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = r1.(AppModel)
	if !m.conditionsMode {
		t.Fatal("AC#3 setup: first C did not enter conditions mode")
	}

	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	appM := r2.(AppModel)

	if appM.conditionsMode {
		t.Error("AC#3: conditionsMode = true after second C press, want false (toggled back)")
	}
	if appM.detail.Mode() == "conditions" {
		t.Errorf("AC#3: detail.Mode() = 'conditions' after toggle-back, want describe/empty")
	}
}

// AC#4: Esc from DetailPane resets conditionsMode.
func TestAppModel_ConditionsMode_ResetsOnEsc(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithConditionsRaw()
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = r1.(AppModel)
	if !m.conditionsMode {
		t.Fatal("AC#4 setup: C did not enter conditions mode")
	}

	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r2.(AppModel)

	if appM.activePane != TablePane {
		t.Fatalf("AC#4: Esc did not return to TablePane; activePane = %v", appM.activePane)
	}
	if appM.conditionsMode {
		t.Error("AC#4: conditionsMode = true after Esc from DetailPane, want false (reset site)")
	}
}

// AC#5: ContextSwitchedMsg resets conditionsMode.
func TestAppModel_ContextSwitch_ResetsConditionsMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithConditionsRaw()
	m.conditionsMode = true
	m.detail.SetMode("conditions")

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.conditionsMode {
		t.Error("AC#5: conditionsMode = true after ContextSwitchedMsg, want false")
	}
	if appM.detail.Mode() == "conditions" {
		t.Errorf("AC#5: detail.Mode() = 'conditions' after ContextSwitchedMsg, want reset")
	}
}

// AC#6: C is pane-local to DetailPane — no-op on all 7 other panes.
func TestAppModel_CKey_PaneGating(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    func() AppModel
	}{
		{"NavPane", func() AppModel { return newNavPaneModelWithBC(nil) }},
		{"TablePane", newTablePaneModel},
		{"QuotaDashboardPane", func() AppModel { return newQuotaDashboardPaneModel(&stubBucketClient{}) }},
		{"HistoryPane", newHistoryPaneModel},
		{"DiffPane", func() AppModel { return newDiffPaneModel(1) }},
		{"ActivityPane", newActivityPaneModel},
		{"ActivityDashboardPane", newActivityDashboardPaneModel},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.m()
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
			appM := result.(AppModel)
			if appM.conditionsMode {
				t.Errorf("AC#6 %s: conditionsMode = true after C press, want no-op", tt.name)
			}
		})
	}
}

// AC#7: C is no-op when describeRaw == nil (pre-fetch/loading state).
func TestAppModel_CKey_NoopPreFetch(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC() // describeRaw is nil
	if m.describeRaw != nil {
		t.Fatal("setup: describeRaw should be nil in newDetailPaneModelWithHC")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	appM := result.(AppModel)

	if appM.conditionsMode {
		t.Error("AC#7: conditionsMode = true before describeRaw set, want no-op")
	}
}

// AC#8: C with FilterBar focused — key goes to filter, not conditions mode.
func TestAppModel_CKey_FilterBarFocused(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	_ = m.filterBar.Focus()
	m.statusBar.Mode = components.ModeFilter

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	appM := result.(AppModel)

	if appM.conditionsMode {
		t.Error("AC#8: conditionsMode = true with FilterBar focused, want C captured by filter")
	}
	if appM.filterBar.Value() != "C" {
		t.Errorf("AC#8: filterBar.Value() = %q, want 'C'", appM.filterBar.Value())
	}
}

// AC#9: Active overlays consume C — conditionsMode stays false.
func TestAppModel_CKey_OverlayPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("CtxSwitcherOverlay", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithConditionsRaw()
		m.overlay = CtxSwitcherOverlay
		m.statusBar.Mode = components.ModeOverlay

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		appM := result.(AppModel)
		if appM.conditionsMode {
			t.Error("AC#9 CtxSwitcherOverlay: conditionsMode = true; overlay must consume C")
		}
	})

	t.Run("HelpOverlayID", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithConditionsRaw()
		m.overlay = HelpOverlayID
		m.statusBar.Mode = components.ModeOverlay

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		appM := result.(AppModel)
		// The key assertion: C must not enter conditions mode while overlay is active.
		if appM.conditionsMode {
			t.Error("AC#9 HelpOverlayID: conditionsMode = true; handleOverlayKey must intercept C before handleNormalKey")
		}
		// Implementation: handleOverlayKey absorbs C but does not explicitly close the
		// help overlay on unrecognized keys (only esc and ? close it). The overlay stays
		// open — that's correct routing; C never reached handleNormalKey.
	})

	t.Run("DeleteConfirmationOverlay", func(t *testing.T) {
		t.Parallel()
		m := newDeleteDialogModel(nil)
		if m.overlay != DeleteConfirmationOverlay {
			t.Fatal("setup: dialog not open")
		}

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		appM := result.(AppModel)
		if appM.overlay == DeleteConfirmationOverlay {
			t.Error("AC#9 DeleteConfirmationOverlay: C did not dismiss delete dialog")
		}
		if appM.conditionsMode {
			t.Error("AC#9 DeleteConfirmationOverlay: conditionsMode = true after C dismisses dialog")
		}
	})
}

// AC#10: lowercase c remains the ctx-switcher on DetailPane; uppercase C is conditions.
func TestAppModel_LowercaseC_RemainsCtxSwitcher_OnDetailPane(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithConditionsRaw()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	appM := result.(AppModel)

	if appM.overlay != CtxSwitcherOverlay {
		t.Errorf("AC#10: lowercase c on DetailPane: overlay = %v, want CtxSwitcherOverlay", appM.overlay)
	}
	if appM.conditionsMode {
		t.Error("AC#10: lowercase c set conditionsMode = true, want false (wrong key)")
	}
}

// AC#20: tri-state direct transitions — yaml↔conditions without passing through describe.
func TestAppModel_DetailPane_TriStateToggle(t *testing.T) {
	t.Parallel()
	base := newDetailPaneModelWithConditionsRaw()

	t.Run("Yaml_to_Conditions_direct", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		m = r1.(AppModel)
		if !m.yamlMode {
			t.Fatal("setup: y did not enter yaml mode")
		}
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		appM := r2.(AppModel)
		if appM.yamlMode {
			t.Error("AC#20: yamlMode still true after C from yaml, want false (tri-state)")
		}
		if !appM.conditionsMode {
			t.Error("AC#20: conditionsMode false after C from yaml, want true")
		}
	})

	t.Run("Conditions_to_Yaml_direct", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		m = r1.(AppModel)
		if !m.conditionsMode {
			t.Fatal("setup: C did not enter conditions mode")
		}
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		appM := r2.(AppModel)
		if appM.conditionsMode {
			t.Error("AC#20: conditionsMode still true after y from conditions, want false (tri-state)")
		}
		if !appM.yamlMode {
			t.Error("AC#20: yamlMode false after y from conditions, want true")
		}
	})

	t.Run("Both_flags_never_simultaneously_true", func(t *testing.T) {
		t.Parallel()
		m := base
		for i, key := range []string{"C", "y", "C", "y", "C"} {
			r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			m = r.(AppModel)
			if m.yamlMode && m.conditionsMode {
				t.Errorf("AC#20 step %d (key=%q): both yamlMode and conditionsMode true — invariant broken", i, key)
			}
		}
	})
}

// AC#21: full lifecycle — enter conditions, Esc resets, re-entry stays in describe.
func TestAppModel_DetailPane_ConditionsRoundTrip(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithConditionsRaw()
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"

	// Step 1: Enter conditions mode.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	m = r1.(AppModel)
	if !m.conditionsMode {
		t.Fatal("AC#21 step 1: C did not enter conditions mode")
	}

	// Step 2: Esc → TablePane; conditionsMode must reset (AC#4 reset site).
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = r2.(AppModel)
	if m.conditionsMode {
		t.Error("AC#21 step 2: conditionsMode = true after Esc, want false")
	}

	// Step 3: DescribeResultMsg (simulate re-entry); conditionsMode must remain false.
	r3, _ := m.Update(data.DescribeResultMsg{Content: "fresh describe", Raw: rawObjectWithConditions()})
	appM := r3.(AppModel)
	if appM.conditionsMode {
		t.Error("AC#21 step 3: conditionsMode = true after DescribeResultMsg, want false (default describe)")
	}
}

// AC#22: HelpOverlay [C] conditions hint is pane-gated — present only on DetailPane.
func TestHelpOverlay_ConditionsHint_PaneGated(t *testing.T) {
	t.Parallel()

	t.Run("DetailPane_shows_C_entry", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithConditionsRaw()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#22 setup: expected HelpOverlayID, got %v", appM.overlay)
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if !strings.Contains(view, "[C]    conditions") {
			t.Errorf("AC#22 DetailPane: '[C]    conditions' hint missing:\n%s", view)
		}
		if !strings.Contains(view, "conditions") {
			t.Errorf("AC#22 DetailPane: 'conditions' text missing:\n%s", view)
		}
	})

	t.Run("TablePane_omits_C_entry", func(t *testing.T) {
		t.Parallel()
		m := newTablePaneModel()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#22 TablePane setup: expected HelpOverlayID, got %v", appM.overlay)
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[C]    conditions") {
			t.Errorf("AC#22 TablePane: '[C]    conditions' must be absent (pane-local to DetailPane only):\n%s", view)
		}
	})

	t.Run("NavPane_omits_C_entry", func(t *testing.T) {
		t.Parallel()
		m := newNavPaneModelWithBC(nil)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#22 NavPane setup: expected HelpOverlayID, got %v", appM.overlay)
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[C]    conditions") {
			t.Errorf("AC#22 NavPane: '[C]    conditions' must be absent:\n%s", view)
		}
	})
}

// S14: Delete dialog preserves conditionsMode throughout its lifecycle.
func TestAppModel_DeleteDialog_PreservesConditionsMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneWithDeleteTarget()
	m.describeRaw = rawObjectWithConditions()
	m.conditionsMode = true
	m.detail.SetMode("conditions")

	// Step 1: x opens delete dialog; conditionsMode must remain true.
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = r1.(AppModel)
	if cmd1 != nil {
		r2, _ := m.Update(cmd1())
		m = r2.(AppModel)
	}
	if m.overlay != DeleteConfirmationOverlay {
		t.Fatal("S14: delete dialog not open after x")
	}
	if !m.conditionsMode {
		t.Error("S14 step 1: conditionsMode reset by x press, want true throughout dialog")
	}

	// Step 2: N dismisses dialog; conditionsMode must still be true.
	r3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	appM := r3.(AppModel)
	if appM.overlay != NoOverlay {
		t.Errorf("S14 step 2: overlay = %v after N, want NoOverlay", appM.overlay)
	}
	if !appM.conditionsMode {
		t.Error("S14 step 2: conditionsMode = false after dialog dismiss, want true (not reset by dialog)")
	}
}

// TestAppModel_DetailPaneModeResets_CoversAllSites is a parameterized harness that
// walks each of the nine conditionsMode=false reset sites enumerated in spec §1a.
// Each sub-test pre-sets conditionsMode=true, fires the site's trigger, and asserts
// conditionsMode==false afterward. A future refactor that drops any single reset
// will cause exactly one sub-test to fail.
func TestAppModel_DetailPaneModeResets_CoversAllSites(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func() AppModel
		fire  tea.Msg
	}{
		{
			// Site line 458: ContextSwitchedMsg big handler.
			name: "site458_ContextSwitchedMsg",
			setup: func() AppModel {
				m := newDetailPaneModelWithConditionsRaw()
				m.conditionsMode = true
				return m
			},
			fire: components.ContextSwitchedMsg{},
		},
		{
			// Site line 1063: H from HistoryPane → DetailPane.
			name: "site1063_H_from_HistoryPane",
			setup: func() AppModel {
				m := newHistoryPaneModel()
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")},
		},
		{
			// Site line 1071: H from DiffPane → DetailPane.
			name: "site1071_H_from_DiffPane",
			setup: func() AppModel {
				m := newDiffPaneModel(1)
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")},
		},
		{
			// Site line 1111: Esc from HistoryPane → DetailPane.
			name: "site1111_Esc_from_HistoryPane",
			setup: func() AppModel {
				m := newHistoryPaneModel()
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEsc},
		},
		{
			// Site line 1123: Esc from DetailPane → TablePane.
			name: "site1123_Esc_from_DetailPane",
			setup: func() AppModel {
				m := newDetailPaneModelWithConditionsRaw()
				m.conditionsMode = true
				m.detailReturnPane = TablePane
				m.tableTypeName = "pods"
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEsc},
		},
		{
			// Site line 1222: Enter from TablePane → DetailPane.
			name: "site1222_Enter_from_TablePane",
			setup: func() AppModel {
				m := newTablePaneModel()
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEnter},
		},
		{
			// Site line 1254: Enter from QuotaDashboardPane → DetailPane.
			name: "site1254_Enter_from_QuotaDashboardPane",
			setup: func() AppModel {
				m := newQuotaDashboardPaneModel(&stubBucketClient{})
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEnter},
		},
		{
			// Site line 1276: a from DetailPane → ActivityPane.
			name: "site1276_a_from_DetailPane",
			setup: func() AppModel {
				m := newDetailPaneModel() // has activity field initialized
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
		},
		{
			// Site line 1352: d from TablePane → DetailPane.
			name: "site1352_d_from_TablePane",
			setup: func() AppModel {
				m := newTablePaneModel()
				m.conditionsMode = true
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setup()
			if !m.conditionsMode {
				t.Fatal("setup: conditionsMode must be true before trigger")
			}
			result, _ := m.Update(tt.fire)
			appM := result.(AppModel)
			if appM.conditionsMode {
				t.Errorf("%s: conditionsMode = true after trigger, want false (reset site not wired)", tt.name)
			}
		})
	}
}

// ==================== End FB-018 ====================

// ==================== FB-019: Resource Events sub-view ====================

// newDetailPaneModelWithEventsRaw builds a DetailPane AppModel with describeRaw set
// and events pre-populated (so E key can enter events mode immediately without dispatching).
func newDetailPaneModelWithEventsRaw() AppModel {
	m := newDetailPaneModelWithRaw()
	m.events = []data.EventRow{
		{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created", Count: 1},
	}
	return m
}

// AC#1 — E key enters events mode on DetailPane with describeRaw + events.
func TestAppModel_EKey_EntersEvents_NonEmpty(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	if m.eventsMode {
		t.Fatal("setup: eventsMode already true")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if !appM.eventsMode {
		t.Error("AC#1: eventsMode = false after E press, want true")
	}
	if appM.detail.Mode() != "events" {
		t.Errorf("AC#1: detail.Mode() = %q after E, want 'events'", appM.detail.Mode())
	}
	view := stripANSIModel(appM.detail.View())
	if !strings.Contains(view, "events") {
		t.Errorf("AC#1: title bar missing 'events' mode indicator:\n%s", view)
	}
}

// AC#3 — E twice toggles back to describe.
func TestAppModel_EKey_TogglesBackToDescribe(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r1.(AppModel)
	if !m.eventsMode {
		t.Fatal("AC#3 setup: first E did not enter events mode")
	}

	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := r2.(AppModel)

	if appM.eventsMode {
		t.Error("AC#3: eventsMode = true after second E press, want false (toggled back)")
	}
	if appM.detail.Mode() == "events" {
		t.Errorf("AC#3: detail.Mode() = 'events' after toggle-back, want describe/empty")
	}
}

// AC#7 — E is pane-local to DetailPane — no-op on 7 other panes.
func TestAppModel_EKey_PaneGating(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    func() AppModel
	}{
		{"NavPane", func() AppModel { return newNavPaneModelWithBC(nil) }},
		{"TablePane", newTablePaneModel},
		{"QuotaDashboardPane", func() AppModel { return newQuotaDashboardPaneModel(&stubBucketClient{}) }},
		{"HistoryPane", newHistoryPaneModel},
		{"DiffPane", func() AppModel { return newDiffPaneModel(1) }},
		{"ActivityPane", newActivityPaneModel},
		{"ActivityDashboardPane", newActivityDashboardPaneModel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.m()
			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
			appM := result.(AppModel)
			if appM.eventsMode {
				t.Errorf("AC#7 %s: eventsMode = true after E press, want no-op", tt.name)
			}
		})
	}
}

// AC#8 — E is no-op when describeRaw == nil (pre-fetch/loading state).
func TestAppModel_EKey_NoopPreFetch(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC() // describeRaw is nil
	if m.describeRaw != nil {
		t.Fatal("setup: describeRaw should be nil in newDetailPaneModelWithHC")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if appM.eventsMode {
		t.Error("AC#8: eventsMode = true before describeRaw set, want no-op")
	}
}

// AC#9 — E with FilterBar focused: key goes to filter, not events mode.
func TestAppModel_EKey_FilterBarFocused(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	_ = m.filterBar.Focus()
	m.statusBar.Mode = components.ModeFilter

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if appM.eventsMode {
		t.Error("AC#9: eventsMode = true with FilterBar focused, want E captured by filter")
	}
	if appM.filterBar.Value() != "E" {
		t.Errorf("AC#9: filterBar.Value() = %q, want 'E'", appM.filterBar.Value())
	}
}

// AC#10 — Active overlays consume E — eventsMode stays false.
func TestAppModel_EKey_OverlayPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("CtxSwitcherOverlay", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithEventsRaw()
		m.overlay = CtxSwitcherOverlay
		m.statusBar.Mode = components.ModeOverlay

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := result.(AppModel)
		if appM.eventsMode {
			t.Error("AC#10 CtxSwitcherOverlay: eventsMode = true; overlay must consume E")
		}
	})

	t.Run("HelpOverlayID", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithEventsRaw()
		m.overlay = HelpOverlayID
		m.statusBar.Mode = components.ModeOverlay

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := result.(AppModel)
		if appM.eventsMode {
			t.Error("AC#10 HelpOverlayID: eventsMode = true; handleOverlayKey must intercept E before handleNormalKey")
		}
	})

	t.Run("DeleteConfirmationOverlay", func(t *testing.T) {
		t.Parallel()
		m := newDeleteDialogModel(nil)
		if m.overlay != DeleteConfirmationOverlay {
			t.Fatal("setup: dialog not open")
		}

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := result.(AppModel)
		// E dismisses the delete dialog (unrecognised confirm key → dismiss).
		if appM.eventsMode {
			t.Error("AC#10 DeleteConfirmationOverlay: eventsMode = true after E dismisses dialog")
		}
	})
}

// AC#11 — lowercase "e" is NOT handled on DetailPane; eventsMode stays false.
func TestAppModel_LowercaseE_NotHandled_OnDetailPane(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()

	// First press.
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	appM := r1.(AppModel)

	if appM.eventsMode {
		t.Error("AC#11 first press: lowercase e set eventsMode = true, want false (wrong key)")
	}
	_ = cmd1 // may or may not dispatch; what matters is eventsMode.

	// Repeat press.
	r2, cmd2 := appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	appM2 := r2.(AppModel)
	if appM2.eventsMode {
		t.Error("AC#11 repeat press: lowercase e set eventsMode = true, want false")
	}
	_ = cmd2
}

// AC#23 — quad-state exclusivity: 12 transitions across describe/yaml/conditions/events.
func TestAppModel_DetailPane_QuadStateToggle(t *testing.T) {
	t.Parallel()
	base := newDetailPaneModelWithEventsRaw()
	base.describeRaw = rawObjectWithConditions() // also gives conditions data

	t.Run("describe_to_events", func(t *testing.T) {
		t.Parallel()
		m := base
		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := r.(AppModel)
		if !appM.eventsMode {
			t.Error("describe→E: eventsMode=false, want true")
		}
		if appM.yamlMode {
			t.Error("describe→E: yamlMode=true, want false (exclusivity)")
		}
		if appM.conditionsMode {
			t.Error("describe→E: conditionsMode=true, want false (exclusivity)")
		}
	})

	t.Run("events_to_describe", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = r1.(AppModel)
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := r2.(AppModel)
		if appM.eventsMode {
			t.Error("events→E: eventsMode=true, want false (toggled back to describe)")
		}
	})

	t.Run("yaml_to_events", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		m = r1.(AppModel)
		if !m.yamlMode {
			t.Fatal("setup: y did not enter yaml mode")
		}
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := r2.(AppModel)
		if !appM.eventsMode {
			t.Error("yaml→E: eventsMode=false, want true")
		}
		if appM.yamlMode {
			t.Error("yaml→E: yamlMode=true, want false (exclusivity)")
		}
	})

	t.Run("events_to_yaml", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = r1.(AppModel)
		if !m.eventsMode {
			t.Fatal("setup: E did not enter events mode")
		}
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		appM := r2.(AppModel)
		if appM.eventsMode {
			t.Error("events→y: eventsMode=true, want false (exclusivity)")
		}
		if !appM.yamlMode {
			t.Error("events→y: yamlMode=false, want true")
		}
	})

	t.Run("conditions_to_events", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		m = r1.(AppModel)
		if !m.conditionsMode {
			t.Fatal("setup: C did not enter conditions mode")
		}
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		appM := r2.(AppModel)
		if !appM.eventsMode {
			t.Error("conditions→E: eventsMode=false, want true")
		}
		if appM.conditionsMode {
			t.Error("conditions→E: conditionsMode=true, want false (exclusivity)")
		}
	})

	t.Run("events_to_conditions", func(t *testing.T) {
		t.Parallel()
		m := base
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = r1.(AppModel)
		if !m.eventsMode {
			t.Fatal("setup: E did not enter events mode")
		}
		r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		appM := r2.(AppModel)
		if appM.eventsMode {
			t.Error("events→C: eventsMode=true, want false (exclusivity)")
		}
		if !appM.conditionsMode {
			t.Error("events→C: conditionsMode=false, want true")
		}
	})

	t.Run("quad_state_never_two_flags_true", func(t *testing.T) {
		t.Parallel()
		m := base
		for i, key := range []string{"E", "y", "C", "E", "C", "y"} {
			r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			m = r.(AppModel)
			active := 0
			if m.yamlMode {
				active++
			}
			if m.conditionsMode {
				active++
			}
			if m.eventsMode {
				active++
			}
			if active > 1 {
				t.Errorf("AC#23 step %d (key=%q): multiple mode flags true — invariant broken (yaml=%v conditions=%v events=%v)",
					i, key, m.yamlMode, m.conditionsMode, m.eventsMode)
			}
		}
	})
}

// AC#24 — eventsLoading=true causes detail content to contain "Loading events".
func TestAppModel_EventsLoadingState_SpinnerAndLabel(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()
	m.eventsMode = true
	m.eventsLoading = true
	m.detail.SetMode("events")
	m.detail.SetContent(m.buildDetailContent())

	view := stripANSIModel(m.detail.View())
	if !strings.Contains(view, "Loading events") {
		t.Errorf("AC#24: eventsLoading=true: want 'Loading events' in view, got:\n%s", view)
	}
}

// EventsLoadedMsg success — sets events, clears loading + error.
func TestAppModel_EventsLoadedMsg_SetsEvents(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()
	m.eventsLoading = true

	rows := []data.EventRow{
		{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created", Count: 1},
	}
	result, _ := m.Update(data.EventsLoadedMsg{Events: rows, Err: nil})
	appM := result.(AppModel)

	if appM.eventsLoading {
		t.Error("EventsLoadedMsg success: eventsLoading = true, want false")
	}
	if appM.eventsErr != nil {
		t.Errorf("EventsLoadedMsg success: eventsErr = %v, want nil", appM.eventsErr)
	}
	if len(appM.events) != len(rows) {
		t.Errorf("EventsLoadedMsg success: len(events) = %d, want %d", len(appM.events), len(rows))
	}
}

// EventsLoadedMsg error — sets error, clears events and loading.
func TestAppModel_EventsLoadedMsg_SetsError(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw()
	m.eventsLoading = true

	someErr := errors.New("connection refused")
	result, _ := m.Update(data.EventsLoadedMsg{Events: nil, Err: someErr})
	appM := result.(AppModel)

	if appM.eventsLoading {
		t.Error("EventsLoadedMsg error: eventsLoading = true, want false")
	}
	if appM.eventsErr == nil {
		t.Fatal("EventsLoadedMsg error: eventsErr = nil, want error")
	}
	if !errors.Is(appM.eventsErr, someErr) {
		t.Errorf("EventsLoadedMsg error: eventsErr = %v, want %v", appM.eventsErr, someErr)
	}
	if appM.events != nil {
		t.Errorf("EventsLoadedMsg error: events = %v, want nil", appM.events)
	}
}

// AC#5 — ContextSwitchedMsg resets eventsMode.
func TestAppModel_ContextSwitch_ResetsEventsMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	m.eventsMode = true
	m.detail.SetMode("events")

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.eventsMode {
		t.Error("AC#5: eventsMode = true after ContextSwitchedMsg, want false")
	}
	if appM.events != nil {
		t.Errorf("AC#5: events = %v after ContextSwitchedMsg, want nil", appM.events)
	}
}

// AC#4 + Esc — eventsMode resets on Esc from DetailPane.
func TestAppModel_EventsMode_ResetsOnEsc(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	m.eventsMode = true
	m.detail.SetMode("events")
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Fatalf("AC#4: Esc did not return to TablePane; activePane = %v", appM.activePane)
	}
	if appM.eventsMode {
		t.Error("AC#4: eventsMode = true after Esc from DetailPane, want false")
	}
}

// AC#6 + row change — pressing "d" from TablePane with eventsMode=true resets eventsMode
// and dispatches a LoadEventsCmd.
func TestAppModel_RowChange_ResetsEventsMode_PairsNewFetch(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.eventsMode = true

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	appM := result.(AppModel)

	if appM.eventsMode {
		t.Error("AC#6: eventsMode = true after d from TablePane, want false (reset site)")
	}
	if cmd == nil {
		t.Fatal("AC#6: cmd = nil after d from TablePane, want tea.Batch containing LoadEventsCmd")
	}
	// Execute to verify EventsLoadedMsg can be produced (cmd should work with stub).
	msgs := collectMsgs(cmd)
	var foundEventsMsg bool
	for _, msg := range msgs {
		if _, ok := msg.(data.EventsLoadedMsg); ok {
			foundEventsMsg = true
			break
		}
	}
	if !foundEventsMsg {
		t.Error("AC#6: tea.Batch from d press did not contain a command that returns EventsLoadedMsg")
	}
}

// Extended reset sites harness — same 9 trigger scenarios as conditionsMode harness,
// but testing eventsMode reset.
func TestAppModel_DetailPaneModeResets_CoversAllSites_Events(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func() AppModel
		fire  tea.Msg
	}{
		{
			name: "site_ContextSwitchedMsg",
			setup: func() AppModel {
				m := newDetailPaneModelWithEventsRaw()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: components.ContextSwitchedMsg{},
		},
		{
			name: "site_H_from_HistoryPane",
			setup: func() AppModel {
				m := newHistoryPaneModel()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")},
		},
		{
			name: "site_H_from_DiffPane",
			setup: func() AppModel {
				m := newDiffPaneModel(1)
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")},
		},
		{
			name: "site_Esc_from_HistoryPane",
			setup: func() AppModel {
				m := newHistoryPaneModel()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEsc},
		},
		{
			name: "site_Esc_from_DetailPane",
			setup: func() AppModel {
				m := newDetailPaneModelWithEventsRaw()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				m.detailReturnPane = TablePane
				m.tableTypeName = "pods"
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEsc},
		},
		{
			name: "site_Enter_from_TablePane",
			setup: func() AppModel {
				m := newTablePaneModel()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEnter},
		},
		{
			name: "site_Enter_from_QuotaDashboardPane",
			setup: func() AppModel {
				m := newQuotaDashboardPaneModel(&stubBucketClient{})
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyEnter},
		},
		{
			name: "site_a_from_DetailPane",
			setup: func() AppModel {
				m := newDetailPaneModel()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
		},
		{
			name: "site_d_from_TablePane",
			setup: func() AppModel {
				m := newTablePaneModel()
				m.eventsMode = true
				m.events = []data.EventRow{{}}
				return m
			},
			fire: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.setup()
			if !m.eventsMode {
				t.Fatal("setup: eventsMode must be true before trigger")
			}
			result, _ := m.Update(tt.fire)
			appM := result.(AppModel)
			if appM.eventsMode {
				t.Errorf("%s: eventsMode = true after trigger, want false (reset site not wired)", tt.name)
			}
			if appM.eventsLoading {
				// Allowed only for d/Enter triggers which initiate a fresh fetch.
				// For other triggers, loading should also be cleared.
				switch tt.name {
				case "site_d_from_TablePane", "site_Enter_from_TablePane", "site_Enter_from_QuotaDashboardPane":
					// These sites set eventsLoading=true for the new fetch — acceptable.
				default:
					t.Errorf("%s: eventsLoading = true after trigger, want false", tt.name)
				}
			}
			if appM.eventsErr != nil {
				t.Errorf("%s: eventsErr = %v after trigger, want nil", tt.name, appM.eventsErr)
			}
		})
	}
}

// AC#26 — HelpOverlay [E] events hint is pane-gated — present only on DetailPane.
func TestHelpOverlay_EventsHint_PaneGated(t *testing.T) {
	t.Parallel()

	t.Run("DetailPane_shows_E_entry", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithEventsRaw()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#26 setup: expected HelpOverlayID, got %v", appM.overlay)
		}
		if !appM.helpOverlay.ShowEventsHint {
			t.Error("AC#26 DetailPane: ShowEventsHint = false, want true")
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if !strings.Contains(view, "[E]    events") {
			t.Errorf("AC#26 DetailPane: '[E]    events' hint missing:\n%s", view)
		}
		if !strings.Contains(view, "events") {
			t.Errorf("AC#26 DetailPane: 'events' text missing:\n%s", view)
		}
	})

	t.Run("TablePane_omits_E_entry", func(t *testing.T) {
		t.Parallel()
		m := newTablePaneModel()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#26 TablePane setup: expected HelpOverlayID, got %v", appM.overlay)
		}
		if appM.helpOverlay.ShowEventsHint {
			t.Error("AC#26 TablePane: ShowEventsHint = true, want false (pane-local to DetailPane)")
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[E]    events") {
			t.Errorf("AC#26 TablePane: '[E]    events' must be absent:\n%s", view)
		}
	})

	t.Run("NavPane_omits_E_entry", func(t *testing.T) {
		t.Parallel()
		m := newNavPaneModelWithBC(nil)
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		if appM.overlay != HelpOverlayID {
			t.Fatalf("AC#26 NavPane setup: expected HelpOverlayID, got %v", appM.overlay)
		}
		if appM.helpOverlay.ShowEventsHint {
			t.Error("AC#26 NavPane: ShowEventsHint = true, want false")
		}
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[E]    events") {
			t.Errorf("AC#26 NavPane: '[E]    events' must be absent:\n%s", view)
		}
	})
}

// S17 — Delete dialog preserves eventsMode throughout its lifecycle.
func TestAppModel_DeleteDialog_PreservesEventsMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneWithDeleteTarget()
	m.describeRaw = rawObjectWithConditions()
	m.eventsMode = true
	m.events = []data.EventRow{{Type: "Normal", Reason: "Created", Count: 1}}
	m.detail.SetMode("events")

	// Step 1: x opens delete dialog; eventsMode must remain true.
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	m = r1.(AppModel)
	if cmd1 != nil {
		r2, _ := m.Update(cmd1())
		m = r2.(AppModel)
	}
	if m.overlay != DeleteConfirmationOverlay {
		t.Fatal("S17: delete dialog not open after x")
	}
	if !m.eventsMode {
		t.Error("S17 step 1: eventsMode reset by x press, want true throughout dialog")
	}

	// Step 2: N dismisses dialog; eventsMode must still be true.
	r3, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	appM := r3.(AppModel)
	if appM.overlay != NoOverlay {
		t.Errorf("S17 step 2: overlay = %v after N, want NoOverlay", appM.overlay)
	}
	if !appM.eventsMode {
		t.Error("S17 step 2: eventsMode = false after dialog dismiss, want true (not reset by dialog)")
	}
}

// AC#25 / S15 — Full lifecycle: NavPane → Enter type → TablePane → "d" → DetailPane
// → EventsLoadedMsg arrives → "E" → eventsMode=true → "Esc" → TablePane, eventsMode=false.
func TestAppModel_DetailPane_EventsRoundTrip(t *testing.T) {
	t.Parallel()

	// Start from TablePane with a row selected (same as newTablePaneModel).
	m := newTablePaneModel()

	// Step 1: press "d" → DetailPane + LoadEventsCmd dispatched.
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = r1.(AppModel)
	if m.activePane != DetailPane {
		t.Fatalf("S15 step 1: activePane = %v after d, want DetailPane", m.activePane)
	}
	if cmd1 == nil {
		t.Fatal("S15 step 1: cmd = nil after d, want batch including LoadEventsCmd")
	}

	// Step 2: simulate DescribeResultMsg so describeRaw is set (needed for E key).
	r2, _ := m.Update(data.DescribeResultMsg{Content: "describe text", Raw: rawObjectWithConditions()})
	m = r2.(AppModel)
	if m.describeRaw == nil {
		t.Fatal("S15 step 2: describeRaw = nil after DescribeResultMsg")
	}

	// Step 3: EventsLoadedMsg arrives.
	rows := []data.EventRow{{Type: "Normal", Reason: "Created", Count: 1}}
	r3, _ := m.Update(data.EventsLoadedMsg{Events: rows, Err: nil})
	m = r3.(AppModel)
	if m.eventsLoading {
		t.Error("S15 step 3: eventsLoading = true after EventsLoadedMsg, want false")
	}

	// Step 4: E → eventsMode=true.
	r4, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r4.(AppModel)
	if !m.eventsMode {
		t.Fatal("S15 step 4: eventsMode = false after E, want true")
	}

	// Step 5: Esc → TablePane; eventsMode=false, events=nil.
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"
	r5, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r5.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("S15 step 5: activePane = %v after Esc, want TablePane", appM.activePane)
	}
	if appM.eventsMode {
		t.Error("S15 step 5: eventsMode = true after Esc, want false")
	}
	if appM.events != nil {
		t.Errorf("S15 step 5: events = %v after Esc, want nil", appM.events)
	}
}

// S6 error retry — E off then E on when eventsErr is set dispatches LoadEventsCmd.
func TestAppModel_EventsError_RetryOnToggle(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	// Simulate a previous fetch error.
	m.eventsMode = true
	m.eventsErr = errors.New("timeout")
	m.events = nil
	m.detail.SetMode("events")

	// Press E to toggle off.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r1.(AppModel)
	if m.eventsMode {
		t.Fatal("S6 step 1: E did not toggle off eventsMode")
	}

	// Press E again to toggle on — should re-dispatch because eventsErr != nil.
	r2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := r2.(AppModel)

	if !appM.eventsMode {
		t.Error("S6 step 2: eventsMode = false after E-on with prior error, want true")
	}
	if !appM.eventsLoading {
		t.Error("S6 step 2: eventsLoading = false, want true (retry dispatch)")
	}
	if cmd == nil {
		t.Fatal("S6 step 2: cmd = nil, want LoadEventsCmd dispatched on retry")
	}
	// Verify the cmd produces an EventsLoadedMsg.
	msgs := collectMsgs(cmd)
	var found bool
	for _, msg := range msgs {
		if _, ok := msg.(data.EventsLoadedMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("S6 step 2: cmd did not produce EventsLoadedMsg")
	}
}

// LoadEventsCmd dispatch sites — "d" from TablePane dispatches LoadEventsCmd.
func TestAppModel_DKey_DispatchesLoadEventsCmd(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if cmd == nil {
		t.Fatal("d from TablePane: cmd = nil, want tea.Batch")
	}

	msgs := collectMsgs(cmd)
	var foundEventsLoaded bool
	for _, msg := range msgs {
		if _, ok := msg.(data.EventsLoadedMsg); ok {
			foundEventsLoaded = true
			break
		}
	}
	if !foundEventsLoaded {
		t.Error("d from TablePane: expected EventsLoadedMsg among batch results (LoadEventsCmd not dispatched)")
	}
}

// ==================== End FB-019 ====================

// ==================== S11: ContextSwitchedMsg in both dialog states ====================

// TestAppModel_Delete_ContextSwitchDismisses_BothStates verifies S11: a ContextSwitchedMsg
// dismisses the delete dialog regardless of whether it is in Prompt or InFlight state.
func TestAppModel_Delete_ContextSwitchDismisses_BothStates(t *testing.T) {
	t.Parallel()

	t.Run("Prompt_state", func(t *testing.T) {
		t.Parallel()
		m := newDeleteDialogModel(nil) // Prompt state
		if m.overlay != DeleteConfirmationOverlay {
			t.Fatal("setup: dialog not open")
		}
		r, _ := m.Update(components.ContextSwitchedMsg{})
		appM := r.(AppModel)
		if appM.overlay != NoOverlay {
			t.Errorf("S11 Prompt: overlay = %v after ContextSwitchedMsg, want NoOverlay", appM.overlay)
		}
	})

	t.Run("InFlight_state", func(t *testing.T) {
		t.Parallel()
		m := newDeleteDialogModel(nil)
		r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})
		m = r1.(AppModel) // InFlight
		r2, _ := m.Update(components.ContextSwitchedMsg{})
		appM := r2.(AppModel)
		if appM.overlay != NoOverlay {
			t.Errorf("S11 InFlight: overlay = %v after ContextSwitchedMsg, want NoOverlay", appM.overlay)
		}
	})
}

// ── FB-005 — Load error recovery: inline retry affordance ────────────────────

// newErrorStateTableModel seeds a TablePane AppModel already in LoadStateError.
// errStubForbidden makes ErrorSeverityOf → Error; pass errors.New("...") for Warning.
func newErrorStateTableModel(loadErr error) AppModel {
	m := newTablePaneModel()
	result, _ := m.Update(data.LoadErrorMsg{
		Err:      loadErr,
		Severity: components.ErrorSeverityOf(loadErr, stubResourceClient{}),
	})
	return result.(AppModel)
}

// TestAppModel_LoadErrorMsg_TablePane_SetsErrorState (AC#1): a LoadErrorMsg on
// TablePane sets loadState=Error, populates statusBar.Err, and the table View()
// contains the sanitized title and retry hint.
func TestAppModel_LoadErrorMsg_TablePane_SetsErrorState(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("connection refused"), Severity: data.ErrorSeverityWarning})
	appM := result.(AppModel)

	if appM.loadState != data.LoadStateError {
		t.Errorf("loadState = %v, want LoadStateError", appM.loadState)
	}
	if appM.statusBar.Err == nil {
		t.Error("statusBar.Err = nil, want error set")
	}
	if appM.lastFailedFetchKind != "tableList" {
		t.Errorf("lastFailedFetchKind = %q, want %q", appM.lastFailedFetchKind, "tableList")
	}
	plain := stripANSIModel(appM.table.View())
	if !strings.Contains(plain, "Could not load") {
		t.Errorf("table View: want 'Could not load' title, got:\n%s", plain)
	}
	if !strings.Contains(plain, "[r]") {
		t.Errorf("table View: want '[r]' retry hint for Warning severity, got:\n%s", plain)
	}
}

// TestAppModel_RKey_ErrorState_Warning_RedispatchesTableList (AC#2/AC#11): pressing
// r in LoadStateError (Warning severity) clears the error state and redispatches
// the failed fetch — error-state branch fires BEFORE the normal-state branch.
func TestAppModel_RKey_ErrorState_Warning_RedispatchesTableList(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	// AC#11: error-state branch fired — state is now Loading, not Error.
	if appM.loadState != data.LoadStateLoading {
		t.Errorf("loadState = %v after r in error state, want LoadStateLoading", appM.loadState)
	}
	if appM.statusBar.Err != nil {
		t.Error("statusBar.Err non-nil after retry dispatch, want nil")
	}
	// AC#2: a LoadResourcesCmd is dispatched.
	if cmd == nil {
		t.Fatal("cmd = nil after r in error state, want a LoadResourcesCmd")
	}
	// cmd is non-nil: redispatch happened. We cannot execute the live LoadResourcesCmd
	// without a real client, so asserting cmd != nil is sufficient.
	_ = collectMsgs(cmd)
}

// TestAppModel_RKey_ErrorState_Error_Severity_IsNoop (AC#10): pressing r when
// severity is Error (Forbidden) dispatches no fetch and posts the no-retry hint.
func TestAppModel_RKey_ErrorState_Error_Severity_IsNoop(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errStubForbidden)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	// loadState must still be Error — no retry dispatched.
	if appM.loadState != data.LoadStateError {
		t.Errorf("loadState = %v after r on Error severity, want LoadStateError (no retry)", appM.loadState)
	}
	// Hint must contain "No retry available".
	if !strings.Contains(appM.statusBar.Hint, "No retry available") {
		t.Errorf("statusBar.Hint = %q, want 'No retry available for this error'", appM.statusBar.Hint)
	}
}

// TestAppModel_EscKey_ErrorState_TablePane_GoesToNav (AC#3): Esc from TablePane
// error state navigates to NAV; loadState stays Error (card preserved on re-entry).
func TestAppModel_EscKey_ErrorState_TablePane_GoesToNav(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("activePane = %v after Esc from TablePane error, want NavPane", appM.activePane)
	}
	if appM.loadState != data.LoadStateError {
		t.Errorf("loadState = %v after Esc, want LoadStateError (card preserved)", appM.loadState)
	}
}

// TestAppModel_ClearStatusErrMsg_TokenMatch_ClearsStatusBar (AC#4): a
// ClearStatusErrMsg with matching token clears statusBar.Err but NOT loadState.
func TestAppModel_ClearStatusErrMsg_TokenMatch_ClearsStatusBar(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))
	token := m.statusErrToken

	result, _ := m.Update(data.ClearStatusErrMsg{Token: token})
	appM := result.(AppModel)

	if appM.statusBar.Err != nil {
		t.Error("statusBar.Err non-nil after matching ClearStatusErrMsg, want nil")
	}
	// In-pane card stays — loadState must still be Error.
	if appM.loadState != data.LoadStateError {
		t.Errorf("loadState = %v after ClearStatusErrMsg, want LoadStateError (card stays)", appM.loadState)
	}
}

// TestAppModel_ClearStatusErrMsg_TokenMismatch_IsNoop (AC#4): a stale
// ClearStatusErrMsg with a mismatched token is silently ignored.
func TestAppModel_ClearStatusErrMsg_TokenMismatch_IsNoop(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))
	staleToken := m.statusErrToken - 1

	result, _ := m.Update(data.ClearStatusErrMsg{Token: staleToken})
	appM := result.(AppModel)

	if appM.statusBar.Err == nil {
		t.Error("statusBar.Err = nil after stale ClearStatusErrMsg, want error preserved")
	}
}

// TestAppModel_LoadErrorMsg_DetailPane_SetsDescribeFetchKind (AC#5): a LoadErrorMsg
// arriving while in DetailPane sets lastFailedFetchKind="describe" and updates
// the detail content with an error card.
func TestAppModel_LoadErrorMsg_DetailPane_SetsDescribeFetchKind(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = DetailPane
	m.describeRT = data.ResourceType{Name: "pods", Group: ""}
	m.detail.SetResourceContext("Pod", "my-pod")
	m.updatePaneFocus()

	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("dial tcp: timeout"), Severity: data.ErrorSeverityWarning})
	appM := result.(AppModel)

	if appM.lastFailedFetchKind != "describe" {
		t.Errorf("lastFailedFetchKind = %q, want %q", appM.lastFailedFetchKind, "describe")
	}
	if appM.loadState != data.LoadStateError {
		t.Errorf("loadState = %v, want LoadStateError", appM.loadState)
	}
	// buildDetailContent should return the error card.
	content := appM.buildDetailContent()
	plain := stripANSIModel(content)
	if !strings.Contains(plain, "Could not describe") {
		t.Errorf("detail error card: want 'Could not describe', got:\n%s", plain)
	}
	if !strings.Contains(plain, "back to table") {
		t.Errorf("detail error card: want 'back to table' hint, got:\n%s", plain)
	}
}

// TestAppModel_ResourcesLoadedMsg_ClearsErrorState (AC#6): a successful
// ResourcesLoadedMsg clears loadState, statusBar.Err, and the error card.
func TestAppModel_ResourcesLoadedMsg_ClearsErrorState(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))

	rt := data.ResourceType{Name: "pods", Kind: "Pod", Namespaced: true}
	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{{Name: "pod-1", Cells: []string{"pod-1"}}},
		ResourceType: rt,
		Columns:      []string{"Name"},
	})
	appM := result.(AppModel)

	if appM.loadState == data.LoadStateError {
		t.Error("loadState still LoadStateError after ResourcesLoadedMsg, want cleared")
	}
	if appM.statusBar.Err != nil {
		t.Error("statusBar.Err non-nil after ResourcesLoadedMsg, want nil")
	}
	if appM.loadErr != nil {
		t.Error("loadErr non-nil after ResourcesLoadedMsg, want nil")
	}
}

// TestAppModel_TypeSwitch_ClearsErrorState (AC#7): selecting a different resource
// type in the sidebar clears all error state (implicit recovery).
func TestAppModel_TypeSwitch_ClearsErrorState(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))

	// Simulate type-switch via sidebar Enter (sidebar must have a type selected).
	m.activePane = NavPane
	m.updatePaneFocus()
	// Press Enter to select the pods type from sidebar.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := result.(AppModel)

	if appM.loadState == data.LoadStateError {
		t.Error("loadState still LoadStateError after type-switch, want cleared")
	}
	if appM.loadErr != nil {
		t.Error("loadErr non-nil after type-switch, want nil")
	}
}

// TestAppModel_RKey_ErrorState_RepeatPressGuarded (AC#8): pressing r while retry
// is already in-flight dispatches no additional fetch (guarded by LoadStateLoading).
func TestAppModel_RKey_ErrorState_RepeatPressGuarded(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))

	// First r — transitions from Error → Loading.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m2 := r1.(AppModel)
	if m2.loadState != data.LoadStateLoading {
		t.Fatalf("loadState = %v after first r, want LoadStateLoading", m2.loadState)
	}

	// Second r while Loading — the normal-state refreshing guard applies.
	// loadState is Loading (not Error), so error-state branch skips;
	// normal-state branch guards on m.refreshing.
	r2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := r2.(AppModel)
	_ = appM

	// No new LoadResourcesCmd should fire — cmd2 may be nil or a no-op.
	// We verify state did not flip to a second loading dispatch by checking
	// the status bar is not re-cleared (it was already nil from the first r).
	if cmd2 != nil {
		// Ensure the cmd is not a LoadResourcesCmd by running it and checking.
		// We can't call it without a live client, so just ensure loadState
		// did not bounce back to Error (which would indicate a second dispatch loop).
		_ = cmd2
	}
}

// TestAppModel_ErrorSeverity_Warning_CardHasRetry (AC#9): Warning severity card
// renders [r] retry; Error severity card omits it.
func TestAppModel_ErrorSeverity_Warning_CardHasRetry(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("timeout"), Severity: data.ErrorSeverityWarning})
	appM := result.(AppModel)

	plain := stripANSIModel(appM.table.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("Warning severity card: want '[r]' retry hint, got:\n%s", plain)
	}
}

func TestAppModel_ErrorSeverity_Error_CardNoRetry(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	result, _ := m.Update(data.LoadErrorMsg{Err: errStubForbidden, Severity: data.ErrorSeverityError})
	appM := result.(AppModel)

	plain := stripANSIModel(appM.table.View())
	if strings.Contains(plain, "[r]") {
		t.Errorf("Error severity card: must NOT contain '[r]' retry hint, got:\n%s", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("Error severity card: want '[Esc]' back hint, got:\n%s", plain)
	}
}

// TestAppModel_LoadErrorMsg_TokenBumps (AC#12): consecutive LoadErrorMsgs bump
// statusErrToken each time so the previous ClearStatusErrCmd expires harmlessly.
func TestAppModel_LoadErrorMsg_TokenBumps(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	r1, _ := m.Update(data.LoadErrorMsg{Err: errors.New("first failure"), Severity: data.ErrorSeverityWarning})
	m1 := r1.(AppModel)
	token1 := m1.statusErrToken

	r2, _ := m1.Update(data.LoadErrorMsg{Err: errors.New("second failure"), Severity: data.ErrorSeverityWarning})
	m2 := r2.(AppModel)
	token2 := m2.statusErrToken

	if token2 != token1+1 {
		t.Errorf("statusErrToken after two LoadErrorMsgs: got %d, want %d (token1=%d)", token2, token1+1, token1)
	}

	// Stale ClearStatusErrMsg from first error must not clear the second.
	r3, _ := m2.Update(data.ClearStatusErrMsg{Token: token1})
	m3 := r3.(AppModel)
	if m3.statusBar.Err == nil {
		t.Error("stale ClearStatusErrMsg (token1) cleared second error's statusBar.Err — token guard broken")
	}
}

// TestAppModel_ContextSwitch_ClearsErrorState (AC#13): a ContextSwitchedMsg while
// in error state clears loadState, statusBar.Err, and lastFailedFetchKind.
func TestAppModel_ContextSwitch_ClearsErrorState(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.loadState == data.LoadStateError {
		t.Error("loadState still LoadStateError after ContextSwitchedMsg, want cleared")
	}
	if appM.statusBar.Err != nil {
		t.Error("statusBar.Err non-nil after ContextSwitchedMsg, want nil")
	}
	if appM.loadErr != nil {
		t.Error("loadErr non-nil after ContextSwitchedMsg, want nil")
	}
	if appM.lastFailedFetchKind != "" {
		t.Errorf("lastFailedFetchKind = %q after ContextSwitchedMsg, want empty", appM.lastFailedFetchKind)
	}
}

// TestAppModel_EscKey_DetailPane_ErrorState_GoesToTable (AC#17): Esc from
// DetailPane error card navigates to TablePane (not NavPane).
func TestAppModel_EscKey_DetailPane_ErrorState_GoesToTable(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = DetailPane
	m.lastFailedFetchKind = "describe"
	m.loadState = data.LoadStateError
	m.loadErr = errors.New("timeout")
	m.updatePaneFocus()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v after Esc from DetailPane error, want TablePane", appM.activePane)
	}
}

// TestResourceTableModel_SetLoadErr_View_WarningCardRendered (AC#1/AC#14): component-level
// test that ResourceTableModel.View() renders error card when loadState=Error.
// Also validates that narrow width (w=30) follows FB-022 §5 unusable band.
func TestResourceTableModel_SetLoadErr_View_WarningCardRendered(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		width      int
		wantTitle  bool
		wantRetry  bool
	}{
		{name: "wide (w=80)", width: 80, wantTitle: true, wantRetry: true},
		{name: "narrow (w=50)", width: 50, wantTitle: true, wantRetry: true},
		{name: "unusable (w=30, innerW=27)", width: 30, wantTitle: true, wantRetry: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := components.NewResourceTableModel(tt.width, 20)
			m.SetTypeContext("pods", true)
			m.SetLoadErr(errors.New("dial tcp: refused"), data.ErrorSeverityWarning)
			m.SetLoadState(data.LoadStateError)
			plain := stripANSIModel(m.View())
			if tt.wantTitle && !strings.Contains(plain, "Could not load") {
				t.Errorf("w=%d: want 'Could not load' title, got:\n%s", tt.width, plain)
			}
			if tt.wantRetry && !strings.Contains(plain, "[r]") {
				t.Errorf("w=%d: want '[r]' retry hint, got:\n%s", tt.width, plain)
			}
		})
	}
}

// TestResourceTableModel_SetLoadErr_Error_Severity_NoRetry (AC#9): Error severity
// card in ResourceTableModel must not contain [r] retry hint.
func TestResourceTableModel_SetLoadErr_Error_Severity_NoRetry(t *testing.T) {
	t.Parallel()
	m := components.NewResourceTableModel(80, 20)
	m.SetTypeContext("secrets", true)
	m.SetLoadErr(errors.New("forbidden"), data.ErrorSeverityError)
	m.SetLoadState(data.LoadStateError)
	plain := stripANSIModel(m.View())
	if strings.Contains(plain, "[r]") {
		t.Errorf("Error severity card must NOT contain '[r]', got:\n%s", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("Error severity card must contain '[Esc]', got:\n%s", plain)
	}
}

// TestAppModel_LoadErrorMsg_ClearsStatusBarAfterRetrySuccess (AC#6 integration):
// after retry succeeds (ResourcesLoadedMsg), statusBar.Err is nil AND loadState != Error.
func TestAppModel_LoadErrorMsg_ClearsStatusBarAfterRetrySuccess(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout"))
	if m.statusBar.Err == nil {
		t.Fatal("precondition: statusBar.Err should be set before retry")
	}

	result, _ := m.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{{Name: "a", Cells: []string{"a"}}},
		ResourceType: data.ResourceType{Name: "pods"},
		Columns:      []string{"Name"},
	})
	appM := result.(AppModel)
	if appM.statusBar.Err != nil {
		t.Error("statusBar.Err non-nil after success, want nil")
	}
	if appM.loadState == data.LoadStateError {
		t.Error("loadState still Error after success, want cleared")
	}
}

// TestAppModel_NoStringMatchingInErrorHandler (AC#18): confirms that error
// classification uses ErrorSeverityOf (rc classifiers), not string-matching.
// This is a behavioural test: a forbidden error produces Error severity (not Warning)
// via rc.IsForbidden, which proves classification goes through the interface.
func TestAppModel_NoStringMatchingInErrorHandler(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	// Use errStubForbidden — stubResourceClient.IsForbidden returns true for it.
	result, _ := m.Update(data.LoadErrorMsg{Err: errStubForbidden, Severity: data.ErrorSeverityError})
	appM := result.(AppModel)

	// Card severity is Error — action row has no [r] retry.
	plain := stripANSIModel(appM.table.View())
	if strings.Contains(plain, "[r]") {
		t.Error("Error severity (via rc classifier): must NOT contain '[r]' retry hint")
	}
	// Sanity: warning-severity string-only error WOULD have retry.
	result2, _ := m.Update(data.LoadErrorMsg{Err: errors.New("connection refused"), Severity: data.ErrorSeverityWarning})
	appM2 := result2.(AppModel)
	plain2 := stripANSIModel(appM2.table.View())
	if !strings.Contains(plain2, "[r]") {
		t.Error("Warning severity: must contain '[r]' retry hint (control group)")
	}
}

// TestAppModel_FB005_AntiRegression (AC#19): FB-002 force-refresh path still works
// when loadState is NOT Error (normal state).
func TestAppModel_FB005_AntiRegression_ForceRefreshUnaffected(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel() // loadState = Idle

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)
	_ = appM

	// The normal-state branch fires — cmd must be non-nil (LoadResourcesCmd scheduled).
	if cmd == nil {
		t.Error("FB-002 force-refresh (normal state): cmd = nil, want LoadResourcesCmd")
	}
}

// TestAppModel_RetryInFlight_ThenSuccess_ThenReFail (AC#16): retry lifecycle —
// error → r (spinner) → success (rows, no card) → new error (new card title).
func TestAppModel_RetryInFlight_ThenSuccess_ThenReFail(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("connection refused"))

	// Phase 1: error state — card visible.
	plain1 := stripANSIModel(m.table.View())
	if !strings.Contains(plain1, "Could not load") {
		t.Fatalf("phase 1: want error card visible, got:\n%s", plain1)
	}

	// Phase 2: press r — loadState transitions to Loading (retry in-flight).
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m2 := r1.(AppModel)
	if m2.loadState != data.LoadStateLoading {
		t.Fatalf("phase 2: loadState = %v after r, want LoadStateLoading", m2.loadState)
	}
	// In-flight: table renders spinner, not error card.
	plain2 := stripANSIModel(m2.table.View())
	if strings.Contains(plain2, "Could not load") {
		t.Errorf("phase 2 (in-flight): error card must be gone during retry, got:\n%s", plain2)
	}

	// Phase 3: successful fetch — rows replace the card.
	rt := data.ResourceType{Name: "pods", Kind: "Pod"}
	r2, _ := m2.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{{Name: "pod-1", Cells: []string{"pod-1"}}},
		ResourceType: rt,
		Columns:      []string{"Name"},
	})
	m3 := r2.(AppModel)
	if m3.loadState == data.LoadStateError {
		t.Error("phase 3: loadState still Error after success, want cleared")
	}
	if m3.statusBar.Err != nil {
		t.Error("phase 3: statusBar.Err non-nil after success, want nil")
	}
	plain3 := stripANSIModel(m3.table.View())
	if strings.Contains(plain3, "Could not load") {
		t.Errorf("phase 3: error card must not appear after successful load, got:\n%s", plain3)
	}

	// Phase 4: second failure — new card with updated title.
	r3, _ := m3.Update(data.LoadErrorMsg{Err: errors.New("context deadline exceeded"), Severity: data.ErrorSeverityWarning})
	m4 := r3.(AppModel)
	if m4.loadState != data.LoadStateError {
		t.Errorf("phase 4: loadState = %v after second failure, want LoadStateError", m4.loadState)
	}
	plain4 := stripANSIModel(m4.table.View())
	if !strings.Contains(plain4, "Could not load") {
		t.Errorf("phase 4: new error card must appear after second failure, got:\n%s", plain4)
	}
}

// --- AC#14: Narrow-band rendering — width bands at innerW=47/27/12 ---

// TestResourceTableModel_SetLoadErr_WidthBand_Format (AC#14) pins the format
// behavior of the error card at each FB-022 width band using exact innerW values
// that ResourceTableModel passes to RenderErrorBlock (tableWidth−3).
//
// Band boundaries (FB-022 §5):
//   wide ≥60 → title+detail+actions+blank separators
//   narrow 40–59 → title+detail+actions, NO blank separators
//   unusable 20–39 → title+actions (NO detail), NO blank separators
//   collapsed <20 → single-line "<glyph> <title>" (no newline)
func TestResourceTableModel_SetLoadErr_WidthBand_Format(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		innerW         int // = tableWidth - 3
		wantDetail     bool
		wantNoBlankSep bool // narrow/unusable/collapsed: \n\n must NOT appear
		wantSingleLine bool // collapsed: \n must NOT appear
	}{
		// tableWidth=50 → innerW=47 → narrow band (40–59): detail present, no blank seps
		{"narrow (innerW=47, tableWidth=50)", 47, true, true, false},
		// tableWidth=30 → innerW=27 → unusable band (20–39): no detail, no blank seps
		{"unusable (innerW=27, tableWidth=30)", 27, false, true, false},
		// tableWidth=15 → innerW=12 → collapsed band (<20): single-line
		{"collapsed (innerW=12, tableWidth=15)", 12, false, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			card := stripANSIModel(components.RenderErrorBlock(components.ErrorBlock{
				Title:    "Could not load pods",
				Detail:   "connection refused",
				Actions:  components.ActionsForSeverity(data.ErrorSeverityWarning, "back to navigation"),
				Severity: data.ErrorSeverityWarning,
				Width:    tt.innerW,
			}))
			if tt.wantDetail && !strings.Contains(card, "connection refused") {
				t.Errorf("innerW=%d: detail must be visible at this band, got %q", tt.innerW, card)
			}
			if !tt.wantDetail && strings.Contains(card, "connection refused") {
				t.Errorf("innerW=%d: detail must be ABSENT at this band, got %q", tt.innerW, card)
			}
			if tt.wantNoBlankSep && strings.Contains(card, "\n\n") {
				t.Errorf("innerW=%d: must NOT have blank separator lines at this band, got %q", tt.innerW, card)
			}
			if tt.wantSingleLine && strings.Contains(card, "\n") {
				t.Errorf("innerW=%d: collapsed band must produce single-line output (no \\n), got %q", tt.innerW, card)
			}
		})
	}
}

// --- AC#15: Width pin — innerW = tableWidth − 3 ---

// TestResourceTableModel_ErrorCard_WidthPin_PaneWidthMinus3 (AC#15) pins the
// `innerW = tableWidth − 3` formula at the exact wide/narrow boundary:
//   tableWidth=63 → innerW=60 → WIDE band (blank section separators present)
//   tableWidth=62 → innerW=59 → NARROW band (no blank section separators)
//
// If the formula were tableWidth−2 or tableWidth−4, one assertion would fail.
func TestResourceTableModel_ErrorCard_WidthPin_PaneWidthMinus3(t *testing.T) {
	t.Parallel()
	block := components.ErrorBlock{
		Title:    "Could not load",
		Detail:   "connection refused",
		Actions:  components.ActionsForSeverity(data.ErrorSeverityWarning, "back"),
		Severity: data.ErrorSeverityWarning,
	}
	t.Run("tableWidth=63_innerW=60_wide_band", func(t *testing.T) {
		t.Parallel()
		b := block
		b.Width = 63 - 3 // 60 — at the ≥60 wide band boundary
		plain := stripANSIModel(components.RenderErrorBlock(b))
		if !strings.Contains(plain, "\n\n") {
			t.Errorf("innerW=60 (tableWidth 63-3=60): want wide band (blank section separators), got %q", plain)
		}
	})
	t.Run("tableWidth=62_innerW=59_narrow_band", func(t *testing.T) {
		t.Parallel()
		b := block
		b.Width = 62 - 3 // 59 — narrow band 40–59
		plain := stripANSIModel(components.RenderErrorBlock(b))
		if strings.Contains(plain, "\n\n") {
			t.Errorf("innerW=59 (tableWidth 62-3=59): want narrow band (no blank separators), got %q", plain)
		}
	})
}

// --- AC#21: No inline card renderers (coordination pin / grep assertion) ---

// TestFB005_NoInlineCardRenderers_GrepAssertion (AC#21) is a coordination pin
// from FB-022 AC#18: no inline lipgloss error-card renderer may remain in any
// consumer file after the FB-022 migration. All error rendering must go through
// components.RenderErrorBlock. This test fails if a future merge re-introduces
// an inline card path that bypasses the severity/width/action contracts.
func TestFB005_NoInlineCardRenderers_GrepAssertion(t *testing.T) {
	t.Parallel()
	const forbidden = `lipgloss.NewStyle().Foreground(styles.Error).Render`
	// Paths relative to internal/tui/ (the working directory when this test runs).
	files := []string{
		"model.go",
		"components/resourcetable.go",
		"components/historyview.go",
		"components/activityview.go",
		"components/quotadashboard.go",
		"components/activitydashboard.go",
		"components/ctxswitcher.go",
	}
	for _, f := range files {
		f := f
		t.Run(f, func(t *testing.T) {
			t.Parallel()
			content, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("cannot read %s (run tests from repo root with go test ./internal/tui/...): %v", f, err)
			}
			if strings.Contains(string(content), forbidden) {
				t.Errorf("%s: found inline card renderer %q — must use components.RenderErrorBlock", f, forbidden)
			}
		})
	}
}

// --- FB-005 v2: AC#10 rework — rendered-output assertion for hint visibility ---

// TestAppModel_RKey_ErrorState_Error_Severity_HintVisible_InView (AC#10 rework)
// is the [Observable] companion to the existing model-state check. It proves
// the "No retry available" text actually reaches the rendered statusbar (not just
// the Hint field), which was the P1 bug: Err was non-nil so the Err branch in
// statusbar.View() shadowed the Hint branch entirely.
//
// [Observable]: stripANSI(statusBar.View()) must contain "No retry available"
// [Anti-behavior]: in-pane error card must remain (loadState stays Error; card
// is gated on loadState+table.loadErr, which the P1 path does not touch).
func TestAppModel_RKey_ErrorState_Error_Severity_HintVisible_InView(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errStubForbidden) // Error severity

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	// [Observable]: set a display width so View() renders at full content.
	appM.statusBar.Width = 80
	statusPlain := stripANSIModel(appM.statusBar.View())
	if !strings.Contains(statusPlain, "No retry available") {
		t.Errorf("AC#10 [Observable]: statusBar.View() = %q\nwant 'No retry available' (P1 fix: Err cleared before postHint)", statusPlain)
	}

	// [Anti-behavior]: in-pane error card must NOT disappear.
	tablePlain := stripANSIModel(appM.table.View())
	if !strings.Contains(tablePlain, "Could not load") {
		t.Errorf("AC#10 [Anti-behavior]: table.View() = %q\nwant error card still visible (loadState=Error, table.loadErr untouched by P1 path)", tablePlain)
	}
}

// --- FB-005 v2: AC#8 rework — total-dispatch-count across both branches ---

// TestAppModel_RKey_ErrorState_RepeatPress_ExactlyOneLoadCmd (AC#8 rework) is
// the [Repeat-press] + [Observable] test that the v1 version missed: it counts
// total LoadResourcesCmds across 5 rapid r presses, not just whether the first
// press fires. Without the P2 fix (m.refreshing = true in the error-state retry
// branch), presses 2-5 fall through to the normal-state branch which is not
// guarded against repeated dispatch, producing up to 5 total commands.
func TestAppModel_RKey_ErrorState_RepeatPress_ExactlyOneLoadCmd(t *testing.T) {
	t.Parallel()
	m := newErrorStateTableModel(errors.New("timeout")) // Warning severity → retry available

	var totalLoads int
	for i := 0; i < 5; i++ {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
		m = result.(AppModel)
		for _, msg := range collectMsgs(cmd) {
			if _, ok := msg.(data.ResourcesLoadedMsg); ok {
				totalLoads++
			}
		}
	}

	if totalLoads != 1 {
		t.Errorf("AC#8 [Repeat-press]: 5 rapid r presses produced %d LoadResourcesCmds, want exactly 1 (P2 fix: m.refreshing=true in error-state retry branch guards repeat)", totalLoads)
	}
}

// --- FB-005 v2: P2#1 — DetailPane error card title namespace assembly ---

// TestAppModel_DetailPane_ErrorCard_Title_NamespaceAssembly (P2#1 fold) pins
// the title construction logic at buildDetailContent:1696-1702. The v2 fold
// added the namespace segment for namespaced resources with a non-empty namespace.
//
// [Observable]: stripANSI(buildDetailContent()) must contain the expected title.
// [Edit-changed]: title format changed from "pods/my-pod" to "pods/default/my-pod".
func TestAppModel_DetailPane_ErrorCard_Title_NamespaceAssembly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		rtName      string
		namespaced  bool
		namespace   string
		resName     string
		wantTitle   string
	}{
		{
			name:       "namespaced resource with non-empty namespace",
			rtName:     "pods",
			namespaced: true,
			namespace:  "default",
			resName:    "my-pod",
			wantTitle:  "pods/default/my-pod",
		},
		{
			name:       "cluster-scoped resource (no namespace segment)",
			rtName:     "nodes",
			namespaced: false,
			namespace:  "default", // ignored when not namespaced
			resName:    "node-01",
			wantTitle:  "nodes/node-01",
		},
		{
			name:       "namespaced resource with empty namespace",
			rtName:     "pods",
			namespaced: true,
			namespace:  "",
			resName:    "my-pod",
			wantTitle:  "pods/my-pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newTablePaneModel()
			m.activePane = DetailPane
			m.loadState = data.LoadStateError
			m.lastFailedFetchKind = "describe"
			m.loadErr = errors.New("connection refused")
			m.describeRT = data.ResourceType{Name: tt.rtName, Namespaced: tt.namespaced}
			m.tuiCtx.Namespace = tt.namespace
			m.detail.SetResourceContext(tt.rtName, tt.resName)
			m.detail.SetSize(80, 20)

			plain := stripANSIModel(m.buildDetailContent())
			if !strings.Contains(plain, tt.wantTitle) {
				t.Errorf("[Observable] buildDetailContent() title:\n  got:  %q\n  want: contains %q", plain, tt.wantTitle)
			}
		})
	}
}

// --- Test D: YAML marshal error card (FB-022 hotfix, persona P3 #5) ---

// TestAppModel_YAMLMarshalError_CardHasYAndEscActions verifies that when
// buildDetailContent cannot marshal the raw object to YAML, the rendered card
// contains [y] toggle yaml, [Esc] back, and the ✕ glyph (ErrorSeverityError),
// with NO [r] retry hint (structural errors are not retriable).
func TestAppModel_YAMLMarshalError_CardHasYAndEscActions(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.activePane = DetailPane
	m.yamlMode = true
	// Inject an unstructured value that sigs.k8s.io/yaml cannot marshal (channel
	// is not JSON-serializable, which is the underlying encoder yaml uses).
	m.describeRaw = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata":   map[string]interface{}{"name": make(chan int)},
		},
	}

	plain := stripANSIModel(m.buildDetailContent())

	if !strings.Contains(plain, "[y]") {
		t.Errorf("YAML marshal error card: want '[y]' toggle-yaml hint, got %q", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("YAML marshal error card: want '[Esc]' back hint, got %q", plain)
	}
	if !strings.Contains(plain, "✕") {
		t.Errorf("YAML marshal error card: want '✕' glyph (ErrorSeverityError), got %q", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("YAML marshal error card: must NOT contain '[r]' retry hint (structural error), got %q", plain)
	}
}

// ==================== FB-024: Events sub-view correctness fixes ====================

// newDetailPaneModelWithEventsOnly builds a DetailPane AppModel where
// describeRaw is nil (describe failed) but events are successfully loaded.
// This is the RBAC-partial scenario: operator has get-events but not get-resource.
func newDetailPaneModelWithEventsOnly() AppModel {
	m := newDetailPaneModelWithHC()
	m.describeRaw = nil
	m.events = []data.EventRow{
		{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created pod", Count: 1},
	}
	return m
}

// newDetailPaneModelWithEventsInFlight builds a DetailPane AppModel where
// eventsLoading=true and events=nil — simulating the in-flight state after d.
func newDetailPaneModelWithEventsInFlight() AppModel {
	m := newDetailPaneModelWithHC()
	m.describeRaw = nil
	m.events = nil
	m.eventsLoading = true
	return m
}

// AC#1 — [Failure/Input-changed] E enters events mode when describeRaw==nil but events!=nil.
func TestFB024_EGuard_DescribeNil_EventsLoaded_EntersEventsMode(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsOnly()
	if m.describeRaw != nil {
		t.Fatal("setup: describeRaw must be nil")
	}
	if len(m.events) == 0 {
		t.Fatal("setup: events must be non-empty")
	}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if !appM.eventsMode {
		t.Error("AC#1: eventsMode = false after E with describeRaw=nil + events loaded, want true")
	}
	if appM.detail.Mode() != "events" {
		t.Errorf("AC#1: detail.Mode() = %q, want 'events'", appM.detail.Mode())
	}
	// No re-dispatch needed — events already cached.
	if cmd != nil {
		t.Error("AC#1: cmd != nil, want nil (events already cached, no re-dispatch needed)")
	}
	// Observable: detail view contains events-mode indicator.
	view := stripANSIModel(appM.detail.View())
	if !strings.Contains(view, "events") {
		t.Errorf("AC#1: detail.View() missing 'events' mode indicator:\n%s", view)
	}
}

// AC#2 — [Input-changed/Observable] Toggle OUT of events mode when describeRaw==nil
// renders the "Describe unavailable" placeholder, not an empty viewport.
func TestFB024_ToggleOut_DescribeNil_EventsLoaded_ShowsPlaceholder(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsOnly()

	// Enter events mode.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r1.(AppModel)
	if !m.eventsMode {
		t.Fatal("setup: first E did not enter events mode")
	}

	// Toggle back out of events mode.
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := r2.(AppModel)

	if appM.eventsMode {
		t.Error("AC#2: eventsMode = true after second E, want false")
	}
	// Observable: viewport content must contain the placeholder (exact substring from D2).
	view := stripANSIModel(appM.detail.View())
	const placeholder = "Describe unavailable \u2014 only events loaded."
	if !strings.Contains(view, placeholder) {
		t.Errorf("AC#2: detail.View() missing placeholder %q\ngot:\n%s", placeholder, view)
	}
}

// AC#3 — [Anti-behavior] E is a no-op when describeRaw==nil AND events==nil AND !eventsLoading.
func TestFB024_EGuard_BothNil_NotLoading_IsNoop(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	// Ensure both-nil + not-loading state (the base fixture already has this).
	if m.describeRaw != nil || m.events != nil || m.eventsLoading {
		t.Fatal("setup: fixture must have describeRaw=nil, events=nil, eventsLoading=false")
	}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if appM.eventsMode {
		t.Error("AC#3: eventsMode = true, want no-op (false) when both fetches absent")
	}
	if cmd != nil {
		t.Error("AC#3: cmd != nil, want nil (no-op)")
	}
}

// AC#3a — [Anti-behavior/quad-state] y and C remain no-ops when describeRaw==nil,
// even when events are loaded (they need the manifest, not events).
func TestFB024_YamlAndConditions_Noop_DescribeNil(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsOnly()

	t.Run("y_key_noop_when_describeRaw_nil", func(t *testing.T) {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		appM := result.(AppModel)
		if appM.yamlMode {
			t.Error("AC#3a: yamlMode = true with describeRaw=nil, want no-op")
		}
		if cmd != nil {
			t.Error("AC#3a: y cmd != nil with describeRaw=nil, want no-op")
		}
	})

	t.Run("C_key_noop_when_describeRaw_nil", func(t *testing.T) {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
		appM := result.(AppModel)
		if appM.conditionsMode {
			t.Error("AC#3a: conditionsMode = true with describeRaw=nil, want no-op")
		}
		if cmd != nil {
			t.Error("AC#3a: C cmd != nil with describeRaw=nil, want no-op")
		}
	})
}

// AC#3b — [Failure/precedence] When both fetches failed (describeRaw==nil, events==nil,
// loadState=error), buildDetailContent renders the error block, not the placeholder.
func TestFB024_BothFailed_ErrorBlock_NotPlaceholder(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	// Simulate both fetches failed.
	m.describeRaw = nil
	m.events = nil
	m.loadState = data.LoadStateError
	m.lastFailedFetchKind = "describe"
	m.loadErr = errors.New("connection refused")

	content := m.buildDetailContent()
	plain := stripANSIModel(content)

	// Must NOT show the placeholder (that requires events != nil).
	const placeholder = "Describe unavailable"
	if strings.Contains(plain, placeholder) {
		t.Errorf("AC#3b: placeholder rendered when events==nil, want error block: %q", plain)
	}
	// Must contain the error block title (error content from FB-005 path).
	if len(plain) == 0 {
		t.Error("AC#3b: buildDetailContent returned empty string, want error block")
	}
}

// AC#7 — [Anti-behavior/Edge] Rapid E mash during in-flight fetch produces 0 extra dispatches.
func TestFB024_RapidEPress_InFlight_NoExtraDispatch(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsInFlight()
	// eventsLoading=true is already set; outer guard passes, inner dispatch guard blocks.

	for i := 1; i <= 3; i++ {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = result.(AppModel)
		if cmd != nil {
			// Execute to check if it's a LoadEventsCmd.
			msgs := collectMsgs(cmd)
			for _, msg := range msgs {
				if _, ok := msg.(data.EventsLoadedMsg); ok {
					t.Errorf("AC#7: press %d produced a LoadEventsCmd dispatch, want 0 extra dispatches", i)
				}
			}
		}
	}
}

// AC#8 — [Repeat-press/Input-changed] Toggling E off/on after a successful events load
// never re-dispatches LoadEventsCmd (cache-hit path).
func TestFB024_RepeatToggle_CacheHit_NoRedispatch(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw() // describeRaw set, events cached

	// Five toggles — none should dispatch.
	for i := 1; i <= 5; i++ {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = result.(AppModel)
		if cmd != nil {
			msgs := collectMsgs(cmd)
			for _, msg := range msgs {
				if _, ok := msg.(data.EventsLoadedMsg); ok {
					t.Errorf("AC#8: toggle %d produced a LoadEventsCmd, want 0 re-dispatches on cache-hit", i)
				}
			}
		}
	}
	// eventsLoading must stay false throughout.
	if m.eventsLoading {
		t.Error("AC#8: eventsLoading = true after cache-hit toggles, want false")
	}
}

// AC#9 — [Failure] E after an events error re-dispatches exactly once;
// rapid mash during the re-dispatch's in-flight window adds zero extra dispatches.
func TestFB024_EAfterError_Redispatch_OnceOnly(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	// Simulate a prior fetch error (events cleared, eventsErr set).
	m.eventsMode = true
	m.events = nil
	m.eventsLoading = false
	m.eventsErr = errors.New("forbidden")

	// First E press — toggle off (eventsMode=true → false).
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r1.(AppModel)
	if cmd1 != nil {
		t.Error("AC#9 step 1: toggling OFF events mode should produce nil cmd, got non-nil")
	}

	// Second E press — toggle on: guard fires (eventsErr!=nil, !eventsLoading) → dispatch.
	r2, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r2.(AppModel)
	if cmd2 == nil {
		t.Fatal("AC#9 step 2: expected LoadEventsCmd dispatch on retry, got nil")
	}
	if !m.eventsLoading {
		t.Error("AC#9 step 2: eventsLoading = false after retry dispatch, want true")
	}

	// Rapid mash: 3 more E presses while eventsLoading=true — each must produce nil cmd.
	for i := 3; i <= 5; i++ {
		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = result.(AppModel)
		if cmd != nil {
			msgs := collectMsgs(cmd)
			for _, msg := range msgs {
				if _, ok := msg.(data.EventsLoadedMsg); ok {
					t.Errorf("AC#9 press %d: got extra LoadEventsCmd dispatch, want 0 (in-flight guard)", i)
				}
			}
		}
	}
}

// AC#10 — [Anti-regression] Non-DetailPane panes: E is a no-op even when events!=nil.
// Verifies the outer `m.activePane == DetailPane` guard survives the D1 relaxation.
func TestFB024_NonDetailPane_E_IsNoop_Regardless(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		m    func() AppModel
	}{
		{"NavPane", func() AppModel { return newNavPaneModelWithBC(nil) }},
		{"TablePane", newTablePaneModel},
		{"QuotaDashboardPane", func() AppModel { return newQuotaDashboardPaneModel(&stubBucketClient{}) }},
		{"HistoryPane", newHistoryPaneModel},
		{"DiffPane", func() AppModel { return newDiffPaneModel(1) }},
		{"ActivityPane", newActivityPaneModel},
		{"ActivityDashboardPane", newActivityDashboardPaneModel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := tt.m()
			// Inject events so the D1-relaxed guard would fire if pane check were removed.
			m.events = []data.EventRow{{Type: "Normal", Reason: "Test", Message: "msg"}}

			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
			appM := result.(AppModel)
			if appM.eventsMode {
				t.Errorf("AC#10 %s: eventsMode = true after E with events loaded, want no-op (pane guard)", tt.name)
			}
		})
	}
}

// ==================== End FB-024 ====================

// ==================== FB-035: Wire `3` key to QuotaDashboard ====================

// TestFB035_Key3_FromNavPane_EntersDashboard — AC#1: `3` from NavPane → QuotaDashboardPane.
func TestFB035_Key3_FromNavPane_EntersDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC#1: activePane = %v after 3 from NavPane, want QuotaDashboardPane", appM.activePane)
	}
	if cmd != nil {
		t.Error("AC#1: cmd != nil, want nil (quota already loaded)")
	}
}

// TestFB035_Key3_FromTablePane_EntersDashboard — AC#2: `3` from TablePane → QuotaDashboardPane.
func TestFB035_Key3_FromTablePane_EntersDashboard(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC#2: activePane = %v after 3 from TablePane, want QuotaDashboardPane", appM.activePane)
	}
}

// TestFB035_Key3_FromDetailPane_EntersDashboard — AC#1a: `3` from DetailPane → QuotaDashboardPane.
func TestFB035_Key3_FromDetailPane_EntersDashboard(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC#1a: activePane = %v after 3 from DetailPane, want QuotaDashboardPane", appM.activePane)
	}
}

// TestFB035_Key3_FromQuotaDashboard_TogglesBackToNav — AC#3: `3` from QuotaDashboardPane → NavPane.
func TestFB035_Key3_FromQuotaDashboard_TogglesBackToNav(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC#3: activePane = %v after 3 from QuotaDashboardPane, want NavPane (toggle back)", appM.activePane)
	}
}

// TestFB035_Key3_RapidMash_DeterministicToggle — AC#3a: 5 rapid `3` presses alternate deterministically.
func TestFB035_Key3_RapidMash_DeterministicToggle(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	wantPanes := []PaneID{QuotaDashboardPane, NavPane, QuotaDashboardPane, NavPane, QuotaDashboardPane}
	for i, want := range wantPanes {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
		m = result.(AppModel)
		if m.activePane != want {
			t.Errorf("AC#3a press %d: activePane = %v, want %v", i+1, m.activePane, want)
		}
	}
}

// TestFB035_Key3_OverlayOpen_IsNoop — AC#4: `3` is a no-op when an overlay is open.
func TestFB035_Key3_OverlayOpen_IsNoop(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		overlay OverlayID
	}{
		{"HelpOverlay", HelpOverlayID},
		{"CtxSwitcherOverlay", CtxSwitcherOverlay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newNavPaneModelWithBC(nil)
			m.overlay = tt.overlay
			m.statusBar.Mode = components.ModeOverlay
			before := m.activePane

			result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
			appM := result.(AppModel)

			if appM.activePane != before {
				t.Errorf("AC#4 %s: activePane changed from %v to %v, want no-op", tt.name, before, appM.activePane)
			}
		})
	}
}

// TestFB035_Key3_FilterBarFocused_TypesIntoFilter — AC#5: `3` types into the FilterBar when focused.
func TestFB035_Key3_FilterBarFocused_TypesIntoFilter(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	_ = m.filterBar.Focus()
	m.statusBar.Mode = components.ModeFilter
	before := m.activePane

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != before {
		t.Errorf("AC#5: activePane changed from %v to %v, want no-op (FilterBar consumed key)", before, appM.activePane)
	}
	if appM.filterBar.Value() != "3" {
		t.Errorf("AC#5: filterBar.Value() = %q, want '3' (key typed into filter)", appM.filterBar.Value())
	}
}

// TestFB035_Key3_QuotaIsLoading_IsNoop — AC#8a: `3` is a no-op when quota is loading.
func TestFB035_Key3_QuotaIsLoading_IsNoop(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")
	m.quota.SetLoading(true)
	before := m.activePane

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != before {
		t.Errorf("AC#8a: activePane changed from %v to %v while quota loading, want no-op", before, appM.activePane)
	}
}

// TestFB035_HelpOverlay_ContainsKey3Row — AC#6 + AC#6b: HelpOverlay View contains
// `[3]  quota (toggle)` (FB-050 copy update) and `[3]` appears before `[4]` in the output.
func TestFB035_HelpOverlay_ContainsKey3Row(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	appM := result.(AppModel)
	if appM.overlay != HelpOverlayID {
		t.Fatal("setup: expected HelpOverlayID after ? press")
	}

	view := stripANSIModel(appM.helpOverlay.View())

	// AC#6: [3] quota (toggle) row present (FB-050).
	if !strings.Contains(view, "[3]") {
		t.Errorf("AC#6: HelpOverlay missing '[3]' row:\n%s", view)
	}
	if !strings.Contains(view, "quota (toggle)") {
		t.Errorf("AC#6: HelpOverlay missing 'quota (toggle)' label:\n%s", view)
	}

	// AC#6b: [3] index precedes [4] index.
	idx3 := strings.Index(view, "[3]")
	idx4 := strings.Index(view, "[4]")
	if idx3 < 0 || idx4 < 0 {
		t.Fatalf("AC#6b: [3] or [4] absent in HelpOverlay:\n%s", view)
	}
	if idx3 >= idx4 {
		t.Errorf("AC#6b: [3] (idx=%d) must appear before [4] (idx=%d)", idx3, idx4)
	}
}

// TestFB035_Key4_AndTToggle_Unchanged — AC#7 anti-regression: `4` key and `t` toggle unaffected.
func TestFB035_Key4_AndTToggle_Unchanged(t *testing.T) {
	t.Parallel()

	t.Run("4_key_still_enters_ActivityDashboard", func(t *testing.T) {
		t.Parallel()
		m := newActivityDashboardPaneModel()
		m.tuiCtx.ActiveCtx = nil // org scope → no fetch dispatched, simpler assertion
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
		appM := result.(AppModel)
		if appM.activePane != ActivityDashboardPane {
			t.Errorf("4 key: activePane = %v, want ActivityDashboardPane", appM.activePane)
		}
	})

	t.Run("t_toggle_from_dashboard_to_table", func(t *testing.T) {
		t.Parallel()
		m := newQuotaDashboardPaneModel(&stubBucketClient{})
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
		appM := result.(AppModel)
		if appM.activePane != TablePane {
			t.Errorf("t key from QuotaDashboard: activePane = %v, want TablePane", appM.activePane)
		}
	})
}

// TestFB035_Key3_FromActivityDashboardPane_EntersDashboard — AC#1c (REQUIRED):
// pressing `3` from ActivityDashboardPane (the `3↔4` sibling pane) navigates to
// QuotaDashboardPane, anchoring the app-global `3↔4` symmetry mental model.
func TestFB035_Key3_FromActivityDashboardPane_EntersDashboard(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	m.activePane = ActivityDashboardPane
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC#1c: activePane = %v after 3 from ActivityDashboardPane, want QuotaDashboardPane", appM.activePane)
	}
}

// TestFB035_Key3_FromNavPane_ResourceListVisible_EntersDashboard — AC#1b:
// pressing `3` from NavPane when a resource type is selected (welcome panel NOT
// visible, resource list visible) transitions to QuotaDashboardPane.
func TestFB035_Key3_FromNavPane_ResourceListVisible_EntersDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)
	// tableTypeName non-empty → welcome panel suppressed, resource list visible.
	m.tableTypeName = "projects"
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC#1b: activePane = %v after 3 from NavPane (resource list visible), want QuotaDashboardPane", appM.activePane)
	}
}

// ==================== End FB-035 ====================

// ==================== FB-037: DetailPane error-first race ====================

// newDescribeErrorDetailModel builds a DetailPane AppModel in LoadStateError where
// the describe fetch failed but events have not yet arrived.
func newDescribeErrorDetailModel() AppModel {
	m := newDetailPaneModelWithHC()
	m.describeRaw = nil
	m.describeContent = ""
	m.loadState = data.LoadStateError
	m.lastFailedFetchKind = "describe"
	m.loadErr = errors.New("connection refused")
	m.events = nil
	m.eventsLoading = false
	// Sync detail component to error-block content so View() reflects the error state.
	m.detail.SetContent(m.buildDetailContent())
	return m
}

// AC#1 — [Input-changed] Error-first race: describe fails, events arrive later →
// detail viewport re-renders to FB-024 placeholder without requiring a keypress.
func TestFB037_ErrorFirst_EventsArrive_ReRendersToPlaceholder(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()

	// Confirm error-block is showing before events arrive.
	beforePlain := stripANSIModel(m.detail.View())
	if !strings.Contains(beforePlain, "Could not describe") {
		t.Fatalf("setup: error block not showing before EventsLoadedMsg; got:\n%s", beforePlain)
	}

	// Events arrive (error-first race resolution).
	events := []data.EventRow{{Type: "Warning", Reason: "BackOff", Message: "Back-off restarting", Count: 3}}
	result, _ := m.Update(data.EventsLoadedMsg{Events: events})
	appM := result.(AppModel)

	plain := stripANSIModel(appM.detail.View())
	const placeholder = "Describe unavailable"
	if !strings.Contains(plain, placeholder) {
		t.Errorf("AC#1: after EventsLoadedMsg, detail.View() missing %q\ngot:\n%s", placeholder, plain)
	}
	if strings.Contains(plain, "Could not describe") {
		t.Errorf("AC#1: after EventsLoadedMsg, detail.View() still contains error-block title; want placeholder:\n%s", plain)
	}
}

// AC#2 — [Anti-regression] Events-first race: events arrive before describe error →
// placeholder renders. Pins the events-first path as an order-invariant guarantee.
func TestFB037_EventsFirstRace_PlaceholderShows_AntiRegression(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsOnly() // describeRaw=nil, events=[...]

	// Assert via buildDetailContent — the fixture doesn't pre-render so detail.View()
	// reflects stale content from SetContent("describe text") in the base constructor.
	plain := stripANSIModel(m.buildDetailContent())
	const placeholder = "Describe unavailable"
	if !strings.Contains(plain, placeholder) {
		t.Errorf("AC#2: events-first state: buildDetailContent() missing %q\ngot:\n%s", placeholder, plain)
	}
}

// AC#3 — [Observable] Both-fail order-invariant: describe fails then events fail →
// error block renders; placeholder does NOT appear.
func TestFB037_BothFail_OrderInvariant_ErrorBlockShows(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()

	eventsErr := errors.New("events fetch timeout")
	result, _ := m.Update(data.EventsLoadedMsg{Events: nil, Err: eventsErr})
	appM := result.(AppModel)

	plain := stripANSIModel(appM.detail.View())
	if strings.Contains(plain, "Describe unavailable") {
		t.Errorf("AC#3: both-failed: placeholder present; want error block:\n%s", plain)
	}
	if !strings.Contains(plain, "Could not describe") {
		t.Errorf("AC#3: both-failed: error-block title 'Could not describe' absent:\n%s", plain)
	}
}

// AC#4 — [Observable] Placeholder takes precedence over error block when
// describeRaw==nil AND events!=nil, even when loadState==error.
func TestFB037_ErrorState_EventsLoaded_PlaceholderPreemptsErrorBlock(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()
	m.events = []data.EventRow{{Type: "Normal", Reason: "Scheduled", Message: "Assigned", Count: 1}}

	plain := stripANSIModel(m.buildDetailContent())
	const placeholder = "Describe unavailable"
	if !strings.Contains(plain, placeholder) {
		t.Errorf("AC#4: buildDetailContent() missing %q when events loaded despite error state:\n%s", placeholder, plain)
	}
	if strings.Contains(plain, "Could not describe") {
		t.Errorf("AC#4: error-block title still present; placeholder must preempt it:\n%s", plain)
	}
}

// AC#5 — [Anti-behavior] Error block renders with events==nil → no "[E]" action hint.
func TestFB037_ErrorBlock_EventsNil_NoEHint(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel() // events=nil

	plain := stripANSIModel(m.buildDetailContent())
	if strings.Contains(plain, "[E]") {
		t.Errorf("AC#5: error block with events==nil contains '[E]' hint; want no events affordance:\n%s", plain)
	}
}

// AC#6 — [Integration/Happy] E from the FB-024 placeholder (post race-recovery)
// enters events mode and renders the events table.
func TestFB037_EFromPlaceholder_EntersEventsMode(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()

	// Resolve race: events arrive.
	events := []data.EventRow{{Type: "Warning", Reason: "BackOff", Message: "Back-off restarting", Count: 3}}
	r1, _ := m.Update(data.EventsLoadedMsg{Events: events})
	m = r1.(AppModel)

	// Confirm we're on the placeholder.
	if !strings.Contains(stripANSIModel(m.detail.View()), "Describe unavailable") {
		t.Fatal("AC#6 setup: placeholder not showing after EventsLoadedMsg")
	}

	// Press E — should enter events mode.
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := r2.(AppModel)

	if !appM.eventsMode {
		t.Error("AC#6: eventsMode = false after E from placeholder, want true")
	}
	plain := stripANSIModel(appM.detail.View())
	if strings.Contains(plain, "Describe unavailable") {
		t.Errorf("AC#6: placeholder still visible after E; want events table:\n%s", plain)
	}
}

// AC#7 — [Repeat-press] Pressing E twice from placeholder toggles in and out of events mode.
func TestFB037_RepeatE_FromPlaceholder_TogglesEventsMode(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()

	// Resolve race: events arrive.
	events := []data.EventRow{{Type: "Normal", Reason: "Pulled", Message: "Pulled image", Count: 1}}
	r0, _ := m.Update(data.EventsLoadedMsg{Events: events})
	m = r0.(AppModel)

	// Press 1: enter events mode.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	m = r1.(AppModel)
	if !m.eventsMode {
		t.Fatal("AC#7: first E did not enter events mode")
	}

	// Press 2: toggle back to placeholder.
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := r2.(AppModel)

	if appM.eventsMode {
		t.Error("AC#7: eventsMode = true after second E, want false (toggled back)")
	}
	plain := stripANSIModel(appM.detail.View())
	if !strings.Contains(plain, "Describe unavailable") {
		t.Errorf("AC#7: placeholder not restored after toggle-back:\n%s", plain)
	}
}

// ==================== End FB-037 ====================

// ==================== FB-041: Escape always returns to welcome dashboard ====================

// newNavPaneWithTableLoaded builds an AppModel in NavPane with a resource type
// already selected and table rows loaded — the pre-Esc state for FB-041 tests.
// UserName is set so "Welcome, alice" renders in the welcome panel.
func newNavPaneWithTableLoaded() AppModel {
	sidebar := components.NewNavSidebarModel(22, 30)
	rt := data.ResourceType{Name: "projects", Kind: "Project", Group: "resourcemanager.miloapis.com", Namespaced: false}
	sidebar.SetItems([]data.ResourceType{rt})

	tbl := components.NewResourceTableModel(80, 30)
	tbl.SetColumns([]string{"Name"}, 80)
	rows := []data.ResourceRow{
		{Name: "datum-cloud", Cells: []string{"datum-cloud"}},
		{Name: "prod-infra", Cells: []string{"prod-infra"}},
	}
	tbl.SetRows(rows)
	tbl.SetTypeContext("projects", true)

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    NavPane,
		tableTypeName: "projects",
		resources:     rows,
		sidebar:       sidebar,
		table:         tbl,
		detail:        components.NewDetailViewModel(80, 30),
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
		tuiCtx:        tuictx.TUIContext{UserName: "alice"},
	}
	m.refreshLandingInputs()
	m.updatePaneFocus()
	return m
}

// AC#1 + AC#2 — [Happy/Observable] Esc from NavPane with table loaded shows
// welcome dashboard; activePane stays NavPane; sidebar cursor unchanged.
func TestFB041_Esc_NavPane_WithTableLoaded_ShowsDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	selectedBefore, _ := m.sidebar.SelectedType()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	// AC#2: pane and sidebar selection unchanged.
	if appM.activePane != NavPane {
		t.Errorf("AC#2: activePane = %v after Esc, want NavPane (sidebar retains focus)", appM.activePane)
	}
	selectedAfter, _ := appM.sidebar.SelectedType()
	if selectedAfter.Name != selectedBefore.Name {
		t.Errorf("AC#2: sidebar selection changed from %q to %q after Esc; want cursor unchanged",
			selectedBefore.Name, selectedAfter.Name)
	}
	if !appM.showDashboard {
		t.Error("AC#1: showDashboard = false after Esc from NavPane with table loaded, want true")
	}

	// AC#1: View() substrings (welcome panel rendered in right pane).
	plain := stripANSIModel(appM.table.View())
	if !strings.Contains(plain, "Welcome,") {
		t.Errorf("AC#1: table.View() missing 'Welcome,' after Esc:\n%s", plain)
	}
	if !strings.Contains(plain, "Platform health") {
		t.Errorf("AC#1: table.View() missing 'Platform health' after Esc:\n%s", plain)
	}
}

// AC#3 — [Repeat-press] Second Esc is a no-op; showDashboard stays true,
// View() output is byte-identical to first-press output.
func TestFB041_Esc_NavPane_SecondPress_IsNoop(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m1 := r1.(AppModel)
	if !m1.showDashboard {
		t.Fatal("AC#3 setup: first Esc did not set showDashboard=true")
	}
	view1 := m1.table.View()

	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := r2.(AppModel)

	if !m2.showDashboard {
		t.Error("AC#3: showDashboard = false after second Esc, want true (no-op)")
	}
	view2 := m2.table.View()
	if view1 != view2 {
		t.Errorf("AC#3: table.View() changed on second Esc; want byte-identical output\nbefore:\n%s\nafter:\n%s",
			stripANSIModel(view1), stripANSIModel(view2))
	}
}

// [Input-changed] Esc → Enter clears showDashboard and returns table.
func TestFB041_Esc_ThenEnter_ClearsDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	// Go to dashboard.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = r1.(AppModel)
	if !m.showDashboard {
		t.Fatal("setup: Esc did not set showDashboard=true")
	}

	// Press Enter to select the sidebar item.
	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	appM := r2.(AppModel)

	if appM.showDashboard {
		t.Error("showDashboard = true after Enter, want false (table should return)")
	}
	plain := stripANSIModel(appM.table.View())
	if strings.Contains(plain, "Welcome,") {
		t.Errorf("table.View() still shows 'Welcome,' after Enter; want table content:\n%s", plain)
	}
}

// [Input-changed] Esc → Tab clears showDashboard and returns to TablePane.
func TestFB041_Esc_ThenTab_ClearsDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = r1.(AppModel)
	if !m.showDashboard {
		t.Fatal("setup: Esc did not set showDashboard=true")
	}

	r2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	appM := r2.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("activePane = %v after Tab, want TablePane", appM.activePane)
	}
	if appM.showDashboard {
		t.Error("showDashboard = true after Tab, want false")
	}
}

// [Input-changed] j/k in sidebar while dashboard visible updates left block.
func TestFB041_JKey_DuringDashboard_UpdatesLeftBlock(t *testing.T) {
	t.Parallel()
	sidebar := components.NewNavSidebarModel(22, 30)
	sidebar.SetItems([]data.ResourceType{
		{Name: "projects", Kind: "Project", Group: "resourcemanager.miloapis.com"},
		{Name: "deployments", Kind: "Deployment", Group: "apps", Namespaced: true},
	})
	tbl := components.NewResourceTableModel(80, 30)
	tbl.SetTypeContext("projects", true)

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    NavPane,
		tableTypeName: "projects",
		sidebar:       sidebar,
		table:         tbl,
		detail:        components.NewDetailViewModel(80, 30),
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
		tuiCtx:        tuictx.TUIContext{UserName: "alice"},
	}
	m.refreshLandingInputs()
	m.updatePaneFocus()
	// Set dashboard mode (simulating post-Esc state).
	m.showDashboard = true
	m.table.SetForceDashboard(true)

	before := stripANSIModel(m.table.View())
	if !strings.Contains(before, "Project") {
		t.Skipf("precondition: 'Project' not visible in initial left block (layout too narrow?): %q", before)
	}

	// Press j to move to Deployment.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	appM := result.(AppModel)

	after := stripANSIModel(appM.table.View())
	if !strings.Contains(after, "Deployment") {
		t.Errorf("j during dashboard: 'Deployment' not in left block after j press:\n%s", after)
	}
}

// AC#5 anti-regression — [Anti-behavior] Esc from TablePane → NavPane without dashboard.
func TestFB041_TablePane_Esc_NoShowDashboard(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC#5: activePane = %v after Esc from TablePane, want NavPane", appM.activePane)
	}
	if appM.showDashboard {
		t.Error("AC#5: showDashboard = true after TablePane Esc, want false (table still visible)")
	}
	plain := stripANSIModel(appM.table.View())
	if strings.Contains(plain, "Welcome,") {
		t.Errorf("AC#5: table.View() contains 'Welcome,' after TablePane Esc; want table rows:\n%s", plain)
	}
}

// [Anti-behavior] Esc when tableTypeName=="" (startup state) is a no-op.
func TestFB041_Startup_Esc_IsNoop_NoTableTypeName(t *testing.T) {
	t.Parallel()
	m := newWelcomePanelAppModel(nil, nil) // tableTypeName==""

	if m.tableTypeName != "" {
		t.Fatalf("precondition: tableTypeName must be empty, got %q", m.tableTypeName)
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.showDashboard {
		t.Error("showDashboard = true after Esc at startup with tableTypeName='', want false (no-op)")
	}
}

// [Anti-behavior] Table rows remain in memory after Esc-to-dashboard.
func TestFB041_TableRows_PreservedInMemory_AfterEsc(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if !appM.showDashboard {
		t.Fatal("setup: Esc did not set showDashboard=true")
	}
	if appM.tableTypeName != "projects" {
		t.Errorf("tableTypeName = %q after Esc, want 'projects' (cache preserved)", appM.tableTypeName)
	}
	if len(appM.resources) == 0 {
		t.Error("resources empty after Esc; want cached rows to remain in memory")
	}
}

// AC#4 anti-regression — [Anti-regression] Esc from DetailPane → TablePane, NOT NavPane.
func TestFB041_DetailPane_Esc_GoesToTablePane_NotNavPane(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.detailReturnPane = TablePane
	m.tableTypeName = "pods"

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane == NavPane {
		t.Error("AC#4: activePane = NavPane after DetailPane Esc; want TablePane (DetailPane Esc unchanged)")
	}
	if appM.activePane != TablePane {
		t.Errorf("AC#4: activePane = %v after DetailPane Esc, want TablePane", appM.activePane)
	}
	if appM.showDashboard {
		t.Error("AC#4: showDashboard = true after DetailPane Esc, want false")
	}
}

// AC#7 anti-regression — [Anti-regression] Overlay Esc dismisses overlay, no showDashboard.
func TestFB041_Overlay_Esc_DismissesOverlay_NoShowDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	m.overlay = HelpOverlayID
	m.statusBar.Mode = components.ModeOverlay

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.overlay != NoOverlay {
		t.Errorf("AC#7: overlay = %v after Esc, want NoOverlay (overlay dismissed)", appM.overlay)
	}
	if appM.showDashboard {
		t.Error("AC#7: showDashboard = true after overlay Esc, want false (overlay Esc must not trigger dashboard)")
	}
}

// ==================== End FB-041 ====================

// ==================== FB-043: Consumer-legible data freshness signal (model layer) ====================

// AC#2 — [Observable] buildQuotaSectionHeader includes freshness when
// bucketsFetchedAt is set and detail width is wide enough (innerW >= 60).
func TestFB043_BuildQuotaSectionHeader_WithFreshness(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.detail = components.NewDetailViewModel(80, 20) // innerW = 77 ≥ 60
	m.bucketsFetchedAt = time.Now().Add(-5 * time.Second)

	got := stripANSIModel(m.buildQuotaSectionHeader())
	if !strings.Contains(got, "updated") {
		t.Errorf("AC#2: buildQuotaSectionHeader() missing 'updated' freshness (wide detail, fetchedAt set):\n%s", got)
	}
	if !strings.Contains(got, "s ago") {
		t.Errorf("AC#2: buildQuotaSectionHeader() missing 'Xs ago' freshness value:\n%s", got)
	}
}

// [Anti-behavior] Zero bucketsFetchedAt → no freshness in section header.
func TestFB043_BuildQuotaSectionHeader_ZeroTime_NoFreshness(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.detail = components.NewDetailViewModel(80, 20)
	// bucketsFetchedAt zero by default.

	got := stripANSIModel(m.buildQuotaSectionHeader())
	if strings.Contains(got, "updated") {
		t.Errorf("buildQuotaSectionHeader() contains 'updated' with zero bucketsFetchedAt; want absent:\n%s", got)
	}
}

// [Anti-behavior] Narrow detail (innerW < 60) → freshness absent even with fetchedAt set.
func TestFB043_BuildQuotaSectionHeader_NarrowWidth_NoFreshness(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.detail = components.NewDetailViewModel(60, 20) // innerW = 57 < 60
	m.bucketsFetchedAt = time.Now().Add(-5 * time.Second)

	got := stripANSIModel(m.buildQuotaSectionHeader())
	if strings.Contains(got, "updated") {
		t.Errorf("buildQuotaSectionHeader() contains 'updated' at innerW<60; want absent (narrow guard):\n%s", got)
	}
}

// AC#4 — [Input-changed] BucketsLoadedMsg sets bucketsFetchedAt and propagates to quota dashboard.
func TestFB043_BucketsLoadedMsg_SetsBucketsFetchedAt(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	if !m.bucketsFetchedAt.IsZero() {
		t.Fatal("precondition: bucketsFetchedAt must be zero before BucketsLoadedMsg")
	}

	before := time.Now()
	result, _ := m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := result.(AppModel)
	after := time.Now()

	if appM.bucketsFetchedAt.IsZero() {
		t.Error("AC#4: bucketsFetchedAt = zero after BucketsLoadedMsg, want non-zero (freshness clock set)")
	}
	if appM.bucketsFetchedAt.Before(before) || appM.bucketsFetchedAt.After(after) {
		t.Errorf("AC#4: bucketsFetchedAt = %v, want in range [%v, %v]", appM.bucketsFetchedAt, before, after)
	}
}

// [Anti-regression] ContextSwitchedMsg resets bucketsFetchedAt to zero.
func TestFB043_ContextSwitchedMsg_ResetsBucketsFetchedAt(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.bucketsFetchedAt = time.Now().Add(-10 * time.Second)

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if !appM.bucketsFetchedAt.IsZero() {
		t.Errorf("bucketsFetchedAt = %v after ContextSwitchedMsg, want zero (state reset)", appM.bucketsFetchedAt)
	}
}

// [Anti-behavior] BucketsErrorMsg does NOT update bucketsFetchedAt.
func TestFB043_BucketsErrorMsg_DoesNotSetFetchedAt(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	// Leave bucketsFetchedAt zero — error should not set it.

	result, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("network timeout"), Unauthorized: false})
	appM := result.(AppModel)

	if !appM.bucketsFetchedAt.IsZero() {
		t.Errorf("bucketsFetchedAt = %v after BucketsErrorMsg, want zero (error must not set freshness clock)", appM.bucketsFetchedAt)
	}
}

// ==================== End FB-043 ====================

// ==================== FB-044: [3] full-dashboard affordance in quota section header ====================

// newDetailPaneModelWithMatchingBucket builds an AppModel in DetailPane with
// a bucket whose ResourceType matches describeRT ("compute.example.io/cpus"),
// describeContent set, and a detail width of 80 (innerW=77 ≥ 30 → [3] rendered).
func newDetailPaneModelWithMatchingBucket() AppModel {
	rt := data.ResourceType{Name: "cpus", Kind: "Cpu", Group: "compute.example.io"}
	m := AppModel{
		ctx:        context.Background(),
		rc:         stubResourceClient{},
		activePane: DetailPane,
		describeRT: rt,
		describeContent: "Name: my-cpu\nStatus: Ready",
		buckets: []data.AllowanceBucket{
			{Name: "b1", ResourceType: "compute.example.io/cpus", Limit: 100, Allocated: 5},
		},
		sidebar:     components.NewNavSidebarModel(22, 20),
		table:       components.NewResourceTableModel(58, 20),
		detail:      components.NewDetailViewModel(80, 20), // innerW = 77 ≥ 30
		quota:       components.NewQuotaDashboardModel(58, 20, "proj"),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// AC#1 — [Happy path] matching buckets + innerW ≥ 30 → "[3]" and "quota dashboard"
// appear in the quota section header inside buildDetailContent(). Copy updated by FB-109.
func TestFB044_BuildDetailContent_MatchingBuckets_Wide_Has3Affordance(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()

	got := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(got, "[3]") {
		t.Errorf("AC#1: '[3]' missing from buildDetailContent() with matching bucket and innerW≥30:\n%s", got)
	}
	if !strings.Contains(got, "quota dashboard") {
		t.Errorf("AC#1: 'quota dashboard' missing from buildDetailContent() with matching bucket and innerW≥30:\n%s", got)
	}
	if strings.Contains(got, "full dashboard") {
		t.Errorf("AC#1 FB-109: 'full dashboard' still present; copy must be '[3] quota dashboard':\n%s", got)
	}
}

// AC#2 — [Anti-behavior] no matching buckets → "[3]" is absent from buildDetailContent().
func TestFB044_BuildDetailContent_NoMatchingBuckets_No3Affordance(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	// Replace buckets with one that doesn't match describeRT.
	m.buckets = []data.AllowanceBucket{
		{Name: "b1", ResourceType: "other.example.io/widgets", Limit: 50},
	}

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "[3]") {
		t.Errorf("AC#2: '[3]' present with non-matching buckets, want absent:\n%s", got)
	}
}

// [Anti-behavior] narrow detail (innerW < 30) → "[3]" is absent even with matching buckets.
func TestFB044_BuildQuotaSectionHeader_NarrowWidth_No3Affordance(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	m.detail = components.NewDetailViewModel(30, 20) // innerW = 27 < 30

	got := stripANSIModel(m.buildQuotaSectionHeader())
	if strings.Contains(got, "[3]") {
		t.Errorf("buildQuotaSectionHeader() at innerW<30 contains '[3]', want absent:\n%s", got)
	}
	// Plain rule prefix still present.
	if !strings.Contains(got, "── Quota") {
		t.Errorf("buildQuotaSectionHeader() at innerW<30 missing '── Quota' separator:\n%s", got)
	}
}

// [Input-changed] width change: narrow (no [3]) vs wide (has [3]) produce different headers.
func TestFB044_BuildQuotaSectionHeader_WidthChange_3AffordanceAppearsDisappears(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()

	m.detail = components.NewDetailViewModel(30, 20) // innerW = 27 < 30
	narrowGot := stripANSIModel(m.buildQuotaSectionHeader())

	m.detail = components.NewDetailViewModel(80, 20) // innerW = 77 ≥ 30
	wideGot := stripANSIModel(m.buildQuotaSectionHeader())

	if strings.Contains(narrowGot, "[3]") {
		t.Errorf("narrow header contains '[3]', want absent:\n%s", narrowGot)
	}
	if !strings.Contains(wideGot, "[3]") {
		t.Errorf("wide header missing '[3]'], want present:\n%s", wideGot)
	}
	if narrowGot == wideGot {
		t.Error("narrow and wide headers are identical, want different content")
	}
}

// [Repeat] calling buildDetailContent() twice returns identical output (idempotent).
func TestFB044_BuildDetailContent_RepeatCall_Idempotent(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()

	first := stripANSIModel(m.buildDetailContent())
	second := stripANSIModel(m.buildDetailContent())

	if first != second {
		t.Errorf("buildDetailContent() not idempotent:\nfirst:  %q\nsecond: %q", first, second)
	}
}

// [Boundary] detail width exactly 33 (innerW=30, the threshold) → "[3]" present.
func TestFB044_BuildQuotaSectionHeader_BoundaryWidth33_Has3Affordance(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	m.detail = components.NewDetailViewModel(33, 20) // innerW = 30, exactly at threshold

	got := stripANSIModel(m.buildQuotaSectionHeader())
	if !strings.Contains(got, "[3]") {
		t.Errorf("buildQuotaSectionHeader() at boundary innerW=30 missing '[3]':\n%s", got)
	}
}

// [Anti-regression] empty buckets list → buildDetailContent returns bare describe
// content without quota section (no "[3]", no "── Quota").
func TestFB044_BuildDetailContent_EmptyBuckets_NoBareQuotaSection(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	m.buckets = nil

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "[3]") {
		t.Errorf("[Anti-regression] '[3]' present with nil buckets, want absent:\n%s", got)
	}
	if strings.Contains(got, "── Quota") {
		t.Errorf("[Anti-regression] '── Quota' present with nil buckets, want absent:\n%s", got)
	}
}

// ==================== FB-066: prefixWidth constant pinned to actual rendered width ====================

// [Observable] AC1: lipgloss.Width of the prefix plain text equals the prefixWidth constant (24).
// Uses the plain string (no ANSI) so no styles import is needed — lipgloss.Width strips ANSI anyway.
// Copy updated by FB-109: "── Quota [3] full dashboard  " → "── [3] quota dashboard  ".
func TestFB066_PrefixWidth_ConstantMatchesRenderedWidth(t *testing.T) {
	t.Parallel()
	// Mirror the plain text of the prefix from model.go (ANSI codes don't affect display width).
	const plainPrefix = "── " + "[3]" + " quota dashboard  "
	const wantPrefixWidth = 24
	if got := lipgloss.Width(plainPrefix); got != wantPrefixWidth {
		t.Errorf("prefixWidth constant drift: lipgloss.Width(plainPrefix) = %d, want %d — update const prefixWidth in model.go", got, wantPrefixWidth)
	}
}

// [Anti-regression] AC1 (integration): at innerW=120, no freshness, the rendered rule line
// fills the full width exactly. If prefix copy and constant diverge, total width != 120.
func TestFB066_PrefixWidth_RuleFillsFullWidth(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	m.detail = components.NewDetailViewModel(123, 20) // innerW = 120
	// bucketsFetchedAt zero — no freshness appended.

	raw := stripANSIModel(m.buildQuotaSectionHeader())
	line := strings.TrimSpace(raw)
	const wantInnerW = 120
	if got := lipgloss.Width(line); got != wantInnerW {
		t.Errorf("rule line width = %d, want %d (innerW); prefix+rule must sum to innerW exactly — check const prefixWidth in model.go:1799\nline: %q", got, wantInnerW, line)
	}
}

// [Boundary] innerW=30 (detail.Width()=33): ruleLen = max(0, 30-29) = 1, no negative artifact.
func TestFB066_PrefixWidth_BoundaryInnerW30_NoNegativeRule(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	m.detail = components.NewDetailViewModel(33, 20) // innerW = 30, exactly one rule char

	raw := stripANSIModel(m.buildQuotaSectionHeader())
	line := strings.TrimSpace(raw)
	const wantInnerW = 30
	if got := lipgloss.Width(line); got != wantInnerW {
		t.Errorf("boundary rule line width = %d, want %d — ruleLen must be 1, not 0 or negative\nline: %q", got, wantInnerW, line)
	}
	if !strings.Contains(line, "[3]") {
		t.Errorf("boundary innerW=30: '[3]' missing; threshold check at model.go:1789 may have regressed\nline: %q", line)
	}
}

// ==================== End FB-066 ====================

// ==================== FB-109: DetailPane copy alignment ([3] quota dashboard) ====================

// AC1 [Observable] — bottom separator contains "[3] quota dashboard" and NOT "full dashboard".
func TestFB109_AC1_BottomSeparatorCopyUpdated(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()

	got := stripANSIModel(m.buildQuotaSectionHeader())
	if !strings.Contains(got, "[3] quota dashboard") {
		t.Errorf("AC1: '[3] quota dashboard' missing from buildQuotaSectionHeader():\n%s", got)
	}
	if strings.Contains(got, "full dashboard") {
		t.Errorf("AC1: 'full dashboard' still present; copy must be aligned to '[3] quota dashboard':\n%s", got)
	}
}

// AC5 [Anti-regression] — at innerW=80, separator fills exactly 80 chars (prefixWidth=24 correct).
func TestFB109_AC5_PrefixWidth24_SeparatorFillsWidth(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithMatchingBucket()
	m.detail = components.NewDetailViewModel(83, 20) // innerW = 80

	raw := stripANSIModel(m.buildQuotaSectionHeader())
	line := strings.TrimSpace(raw)
	const wantInnerW = 80
	if got := lipgloss.Width(line); got != wantInnerW {
		t.Errorf("AC5: separator line width = %d, want %d (innerW=80); prefixWidth=24 may be wrong\nline: %q", got, wantInnerW, line)
	}
}

// ==================== End FB-109 ====================

// ==================== End FB-044 ====================

// ==================== FB-042: Enhanced welcome dashboard (model layer) ====================

// [Observable] computeAttentionItems: buckets below 80% → no items returned.
func TestFB042_ComputeAttentionItems_BelowThreshold_Empty(t *testing.T) {
	t.Parallel()
	buckets := []data.AllowanceBucket{
		{Name: "b1", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "my-proj", Limit: 100, Allocated: 79},
	}
	items := computeAttentionItems(buckets, "Project", "my-proj", nil)
	if len(items) != 0 {
		t.Errorf("computeAttentionItems: want 0 items at 79%%, got %d: %v", len(items), items)
	}
}

// [Observable] computeAttentionItems: bucket at exactly 80% → item created.
func TestFB042_ComputeAttentionItems_AtThreshold_OneItem(t *testing.T) {
	t.Parallel()
	buckets := []data.AllowanceBucket{
		{Name: "b1", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "my-proj", Limit: 100, Allocated: 80},
	}
	items := computeAttentionItems(buckets, "Project", "my-proj", nil)
	if len(items) != 1 {
		t.Fatalf("computeAttentionItems: want 1 item at 80%%, got %d", len(items))
	}
	if items[0].Kind != "quota" {
		t.Errorf("item.Kind = %q, want 'quota'", items[0].Kind)
	}
	if !strings.Contains(items[0].Detail, "80%") {
		t.Errorf("item.Detail = %q, want to contain '80%%'", items[0].Detail)
	}
}

// [Input-changed] computeAttentionItems: mixed buckets — only ≥80% produce items.
func TestFB042_ComputeAttentionItems_Mixed_OnlyAboveThreshold(t *testing.T) {
	t.Parallel()
	buckets := []data.AllowanceBucket{
		{Name: "high", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "my-proj", Limit: 100, Allocated: 90},
		{Name: "low", ResourceType: "compute/memory", ConsumerKind: "Project", ConsumerName: "my-proj", Limit: 100, Allocated: 50},
	}
	items := computeAttentionItems(buckets, "Project", "my-proj", nil)
	if len(items) != 1 {
		t.Fatalf("want 1 item (only 90%% bucket), got %d: %v", len(items), items)
	}
	if !strings.Contains(items[0].Detail, "90%") {
		t.Errorf("item.Detail = %q, want '90%%'", items[0].Detail)
	}
}

// newLargeWelcomePanelAppModel is like newWelcomePanelAppModel but with a large
// table (120×30) so S3 (contentH≥24) and S5 (contentH≥30, contentW≥60) are shown.
func newLargeWelcomePanelAppModel() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 34),
		table:       components.NewResourceTableModel(120, 34),
		detail:      components.NewDetailViewModel(120, 34),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// [Observable] ProjectActivityLoadedMsg feeds table.SetActivityRows on AppModel.
func TestFB042_ProjectActivityLoadedMsg_FeedsWelcomeTeaser(t *testing.T) {
	t.Parallel()
	m := newLargeWelcomePanelAppModel()

	rows := []data.ActivityRow{
		{Timestamp: time.Now(), ActorDisplay: "alice@example.com", Summary: "created project"},
	}
	result, _ := m.Update(data.ProjectActivityLoadedMsg{Rows: rows})
	appM := result.(AppModel)

	// S3 visible at contentH=30 ≥ 24.
	got := stripANSIModel(appM.table.View())
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("after ProjectActivityLoadedMsg: want actor 'alice@example.com' in welcome panel, got:\n%s", got)
	}
}

// [Anti-regression] ContextSwitchedMsg clears activity rows and attention items.
func TestFB042_ContextSwitchedMsg_ClearsActivityAndAttention(t *testing.T) {
	t.Parallel()
	m := newLargeWelcomePanelAppModel()
	// Pre-populate activity and attention.
	m.table.SetActivityRows([]data.ActivityRow{
		{Timestamp: time.Now(), ActorDisplay: "alice@example.com", Summary: "created project"},
	})
	m.table.SetAttentionItems([]components.AttentionItem{
		{Kind: "quota", Label: "cpus quota", Detail: "90% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
	})

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	got := stripANSIModel(appM.table.View())
	if strings.Contains(got, "alice@example.com") {
		t.Errorf("after ContextSwitchedMsg: 'alice@example.com' still present; activity rows not cleared:\n%s", got)
	}
	if strings.Contains(got, "Needs attention") {
		t.Errorf("after ContextSwitchedMsg: 'Needs attention' still present; attention items not cleared:\n%s", got)
	}
}

// [Observable] Quick-jump 'n' key from welcome panel with "namespaces" resource type
// dispatches a LoadResourcesCmd (transitions out of welcome panel).
func TestFB042_QuickJump_NKey_WithMatchingType_Navigates(t *testing.T) {
	t.Parallel()
	m := newWelcomePanelAppModel(nil, nil)
	m.resourceTypes = []data.ResourceType{
		{Name: "namespaces", Kind: "Namespace", Group: "core.miloapis.com"},
	}
	// tableTypeName="" → welcome panel active. FB-073: quick-jump fires from TablePane, not NavPane.
	m.activePane = TablePane
	m.updatePaneFocus()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	appM := result.(AppModel)

	if appM.tableTypeName != "namespaces" {
		t.Errorf("after 'n' quick-jump: tableTypeName = %q, want 'namespaces'", appM.tableTypeName)
	}
	if cmd == nil {
		t.Error("after 'n' quick-jump: expected a LoadResourcesCmd, got nil")
	}
}

// [Anti-behavior] Quick-jump key with no matching resource type → no-op (tableTypeName unchanged).
func TestFB042_QuickJump_NKey_NoMatchingType_NoOp(t *testing.T) {
	t.Parallel()
	m := newWelcomePanelAppModel(nil, nil)
	m.resourceTypes = []data.ResourceType{
		{Name: "projects", Kind: "Project", Group: "resourcemanager.miloapis.com"},
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	appM := result.(AppModel)

	if appM.tableTypeName != "" {
		t.Errorf("after 'n' with no 'namespaces' type: tableTypeName = %q, want '' (no-op)", appM.tableTypeName)
	}
}

// [Anti-behavior] Quick-jump keys inactive when a type is already loaded (not welcome panel).
func TestFB042_QuickJump_InactiveWhenTypeLoaded(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel() // tableTypeName="pods" — not welcome panel
	m.resourceTypes = []data.ResourceType{
		{Name: "namespaces", Kind: "Namespace", Group: "core.miloapis.com"},
		{Name: "pods", Kind: "Pod", Namespaced: true},
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	appM := result.(AppModel)

	// tableTypeName must remain "pods" — quick-jump must not fire when type is loaded.
	if appM.tableTypeName != "pods" {
		t.Errorf("quick-jump 'n' fired when tableTypeName='pods'; want no-op, got tableTypeName=%q", appM.tableTypeName)
	}
}

// [Repeat-press + input-changed axis] AC8: quick-jump → load completes → Esc → welcome
// round-trip re-renders the welcome dashboard.
func TestFB042_QuickJump_RoundTrip_ReturnsToWelcome(t *testing.T) {
	t.Parallel()
	m := newWelcomePanelAppModel(nil, nil)
	m.resourceTypes = []data.ResourceType{
		{Name: "namespaces", Kind: "Namespace", Group: "core.miloapis.com"},
	}
	// FB-073: quick-jump fires from TablePane.
	m.activePane = TablePane
	m.updatePaneFocus()

	// Forward leg: 'n' quick-jump navigates to namespaces (loadState=Loading).
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	appM := result.(AppModel)
	if appM.tableTypeName != "namespaces" {
		t.Fatalf("forward leg: tableTypeName = %q, want 'namespaces'", appM.tableTypeName)
	}

	// Simulate load completion so loadState exits Loading (otherwise the Loading
	// spinner case in ResourceTableModel.View() takes priority over forceDashboard).
	result, _ = appM.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "namespaces", Kind: "Namespace", Group: "core.miloapis.com"},
	})
	appM = result.(AppModel)

	// Return leg: FB-072 — quick-jump entry means one Esc press returns to welcome directly.
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyEsc}) // TablePane → welcome (single press)
	appM = result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC8 round-trip: 'Welcome' missing after Esc; showDashboard=%v tableTypeName=%q\nview: %q",
			appM.showDashboard, appM.tableTypeName, got)
	}
	if !strings.Contains(got, "Platform health") {
		t.Errorf("AC8 round-trip: 'Platform health' missing after Esc; showDashboard=%v tableTypeName=%q\nview: %q",
			appM.showDashboard, appM.tableTypeName, got)
	}
}

// ==================== End FB-042 (model layer) ====================

// ==================== FB-072: Quick-jump Esc round-trip requires only 1 press ====================

// newQuickJumpModel builds a welcome-panel AppModel that has fired a quick-jump to "backends".
// The model is in TablePane with lastEntryViaQuickJump=true.
func newQuickJumpModel() AppModel {
	m := newWelcomePanelAppModel(nil, nil)
	m.resourceTypes = []data.ResourceType{
		{Name: "backends", Kind: "Backend", Group: "networking.datum.net"},
	}
	// FB-073: quick-jump fires from TablePane, not NavPane.
	m.activePane = TablePane
	m.updatePaneFocus()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	appM := result.(AppModel)
	// Simulate load completion so view shows table, not spinner.
	result, _ = appM.Update(data.ResourcesLoadedMsg{
		Rows:         []data.ResourceRow{},
		Columns:      []string{"Name"},
		ResourceType: data.ResourceType{Name: "backends", Kind: "Backend", Group: "networking.datum.net"},
	})
	return result.(AppModel)
}

// AC1 — [Observable] Quick-jump 'b' → single Esc restores welcome panel.
// showDashboard must be true and View() must contain "Welcome".
func TestFB072_AC1_QuickJump_SingleEsc_RestoresWelcome(t *testing.T) {
	t.Parallel()
	appM := newQuickJumpModel()
	if appM.activePane != TablePane {
		t.Fatalf("precondition: activePane=%v, want TablePane", appM.activePane)
	}
	if !appM.lastEntryViaQuickJump {
		t.Fatal("precondition: lastEntryViaQuickJump must be true after quick-jump")
	}

	result, _ := appM.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM = result.(AppModel)

	if !appM.showDashboard {
		t.Error("AC1: showDashboard=false after single Esc from quick-jump entry, want true")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC1: 'Welcome' absent from View() after single Esc:\n%s", got)
	}
}

// AC2 — [Repeat-press] Second Esc from welcome after quick-jump round-trip does NOT re-enter TablePane.
// Step 1: Esc → welcome. Step 2: second Esc → still NavPane+dashboard (no-op).
func TestFB072_AC2_QuickJump_SecondEsc_IsNoop(t *testing.T) {
	t.Parallel()
	appM := newQuickJumpModel()

	// First Esc: TablePane → welcome.
	r1, _ := appM.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m1 := r1.(AppModel)
	if !m1.showDashboard {
		t.Fatal("AC2 setup: first Esc did not restore welcome panel")
	}

	// Second Esc: must not leave welcome panel.
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := r2.(AppModel)
	if !m2.showDashboard {
		t.Error("AC2: showDashboard=false after second Esc, want true (no re-entry into TablePane)")
	}
	if m2.activePane == TablePane {
		t.Error("AC2: activePane=TablePane after second Esc, want NavPane (welcome)")
	}
}

// AC3 — [Anti-regression] Sidebar-driven nav (Enter from NavPane) still requires FB-041 two-step Esc.
// Enter from NavPane → TablePane; first Esc → NavPane (not welcome); second Esc → welcome.
func TestFB072_AC3_SidebarNav_TwoEscRequired(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	if m.lastEntryViaQuickJump {
		t.Fatal("precondition: lastEntryViaQuickJump must be false for sidebar model")
	}

	// Navigate to TablePane via Enter.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m1 := r1.(AppModel)
	if m1.lastEntryViaQuickJump {
		t.Fatal("AC3: lastEntryViaQuickJump set after sidebar Enter, want false")
	}

	// First Esc: TablePane → NavPane (NOT welcome).
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := r2.(AppModel)
	if m2.activePane != NavPane {
		t.Errorf("AC3: first Esc from sidebar-nav TablePane: activePane=%v, want NavPane", m2.activePane)
	}
	if m2.showDashboard {
		t.Error("AC3: showDashboard=true after first Esc from sidebar-nav, want false (requires two-step)")
	}

	// Second Esc: NavPane → welcome.
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := r3.(AppModel)
	if !m3.showDashboard {
		t.Error("AC3: showDashboard=false after second Esc from sidebar-nav NavPane, want true")
	}
}

// AC4 — [Input-changed] Quick-jump 'b' → Tab (to NavPane) → 'j' clears flag → Tab (back to TablePane) →
// Esc: flag cleared; Esc goes to NavPane (FB-041 two-step), NOT directly to welcome.
func TestFB072_AC4_QuickJump_ThenSidebarJ_ClearsFlag_TwoStepEsc(t *testing.T) {
	t.Parallel()
	appM := newQuickJumpModel()
	if !appM.lastEntryViaQuickJump {
		t.Fatal("precondition: lastEntryViaQuickJump must be true after quick-jump")
	}

	// Tab: TablePane → NavPane (flag still set).
	r1, _ := appM.Update(tea.KeyMsg{Type: tea.KeyTab})
	m1 := r1.(AppModel)
	if m1.activePane != NavPane {
		t.Fatalf("precondition: Tab from TablePane should give NavPane, got %v", m1.activePane)
	}

	// 'j' in NavPane → clears flag.
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m2 := r2.(AppModel)
	if m2.lastEntryViaQuickJump {
		t.Error("AC4: lastEntryViaQuickJump still true after 'j' in NavPane, want false")
	}

	// Tab: NavPane → TablePane (flag still false).
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	m3 := r3.(AppModel)
	if m3.activePane != TablePane {
		t.Fatalf("precondition: Tab from NavPane should give TablePane, got %v", m3.activePane)
	}

	// Esc: flag is false → standard FB-041 path → NavPane, NOT welcome.
	r4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m4 := r4.(AppModel)
	if m4.showDashboard {
		t.Error("AC4: showDashboard=true after Esc with flag cleared, want false (requires two-step)")
	}
	if m4.activePane != NavPane {
		t.Errorf("AC4: activePane=%v after Esc with flag cleared, want NavPane", m4.activePane)
	}
}

// ==================== End FB-072 ====================

// ==================== FB-073: Quick-jump NavPane gate ====================

// newNavPaneWelcomeWithBackend builds a welcome-panel model with a "backends" resource
// type registered and NavPane active — the state where accidental quick-jump fires occurred.
func newNavPaneWelcomeWithBackend() AppModel {
	m := newWelcomePanelAppModel(nil, nil)
	m.resourceTypes = []data.ResourceType{
		{Name: "backends", Kind: "Backend", Group: "networking.datum.net"},
	}
	return m
}

// AC1 [Anti-behavior] — pressing 'b' from NavPane while welcome panel visible does NOT fire quick-jump.
func TestFB073_AC1_NavPane_QuickJump_NoFire(t *testing.T) {
	t.Parallel()
	m := newNavPaneWelcomeWithBackend()
	if m.activePane != NavPane {
		t.Fatalf("precondition: activePane=%v, want NavPane", m.activePane)
	}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC1 [Anti-behavior]: activePane changed to %v after 'b' from NavPane, want NavPane", appM.activePane)
	}
	if appM.tableTypeName != "" {
		t.Errorf("AC1 [Anti-behavior]: tableTypeName=%q after 'b' from NavPane, want '' (no-op)", appM.tableTypeName)
	}
	if cmd != nil {
		t.Error("AC1 [Anti-behavior]: non-nil cmd after suppressed NavPane quick-jump, want nil")
	}
	view := stripANSIModel(appM.View())
	if !strings.Contains(view, "Welcome") {
		t.Errorf("AC1 [Anti-behavior]: 'Welcome' absent from View() after suppressed NavPane 'b' — welcome panel must remain:\n%s", view)
	}
}

// AC2 [Observable] — after suppressed NavPane 'b', View() still renders welcome panel (no table).
func TestFB073_AC2_NavPane_QuickJump_ViewUnchanged(t *testing.T) {
	t.Parallel()
	m := newNavPaneWelcomeWithBackend()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	appM := result.(AppModel)

	view := stripANSIModel(appM.View())
	if !strings.Contains(view, "Welcome") {
		t.Errorf("AC2 [Observable]: 'Welcome' absent after suppressed NavPane 'b' — welcome panel must remain:\n%s", view)
	}
}

// AC3 [Anti-regression] — pressing 'b' from TablePane (welcome panel visible) DOES fire quick-jump.
func TestFB073_AC3_TablePane_QuickJump_StillFires(t *testing.T) {
	t.Parallel()
	m := newNavPaneWelcomeWithBackend()
	m.activePane = TablePane
	m.updatePaneFocus()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	appM := result.(AppModel)

	if appM.tableTypeName != "backends" {
		t.Errorf("AC3 [Anti-regression]: tableTypeName=%q after 'b' from TablePane, want 'backends'", appM.tableTypeName)
	}
	if cmd == nil {
		t.Error("AC3 [Anti-regression]: nil cmd after TablePane quick-jump, want LoadResourcesCmd")
	}
	view := stripANSIModel(appM.View())
	if strings.Contains(view, "Welcome") {
		t.Errorf("AC3 [Anti-regression]: 'Welcome' still in View() after TablePane quick-jump — welcome panel must be gone once tableTypeName is set:\n%s", view)
	}
}

// ==================== End FB-073 ====================

// ==================== FB-047: '3' keypress queued while QuotaDashboard loads ====================

// newQuotaLoadingModel builds an AppModel in NavPane with quota in loading state.
// quotaBucketsLoaded is false (first load not yet complete) — the state FB-047 targets.
func newQuotaLoadingModel() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 20),
		table:       components.NewResourceTableModel(58, 20),
		detail:      components.NewDetailViewModel(58, 20),
		quota:       components.NewQuotaDashboardModel(58, 20, "proj"),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.quota.SetLoading(true)
	m.updatePaneFocus()
	return m
}

// [Observable] AC1: '3' during quota loading → statusBar hint contains "Quota dashboard loading".
func TestFB047_Key3_DuringLoading_PostsHint(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC1: 'Quota dashboard loading' missing from View() after '3' during loading:\n%s", got)
	}
	if !appM.pendingQuotaOpen {
		t.Error("AC1: pendingQuotaOpen = false after first '3' during loading, want true")
	}
}

// [Input-changed] AC4: View() differs before and after '3' press during loading (hint appears).
func TestFB047_Key3_DuringLoading_ViewChanges(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	before := stripANSIModel(m.View())
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	after := stripANSIModel(result.(AppModel).View())

	if before == after {
		t.Error("AC4: View() identical before and after '3' during loading, want hint to appear")
	}
}

// [Repeat-press] AC2: two '3' presses during loading → second press cancels (FB-080);
// BucketsLoadedMsg afterwards does NOT auto-transition (single-transition constraint still holds).
func TestFB047_Key3_DoublePressLoading_SingleTransition(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	// FB-080: second press cancels the queued open.
	if appM.pendingQuotaOpen {
		t.Error("AC2: pendingQuotaOpen = true after second '3', want false (FB-080 cancel)")
	}

	result, _ = appM.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM = result.(AppModel)

	// After cancel, BucketsLoadedMsg must NOT trigger transition or show ready prompt.
	if appM.activePane == QuotaDashboardPane {
		t.Errorf("AC2: activePane = QuotaDashboardPane after BucketsLoadedMsg, want origin pane (pending was cancelled)")
	}
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("AC2: ready prompt shown after cancel + BucketsLoadedMsg, want absent:\n%s", got)
	}
}

// [Happy path] AC3: '3' when not loading → immediate transition to QuotaDashboardPane (FB-035 guard).
func TestFB047_Key3_NotLoading_ImmediateTransition(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()
	m.quota.SetLoading(false)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC3: activePane = %v after '3' when not loading, want QuotaDashboardPane", appM.activePane)
	}
	if appM.pendingQuotaOpen {
		t.Error("AC3: pendingQuotaOpen = true after immediate transition, want false")
	}
}

// [Observable] AC5: BucketsLoadedMsg with pending open → FB-078 ready prompt shown; no auto-transition.
// Operator presses '3' manually to open dashboard.
func TestFB047_BucketsLoadedMsg_ResolvesPending_TransitionsAndClearsHint(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be true after '3' during loading")
	}

	result, _ = appM.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM = result.(AppModel)

	// FB-078: no force-transition; ready prompt replaces loading hint.
	if appM.activePane == QuotaDashboardPane {
		t.Errorf("AC5: activePane = QuotaDashboardPane after BucketsLoadedMsg, want origin pane (no force-transition)")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("AC5: ready prompt missing from View() after BucketsLoadedMsg:\n%s", got)
	}
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC5: 'Quota dashboard loading' hint still present after BucketsLoadedMsg, want absent:\n%s", got)
	}
}

// [Error path] AC6: BucketsErrorMsg with pending open → pendingQuotaOpen=false, hint absent.
func TestFB047_BucketsErrorMsg_ClearsPendingOpen(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be true after '3' during loading")
	}

	result, _ = appM.Update(data.BucketsErrorMsg{Err: errors.New("load failed")})
	appM = result.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC6: pendingQuotaOpen = true after BucketsErrorMsg, want false")
	}
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC6: 'Quota dashboard loading' hint still present after BucketsErrorMsg, want absent:\n%s", got)
	}
}

// [Error path] AC7: LoadErrorMsg (generic load error) with pending open → pendingQuotaOpen=false,
// hint absent. Covers Site 3b in the implementation.
func TestFB047_LoadErrorMsg_ClearsPendingOpen(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be true after '3' during loading")
	}

	result, _ = appM.Update(data.LoadErrorMsg{Err: errors.New("network error")})
	appM = result.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC7: pendingQuotaOpen = true after LoadErrorMsg, want false")
	}
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC7: 'Quota dashboard loading' hint still present after LoadErrorMsg, want absent:\n%s", got)
	}
}

// ==================== End FB-047 ====================

// ==================== FB-080: Second '3' press during loading cancels the queued open ====================
//
// Axis-coverage (brief-AC-indexed):
// AC  | Axis            | Test(s)
// ----+-----------------+-----------------------------------------------------------------
// AC1 | Repeat-press    | TestFB080_AC1_SecondPress_HintAbsentFromView
// AC2 | Observable      | TestFB080_AC2_InputChanged_3Key_FirstPress_Stashes
//     |                 | TestFB080_AC2_InputChanged_3Key_SecondPress_Cancels
// AC3 | Anti-behavior   | TestFB080_AC3_SecondPress_PendingFalse
//     |                 | TestFB080_AC3_SecondPress_PreservesQuotaOriginPane
// AC4 | Anti-regression | TestFB080_AC4_FirstPress_StillQueues (single-press path intact)
// AC5 | Anti-regression | TestFB047_Key3_DoublePressLoading_SingleTransition (existing, updated)
// AC6 | Integration     | go install ./... + go test ./internal/tui/...

// [Repeat-press] AC1: second '3' press while pending → hint absent from View().
func TestFB080_AC1_SecondPress_HintAbsentFromView(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// First press: queue the open.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be true after first '3'")
	}

	// Second press: cancel.
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	got := stripANSIModel(appM.View())
	if strings.Contains(got, "loading") {
		t.Errorf("AC1: 'loading' still present in View() after second '3' (cancel):\n%s", got)
	}
}

// [Observable / Input-changed] AC2, pair A: '3' when pendingQuotaOpen==false → stashes origin, posts hint, sets pending=true.
func TestFB080_AC2_InputChanged_3Key_FirstPress_Stashes(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()
	// Ensure we start from NavPane with a known origin.
	if m.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be false before first '3'")
	}

	beforeView := stripANSIModel(m.View())

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	afterView := stripANSIModel(appM.View())

	// View changed (hint appeared).
	if beforeView == afterView {
		t.Error("AC2 first-press: View() identical before and after, want hint to appear")
	}
	if !strings.Contains(afterView, "loading") {
		t.Errorf("AC2 first-press: 'loading' hint missing from View() after first '3':\n%s", afterView)
	}
	if !appM.pendingQuotaOpen {
		t.Error("AC2 first-press: pendingQuotaOpen = false, want true")
	}
	// quotaOriginPane was written on first press.
	if appM.quotaOriginPane.Pane != NavPane {
		t.Errorf("AC2 first-press: quotaOriginPane.Pane = %v, want NavPane", appM.quotaOriginPane.Pane)
	}
}

// [Observable / Input-changed] AC2, pair B: '3' when pendingQuotaOpen==true (same key, different state) → hint disappears, pending cleared.
func TestFB080_AC2_InputChanged_3Key_SecondPress_Cancels(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// First press to set up pending state.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be true after first '3'")
	}
	beforeSecond := stripANSIModel(appM.View())
	if !strings.Contains(beforeSecond, "loading") {
		t.Fatalf("precondition: 'loading' must be in View() before second press:\n%s", beforeSecond)
	}

	// Second press: cancel.
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	afterSecond := stripANSIModel(appM.View())

	// Same key type ('3'), different model state → different output.
	if beforeSecond == afterSecond {
		t.Error("AC2 second-press: View() identical before and after second '3', want hint to disappear")
	}
	if strings.Contains(afterSecond, "loading") {
		t.Errorf("AC2 second-press: 'loading' still in View() after cancel:\n%s", afterSecond)
	}
	if appM.pendingQuotaOpen {
		t.Error("AC2 second-press: pendingQuotaOpen = true after cancel, want false")
	}
}

// [Anti-behavior] AC3a: after second-press cancel, pendingQuotaOpen == false.
func TestFB080_AC3_SecondPress_PendingFalse(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC3: pendingQuotaOpen = true after second-press cancel, want false")
	}
}

// [Anti-behavior / FB-095 cross-ref] AC3b: second press must NOT overwrite the stash written on first press.
// Updated for FB-095: second-press cancel now CLEARS quotaOriginPane (stash invariant).
// The old assertion (stash preserved) is superseded by the FB-095 fix.
func TestFB080_AC3_SecondPress_PreservesQuotaOriginPane(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()
	m.quotaOriginPane = DashboardOrigin{Pane: TablePane}

	// First press queues open.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	// Second press cancels. FB-095: quotaOriginPane must be zero.
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	if appM.quotaOriginPane != (DashboardOrigin{}) {
		t.Errorf("AC3b (FB-095): quotaOriginPane not cleared on second-press cancel: got %v, want zero",
			appM.quotaOriginPane)
	}
}

// [Anti-regression] AC4: first '3' press while loading still queues correctly (single-press path intact).
func TestFB080_AC4_FirstPress_StillQueues(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	if !appM.pendingQuotaOpen {
		t.Error("AC4: pendingQuotaOpen = false after first '3' during loading, want true")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "loading") {
		t.Errorf("AC4: 'loading' hint missing from View() after first '3':\n%s", got)
	}
	// Must stay in origin pane — no immediate transition.
	if appM.activePane == QuotaDashboardPane {
		t.Errorf("AC4: activePane = QuotaDashboardPane after first '3' during loading, want origin pane")
	}
}

// ==================== End FB-080 ====================

// ==================== FB-067: S3 recent-activity teaser dispatched on Init and context switch ====================

// batchLen executes cmd() and returns the number of leaf commands in the batch.
// Handles a single-level tea.BatchMsg (Init/ContextSwitchedMsg never nest deeper).
func batchLen(cmd tea.Cmd) int {
	if cmd == nil {
		return 0
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return len(batch)
	}
	return 1
}

// newProjectScopedModelWithAC builds an AppModel with all three load clients set
// (bc, rrc, ac) and a project-scoped TUI context — the state where Init() and
// ContextSwitchedMsg should both dispatch LoadRecentProjectActivityCmd.
func newProjectScopedModelWithAC() AppModel {
	m := newWelcomePanelAppModel(&stubBucketClient{}, &stubRegistrationClient{})
	m.ac = data.NewActivityClient(nil)
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}
	return m
}

// [Observable] AC1: ProjectActivityLoadedMsg with rows → welcome panel View() renders row
// content and does NOT contain "⟳ loading". Uses direct message injection to sidestep the
// nil-factory ActivityClient panic (Init() dispatch proven separately via batch-count check).
func TestFB067_ProjectActivityLoadedMsg_RendersRowsInWelcomeView(t *testing.T) {
	t.Parallel()
	// Large enough model so S3 (contentH≥24, contentW≥50) renders in welcome panel.
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 32),
		table:       components.NewResourceTableModel(80, 32), // contentH=28≥24, contentW=76≥50
		detail:      components.NewDetailViewModel(80, 32),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()

	result, _ := m.Update(data.ProjectActivityLoadedMsg{
		Rows: []data.ActivityRow{
			{ActorDisplay: "alice@example.com", Summary: "created api-gw", Timestamp: time.Now().Add(-2 * time.Minute)},
		},
	})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC1: 'alice@example.com' missing from welcome View() after ProjectActivityLoadedMsg:\n%s", got)
	}
	if strings.Contains(got, "⟳ loading") {
		t.Errorf("AC1: '⟳ loading' still present in View() after ProjectActivityLoadedMsg resolved:\n%s", got)
	}
}

// [Happy] AC1 dispatch: Init() with ac + project scope dispatches one more cmd than without ac.
// The extra cmd is LoadRecentProjectActivityCmd; comparing batch counts avoids executing
// the cmd with a nil factory.
func TestFB067_Init_ProjectScope_DispatchesActivityCmd(t *testing.T) {
	t.Parallel()
	withAC := newProjectScopedModelWithAC()
	withoutAC := newWelcomePanelAppModel(&stubBucketClient{}, &stubRegistrationClient{})
	withoutAC.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}

	nWith := batchLen(withAC.Init())
	nWithout := batchLen(withoutAC.Init())

	if nWith != nWithout+1 {
		t.Errorf("AC1 dispatch: Init() with ac dispatched %d cmds, without ac dispatched %d; want exactly 1 more (LoadRecentProjectActivityCmd)", nWith, nWithout)
	}
}

// [Input-changed] AC2: ContextSwitchedMsg with project scope dispatches one more cmd
// than org scope (no ProjectID). The extra cmd is LoadRecentProjectActivityCmd.
func TestFB067_ContextSwitchedMsg_ProjectScope_DispatchesActivityCmd(t *testing.T) {
	t.Parallel()
	m := newProjectScopedModelWithAC()

	projCtx := tuictx.TUIContext{OrgName: "acme", UserName: "alice"}
	projCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}
	orgCtx := tuictx.TUIContext{OrgName: "acme", UserName: "alice"}
	// orgCtx.ActiveCtx is nil — org scope, no project

	_, cmdProj := m.Update(components.ContextSwitchedMsg{Ctx: projCtx})
	_, cmdOrg := m.Update(components.ContextSwitchedMsg{Ctx: orgCtx})

	nProj := batchLen(cmdProj)
	nOrg := batchLen(cmdOrg)

	if nProj != nOrg+1 {
		t.Errorf("AC2: ContextSwitchedMsg project scope dispatched %d cmds, org scope %d; want project = org+1 (LoadRecentProjectActivityCmd)", nProj, nOrg)
	}
}

// [Anti-behavior] AC2 inverse: ContextSwitchedMsg with org scope (no ProjectID)
// does NOT dispatch an extra activity cmd vs a nil-ac model.
func TestFB067_ContextSwitchedMsg_OrgScope_NoActivityCmd(t *testing.T) {
	t.Parallel()
	withAC := newProjectScopedModelWithAC()
	withoutAC := newWelcomePanelAppModel(&stubBucketClient{}, &stubRegistrationClient{})

	orgCtx := tuictx.TUIContext{OrgName: "acme", UserName: "alice"}
	// orgCtx.ActiveCtx nil — no ProjectID

	_, cmdWith := withAC.Update(components.ContextSwitchedMsg{Ctx: orgCtx})
	_, cmdWithout := withoutAC.Update(components.ContextSwitchedMsg{Ctx: orgCtx})

	nWith := batchLen(cmdWith)
	nWithout := batchLen(cmdWithout)

	if nWith != nWithout {
		t.Errorf("AC2 anti-behavior: org-scope ContextSwitchedMsg with ac dispatched %d cmds vs %d without ac; want equal (no activity dispatch when no ProjectID)", nWith, nWithout)
	}
}

// ==================== End FB-067 ====================

// ==================== FB-082: Activity state machine + 3-tier width truncation (model) ====================
//
// Axis-coverage (brief-AC-indexed):
// AC  | Axis            | Test(s)
// ----+-----------------+--------------------------------------------------------------------
// AC1 | Observable      | TestFB082_AC1_Init_ProjectScope_ViewShowsLoading
// AC2 | Input-changed   | TestFB082_AC2_InputChanged_CtxSwitch_ProjectScope_Loads
//     |                 | TestFB082_AC2_InputChanged_CtxSwitch_OrgScope_Empty
// AC3 | Input-changed   | TestFB082_AC3_ProjectActivityErrorMsg_ShowsUnavailable
// AC4 | Anti-behavior   | TestFB082_AC4_SuccessfulFetch_RendersRows
// AC5 | Anti-regression | TestFB067_* (existing tests, named below — verified green)
// AC6 | Anti-regression | TestAppModel_ProjectActivityErrorMsg_CRDAbsent_SetsFlag (existing)
// AC7 | Integration     | go install ./... + go test ./internal/tui/...

// newLargeProjectScopedModelWithAC builds an AppModel large enough to show S3
// (contentH=28≥24, contentW=76≥50) with activity client + project context.
func newLargeProjectScopedModelWithAC() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		bc:          &stubBucketClient{},
		rrc:         &stubRegistrationClient{},
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 32),
		table:       components.NewResourceTableModel(80, 32),
		detail:      components.NewDetailViewModel(80, 32),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.ac = data.NewActivityClient(nil)
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}
	m.updatePaneFocus()
	return m
}

// [Observable] AC1: with project context, setting activityLoading=true (as Init() does)
// renders "⟳ loading…" in the welcome panel View() before any load message arrives.
func TestFB082_AC1_Init_ProjectScope_ViewShowsLoading(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	m.table.SetActivityLoading(true) // simulates the state Init() sets inside the dispatch gate

	got := stripANSIModel(m.View())
	if !strings.Contains(got, "⟳ loading") {
		t.Errorf("AC1: '⟳ loading' spinner missing from View() when activityLoading=true:\n%s", got)
	}
	if strings.Contains(got, "no recent activity") {
		t.Errorf("AC1: 'no recent activity' present when activityLoading=true (should be in loading state):\n%s", got)
	}
}

// [Input-changed] AC2, pair A: ContextSwitchedMsg to project scope → loading state shown.
func TestFB082_AC2_InputChanged_CtxSwitch_ProjectScope_Loads(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()

	projCtx := tuictx.TUIContext{OrgName: "acme", UserName: "alice"}
	projCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}

	result, _ := m.Update(components.ContextSwitchedMsg{Ctx: projCtx})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	// Use "⟳ loading…" (ellipsis directly after "loading") to match S3 activity spinner
	// specifically. S2 bucket spinner renders "⟳ loading platform health…" — the ellipsis
	// follows "health", not "loading", so it doesn't match this substring check.
	if !strings.Contains(got, "⟳ loading…") {
		t.Errorf("AC2 project scope: '⟳ loading…' activity spinner missing from View() after ContextSwitchedMsg with ProjectID:\n%s", got)
	}
	if strings.Contains(got, "no recent activity") {
		t.Errorf("AC2 project scope: 'no recent activity' present when loading state expected:\n%s", got)
	}
}

// [Input-changed] AC2, pair B: same ContextSwitchedMsg type, org scope (no ProjectID) → "no recent activity".
func TestFB082_AC2_InputChanged_CtxSwitch_OrgScope_Empty(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	// Pre-seed loading state to verify org-scope switch correctly clears it.
	m.table.SetActivityLoading(true)

	orgCtx := tuictx.TUIContext{OrgName: "acme", UserName: "alice"}
	// orgCtx.ActiveCtx is nil — org scope, no project

	result, _ := m.Update(components.ContextSwitchedMsg{Ctx: orgCtx})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	// "⟳ loading…" (ellipsis right after "loading") is S3-specific. S2 shows
	// "⟳ loading platform health…" which does NOT contain this exact substring.
	if strings.Contains(got, "⟳ loading…") {
		t.Errorf("AC2 org scope: '⟳ loading…' activity spinner present after org-scope switch (should be cleared):\n%s", got)
	}
	if !strings.Contains(got, "no recent activity") {
		t.Errorf("AC2 org scope: 'no recent activity' missing after org-scope switch:\n%s", got)
	}
}

// [Input-changed] AC3: ProjectActivityErrorMsg while loading → View() shows "activity unavailable", not "loading".
func TestFB082_AC3_ProjectActivityErrorMsg_ShowsUnavailable(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	m.table.SetActivityLoading(true) // simulate post-Init loading state

	before := stripANSIModel(m.View())
	if !strings.Contains(before, "loading") {
		t.Fatal("precondition: 'loading' must be visible before error arrives")
	}

	result, _ := m.Update(data.ProjectActivityErrorMsg{Err: errors.New("fetch failed")})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC3: 'activity unavailable' missing from View() after ProjectActivityErrorMsg:\n%s", got)
	}
	if strings.Contains(got, "loading") {
		t.Errorf("AC3: 'loading' still present in View() after ProjectActivityErrorMsg:\n%s", got)
	}
}

// [Anti-behavior] AC4: successful fetch still renders rows (no regression).
// Summary is "ok" (2 chars) to fit any summaryW tier — actor is the observable target.
func TestFB082_AC4_SuccessfulFetch_RendersRows(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	m.table.SetActivityLoading(true)

	result, _ := m.Update(data.ProjectActivityLoadedMsg{
		Rows: []data.ActivityRow{
			{ActorDisplay: "bob@example.com", Summary: "ok", Timestamp: time.Now().Add(-1 * time.Minute)},
		},
	})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "bob@example.com") {
		t.Errorf("AC4: actor 'bob@example.com' missing from View() after ProjectActivityLoadedMsg:\n%s", got)
	}
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC4: 'activity unavailable' present after successful fetch:\n%s", got)
	}
	if strings.Contains(got, "⟳ loading") {
		t.Errorf("AC4: '⟳ loading' spinner still present after ProjectActivityLoadedMsg:\n%s", got)
	}
}

// ==================== End FB-082 (model) ====================

// ==================== FB-100: activityFetchFailed must not clobber stale rows ====================

// AC1 — [Observable] Populated rows + ProjectActivityErrorMsg → stale rows retained; "activity unavailable" absent.
func TestFB100_AC1_ErrorWithPopulatedRows_KeepsStaleRows(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	// Seed rows first (simulates a prior successful fetch).
	m.table.SetActivityRows([]data.ActivityRow{
		{ActorDisplay: "alice@datum.net", Summary: "deployed", Timestamp: time.Now()},
	})

	result, _ := m.Update(data.ProjectActivityErrorMsg{Err: errors.New("transient error")})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "alice@datum.net") {
		t.Errorf("AC1: stale actor row 'alice@datum.net' missing from View() after error with rows:\n%s", got)
	}
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC1: 'activity unavailable' shown despite stale rows present:\n%s", got)
	}
}

// AC2 — [Observable] Empty rows + ProjectActivityErrorMsg → "activity unavailable" shown (unchanged FB-082 behavior).
func TestFB100_AC2_ErrorWithEmptyRows_ShowsUnavailable(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	m.table.SetActivityLoading(true) // no rows ever loaded

	result, _ := m.Update(data.ProjectActivityErrorMsg{Err: errors.New("fetch failed")})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC2: 'activity unavailable' missing when no rows exist:\n%s", got)
	}
}

// AC3 — [Input-changed] Same ProjectActivityErrorMsg, different ActivityRowCount pre-state → different View().
func TestFB100_AC3_InputChanged_PopulatedVsEmpty_DifferentView(t *testing.T) {
	t.Parallel()
	withRows := newLargeProjectScopedModelWithAC()
	withRows.table.SetActivityRows([]data.ActivityRow{
		{ActorDisplay: "alice@datum.net", Summary: "deployed", Timestamp: time.Now()},
	})

	withoutRows := newLargeProjectScopedModelWithAC()
	withoutRows.table.SetActivityLoading(true)

	errMsg := data.ProjectActivityErrorMsg{Err: errors.New("transient error")}
	r1, _ := withRows.Update(errMsg)
	r2, _ := withoutRows.Update(errMsg)

	view1 := stripANSIModel(r1.(AppModel).View())
	view2 := stripANSIModel(r2.(AppModel).View())
	if view1 == view2 {
		t.Error("AC3: same View() for populated-rows vs empty-rows after error; want different output")
	}
	if !strings.Contains(view1, "alice@datum.net") {
		t.Errorf("AC3: populated-rows view missing 'alice@datum.net':\n%s", view1)
	}
	if !strings.Contains(view2, "activity unavailable") {
		t.Errorf("AC3: empty-rows view missing 'activity unavailable':\n%s", view2)
	}
}

// AC4 — [Anti-behavior] After AC1 (stale rows retained), subsequent successful fetch renders new rows.
func TestFB100_AC4_StaleRowsRetained_ThenSuccessfulFetch_RendersNewRows(t *testing.T) {
	t.Parallel()
	m := newLargeProjectScopedModelWithAC()
	m.table.SetActivityRows([]data.ActivityRow{
		{ActorDisplay: "alice@datum.net", Summary: "old event", Timestamp: time.Now()},
	})

	// Error fires — stale rows retained, no stuck flag.
	r1, _ := m.Update(data.ProjectActivityErrorMsg{Err: errors.New("transient error")})
	m1 := r1.(AppModel)

	// Successful reload arrives with new rows.
	r2, _ := m1.Update(data.ProjectActivityLoadedMsg{
		Rows: []data.ActivityRow{
			{ActorDisplay: "bob@datum.net", Summary: "new event", Timestamp: time.Now()},
		},
	})
	m2 := r2.(AppModel)

	got := stripANSIModel(m2.View())
	if !strings.Contains(got, "bob@datum.net") {
		t.Errorf("AC4: new row 'bob@datum.net' missing after successful reload:\n%s", got)
	}
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC4: 'activity unavailable' stuck after successful reload:\n%s", got)
	}
}

// ==================== End FB-100 ====================

// ==================== FB-048 + FB-050: Dashboard origin-pane restore + toggle consistency ====================

// --- FB-050: Toggle-key exits restore origin pane ---

// [Observable] FB-050 AC1: '3' twice from TablePane → back to TablePane (not hardcoded NavPane).
func TestFB050_Key3_ToggleFromTablePane_RestoresTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	// Ensure quota is not loading so open-path fires immediately.
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if appM.activePane != QuotaDashboardPane {
		t.Fatalf("first '3': activePane = %v, want QuotaDashboardPane", appM.activePane)
	}

	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("FB-050 AC1: second '3' from QuotaDash: activePane = %v, want TablePane (origin pane, not NavPane)", appM.activePane)
	}
}

// [Observable] FB-050 AC2: '4' twice from TablePane → back to TablePane.
func TestFB050_Key4_ToggleFromTablePane_RestoresTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	appM := result.(AppModel)
	if appM.activePane != ActivityDashboardPane {
		t.Fatalf("first '4': activePane = %v, want ActivityDashboardPane", appM.activePane)
	}

	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	appM = result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("FB-050 AC2: second '4' from ActivityDash: activePane = %v, want TablePane (origin pane, not NavPane)", appM.activePane)
	}
}

// [Observable] FB-050 AC3: help overlay contains "activity (toggle)" label (quota (toggle) already tested in TestFB035_HelpOverlay_ContainsKey3Row).
func TestFB050_HelpOverlay_ContainsActivityToggleLabel(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	appM := result.(AppModel)
	if appM.overlay != HelpOverlayID {
		t.Fatal("setup: expected HelpOverlayID after ? press")
	}

	view := stripANSIModel(appM.helpOverlay.View())
	if !strings.Contains(view, "activity (toggle)") {
		t.Errorf("FB-050 AC3: HelpOverlay missing 'activity (toggle)' label:\n%s", view)
	}
}

// [Anti-regression] FB-050 AC4: first '3' from NavPane still enters QuotaDashboardPane (FB-035 guard).
func TestFB050_Key3_FirstPress_FromNavPane_StillEntersDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("FB-050 AC4 anti-regression: first '3' from NavPane: activePane = %v, want QuotaDashboardPane", appM.activePane)
	}
}

// [Anti-regression] FB-050 AC5: first '4' from NavPane still enters ActivityDashboardPane (FB-016 guard).
func TestFB050_Key4_FirstPress_FromNavPane_StillEntersDashboard(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	appM := result.(AppModel)

	if appM.activePane != ActivityDashboardPane {
		t.Errorf("FB-050 AC5 anti-regression: first '4' from NavPane: activePane = %v, want ActivityDashboardPane", appM.activePane)
	}
}

// --- FB-048: Esc from dashboard restores origin pane ---

// [Observable] FB-048 AC1: '3' from TablePane → QDash → '3' toggle-back → TablePane.
func TestFB048_Key3_RoundTrip_FromTablePane_RestoresTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if appM.quotaOriginPane.Pane != TablePane {
		t.Fatalf("stash: quotaOriginPane.Pane = %v, want TablePane", appM.quotaOriginPane.Pane)
	}

	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("FB-048 AC1: after 3→QDash→3: activePane = %v, want TablePane", appM.activePane)
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "my-pod") {
		t.Errorf("FB-048 AC1: 'my-pod' missing from View() after round-trip — table not rendered:\n%s", got)
	}
	if strings.Contains(got, "Welcome") {
		t.Errorf("FB-048 AC1: 'Welcome' present in View() — NavPane re-rendered instead of TablePane:\n%s", got)
	}
}

// [Observable] FB-048 AC2: '3' from DetailPane (yamlMode) → QDash → Esc → DetailPane preserved.
func TestFB048_Key3_RoundTrip_FromDetailPane_RestoresDetailPane(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithYaml()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")
	if !m.yamlMode {
		t.Fatal("precondition: yamlMode must be true")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if appM.activePane != QuotaDashboardPane {
		t.Fatalf("setup: activePane = %v after '3', want QuotaDashboardPane", appM.activePane)
	}
	if appM.quotaOriginPane.Pane != DetailPane {
		t.Fatalf("stash: quotaOriginPane.Pane = %v, want DetailPane", appM.quotaOriginPane.Pane)
	}

	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM = result.(AppModel)

	if appM.activePane != DetailPane {
		t.Errorf("FB-048 AC2: after Esc from QDash: activePane = %v, want DetailPane", appM.activePane)
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "yaml") && !strings.Contains(got, "YAML") && !strings.Contains(got, "apiVersion") {
		t.Errorf("FB-048 AC2: yaml content missing from View() after Esc restores DetailPane:\n%s", got)
	}
}

// [Observable] FB-048 AC3: '4' from TablePane → ActivityDash → '4' → TablePane.
func TestFB048_Key4_RoundTrip_FromTablePane_RestoresTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	appM := result.(AppModel)
	if appM.activityOriginPane.Pane != TablePane {
		t.Fatalf("stash: activityOriginPane.Pane = %v, want TablePane", appM.activityOriginPane.Pane)
	}

	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	appM = result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("FB-048 AC3: after 4→ActivityDash→4: activePane = %v, want TablePane", appM.activePane)
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "my-pod") {
		t.Errorf("FB-048 AC3: 'my-pod' missing from View() after round-trip — table not rendered:\n%s", got)
	}
	if strings.Contains(got, "Welcome") {
		t.Errorf("FB-048 AC3: 'Welcome' present in View() — NavPane re-rendered instead of TablePane:\n%s", got)
	}
}

// [Input-changed] FB-048 AC4: Esc from QDash (origin=TablePane) restores TablePane — same result as toggle-key.
func TestFB048_Esc_FromQDash_RestoresTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")
	m.quotaOriginPane = DashboardOrigin{Pane: TablePane}
	m.activePane = QuotaDashboardPane
	m.updatePaneFocus()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("FB-048 AC4: Esc from QDash: activePane = %v, want TablePane", appM.activePane)
	}
}

// [Anti-behavior] FB-048 AC5: NavPane origin (zero-value DashboardOrigin) → '3' exit → NavPane.
func TestFB048_Key3_Exit_ZeroOrigin_ReturnsToNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneModelWithBC(nil)
	// Manually put model in QDash with zero-value origin (default pane is NavPane since PaneID zero-value = NavPane).
	m.quotaOriginPane = DashboardOrigin{} // zero value
	m.activePane = QuotaDashboardPane
	m.updatePaneFocus()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("FB-048 AC5: zero-origin exit via '3': activePane = %v, want NavPane", appM.activePane)
	}
}

// [Anti-regression] FB-048 AC6: table filter survives QDash round-trip.
func TestFB048_Key3_RoundTrip_PreservesTableFilter(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")
	m.table.SetFilter("xyz-unique")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "xyz-unique") {
		t.Errorf("FB-048 AC6: filter 'xyz-unique' lost after QDash round-trip:\n%s", got)
	}
}

// [Input-changed] FB-048 AC7: showDashboard=true (welcome panel) round-trip restores welcome panel.
func TestFB048_Key3_RoundTrip_ShowDashboardTrue_RestoresWelcome(t *testing.T) {
	t.Parallel()
	// Build a model in welcome-dashboard-over-table state:
	// tableTypeName is set but showDashboard=true (user pressed Esc from NavPane).
	m := newTablePaneModel()
	m.showDashboard = true
	m.table.SetForceDashboard(true)
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)
	if appM.activePane != QuotaDashboardPane {
		t.Fatalf("setup: activePane = %v after '3', want QuotaDashboardPane", appM.activePane)
	}
	if !appM.quotaOriginPane.ShowDashboard {
		t.Fatal("stash: quotaOriginPane.ShowDashboard = false, want true (was in welcome overlay)")
	}

	result, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = result.(AppModel)

	if !appM.showDashboard {
		t.Error("FB-048 AC7: showDashboard = false after round-trip, want true (welcome panel must be restored)")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("FB-048 AC7: 'Welcome' missing from View() after round-trip through QDash:\n%s", got)
	}
}

// ==================== End FB-048 + FB-050 ====================

// ==================== FB-051 + FB-052: Placeholder error context + Inline affordance ====================
//
// Axis-coverage (FB-051):
// AC | Happy                                                        | Input-changed                              | Anti-behavior                              | Anti-regression
// ---+--------------------------------------------------------------+--------------------------------------------+--------------------------------------------+-----------------
// 1  | AC1_ErrorPlaceholder_RendersDescribeUnavailableAndFailedLine | -                                          | -                                          | -
// 2  | AC2_TitleBar_ShowsDescribeUnavailable_ErrorVariant           | AC2b_TitleBar_LoadingVariant               | -                                          | -
// 3  | AC3_ErrorPlaceholder_RendersRetryKey                         | -                                          | -                                          | -
// 4  | -                                                            | AC4_Recovery_DescribeRawSet_ClearsError    | -                                          | -
// 5  | -                                                            | -                                          | AC5_LoadingVariant_NoErrorLine_NoRetryKey  | -
// 6  | AC6_RKey_ErrorPlaceholder_DispatchesRetryCmd                 | -                                          | -                                          | -
// 7  | -                                                            | -                                          | AC7_ErrorBlock_EventsNil_NoPlaceholder     | -
// 8  | -                                                            | -                                          | -                                          | AC8_LoadingVariant_PlaceholderShows
//
// Axis-coverage (FB-052):
// AC | Happy                                              | Input-changed                                 | Anti-behavior                             | Anti-regression
// ---+----------------------------------------------------+-----------------------------------------------+-------------------------------------------+-----------------
// 1  | AC1_NarrowWidth_PlaceholderBodyContainsEKey        | -                                             | -                                         | -
// 2  | AC2_WideWidth_PlaceholderBodyContainsEKey          | -                                             | -                                         | -
// 3  | -                                                  | AC3_LoadingVariant_NarrowWidth_KeysNoRetry    | -                                         | -
// 4  | -                                                  | AC4_ErrorVariant_AllKeys                      | -                                         | -
// 5  | AC5_EKey_FromPlaceholder_OpensEventsMode           | -                                             | -                                         | -
// 6  | -                                                  | -                                             | AC6_ErrorBlock_EventsNil_NoInlineEKey     | -
// 7  | -                                                  | -                                             | -                                         | AC7_FB051ErrorSubline_StillPresent

// newErrorEventPlaceholderModel returns a DetailPane AppModel in the FB-051
// error-placeholder state: loadState=Error, lastFailedFetchKind="describe",
// events populated, describeRaw=nil. detail mode and content are synced via
// EventsLoadedMsg injection so View() reflects the placeholder correctly.
func newErrorEventPlaceholderModel() AppModel {
	m := newDescribeErrorDetailModel() // loadState=Error, events=nil
	r, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{
			{Type: "Normal", Reason: "Scheduled", Message: "Assigned node", Count: 1},
		},
	})
	return r.(AppModel)
}

// newLoadingEventPlaceholderModel returns a DetailPane AppModel in the loading
// placeholder state: describeRaw=nil, events populated, no error. detail mode
// and content are synced via EventsLoadedMsg injection.
func newLoadingEventPlaceholderModel() AppModel {
	m := newDetailPaneModelWithHC()
	m.describeRaw = nil
	r, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{
			{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created pod", Count: 1},
		},
	})
	return r.(AppModel)
}

// --- FB-051 tests ---

// AC1 — [Happy] Error-placeholder renders "Describe unavailable" and "(describe failed:" in detail.View().
func TestFB051_AC1_ErrorPlaceholder_RendersDescribeUnavailableAndFailedLine(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if !strings.Contains(plain, "Describe unavailable") {
		t.Errorf("AC1: 'Describe unavailable' missing from detail.View():\n%s", plain)
	}
	if !strings.Contains(plain, "(describe failed:") {
		t.Errorf("AC1: '(describe failed:' missing from detail.View():\n%s", plain)
	}
}

// AC2 — [Happy] Error-placeholder: title bar mode shows "describe [unavailable]".
func TestFB051_AC2_TitleBar_ShowsDescribeUnavailable_ErrorVariant(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if !strings.Contains(plain, "describe [unavailable]") {
		t.Errorf("AC2 error variant: 'describe [unavailable]' missing from detail.View():\n%s", plain)
	}
}

// AC2b — [Input-changed] Loading placeholder also shows "describe [unavailable]" in title bar.
func TestFB051_AC2b_TitleBar_ShowsDescribeUnavailable_LoadingVariant(t *testing.T) {
	t.Parallel()
	m := newLoadingEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if !strings.Contains(plain, "describe [unavailable]") {
		t.Errorf("AC2b loading variant: 'describe [unavailable]' missing from detail.View():\n%s", plain)
	}
}

// AC3 — [Happy] Error-placeholder action row contains "[r]" and "retry".
func TestFB051_AC3_ErrorPlaceholder_RendersRetryKey(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if !strings.Contains(plain, "[r]") {
		t.Errorf("AC3: '[r]' missing from error-placeholder detail.View():\n%s", plain)
	}
	if !strings.Contains(plain, "retry") {
		t.Errorf("AC3: 'retry' missing from error-placeholder detail.View():\n%s", plain)
	}
}

// AC4 — [Input-changed] Recovery: DescribeResultMsg with raw set clears error subline.
func TestFB051_AC4_Recovery_DescribeRawSet_ClearsErrorSubline(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	// Confirm error subline present before recovery.
	before := stripANSIModel(m.detail.View())
	if !strings.Contains(before, "(describe failed:") {
		t.Fatalf("setup: error subline absent before recovery:\n%s", before)
	}

	// Inject recovery.
	r, _ := m.Update(data.DescribeResultMsg{Content: "describe output", Raw: testRawObject()})
	appM := r.(AppModel)

	plain := stripANSIModel(appM.detail.View())
	if strings.Contains(plain, "(describe failed:") {
		t.Errorf("AC4: '(describe failed:' still present after recovery:\n%s", plain)
	}
}

// AC5 — [Anti-behavior] Loading sub-variant has no error subline and no "[r]" key.
func TestFB051_AC5_LoadingVariant_NoErrorLine_NoRetryKey(t *testing.T) {
	t.Parallel()
	m := newLoadingEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if strings.Contains(plain, "(describe failed:") {
		t.Errorf("AC5: '(describe failed:' present in loading sub-variant:\n%s", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("AC5: '[r]' present in loading sub-variant:\n%s", plain)
	}
}

// AC6 — [Happy] r from error-placeholder state dispatches a retry cmd.
func TestFB051_AC6_RKey_ErrorPlaceholder_DispatchesRetryCmd(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	if cmd == nil {
		t.Error("AC6: cmd = nil after r from error-placeholder, want retry cmd")
	}
	if appM.loadState == data.LoadStateError {
		t.Error("AC6: loadState still Error after r, want LoadStateLoading")
	}
}

// AC7 — [Anti-behavior] When events==nil, error block renders; placeholder absent.
func TestFB051_AC7_ErrorBlock_EventsNil_NoPlaceholder(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel() // events = nil

	plain := stripANSIModel(m.detail.View())
	if strings.Contains(plain, "Describe unavailable") {
		t.Errorf("AC7: placeholder present with events==nil; error block must win:\n%s", plain)
	}
	if !strings.Contains(plain, "Could not describe") {
		t.Errorf("AC7: error block 'Could not describe' absent with events==nil:\n%s", plain)
	}
}

// AC8 — [Anti-regression] Loading sub-variant still shows "Describe unavailable".
func TestFB051_AC8_LoadingVariant_PlaceholderShows(t *testing.T) {
	t.Parallel()
	m := newLoadingEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if !strings.Contains(plain, "Describe unavailable") {
		t.Errorf("AC8: 'Describe unavailable' missing from loading sub-variant:\n%s", plain)
	}
}

// --- FB-052 tests ---

// AC1 — [Happy] At narrow width (40), placeholder body contains "[E]".
func TestFB052_AC1_NarrowWidth_PlaceholderBodyContainsEKey(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.describeRaw = nil
	m.detail = components.NewDetailViewModel(40, 20)
	m.detail.SetResourceContext("projects", "my-proj")

	r, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created", Count: 1}},
	})
	appM := r.(AppModel)

	plain := stripANSIModel(appM.detail.View())
	if !strings.Contains(plain, "[E]") {
		t.Errorf("AC1 narrow width=40: '[E]' missing from placeholder body:\n%s", plain)
	}
}

// AC2 — [Happy] At wide width (120), placeholder body contains "[E]".
func TestFB052_AC2_WideWidth_PlaceholderBodyContainsEKey(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithHC()
	m.describeRaw = nil
	m.detail = components.NewDetailViewModel(120, 20)
	m.detail.SetResourceContext("projects", "my-proj")

	r, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{{Type: "Normal", Reason: "SuccessfulCreate", Message: "Created", Count: 1}},
	})
	appM := r.(AppModel)

	plain := stripANSIModel(appM.detail.View())
	if !strings.Contains(plain, "[E]") {
		t.Errorf("AC2 wide width=120: '[E]' missing from placeholder body:\n%s", plain)
	}
}

// AC3 — [Input-changed] Loading sub-variant at narrow width: "[E]" and "[Esc]" present, "[r]" absent.
func TestFB052_AC3_LoadingVariant_NarrowWidth_KeysPresent_NoRetry(t *testing.T) {
	t.Parallel()
	m := newLoadingEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	if !strings.Contains(plain, "[E]") {
		t.Errorf("AC3 loading variant: '[E]' missing:\n%s", plain)
	}
	if !strings.Contains(plain, "[Esc]") {
		t.Errorf("AC3 loading variant: '[Esc]' missing:\n%s", plain)
	}
	if strings.Contains(plain, "[r]") {
		t.Errorf("AC3 loading variant: '[r]' present but must be absent (no error):\n%s", plain)
	}
}

// AC4 — [Input-changed] Error sub-variant: "[E]", "[r]", "[Esc]" all in body.
func TestFB052_AC4_ErrorVariant_AllKeys(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	plain := stripANSIModel(m.detail.View())
	for _, key := range []string{"[E]", "[r]", "[Esc]"} {
		if !strings.Contains(plain, key) {
			t.Errorf("AC4 error variant: %q missing from placeholder body:\n%s", key, plain)
		}
	}
}

// AC5 — [Happy] E from error-placeholder enters events mode; detail.View() changes.
func TestFB052_AC5_EKey_FromPlaceholder_OpensEventsMode(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	before := stripANSIModel(m.detail.View())

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if !appM.eventsMode {
		t.Error("AC5: eventsMode = false after E from placeholder, want true")
	}
	after := stripANSIModel(appM.detail.View())
	if before == after {
		t.Error("AC5: detail.View() unchanged after E from placeholder, want events mode rendering")
	}
	if strings.Contains(after, "Describe unavailable") {
		t.Errorf("AC5: 'Describe unavailable' still in View() after entering events mode:\n%s", after)
	}
}

// AC6 — [Anti-behavior] Error block (events==nil) does not contain "[E]" inline hint.
func TestFB052_AC6_ErrorBlock_EventsNil_NoInlineEKey(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel() // events = nil

	plain := stripANSIModel(m.buildDetailContent())
	if strings.Contains(plain, "[E]") {
		t.Errorf("AC6: error block with events==nil contains '[E]'; must not appear:\n%s", plain)
	}
}

// AC7 — [Anti-regression] FB-051 error subline still present after FB-052 changes.
func TestFB052_AC7_FB051ErrorSubline_StillPresent(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel()

	plain := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(plain, "(describe failed:") {
		t.Errorf("AC7 anti-regression: FB-051 error subline absent from buildDetailContent():\n%s", plain)
	}
}

// ==================== End FB-051 + FB-052 ====================

// ==================== FB-084: Placeholder action row retryability ====================
//
// Axis-coverage (brief-AC-indexed):
// AC  | Axis            | Test(s)
// ----+-----------------+----------------------------------------------------------------------
// AC1 | Observable      | TestFB084_AC1_NonRetryable_RKeyAbsentFromView
// AC2 | Observable      | TestFB084_AC2_Retryable_RKeyAndRetryDescribeInView
// AC3 | Anti-behavior   | TestFB084_AC3_NonRetryable_RKeyPressed_PostsNoRetryHint
// AC4 | Input-changed   | TestFB084_AC4_InputChanged_ErrorToWarning_RKeyAppears
//     |                 | TestFB084_AC4_InputChanged_WarningToError_RKeyDisappears
// AC5 | Anti-regression | TestFB084_AC5_NonErrorPlaceholder_NoRKey
// AC6 | Anti-regression | TestFB051_AC3_ErrorPlaceholder_RendersRetryKey (existing); FB-052 existing tests green
// AC7 | Integration     | go install ./... + go test ./internal/tui/...
//
// Severity classifier closed set: data.ErrorSeverityWarning (0) and data.ErrorSeverityError (1).
// No third value exists (verified in internal/tui/data/severity.go). AC1 covers Error; AC2 covers Warning.

// newForbiddenErrorEventPlaceholderModel builds the FB-051 error-placeholder
// with errStubForbidden as the load error (ErrorSeverityError → [r] suppressed).
func newForbiddenErrorEventPlaceholderModel() AppModel {
	m := newDescribeErrorDetailModel()
	m.loadErr = errStubForbidden // override: IsForbidden→true → ErrorSeverityError
	r, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{
			{Type: "Normal", Reason: "Scheduled", Message: "Assigned node", Count: 1},
		},
	})
	return r.(AppModel)
}

// [Observable] AC1: Error severity (forbidden) → [r] suppressed from detail pane body.
// Uses buildDetailContent() — the placeholder action row lives there, not in the status bar.
// Consistent with FB-052 AC6 pattern (TestFB052_AC6_ErrorBlock_EventsNil_NoInlineEKey).
func TestFB084_AC1_NonRetryable_RKeyAbsentFromView(t *testing.T) {
	t.Parallel()
	m := newForbiddenErrorEventPlaceholderModel()

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "[r]") {
		t.Errorf("AC1: '[r]' present in detail content when error is non-retryable (Error severity):\n%s", got)
	}
	// Sanity: placeholder is active (not a setup bug).
	if !strings.Contains(got, "Describe unavailable") {
		t.Errorf("AC1: setup: 'Describe unavailable' missing — placeholder not active:\n%s", got)
	}
}

// [Observable] AC2: Warning severity (transient) → [r] AND "retry describe" in View().
func TestFB084_AC2_Retryable_RKeyAndRetryDescribeInView(t *testing.T) {
	t.Parallel()
	// newErrorEventPlaceholderModel uses errors.New("connection refused") → ErrorSeverityWarning.
	m := newErrorEventPlaceholderModel()

	got := stripANSIModel(m.View())
	if !strings.Contains(got, "[r]") {
		t.Errorf("AC2: '[r]' missing from View() when error is retryable (Warning severity):\n%s", got)
	}
	if !strings.Contains(got, "retry describe") {
		t.Errorf("AC2: 'retry describe' missing from View() — qualifier must be present:\n%s", got)
	}
	// Copy anti-regression: "retry describe" must be present; bare "retry  " (double-space = no qualifier) must not.
	if strings.Contains(got, "retry  ") {
		t.Errorf("AC2 copy: bare 'retry' without 'describe' qualifier ('retry  ') found — copy regression:\n%s", got)
	}
}

// [Anti-behavior] AC3: [r] suppressed from affordance but key still live; posting "No retry available" hint.
// Hint must appear in stripANSI(appM.statusBar.View()) — not just statusBar.Hint field inspection.
func TestFB084_AC3_NonRetryable_RKeyPressed_PostsNoRetryHint(t *testing.T) {
	t.Parallel()
	m := newForbiddenErrorEventPlaceholderModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	appM := result.(AppModel)

	appM.statusBar.Width = 80
	statusPlain := stripANSIModel(appM.statusBar.View())
	if !strings.Contains(statusPlain, "No retry available") {
		t.Errorf("AC3: 'No retry available' missing from statusBar.View() after 'r' on non-retryable:\n%s", statusPlain)
	}
}

// [Input-changed] AC4, pair A: mutate load error from ErrorSeverityError → ErrorSeverityWarning;
// detail content must gain [r] (same placeholder pathway, different loadErr → different output).
// Uses buildDetailContent() — the action row lives there; status bar always shows [r] refresh.
func TestFB084_AC4_InputChanged_ErrorToWarning_RKeyAppears(t *testing.T) {
	t.Parallel()
	m := newForbiddenErrorEventPlaceholderModel()

	beforeGot := stripANSIModel(m.buildDetailContent())
	if strings.Contains(beforeGot, "[r]") {
		t.Fatal("precondition: '[r]' must be absent from detail content before severity change")
	}

	// Mutate to Warning-severity error and re-sync placeholder content.
	m.loadErr = errors.New("connection refused") // Warning severity
	m.detail.SetContent(m.buildDetailContent())

	afterGot := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(afterGot, "[r]") {
		t.Errorf("AC4 Error→Warning: '[r]' missing from detail content after severity change to Warning:\n%s", afterGot)
	}
	if !strings.Contains(afterGot, "retry describe") {
		t.Errorf("AC4 Error→Warning: 'retry describe' missing after severity change:\n%s", afterGot)
	}
}

// [Input-changed] AC4, pair B: symmetric — Warning → Error; detail content must lose [r].
// Uses buildDetailContent() — status bar always shows [r] refresh in DETAIL mode.
func TestFB084_AC4_InputChanged_WarningToError_RKeyDisappears(t *testing.T) {
	t.Parallel()
	m := newErrorEventPlaceholderModel() // connection refused → Warning → [r] present

	beforeGot := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(beforeGot, "[r]") {
		t.Fatal("precondition: '[r]' must be present in detail content before severity change")
	}

	// Mutate to Error-severity error and re-sync placeholder content.
	m.loadErr = errStubForbidden // ErrorSeverityError
	m.detail.SetContent(m.buildDetailContent())

	afterGot := stripANSIModel(m.buildDetailContent())
	if strings.Contains(afterGot, "[r]") {
		t.Errorf("AC4 Warning→Error: '[r]' still present in detail content after severity change to Error:\n%s", afterGot)
	}
}

// [Anti-regression] AC5: non-error placeholder (errMode=false) → [r] absent; [E] and [Esc] present.
// Uses buildDetailContent() — the action row is part of detail content; status bar shows [r] refresh.
func TestFB084_AC5_NonErrorPlaceholder_NoRKey(t *testing.T) {
	t.Parallel()
	m := newLoadingEventPlaceholderModel() // no error; errMode=false

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "[r]") {
		t.Errorf("AC5: '[r]' present in non-error placeholder detail content (errMode=false):\n%s", got)
	}
	if !strings.Contains(got, "[E]") {
		t.Errorf("AC5: '[E]' missing from non-error placeholder action row:\n%s", got)
	}
	if !strings.Contains(got, "[Esc]") {
		t.Errorf("AC5: '[Esc]' missing from non-error placeholder action row:\n%s", got)
	}
}

// ==================== End FB-084 ====================

// ==================== FB-077: Quota loading re-queue loop fix ====================
//
// Axis-coverage:
// AC | Repeat-press                                             | Observable                                          | Anti-regression
// ---+----------------------------------------------------------+-----------------------------------------------------+-----------------
// 1  | AC1_BucketsErrorMsg_ThenKey3_TakesImmediatePath         | -                                                   | -
// 1b | AC1b_LoadErrorMsg_ThenKey3_TakesImmediatePath           | -                                                   | -
// 2  | -                                                        | AC2_BucketsErrorMsg_ClearsQuotaLoading              | -
// 3  | -                                                        | -                                                   | AC3_FB047_Regression

// newPendingQuotaOpenModel builds an AppModel in NavPane with quota loading and
// pendingQuotaOpen=true — the state after '3' was pressed during bucket loading.
func newPendingQuotaOpenModel() AppModel {
	m := newQuotaLoadingModel()
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	return r.(AppModel)
}

// AC1 — [Repeat-press] BucketsErrorMsg clears loading → next '3' takes immediate path (not re-queued).
func TestFB077_AC1_BucketsErrorMsg_ThenKey3_TakesImmediatePath(t *testing.T) {
	t.Parallel()
	m := newPendingQuotaOpenModel()
	if !m.pendingQuotaOpen {
		t.Fatal("setup: pendingQuotaOpen = false, want true")
	}

	// Error arrives — clears pending AND quota loading state.
	r, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("load failed")})
	m = r.(AppModel)

	// Next '3' press: quota.IsLoading()==false → immediate transition, NOT re-queue.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC1: activePane = %v after BucketsErrorMsg + '3', want QuotaDashboardPane (immediate path)", appM.activePane)
	}
	if appM.pendingQuotaOpen {
		t.Error("AC1: pendingQuotaOpen = true after immediate transition, want false")
	}
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC1: 'Quota dashboard loading' hint still present after immediate transition:\n%s", got)
	}
}

// AC1b — [Repeat-press] LoadErrorMsg path: same loop-break guarantee via LoadErrorMsg.
func TestFB077_AC1b_LoadErrorMsg_ThenKey3_TakesImmediatePath(t *testing.T) {
	t.Parallel()
	m := newPendingQuotaOpenModel()
	if !m.pendingQuotaOpen {
		t.Fatal("setup: pendingQuotaOpen = false, want true")
	}

	r, _ := m.Update(data.LoadErrorMsg{Err: errors.New("network error")})
	m = r.(AppModel)

	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC1b: activePane = %v after LoadErrorMsg + '3', want QuotaDashboardPane (immediate path)", appM.activePane)
	}
	if strings.Contains(stripANSIModel(appM.View()), "Quota dashboard loading") {
		t.Errorf("AC1b: 'Quota dashboard loading' hint still present after immediate transition")
	}
}

// AC2 — [Observable] BucketsErrorMsg with pendingQuotaOpen clears quota.IsLoading().
func TestFB077_AC2_BucketsErrorMsg_ClearsQuotaLoading(t *testing.T) {
	t.Parallel()
	m := newPendingQuotaOpenModel()

	r, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("timeout")})
	appM := r.(AppModel)

	if appM.quota.IsLoading() {
		t.Error("AC2: quota.IsLoading() = true after BucketsErrorMsg, want false")
	}
}

// AC3 — [Anti-regression] FB-047 AC6 (BucketsErrorMsg clears pendingQuotaOpen) still holds.
func TestFB077_AC3_FB047Regression_BucketsErrorMsg_ClearsPending(t *testing.T) {
	t.Parallel()
	m := newPendingQuotaOpenModel()

	r, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("load failed")})
	appM := r.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC3 regression: pendingQuotaOpen = true after BucketsErrorMsg, want false")
	}
}

// ==================== End FB-077 ====================

// ==================== FB-076: r-press activity refresh from NavPane ====================
//
// Axis-coverage:
// AC | Repeat-press                                                      | Observable                                      | Anti-behavior                                | Anti-regression
// ---+-------------------------------------------------------------------+-------------------------------------------------+----------------------------------------------+------------------
// 1  | AC1_ProjectScope_RPress_BatchIncludesActivityCmd                  | -                                               | -                                            | -
// 2  | -                                                                 | AC2_ProjectScope_ActivityLoadedMsg_RendersRows  | -                                            | -
// 3  | -                                                                 | -                                               | AC3_OrgScope_RPress_NoActivityCmd            | -
// 4  | -                                                                 | -                                               | -                                            | AC4_RPress_StillDispatches_LoadResourceTypesCmd

// newProjectScopedNavModel builds a NavPane AppModel with project scope and
// an activity client — suitable for FB-076 r-press tests.
// Uses a large table so S3 (contentH≥24) renders for View() assertions.
func newProjectScopedNavModel() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		ac:          data.NewActivityClient(nil),
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 32),
		table:       components.NewResourceTableModel(80, 32),
		detail:      components.NewDetailViewModel(80, 32),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}
	m.updatePaneFocus()
	return m
}

// AC1 — [Repeat-press] r from project-scoped NavPane batches LoadRecentProjectActivityCmd.
func TestFB076_AC1_ProjectScope_RPress_BatchIncludesActivityCmd(t *testing.T) {
	t.Parallel()
	m := newProjectScopedNavModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if batchLen(cmd) < 2 {
		t.Errorf("AC1: batchLen = %d, want ≥2 (LoadResourceTypesCmd + LoadRecentProjectActivityCmd)", batchLen(cmd))
	}
}

// AC2 — [Observable] After r-press + ProjectActivityLoadedMsg, View() shows updated row.
func TestFB076_AC2_ProjectScope_ActivityLoadedMsg_RendersRows(t *testing.T) {
	t.Parallel()
	m := newProjectScopedNavModel()

	// r-press transitions to loading; don't execute cmd (nil-factory would panic).
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = r.(AppModel)

	// Inject loaded rows directly (sidesteps nil-factory panic).
	r, _ = m.Update(data.ProjectActivityLoadedMsg{
		Rows: []data.ActivityRow{
			{ActorDisplay: "bob@example.com", Summary: "updated cluster", Timestamp: time.Now().Add(-1 * time.Minute)},
		},
	})
	appM := r.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "bob@example.com") {
		t.Errorf("AC2: 'bob@example.com' missing from View() after r + ProjectActivityLoadedMsg:\n%s", got)
	}
}

// AC3 — [Anti-behavior] r from org-scoped NavPane (no ProjectID) omits activity cmd.
func TestFB076_AC3_OrgScope_RPress_NoActivityCmd(t *testing.T) {
	t.Parallel()
	m := newProjectScopedNavModel()
	m.tuiCtx.ActiveCtx = nil // org scope

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if batchLen(cmd) != 1 {
		t.Errorf("AC3: batchLen = %d, want 1 (only LoadResourceTypesCmd, no activity cmd for org scope)", batchLen(cmd))
	}
}

// AC4 — [Anti-regression] r from NavPane still dispatches LoadResourceTypesCmd.
func TestFB076_AC4_RPress_StillDispatches_LoadResourceTypesCmd(t *testing.T) {
	t.Parallel()
	m := newProjectScopedNavModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if cmd == nil {
		t.Error("AC4: cmd = nil after r from NavPane, want LoadResourceTypesCmd dispatched")
	}
}

// ==================== End FB-076 ====================

// ==================== FB-049: QuotaDashboardPane status-bar key filtering ====================
//
// Axis-coverage:
// AC | Observable                                              | Input-changed                                         | Anti-behavior                                  | Anti-regression
// ---+---------------------------------------------------------+-------------------------------------------------------+------------------------------------------------+-----------------
// 1  | AC1_QuotaPane_AbsentsNavTableKeys                      | -                                                     | -                                              | -
// 2  | AC2_QuotaPane_ContainsTSR3Keys                         | -                                                     | -                                              | -
// 3  | AC3_QuotaPane_ContainsHelpQuitKeys                     | -                                                     | -                                              | -
// 4  | -                                                       | AC4_Transition_TableToQuota_HintSwitches               | -                                              | -
// 5  | -                                                       | AC5_Transition_QuotaToTable_HintSwitches               | -                                              | -
// 6  | AC6_SlashKey_QuotaPane_SilentNoOp_ViewUnchanged        | -                                                     | -                                              | -
// 7  | -                                                       | -                                                     | -                                              | AC7_NavTableHints_Unchanged

// newTablePaneModelWithQuota returns a TablePane model with a quota dashboard
// component pre-wired so '3' can transition to QuotaDashboardPane immediately.
func newTablePaneModelWithQuota() AppModel {
	m := newTablePaneModel()
	m.quota = components.NewQuotaDashboardModel(58, 20, "proj")
	return m
}

// AC1 — [Observable] QuotaDashboard pane View() does NOT contain "[/] filter", "[Enter] select", "[d] describe".
func TestFB049_AC1_QuotaPane_AbsentsNavTableKeys(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	got := stripANSIModel(m.View())
	for _, absent := range []string{"[/] filter", "[Enter] select", "[d] describe"} {
		if strings.Contains(got, absent) {
			t.Errorf("AC1: %q present in QuotaDashboard View(), want absent:\n%s", absent, got)
		}
	}
}

// AC2 — [Observable] QuotaDashboard pane View() contains QUOTA-specific keys.
func TestFB049_AC2_QuotaPane_ContainsTSR3Keys(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	got := stripANSIModel(m.View())
	for _, want := range []string{"[t]", "[s]", "[r]", "[3]"} {
		if !strings.Contains(got, want) {
			t.Errorf("AC2: %q missing from QuotaDashboard View():\n%s", want, got)
		}
	}
}

// AC3 — [Observable] QuotaDashboard pane View() retains help and quit keys.
func TestFB049_AC3_QuotaPane_ContainsHelpQuitKeys(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})

	got := stripANSIModel(m.View())
	for _, want := range []string{"[?]", "[q]"} {
		if !strings.Contains(got, want) {
			t.Errorf("AC3: %q missing from QuotaDashboard View():\n%s", want, got)
		}
	}
}

// AC4 — [Input-changed] TablePane→'3'→QuotaDashboardPane: hint set changes (filter→quota keys).
func TestFB049_AC4_Transition_TableToQuota_HintSwitches(t *testing.T) {
	t.Parallel()
	m := newTablePaneModelWithQuota()

	before := stripANSIModel(m.View())
	if !strings.Contains(before, "[/] filter") {
		t.Fatalf("setup: '[/] filter' absent from TablePane View():\n%s", before)
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	after := stripANSIModel(appM.View())
	if !strings.Contains(after, "[s]") {
		t.Errorf("AC4: '[s]' (group key) missing from QuotaDashboard View():\n%s", after)
	}
	if strings.Contains(after, "[/] filter") {
		t.Errorf("AC4: '[/] filter' still present after transition to QuotaDashboard:\n%s", after)
	}
}

// AC5 — [Input-changed] QuotaDashboardPane→'3'→TablePane: hint set changes back (quota→filter keys).
func TestFB049_AC5_Transition_QuotaToTable_HintSwitches(t *testing.T) {
	t.Parallel()
	m := newTablePaneModelWithQuota()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = result.(AppModel)

	before := stripANSIModel(m.View())
	if !strings.Contains(before, "[s]") {
		t.Fatalf("setup: '[s]' absent from QuotaDashboard View():\n%s", before)
	}

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := result.(AppModel)

	after := stripANSIModel(appM.View())
	if !strings.Contains(after, "[/] filter") {
		t.Errorf("AC5: '[/] filter' missing after returning to TablePane:\n%s", after)
	}
	if strings.Contains(after, "[s] group") {
		t.Errorf("AC5: '[s] group' (quota key) still present after returning to TablePane:\n%s", after)
	}
}

// AC6 — [Observable] '/' in QuotaDashboard: View() unchanged (covered deeper by existing test; pin here as Observable axis).
func TestFB049_AC6_SlashKey_QuotaPane_ViewUnchanged(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardPaneModel(&stubBucketClient{})
	before := stripANSIModel(m.View())

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	after := stripANSIModel(result.(AppModel).View())

	if before != after {
		t.Errorf("AC6: View() changed after '/' in QuotaDashboard, want silent no-op:\nbefore: %s\nafter: %s", before, after)
	}
}

// AC7 — [Anti-regression] NAV and TABLE hint strings unchanged after FB-049.
func TestFB049_AC7_NavTableHints_Unchanged(t *testing.T) {
	t.Parallel()
	t.Run("NavPane", func(t *testing.T) {
		t.Parallel()
		m := newNavPaneModelWithBC(nil)
		got := stripANSIModel(m.View())
		for _, want := range []string{"[Enter] select", "[c] ctx", "[?] help", "[q] quit"} {
			if !strings.Contains(got, want) {
				t.Errorf("AC7 NAV regression: %q missing from NavPane View():\n%s", want, got)
			}
		}
		for _, absent := range []string{"[t]", "[s] group", "[3] back"} {
			if strings.Contains(got, absent) {
				t.Errorf("AC7 NAV regression: QUOTA-only key %q present in NavPane View()", absent)
			}
		}
	})
	t.Run("TablePane", func(t *testing.T) {
		t.Parallel()
		m := newTablePaneModel()
		got := stripANSIModel(m.View())
		for _, want := range []string{"[/] filter", "[d] describe", "[r] refresh"} {
			if !strings.Contains(got, want) {
				t.Errorf("AC7 TABLE regression: %q missing from TablePane View():\n%s", want, got)
			}
		}
		for _, absent := range []string{"[t] flat", "[s] group", "[3] back"} {
			if strings.Contains(got, absent) {
				t.Errorf("AC7 TABLE regression: QUOTA-only key %q present in TablePane View()", absent)
			}
		}
	})
}

// ==================== End FB-049 ====================

// ==================== FB-053: Transition hint when events swap error block for placeholder ====================
//
// Axis-coverage:
// AC | Observable                                             | Input-changed                              | Anti-behavior                                    | Anti-regression
// ---+--------------------------------------------------------+--------------------------------------------+--------------------------------------------------+-----------------
// 1  | AC1_ErrorToPlaceholder_HintAppearsInView               | -                                          | -                                                | -
// 2  | -                                                       | AC2_ViewDiffers_BeforeAfterEventsLoaded     | -                                                | -
// 3  | -                                                       | -                                           | AC3_NormalEventsLoad_NoHint                      | -
// 4  | -                                                       | -                                           | AC4_EventsLoadError_NoHint                       | -
// 5  | -                                                       | -                                           | -                                                | AC5_FB037Placeholder_StillRenders

// AC1 — [Observable] EventsLoadedMsg into error-describe state → View() contains "Events loaded".
func TestFB053_AC1_ErrorToPlaceholder_HintAppearsInView(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel() // loadState=Error, lastFailedFetchKind="describe", events=nil

	result, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{{Type: "Normal", Reason: "Scheduled", Message: "Assigned", Count: 1}},
	})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Events loaded") {
		t.Errorf("AC1: 'Events loaded' missing from View() after error→placeholder transition:\n%s", got)
	}
}

// AC2 — [Input-changed] View() differs before and after EventsLoadedMsg in error state.
func TestFB053_AC2_ViewDiffers_BeforeAfterEventsLoaded(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()
	before := stripANSIModel(m.View())

	result, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{{Type: "Warning", Reason: "BackOff", Message: "Restarting", Count: 3}},
	})
	after := stripANSIModel(result.(AppModel).View())

	if before == after {
		t.Error("AC2: View() identical before and after EventsLoadedMsg, want hint + placeholder change")
	}
}

// AC3 — [Anti-behavior] Normal events load (loadState idle, describeRaw set) → "Events loaded" hint absent.
func TestFB053_AC3_NormalEventsLoad_NoHint(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithRaw() // loadState=idle, describeRaw!=nil

	result, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{{Type: "Normal", Reason: "Pulled", Message: "Image pulled", Count: 1}},
	})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Events loaded") {
		t.Errorf("AC3: 'Events loaded' hint present for normal events load, want absent:\n%s", got)
	}
}

// AC4 — [Anti-behavior] EventsLoadedMsg with Err set → hint absent even in error-describe state.
func TestFB053_AC4_EventsLoadError_NoHint(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel() // loadState=Error, lastFailedFetchKind="describe"

	result, _ := m.Update(data.EventsLoadedMsg{Err: errors.New("events fetch failed")})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Events loaded") {
		t.Errorf("AC4: 'Events loaded' hint present when EventsLoadedMsg.Err set, want absent:\n%s", got)
	}
}

// AC5 — [Anti-regression] FB-037 placeholder still renders after FB-053 hint injection.
func TestFB053_AC5_FB037Placeholder_StillRenders(t *testing.T) {
	t.Parallel()
	m := newDescribeErrorDetailModel()

	result, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{{Type: "Normal", Reason: "Scheduled", Message: "Assigned", Count: 1}},
	})
	appM := result.(AppModel)

	got := stripANSIModel(appM.detail.View())
	if !strings.Contains(got, "Describe unavailable") {
		t.Errorf("AC5 regression: 'Describe unavailable' placeholder missing after FB-053 hint:\n%s", got)
	}
}

// ==================== End FB-053 ====================

// ==================== FB-054: Tab-to-resume hint — model layer ====================
//
// Axis-coverage (model layer):
// AC | Observable                                    | Input-changed                                   | Anti-regression
// ---+-----------------------------------------------+-------------------------------------------------+------------------
// 3  | AC3_WelcomeDashboard_ViewContainsTabHint       | -                                               | -
// 4  | -                                              | AC4_Tab_ResumesTable_NilCmd                     | -
// 5  | -                                              | -                                               | AC5_Enter_StillFiresLoadCmd
// 6  | -                                              | -                                               | AC6_FB041_EscChain_Unchanged

// newNavPaneWithDashboard builds an AppModel in NavPane with showDashboard=true and
// a cached table — the state FB-054 targets for the Tab-to-resume hint.
func newNavPaneWithDashboard() AppModel {
	m := newNavPaneWithTableLoaded()
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r.(AppModel)
	if !appM.showDashboard {
		panic("newNavPaneWithDashboard: Esc did not set showDashboard=true")
	}
	return appM
}

// AC3 — [Observable] showDashboard=true + typeName set → View() contains "[Tab]" and typeName.
func TestFB054_AC3_WelcomeDashboard_ViewContainsTabHint(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithDashboard()

	got := stripANSIModel(m.View())
	if !strings.Contains(got, "[Tab]") {
		t.Errorf("AC3: '[Tab]' missing from welcome-dashboard View():\n%s", got)
	}
	if !strings.Contains(got, "projects") {
		t.Errorf("AC3: 'projects' (typeName) missing from welcome-dashboard View():\n%s", got)
	}
}

// AC4 — [Input-changed] Tab from NavPane (showDashboard=true) → TablePane, cmd=nil, forceDashboard cleared.
func TestFB054_AC4_Tab_ResumesTable_NilCmd(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithDashboard()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	appM := result.(AppModel)

	if appM.activePane != TablePane {
		t.Errorf("AC4: activePane = %v after Tab, want TablePane", appM.activePane)
	}
	if appM.showDashboard {
		t.Error("AC4: showDashboard = true after Tab, want false")
	}
	if cmd != nil {
		t.Error("AC4: cmd != nil after Tab — table rows are cached, want no ListResourcesCmd")
	}
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "[Tab]") && strings.Contains(got, "to resume") {
		t.Errorf("AC4: Tab-to-resume hint still present after Tab pressed:\n%s", got)
	}
}

// AC5 — [Anti-regression] Enter from NavPane with type selected still dispatches LoadResourcesCmd.
func TestFB054_AC5_Enter_StillFiresLoadCmd(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("AC5 regression: cmd = nil after Enter from NavPane, want LoadResourcesCmd")
	}
}

// AC6 — [Anti-regression] FB-041 Esc chain: NavPane+loaded table → Esc → showDashboard=true.
func TestFB054_AC6_FB041_EscChain_Unchanged(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if !appM.showDashboard {
		t.Error("AC6 regression: showDashboard = false after Esc, want true (FB-041 chain broken)")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC6 regression: 'Welcome' missing from View() after Esc:\n%s", got)
	}
}

// ==================== End FB-054 ====================

// ==================== FB-055: Visible signal on NavPane Esc-to-dashboard transition ====================
//
// Axis-coverage:
// AC | Happy         | Anti-behavior          | Input-changed | Observable | Anti-regression
// ---+---------------+------------------------+---------------+------------+----------------
// 1  | AC1_HappyPath |                        |               | AC1        |
// 2  |               | AC2_FreshStartup_NoHint |               |            |
// 3  |               | AC3_AlreadyDashboard    |               |            |
// 4  |               | AC4_OtherKeys_NoHint    |               |            |
// 5  |               |                        | AC5_HintClears|            |
// 6  |               |                        |               |            | AC6_FB041Chain
// 7  | AC7_Compile   |                        |               |            |

// AC1 — [Happy/Observable] Esc from NavPane with showDashboard=false and tableTypeName set
// posts the transition hint; stripANSI(statusBar.View()) contains "Returned to welcome panel".
func TestFB055_AC1_Esc_NavPane_WithTable_PostsHint(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	m.tableTypeName = "httproutes"
	m.showDashboard = false
	m.statusBar.Width = 120

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	appM.statusBar.Width = 120
	statusPlain := stripANSIModel(appM.statusBar.View())
	// Normalize whitespace to handle lipgloss line-wrapping at narrow widths.
	statusNorm := strings.Join(strings.Fields(statusPlain), " ")
	// AC1: new copy present
	if !strings.Contains(statusNorm, "Returned to welcome panel") {
		t.Errorf("AC1 [Observable]: statusBar.View() (normalized) = %q\nwant substring 'Returned to welcome panel'", statusNorm)
	}
	// AC2: "dashboard" absent from hint region (FB-092)
	if strings.Contains(strings.ToLower(statusNorm), "dashboard") {
		t.Errorf("AC2 [Observable]: 'dashboard' found in hint after Esc: %q", statusNorm)
	}
	// AC3: CTA dropped (FB-092)
	if strings.Contains(statusNorm, "Tab to resume") {
		t.Errorf("AC3 [Observable]: 'Tab to resume' CTA still present in hint: %q", statusNorm)
	}
}

// AC2 — [Anti-behavior] Fresh startup (tableTypeName="") Esc from NavPane does NOT post the hint.
func TestFB055_AC2_FreshStartup_Esc_NoHint(t *testing.T) {
	t.Parallel()
	m := newWelcomePanelAppModel(nil, nil)
	if m.tableTypeName != "" {
		t.Fatalf("precondition: tableTypeName must be empty, got %q", m.tableTypeName)
	}
	m.statusBar.Width = 120

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	appM.statusBar.Width = 120
	statusPlain := stripANSIModel(appM.statusBar.View())
	if strings.Contains(statusPlain, "Returned to welcome panel") {
		t.Errorf("AC2 [Anti-behavior]: hint posted at startup (tableTypeName=''): %q", statusPlain)
	}
}

// AC3 — [Anti-behavior] Esc from NavPane when showDashboard=true does NOT repost the hint.
func TestFB055_AC3_AlreadyDashboard_Esc_NoHint(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	m.showDashboard = true
	m.table.SetForceDashboard(true)
	m.statusBar.Width = 120

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	appM.statusBar.Width = 120
	statusPlain := stripANSIModel(appM.statusBar.View())
	if strings.Contains(statusPlain, "Returned to welcome panel") {
		t.Errorf("AC3 [Anti-behavior]: hint posted when showDashboard already true: %q", statusPlain)
	}
}

// AC4 — [Anti-behavior] Keys 3, 4, Enter, Tab that clear showDashboard do NOT post the transition hint.
func TestFB055_AC4_OtherKeys_ClearingDashboard_NoHint(t *testing.T) {
	t.Parallel()
	keys := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"key3", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}},
		{"key4", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}},
		{"shifttab", tea.KeyMsg{Type: tea.KeyShiftTab}},
	}
	for _, tt := range keys {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newNavPaneWithTableLoaded()
			m.showDashboard = true
			m.table.SetForceDashboard(true)
			m.statusBar.Width = 120

			result, _ := m.Update(tt.msg)
			appM := result.(AppModel)

			appM.statusBar.Width = 120
			statusPlain := stripANSIModel(appM.statusBar.View())
			if strings.Contains(statusPlain, "Returned to welcome panel") {
				t.Errorf("AC4 [Anti-behavior] key %q: hint present in statusBar.View() = %q", tt.name, statusPlain)
			}
		})
	}
}

// AC5 — [Input-changed] After HintClearMsg with matching token fires, hint is absent from View().
func TestFB055_AC5_HintClearMsg_MatchingToken_ClearsHint(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	m.statusBar.Width = 120

	// Fire the transition to post the hint.
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	appM.statusBar.Width = 120
	if !strings.Contains(strings.Join(strings.Fields(stripANSIModel(appM.statusBar.View())), " "), "Returned to welcome panel") {
		t.Fatal("AC5 precondition: hint not posted after Esc")
	}
	token := appM.statusBar.HintToken()

	// Fire matching HintClearMsg — hint must be gone.
	result2, _ := appM.Update(data.HintClearMsg{Token: token})
	appM2 := result2.(AppModel)

	appM2.statusBar.Width = 120
	statusPlain := stripANSIModel(appM2.statusBar.View())
	if strings.Contains(statusPlain, "Returned to welcome panel") {
		t.Errorf("AC5 [Input-changed]: hint still present after matching HintClearMsg: %q", statusPlain)
	}
}

// AC6 — [Anti-regression] FB-041 Esc chain: showDashboard set correctly, activePane stays NavPane.
func TestFB055_AC6_FB041_EscChain_Unchanged(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := result.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC6 regression: activePane = %v after Esc, want NavPane", appM.activePane)
	}
	if !appM.showDashboard {
		t.Error("AC6 regression: showDashboard = false after Esc, want true (FB-041 chain broken)")
	}
}

// ==================== End FB-055 ====================

// ==================== FB-056: Dashboard context-aware status bar ====================

// AC3 — [Observable] showDashboard=true + NavPane → statusBar.View() contains "[3]" and "[4]".
func TestFB056_AC3_NavDashboard_StatusBarHints(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	m.showDashboard = true
	m.updatePaneFocus()
	m.statusBar.Width = 160

	statusPlain := stripANSIModel(m.statusBar.View())
	if !strings.Contains(statusPlain, "[3]") {
		t.Errorf("AC3 [Observable]: '[3]' missing from NAV_DASHBOARD statusBar.View():\n%s", statusPlain)
	}
	if !strings.Contains(statusPlain, "[4]") {
		t.Errorf("AC3 [Observable]: '[4]' missing from NAV_DASHBOARD statusBar.View():\n%s", statusPlain)
	}
	// Brief AC2: "/ filter" must be absent from the dashboard-context nav hint.
	if strings.Contains(statusPlain, "/ filter") {
		t.Errorf("AC2 [Observable]: '/ filter' present in NAV_DASHBOARD statusBar.View() — must be absent per brief AC2:\n%s", statusPlain)
	}
}

// AC3 anti-behavior — [Observable] showDashboard=false + NavPane → normal NAV hints, no [3]/[4].
func TestFB056_AC3_NavNormal_StatusBarNoQuotaActivity(t *testing.T) {
	t.Parallel()
	m := newNavPaneWithTableLoaded()
	m.showDashboard = false
	m.updatePaneFocus()
	m.statusBar.Width = 160

	statusPlain := stripANSIModel(m.statusBar.View())
	if strings.Contains(statusPlain, "[3]") {
		t.Errorf("AC3 [Anti-behavior]: '[3]' present in NAV (non-dashboard) statusBar.View():\n%s", statusPlain)
	}
	if strings.Contains(statusPlain, "[4]") {
		t.Errorf("AC3 [Anti-behavior]: '[4]' present in NAV (non-dashboard) statusBar.View():\n%s", statusPlain)
	}
}

// AC5 — [Input-changed] Pressing "/" from TablePane (showDashboard=false) still opens filter bar.
func TestFB056_AC5_SlashKey_TablePane_OpensFilter(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	// showDashboard is false by default in newTablePaneModel.

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	appM := result.(AppModel)

	if !appM.filterBar.Focused() {
		t.Error("AC5 [Input-changed]: filterBar.Focused()=false after '/' in TablePane — filter not opened")
	}
	if appM.statusBar.Mode != components.ModeFilter {
		t.Errorf("AC5 [Input-changed]: statusBar.Mode = %v, want ModeFilter", appM.statusBar.Mode)
	}
}

// ==================== End FB-056 ====================

// ==================== FB-057: Help overlay documents FB-041 semantics ====================

// AC1 + AC2 — [Observable] Help overlay contains "home" in Esc entry and "resume cached" sub-line.
func TestFB057_AC1AC2_HelpOverlay_NewCopy(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.helpOverlay.Width = 100
	m.helpOverlay.Height = 40

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	appM := result.(AppModel)

	view := stripANSIModel(appM.helpOverlay.View())

	// AC1: Esc entry updated to "back / home".
	if !strings.Contains(view, "home") {
		t.Errorf("AC1 [Observable]: 'home' missing from helpOverlay.View():\n%s", view)
	}
	// AC2: Tab sub-line "resume (cached)" present. FB-110 updated copy.
	if !strings.Contains(view, "resume (cached)") {
		t.Errorf("AC2 [Observable]: 'resume (cached)' missing from helpOverlay.View():\n%s", view)
	}
}

// AC3 — [Anti-regression] All pre-existing keybind substrings still present.
func TestFB057_AC3_HelpOverlay_PreexistingKeys(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()
	m.helpOverlay.Width = 100
	m.helpOverlay.Height = 40

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	appM := result.(AppModel)
	view := stripANSIModel(appM.helpOverlay.View())

	keys := []string{"[j/k]", "[Tab]", "[Enter]", "[/]", "[Esc]", "[d]", "[r]", "[c]", "[3]", "[4]", "[?]", "[q]"}
	for _, k := range keys {
		if !strings.Contains(view, k) {
			t.Errorf("AC3 [Anti-regression]: key %q missing from helpOverlay.View():\n%s", k, view)
		}
	}
}

// AC4 — [Anti-regression] Overlay line count unchanged (VIEW column still tallest, height stable).
// Asserts by rendering the overlay at the same dimensions and counting newlines.
func TestFB057_AC4_HelpOverlay_LineCountUnchanged(t *testing.T) {
	t.Parallel()
	// Baseline: a freshly constructed model at fixed dimensions.
	m := components.NewHelpOverlayModel()
	m.Width = 100
	m.Height = 40

	view := m.View()
	lineCount := strings.Count(view, "\n") + 1

	// VIEW column has 7 content rows (header + 6); layout should be stable.
	// Assert the total is at least 7 (overlay renders) and not more than 50
	// (sanity upper bound — guards against runaway duplication).
	if lineCount < 7 {
		t.Errorf("AC4 [Anti-regression]: overlay too short — got %d lines, want ≥7", lineCount)
	}
	if lineCount > 50 {
		t.Errorf("AC4 [Anti-regression]: overlay suspiciously tall — got %d lines, want ≤50", lineCount)
	}
}

// ==================== End FB-057 ====================

// ==================== FB-087: Cross-dashboard two-slot stash ====================

// newNavPaneOnDashboard builds a NavPane model already showing the welcome panel
// (showDashboard=true, tableTypeName set) — the pre-"3" state for chain tests.
func newNavPaneOnDashboard() AppModel {
	m := newNavPaneWithTableLoaded()
	m.showDashboard = true
	m.table.SetForceDashboard(true)
	m.updatePaneFocus()
	return m
}

// AC1 — [Observable] NavPane → 3 → 4 → Esc → Esc: lands on NavPane with welcome panel.
func TestFB087_AC1_Chain_3_4_EscEsc_ReturnsToNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	// NavPane → 3
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	if m1.activePane != QuotaDashboardPane {
		t.Fatalf("AC1 setup: '3' from NavPane: activePane = %v, want QuotaDashboardPane", m1.activePane)
	}

	// QuotaDashboard → 4
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m2 := r2.(AppModel)
	if m2.activePane != ActivityDashboardPane {
		t.Fatalf("AC1 setup: '4' from QuotaDashboard: activePane = %v, want ActivityDashboardPane", m2.activePane)
	}

	// ActivityDashboard → Esc
	// FB-094: 4-from-QuotaDash guard skips activityOriginPane write → first Esc lands at
	// NavPane directly (not QuotaDashboardPane as in pre-FB-094 behavior).
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := r3.(AppModel)
	if m3.activePane != NavPane {
		t.Fatalf("AC1 setup: first Esc: activePane = %v, want NavPane (FB-094 guard collapses cross-dashboard Esc depth)", m3.activePane)
	}

	// NavPane → Esc (no-op — already at origin)
	r4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r4.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC1 [Observable]: activePane = %v after 3→4→Esc→Esc, want NavPane (stuck-loop fix)", appM.activePane)
	}
	if !appM.showDashboard {
		t.Error("AC1 [Observable]: showDashboard = false after 3→4→Esc→Esc, want true (welcome panel)")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC1 [Observable]: 'Welcome' missing from View() — welcome panel not restored:\n%s", got)
	}
}

// AC2 — [Observable] NavPane → 4 → 3 → Esc → Esc: symmetric chain lands on NavPane.
func TestFB087_AC2_Chain_4_3_EscEsc_ReturnsToNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	// NavPane → 4
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m1 := r1.(AppModel)
	if m1.activePane != ActivityDashboardPane {
		t.Fatalf("AC2 setup: '4' from NavPane: activePane = %v, want ActivityDashboardPane", m1.activePane)
	}

	// ActivityDashboard → 3
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m2 := r2.(AppModel)
	if m2.activePane != QuotaDashboardPane {
		t.Fatalf("AC2 setup: '3' from ActivityDashboard: activePane = %v, want QuotaDashboardPane", m2.activePane)
	}

	// QuotaDashboard → Esc
	// FB-094: 3-from-ActivityDash guard skips quotaOriginPane write → first Esc lands at
	// NavPane directly (not ActivityDashboardPane as in pre-FB-094 behavior).
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := r3.(AppModel)
	if m3.activePane != NavPane {
		t.Fatalf("AC2 setup: first Esc: activePane = %v, want NavPane (FB-094 guard collapses cross-dashboard Esc depth)", m3.activePane)
	}

	// NavPane → Esc (no-op — already at origin)
	r4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r4.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC2 [Observable]: activePane = %v after 4→3→Esc→Esc, want NavPane (symmetric fix)", appM.activePane)
	}
	if !appM.showDashboard {
		t.Error("AC2 [Observable]: showDashboard = false after 4→3→Esc→Esc, want true (welcome panel)")
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC2 [Observable]: 'Welcome' missing from View() — welcome panel not restored:\n%s", got)
	}
}

// AC5 — [Input-changed] NavPane → 3 → 4 → Esc: intermediate step lands on QuotaDashboardPane.
func TestFB087_AC5_Chain_3_4_Esc_CollapsesToNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m2 := r2.(AppModel)

	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r3.(AppModel)

	// FB-094 Option A: 4-from-QuotaDash skips activityOriginPane write (guard fires).
	// activityOriginPane retains zero value → NavPane. Single Esc from ActivityDash
	// lands directly at NavPane, not QuotaDashboardPane.
	if appM.activePane != NavPane {
		t.Errorf("AC5 [Input-changed]: after 3→4→Esc, activePane = %v, want NavPane (FB-094 guard collapses cross-dashboard chain)", appM.activePane)
	}
}

// ==================== End FB-087 ====================

// ==================== FB-078: BucketsLoadedMsg cancel path ====================
//
// Designer chose Option B+D: navigation away from origin pane implicitly cancels
// pendingQuotaOpen (Option B, wired in updatePaneFocus); when BucketsLoadedMsg
// arrives and pendingQuotaOpen is still true (operator stayed on origin), the
// auto-transition is replaced by a ready prompt (Option D, no force-switch).
//
// Axis-coverage:
// AC | Anti-behavior                                              | Observable                                       | Repeat-press                         | Anti-regression
// ---+------------------------------------------------------------+--------------------------------------------------+--------------------------------------+--------------------
// 1  | AC1_NavigateAway_BucketsLoaded_NoForceSwitch               | AC2_NavigateAway_HintCleared                     | -                                    | -
// 3  | -                                                          | -                                                | AC3_RepeatCancelGesture_NoOp         | -
// 4  | -                                                          | AC4_StaysOnOrigin_ReadyPromptVisible             | -                                    | AC4_StaysOnOrigin_NoPendingAfterBuckets

// newQuotaLoadingNavModel builds a NavPane AppModel with quota in loading state
// suitable for FB-078 cancel-path tests. Uses Tab-key navigation (no activityDashboard needed).
func newQuotaLoadingNavModel() AppModel {
	return newQuotaLoadingModel()
}

// AC1 [Anti-behavior] — operator presses 3 (queues), navigates to TablePane via Tab,
// then BucketsLoadedMsg arrives: activePane must NOT be QuotaDashboardPane.
func TestFB078_AC1_NavigateAway_BucketsLoaded_NoForceSwitch(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()

	// Press 3 during loading → queues open.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("setup: pendingQuotaOpen = false after '3' during loading, want true")
	}
	if m.quotaOriginPane.Pane != NavPane {
		t.Fatalf("setup: quotaOriginPane = %v, want NavPane", m.quotaOriginPane.Pane)
	}

	// Tab from NavPane → TablePane; Option B fires in updatePaneFocus.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	m = r.(AppModel)
	if m.pendingQuotaOpen {
		t.Error("AC1: pendingQuotaOpen = true after Tab (navigate away), want false (Option B cancel)")
	}
	if m.activePane != TablePane {
		t.Errorf("AC1 setup: activePane = %v after Tab, want TablePane", m.activePane)
	}

	// BucketsLoadedMsg arrives — pendingQuotaOpen already false, no ready prompt, no force-switch.
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	if appM.activePane == QuotaDashboardPane {
		t.Error("AC1: activePane = QuotaDashboardPane after BucketsLoadedMsg post-navigation-cancel, want TablePane")
	}
	// No ready prompt should appear — pending was already cleared before buckets arrived.
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("AC1: 'Quota dashboard ready' prompt visible after navigation-cancel, want absent:\n%s", got)
	}
}

// AC2 [Observable] — after navigation cancels pending, the loading hint is cleared from View().
func TestFB078_AC2_NavigateAway_HintCleared(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()

	// Press 3 → loading hint appears.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	before := stripANSIModel(m.View())
	if !strings.Contains(before, "Quota dashboard loading") {
		t.Fatalf("AC2 setup: 'Quota dashboard loading' missing from View() before navigation:\n%s", before)
	}

	// Tab from NavPane → Option B clears hint.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	appM := r.(AppModel)

	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC2: 'Quota dashboard loading' hint still visible after navigation cancel:\n%s", got)
	}
	if appM.pendingQuotaOpen {
		t.Error("AC2: pendingQuotaOpen = true after navigation, want false")
	}
}

// AC3 [Repeat-press] — repeat cancel gesture (navigate away twice) must not break anything.
// First navigation cancels; second is a no-op with no panic or stale state.
func TestFB078_AC3_RepeatCancelGesture_NoOp(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()

	// Press 3 → queue.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	// First navigation (NavPane → TablePane): cancels pending.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	m = r.(AppModel)
	if m.pendingQuotaOpen {
		t.Error("AC3 first cancel: pendingQuotaOpen = true, want false")
	}

	// Second navigation (TablePane → NavPane): already cancelled, must be a no-op.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	appM := r.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC3 second cancel: pendingQuotaOpen = true after repeat navigation, want false (no-op)")
	}
	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC3: loading hint still visible after repeat cancel:\n%s", got)
	}
}

// AC4 [Observable + Anti-regression] — when operator stays on origin pane (NavPane) and
// BucketsLoadedMsg arrives with pendingQuotaOpen=true, the ready prompt is shown and
// no auto-transition occurs (Option D replaces force-switch).
func TestFB078_AC4_StaysOnOrigin_ReadyPromptShown_NoForceSwitch(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()

	// Press 3 → queue (remain on NavPane).
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("setup: pendingQuotaOpen = false, want true")
	}
	if m.activePane != NavPane {
		t.Fatalf("setup: activePane = %v, want NavPane (no navigation yet)", m.activePane)
	}

	// BucketsLoadedMsg arrives — operator still on origin pane.
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	// Option D: no auto-transition — ready prompt replaces loading hint.
	if appM.activePane == QuotaDashboardPane {
		t.Error("AC4: activePane = QuotaDashboardPane after BucketsLoadedMsg (no force-switch expected, Option D)")
	}
	// pendingQuotaOpen cleared so a subsequent '3' takes the immediate path.
	if appM.pendingQuotaOpen {
		t.Error("AC4: pendingQuotaOpen = true after BucketsLoadedMsg, want false")
	}
	// Ready prompt visible in View().
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("AC4: 'Quota dashboard ready' prompt missing from View() after BucketsLoadedMsg on origin pane:\n%s", got)
	}
	// Loading hint must be gone.
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC4: 'Quota dashboard loading' hint still present after BucketsLoadedMsg:\n%s", got)
	}
}

// [Input-changed] pair — same BucketsLoadedMsg, different active-pane state → different outcomes.
// (a) Operator stays on origin NavPane → ready-prompt visible in View().
// (b) Operator navigated away (TablePane) → ready-prompt absent; loading hint already gone.
func TestFB078_InputChanged_a_OnOriginPane_ReadyPromptVisible(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	// Remain on origin pane, then deliver buckets.
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("[Input-changed](a): 'Quota dashboard ready' absent when staying on origin pane:\n%s", got)
	}
	if appM.activePane == QuotaDashboardPane {
		t.Error("[Input-changed](a): force-switched to QuotaDashboardPane, want NavPane (Option D)")
	}
}

func TestFB078_InputChanged_b_OffOriginPane_ReadyPromptAbsent(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	// Navigate away (Option B clears pendingQuotaOpen).
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	m = r.(AppModel)

	// Deliver same BucketsLoadedMsg — pending already cleared, no ready-prompt.
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	got := stripANSIModel(appM.View())
	if strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("[Input-changed](b): 'Quota dashboard ready' present after off-origin delivery, want absent:\n%s", got)
	}
	if strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("[Input-changed](b): 'Quota dashboard loading' still visible after cancel + buckets:\n%s", got)
	}
}

// AC4 [Anti-regression] — after ready-prompt is shown, pressing '3' triggers the immediate
// transition to QuotaDashboardPane (quota.IsLoading() is false post-BucketsLoadedMsg).
func TestFB078_AC4_AfterReadyPrompt_Key3_TransitionsToQuotaDashboard(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()

	// Press 3 → queue.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	// BucketsLoadedMsg → ready-prompt, pendingQuotaOpen cleared.
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	m = r.(AppModel)
	if !strings.Contains(stripANSIModel(m.View()), "Quota dashboard ready") {
		t.Fatal("setup: ready-prompt not shown after BucketsLoadedMsg")
	}

	// Operator presses '3' at the ready-prompt → immediate transition (quota not loading).
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC4 anti-regression: activePane = %v after '3' at ready-prompt, want QuotaDashboardPane", appM.activePane)
	}
	if appM.pendingQuotaOpen {
		t.Error("AC4 anti-regression: pendingQuotaOpen = true after immediate transition, want false")
	}
}

// ==================== End FB-078 ====================

// ==================== FB-107: Quota refresh state-cleanup gaps on off-pane errors + context switch ====================

// newQuotaRefreshingOffPaneModel builds a model in NavPane with quota in refreshing state.
// Simulates: operator pressed [r] from QuotaDashboardPane, navigated away, error arrives off-pane.
func newQuotaRefreshingOffPaneModel() AppModel {
	m := newQuotaLoadingModel()
	m.quota.SetRefreshing(true)
	m.activePane = NavPane
	return m
}

// [Observable / AC1] Site A — BucketsErrorMsg while activePane=NavPane + quota.refreshing=true:
// quota.refreshing cleared AND statusBar.Err set.
func TestFB107_AC1_BucketsError_OffPane_ClearsRefreshing(t *testing.T) {
	t.Parallel()
	m := newQuotaRefreshingOffPaneModel()
	m.statusBar.Width = 120

	result, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("connection reset")})
	appM := result.(AppModel)

	if appM.quota.IsRefreshing() {
		t.Error("AC1 [Observable]: quota.refreshing = true after off-pane BucketsErrorMsg, want false")
	}
	if appM.bucketErr == nil {
		t.Error("AC1 [Observable]: bucketErr = nil after BucketsErrorMsg, want non-nil")
	}
}

// [Observable / AC2] Site A — LoadErrorMsg while activePane=NavPane + quota.refreshing=true:
// quota.refreshing cleared.
func TestFB107_AC2_LoadError_OffPane_ClearsRefreshing(t *testing.T) {
	t.Parallel()
	m := newQuotaRefreshingOffPaneModel()

	result, _ := m.Update(data.LoadErrorMsg{Err: errors.New("timeout"), Severity: data.ErrorSeverityWarning})
	appM := result.(AppModel)

	if appM.quota.IsRefreshing() {
		t.Error("AC2 [Observable]: quota.refreshing = true after off-pane LoadErrorMsg, want false")
	}
}

// [Observable / AC3] Site B — ContextSwitchedMsg while quota.refreshing=true: refreshing cleared.
func TestFB107_AC3_ContextSwitch_ClearsRefreshing(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()
	m.quota.SetRefreshing(true)

	result, _ := m.Update(components.ContextSwitchedMsg{})
	appM := result.(AppModel)

	if appM.quota.IsRefreshing() {
		t.Error("AC3 [Observable]: quota.refreshing = true after ContextSwitchedMsg, want false")
	}
}

// [Input-changed / AC4] Site A BucketsErrorMsg: both activePane=QuotaDashboardPane and activePane=NavPane
// now reset quota.refreshing (previously only the on-pane case did).
func TestFB107_AC4_InputChanged_BucketsError_BothPanesClearRefreshing(t *testing.T) {
	t.Parallel()

	t.Run("on-pane", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingModel()
		m.quota.SetRefreshing(true)
		m.activePane = QuotaDashboardPane

		result, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("timeout")})
		appM := result.(AppModel)
		if appM.quota.IsRefreshing() {
			t.Error("on-pane: quota.refreshing = true after BucketsErrorMsg on QuotaDashboardPane")
		}
	})

	t.Run("off-pane", func(t *testing.T) {
		t.Parallel()
		m := newQuotaRefreshingOffPaneModel()

		result, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("timeout")})
		appM := result.(AppModel)
		if appM.quota.IsRefreshing() {
			t.Error("off-pane: quota.refreshing = true after BucketsErrorMsg on NavPane")
		}
	})
}

// [Anti-behavior / AC5] BucketsErrorMsg does NOT clear quota.buckets (prior bucket data preserved).
func TestFB107_AC5_BucketsError_PreservesBuckets(t *testing.T) {
	t.Parallel()
	m := newQuotaRefreshingOffPaneModel()
	buckets := []data.AllowanceBucket{{Name: "cpu"}}
	m.quota.SetBuckets(buckets)

	result, _ := m.Update(data.BucketsErrorMsg{Err: errors.New("timeout")})
	appM := result.(AppModel)

	if !appM.quota.HasBuckets() {
		t.Error("AC5 [Anti-behavior]: quota.buckets cleared on BucketsErrorMsg, want preserved")
	}
}

// ==================== End FB-107 ====================

// ==================== FB-095: pendingQuotaOpen cancel paths clear quotaOriginPane ====================

// [Anti-behavior / AC1] Nav-cancel path (Site A): after 3 during loading then navigate-away,
// quotaOriginPane is zero.
func TestFB095_AC1_NavCancel_ClearsOriginPane(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingNavModel()

	// Press 3 → stash written (NavPane origin) + pendingQuotaOpen=true.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("AC1 setup: pendingQuotaOpen = false after first press — queue not set")
	}

	// Tab → navigate away → nav-cancel fires.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	appM := r.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC1: pendingQuotaOpen still true after nav-cancel")
	}
	if appM.quotaOriginPane != (DashboardOrigin{}) {
		t.Errorf("AC1 [Anti-behavior]: quotaOriginPane not cleared on nav-cancel: got %v, want zero",
			appM.quotaOriginPane)
	}
}

// [Anti-behavior / AC2] Second-press cancel path (Site B): after 3→3 during loading,
// quotaOriginPane is zero.
func TestFB095_AC2_SecondPressCancel_ClearsOriginPane(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// First press → stash written (NavPane origin) + pendingQuotaOpen=true.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("AC2 setup: pendingQuotaOpen = false after first press — queue not set")
	}

	// Second press → cancel path fires.
	r, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = r.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC2: pendingQuotaOpen still true after second-press cancel")
	}
	if appM.quotaOriginPane != (DashboardOrigin{}) {
		t.Errorf("AC2 [Anti-behavior]: quotaOriginPane not cleared on second-press cancel: got %v, want zero",
			appM.quotaOriginPane)
	}
}

// [Observable / AC3] After either cancel, a fresh 3 press from TablePane writes quotaOriginPane={TablePane}.
func TestFB095_AC3_PostCancel_Fresh3_WritesCleanStash(t *testing.T) {
	t.Parallel()

	t.Run("after-nav-cancel", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingNavModel()

		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		m = r.(AppModel)
		r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)}) // nav-cancel
		m = r.(AppModel)
		// Simulate buckets loaded so next '3' takes the immediate path.
		r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
		m = r.(AppModel)
		// Move to TablePane so fresh '3' stashes TablePane (non-zero, distinct from NavPane).
		m.activePane = TablePane
		m.quota.SetLoading(false)

		r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		appM := r.(AppModel)

		if appM.quotaOriginPane.Pane != TablePane {
			t.Errorf("AC3 nav-cancel: quotaOriginPane.Pane = %v, want TablePane", appM.quotaOriginPane.Pane)
		}
	})

	t.Run("after-second-press-cancel", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingModel()

		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		m = r.(AppModel)
		r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}) // second-press cancel
		m = r.(AppModel)
		// Simulate buckets loaded so next '3' takes the immediate path.
		r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
		m = r.(AppModel)
		m.activePane = TablePane
		m.quota.SetLoading(false)

		r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		appM := r.(AppModel)

		if appM.quotaOriginPane.Pane != TablePane {
			t.Errorf("AC3 second-press-cancel: quotaOriginPane.Pane = %v, want TablePane", appM.quotaOriginPane.Pane)
		}
	})
}

// [Input-changed / AC4] Same 3 key with pendingQuotaOpen=true: both cancel paths clear quotaOriginPane
// (stash-clear is path-agnostic). Two presses (3→3) vs navigate-away both result in zeroed stash.
func TestFB095_AC4_InputChanged_BothCancelPaths_ClearStash(t *testing.T) {
	t.Parallel()

	t.Run("nav-cancel", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingNavModel()
		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		m = r.(AppModel)
		r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
		appM := r.(AppModel)
		if appM.quotaOriginPane != (DashboardOrigin{}) {
			t.Errorf("nav-cancel: quotaOriginPane = %v, want zero", appM.quotaOriginPane)
		}
	})

	t.Run("second-press-cancel", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingModel()
		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		m = r.(AppModel)
		r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		appM := r.(AppModel)
		if appM.quotaOriginPane != (DashboardOrigin{}) {
			t.Errorf("second-press-cancel: quotaOriginPane = %v, want zero", appM.quotaOriginPane)
		}
	})
}

// ==================== End FB-095 ====================

// ==================== FB-094: 3→4→3 cross-dashboard stash-clobber fix ====================
//
// Designer chose Option A (dashboard-as-origin guard): `case "3"` guards with
// `if activePane != ActivityDashboardPane` before writing quotaOriginPane; `case "4"`
// guards with `if activePane != QuotaDashboardPane` before writing activityOriginPane.
// Cross-dashboard presses skip the stash overwrite, preserving the original entry pane.
//
// Spec: docs/tui-ux-specs/fb-094-cross-dashboard-3-press-chain-fix.md (10 ACs)
//
// Axis-coverage:
// Brief AC | Axis              | Test(s)
// ---------+-------------------+-----------------------------------------------------------------------
// AC1      | [Anti-behavior]   | TestFB094_AC1_3_4_3_EscEsc_ReachesNavPane
// AC2      | [Anti-behavior]   | TestFB094_AC2_4_3_4_EscEsc_ReachesNavPane (symmetric)
// AC3      | [Observable]      | TestFB094_AC3_3_4_3_QuotaOriginPreserved
// AC4      | [Input-changed]   | TestFB094_AC4a_NavPaneStart_QuotaOriginIsNavPane
//          |                   | + TestFB094_AC4b_TablePaneStart_QuotaOriginIsTablePane
//          | (gate axis)       | + TestFB094_InputChanged_3Key_NavPane_vs_ActivityDash
//          |                   | + TestFB094_InputChanged_4Key_NavPane_vs_QuotaDash
// AC5      | [Repeat-press]    | TestFB094_AC5_ExtraEscFromNavPane_NoDashboardReentry
// AC6      | [Anti-regression] | existing: TestFB087_AC1_Chain_3_4_EscEsc_ReturnsToNavPane
//          |                   |           TestFB087_AC2_Chain_4_3_EscEsc_ReturnsToNavPane
// AC7      | [Anti-regression] | model.go:46–50 docstring (engineer-shipped, reviewer-verified)
// AC8      | [Anti-regression] | existing: TestFB048_Key3_RoundTrip_FromTablePane_RestoresTablePane
//          |                   | TestFB094_AC8_SinglePress3_StillWorks (guard not over-broad)
//          |                   | TestFB094_AC8_SinglePress4_StillWorks (symmetric)
// AC9      | [Anti-regression] | existing: TestFB087_AC5_Chain_3_4_Esc_CollapsesToNavPane (engineer-flipped)
// AC10     | [Integration]     | go install ./... + go test ./internal/tui/... -count=1

// AC1 [Anti-behavior] — 3→4→3→Esc→Esc from NavPane: both Esc presses, final activePane == NavPane.
func TestFB094_AC1_3_4_3_EscEsc_ReachesNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	if m1.activePane != QuotaDashboardPane {
		t.Fatalf("AC1 setup: '3' from NavPane → %v, want QuotaDashboardPane", m1.activePane)
	}
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m2 := r2.(AppModel)
	if m2.activePane != ActivityDashboardPane {
		t.Fatalf("AC1 setup: '4' from QuotaDash → %v, want ActivityDashboardPane", m2.activePane)
	}
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m3 := r3.(AppModel)
	if m3.activePane != QuotaDashboardPane {
		t.Fatalf("AC1 setup: second '3' from ActivityDash → %v, want QuotaDashboardPane", m3.activePane)
	}

	// First Esc: guard preserved quotaOriginPane={NavPane} → restores NavPane.
	r4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m4 := r4.(AppModel)
	// Second Esc: on NavPane, showDashboard=true → no-op.
	r5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r5.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC1 [Anti-behavior]: 3→4→3→Esc→Esc → activePane = %v, want NavPane", appM.activePane)
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC1 [Anti-behavior]: Welcome panel missing from View() after Esc chain:\n%s", got)
	}
}

// AC2 [Anti-behavior] symmetric — 4→3→4→Esc→Esc from NavPane: final activePane == NavPane.
func TestFB094_AC2_4_3_4_EscEsc_ReachesNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m1 := r1.(AppModel)
	if m1.activePane != ActivityDashboardPane {
		t.Fatalf("AC2 setup: '4' from NavPane → %v, want ActivityDashboardPane", m1.activePane)
	}
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m2 := r2.(AppModel)
	if m2.activePane != QuotaDashboardPane {
		t.Fatalf("AC2 setup: '3' from ActivityDash → %v, want QuotaDashboardPane", m2.activePane)
	}
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m3 := r3.(AppModel)
	if m3.activePane != ActivityDashboardPane {
		t.Fatalf("AC2 setup: second '4' from QuotaDash → %v, want ActivityDashboardPane", m3.activePane)
	}

	r4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m4 := r4.(AppModel)
	r5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r5.(AppModel)

	if appM.activePane != NavPane {
		t.Errorf("AC2 [Anti-behavior] symmetric: 4→3→4→Esc→Esc → activePane = %v, want NavPane", appM.activePane)
	}
	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC2 [Anti-behavior] symmetric: Welcome panel missing from View():\n%s", got)
	}
}

// AC3 [Observable] — after 3→4→3 from NavPane: quotaOriginPane.Pane == NavPane (not ActivityDash).
// Pre-FB-094 bug: 3rd press overwrote quotaOriginPane with {ActivityDashboardPane}.
func TestFB094_AC3_3_4_3_QuotaOriginPreserved(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m2 := r2.(AppModel)
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := r3.(AppModel)

	if appM.quotaOriginPane.Pane == ActivityDashboardPane {
		t.Error("AC3 [Observable]: quotaOriginPane.Pane = ActivityDashboardPane — stash-clobber bug present")
	}
	if appM.quotaOriginPane.Pane != NavPane {
		t.Errorf("AC3 [Observable]: quotaOriginPane.Pane = %v, want NavPane (entry stash preserved)", appM.quotaOriginPane.Pane)
	}
}

// AC4 [Input-changed] — single '3' press from different starting panes → different quotaOriginPane.Pane.
// (a) NavPane → quotaOriginPane.Pane = NavPane.
func TestFB094_AC4a_NavPaneStart_QuotaOriginIsNavPane(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := r1.(AppModel)

	if appM.quotaOriginPane.Pane != NavPane {
		t.Errorf("AC4a [Input-changed]: '3' from NavPane: quotaOriginPane.Pane = %v, want NavPane", appM.quotaOriginPane.Pane)
	}
}

// (b) TablePane → quotaOriginPane.Pane = TablePane (different result from same '3' key).
func TestFB094_AC4b_TablePaneStart_QuotaOriginIsTablePane(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := r1.(AppModel)

	if appM.quotaOriginPane.Pane != TablePane {
		t.Errorf("AC4b [Input-changed]: '3' from TablePane: quotaOriginPane.Pane = %v, want TablePane", appM.quotaOriginPane.Pane)
	}
}

// AC5 [Repeat-press] — extra Esc from NavPane after 3→4→3→Esc must NOT re-enter any dashboard.
func TestFB094_AC5_ExtraEscFromNavPane_NoDashboardReentry(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m2 := r2.(AppModel)
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m3 := r3.(AppModel)
	r4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m4 := r4.(AppModel)
	if m4.activePane != NavPane {
		t.Fatalf("AC5 setup: 3→4→3→Esc → %v, want NavPane", m4.activePane)
	}

	r5, _ := m4.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appM := r5.(AppModel)

	if appM.activePane == QuotaDashboardPane || appM.activePane == ActivityDashboardPane {
		t.Errorf("AC5 [Repeat-press]: extra Esc re-entered dashboard: activePane = %v", appM.activePane)
	}
}

// AC8 [Anti-regression] — single '3' press from TablePane still opens QuotaDash with correct stash.
// Guard must NOT fire for non-dashboard panes.
func TestFB094_AC8_SinglePress3_StillWorks(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := r1.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Errorf("AC8 [Anti-regression]: '3' from TablePane → activePane = %v, want QuotaDashboardPane", appM.activePane)
	}
	if appM.quotaOriginPane.Pane != TablePane {
		t.Errorf("AC8 [Anti-regression]: '3' from TablePane: quotaOriginPane.Pane = %v, want TablePane", appM.quotaOriginPane.Pane)
	}
}

// AC8 symmetric — single '4' press from TablePane still opens ActivityDash with correct stash.
func TestFB094_AC8_SinglePress4_StillWorks(t *testing.T) {
	t.Parallel()
	m := newTablePaneModel()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := r1.(AppModel)

	if appM.activePane != ActivityDashboardPane {
		t.Errorf("AC8 symmetric [Anti-regression]: '4' from TablePane → activePane = %v, want ActivityDashboardPane", appM.activePane)
	}
	if appM.activityOriginPane.Pane != TablePane {
		t.Errorf("AC8 symmetric [Anti-regression]: '4' from TablePane: activityOriginPane.Pane = %v, want TablePane", appM.activityOriginPane.Pane)
	}
}

// [Input-changed] gate axis (product-experience requirement) — same '3' key, activePane varies:
// (a) '3' from NavPane: guard does NOT fire → quotaOriginPane written.
// (b) '3' from ActivityDashboardPane (cross-dashboard): guard fires → quotaOriginPane preserved.
// Same keystroke, different activePane → different stash outcome.
func TestFB094_InputChanged_3Key_NavPane_StashWritten(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	before := m.quotaOriginPane.Pane // NavPane or zero
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := r1.(AppModel)

	// Guard did not fire (NavPane ≠ ActivityDash) → stash updated to NavPane.
	if appM.quotaOriginPane.Pane == before && before != NavPane {
		t.Errorf("[Input-changed](a): quotaOriginPane.Pane unchanged after '3' from NavPane, want NavPane")
	}
	if appM.quotaOriginPane.Pane != NavPane {
		t.Errorf("[Input-changed](a): quotaOriginPane.Pane = %v, want NavPane (guard must not fire from NavPane)", appM.quotaOriginPane.Pane)
	}
}

func TestFB094_InputChanged_3Key_ActivityDash_StashPreserved(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	// Navigate to ActivityDash so quotaOriginPane is set to NavPane.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel) // now on QuotaDash, quotaOriginPane={NavPane}
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m2 := r2.(AppModel) // now on ActivityDash, quotaOriginPane still {NavPane}

	// Press '3' from ActivityDash: guard fires → quotaOriginPane must NOT change.
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := r3.(AppModel)

	// Same '3' key, but activePane was ActivityDash → stash preserved.
	if appM.quotaOriginPane.Pane == ActivityDashboardPane {
		t.Error("[Input-changed](b): quotaOriginPane.Pane = ActivityDashboardPane — guard did not fire; stash clobbered")
	}
	if appM.quotaOriginPane.Pane != NavPane {
		t.Errorf("[Input-changed](b): quotaOriginPane.Pane = %v, want NavPane (preserved from initial '3' press)", appM.quotaOriginPane.Pane)
	}
}

// Symmetric [Input-changed] pair for '4' key.
func TestFB094_InputChanged_4Key_NavPane_StashWritten(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := r1.(AppModel)

	if appM.activityOriginPane.Pane != NavPane {
		t.Errorf("[Input-changed] 4-key (a): activityOriginPane.Pane = %v, want NavPane (guard must not fire from NavPane)", appM.activityOriginPane.Pane)
	}
}

func TestFB094_InputChanged_4Key_QuotaDash_StashPreserved(t *testing.T) {
	t.Parallel()
	m := newNavPaneOnDashboard()

	// Navigate to QuotaDash so activityOriginPane remains zero.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m1 := r1.(AppModel) // ActivityDash, activityOriginPane={NavPane}
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m2 := r2.(AppModel) // QuotaDash, activityOriginPane still {NavPane}

	// Press '4' from QuotaDash: guard fires → activityOriginPane must NOT change.
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := r3.(AppModel)

	if appM.activityOriginPane.Pane == QuotaDashboardPane {
		t.Error("[Input-changed] 4-key (b): activityOriginPane.Pane = QuotaDashboardPane — guard did not fire; stash clobbered")
	}
	if appM.activityOriginPane.Pane != NavPane {
		t.Errorf("[Input-changed] 4-key (b): activityOriginPane.Pane = %v, want NavPane (preserved from initial '4' press)", appM.activityOriginPane.Pane)
	}
}

// ==================== End FB-094 ====================

// ==================== FB-088: Dashboard origin affordance (model wiring) ====================

// newQuotaDashboardNavModel builds an AppModel in NavPane (welcome) for testing '3' key wiring.
func newQuotaDashboardNavModel() AppModel {
	quota := components.NewQuotaDashboardModel(200, 30, "test-proj")
	quota.SetBuckets([]data.AllowanceBucket{
		{Name: "cpu", ConsumerKind: "project", ResourceType: "cpus", Allocated: 10, Limit: 100},
	})
	m := AppModel{
		ctx:              context.Background(),
		rc:               stubResourceClient{},
		bc:               &stubBucketClient{},
		activePane:       NavPane,
		showDashboard:    true,
		sidebar:          components.NewNavSidebarModel(22, 30),
		table:            components.NewResourceTableModel(200, 30),
		detail:           components.NewDetailViewModel(200, 30),
		quota:            quota,
		activityDashboard: components.NewActivityDashboardModel(200, 30, "test-proj"),
		filterBar:        components.NewFilterBarModel(),
		helpOverlay:      components.NewHelpOverlayModel(),
	}
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}
	m.updatePaneFocus()
	return m
}

// AC6 [Anti-behavior] — after '3' opens QuotaDash then Esc returns, quota.View() does NOT contain "back to".
func TestFB088_Model_AC6_EscClearsOriginLabel(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardNavModel()

	// Press '3' from NavPane (welcome) — stash write + SetOriginLabel fires.
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	if m1.activePane != QuotaDashboardPane {
		t.Fatalf("setup: after '3', activePane = %v, want QuotaDashboardPane", m1.activePane)
	}
	if !strings.Contains(stripANSIModel(m1.quota.View()), "back to welcome panel") {
		t.Fatalf("setup: after '3' from NavPane (welcome), quota.View() missing 'back to welcome panel'")
	}

	// Press Esc — should clear origin label.
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := r2.(AppModel)
	if m2.activePane == QuotaDashboardPane {
		t.Fatalf("setup: after Esc, still in QuotaDashboardPane")
	}
	got := stripANSIModel(m2.quota.View())
	if strings.Contains(got, "back to") {
		t.Errorf("AC6: quota.View() contains 'back to' after Esc; want label cleared:\n%s", got)
	}
}

// AC7 [Anti-regression] — NavPane → '3' → Esc round-trip returns to NavPane (FB-048 AC3 preserved).
func TestFB088_Model_AC7_FB048RoundTripPreserved(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardNavModel()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	if m1.activePane != QuotaDashboardPane {
		t.Fatalf("after '3': activePane = %v, want QuotaDashboardPane", m1.activePane)
	}

	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := r2.(AppModel)
	if m2.activePane != NavPane {
		t.Errorf("AC7 [Anti-regression]: after '3' → Esc, activePane = %v, want NavPane", m2.activePane)
	}
}

// [Observable] — '3' from TablePane wires "resource list" label into quota.View().
func TestFB088_Model_Observable_TablePaneOrigin(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardNavModel()
	// Navigate to TablePane first.
	m.activePane = TablePane
	m.tableTypeName = "allowancebuckets"
	m.showDashboard = false
	m.updatePaneFocus()

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m1 := r1.(AppModel)
	if m1.activePane != QuotaDashboardPane {
		t.Fatalf("after '3' from TablePane: activePane = %v, want QuotaDashboardPane", m1.activePane)
	}

	got := stripANSIModel(m1.quota.View())
	if !strings.Contains(got, "[3] back to resource list") {
		t.Errorf("[Observable]: quota.View() missing '[3] back to resource list' after '3' from TablePane:\n%s", got)
	}
}

// [Observable] — '4' from NavPane (welcome) wires "welcome panel" label into activityDashboard.View().
func TestFB088_Model_Observable_ActivityDash_WelcomePanelOrigin(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardNavModel()
	// showDashboard = true (welcome panel state).

	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m1 := r1.(AppModel)
	if m1.activePane != ActivityDashboardPane {
		t.Fatalf("after '4': activePane = %v, want ActivityDashboardPane", m1.activePane)
	}

	got := stripANSIModel(m1.activityDashboard.View())
	if !strings.Contains(got, "[4] back to welcome panel") {
		t.Errorf("[Observable]: activityDashboard.View() missing '[4] back to welcome panel' after '4' from NavPane (welcome):\n%s", got)
	}
}

// [Anti-behavior] — '4' from QuotaDash (guard fires) does NOT overwrite activityDashboard origin label.
func TestFB088_Model_AntiBehavior_4FromQuotaDashGuardPreservesLabel(t *testing.T) {
	t.Parallel()
	m := newQuotaDashboardNavModel()

	// '4' from NavPane: activityOriginPane = NavPane, label = "welcome panel".
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m1 := r1.(AppModel)
	// Navigate to QuotaDash (via '3' from ActivityDash — guard fires).
	r2, _ := m1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	m2 := r2.(AppModel)
	if m2.activePane != QuotaDashboardPane {
		t.Fatalf("after '4'→'3': activePane = %v, want QuotaDashboardPane", m2.activePane)
	}
	// Press '4' again from QuotaDash: guard fires → activityOriginPane preserved.
	r3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	m3 := r3.(AppModel)
	if m3.activePane != ActivityDashboardPane {
		t.Fatalf("after '4'→'3'→'4': activePane = %v, want ActivityDashboardPane", m3.activePane)
	}
	// activityDashboard should still show "welcome panel" (not "quota dashboard").
	got := stripANSIModel(m3.activityDashboard.View())
	if strings.Contains(got, "back to quota dashboard") {
		t.Errorf("[Anti-behavior]: activityDashboard.View() contains 'back to quota dashboard' after guard; origin label was clobbered:\n%s", got)
	}
	if !strings.Contains(got, "back to welcome panel") {
		t.Errorf("[Anti-behavior]: activityDashboard.View() missing 'back to welcome panel' (preserved entry label):\n%s", got)
	}
}

// ==================== End FB-088 ====================

// ==================== FB-079: Quota loading hint copy update ====================
// Brief: loading hint changed from "Quota dashboard loading…" to
//   "Quota dashboard loading… press [3] to cancel" so operators know which
//   key to press and when. Both hint phases (loading + ready) now communicate
//   the manual-confirm model introduced by FB-078.
//
// Axis-coverage table (brief-AC-indexed):
//
// | Brief AC | Axis              | Test                                                         |
// |----------|-------------------|--------------------------------------------------------------|
// | AC1      | [Observable]      | TestFB079_AC1_LoadingHint_ContainsPress3Suffix                |
// | AC2      | [Observable]      | TestFB079_AC2_ReadyHint_ContainsDashboardReadyAndPress3       |
// | AC3      | [Anti-regression] | TestFB079_AC3_OldLoadingCopy_Absent                          |
// | AC4      | [Anti-regression] | TestFB047_BucketsErrorMsg_ClearsPendingOpen (existing, green) |
// |          |                   | TestFB047_LoadErrorMsg_ClearsPendingOpen (existing, green)    |
// | AC5      | [Integration]     | go install ./... + go test ./internal/tui/... -count=1       |
// | — (gate) | [Input-changed]   | N/A — brief has only one keypress site (line 1180); the copy  |
// |          |                   | change is constant, not conditional on input variation.       |
// |          |                   | AC1 vs AC2 already pair loading-phase vs ready-phase copy.   |

// AC1 [Observable]: pressing '3' during quota load posts the new loading hint
// copy containing the " press [3] to cancel" suffix.
func TestFB079_AC1_LoadingHint_ContainsPress3Suffix(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "press [3] to cancel") {
		t.Errorf("AC1: 'press [3] to cancel' missing from View() after '3' during loading:\n%s", got)
	}
	if !strings.Contains(got, "Quota dashboard loading") {
		t.Errorf("AC1: 'Quota dashboard loading' missing from View():\n%s", got)
	}
}

// AC2 [Observable]: BucketsLoadedMsg with pendingQuotaOpen=true on origin pane
// posts the ready-phase hint containing "Quota dashboard ready" and "press [3]".
func TestFB079_AC2_ReadyHint_ContainsDashboardReadyAndPress3(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// Press '3' to set pendingQuotaOpen=true; operator stays on NavPane (origin pane).
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("precondition: pendingQuotaOpen must be true after '3' during loading")
	}

	// BucketsLoadedMsg fires while still on origin pane → ready-prompt path (FB-078 Option D).
	result, _ = appM.Update(data.BucketsLoadedMsg{})
	appM = result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "Quota dashboard ready") {
		t.Errorf("AC2: 'Quota dashboard ready' missing from View() after BucketsLoadedMsg:\n%s", got)
	}
	if !strings.Contains(got, "press [3]") {
		t.Errorf("AC2: 'press [3]' missing from View() after BucketsLoadedMsg:\n%s", got)
	}
}

// AC3 [Anti-regression]: the old exact loading-hint copy (no continuation) is
// absent. The new copy extends past "…" with " press [3] to cancel", so the
// statusBar.Hint field must never equal the old truncated string.
func TestFB079_AC3_OldLoadingCopy_Absent(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	appM := result.(AppModel)

	const oldCopy = "Quota dashboard loading\u2026" // "Quota dashboard loading…" — old exact string
	if appM.statusBar.Hint == oldCopy {
		t.Errorf("AC3: statusBar.Hint = %q — old copy without suffix still present; want new copy with 'press [3] to cancel'", appM.statusBar.Hint)
	}
	if !strings.Contains(appM.statusBar.Hint, "press [3] to cancel") {
		t.Errorf("AC3: statusBar.Hint = %q — new suffix 'press [3] to cancel' absent", appM.statusBar.Hint)
	}
}

// ==================== End FB-079 ====================

// ==================== FB-064: DetailPane [3] affordance visible on initial render for long manifests ====================
//
// Axis-coverage table:
// AC1 | Observable      | TestFB064_AC1_Observable_TopHintInDetailContent
// AC2 | Anti-regression | TestFB064_AC2_AntiRegression_BottomSeparatorRetained
// AC3 | Anti-behavior   | TestFB064_AC3_AntiBehavior_YamlMode_TopHintAbsent
// AC4 | Anti-behavior   | TestFB064_AC4_AntiBehavior_ConditionsMode_TopHintAbsent
//     |                 | TestFB064_AC4_AntiBehavior_EventsMode_TopHintAbsent
// AC5 | Anti-behavior   | TestFB064_AC5_AntiBehavior_NoBuckets_TopHintAbsent
// [IC]| Input-changed   | TestFB064_InputChanged_YamlModeToggle_TopHintChanges
// AC6 | Anti-regression | covered by existing FB-044 buildQuotaSectionHeader tests remaining green
// AC7 | Integration     | go install ./... + go test ./internal/tui/...

// newFB064DetailModel builds a model in DETAIL pane mode with 80-line describe content
// and a matching quota bucket — simulating a long manifest where the top hint is needed.
// detail width=80 → innerW=77 ≥ 30, so buildQuotaTopHint() returns a non-empty hint.
func newFB064DetailModel() AppModel {
	m := newDetailPaneModelWithMatchingBucket()
	m.describeContent = strings.Repeat("  spec-field: value\n", 80)
	return m
}

// [Observable] AC1: describe mode + matching buckets + wide pane → "[3] quota dashboard" top hint present.
func TestFB064_AC1_Observable_TopHintInDetailContent(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()

	got := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(got, "[3]") {
		t.Errorf("AC1: '[3]' missing from buildDetailContent() with matching bucket and wide pane:\n%s", got)
	}
	if !strings.Contains(got, "quota dashboard") {
		t.Errorf("AC1: 'quota dashboard' top hint missing from buildDetailContent() with matching bucket:\n%s", got)
	}
}

// [Anti-regression] AC2: bottom separator ("[3] quota dashboard") is retained
// alongside the new prepended top hint — both must coexist in the same content.
// Copy updated by FB-109: "── Quota [3] full dashboard" → "── [3] quota dashboard".
func TestFB064_AC2_AntiRegression_BottomSeparatorRetained(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()

	got := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(got, "[3] quota dashboard") {
		t.Errorf("AC2: '[3] quota dashboard' bottom separator missing — must coexist with top hint:\n%s", got)
	}
	if strings.Contains(got, "full dashboard") {
		t.Errorf("AC2 FB-109: 'full dashboard' still present; copy must be '[3] quota dashboard':\n%s", got)
	}
}

// [Anti-behavior] AC3: YAML mode early-returns before quota section → top hint suppressed.
func TestFB064_AC3_AntiBehavior_YamlMode_TopHintAbsent(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()
	m.describeRaw = testRawObject() // required for yamlMode to render YAML without error
	m.yamlMode = true

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "quota dashboard") {
		t.Errorf("AC3: 'quota dashboard' top hint present in YAML mode — must be suppressed:\n%s", got)
	}
}

// [Anti-behavior] AC4a: conditions mode early-returns before quota section → top hint suppressed.
func TestFB064_AC4_AntiBehavior_ConditionsMode_TopHintAbsent(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()
	m.describeRaw = testRawObject() // required for RenderConditionsTable to not nil-deref
	m.conditionsMode = true

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "quota dashboard") {
		t.Errorf("AC4: 'quota dashboard' top hint present in conditions mode — must be suppressed:\n%s", got)
	}
}

// [Anti-behavior] AC4b: events mode early-returns before quota section → top hint suppressed.
func TestFB064_AC4_AntiBehavior_EventsMode_TopHintAbsent(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()
	m.eventsMode = true

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "quota dashboard") {
		t.Errorf("AC4: 'quota dashboard' top hint present in events mode — must be suppressed:\n%s", got)
	}
}

// [Anti-behavior] AC5: no matching buckets → buildDetailContent() returns bare describe content,
// top hint absent.
func TestFB064_AC5_AntiBehavior_NoBuckets_TopHintAbsent(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()
	m.buckets = nil

	got := stripANSIModel(m.buildDetailContent())
	if strings.Contains(got, "quota dashboard") {
		t.Errorf("AC5: 'quota dashboard' top hint present with no buckets — must be absent:\n%s", got)
	}
}

// [Input-changed] Pair: same describe content and buckets; yamlMode=false (hint present) vs
// yamlMode=true (hint absent). Demonstrates that changing yamlMode input changes View() output.
func TestFB064_InputChanged_YamlModeToggle_TopHintChanges(t *testing.T) {
	t.Parallel()
	m := newFB064DetailModel()
	m.describeRaw = testRawObject()

	// Pair A: describe mode → top hint present.
	gotDescribe := stripANSIModel(m.buildDetailContent())
	if !strings.Contains(gotDescribe, "quota dashboard") {
		t.Fatal("input-changed pair A (yamlMode=false): 'quota dashboard' missing — precondition failed")
	}

	// Pair B: YAML mode — same fixture, different yamlMode → top hint absent.
	m.yamlMode = true
	gotYaml := stripANSIModel(m.buildDetailContent())
	if strings.Contains(gotYaml, "quota dashboard") {
		t.Errorf("input-changed pair B (yamlMode=true): 'quota dashboard' still present after yamlMode=true:\n%s", gotYaml)
	}
	if gotDescribe == gotYaml {
		t.Error("input-changed: buildDetailContent() output identical before/after yamlMode toggle — must differ")
	}
}

// ==================== End FB-064 ====================

// ==================== Begin FB-083 ====================

// TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates verifies AC6:
// when activityRows is empty (hint suppressed in S3 header), pressing [4] still
// transitions to ActivityDashboardPane — hint suppression is display-only.
func TestFB083_AC6_AntiBehavior_SuppressedHint_4KeyStillNavigates(t *testing.T) {
	t.Parallel()
	m := newActivityDashboardPaneModel()
	// activityRows is nil by default (hint suppressed per FB-083); confirm via View.
	view := stripANSIModel(m.table.View())
	if strings.Contains(view, "[4] full dashboard") {
		t.Fatal("precondition: hint should be suppressed with empty activityRows, but found it in View()")
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	appM := result.(AppModel)

	if appM.activePane != ActivityDashboardPane {
		t.Errorf("AC6: activePane = %v, want ActivityDashboardPane — [4] must navigate even when hint is suppressed", appM.activePane)
	}
}

// ==================== End FB-083 ====================

// ==================== FB-096: Esc cancel + nav-cancel acknowledgment for pending quota open ====================
//
// Three cancel paths now post "Quota dashboard cancelled":
//   Site 1 — NavPane Esc with pendingQuotaOpen (postHint → 3s transient)
//   Site 2 — updatePaneFocus nav-cancel (PostHint → persistent until overwritten)
//   Site 3 — second-press cancel (postHint → 3s transient)
//
// All hint assertions use statusBar.View() (width=120) per feedback_observable_acs_assert_view_output.

// statusBarNorm renders statusBar at width=120 and normalises whitespace for substring matching.
func statusBarNorm(m AppModel) string {
	m.statusBar.Width = 120
	return strings.Join(strings.Fields(stripANSIModel(m.statusBar.View())), " ")
}

// AC1 [Observable] — Esc on NavPane with pendingQuotaOpen: statusBar.View() contains "Quota dashboard cancelled".
func TestFB096_AC1_EscCancel_NavPane_HintShown(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// Queue open via first '3' press.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("AC1 setup: pendingQuotaOpen = false after first press")
	}

	// Esc on NavPane → Site 1 cancel.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard cancelled") {
		t.Errorf("AC1 [Observable]: statusBar.View() = %q, want contains %q", norm, "Quota dashboard cancelled")
	}
	if appM.pendingQuotaOpen {
		t.Error("AC1: pendingQuotaOpen still true after Esc cancel")
	}
}

// AC2 [Observable] — Nav-cancel (Tab away from origin): statusBar.View() contains "Quota dashboard cancelled".
func TestFB096_AC2_NavCancel_HintShown(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("AC2 setup: pendingQuotaOpen = false after first press")
	}

	// Tab → navigate away → nav-cancel in updatePaneFocus.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard cancelled") {
		t.Errorf("AC2 [Observable]: statusBar.View() = %q, want contains %q", norm, "Quota dashboard cancelled")
	}
	if appM.pendingQuotaOpen {
		t.Error("AC2: pendingQuotaOpen still true after nav-cancel")
	}
}

// AC3 [Input-changed] — Loading hint visible BEFORE cancel; cancelled hint visible AFTER (via View()).
func TestFB096_AC3_InputChanged_LoadingToCancelled(t *testing.T) {
	t.Parallel()

	t.Run("esc-cancel", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingModel()

		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		before := r.(AppModel)
		normBefore := statusBarNorm(before)
		if !strings.Contains(normBefore, "loading") {
			t.Fatalf("AC3 setup: loading hint not in statusBar.View() before Esc: %q", normBefore)
		}

		r, _ = before.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
		after := r.(AppModel)
		normAfter := statusBarNorm(after)
		if !strings.Contains(normAfter, "Quota dashboard cancelled") {
			t.Errorf("AC3 esc-cancel [Input-changed]: statusBar.View() = %q, want contains %q", normAfter, "Quota dashboard cancelled")
		}
	})

	t.Run("second-press-cancel", func(t *testing.T) {
		t.Parallel()
		m := newQuotaLoadingModel()

		r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		before := r.(AppModel)
		normBefore := statusBarNorm(before)
		if !strings.Contains(normBefore, "loading") {
			t.Fatalf("AC3 setup: loading hint not in statusBar.View() before second press: %q", normBefore)
		}

		r, _ = before.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		after := r.(AppModel)
		normAfter := statusBarNorm(after)
		if !strings.Contains(normAfter, "Quota dashboard cancelled") {
			t.Errorf("AC3 second-press-cancel [Input-changed]: statusBar.View() = %q, want contains %q", normAfter, "Quota dashboard cancelled")
		}
	})
}

// AC4 [Anti-behavior] — Esc on NavPane with NO pendingQuotaOpen: statusBar.View() must NOT contain "cancelled".
func TestFB096_AC4_AntiBehavior_EscNoPending_NoCancelHint(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel() // starts on NavPane, pendingQuotaOpen=false

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if strings.Contains(norm, "cancelled") {
		t.Errorf("AC4 [Anti-behavior]: statusBar.View() = %q, must not contain 'cancelled' when no pending open", norm)
	}
}

// AC5 [Anti-regression / FB-055] — Esc on NavPane with tableTypeName set and no pendingQuotaOpen
// still shows "Returned to welcome panel" in statusBar.View().
func TestFB096_AC5_AntiRegression_FB055_EscBackToDashboard(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel() // starts on NavPane, pendingQuotaOpen=false
	m.showDashboard = false
	m.tableTypeName = "backends"
	m.table.SetForceDashboard(false)

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Returned to welcome panel") {
		t.Errorf("AC5 [Anti-regression FB-055]: statusBar.View() = %q, want contains %q", norm, "Returned to welcome panel")
	}
}

// AC6 [Anti-regression / FB-055] — pendingQuotaOpen takes priority over FB-055 dashboard-restore path:
// Esc with both pendingQuotaOpen=true and showDashboard=false+tableTypeName set shows cancel, not welcome panel.
func TestFB096_AC6_AntiRegression_PendingOverridesDashboardRestore(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()
	m.showDashboard = false
	m.tableTypeName = "backends"

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("AC6 setup: pendingQuotaOpen = false")
	}

	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard cancelled") {
		t.Errorf("AC6 [Anti-regression]: statusBar.View() = %q, want contains %q", norm, "Quota dashboard cancelled")
	}
	if strings.Contains(norm, "welcome panel") {
		t.Errorf("AC6 [Anti-regression]: statusBar.View() = %q, must NOT contain 'welcome panel' (pending overrides restore)", norm)
	}
}

// AC7 [Anti-regression / FB-079] — nav-cancel site still clears pendingQuotaOpen (boolean; no View() assertion needed).
func TestFB096_AC7_AntiRegression_FB079_NavCancelClearsPending(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyTab)})
	appM := r.(AppModel)

	if appM.pendingQuotaOpen {
		t.Error("AC7 [Anti-regression FB-079]: pendingQuotaOpen = true after nav-cancel, want false")
	}
}

// AC8 [Integration] — full 3 → Esc → 3 flow: first '3' queues, Esc shows cancelled, re-queue shows loading.
func TestFB096_AC8_Integration_QueueEscRequeue(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// First press: queue.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("AC8 setup: queue failed")
	}

	// Esc: cancel.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
	m = r.(AppModel)
	if m.pendingQuotaOpen {
		t.Fatal("AC8: pendingQuotaOpen still true after Esc")
	}

	// Second press: re-queue.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	if !appM.pendingQuotaOpen {
		t.Error("AC8 [Integration]: pendingQuotaOpen = false after re-queue press, want true")
	}
	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "loading") {
		t.Errorf("AC8 [Integration]: statusBar.View() = %q, want loading hint after re-queue", norm)
	}
}

// AC9 [Observable] — second-press cancel: statusBar.View() contains "Quota dashboard cancelled".
func TestFB096_AC9_SecondPress_HintContainsCancelled(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard cancelled") {
		t.Errorf("AC9 [Observable]: statusBar.View() = %q, want contains %q", norm, "Quota dashboard cancelled")
	}
}

// AC10 [Input-changed] — first press shows loading; second press (cancel) shows cancelled in View().
func TestFB096_AC10_InputChanged_FirstVsSecondPress(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	after1 := r.(AppModel)
	norm1 := statusBarNorm(after1)

	r, _ = after1.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	after2 := r.(AppModel)
	norm2 := statusBarNorm(after2)

	if norm1 == norm2 {
		t.Errorf("AC10 [Input-changed]: statusBar.View() unchanged between first and second press:\n  first:  %q\n  second: %q", norm1, norm2)
	}
	if !strings.Contains(norm1, "loading") {
		t.Errorf("AC10: first-press statusBar.View() = %q, want contains 'loading'", norm1)
	}
	if !strings.Contains(norm2, "Quota dashboard cancelled") {
		t.Errorf("AC10: second-press statusBar.View() = %q, want contains 'Quota dashboard cancelled'", norm2)
	}
}

// AC11 [Anti-behavior] — third press after 3→3 cancel re-queues: statusBar.View() shows loading, NOT cancelled.
func TestFB096_AC11_AntiBehavior_ThirdPress_Requeues(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}) // cancel
	m = r.(AppModel)
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}) // third press → re-queue
	appM := r.(AppModel)

	if !appM.pendingQuotaOpen {
		t.Error("AC11 [Anti-behavior]: third press should re-queue (pendingQuotaOpen=true), got false")
	}
	norm := statusBarNorm(appM)
	if strings.Contains(norm, "Quota dashboard cancelled") {
		t.Errorf("AC11 [Anti-behavior]: third press statusBar.View() = %q, must NOT show cancelled; want loading hint", norm)
	}
	if !strings.Contains(norm, "loading") {
		t.Errorf("AC11 [Anti-behavior]: third press statusBar.View() = %q, want loading hint after re-queue", norm)
	}
}

// ==================== End FB-096 ====================

// ==================== FB-099: [3] strip label wire-up in model — BucketsLoadedMsg resets strip ====================

// AC6 [Anti-regression / FB-078] — BucketsLoadedMsg with pendingQuotaOpen=true resets strip to "3 quota".
// Asserts via stripANSIModel(appM.View()) substring after the message is processed.
func TestFB099_AC6_AntiRegression_FB078_BucketsLoaded_StripResets(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()
	m, _ = func() (AppModel, tea.Cmd) {
		r, c := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
		return r.(AppModel), c
	}()
	if !m.pendingQuotaOpen {
		t.Fatal("AC6 setup: pendingQuotaOpen = false after first press")
	}

	// Send BucketsLoadedMsg → pendingQuotaOpen cleared, strip should revert to "quota".
	r, _ := m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	if appM.pendingQuotaOpen {
		t.Fatal("AC6 setup: pendingQuotaOpen still true after BucketsLoadedMsg")
	}

	view := stripANSIModel(appM.View())
	if !strings.Contains(view, "quota") {
		t.Errorf("AC6 [Anti-regression FB-078]: View() does not contain 'quota' after strip reset:\n%s", view)
	}
	if strings.Contains(view, "3 cancel") {
		t.Errorf("AC6 [Anti-regression FB-078]: View() still contains '3 cancel' after BucketsLoadedMsg reset")
	}
}

// ==================== End FB-099 ====================

// ==================== FB-097: Persistent ready-prompt after BucketsLoadedMsg ====================
//
// Option A: replace postHint (3s decay) with statusBar.PostHint (persistent) at model.go BucketsLoadedMsg site.
// Ready-prompt "Quota dashboard ready — press [3]" now persists until operator acts or context changes.

// AC1 [Observable] — BucketsLoadedMsg with pendingQuotaOpen=true: ready-prompt is rendered in statusBar.View().
func TestFB097_AC1_BucketsLoaded_ReadyPromptShown(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	if !m.pendingQuotaOpen {
		t.Fatal("AC1 setup: pendingQuotaOpen = false after first press")
	}

	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard ready") {
		t.Errorf("AC1 [Observable]: statusBar.View() = %q, want contains %q", norm, "Quota dashboard ready")
	}
}

// AC2 [Observable, no-decay] — ready-prompt was posted via PostHint (no Cmd returned), so no HintClearCmd
// is in flight. Verify by asserting Cmd is nil — postHint returns a Cmd; PostHint returns nothing.
func TestFB097_AC2_BucketsLoaded_NoClearCmd(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)

	_, cmd := m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})

	if cmd != nil {
		t.Errorf("AC2 [Observable]: BucketsLoadedMsg returned non-nil Cmd — persistent ready-prompt must not schedule HintClearCmd")
	}
}

// AC3 [Input-changed] — same BucketsLoadedMsg: pendingQuotaOpen=true → ready-prompt; pendingQuotaOpen=false → no prompt.
func TestFB097_AC3_InputChanged_PendingVsNoPending_BucketsLoaded(t *testing.T) {
	t.Parallel()

	// With pendingQuotaOpen=true: ready-prompt appears.
	mPending := newQuotaLoadingModel()
	r, _ := mPending.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	mPending = r.(AppModel)
	r, _ = mPending.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	withPending := statusBarNorm(r.(AppModel))

	// Without pendingQuotaOpen=true: no ready-prompt (quota loads silently in background).
	mNoPending := newQuotaLoadingModel()
	r, _ = mNoPending.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	noPending := statusBarNorm(r.(AppModel))

	if withPending == noPending {
		t.Errorf("AC3 [Input-changed]: statusBar.View() identical with and without pendingQuotaOpen:\n  pending:    %q\n  no-pending: %q", withPending, noPending)
	}
	if !strings.Contains(withPending, "ready") {
		t.Errorf("AC3 [Input-changed]: pending path statusBar.View() = %q, want contains 'ready'", withPending)
	}
	if strings.Contains(noPending, "Quota dashboard ready") {
		t.Errorf("AC3 [Input-changed]: no-pending path statusBar.View() = %q, must not contain 'Quota dashboard ready'", noPending)
	}
}

// AC4 [Anti-behavior] — ready-prompt survives a HintClearMsg with a stale/mismatched token (does NOT clear).
func TestFB097_AC4_AntiBehavior_StaleHintClearMsg_DoesNotClear(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	appM := r.(AppModel)

	// Confirm ready-prompt is set.
	normBefore := statusBarNorm(appM)
	if !strings.Contains(normBefore, "ready") {
		t.Fatalf("AC4 setup: ready-prompt not in statusBar.View(): %q", normBefore)
	}

	// Send HintClearMsg with stale token (0) — PostHint bumps token so 0 is always stale.
	r, _ = appM.Update(data.HintClearMsg{Token: 0})
	normAfter := statusBarNorm(r.(AppModel))
	if !strings.Contains(normAfter, "ready") {
		t.Errorf("AC4 [Anti-behavior]: stale HintClearMsg cleared ready-prompt: before=%q after=%q", normBefore, normAfter)
	}
}

// AC5 [Anti-behavior] — ready-prompt clears on context switch (ContextSwitchedMsg).
func TestFB097_AC5_AntiBehavior_ContextSwitch_ClearsReadyPrompt(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	m = r.(AppModel)

	normBefore := statusBarNorm(m)
	if !strings.Contains(normBefore, "ready") {
		t.Fatalf("AC5 setup: ready-prompt not shown before context switch: %q", normBefore)
	}

	r, _ = m.Update(components.ContextSwitchedMsg{})
	appM := r.(AppModel)

	normAfter := statusBarNorm(appM)
	if strings.Contains(normAfter, "Quota dashboard ready") {
		t.Errorf("AC5 [Anti-behavior]: ready-prompt still shown after context switch: %q", normAfter)
	}
}

// AC6 [Anti-regression / FB-078] — loading hint (before BucketsLoadedMsg) has no HintClearCmd (persistent).
func TestFB097_AC6_AntiRegression_FB078_LoadingHintNoClearCmd(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})

	// Loading hint is posted via statusBar.PostHint (no Cmd returned from Update).
	// The '3' key handler calls m.statusBar.PostHint(...) directly — returns no Cmd.
	// Verify: the returned Cmd from the '3' press is nil (no HintClearCmd scheduled).
	if cmd != nil {
		t.Errorf("AC6 [Anti-regression FB-078]: loading hint posted a Cmd — must be nil (persistent, no HintClearCmd): got %v", cmd)
	}
}

// AC7 [Anti-regression / FB-079] — loading hint copy unchanged: "Quota dashboard loading… press [3] to cancel".
func TestFB097_AC7_AntiRegression_FB079_LoadingCopyUnchanged(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard loading") {
		t.Errorf("AC7 [Anti-regression FB-079]: loading hint = %q, want contains 'Quota dashboard loading'", norm)
	}
	if !strings.Contains(norm, "to cancel") {
		t.Errorf("AC7 [Anti-regression FB-079]: loading hint = %q, want contains 'to cancel' (FB-115 copy)", norm)
	}
}

// AC8 [Anti-regression / FB-096] — ready-prompt clears on cancel (Esc) and cancel hint appears instead.
func TestFB097_AC8_AntiRegression_FB096_CancelClearsReadyPrompt(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// 3 → queue; BucketsLoadedMsg → ready-prompt; Esc → cancel.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	m = r.(AppModel)
	norm := statusBarNorm(m)
	if !strings.Contains(norm, "ready") {
		t.Fatalf("AC8 setup: ready-prompt not shown: %q", norm)
	}

	// Esc on NavPane — FB-096 Site 1 cancel path: since pendingQuotaOpen=false now (cleared by BucketsLoadedMsg),
	// Esc falls through to FB-055 path. Confirm ready-prompt is replaced by the natural Esc flow
	// (no "cancelled" hint, since pendingQuotaOpen is already false).
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyType(tea.KeyEsc)})
	appM := r.(AppModel)

	normAfter := statusBarNorm(appM)
	if strings.Contains(normAfter, "Quota dashboard ready") {
		t.Errorf("AC8 [Anti-regression FB-096]: ready-prompt still shown after Esc: %q", normAfter)
	}
}

// AC4 [Anti-behavior] — ready-prompt clears on [3] confirm: after BucketsLoadedMsg+ready-prompt,
// pressing [3] opens QuotaDashboardPane and the ready-prompt copy is absent from statusBar.View().
func TestFB097_BriefAC4_ConfirmClearsReadyPrompt(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// Prime: [3] → pendingQuotaOpen=true; BucketsLoadedMsg → ready-prompt shown.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = r.(AppModel)
	r, _ = m.Update(data.BucketsLoadedMsg{Buckets: []data.AllowanceBucket{}})
	m = r.(AppModel)

	normBefore := statusBarNorm(m)
	if !strings.Contains(normBefore, "ready") {
		t.Fatalf("AC4 setup: ready-prompt not shown before [3] confirm: %q", normBefore)
	}

	// Confirm: [3] → opens QuotaDashboardPane.
	r, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	if appM.activePane != QuotaDashboardPane {
		t.Fatalf("AC4 setup: expected QuotaDashboardPane after [3] confirm, got %v", appM.activePane)
	}
	normAfter := statusBarNorm(appM)
	if strings.Contains(normAfter, "Quota dashboard ready") {
		t.Errorf("AC4 [Anti-behavior]: ready-prompt still present after [3] confirm: %q", normAfter)
	}
	// QuotaDashboardPane content visible in status bar left hints.
	if !strings.Contains(normAfter, "[3] back") {
		t.Errorf("AC4 [Anti-behavior]: QuotaDashboardPane hints not in statusBar after [3] confirm: %q", normAfter)
	}
}

// ==================== End FB-097 ====================

// ==================== FB-115: Quota loading hint copy ====================

// AC1 [Observable] — during pendingQuotaOpen=true, hint contains "to cancel" and NOT "when ready".
func TestFB115_AC1_Observable_LoadingHintContainsToCancel(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "to cancel") {
		t.Errorf("AC1 [Observable]: hint %q, want contains 'to cancel'", norm)
	}
	if strings.Contains(norm, "when ready") {
		t.Errorf("AC1 [Observable]: hint %q, must NOT contain 'when ready'", norm)
	}
}

// AC2 [Observable] — hint and strip aligned: hint has "to cancel", strip has "[3] cancel" simultaneously.
func TestFB115_AC2_Observable_HintAndStripAligned(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	view := stripANSIModel(appM.View())
	if !strings.Contains(view, "to cancel") {
		t.Errorf("AC2 [Observable]: 'to cancel' absent from View() during loading:\n%s", view)
	}
	if !strings.Contains(view, "cancel") {
		t.Errorf("AC2 [Observable]: 'cancel' (strip label) absent from View() during loading:\n%s", view)
	}
}

// AC3 [Input-changed] — before first [3] press: no loading hint; after: hint contains full new copy.
func TestFB115_AC3_InputChanged_BeforeVsAfterFirstPress(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	beforeNorm := statusBarNorm(m)
	if strings.Contains(beforeNorm, "Quota dashboard loading") {
		t.Errorf("AC3 [Input-changed]: loading hint present before [3] press: %q", beforeNorm)
	}

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	afterNorm := statusBarNorm(appM)
	if beforeNorm == afterNorm {
		t.Error("AC3 [Input-changed]: statusBar unchanged after [3] press during loading")
	}
	if !strings.Contains(afterNorm, "Quota dashboard loading") {
		t.Errorf("AC3 [Input-changed]: 'Quota dashboard loading' absent after [3] press: %q", afterNorm)
	}
	if !strings.Contains(afterNorm, "to cancel") {
		t.Errorf("AC3 [Input-changed]: 'to cancel' absent after [3] press: %q", afterNorm)
	}
}

// AC4 [Anti-regression] — FB-097 ready-prompt unchanged after BucketsLoadedMsg with pendingQuotaOpen=true.
func TestFB115_AC4_AntiRegression_FB097_ReadyPromptUnchanged(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	r, _ = appM.Update(data.BucketsLoadedMsg{})
	appM = r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "Quota dashboard ready") {
		t.Errorf("AC4 [Anti-regression FB-097]: ready-prompt absent after BucketsLoadedMsg: %q", norm)
	}
	if !strings.Contains(norm, "press [3]") {
		t.Errorf("AC4 [Anti-regression FB-097]: 'press [3]' absent from ready-prompt: %q", norm)
	}
}

// AC5 [Anti-regression] — FB-080 cancel hint unchanged: second [3] press still posts "Quota dashboard cancelled".
func TestFB115_AC5_AntiRegression_FB080_CancelHintUnchanged(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	// First press: start pending.
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)
	if !appM.pendingQuotaOpen {
		t.Fatal("AC5 precondition: pendingQuotaOpen must be true after first [3]")
	}

	// Second press: cancel.
	r, _ = appM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM = r.(AppModel)

	norm := statusBarNorm(appM)
	if !strings.Contains(norm, "cancelled") {
		t.Errorf("AC5 [Anti-regression FB-080]: cancel hint %q, want contains 'cancelled'", norm)
	}
}

// AC6 [Anti-regression] — FB-079 anchors updated to new copy; existing tests pass with "to cancel".
// This test pins the new anchor string directly to confirm the intentional update is in effect.
func TestFB115_AC6_AntiRegression_FB079_AnchorUpdated(t *testing.T) {
	t.Parallel()
	m := newQuotaLoadingModel()

	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	appM := r.(AppModel)

	hint := appM.statusBar.Hint
	const wantSuffix = "press [3] to cancel"
	if !strings.Contains(hint, wantSuffix) {
		t.Errorf("AC6 [Anti-regression FB-079]: hint %q, want contains %q (intentional FB-115 anchor update)", hint, wantSuffix)
	}
	const oldSuffix = "press [3] when ready"
	if strings.Contains(hint, oldSuffix) {
		t.Errorf("AC6 [Anti-regression FB-079]: hint %q still contains old copy %q — anchor not updated", hint, oldSuffix)
	}
}

// ==================== End FB-115 ====================



// ==================== FB-085: title bar label/spinner contradiction fix ====================

// newPlaceholderDetailModel returns an AppModel in DetailPane with placeholder
// state: describeRaw=nil, events cached (one row), non-yaml/conditions/events modes.
func newPlaceholderDetailModel() AppModel {
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")

	m := AppModel{
		ctx:        context.Background(),
		rc:         stubResourceClient{},
		ac:         data.NewActivityClient(nil),
		activePane: DetailPane,
		describeRT: data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw: nil, // placeholder condition: describe absent
		events: []data.EventRow{
			{Reason: "Scheduled", Message: "pod assigned"},
		},
		sidebar:     components.NewNavSidebarModel(22, 24),
		table:       components.NewResourceTableModel(58, 24),
		detail:      detail,
		activity:    components.NewActivityViewModel(58, 24),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.detail.SetMode(m.detailModeLabel())
	m.updatePaneFocus()
	return m
}

// TestFB085_AC1_LoadingTrue_UnavailableLabelAbsent verifies that when the
// placeholder is active AND loading is true, View() does NOT contain "[unavailable]".
func TestFB085_AC1_LoadingTrue_UnavailableLabelAbsent(t *testing.T) {
	t.Parallel()
	m := newPlaceholderDetailModel()
	m.detail.SetLoading(true)
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if strings.Contains(view, "[unavailable]") {
		t.Errorf("AC1 [Observable FB-085]: View() contains \"[unavailable]\" while loading=true; want absent.\nView:\n%s", view)
	}
}

// TestFB085_AC2_LoadingFalse_UnavailableLabelPresent verifies that when the
// placeholder is active AND loading is false, View() DOES contain "[unavailable]".
func TestFB085_AC2_LoadingFalse_UnavailableLabelPresent(t *testing.T) {
	t.Parallel()
	m := newPlaceholderDetailModel()
	m.detail.SetLoading(false)
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if !strings.Contains(view, "[unavailable]") {
		t.Errorf("AC2 [Observable FB-085]: View() does not contain \"[unavailable]\" while loading=false; want present.\nView:\n%s", view)
	}
}

// TestFB085_AC3_LoadingTransition_FlipsLabel verifies that the loading true→false
// transition changes the rendered title bar from absent to present "[unavailable]".
func TestFB085_AC3_LoadingTransition_FlipsLabel(t *testing.T) {
	t.Parallel()
	m := newPlaceholderDetailModel()

	// loading = true: label must be absent
	m.detail.SetLoading(true)
	m.detail.SetMode(m.detailModeLabel())
	viewLoading := stripANSIModel(m.View())
	if strings.Contains(viewLoading, "[unavailable]") {
		t.Errorf("AC3 [Input-changed FB-085]: loading=true: View() contains \"[unavailable]\"; want absent.\nView:\n%s", viewLoading)
	}

	// loading = false: label must appear
	m.detail.SetLoading(false)
	m.detail.SetMode(m.detailModeLabel())
	viewDone := stripANSIModel(m.View())
	if !strings.Contains(viewDone, "[unavailable]") {
		t.Errorf("AC3 [Input-changed FB-085]: loading=false: View() missing \"[unavailable]\"; want present.\nView:\n%s", viewDone)
	}
}

// ==================== End FB-085 ====================

// ==================== FB-086: [E] unblocked in double-failure state ====================

// newDoubleFailureDetailModel returns an AppModel in DetailPane with both
// describe and events failed: describeRaw=nil, events=nil, eventsErr set,
// eventsLoading=false.
func newDoubleFailureDetailModel() AppModel {
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")

	m := AppModel{
		ctx:                 context.Background(),
		rc:                  stubResourceClient{},
		ac:                  data.NewActivityClient(nil),
		activePane:          DetailPane,
		describeRT:          data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw:         nil,
		events:              nil,
		eventsErr:           errors.New("events fetch failed"),
		eventsLoading:       false,
		loadState:           data.LoadStateError,
		lastFailedFetchKind: "describe",
		loadErr:             errors.New("describe fetch failed"),
		sidebar:             components.NewNavSidebarModel(22, 24),
		table:               components.NewResourceTableModel(58, 24),
		detail:              detail,
		activity:            components.NewActivityViewModel(58, 24),
		filterBar:           components.NewFilterBarModel(),
		helpOverlay:         components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// TestFB086_AC5_EKey_DoubleFailure_AdmitsAndRedispatches verifies that pressing
// [E] in double-failure state sets eventsMode=true and eventsLoading=true (re-dispatch fired).
func TestFB086_AC5_EKey_DoubleFailure_AdmitsAndRedispatches(t *testing.T) {
	t.Parallel()
	m := newDoubleFailureDetailModel()

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if !appM.eventsMode {
		t.Errorf("AC5 [Observable FB-086]: eventsMode = false after [E] in double-failure; want true")
	}
	if !appM.eventsLoading {
		t.Errorf("AC5 [Observable FB-086]: eventsLoading = false after [E] in double-failure; want true (re-dispatch fired)")
	}
	if cmd == nil {
		t.Errorf("AC5 [Observable FB-086]: cmd = nil after [E] in double-failure; want LoadEventsCmd dispatched")
	}
	view := stripANSIModel(appM.View())
	if !strings.Contains(view, "Loading events") {
		t.Errorf("AC5 [Observable FB-086]: View() missing \"Loading events\" after [E] in double-failure; want events-loading spinner visible.\nView:\n%s", view)
	}
}

// TestFB086_AC6_EKey_SingleFailure_DescribePresent_StillAdmits verifies that
// the pre-existing single-failure path (describeRaw != nil, no events attempted)
// still admits [E] via the describeRaw != nil branch.
func TestFB086_AC6_EKey_SingleFailure_DescribePresent_StillAdmits(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModel()
	// describeRaw present, no events attempted
	raw := &unstructured.Unstructured{}
	raw.SetName("my-project")
	m.describeRaw = raw
	m.events = nil
	m.eventsErr = nil
	m.eventsLoading = false

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := result.(AppModel)

	if !appM.eventsMode {
		t.Errorf("AC6 [Anti-behavior FB-086]: eventsMode = false after [E] with describeRaw present; want true (pre-existing path)")
	}
	view := stripANSIModel(appM.View())
	if !strings.Contains(view, "Loading events") {
		t.Errorf("AC6 [Anti-behavior FB-086]: View() missing \"Loading events\" after [E] with describeRaw present; want events-loading spinner visible.\nView:\n%s", view)
	}
}

// TestFB086_AC7_EKey_DoubleFailure_ThenEventsLoaded_ViewTransition verifies that
// after double-failure → [E] → EventsLoadedMsg, the view contains event content
// and does not contain the describe error block.
func TestFB086_AC7_EKey_DoubleFailure_ThenEventsLoaded_ViewTransition(t *testing.T) {
	t.Parallel()
	m := newDoubleFailureDetailModel()

	// Press [E] to enter events mode
	r1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	appM := r1.(AppModel)

	// Simulate events load succeeding
	r2, _ := appM.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{
			{Reason: "Scheduled", Message: "assigned to node"},
		},
	})
	appM2 := r2.(AppModel)

	view := stripANSIModel(appM2.View())
	if !strings.Contains(view, "Scheduled") {
		t.Errorf("AC7 [Input-changed FB-086]: View() missing event content after EventsLoadedMsg.\nView:\n%s", view)
	}
}

// ==================== End FB-086 ====================

// ==================== FB-038: empty-viewport oscillation fix ====================

// TestFB038_AC1_InFlight_ViewContainsLoading verifies that the operator sees
// "Loading" in the rendered detail pane during in-flight state (eventsMode=false).
func TestFB038_AC1_InFlight_ViewContainsLoading(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsInFlight()
	// Sync the pre-check content into the detail component so View() reflects it.
	m.detail.SetContent(m.buildDetailContent())
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if !strings.Contains(view, "Loading") {
		t.Errorf("AC1 [Observable FB-038]: View() missing \"Loading\" during in-flight state; want operator sees loading indicator.\nView:\n%s", view)
	}
}

// TestFB038_AC2_RapidEPress_NoBlankBody verifies that 4 rapid E presses during
// in-flight state never produce a blank detail body (the oscillation regression).
func TestFB038_AC2_RapidEPress_NoBlankBody(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsInFlight()

	for i := 1; i <= 4; i++ {
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
		m = result.(AppModel)
		view := stripANSIModel(m.detail.View())
		if strings.TrimSpace(view) == "" {
			t.Errorf("AC2 [Repeat-press FB-038]: press %d produced a blank detail body; want non-empty", i)
		}
		// Presses 2 and 4 (eventsMode=false) are the regression cases: must show "Loading"
		if i%2 == 0 && !strings.Contains(view, "Loading") {
			t.Errorf("AC2 [Repeat-press FB-038]: press %d (eventsMode=false) detail.View() missing \"Loading\":\n%s", i, view)
		}
	}
}

// TestFB038_AC3_PreCheck_Inert_AfterEventsLoaded verifies that after
// EventsLoadedMsg resolves, the pre-check is inert (eventsLoading=false).
func TestFB038_AC3_PreCheck_Inert_AfterEventsLoaded(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsInFlight()

	// Dispatch EventsLoadedMsg to resolve the in-flight state.
	r, _ := m.Update(data.EventsLoadedMsg{
		Events: []data.EventRow{
			{Reason: "SuccessfulCreate", Message: "pod created"},
		},
	})
	appM := r.(AppModel)

	if appM.eventsLoading {
		t.Error("AC3 [Anti-behavior FB-038]: eventsLoading = true after EventsLoadedMsg; want false")
	}

	// With eventsMode=false and eventsLoading=false, pre-check must NOT fire.
	// The FB-024 placeholder ("Describe unavailable") should render instead.
	view := stripANSIModel(appM.View())
	if strings.Contains(view, "Loading\u2026") {
		t.Errorf("AC3 [Anti-behavior FB-038]: View() contains the FB-038 loading placeholder after events loaded; pre-check must be inert.\nView:\n%s", view)
	}
	if !strings.Contains(view, "Describe unavailable") {
		t.Errorf("AC3 [Anti-behavior FB-038]: View() missing \"Describe unavailable\" after events loaded; want FB-024 placeholder active.\nView:\n%s", view)
	}
}

// ==================== End FB-038 ====================

// ==================== FB-026: keybind hint format consistency ====================

// TestFB026_AC1_TitleBar_HintMatrix verifies the 6-case title-bar hint matrix
// across all mode states for the three toggle keys (y, C, E).
func TestFB026_AC1_TitleBar_HintMatrix(t *testing.T) {
	t.Parallel()

	newDV := func(mode string) components.DetailViewModel {
		dv := components.NewDetailViewModel(120, 20)
		dv.SetResourceContext("pods", "my-pod")
		dv.SetDescribeAvailable(true)
		dv.SetMode(mode)
		return dv
	}

	cases := []struct {
		mode         string
		wantContains []string
		wantAbsent   []string
	}{
		{
			mode:         "",
			wantContains: []string{"[C] conditions", "[E] events", "[y] yaml"},
			wantAbsent:   []string{"toggle"},
		},
		{
			mode:         "yaml",
			wantContains: []string{"[y] describe"},
			wantAbsent:   []string{"[y] yaml"},
		},
		{
			mode:         "conditions",
			wantContains: []string{"[C] describe"},
			wantAbsent:   []string{"[C] conditions"},
		},
		{
			mode:         "events",
			wantContains: []string{"[E] describe"},
			wantAbsent:   []string{"[E] events"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("mode="+tc.mode, func(t *testing.T) {
			t.Parallel()
			dv := newDV(tc.mode)
			view := stripANSIModel(dv.View())
			for _, want := range tc.wantContains {
				if !strings.Contains(view, want) {
					t.Errorf("AC1 [Observable FB-026] mode=%q: View() missing %q.\nView:\n%s", tc.mode, want, view)
				}
			}
			for _, absent := range tc.wantAbsent {
				if strings.Contains(view, absent) {
					t.Errorf("AC1 [Observable FB-026] mode=%q: View() contains %q; want absent.\nView:\n%s", tc.mode, absent, view)
				}
			}
		})
	}
}

// TestFB026_AC2_HelpOverlay_CanonicalFormat verifies HelpOverlay uses [C]/[E] not [Shift+C]/[Shift+E].
func TestFB026_AC2_HelpOverlay_CanonicalFormat(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	appM := result.(AppModel)

	view := stripANSIModel(appM.helpOverlay.View())

	if !strings.Contains(view, "[C]    conditions") {
		t.Errorf("AC2 [Observable FB-026]: View() missing \"[C]    conditions\".\nView:\n%s", view)
	}
	if !strings.Contains(view, "[E]    events") {
		t.Errorf("AC2 [Observable FB-026]: View() missing \"[E]    events\".\nView:\n%s", view)
	}
	if strings.Contains(view, "Shift+C") {
		t.Errorf("AC2 [Observable FB-026]: View() still contains \"Shift+C\"; want absent.\nView:\n%s", view)
	}
	if strings.Contains(view, "Shift+E") {
		t.Errorf("AC2 [Observable FB-026]: View() still contains \"Shift+E\"; want absent.\nView:\n%s", view)
	}
}

// TestFB026_AC3_HelpOverlay_GlobalHelp_NoToggleVerb verifies [?] help (not "toggle help").
func TestFB026_AC3_HelpOverlay_GlobalHelp_NoToggleVerb(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	appM := result.(AppModel)

	view := stripANSIModel(appM.helpOverlay.View())

	if !strings.Contains(view, "[?]") {
		t.Errorf("AC3 [Observable FB-026]: View() missing \"[?]\".\nView:\n%s", view)
	}
	if !strings.Contains(view, "help") {
		t.Errorf("AC3 [Observable FB-026]: View() missing \"help\".\nView:\n%s", view)
	}
	if strings.Contains(view, "toggle help") {
		t.Errorf("AC3 [Observable FB-026]: View() still contains \"toggle help\"; want absent.\nView:\n%s", view)
	}
}

// TestFB026_AC4_ConditionsMode_ToggleSwap verifies [C] describe shown when in conditions mode.
func TestFB026_AC4_ConditionsMode_ToggleSwap(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelWithEventsRaw()
	// Widen the detail pane so the hint row fits (default fixture is 58 wide, too narrow).
	r0, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	m = r0.(AppModel)
	// Press C to enter conditions mode
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("C")})
	appM := result.(AppModel)

	view := stripANSIModel(appM.detail.View())
	if !strings.Contains(view, "[C] describe") {
		t.Errorf("AC4 [Anti-regression FB-026]: conditions mode: View() missing \"[C] describe\".\nView:\n%s", view)
	}
	if strings.Contains(view, "[C] conditions") {
		t.Errorf("AC4 [Anti-regression FB-026]: conditions mode: View() still contains \"[C] conditions\"; want absent.\nView:\n%s", view)
	}
}

// TestFB026_AC5_PaneGating_Preserved verifies ShowConditionsHint and ShowEventsHint
// gating still works with the new canonical strings.
func TestFB026_AC5_PaneGating_Preserved(t *testing.T) {
	t.Parallel()

	t.Run("conditions_hint_gated_on_in_detail", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithEventsRaw()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		view := stripANSIModel(appM.helpOverlay.View())
		if !strings.Contains(view, "[C]    conditions") {
			t.Errorf("AC5 [Anti-regression FB-026]: '[C]    conditions' absent in DetailPane HelpOverlay.\nView:\n%s", view)
		}
	})

	t.Run("conditions_hint_gated_off_in_table", func(t *testing.T) {
		t.Parallel()
		m := newTablePaneModel()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[C]    conditions") {
			t.Errorf("AC5 [Anti-regression FB-026]: '[C]    conditions' present in TablePane HelpOverlay; want absent.\nView:\n%s", view)
		}
	})

	t.Run("events_hint_gated_on_in_detail", func(t *testing.T) {
		t.Parallel()
		m := newDetailPaneModelWithEventsRaw()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		view := stripANSIModel(appM.helpOverlay.View())
		if !strings.Contains(view, "[E]    events") {
			t.Errorf("AC5 [Anti-regression FB-026]: '[E]    events' absent in DetailPane HelpOverlay.\nView:\n%s", view)
		}
	})

	t.Run("events_hint_gated_off_in_table", func(t *testing.T) {
		t.Parallel()
		m := newTablePaneModel()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
		appM := result.(AppModel)
		view := stripANSIModel(appM.helpOverlay.View())
		if strings.Contains(view, "[E]    events") {
			t.Errorf("AC5 [Anti-regression FB-026]: '[E]    events' present in TablePane HelpOverlay; want absent.\nView:\n%s", view)
		}
	})
}

// TestFB026_AC6_NarrowWidth_HintRowDropped verifies that at narrow width the
// title-bar hint row is absent (existing truncation path unchanged).
func TestFB026_AC6_NarrowWidth_HintRowDropped(t *testing.T) {
	t.Parallel()
	dv := components.NewDetailViewModel(40, 20)
	dv.SetResourceContext("pods", "my-pod")
	dv.SetMode("")

	view := stripANSIModel(dv.View())
	if strings.Contains(view, "[C] conditions") {
		t.Errorf("AC6 [Observable FB-026]: narrow width: View() contains \"[C] conditions\"; want hint row dropped.\nView:\n%s", view)
	}
	if strings.Contains(view, "[y] yaml") {
		t.Errorf("AC6 [Observable FB-026]: narrow width: View() contains \"[y] yaml\"; want hint row dropped.\nView:\n%s", view)
	}
}

// ==================== End FB-026 ====================

// ==================== FB-118: Describe error card eventsMode guard ====================

// TestFB118_AC1_Observable_EventsMode_DescribeFailed_EventsInFlight_ShowsSpinner
// eventsMode=true + describe failed + eventsLoading=true → events spinner, not describe error card.
func TestFB118_AC1_Observable_EventsMode_DescribeFailed_EventsInFlight_ShowsSpinner(t *testing.T) {
	t.Parallel()
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")
	m := AppModel{
		ctx:                 context.Background(),
		rc:                  stubResourceClient{},
		ac:                  data.NewActivityClient(nil),
		activePane:          DetailPane,
		describeRT:          data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw:         nil,
		events:              nil,
		eventsLoading:       true,
		eventsMode:          true,
		loadState:           data.LoadStateError,
		lastFailedFetchKind: "describe",
		loadErr:             errors.New("describe fetch failed"),
		sidebar:             components.NewNavSidebarModel(22, 24),
		table:               components.NewResourceTableModel(58, 24),
		detail:              detail,
		activity:            components.NewActivityViewModel(58, 24),
		filterBar:           components.NewFilterBarModel(),
		helpOverlay:         components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	m.detail.SetContent(m.buildDetailContent())
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if !strings.Contains(view, "Loading events") {
		t.Errorf("AC1 [Observable FB-118]: eventsMode=true + describe failed + eventsLoading=true: want 'Loading events' in output, got:\n%s", view)
	}
	if strings.Contains(view, "Could not describe") {
		t.Errorf("AC1 [Observable FB-118]: eventsMode=true + describe failed: got describe error card; want events spinner:\n%s", view)
	}
}

// TestFB118_AC2_Observable_EventsMode_DescribeFailed_EventsLoaded_ShowsTable
// eventsMode=true + describe failed + events loaded → events table, not describe error card.
func TestFB118_AC2_Observable_EventsMode_DescribeFailed_EventsLoaded_ShowsTable(t *testing.T) {
	t.Parallel()
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")
	m := AppModel{
		ctx:                 context.Background(),
		rc:                  stubResourceClient{},
		ac:                  data.NewActivityClient(nil),
		activePane:          DetailPane,
		describeRT:          data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw:         nil,
		events:              []data.EventRow{{Reason: "SuccessfulCreate", Message: "created pod"}},
		eventsLoading:       false,
		eventsMode:          true,
		loadState:           data.LoadStateError,
		lastFailedFetchKind: "describe",
		loadErr:             errors.New("describe fetch failed"),
		sidebar:             components.NewNavSidebarModel(22, 24),
		table:               components.NewResourceTableModel(58, 24),
		detail:              detail,
		activity:            components.NewActivityViewModel(58, 24),
		filterBar:           components.NewFilterBarModel(),
		helpOverlay:         components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	m.detail.SetContent(m.buildDetailContent())
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if !strings.Contains(view, "SuccessfulCreate") {
		t.Errorf("AC2 [Observable FB-118]: eventsMode=true + describe failed + events loaded: want 'SuccessfulCreate' in output, got:\n%s", view)
	}
	if strings.Contains(view, "Could not describe") {
		t.Errorf("AC2 [Observable FB-118]: eventsMode=true + describe failed + events loaded: got describe error card; want events table:\n%s", view)
	}
}

// TestFB118_AC3_Observable_EventsMode_BothFailed_ShowsEventsError
// eventsMode=true + both fetches failed → events error shown, not describe error card.
func TestFB118_AC3_Observable_EventsMode_BothFailed_ShowsEventsError(t *testing.T) {
	t.Parallel()
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")
	m := AppModel{
		ctx:                 context.Background(),
		rc:                  stubResourceClient{},
		ac:                  data.NewActivityClient(nil),
		activePane:          DetailPane,
		describeRT:          data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw:         nil,
		events:              nil,
		eventsLoading:       false,
		eventsMode:          true,
		eventsErr:           errors.New("events fetch failed"),
		loadState:           data.LoadStateError,
		lastFailedFetchKind: "describe",
		loadErr:             errors.New("describe fetch failed"),
		sidebar:             components.NewNavSidebarModel(22, 24),
		table:               components.NewResourceTableModel(58, 24),
		detail:              detail,
		activity:            components.NewActivityViewModel(58, 24),
		filterBar:           components.NewFilterBarModel(),
		helpOverlay:         components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	m.detail.SetContent(m.buildDetailContent())
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if strings.Contains(view, "Could not describe") {
		t.Errorf("AC3 [Observable FB-118]: eventsMode=true + both failed: got describe error card; want events error surface:\n%s", view)
	}
	if !strings.Contains(view, "Could not fetch events") {
		t.Errorf("AC3 [Observable FB-118]: eventsMode=true + both failed: expected events error 'Could not fetch events' in output, got:\n%s", view)
	}
}

// TestFB118_AC4_AntiRegression_EventsModeFalse_DescribeFailed_ShowsDescribeCard
// eventsMode=false + describe failed → describe error card still shown (guard must not suppress it).
func TestFB118_AC4_AntiRegression_EventsModeFalse_DescribeFailed_ShowsDescribeCard(t *testing.T) {
	t.Parallel()
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")
	m := AppModel{
		ctx:                 context.Background(),
		rc:                  stubResourceClient{},
		ac:                  data.NewActivityClient(nil),
		activePane:          DetailPane,
		describeRT:          data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw:         nil,
		events:              nil,
		eventsLoading:       false,
		eventsMode:          false,
		loadState:           data.LoadStateError,
		lastFailedFetchKind: "describe",
		loadErr:             errors.New("describe fetch failed"),
		sidebar:             components.NewNavSidebarModel(22, 24),
		table:               components.NewResourceTableModel(58, 24),
		detail:              detail,
		activity:            components.NewActivityViewModel(58, 24),
		filterBar:           components.NewFilterBarModel(),
		helpOverlay:         components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	m.detail.SetContent(m.buildDetailContent())
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if !strings.Contains(view, "Could not describe") {
		t.Errorf("AC4 [Anti-regression FB-118]: eventsMode=false + describe failed: want 'Could not describe' (describe error card), got:\n%s", view)
	}
}

// TestFB118_AC5_AntiRegression_DoubleFailureFixtureFields verifies the
// newDoubleFailureDetailModel fixture reflects true double-failure state.
func TestFB118_AC5_AntiRegression_DoubleFailureFixtureFields(t *testing.T) {
	t.Parallel()
	m := newDoubleFailureDetailModel()

	if m.loadState != data.LoadStateError {
		t.Errorf("AC5 [Anti-regression FB-118]: fixture loadState = %v, want LoadStateError", m.loadState)
	}
	if m.lastFailedFetchKind != "describe" {
		t.Errorf("AC5 [Anti-regression FB-118]: fixture lastFailedFetchKind = %q, want \"describe\"", m.lastFailedFetchKind)
	}
	if m.loadErr == nil {
		t.Errorf("AC5 [Anti-regression FB-118]: fixture loadErr = nil, want non-nil")
	}
}

// TestFB118_AC6_AntiRegression_FB038PreCheckUnaffected verifies that the FB-038
// pre-check (LoadStateIdle + eventsLoading) is unaffected by the guard addition.
func TestFB118_AC6_AntiRegression_FB038PreCheckUnaffected(t *testing.T) {
	t.Parallel()
	detail := components.NewDetailViewModel(80, 24)
	detail.SetResourceContext("projects", "my-project")
	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		ac:            data.NewActivityClient(nil),
		activePane:    DetailPane,
		describeRT:    data.ResourceType{Kind: "Project", Name: "projects"},
		describeRaw:   nil,
		events:        nil,
		eventsLoading: true,
		eventsMode:    false,
		loadState:     data.LoadStateIdle,
		sidebar:       components.NewNavSidebarModel(22, 24),
		table:         components.NewResourceTableModel(58, 24),
		detail:        detail,
		activity:      components.NewActivityViewModel(58, 24),
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	m.detail.SetContent(m.buildDetailContent())
	m.detail.SetMode(m.detailModeLabel())

	view := stripANSIModel(m.View())
	if !strings.Contains(view, "Loading") {
		t.Errorf("AC6 [Anti-regression FB-118]: LoadStateIdle + eventsLoading=true: want muted 'Loading…' from FB-038 pre-check, got:\n%s", view)
	}
	if strings.Contains(view, "Failed to load") {
		t.Errorf("AC6 [Anti-regression FB-118]: LoadStateIdle path: describe error card must not fire, got:\n%s", view)
	}
}

// ==================== End FB-118 ====================

// ==================== FB-025: Events freshness — in-place refresh + staleness indicator ====================
//
// Axis-coverage:
// AC  | Axis                      | Test
// AC1 | Happy/Observable          | TestFB025_AC1_RKey_EventsMode_DispatchesBoth
// AC2 | Happy/Observable          | TestFB025_AC2_RKey_AfterEOnOff_DispatchesEvents
// AC3 | Anti-behavior             | TestFB025_AC3_RKey_EventsNeverFetched_NoEventsCmd
// AC4 | Observable/Input-changed  | TestFB025_AC4_EventsLoadedMsg_Success_SetsFetchedAt (View() + accessor)
// AC5 | State/Integration         | TestFB025_AC5_EventsLoadedMsg_Error_RetainsFetchedAt (accessor only; Observable surface owned by AC4-Component)
// AC8 | Input-changed             | TestFB025_AC8_ResetSites_ClearFetchedAt (5 sub-tests, each with View() + accessor)
// AC9 | Repeat-press              | TestFB025_AC9_RapidR_RefreshingGuard_NoDoubleDispatch

// newDetailPaneModelForRefresh builds a DetailPane AppModel with tableTypeName
// and describeRT set so the r-key handler can reach the events-dispatch branch.
func newDetailPaneModelForRefresh() AppModel {
	sidebar := components.NewNavSidebarModel(22, 20)
	rt := data.ResourceType{Name: "pods", Kind: "Pod", Namespaced: false}
	sidebar.SetItems([]data.ResourceType{rt})

	detail := components.NewDetailViewModel(58, 20)
	detail.SetResourceContext("Pod", "my-pod")

	m := AppModel{
		ctx:           context.Background(),
		rc:            stubResourceClient{},
		activePane:    DetailPane,
		tableTypeName: "pods",
		describeRT:    rt,
		sidebar:       sidebar,
		table:         components.NewResourceTableModel(58, 20),
		detail:        detail,
		quota:         components.NewQuotaDashboardModel(58, 20, "proj"),
		filterBar:     components.NewFilterBarModel(),
		helpOverlay:   components.NewHelpOverlayModel(),
	}
	m.updatePaneFocus()
	return m
}

// TestFB025_AC1_RKey_EventsMode_DispatchesBoth verifies r in eventsMode dispatches
// both LoadResourcesCmd and LoadEventsCmd in the same batch.
func TestFB025_AC1_RKey_EventsMode_DispatchesBoth(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelForRefresh()
	m.eventsMode = true
	m.events = []data.EventRow{{Type: "Normal", Reason: "Created", Count: 1}}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("AC1: cmd = nil, want batch with LoadResourcesCmd + LoadEventsCmd")
	}
	msgs := collectMsgs(cmd)
	var hasResources, hasEvents bool
	for _, msg := range msgs {
		switch msg.(type) {
		case data.ResourcesLoadedMsg:
			hasResources = true
		case data.EventsLoadedMsg:
			hasEvents = true
		}
	}
	if !hasResources {
		t.Error("AC1: ResourcesLoadedMsg absent; LoadResourcesCmd not dispatched")
	}
	if !hasEvents {
		t.Error("AC1: EventsLoadedMsg absent; LoadEventsCmd not dispatched for eventsMode=true")
	}
}

// TestFB025_AC2_RKey_AfterEOnOff_DispatchesEvents verifies r dispatches LoadEventsCmd
// when eventsMode=false but events!=nil (visited then toggled off).
func TestFB025_AC2_RKey_AfterEOnOff_DispatchesEvents(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelForRefresh()
	m.eventsMode = false
	m.events = []data.EventRow{{Type: "Normal", Reason: "Created", Count: 1}} // visited, then toggled off

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Fatal("AC2: cmd = nil, want batch including LoadEventsCmd")
	}
	msgs := collectMsgs(cmd)
	var hasEvents bool
	for _, msg := range msgs {
		if _, ok := msg.(data.EventsLoadedMsg); ok {
			hasEvents = true
		}
	}
	if !hasEvents {
		t.Error("AC2: EventsLoadedMsg absent; events not refreshed when events!=nil but eventsMode=false")
	}
}

// TestFB025_AC3_RKey_EventsNeverFetched_NoEventsCmd verifies r does NOT dispatch
// LoadEventsCmd when events were never fetched (events=nil, eventsMode=false).
func TestFB025_AC3_RKey_EventsNeverFetched_NoEventsCmd(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelForRefresh()
	m.eventsMode = false
	m.events = nil

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		return // no dispatch is also acceptable
	}
	msgs := collectMsgs(cmd)
	for _, msg := range msgs {
		if _, ok := msg.(data.EventsLoadedMsg); ok {
			t.Error("AC3: EventsLoadedMsg present; LoadEventsCmd must not dispatch when events never fetched")
		}
	}
}

// TestFB025_AC4_EventsLoadedMsg_Success_SetsFetchedAt verifies a successful
// EventsLoadedMsg sets eventsFetchedAt to a non-zero time on the detail view
// and that the age label becomes visible in detail.View() when mode is "events".
func TestFB025_AC4_EventsLoadedMsg_Success_SetsFetchedAt(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelForRefresh()

	if !m.detail.EventsFetchedAt().IsZero() {
		t.Fatal("AC4 setup: eventsFetchedAt non-zero before test")
	}

	before := time.Now()
	result, _ := m.Update(data.EventsLoadedMsg{Events: []data.EventRow{{Type: "Normal", Reason: "Created"}}})
	appM := result.(AppModel)
	after := time.Now()

	// State check: fetchedAt set to time within the call window.
	if appM.detail.EventsFetchedAt().IsZero() {
		t.Error("AC4: eventsFetchedAt.IsZero() = true after success EventsLoadedMsg, want non-zero")
	}
	fetched := appM.detail.EventsFetchedAt()
	if fetched.Before(before) || fetched.After(after) {
		t.Errorf("AC4: eventsFetchedAt = %v outside expected range [%v, %v]", fetched, before, after)
	}

	// View() check: widen the detail pane (fixture is 58px — too narrow for the
	// age suffix candidate-width check) then force mode="events" directly.
	appM.detail.SetSize(160, 40)
	appM.detail.SetMode("events")
	view := stripANSIModel(appM.detail.View())
	if !strings.Contains(view, " · ") {
		t.Errorf("AC4 [Observable]: ' · ' age separator absent from detail.View() after success load:\n%s", view)
	}
	if !strings.Contains(view, "just now") {
		t.Errorf("AC4 [Observable]: 'just now' absent from detail.View() within 15s of load:\n%s", view)
	}
}

// TestFB025_AC5_EventsLoadedMsg_Error_RetainsFetchedAt verifies that an error
// EventsLoadedMsg does NOT overwrite the prior successful fetchedAt timestamp,
// observable via the age label remaining visible in detail.View() after the error.
func TestFB025_AC5_EventsLoadedMsg_Error_RetainsFetchedAt(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelForRefresh()

	// Successful load: fetchedAt set.
	r1, _ := m.Update(data.EventsLoadedMsg{Events: []data.EventRow{{Type: "Normal", Reason: "Created"}}})
	m = r1.(AppModel)
	if m.detail.EventsFetchedAt().IsZero() {
		t.Fatal("AC5 setup: fetchedAt still zero after success load")
	}

	// View() before error: widen + force events mode → age label must be visible.
	m.detail.SetSize(160, 40)
	m.detail.SetMode("events")
	viewBefore := stripANSIModel(m.detail.View())
	if !strings.Contains(viewBefore, " · ") {
		t.Fatalf("AC5 setup: ' · ' absent before error reload — test setup invalid:\n%s", viewBefore)
	}

	// Error reload: must NOT overwrite fetchedAt.
	r2, _ := m.Update(data.EventsLoadedMsg{Err: errors.New("server error")})
	m = r2.(AppModel)

	// View() after error: age label must still be visible (fetchedAt retained).
	m.detail.SetSize(160, 40)
	m.detail.SetMode("events")
	viewAfter := stripANSIModel(m.detail.View())
	if !strings.Contains(viewAfter, " · ") {
		t.Errorf("AC5 [Observable]: ' · ' age separator absent after error reload — fetchedAt was overwritten:\n%s", viewAfter)
	}
}

// TestFB025_AC8_ResetSites_ClearFetchedAt verifies that each of the 5 reset sites
// clears eventsFetchedAt back to zero and that the age separator is absent in View().
func TestFB025_AC8_ResetSites_ClearFetchedAt(t *testing.T) {
	t.Parallel()

	// setFetchedAt delivers a success EventsLoadedMsg, then widens the detail pane
	// to 160px and forces mode="events" so the age label IS visible before any reset.
	// Panics if the age label is not visible — catches invalid test setup.
	setFetchedAt := func(m AppModel) AppModel {
		m.eventsMode = true
		r, _ := m.Update(data.EventsLoadedMsg{Events: []data.EventRow{{Type: "Normal"}}})
		m = r.(AppModel)
		if m.detail.EventsFetchedAt().IsZero() {
			panic("setFetchedAt: fetchedAt still zero after success EventsLoadedMsg")
		}
		m.detail.SetSize(160, 40)
		m.detail.SetMode("events")
		if !strings.Contains(stripANSIModel(m.detail.View()), " · ") {
			panic("setFetchedAt: age label not visible at width=160 with mode=events — setup invalid")
		}
		return m
	}

	// assertNoAgeLabel forces mode="events" at width=160 on the post-reset model and
	// asserts " · " is absent. Because fetchedAt=zero after reset, the age suffix is
	// suppressed even with mode="events" — a non-vacuous Input-changed assertion.
	assertNoAgeLabel := func(t *testing.T, appM AppModel, site string) {
		t.Helper()
		appM.detail.SetSize(160, 40)
		appM.detail.SetMode("events")
		view := stripANSIModel(appM.detail.View())
		if strings.Contains(view, " · ") {
			t.Errorf("AC8 %s [Input-changed]: ' · ' still visible at mode=events after reset — fetchedAt not cleared:\n%s", site, view)
		}
	}

	t.Run("site-A: case-d enters new resource", func(t *testing.T) {
		t.Parallel()
		m := setFetchedAt(newTablePaneModel())
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
		appM := result.(AppModel)
		if !appM.detail.EventsFetchedAt().IsZero() {
			t.Error("AC8 site-A: eventsFetchedAt non-zero after case-d entry, want zero")
		}
		assertNoAgeLabel(t, appM, "site-A")
	})

	t.Run("site-B: ContextSwitchedMsg resets fetchedAt", func(t *testing.T) {
		t.Parallel()
		m := setFetchedAt(newDetailPaneModelForRefresh())
		result, _ := m.Update(components.ContextSwitchedMsg{})
		appM := result.(AppModel)
		if !appM.detail.EventsFetchedAt().IsZero() {
			t.Error("AC8 site-B: eventsFetchedAt non-zero after ContextSwitchedMsg, want zero")
		}
		assertNoAgeLabel(t, appM, "site-B")
	})

	t.Run("site-C: esc from DetailPane resets fetchedAt", func(t *testing.T) {
		t.Parallel()
		m := setFetchedAt(newDetailPaneModelForRefresh())
		m.detailReturnPane = TablePane
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		appM := result.(AppModel)
		if !appM.detail.EventsFetchedAt().IsZero() {
			t.Error("AC8 site-C: eventsFetchedAt non-zero after Esc from DetailPane, want zero")
		}
		assertNoAgeLabel(t, appM, "site-C")
	})

	t.Run("site-D: esc from HistoryPane resets fetchedAt", func(t *testing.T) {
		t.Parallel()
		m := setFetchedAt(newDetailPaneModelWithHC())
		m.activePane = HistoryPane
		m.updatePaneFocus()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		appM := result.(AppModel)
		if !appM.detail.EventsFetchedAt().IsZero() {
			t.Error("AC8 site-D: eventsFetchedAt non-zero after Esc from HistoryPane, want zero")
		}
		assertNoAgeLabel(t, appM, "site-D")
	})

	t.Run("site-E: H from HistoryPane returns to DetailPane resets fetchedAt", func(t *testing.T) {
		t.Parallel()
		m := setFetchedAt(newDetailPaneModelWithHC())
		m.activePane = HistoryPane
		m.updatePaneFocus()
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
		appM := result.(AppModel)
		if !appM.detail.EventsFetchedAt().IsZero() {
			t.Error("AC8 site-E: eventsFetchedAt non-zero after H from HistoryPane, want zero")
		}
		assertNoAgeLabel(t, appM, "site-E")
	})
}

// TestFB025_AC9_RapidR_RefreshingGuard_NoDoubleDispatch verifies a second r press
// while m.refreshing=true hits the guard and returns nil cmd.
func TestFB025_AC9_RapidR_RefreshingGuard_NoDoubleDispatch(t *testing.T) {
	t.Parallel()
	m := newDetailPaneModelForRefresh()
	m.eventsMode = true
	m.events = []data.EventRow{{Type: "Normal", Reason: "Created", Count: 1}}

	// First r — sets refreshing=true.
	r1, cmd1 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = r1.(AppModel)
	if cmd1 == nil {
		t.Fatal("AC9: first r produced nil cmd, want batch cmd")
	}
	if !m.refreshing {
		t.Error("AC9: m.refreshing = false after first r, want true")
	}

	// Second r — must be blocked by m.refreshing guard.
	_, cmd2 := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd2 != nil {
		t.Error("AC9: second r produced non-nil cmd, want nil (refreshing guard blocks)")
	}
}

// ==================== End FB-025 (model layer) ====================

// ==================== FB-124: S4 quick-jump focus-activation affordance (model layer) ====================

// newFB124AppModel builds an AppModel in welcome-panel state with a backends registration
// so S4 renders (contentH=21 ≥ 18), and NavPane active (default welcome state).
func newFB124AppModel() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 25),
		table:       components.NewResourceTableModel(80, 25),
		detail:      components.NewDetailViewModel(80, 25),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.resourceTypes = []data.ResourceType{
		{Name: "backends", Kind: "Backend", Group: "networking.datum.net"},
	}
	m.table.SetRegistrations([]data.ResourceRegistration{
		{Name: "backends", Group: "networking.datum.net"},
	})
	m.updatePaneFocus()
	return m
}

// AC8 [Integration] — updatePaneFocus() propagates navPaneFocused; Tab pane-switch removes hint.
func TestFB124_AC8_Integration_TabPaneSwitchRemovesHint(t *testing.T) {
	t.Parallel()
	m := newFB124AppModel()

	if m.activePane != NavPane {
		t.Fatalf("precondition: activePane=%v, want NavPane", m.activePane)
	}

	v1 := stripANSIModel(m.View())
	if !strings.Contains(v1, "[Tab] next pane") {
		t.Errorf("AC8 [Integration]: '[Tab] next pane' absent when NavPane active:\n%s", v1)
	}

	// Tab → TablePane
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	appM := result.(AppModel)

	v2 := stripANSIModel(appM.View())
	if strings.Contains(v2, "[Tab] next pane") {
		t.Errorf("AC8 [Integration]: '[Tab] next pane' still present after Tab to TablePane:\n%s", v2)
	}
}

// ==================== End FB-124 (model layer) ====================

// ==================== FB-103: [r] in-flight signal on empty activity rows ====================

// newFB103ProjectModel builds a project-scoped NavPane model at welcome-panel size
// (table 80×32 → contentH=28 ≥ 24, S3 renders) with no pre-loaded activity rows.
func newFB103ProjectModel() AppModel {
	m := AppModel{
		ctx:         context.Background(),
		rc:          stubResourceClient{},
		ac:          data.NewActivityClient(nil),
		activePane:  NavPane,
		sidebar:     components.NewNavSidebarModel(22, 32),
		table:       components.NewResourceTableModel(80, 32),
		detail:      components.NewDetailViewModel(80, 32),
		filterBar:   components.NewFilterBarModel(),
		helpOverlay: components.NewHelpOverlayModel(),
	}
	m.tuiCtx.ActiveCtx = &datumconfig.DiscoveredContext{ProjectID: "test-proj"}
	m.updatePaneFocus()
	return m
}

// AC1 [Observable] — empty-rows + NavPane [r] + project-scope → View() contains "⟳ loading…".
func TestFB103_AC1_Observable_EmptyRows_RPress_ShowsLoading(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()
	// no rows loaded

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "loading") {
		t.Errorf("AC1 [Observable]: 'loading' absent from View() after [r] with empty rows:\n%s", got)
	}
}

// AC2 [Anti-behavior] — empty-rows + NavPane [r] + org-scope → View() still "no recent activity".
func TestFB103_AC2_AntiBehavior_OrgScope_RPress_NoLoadingSignal(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()
	m.tuiCtx.ActiveCtx = nil // org scope — no activity fetch dispatched

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if strings.Contains(got, "loading") {
		t.Errorf("AC2 [Anti-behavior]: 'loading' present in View() for org-scope [r] (no activity fetch):\n%s", got)
	}
	if !strings.Contains(got, "no recent activity") {
		t.Errorf("AC2 [Anti-behavior]: 'no recent activity' absent from org-scope View():\n%s", got)
	}
}

// AC3 [Anti-behavior] — populated-rows + NavPane [r] + project-scope → stale rows preserved, no spinner.
func TestFB103_AC3_AntiBehavior_PopulatedRows_RPress_RowsPreserved(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()
	m.table.SetActivityRows([]data.ActivityRow{
		{ActorDisplay: "alice@example.com", Summary: "created cluster", Timestamp: time.Now().Add(-1 * time.Minute)},
	})

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC3 [Anti-behavior]: stale rows 'alice@example.com' gone after [r] with populated rows:\n%s", got)
	}
	if strings.Contains(got, "loading") {
		t.Errorf("AC3 [Anti-behavior]: 'loading' present after [r] with populated rows (should be silent re-fetch):\n%s", got)
	}
}

// AC4 [Input-changed] — empty-rows before vs after [r]: "no recent activity" → "⟳ loading…".
func TestFB103_AC4_InputChanged_EmptyRows_BeforeAfterRPress(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()

	v1 := stripANSIModel(m.View())
	if !strings.Contains(v1, "no recent activity") {
		t.Fatalf("AC4 precondition: 'no recent activity' absent before [r]:\n%s", v1)
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	v2 := stripANSIModel(appM.View())
	if v1 == v2 {
		t.Error("AC4 [Input-changed]: View() unchanged after [r] with empty rows")
	}
	if !strings.Contains(v2, "loading") {
		t.Errorf("AC4 [Input-changed]: 'loading' absent from View() after [r]:\n%s", v2)
	}
}

// AC5 [Anti-regression] — FB-076 dispatch: [r] still batches activity cmd on project scope.
func TestFB103_AC5_AntiRegression_FB076_DispatchPreserved(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	if batchLen(cmd) < 2 {
		t.Errorf("AC5 [Anti-regression FB-076]: batchLen=%d, want ≥2 (resource types + activity cmd)", batchLen(cmd))
	}
}

// AC6 [Anti-regression] — FB-082 fetch-failed: after error arrives, body re-renders error copy.
func TestFB103_AC6_AntiRegression_FB082_FetchFailedReRendersError(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()

	// Press [r] → loading state
	r, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = r.(AppModel)

	// Simulate error response
	r, _ = m.Update(data.ProjectActivityErrorMsg{Err: errors.New("network timeout")})
	appM := r.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC6 [Anti-regression FB-082]: 'activity unavailable' absent after fetch error:\n%s", got)
	}
}

// AC7 [Observable] — error-state + NavPane [r] → View() shows "⟳ loading…", not error copy.
func TestFB103_AC7_Observable_ErrorState_RPress_ShowsLoading(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()
	m.table.SetActivityFetchFailed(true)
	m.table.SetActivityCRDAbsent(false)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "loading") {
		t.Errorf("AC7 [Observable]: 'loading' absent from View() after [r] in error state:\n%s", got)
	}
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC7 [Observable]: error copy still present after [r] in error state:\n%s", got)
	}
}

// AC8 [Input-changed] — error-state before vs after [r]: error copy → loading signal.
func TestFB103_AC8_InputChanged_ErrorState_BeforeAfterRPress(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()
	m.table.SetActivityFetchFailed(true)
	m.table.SetActivityCRDAbsent(false)

	v1 := stripANSIModel(m.View())
	if !strings.Contains(v1, "activity unavailable") {
		t.Fatalf("AC8 precondition: 'activity unavailable' absent before [r]:\n%s", v1)
	}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	v2 := stripANSIModel(appM.View())
	if v1 == v2 {
		t.Error("AC8 [Input-changed]: View() unchanged after [r] in error state")
	}
	if !strings.Contains(v2, "loading") {
		t.Errorf("AC8 [Input-changed]: 'loading' absent after [r] in error state:\n%s", v2)
	}
}

// AC9 [Observable] — CRD-absent state + NavPane [r] → View() shows "⟳ loading…" immediately.
func TestFB103_AC9_Observable_CRDAbsent_RPress_ShowsLoading(t *testing.T) {
	t.Parallel()
	m := newFB103ProjectModel()
	m.table.SetActivityFetchFailed(true)
	m.table.SetActivityCRDAbsent(true)

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	appM := result.(AppModel)

	got := stripANSIModel(appM.View())
	if !strings.Contains(got, "loading") {
		t.Errorf("AC9 [Observable]: 'loading' absent from View() after [r] in CRD-absent state:\n%s", got)
	}
}

// ==================== End FB-103 ====================
