package components

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"go.datum.net/datumctl/internal/console/data"
)

var ansiEscapeActivity = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSIActivity(s string) string {
	return ansiEscapeActivity.ReplaceAllString(s, "")
}

// newActivityDashboard builds an ActivityDashboardModel with the given dimensions.
// Uses a generous size by default to avoid height-band collapsing in most tests.
func newActivityDashboard(width, height int) ActivityDashboardModel {
	return NewActivityDashboardModel(width, height, "test-project")
}

// testRows returns a slice of ActivityRow with predictable fields for assertions.
func testRows(n int) []data.ActivityRow {
	rows := make([]data.ActivityRow, n)
	for i := 0; i < n; i++ {
		rows[i] = data.ActivityRow{
			Timestamp:    time.Now().Add(-time.Duration(i+1) * time.Hour),
			ActorDisplay: "alice@example.com",
			Summary:      "updated spec",
			ResourceRef: &data.ResourceRef{
				APIGroup: "apps",
				Kind:     "Deployment",
				Name:     "checkout-api",
			},
		}
	}
	return rows
}

// ==================== State 1: Loading ====================

// TestActivityDashboardModel_Loading verifies State 1: while loading=true,
// the pane shows the "loading recent activity" spinner placeholder.
func TestActivityDashboardModel_Loading(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoading(true)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "loading recent activity") {
		t.Errorf("State 1: want 'loading recent activity' when loading=true, got: %q", got)
	}
	// [r] refresh must be ABSENT while loading (AC#15 gate).
	if strings.Contains(got, "[r]") {
		t.Errorf("State 1: [r] must be absent while loading, got: %q", got)
	}
}

// ==================== State 2: Loaded with rows (AC#1) ====================

// TestActivityDashboardModel_LoadedWithRows verifies AC#1/State 2: loaded rows
// appear in the view with the event count in the header.
func TestActivityDashboardModel_LoadedWithRows(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetRows(testRows(5))

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "5 events") {
		t.Errorf("AC#1: want '5 events' in header, got: %q", got)
	}
	if !strings.Contains(got, "checkout-api") {
		t.Errorf("AC#1: want resource name 'checkout-api' in rows, got: %q", got)
	}
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC#1: want actor 'alice@example.com' in rows (wide band), got: %q", got)
	}
	// Loading placeholder must be absent.
	if strings.Contains(got, "loading recent activity") {
		t.Errorf("AC#1: 'loading recent activity' must be absent when rows loaded, got: %q", got)
	}
}

// TestActivityDashboardModel_LoadedWithRows_RepeatEntry verifies AC#1 repeat-press:
// calling View() multiple times returns stable content (no re-fetch, no spinner).
func TestActivityDashboardModel_LoadedWithRows_RepeatEntry(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetRows(testRows(3))

	v1 := stripANSIActivity(m.View())
	v2 := stripANSIActivity(m.View())
	if v1 != v2 {
		t.Errorf("AC#1 repeat: View() not stable across calls:\nfirst:  %q\nsecond: %q", v1, v2)
	}
	if strings.Contains(v1, "loading recent activity") {
		t.Errorf("AC#1 repeat: spinner must not appear on repeat view of loaded state")
	}
}

// ==================== State 3: Empty (AC#5) ====================

// TestActivityDashboardModel_EmptyState verifies AC#5/State 3: when rows is nil
// and no error, the pane renders the "No recent human activity" message.
func TestActivityDashboardModel_EmptyState(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	// rows is nil by default; explicitly set empty to confirm no-error path.
	m.SetRows([]data.ActivityRow{})

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "No recent human activity") {
		t.Errorf("AC#5: want 'No recent human activity' in empty state, got: %q", got)
	}
	// "events" count suffix must NOT appear in the header (no rows).
	if strings.Contains(got, "events") {
		t.Errorf("AC#5: 'events' must be absent when state is empty, got: %q", got)
	}
}

// ==================== State 4: Unauthorized (AC#6) ====================

