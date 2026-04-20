package components

import (
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/tui/data"
)

// testHistoryRows returns a fixture slice: 3 rows (oldest→newest), with sources
// human, system, human so the c-filter has a non-trivial result.
func testHistoryRows() []data.HistoryRow {
	return []data.HistoryRow{
		{Rev: 1, User: "alice@example.com", UserDisp: "alice@example.com", Source: "human", Verb: "create", Summary: "Created", Parseable: true, Timestamp: time.Now().Add(-2 * time.Hour)},
		{Rev: 2, User: "system:reconciler", UserDisp: "system:reconciler", Source: "system", Verb: "update", Summary: "metadata only", Parseable: true, Timestamp: time.Now().Add(-1 * time.Hour)},
		{Rev: 3, User: "bob@example.com", UserDisp: "bob@example.com", Source: "human", Verb: "update", Summary: "spec.nodeName", Parseable: true, Timestamp: time.Now()},
	}
}

// --- HistoryViewModel state rendering ---

func TestHistoryViewModel_View_LoadingState(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetLoading(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "loading history") {
		t.Errorf("loading state: want 'loading history' in view, got %q", got)
	}
}

func TestHistoryViewModel_View_ErrorState(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetError(errTest("context deadline exceeded"), false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Could not load history") {
		t.Errorf("error state: want 'Could not load history', got %q", got)
	}
	if !strings.Contains(got, "context deadline exceeded") {
		t.Errorf("error state: want error detail in view, got %q", got)
	}
	if !strings.Contains(got, "[r] retry") {
		t.Errorf("error state: want '[r] retry' hint, got %q", got)
	}
}

func TestHistoryViewModel_View_UnauthorizedState(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetError(errTest("forbidden"), true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Audit history is not enabled") {
		t.Errorf("unauthorized state: want 'Audit history is not enabled', got %q", got)
	}
	if strings.Contains(got, "[r] retry") {
		t.Errorf("unauthorized state: should NOT show '[r] retry' (retry won't fix 403), got %q", got)
	}
	if !strings.Contains(got, "[Esc] back") {
		t.Errorf("unauthorized state: want '[Esc] back' hint, got %q", got)
	}
}

func TestHistoryViewModel_View_EmptyState(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetRows([]data.HistoryRow{}, false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "No change history recorded") {
		t.Errorf("empty state: want 'No change history recorded', got %q", got)
	}
	if !strings.Contains(got, "30 days") {
		t.Errorf("empty state: want 30-day caveat text, got %q", got)
	}
}