// TestActivityDashboardModel_Unauthorized verifies AC#6/State 4: 403 response
// renders "insufficient permissions".
func TestActivityDashboardModel_Unauthorized(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(errors.New("forbidden"), true, false)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "insufficient permissions") {
		t.Errorf("AC#6: want 'insufficient permissions' for unauthorized state, got: %q", got)
	}
	// [r] refresh must be absent (retry won't fix 403).
	if strings.Contains(got, "[r]") {
		t.Errorf("AC#6: [r] must be absent in unauthorized state (retry won't fix 403), got: %q", got)
	}
}

// ==================== State 5: CRD-absent (AC#7) ====================

// TestActivityDashboardModel_CRDAbsent verifies AC#7/State 5: ErrActivityCRDAbsent
// renders the "not available on this cluster" message (one-shot session flag).
func TestActivityDashboardModel_CRDAbsent(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(data.ErrActivityCRDAbsent, false, true)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "not available on this cluster") {
		t.Errorf("AC#7: want 'not available on this cluster' for CRD-absent, got: %q", got)
	}
	// [r] refresh must be absent (not retryable).
	if strings.Contains(got, "[r]") {
		t.Errorf("AC#7: [r] must be absent in CRD-absent state, got: %q", got)
	}
}

// TestActivityDashboardModel_CRDAbsent_ClearOnContextSwitch verifies AC#7 input-changed:
// ClearCRDAbsentFlag resets crdAbsent so the next entry would re-fetch.
func TestActivityDashboardModel_CRDAbsent_ClearOnContextSwitch(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(data.ErrActivityCRDAbsent, false, true)
	if !m.CRDAbsent() {
		t.Fatal("precondition: CRDAbsent() must be true after SetLoadErr with crdAbsent=true")
	}

	m.ClearCRDAbsentFlag()

	if m.CRDAbsent() {
		t.Error("AC#7 input-changed: CRDAbsent() = true after ClearCRDAbsentFlag, want false")
	}
	got := stripANSIActivity(m.View())
	if strings.Contains(got, "not available on this cluster") {
		t.Errorf("AC#7 input-changed: CRD-absent message must be absent after ClearCRDAbsentFlag, got: %q", got)
	}
}

// ==================== State 5: CRD-partial (AC#8) ====================

// TestActivityDashboardModel_CRDPartial verifies AC#8: ErrActivityCRDPartial
// renders the same "not available on this cluster" as CRD-absent (collapsed per D4).
func TestActivityDashboardModel_CRDPartial(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(data.ErrActivityCRDPartial, false, true)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "not available on this cluster") {
		t.Errorf("AC#8: want 'not available on this cluster' for CRD-partial (same render as CRD-absent per D4), got: %q", got)
	}
}

// ==================== State 6: Transient error (AC#9) ====================

// TestActivityDashboardModel_TransientError verifies AC#9/State 6: a generic
// 5xx error renders "temporarily unavailable" with an inline [r] retry affordance.
func TestActivityDashboardModel_TransientError(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(errors.New("server error: 500 internal server error"), false, false)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "temporarily unavailable") {
		t.Errorf("AC#9: want 'temporarily unavailable' for transient error, got: %q", got)
	}
	if !strings.Contains(got, "retry") {
		t.Errorf("AC#9: want 'retry' affordance for transient error, got: %q", got)
	}
}

// TestActivityDashboardModel_TransientError_Input_Changed verifies AC#9 (input-changed):
// unauthorized error does NOT render the retry affordance (distinct from state 6).
func TestActivityDashboardModel_TransientError_Input_Changed(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(errors.New("forbidden"), true, false) // 403 path

	got := stripANSIActivity(m.View())
	if strings.Contains(got, "temporarily unavailable") {
		t.Errorf("AC#9 input-changed: 403 must NOT render 'temporarily unavailable', got: %q", got)
	}
}

// ==================== AC#10: Enter no-op ====================

// TestActivityDashboardModel_EnterNoOp verifies AC#10: pressing Enter in the
// ActivityDashboardModel dispatches no command and produces no state change.
func TestActivityDashboardModel_EnterNoOp(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetRows(testRows(5))
	m.cursor = 2

	result, cmd := m.Update(keyMsg("enter"))
	if cmd != nil {
		t.Error("AC#10: cmd != nil after Enter, want nil (Enter is no-op in ActivityDashboard)")
	}
	if result.cursor != 2 {
		t.Errorf("AC#10: cursor moved on Enter: got %d, want 2", result.cursor)
	}
}

// ==================== AC#11: 'd' key no-op ====================

// TestActivityDashboardModel_DescribeKeyNoOp verifies AC#11: pressing 'd' in the
// ActivityDashboardModel dispatches no command.
func TestActivityDashboardModel_DescribeKeyNoOp(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetRows(testRows(3))

	_, cmd := m.Update(keyMsg("d"))
	if cmd != nil {
		t.Error("AC#11: cmd != nil after 'd', want nil (d is no-op in ActivityDashboard)")
	}
}

// ==================== AC#12: j/k clamp, no fetch ====================

// TestActivityDashboardModel_ScrollClamp verifies AC#12: j/k cursor moves within
// bounds and clamps at 0 and len(rows)-1 without dispatching any command.
func TestActivityDashboardModel_ScrollClamp(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetRows(testRows(3))

	// j moves cursor down.
	result, cmd := m.Update(keyMsg("j"))
	if result.cursor != 1 {
		t.Errorf("AC#12: after j, cursor = %d, want 1", result.cursor)
	}
	if cmd != nil {
		t.Error("AC#12: j dispatched a cmd, want nil")
	}

	// j again.
	result, _ = result.Update(keyMsg("j"))
	if result.cursor != 2 {
		t.Errorf("AC#12: after second j, cursor = %d, want 2", result.cursor)
	}

	// j clamps at last index (2).
	result, _ = result.Update(keyMsg("j"))
	if result.cursor != 2 {
		t.Errorf("AC#12: j past end — cursor = %d, want 2 (clamped)", result.cursor)
	}

	// k moves back up.
	result, _ = result.Update(keyMsg("k"))
	if result.cursor != 1 {
		t.Errorf("AC#12: after k, cursor = %d, want 1", result.cursor)
	}

	// k to 0.
	result, _ = result.Update(keyMsg("k"))
	// k clamps at 0.
	result, _ = result.Update(keyMsg("k"))
	if result.cursor != 0 {
		t.Errorf("AC#12: k past start — cursor = %d, want 0 (clamped)", result.cursor)
	}
}

// ==================== AC#13: Org-scope hint ====================

// TestActivityDashboardModel_OrgScopeHint verifies AC#13: when orgScope=true,
// the pane renders the "Select a project" hint and NO rows or event count.
func TestActivityDashboardModel_OrgScopeHint(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetOrgScope(true)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "Select a project to see recent activity") {
		t.Errorf("AC#13: want 'Select a project' hint when orgScope=true, got: %q", got)
	}
	if strings.Contains(got, "events") {
		t.Errorf("AC#13: 'events' must be absent in org-scope mode, got: %q", got)
	}
}

// TestActivityDashboardModel_OrgScopeHint_InputChanged verifies AC#13 input-changed:
// after SetOrgScope(false) the hint disappears.
func TestActivityDashboardModel_OrgScopeHint_InputChanged(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetOrgScope(true)
	m.SetOrgScope(false)

	got := stripANSIActivity(m.View())
	if strings.Contains(got, "Select a project") {
		t.Errorf("AC#13 input-changed: hint must be absent when orgScope=false, got: %q", got)
	}
}

// ==================== AC#15: 'r' refresh gating ====================

// TestActivityDashboardModel_RefreshKey_CRDAbsent_Gated verifies AC#15 anti-behavior:
// pressing 'r' when CRD-absent dispatches no command (gated by crdAbsent flag).
func TestActivityDashboardModel_RefreshKey_CRDAbsent_Gated(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetLoadErr(data.ErrActivityCRDAbsent, false, true)

	_, cmd := m.Update(keyMsg("r"))
	if cmd != nil {
		t.Error("AC#15: r dispatched a cmd in CRD-absent state, want nil (not retryable)")
	}
	// CRD-absent keybind strip must not show [r].
	got := stripANSIActivity(m.View())
	if strings.Contains(got, "[r]") {
		t.Errorf("AC#15: [r] must be absent from keybind strip in CRD-absent state, got: %q", got)
	}
}