func TestHistoryViewModel_View_WithRows_ShowsColumnHeader(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(120, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetRows(testHistoryRows(), false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "REV") {
		t.Errorf("rows state: want 'REV' column header, got %q", got)
	}
	if !strings.Contains(got, "TIMESTAMP") {
		t.Errorf("rows state: want 'TIMESTAMP' column header at width≥100, got %q", got)
	}
}

func TestHistoryViewModel_View_WithRows_EndOfHistoryMarker(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetResourceContext("Pod", "my-pod")
	m.SetRows(testHistoryRows(), false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "end of history") {
		t.Errorf("rows state: want '— end of history —' marker, got %q", got)
	}
}

// --- ToggleHumanFilter ---

func TestHistoryViewModel_ToggleHumanFilter_FirstPress_FiltersToHumanRows(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	// Initial state: 3 visible (all rows).
	if len(m.visibleIdx) != 3 {
		t.Fatalf("precondition: visibleIdx len = %d, want 3", len(m.visibleIdx))
	}

	m.ToggleHumanFilter()

	if !m.filterHuman {
		t.Error("ToggleHumanFilter: filterHuman = false after first press, want true")
	}
	// testHistoryRows has 2 human rows (rev 1 and rev 3).
	if len(m.visibleIdx) != 2 {
		t.Errorf("ToggleHumanFilter first-press: visibleIdx len = %d, want 2 (human only)", len(m.visibleIdx))
	}
	// All visible rows must be human.
	for _, idx := range m.visibleIdx {
		if m.rows[idx].Source != "human" {
			t.Errorf("ToggleHumanFilter: visible row at rows[%d] has source %q, want 'human'", idx, m.rows[idx].Source)
		}
	}
}

func TestHistoryViewModel_ToggleHumanFilter_SecondPress_RestoresAllRows(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	// Simulate cursor on the system row (index 1 in visibleIdx when all shown = newest first,
	// so visibleIdx = [2,1,0] — cursor=1 points to rows[1]=system).
	m.cursor = 1
	m.ToggleHumanFilter() // on
	m.ToggleHumanFilter() // off

	if m.filterHuman {
		t.Error("ToggleHumanFilter second-press: filterHuman = true, want false")
	}
	if len(m.visibleIdx) != 3 {
		t.Errorf("ToggleHumanFilter second-press: visibleIdx len = %d, want 3 (all rows)", len(m.visibleIdx))
	}
}

func TestHistoryViewModel_ToggleHumanFilter_ZeroHumanRows_EmptyVisible(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	// Only system rows.
	rows := []data.HistoryRow{
		{Rev: 1, Source: "system", Verb: "update", Parseable: true},
		{Rev: 2, Source: "system", Verb: "update", Parseable: true},
	}
	m.SetRows(rows, false)
	m.ToggleHumanFilter()

	if len(m.visibleIdx) != 0 {
		t.Errorf("ToggleHumanFilter zero-human: visibleIdx len = %d, want 0", len(m.visibleIdx))
	}
	// View must not crash and should render empty notice.
	got := stripANSI(m.View())
	if !strings.Contains(got, "No human-source revisions") {
		t.Errorf("ToggleHumanFilter zero-human: want 'No human-source revisions' in view, got %q", got)
	}
}

func TestHistoryViewModel_ToggleHumanFilter_FilterBannerVisible(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.ToggleHumanFilter()

	got := stripANSI(m.View())
	if !strings.Contains(got, "filter: human only") {
		t.Errorf("filter active: want 'filter: human only' banner, got %q", got)
	}
	// X of Y count.
	if !strings.Contains(got, "2 of 3") {
		t.Errorf("filter active: want '2 of 3' count in banner, got %q", got)
	}
}

func TestHistoryViewModel_ToggleHumanFilter_FilterBannerAbsentWhenOff(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.ToggleHumanFilter() // on
	m.ToggleHumanFilter() // off

	got := stripANSI(m.View())
	if strings.Contains(got, "filter: human only") {
		t.Errorf("filter off: 'filter: human only' banner should not be visible, got %q", got)
	}
}

// --- ResetFilter ---

func TestHistoryViewModel_ResetFilter_ClearsFilterWithoutRestoringCursor(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.cursor = 2
	m.ToggleHumanFilter() // on: cursor → 0, preFilterCur = 2
	m.ResetFilter()

	if m.filterHuman {
		t.Error("ResetFilter: filterHuman = true after reset, want false")
	}
	if len(m.visibleIdx) != 3 {
		t.Errorf("ResetFilter: visibleIdx len = %d, want 3 (all rows)", len(m.visibleIdx))
	}
	// cursor is not restored by ResetFilter (that's ToggleHumanFilter's job).
}

// --- SelectedRow ---

func TestHistoryViewModel_SelectedRow_NewestFirst(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)

	// cursor=0 should be the newest row (rev 3).
	row, idx, ok := m.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false, want true")
	}
	if row.Rev != 3 {
		t.Errorf("SelectedRow().Rev = %d, want 3 (newest first)", row.Rev)
	}
	if idx != 2 {
		t.Errorf("SelectedRow() manifest idx = %d, want 2 (0-based oldest-first)", idx)
	}
}

func TestHistoryViewModel_SelectedRow_EmptyRows_ReturnsFalse(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	_, _, ok := m.SelectedRow()
	if ok {
		t.Error("SelectedRow() on empty model returned true, want false")
	}
}

// --- CursorUp / CursorDown ---

func TestHistoryViewModel_CursorDown_MovesToOlderRevision(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	// cursor=0 = newest (rev 3). CursorDown → rev 2.
	m.CursorDown()
	row, _, ok := m.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() false after CursorDown")
	}
	if row.Rev != 2 {
		t.Errorf("CursorDown: Rev = %d, want 2 (system row)", row.Rev)
	}
}

func TestHistoryViewModel_CursorUp_MovesToNewerRevision(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.CursorDown() // → rev 2
	m.CursorUp()   // → rev 3
	row, _, ok := m.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() false after CursorUp")
	}
	if row.Rev != 3 {
		t.Errorf("CursorUp: Rev = %d, want 3 (newest)", row.Rev)
	}
}

func TestHistoryViewModel_CursorBottom_MovesToOldestRevision(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.CursorBottom()
	row, _, ok := m.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() false after CursorBottom")
	}
	if row.Rev != 1 {
		t.Errorf("CursorBottom: Rev = %d, want 1 (oldest)", row.Rev)
	}
}

func TestHistoryViewModel_CursorTop_MovesToNewestRevision(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.CursorBottom()
	m.CursorTop()
	row, _, ok := m.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() false after CursorTop")
	}
	if row.Rev != 3 {
		t.Errorf("CursorTop: Rev = %d, want 3 (newest)", row.Rev)
	}
}

// --- HasRows ---

func TestHistoryViewModel_HasRows(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	if m.HasRows() {
		t.Error("HasRows() = true on empty model, want false")
	}
	m.SetRows(testHistoryRows(), false)
	if !m.HasRows() {
		t.Error("HasRows() = false after SetRows, want true")
	}
}

// --- Reset ---

func TestHistoryViewModel_Reset_ClearsAllState(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.ToggleHumanFilter()
	m.CursorDown()
	m.Reset()

	if m.HasRows() {
		t.Error("Reset: HasRows() = true after reset, want false")
	}
	if m.filterHuman {
		t.Error("Reset: filterHuman = true after reset, want false")
	}
	if m.cursor != 0 {
		t.Errorf("Reset: cursor = %d after reset, want 0", m.cursor)
	}
}

// --- SetRows preserves filter flag state (10f input-changed) ---

// TestHistoryViewModel_SetRows_FilterFlagSurvives verifies that calling SetRows
// while the human filter is active re-applies the filter to the new rows.
// This is the 10f input-changed axis: "filter persists on in-pane refresh."
func TestHistoryViewModel_SetRows_FilterFlagSurvives(t *testing.T) {
	t.Parallel()
	m := NewHistoryViewModel(80, 20)
	m.SetRows(testHistoryRows(), false)
	m.ToggleHumanFilter() // filterHuman = true; 2 human rows visible

	// Simulate a refresh arriving: SetRows called again with same fixture.
	m.SetRows(testHistoryRows(), false)

	// filterHuman must survive the SetRows call.
	if !m.filterHuman {
		t.Error("SetRows: filterHuman = false after refresh with filter on, want true (filter persists)")
	}
	// visibleIdx must be rebuilt with the filter still applied: 2 human rows.
	if len(m.visibleIdx) != 2 {
		t.Errorf("SetRows with filter: visibleIdx len = %d, want 2 (human only)", len(m.visibleIdx))
	}
	for _, idx := range m.visibleIdx {
		if m.rows[idx].Source != "human" {
			t.Errorf("SetRows with filter: visible rows[%d].Source = %q, want 'human'", idx, m.rows[idx].Source)
		}
	}
}

// errTest is a minimal error for test fixtures.
type errTest string

func (e errTest) Error() string { return string(e) }