// ==================== AC#19: Stale-refresh overlay ====================

// TestActivityDashboardModel_StaleRefresh_RowsRetained verifies AC#19 first half:
// when rows are cached and a refresh fails, the stale strip appears ABOVE the
// cached rows (both "refresh failed" and the resource name are in the view).
func TestActivityDashboardModel_StaleRefresh_RowsRetained(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	// Load rows successfully first.
	m.SetRows(testRows(3))
	// Now a refresh fails.
	m.SetLoadErr(errors.New("connection refused"), false, false)

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "refresh failed") {
		t.Errorf("AC#19: want 'refresh failed' stale strip when rows cached + refresh fails, got: %q", got)
	}
	if !strings.Contains(got, "checkout-api") {
		t.Errorf("AC#19: want cached rows still visible after refresh failure, got: %q", got)
	}
}

// TestActivityDashboardModel_StaleRefresh_EmptyCache_PlainError verifies AC#19 anti-behavior:
// when the cache is empty AND a fetch fails, no stale strip — plain error instead.
func TestActivityDashboardModel_StaleRefresh_EmptyCache_PlainError(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	// No rows loaded — cache is empty.
	m.SetLoadErr(errors.New("connection refused"), false, false)

	got := stripANSIActivity(m.View())
	if strings.Contains(got, "refresh failed") {
		t.Errorf("AC#19 anti-behavior: 'refresh failed' must NOT appear when cache is empty, got: %q", got)
	}
	if !strings.Contains(got, "temporarily unavailable") {
		t.Errorf("AC#19 anti-behavior: want 'temporarily unavailable' (plain error) when cache empty, got: %q", got)
	}
}

// TestActivityDashboardModel_StaleRefresh_ClearedOnSuccess verifies AC#19 input-changed:
// a successful refresh after a stale state clears the stale strip.
func TestActivityDashboardModel_StaleRefresh_ClearedOnSuccess(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(100, 30)
	m.SetRows(testRows(3))
	m.SetLoadErr(errors.New("network error"), false, false) // stale state
	m.SetRows(testRows(5))                                  // successful refresh

	got := stripANSIActivity(m.View())
	if strings.Contains(got, "refresh failed") {
		t.Errorf("AC#19 input-changed: 'refresh failed' must disappear after successful refresh, got: %q", got)
	}
	if !strings.Contains(got, "5 events") {
		t.Errorf("AC#19 input-changed: want updated '5 events' after successful refresh, got: %q", got)
	}
}

// ==================== AC#3: Resource column display name ====================

// TestActivityDashboardModel_ResourceColumn_WithResolver verifies AC#3:
// when registrations are set and a resolver match exists, RESOURCE column shows
// the resolved display name; without registrations it falls back to Kind literal.
func TestActivityDashboardModel_ResourceColumn_WithResolver(t *testing.T) {
	t.Parallel()
	rows := []data.ActivityRow{
		{
			Timestamp:    time.Now().Add(-time.Hour),
			ActorDisplay: "alice@example.com",
			Summary:      "updated config",
			ResourceRef: &data.ResourceRef{
				APIGroup: "apps.example.com",
				Kind:     "WebApp",
				Name:     "my-app",
			},
		},
	}
	// Name must match ref.Kind since renderRow calls ResolveDescription(registrations, apiGroup, kind).
	regs := []data.ResourceRegistration{
		{Group: "apps.example.com", Name: "WebApp", Description: "Web Applications"},
	}

	// Without resolver — fallback to Kind literal.
	m1 := newActivityDashboard(100, 30)
	m1.SetRows(rows)
	got1 := stripANSIActivity(m1.View())
	if !strings.Contains(got1, "WebApp/my-app") {
		t.Errorf("AC#3 without resolver: want 'WebApp/my-app' fallback, got: %q", got1)
	}

	// With resolver — display name shown.
	m2 := newActivityDashboard(100, 30)
	m2.SetRegistrations(regs)
	m2.SetRows(rows)
	got2 := stripANSIActivity(m2.View())
	if !strings.Contains(got2, "Web Applications/my-app") {
		t.Errorf("AC#3 with resolver: want 'Web Applications/my-app' display name, got: %q", got2)
	}
}

// ==================== AC#2: Timestamp formatting ====================

// TestActivityDashboardModel_TimestampFormatting verifies AC#2: rows < 48h old
// render HH:MM in compact bands; rows ≥ 48h render MM-DD.
func TestActivityDashboardModel_TimestampFormatting(t *testing.T) {
	t.Parallel()
	recent := time.Now().Add(-2 * time.Hour)   // < 48h
	old := time.Now().Add(-72 * time.Hour)     // ≥ 48h

	recentFmt := recent.Local().Format("15:04")
	oldFmt := old.Local().Format("01-02")

	// Use standard band (60 ≤ w < 80) where timestamps are in compact format.
	m := newActivityDashboard(70, 30)
	m.SetRows([]data.ActivityRow{
		{Timestamp: recent, ActorDisplay: "alice@example.com", Summary: "recent change",
			ResourceRef: &data.ResourceRef{Kind: "Pod", Name: "recent-pod"}},
		{Timestamp: old, ActorDisplay: "bob@example.com", Summary: "old change",
			ResourceRef: &data.ResourceRef{Kind: "Pod", Name: "old-pod"}},
	})

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, recentFmt) {
		t.Errorf("AC#2: recent row (<48h): want HH:MM format %q, got: %q", recentFmt, got)
	}
	if !strings.Contains(got, oldFmt) {
		t.Errorf("AC#2: old row (≥48h): want MM-DD format %q, got: %q", oldFmt, got)
	}
}

// ==================== AC#23: Width band boundaries ====================

// TestActivityDashboardModel_WidthBandBoundaries verifies AC#23: the six boundary
// widths across four bands produce the correct column sets.
//
// Band summary (contentW == width since there's no lipgloss border padding here):
//
//	[0,  40): unusable → "Terminal too narrow"
//	[40, 60): narrow   → stacked 2-line rows, no ACTOR col
//	[60, 80): standard → TIMESTAMP + RESOURCE + SUMMARY, no ACTOR
//	[80, ∞): wide     → TIMESTAMP + RESOURCE + ACTOR + SUMMARY
func TestActivityDashboardModel_WidthBandBoundaries(t *testing.T) {
	t.Parallel()
	rows := testRows(1) // one row is enough to observe column presence

	tests := []struct {
		name     string
		width    int
		unusable bool
		hasActor bool
	}{
		{"w=39 unusable", 39, true, false},
		{"w=40 narrow", 40, false, false},
		{"w=59 narrow", 59, false, false},
		{"w=60 standard", 60, false, false},
		{"w=79 standard", 79, false, false},
		{"w=80 wide", 80, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newActivityDashboard(tt.width, 30)
			m.SetRows(rows)

			got := stripANSIActivity(m.View())

			if tt.unusable {
				if !strings.Contains(got, "Terminal too narrow") {
					t.Errorf("AC#23 %s: want 'Terminal too narrow', got: %q", tt.name, got)
				}
				return
			}
			// All non-unusable bands should show rows (not "Terminal too narrow").
			if strings.Contains(got, "Terminal too narrow") {
				t.Errorf("AC#23 %s: unexpected 'Terminal too narrow' at width=%d, got: %q", tt.name, tt.width, got)
			}
			// Actor column check.
			if tt.hasActor && !strings.Contains(got, "alice@example.com") {
				t.Errorf("AC#23 %s: want ACTOR column (alice@example.com) at width=%d, got: %q", tt.name, tt.width, got)
			}
			if !tt.hasActor && strings.Contains(got, "alice@example.com") {
				t.Errorf("AC#23 %s: ACTOR column must be absent at width=%d, got: %q", tt.name, tt.width, got)
			}
		})
	}
}

// TestActivityDashboardModel_WidthBand_OffByOne_ActorAtExactly80 verifies AC#23
// anti-behavior: ACTOR is present at exactly w=80 and absent at w=79.
func TestActivityDashboardModel_WidthBand_OffByOne_ActorAtExactly80(t *testing.T) {
	t.Parallel()
	rows := testRows(1)

	m79 := newActivityDashboard(79, 30)
	m79.SetRows(rows)
	got79 := stripANSIActivity(m79.View())

	m80 := newActivityDashboard(80, 30)
	m80.SetRows(rows)
	got80 := stripANSIActivity(m80.View())

	if strings.Contains(got79, "alice@example.com") {
		t.Errorf("AC#23 off-by-one: ACTOR must be absent at w=79, got: %q", got79)
	}
	if !strings.Contains(got80, "alice@example.com") {
		t.Errorf("AC#23 off-by-one: ACTOR must be present at w=80, got: %q", got80)
	}
}

// keyMsg builds a tea.KeyPressMsg for the given key string (for use in Update calls).
func keyMsg(s string) tea.Msg {
	if s == "enter" {
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	}
	runes := []rune(s)
	if len(runes) == 1 {
		return tea.KeyPressMsg{Code: runes[0], Text: s}
	}
	return tea.KeyPressMsg{Text: s}
}

// ==================== FB-088: ActivityDashboard origin affordance ====================

// AC1 (ActivityDashboard) [Observable] — QuotaDashboard chain origin → View contains "[4] back to quota dashboard".
func TestFB088_ActivityDashboard_AC1_QuotaDashOrigin(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(200, 30)
	m.SetRows(testRows(3))
	m.SetOriginLabel("quota dashboard")

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "[4] back to quota dashboard") {
		t.Errorf("AC1: View() missing '[4] back to quota dashboard', got:\n%s", got)
	}
}

// AC2 (ActivityDashboard) [Observable] — resource list origin → View contains "[4] back to resource list".
func TestFB088_ActivityDashboard_AC2_ResourceListOrigin(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(200, 30)
	m.SetRows(testRows(3))
	m.SetOriginLabel("resource list")

	got := stripANSIActivity(m.View())
	if !strings.Contains(got, "[4] back to resource list") {
		t.Errorf("AC2: View() missing '[4] back to resource list', got:\n%s", got)
	}
}

// AC4 (ActivityDashboard) [Input-changed] — different origin labels produce different View content.
func TestFB088_ActivityDashboard_AC4_InputChanged(t *testing.T) {
	t.Parallel()
	m1 := newActivityDashboard(200, 30)
	m1.SetRows(testRows(3))
	m1.SetOriginLabel("resource list")
	got1 := stripANSIActivity(m1.View())

	m2 := newActivityDashboard(200, 30)
	m2.SetRows(testRows(3))
	m2.SetOriginLabel("detail view")
	got2 := stripANSIActivity(m2.View())

	if !strings.Contains(got1, "resource list") {
		t.Errorf("AC4: got1 missing 'resource list': %s", got1)
	}
	if !strings.Contains(got2, "detail view") {
		t.Errorf("AC4: got2 missing 'detail view': %s", got2)
	}
	if got1 == got2 {
		t.Error("AC4: View() output identical for different origin labels; want distinct content")
	}
}

// AC5 (ActivityDashboard) [Anti-behavior] — empty originLabel → View does NOT contain "back to".
func TestFB088_ActivityDashboard_AC5_FreshStartupNoHint(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(200, 30)
	m.SetRows(testRows(3))
	// No SetOriginLabel call.

	got := stripANSIActivity(m.View())
	if strings.Contains(got, "back to") {
		t.Errorf("AC5: View() contains 'back to' with empty originLabel; want absent:\n%s", got)
	}
}

// AC6 (ActivityDashboard) [Anti-behavior] — after SetOriginLabel("") the hint is suppressed.
func TestFB088_ActivityDashboard_AC6_ClearOriginSuppressesHint(t *testing.T) {
	t.Parallel()
	m := newActivityDashboard(200, 30)
	m.SetRows(testRows(3))
	m.SetOriginLabel("resource list")
	m.SetOriginLabel("") // simulate Esc clear

	got := stripANSIActivity(m.View())
	if strings.Contains(got, "back to") {
		t.Errorf("AC6: View() contains 'back to' after SetOriginLabel(\"\"); want absent:\n%s", got)
	}
}

// ==================== End FB-088 (ActivityDashboard) ====================
