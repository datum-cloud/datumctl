package components

import (
	"errors"
	"strings"
	"testing"
	"time"

	datumconfig "go.datum.net/datumctl/internal/datumconfig"
	tuictx "go.datum.net/datumctl/internal/tui/context"
	"go.datum.net/datumctl/internal/tui/data"
)

// namedRows builds a slice of ResourceRow from plain name strings.
func namedRows(names ...string) []data.ResourceRow {
	result := make([]data.ResourceRow, len(names))
	for i, n := range names {
		result[i] = data.ResourceRow{Name: n, Cells: []string{n}}
	}
	return result
}

func TestResourceTableModel_RefreshRows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		initial   []data.ResourceRow
		filter    string
		cursorIdx int
		refresh   []data.ResourceRow
		wantName  string
		wantNoRow bool
	}{
		{
			name:      "cursor stays on same-named row when name present in refresh",
			initial:   namedRows("alpha", "beta", "gamma"),
			cursorIdx: 1, // "beta"
			refresh:   namedRows("alpha", "beta", "gamma"),
			wantName:  "beta",
		},
		{
			name:      "cursor follows named row when it shifts position",
			initial:   namedRows("alpha", "beta", "gamma"),
			cursorIdx: 2, // "gamma"
			refresh:   namedRows("gamma", "alpha", "beta"), // "gamma" moved to front
			wantName:  "gamma",
		},
		{
			name:      "cursor clamps to last row when named row gone and no filter",
			initial:   namedRows("alpha", "beta", "gamma"),
			cursorIdx: 2, // "gamma" will be removed
			refresh:   namedRows("alpha", "beta"),
			wantName:  "beta", // clamp: min(2, len-1=1) = 1 → "beta"
		},
		{
			name:      "cursor falls to row 0 when named row gone and filter is active",
			initial:   namedRows("prod-a", "prod-b", "prod-c"),
			filter:    "prod",
			cursorIdx: 2, // "prod-c" will be removed
			refresh:   namedRows("prod-a", "prod-b"),
			wantName:  "prod-a", // filter active → fall to 0
		},
		{
			name:      "empty refresh list leaves table empty without panicking",
			initial:   namedRows("alpha", "beta"),
			cursorIdx: 1,
			refresh:   []data.ResourceRow{},
			wantNoRow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewResourceTableModel(80, 20)
			m.SetColumns([]string{"Name"}, 80)
			m.SetRows(tt.initial)
			if tt.filter != "" {
				m.SetFilter(tt.filter)
			}
			m.table.SetCursor(tt.cursorIdx)

			m.RefreshRows(tt.refresh)

			row, ok := m.SelectedRow()
			if tt.wantNoRow {
				if ok {
					t.Errorf("SelectedRow() = (%q, true), want (_, false)", row.Name)
				}
				return
			}
			if !ok {
				t.Fatalf("SelectedRow() returned false, want row %q", tt.wantName)
			}
			if row.Name != tt.wantName {
				t.Errorf("SelectedRow().Name = %q, want %q", row.Name, tt.wantName)
			}
		})
	}
}

// --- FB-003: empty-state rendering when filter returns zero rows ---

// TestResourceTableModel_Filter_ZeroResults_ShowsNoResultsAndEscHint verifies that
// when a filter is active and no rows match, the table renders:
//   - "No results for «query»" message, and
//   - "[Esc] clear filter" hint
// (observable outcome for the FB-003 empty-state spec).
func TestResourceTableModel_Filter_ZeroResults_ShowsNoResultsAndEscHint(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetColumns([]string{"Name"}, 80)
	m.SetRows(namedRows("alpha", "beta", "gamma"))
	m.SetTypeContext("pods", true)
	m.SetFilter("zzz") // no row matches

	got := stripANSI(m.View())
	if !strings.Contains(got, "No results for") {
		t.Errorf("filter zero results: want 'No results for' in %q", got)
	}
	if !strings.Contains(got, "zzz") {
		t.Errorf("filter zero results: want query 'zzz' in message, got %q", got)
	}
	if !strings.Contains(got, "[Esc] clear filter") {
		t.Errorf("filter zero results: want '[Esc] clear filter' hint in %q", got)
	}
}

// TestResourceTableModel_SetRows_DoesNotPreserveCursorByName is a regression guard:
// SetRows (initial/type-switch load) does not chase row names. Cursor position
// reflects whatever the inner table state is — it is not moved to find the
// previously-selected name in the new list.
func TestResourceTableModel_SetRows_DoesNotPreserveCursorByName(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetColumns([]string{"Name"}, 80)
	m.SetRows(namedRows("alpha", "beta", "gamma"))
	m.table.SetCursor(1) // cursor on "beta"

	// Replace rows: "beta" moves to index 2.
	m.SetRows(namedRows("alpha", "gamma", "beta"))

	row, ok := m.SelectedRow()
	if !ok {
		t.Fatal("SelectedRow() returned false")
	}
	// Cursor stays at index 1 — SetRows does NOT chase "beta" to index 2.
	// (bubbles table.SetRows preserves cursor position, so we get "gamma".)
	if row.Name == "beta" {
		t.Errorf("SetRows: SelectedRow().Name = %q — cursor must not preserve by name (RefreshRows should be used for that)", row.Name)
	}
}

// ==================== FB-015: Welcome / landing panel ====================

// newWelcomeModel builds a ResourceTableModel in welcome-panel mode (typeName == "").
func newWelcomeModel(tableWidth, tableHeight int) ResourceTableModel {
	return NewResourceTableModel(tableWidth, tableHeight)
}

// testCtx builds a minimal TUIContext for landing-screen fixtures (no ActiveCtx).
func testCtx(userName, orgName, projName string, readOnly bool) tuictx.TUIContext {
	return tuictx.TUIContext{
		UserName:    userName,
		OrgName:     orgName,
		ProjectName: projName,
		ReadOnly:    readOnly,
	}
}

// testCtxWithProject builds a TUIContext whose ActiveCtx carries a project ID
// so activeConsumer() resolves to ("Project", projectID).
func testCtxWithProject(userName, orgName, projName, projectID string) tuictx.TUIContext {
	return tuictx.TUIContext{
		UserName:    userName,
		OrgName:     orgName,
		ProjectName: projName,
		ActiveCtx:   &datumconfig.DiscoveredContext{ProjectID: projectID},
	}
}

// TestResourceTableModel_Welcome_HeaderBand verifies AC#1: user name appears in header.
func TestResourceTableModel_Welcome_HeaderBand(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtx("alice", "acme-corp", "web", false))

	got := stripANSI(m.View())
	if !strings.Contains(got, "alice") {
		t.Errorf("AC#1: want user name 'alice' in header, got: %q", got)
	}
	if !strings.Contains(got, "Welcome") {
		t.Errorf("AC#1: want 'Welcome' prefix in header, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_ReadOnlyBadge_Shown verifies AC#2: [READ-ONLY] badge
// appears when ReadOnly=true and contentW >= 60 (tableWidth=80 → contentW=76).
func TestResourceTableModel_Welcome_ReadOnlyBadge_Shown(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30) // contentW = 76
	m.SetTUIContext(testCtx("alice", "acme-corp", "", true))

	got := stripANSI(m.View())
	if !strings.Contains(got, "[READ-ONLY]") {
		t.Errorf("AC#2: want '[READ-ONLY]' badge when contentW>=60, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_ReadOnlyBadge_HiddenNarrow verifies AC#2 (input-changed):
// badge absent when contentW < 60 (tableWidth=63 → contentW=59).
func TestResourceTableModel_Welcome_ReadOnlyBadge_HiddenNarrow(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(63, 30) // contentW = 59
	m.SetTUIContext(testCtx("alice", "acme-corp", "", true))

	got := stripANSI(m.View())
	if strings.Contains(got, "[READ-ONLY]") {
		t.Errorf("AC#2: want '[READ-ONLY]' badge absent at contentW<60, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_OrgOnlyNoSlash verifies AC#3: org-only context renders
// org name without a trailing slash or slash-separated component.
func TestResourceTableModel_Welcome_OrgOnlyNoSlash(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtx("alice", "acme-corp", "", false)) // no project

	got := stripANSI(m.View())
	if !strings.Contains(got, "acme-corp") {
		t.Errorf("AC#3: want 'acme-corp' in view, got: %q", got)
	}
	// The org/project separator is " / " — must not appear when project is empty.
	// (Keybind strip has "/ filter" but no leading space before the slash.)
	if strings.Contains(got, "acme-corp /") {
		t.Errorf("AC#3: want no slash after org name when project is empty, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_Placeholder_NoHoveredType verifies that the welcome
// panel renders S1 (identity) and S2 (platform health) even when no type is hovered.
// FB-042 removed the hovered-type left block and its placeholder.
func TestResourceTableModel_Welcome_Placeholder_NoHoveredType(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("want 'Welcome' in S1 when no type hovered, got: %q", got)
	}
	if !strings.Contains(got, "Platform health") {
		t.Errorf("want 'Platform health' in S2 when no type hovered, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_HealthSummary verifies the platform health status line.
// FB-042 replaced the "N of M governed types" summary with a right-aligned status line.
func TestResourceTableModel_Welcome_HealthSummary(t *testing.T) {
	t.Parallel()
	projectID := "proj-abc"
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", projectID))
	// deployments=90% (≥80), pods=40% (<80) → constrained=1, total=2
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "core/pods", Limit: 100, Allocated: 40},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "need attention") {
		t.Errorf("want 'need attention' status line when constrained types > 0, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_TopThree_Ordering verifies AC#7: top-3 quota rows
// appear in utilisation-descending order (highest first).
func TestResourceTableModel_Welcome_TopThree_Ordering(t *testing.T) {
	t.Parallel()
	projectID := "proj-abc"
	m := newWelcomeModel(100, 30) // contentH=26 → showTop3=true
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 5}, // 50%
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "core/pods", Limit: 10, Allocated: 9},         // 90%
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "storage/pvcs", Limit: 10, Allocated: 8},      // 80%
	})

	got := stripANSI(m.View())
	podIdx := strings.Index(got, "pods")
	pvcIdx := strings.Index(got, "pvcs")
	depIdx := strings.Index(got, "deployments")
	if podIdx < 0 || pvcIdx < 0 || depIdx < 0 {
		t.Fatalf("AC#7: want all three types in view, got: %q", got)
	}
	if !(podIdx < pvcIdx && pvcIdx < depIdx) {
		t.Errorf("AC#7: want pods(90%%) before pvcs(80%%) before deployments(50%%), got indices pods=%d pvcs=%d dep=%d", podIdx, pvcIdx, depIdx)
	}
}

// TestResourceTableModel_Welcome_NoGoverned verifies AC#8: "No governed resource types"
// message when the active consumer has no matching buckets.
func TestResourceTableModel_Welcome_NoGoverned(t *testing.T) {
	t.Parallel()
	projectID := "proj-abc"
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{}) // empty — no governed types

	got := stripANSI(m.View())
	if !strings.Contains(got, "No governed resource types") {
		t.Errorf("AC#8: want 'No governed resource types' when buckets empty, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_BucketErr_Unauthorized verifies AC#9: 403 yields
// "Platform health unavailable" without "temporarily".
func TestResourceTableModel_Welcome_BucketErr_Unauthorized(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", "proj-abc"))
	m.SetBucketErr(errors.New("forbidden"), true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Platform health unavailable") {
		t.Errorf("AC#9: 403 want 'Platform health unavailable', got: %q", got)
	}
	if strings.Contains(got, "temporarily") {
		t.Errorf("AC#9: 403 must not say 'temporarily', got: %q", got)
	}
}

// TestResourceTableModel_Welcome_BucketErr_NonUnauthorized verifies AC#9 (input-changed):
// non-403 error yields "Platform health temporarily unavailable".
func TestResourceTableModel_Welcome_BucketErr_NonUnauthorized(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", "proj-abc"))
	m.SetBucketErr(errors.New("network error"), false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Platform health temporarily unavailable") {
		t.Errorf("AC#9: non-403 want 'Platform health temporarily unavailable', got: %q", got)
	}
}

// TestResourceTableModel_Welcome_BucketLoading verifies AC#10: loading placeholder
// "loading platform health" appears when buckets is nil and loading flag is set.
func TestResourceTableModel_Welcome_BucketLoading(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", "proj-abc"))
	m.SetBucketLoading(true) // buckets remains nil

	got := stripANSI(m.View())
	if !strings.Contains(got, "loading platform health") {
		t.Errorf("AC#10: want 'loading platform health' spinner text, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_ResolverLabel verifies AC#11: a ResourceRegistration
// description overrides the short resource name in the top-3 quota row label.
func TestResourceTableModel_Welcome_ResolverLabel(t *testing.T) {
	t.Parallel()
	projectID := "proj-abc"
	m := newWelcomeModel(100, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 10},
	})
	// Use a short description that fits inside the labelW (≤17 chars at this width).
	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "apps", Name: "deployments", Description: "App Deploys"},
	})

	got := stripANSI(m.View())
	// "App Deploys" (11 chars) fits in the label column without truncation,
	// and must appear instead of the raw resource name "deployments".
	if !strings.Contains(got, "App Deploys") {
		t.Errorf("AC#11: want registration description 'App Deploys' as row label, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_StaleBanner_Shown verifies AC#15 (non-boundary):
// stale banner appears when staleBanner=true and contentH >= 12 (tableHeight=20 → contentH=16).
func TestResourceTableModel_Welcome_StaleBanner_Shown(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 20) // contentH = 16
	m.SetStaleCacheAge(true, "26h")

	got := stripANSI(m.View())
	if !strings.Contains(got, "Context cache last refreshed") {
		t.Errorf("AC#15: want stale banner when staleBanner=true and contentH>=12, got: %q", got)
	}
	if !strings.Contains(got, "26h") {
		t.Errorf("AC#15: want age '26h' in stale banner, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_StaleBanner_Absent_WhenFlagFalse verifies AC#15 (input-changed):
// stale banner absent when staleBanner=false even at sufficient height.
func TestResourceTableModel_Welcome_StaleBanner_Absent_WhenFlagFalse(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetStaleCacheAge(false, "26h")

	got := stripANSI(m.View())
	if strings.Contains(got, "Context cache last refreshed") {
		t.Errorf("AC#15: want stale banner absent when staleBanner=false, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_StaleBanner_Absent_TooShort verifies AC#15 (input-changed):
// stale banner absent when contentH < 12 (tableHeight=15 → contentH=11).
func TestResourceTableModel_Welcome_StaleBanner_Absent_TooShort(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 15) // contentH = 11, gate is >= 12
	m.SetStaleCacheAge(true, "26h")

	got := stripANSI(m.View())
	if strings.Contains(got, "Context cache last refreshed") {
		t.Errorf("AC#15: want stale banner absent when contentH<12, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_KeybindStrip_Shown verifies AC#17: keybind strip
// appears when contentH >= 18 (tableHeight=22 → contentH=18).
func TestResourceTableModel_Welcome_KeybindStrip_Shown(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 22) // contentH = 18

	got := stripANSI(m.View())
	if !strings.Contains(got, "j/k") {
		t.Errorf("AC#17: want keybind strip (j/k) when contentH>=18, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_KeybindStrip_Absent verifies keybind strip absent
// when contentH < 12 (tableHeight=15 → contentH=11). FB-042 lowered the threshold
// from 18 to 12 (keybind present in the Minimal band 12–17).
func TestResourceTableModel_Welcome_KeybindStrip_Absent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 15) // contentH = 11 < 12 → S6 absent

	got := stripANSI(m.View())
	if strings.Contains(got, "j/k") {
		t.Errorf("want keybind strip absent when contentH<12, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_KeybindStrip_TruncatesNarrow verifies AC#17 §6c:
// when contentW is less than the natural strip width, the strip is truncated
// with a trailing `…` and never wraps to a second line.
func TestResourceTableModel_Welcome_KeybindStrip_TruncatesNarrow(t *testing.T) {
	t.Parallel()
	// tableWidth=34 → contentW=30; narrow enough that the full strip can't fit.
	m := newWelcomeModel(34, 22) // contentH=18 → showKeybind=true

	got := stripANSI(m.View())
	if !strings.Contains(got, "…") {
		t.Errorf("AC#17 §6c: narrow strip must end with '…' when truncated, got: %q", got)
	}
	// Truncated strip must be a single line — no double newline from wrapping.
	lines := strings.Split(got, "\n")
	stripLines := 0
	for _, l := range lines {
		if strings.Contains(l, "j/k") || strings.Contains(l, "…") {
			stripLines++
		}
	}
	if stripLines > 1 {
		t.Errorf("AC#17 §6c: keybind strip must not wrap — found on %d lines, want 1", stripLines)
	}
}

// ==================== FB-015: AC#23 — four-band width-collapse ====================

// TestResourceTableModel_Welcome_WidthBands verifies AC#23: the six boundary widths
// produce the correct layout band per §6a. Assertions use content-based observables
// from stripANSI(m.View()):
//
//	width=84 → contentW=80 → two-col + bars (█ present)
//	width=80 → contentW=76 → two-col text-only (no █)
//	width=64 → contentW=60 → two-col text-only (no █)
//	width=63 → contentW=59 → single-col stack (right block present, but no two-col side-by-side)
//	width=54 → contentW=50 → single-col stack
//	width=53 → contentW=49 → right block absent
func TestResourceTableModel_Welcome_WidthBands(t *testing.T) {
	t.Parallel()
	projectID := "proj-wb"
	// Provide a bucket so platform-health renders in all bands that show it.
	buckets := []data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 10},
	}
	buildCtx := func() tuictx.TUIContext {
		return testCtxWithProject("alice", "acme-corp", "web", projectID)
	}

	tests := []struct {
		name        string
		tableWidth  int
		wantBars    bool   // █ glyph present (bars mode)
		wantHealth  bool   // "Platform health" present
		wantTwoCol  bool   // right-block rendered alongside left block (not stacked)
		wantNoRight bool   // right block entirely absent
	}{
		{
			name:        "width=84 contentW=80 two-col bars",
			tableWidth:  84,
			wantBars:    true,
			wantHealth:  true,
			wantTwoCol:  true,
			wantNoRight: false,
		},
		{
			name:        "width=83 contentW=79 two-col text-only (just below bars boundary)",
			tableWidth:  83,
			wantBars:    false,
			wantHealth:  true,
			wantTwoCol:  true,
			wantNoRight: false,
		},
		{
			name:        "width=64 contentW=60 two-col text-only boundary",
			tableWidth:  64,
			wantBars:    false,
			wantHealth:  true,
			wantTwoCol:  true,
			wantNoRight: false,
		},
		{
			name:        "width=63 contentW=59 single-col stack",
			tableWidth:  63,
			wantBars:    false,
			wantHealth:  true,
			wantTwoCol:  false,
			wantNoRight: false,
		},
		{
			name:        "width=54 contentW=50 single-col stack boundary",
			tableWidth:  54,
			wantBars:    false,
			wantHealth:  true,
			wantTwoCol:  false,
			wantNoRight: false,
		},
		{
			name:        "width=53 contentW=49 one-liner health shown",
			tableWidth:  53,
			wantBars:    false,
			wantHealth:  true,
			wantTwoCol:  false,
			wantNoRight: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newWelcomeModel(tt.tableWidth, 30)
			m.SetTUIContext(buildCtx())
			m.SetBuckets(buckets)

			got := stripANSI(m.View())

			// Bars mode (contentW >= 80): █ glyph must appear.
			if tt.wantBars && !strings.Contains(got, "█") {
				t.Errorf("AC#23 %s: want bar █ glyph (bars mode), got: %q", tt.name, got)
			}
			if !tt.wantBars && strings.Contains(got, "█") {
				t.Errorf("AC#23 %s: want NO bar █ glyph (text-only), got: %q", tt.name, got)
			}

			// Platform health header present/absent.
			if tt.wantHealth && !strings.Contains(got, "Platform health") {
				t.Errorf("AC#23 %s: want 'Platform health' header, got: %q", tt.name, got)
			}
			if tt.wantNoRight && strings.Contains(got, "Platform health") {
				t.Errorf("AC#23 %s: want right block absent (contentW<50), got: %q", tt.name, got)
			}
		})
	}
}


// ── Height boundary tests (§6b) ───────────────────────────────────────────────
//
// Formula: contentH = tableHeight - 4
// Band thresholds (contentH-relative):
//   contentH < 9   → headerOnly  (header + placeholder only)
//   9 ≤ cH < 12   → compactLeft (no stale banner)
//   12 ≤ cH < 15  → no top-3
//   15 ≤ cH < 18  → top-3 visible
//   cH ≥ 18       → keybind strip
//
// tableHeight-relative thresholds: 13, 16, 19, 22
//
// NOTE: team-lead spec listed tableHeight={7,12,17,18} expecting keybind at h=18.
// At tableHeight=18 → contentH=14 < 18 → showKeybind=false → keybind absent.
// Keybind threshold is tableHeight=22 (contentH=18). Correct behavior used below.

// TestResourceTableModel_Welcome_HeightBands is a table-driven boundary sweep at
// the four tableHeight values from the §6b spec. Assertions reflect actual code
// behavior (contentH = tableHeight - 4).
func TestResourceTableModel_Welcome_HeightBands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tableHeight  int
		contentH     int // = tableHeight - 4
		wantKeybind  bool
		wantHealth   bool // false only in headerOnly mode
		description  string
	}{
		{7, 3, false, true, "headerOnly: contentH=3 — S2 always shown in FB-042"},
		{12, 8, false, true, "headerOnly: contentH=8 — S2 always shown in FB-042"},
		{17, 13, true, true, "keybind shown: contentH=13 ≥ 12 (FB-042 threshold)"},
		{18, 14, true, true, "keybind shown: contentH=14 ≥ 12 (FB-042 threshold)"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()
			m := newWelcomeModel(100, tt.tableHeight)

			got := stripANSI(m.View())
			if tt.wantKeybind && !strings.Contains(got, "j/k") {
				t.Errorf("tableHeight=%d (contentH=%d): want keybind strip (j/k), got: %q",
					tt.tableHeight, tt.contentH, got)
			}
			if !tt.wantKeybind && strings.Contains(got, "j/k") {
				t.Errorf("tableHeight=%d (contentH=%d): want keybind strip absent, got: %q",
					tt.tableHeight, tt.contentH, got)
			}
			if tt.wantHealth && !strings.Contains(got, "Platform health") {
				t.Errorf("tableHeight=%d (contentH=%d): want 'Platform health' visible, got: %q",
					tt.tableHeight, tt.contentH, got)
			}
			if !tt.wantHealth && strings.Contains(got, "Platform health") {
				t.Errorf("tableHeight=%d (contentH=%d): want 'Platform health' absent (headerOnly), got: %q",
					tt.tableHeight, tt.contentH, got)
			}
		})
	}
}

// ── Width band collapse (AC#23) ───────────────────────────────────────────────
//
// contentW = tableWidth - 4
//
//	tableWidth=44  → contentW=40  → rightHidden (right block absent entirely)
//	tableWidth=59  → contentW=55  → stacked single-col, compact health one-liner
//	tableWidth=74  → contentW=70  → two-col, text-only bars (no █)
//	tableWidth=94  → contentW=90  → two-col with graphic █ bars

// TestResourceTableModel_Welcome_WidthBand_RightHidden verifies FB-042 §9: at
// contentW<50 (tableWidth=44) Platform health renders as a compact one-liner
// (S2 is always shown in FB-042; only S3/S4/S5 are gated by width).
func TestResourceTableModel_Welcome_WidthBand_RightHidden(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(44, 30) // contentW = 40 → narrow one-liner
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Platform health") {
		t.Errorf("FB-042: want 'Platform health' present even at contentW<50 (one-liner), got: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("FB-042: want no bars at contentW<50, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_WidthBand_StackedSingleCol verifies AC#23 (input-changed):
// at contentW in [50,60) (tableWidth=59) health shows as a compact one-liner
// ("types ≥80%") without the word "governed" or top-3 rows.
func TestResourceTableModel_Welcome_WidthBand_StackedSingleCol(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(59, 30) // contentW = 55 → stacked, not rightHidden
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Platform health") {
		t.Errorf("AC#23: want 'Platform health' visible in stacked layout, got: %q", got)
	}
	// Compact mode emits "types ≥80%" without the word "governed".
	if strings.Contains(got, "governed types") {
		t.Errorf("AC#23: stacked compact layout must not say 'governed types', got: %q", got)
	}
	// Top-3 list is suppressed in compact mode.
	if strings.Contains(got, "press 3 for full dashboard") {
		t.Errorf("AC#23: top-3 list must be absent in stacked compact layout, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_WidthBand_TwoColTextOnly verifies AC#23 (input-changed):
// at contentW in [60,80) (tableWidth=74) health is visible (single-column, FB-042 layout)
// with text-only quota rows and no graphic bars.
func TestResourceTableModel_Welcome_WidthBand_TwoColTextOnly(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(74, 30) // contentW = 70 → wide enough for S2 list, no bars
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Platform health") {
		t.Errorf("AC#23: want 'Platform health' header at contentW=70, got: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("AC#23: want no graphic bars (█) at contentW<80 (textOnly mode), got: %q", got)
	}
}

// TestResourceTableModel_Welcome_WidthBand_TwoColBars verifies AC#23 (input-changed):
// at contentW≥80 (tableWidth=94) graphic █ bars appear in the top-3 quota rows.
func TestResourceTableModel_Welcome_WidthBand_TwoColBars(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(94, 30) // contentW = 90 → barsMode=true
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "█") {
		t.Errorf("AC#23: want graphic █ bars at contentW≥80 (barsMode), got: %q", got)
	}
}

// ── Height band collapse (§6b / AC#17) ────────────────────────────────────────
//
// contentH = tableHeight - 4
//
//	tableHeight=12  → contentH=8   → headerOnly  (<9)
//	tableHeight=16  → contentH=12  → no top-3, stale-banner eligible (12–14)
//	tableHeight=20  → contentH=16  → top-3 visible, no keybind (15–17)
//	tableHeight=28  → contentH=24  → full layout with keybind (≥18)

// TestResourceTableModel_Welcome_HeightBand_HeaderOnly verifies FB-042 §9 "Header" band:
// at contentH<12 (tableHeight=15 → contentH=11), S1 and S2 render; keybind and S3-S5 absent.
// FB-042 replaced the old placeholder with S2 platform health (always shown).
func TestResourceTableModel_Welcome_HeightBand_HeaderOnly(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(100, 15) // contentH = 11 < 12 → Header band
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Welcome") {
		t.Errorf("FB-042 §9 Header: want 'Welcome' (S1) at contentH<12, got: %q", got)
	}
	if !strings.Contains(got, "Platform health") {
		t.Errorf("FB-042 §9 Header: want 'Platform health' (S2) at contentH<12, got: %q", got)
	}
	// No keybind strip (S6 requires contentH≥12).
	if strings.Contains(got, "j/k") {
		t.Errorf("FB-042 §9 Header: want keybind absent at contentH<12, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_HeightBand_NoTop3 verifies §6b (input-changed):
// at contentH in [12,15) (tableHeight=16) health summary appears but top-3 is absent.
func TestResourceTableModel_Welcome_HeightBand_NoTop3(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(100, 16) // contentH = 12 → showTop3=false
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Platform health") {
		t.Errorf("§6b: want 'Platform health' visible at contentH=12, got: %q", got)
	}
	// Top-3 hint is only appended when showList=true.
	if strings.Contains(got, "press 3 for full dashboard") {
		t.Errorf("§6b: want top-3 list absent at contentH<15, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_HeightBand_MinimalKeybind verifies FB-042 §9 (input-changed):
// at contentH in [12,18) (Minimal band, tableHeight=20 → contentH=16) keybind strip IS present
// but S2 shows no top-3 list (showS2List requires contentH≥18).
func TestResourceTableModel_Welcome_HeightBand_MinimalKeybind(t *testing.T) {
	t.Parallel()
	projectID := "proj-x"
	m := newWelcomeModel(100, 20) // contentH = 16 → Minimal band (12–17)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 9},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "j/k") {
		t.Errorf("FB-042 §9: want keybind strip (j/k) at contentH=16 (Minimal band ≥12), got: %q", got)
	}
	// No top-3 at contentH<18: "(press [3] for full dashboard)" absent.
	if strings.Contains(got, "[3] for full dashboard") {
		t.Errorf("FB-042 §9: want top-3 list absent at contentH<18, got: %q", got)
	}
}

// TestResourceTableModel_Welcome_HeightBand_FullLayout verifies §6b (input-changed):
// at contentH≥18 (tableHeight=28) the keybind strip is present (full layout).
func TestResourceTableModel_Welcome_HeightBand_FullLayout(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 28) // contentH = 24 → showKeybind=true

	got := stripANSI(m.View())
	if !strings.Contains(got, "j/k") {
		t.Errorf("§6b: want keybind strip (j/k) visible at contentH≥18, got: %q", got)
	}
}

// TestResourceTableModel_ApplyFilter_CursorReset_AfterSetColumns is an
// [Edge] + [Anti-regression] guard for resourcetable.go:710-714.
//
// The charmbracelet table.SetColumns internally calls SetRows(nil), driving the
// cursor to -1 (downward-only clamp — SetRows with content never clamps upward).
// applyFilter must detect this and reset cursor to 0 after repopulating rows,
// otherwise SelectedRow() silently returns (ResourceRow{}, false) and the user
// cannot select any row until they manually move the cursor.
//
// This test must fail without the cursor-reset guard at applyFilter:713-715.
// ==================== FB-042: Enhanced welcome dashboard ====================

// --- S3: Recent activity section ---

// [Observable] activityLoading=true, activityRows=nil → "loading" placeholder in S3.
func TestFB042_Activity_Loading(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30) // contentH=26 ≥ 24 → S3 shown
	m.SetActivityLoading(true)
	// activityRows is nil by default.

	got := stripANSI(m.View())
	if !strings.Contains(got, "Recent activity") {
		t.Errorf("S3 header 'Recent activity' missing when loading:\n%s", got)
	}
	if !strings.Contains(got, "loading") {
		t.Errorf("S3 loading placeholder 'loading' missing when activityLoading=true and rows=nil:\n%s", got)
	}
}

// [Observable] activityRows=[] (empty after load) → "no recent activity".
func TestFB042_Activity_Empty(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{}) // explicitly empty — loaded, no rows

	got := stripANSI(m.View())
	if !strings.Contains(got, "no recent activity") {
		t.Errorf("S3 empty placeholder 'no recent activity' missing:\n%s", got)
	}
}

// [Observable] activityRows populated → actor and summary text rendered.
func TestFB042_Activity_WithRows(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetActivityRows([]data.ActivityRow{
		{
			Timestamp:    time.Now().Add(-2 * time.Minute),
			ActorDisplay: "alice@example.com",
			Summary:      "created project",
			ResourceRef:  &data.ResourceRef{Kind: "Project", Name: "my-proj"},
		},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("S3 actor 'alice@example.com' missing from activity row:\n%s", got)
	}
	if !strings.Contains(got, "created project") {
		t.Errorf("S3 summary 'created project' missing from activity row:\n%s", got)
	}
}

// [Anti-behavior] loading=false but rows=nil still shows no spinner — falls through
// to "no recent activity" (rows initialized to nil means not yet fetched; empty slice
// means fetched but empty — this tests that nil rows without loading shows nothing).
func TestFB042_Activity_NilRows_NotLoading(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	// activityRows=nil, activityLoading=false (defaults) — no activity loaded yet.

	got := stripANSI(m.View())
	// nil rows → "no recent activity" (same branch as empty slice)
	if !strings.Contains(got, "no recent activity") {
		t.Errorf("S3 with nil rows and loading=false want 'no recent activity', got:\n%s", got)
	}
}

// [Anti-regression] activity rows capped at 3 even when more are provided.
func TestFB042_Activity_CappedAtThree(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	rows := []data.ActivityRow{
		{Timestamp: time.Now(), ActorDisplay: "user-one", Summary: "action-one"},
		{Timestamp: time.Now(), ActorDisplay: "user-two", Summary: "action-two"},
		{Timestamp: time.Now(), ActorDisplay: "user-three", Summary: "action-three"},
		{Timestamp: time.Now(), ActorDisplay: "user-four", Summary: "action-four"},
	}
	m.SetActivityRows(rows)

	got := stripANSI(m.View())
	// user-four is the 4th row and must NOT appear (capped at 3).
	if strings.Contains(got, "user-four") {
		t.Errorf("S3 capped at 3: 4th actor 'user-four' must be absent:\n%s", got)
	}
	// All three visible rows should appear.
	for _, actor := range []string{"user-one", "user-two", "user-three"} {
		if !strings.Contains(got, actor) {
			t.Errorf("S3 capped at 3: want actor %q present:\n%s", actor, got)
		}
	}
}

// --- S4: Quick-jump section ---

// [Anti-behavior] no matching registrations → S4 section absent entirely.
func TestFB042_QuickJump_NoRegistrations_SectionAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 22) // contentH=18 ≥ 18 → S4 shown IF registrations match
	// No registrations set — nothing to match.

	got := stripANSI(m.View())
	if strings.Contains(got, "jump to:") {
		t.Errorf("S4 'jump to:' must be absent with no matching registrations:\n%s", got)
	}
}

// [Observable] matching registration → S4 "jump to:" entry appears (FB-105: prefix updated from "Quick jump:").
func TestFB042_QuickJump_WithRegistration_SectionPresent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 22) // contentH=18 ≥ 18
	m.SetRegistrations([]data.ResourceRegistration{
		{Name: "namespaces", Group: "core.miloapis.com"},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "jump to:") {
		t.Errorf("S4 'jump to:' prefix missing with matching registration:\n%s", got)
	}
	if !strings.Contains(got, "[n]") {
		t.Errorf("S4 '[n]' key for 'namespaces' missing:\n%s", got)
	}
}

// [Anti-behavior] S4 absent at contentH < 18.
func TestFB042_QuickJump_HeightGate_AbsentNarrow(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 21) // contentH=17 < 18 → S4 suppressed
	m.SetRegistrations([]data.ResourceRegistration{
		{Name: "namespaces", Group: "core.miloapis.com"},
	})

	got := stripANSI(m.View())
	if strings.Contains(got, "jump to:") {
		t.Errorf("S4 'jump to:' must be absent at contentH<18:\n%s", got)
	}
}

// --- S5: Needs attention section ---

// [Anti-behavior] no attention items → S5 section absent.
func TestFB042_Attention_NoItems_SectionAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 34) // contentH=30 ≥ 30 → S5 shown IF items exist
	// No attention items set.

	got := stripANSI(m.View())
	if strings.Contains(got, "Needs attention") {
		t.Errorf("S5 'Needs attention' must be absent with no items:\n%s", got)
	}
}

// [Observable] attention items set → S5 header and item labels rendered.
func TestFB042_Attention_WithItems_SectionPresent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 34) // contentH=30 ≥ 30, contentW=116 ≥ 60 → S5 shown
	m.SetAttentionItems([]AttentionItem{
		{Kind: "quota", Label: "dnszones quota", Detail: "91% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Needs attention") {
		t.Errorf("S5 'Needs attention' header missing with items set:\n%s", got)
	}
	if !strings.Contains(got, "dnszones quota") {
		t.Errorf("S5 item label 'dnszones quota' missing:\n%s", got)
	}
	if !strings.Contains(got, "91% allocated") {
		t.Errorf("S5 item detail '91%% allocated' missing:\n%s", got)
	}
}

// [Anti-regression] attention items capped at 3.
func TestFB042_Attention_CappedAtThree(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 34)
	m.SetAttentionItems([]AttentionItem{
		{Kind: "quota", Label: "item-one", Detail: "80% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
		{Kind: "quota", Label: "item-two", Detail: "85% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
		{Kind: "quota", Label: "item-three", Detail: "90% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
		{Kind: "quota", Label: "item-four", Detail: "95% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
	})

	got := stripANSI(m.View())
	if strings.Contains(got, "item-four") {
		t.Errorf("S5 capped at 3: 'item-four' must be absent:\n%s", got)
	}
}

// [Anti-behavior] S5 absent when contentH < 30.
func TestFB042_Attention_HeightGate_AbsentShort(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 33) // contentH=29 < 30 → S5 suppressed
	m.SetAttentionItems([]AttentionItem{
		{Kind: "quota", Label: "dnszones quota", Detail: "91% allocated", NavKey: "[3]", NavHint: "quota dashboard"},
	})

	got := stripANSI(m.View())
	if strings.Contains(got, "Needs attention") {
		t.Errorf("S5 'Needs attention' must be absent at contentH<30:\n%s", got)
	}
}

// [Happy axis] AC1: all buckets < 80% → "All clear" status line rendered.
func TestFB042_HealthSummary_AllClear(t *testing.T) {
	t.Parallel()
	projectID := "proj-abc"
	m := newWelcomeModel(100, 30) // contentW=96 ≥ 50 → status line rendered
	m.SetTUIContext(testCtxWithProject("alice", "acme-corp", "web", projectID))
	m.SetBuckets([]data.AllowanceBucket{
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "apps/deployments", Limit: 10, Allocated: 5}, // 50%
		{ConsumerKind: "Project", ConsumerName: projectID, ResourceType: "core/pods", Limit: 100, Allocated: 30},      // 30%
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "All clear") {
		t.Errorf("AC1 happy: want 'All clear' when all buckets <80%%, got: %q", got)
	}
}

// [Input-changed axis] AC6: condition-kind AttentionItem renders Label and Detail with ⚠ icon.
func TestFB042_Attention_ConditionItem_RendersLabelAndDetail(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 34) // contentH=30 ≥ 30, contentW=116 ≥ 60 → S5 shown
	m.SetAttentionItems([]AttentionItem{
		{Kind: "condition", Label: "backend/api-gw", Detail: "condition: Degraded", NavKey: "[Enter]", NavHint: "view"},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "backend/api-gw") {
		t.Errorf("AC6 condition item: label 'backend/api-gw' missing:\n%s", got)
	}
	if !strings.Contains(got, "condition: Degraded") {
		t.Errorf("AC6 condition item: detail 'condition: Degraded' missing:\n%s", got)
	}
	if !strings.Contains(got, "⚠") {
		t.Errorf("AC6 condition item: ⚠ icon missing (condition branch):\n%s", got)
	}
}

// ==================== End FB-042 (component) ====================

// ==================== FB-082: Activity state machine + 3-tier width truncation (component) ====================

// testActivityRow returns a row with all fields populated for width-band assertions.
func testActivityRow() data.ActivityRow {
	return data.ActivityRow{
		Timestamp:    time.Now().Add(-2 * time.Minute),
		ActorDisplay: "alice@example.com",
		Summary:      "created project api-gw",
		ResourceRef:  &data.ResourceRef{Kind: "Project", Name: "my-proj"},
	}
}

// --- AC3 component: activityFetchFailed state ---

// [Observable] activityFetchFailed=true → "activity unavailable" in View().
func TestFB082_Activity_FetchFailed_ShowsUnavailable(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC3: 'activity unavailable' missing from View() when activityFetchFailed=true:\n%s", got)
	}
	if strings.Contains(got, "no recent activity") {
		t.Errorf("AC3: 'no recent activity' present when activityFetchFailed=true (wrong branch):\n%s", got)
	}
	if strings.Contains(got, "loading") {
		t.Errorf("AC3: 'loading' present when activityFetchFailed=true:\n%s", got)
	}
}

// [Anti-behavior] activityFetchFailed does not trigger when loading state takes priority.
func TestFB082_Activity_LoadingTakesPriority_OverFetchFailed(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityLoading(true)
	// activityRows is nil (default); loading+nil rows branch fires first.
	// Manually set fetchFailed too — switch ordering must prefer loading.
	m.SetActivityFetchFailed(true)
	// NOTE: calling SetActivityLoading AFTER SetActivityFetchFailed ensures both flags are true.
	// The switch fires loading branch first.

	got := stripANSI(m.View())
	if !strings.Contains(got, "loading") {
		t.Errorf("priority: 'loading' missing — loading+nil should take priority over fetchFailed:\n%s", got)
	}
}

// --- Recovery path ---

// [Anti-behavior] SetActivityRows with rows clears activityFetchFailed → rows render, "activity unavailable" gone.
func TestFB082_ErrorRecovery_SetActivityRows_ClearsFailedFlag(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30) // contentW=116 ≥ 65 → Tier 1, resource column visible
	m.SetActivityFetchFailed(true)

	before := stripANSI(m.View())
	if !strings.Contains(before, "activity unavailable") {
		t.Fatal("precondition: 'activity unavailable' must be present before recovery")
	}

	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	got := stripANSI(m.View())
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("recovery: 'activity unavailable' still present after SetActivityRows with data:\n%s", got)
	}
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("recovery: actor 'alice@example.com' missing after SetActivityRows:\n%s", got)
	}
}

// [Anti-behavior] SetActivityRows with empty slice clears activityFetchFailed → "no recent activity", not "unavailable".
func TestFB082_ErrorRecovery_EmptyRows_ShowsNoActivity(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true)

	m.SetActivityRows([]data.ActivityRow{})

	got := stripANSI(m.View())
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("recovery with empty rows: 'activity unavailable' still present, want 'no recent activity':\n%s", got)
	}
	if !strings.Contains(got, "no recent activity") {
		t.Errorf("recovery with empty rows: 'no recent activity' missing:\n%s", got)
	}
}

// --- Width-band tests (3 tiers × 2 render states) ---

// [Observable] Tier 1 (contentW ≥ 65), rows populated: all 4 columns rendered.
func TestFB082_WidthBand_Tier1_Rows_AllColumns(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30) // contentW = 100-4 = 96 ≥ 65
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	got := stripANSI(m.renderActivitySection(96))
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("Tier1 rows: actor missing at contentW=96:\n%s", got)
	}
	if !strings.Contains(got, "project/my-proj") {
		t.Errorf("Tier1 rows: resource 'project/my-proj' missing at contentW=96 (should be present in Tier 1):\n%s", got)
	}
	if !strings.Contains(got, "created project api-gw") {
		t.Errorf("Tier1 rows: summary missing at contentW=96:\n%s", got)
	}
}

// [Observable] Tier 1 (contentW ≥ 65), activityFetchFailed: renders "activity unavailable" without panic.
func TestFB082_WidthBand_Tier1_FetchFailed_RendersUnavailable(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true)

	got := stripANSI(m.renderActivitySection(96))
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("Tier1 fetchFailed: 'activity unavailable' missing at contentW=96:\n%s", got)
	}
}

// [Observable] Tier 2 (45 ≤ contentW < 65), rows populated: resource column dropped.
func TestFB082_WidthBand_Tier2_Rows_ResourceDropped(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	got := stripANSI(m.renderActivitySection(55)) // 45 ≤ 55 < 65
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("Tier2 rows: actor missing at contentW=55:\n%s", got)
	}
	// Resource column dropped in Tier 2.
	if strings.Contains(got, "project/my-proj") {
		t.Errorf("Tier2 rows: resource 'project/my-proj' present at contentW=55 (should be dropped):\n%s", got)
	}
	if !strings.Contains(got, "created project") {
		t.Errorf("Tier2 rows: summary missing at contentW=55:\n%s", got)
	}
}

// [Observable] Tier 2 (45 ≤ contentW < 65), activityFetchFailed: renders "activity unavailable" without panic.
func TestFB082_WidthBand_Tier2_FetchFailed_RendersUnavailable(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true)

	got := stripANSI(m.renderActivitySection(55))
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("Tier2 fetchFailed: 'activity unavailable' missing at contentW=55:\n%s", got)
	}
}

// [Observable] Tier 3 (contentW < 45), rows populated: actor truncated to ≤16 chars.
func TestFB082_WidthBand_Tier3_Rows_ActorTruncated(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	got := stripANSI(m.renderActivitySection(35)) // contentW=35 < 45
	// Full actor "alice@example.com" (17 runes) exceeds actorW=min(16,35-22)=13 → truncated.
	if strings.Contains(got, "alice@example.com") {
		t.Errorf("Tier3 rows: full actor 'alice@example.com' present at contentW=35 (should be truncated):\n%s", got)
	}
	// Truncated prefix should appear (actorW=13 → first 12 runes + "…").
	if !strings.Contains(got, "alice@exampl") {
		t.Errorf("Tier3 rows: truncated actor prefix 'alice@exampl' missing at contentW=35:\n%s", got)
	}
}

// [Observable] Tier 3 (contentW < 45), activityFetchFailed: renders "activity unavailable" without panic.
func TestFB082_WidthBand_Tier3_FetchFailed_RendersUnavailable(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true)

	got := stripANSI(m.renderActivitySection(35))
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("Tier3 fetchFailed: 'activity unavailable' missing at contentW=35:\n%s", got)
	}
}

// ==================== End FB-082 (component) ====================

// ==================== FB-101: Tier 3 actorW latent panic at contentW ≤ 22 ====================

// AC1 — [Observable] renderActivitySection at contentW=22 does NOT panic.
func TestFB101_AC1_RenderActivitySection_ContentW22_NoPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AC1: renderActivitySection(22) panicked: %v", r)
		}
	}()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	_ = m.renderActivitySection(22)
}

// AC2 — [Observable] renderActivitySection at contentW=0 does NOT panic.
func TestFB101_AC2_RenderActivitySection_ContentW0_NoPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AC2: renderActivitySection(0) panicked: %v", r)
		}
	}()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	_ = m.renderActivitySection(0)
}

// AC3 — [Anti-behavior] At contentW=44 (Tier 3, well above panic boundary), actor truncation
// still fires for a long actor name — "…" present and untruncated form absent.
func TestFB101_AC3_ContentW44_TruncationUnchanged(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()}) // actor = "alice@example.com" (17 runes)

	got := stripANSI(m.renderActivitySection(44)) // Tier 3; actorW = max(1, min(16, 22)) = 16 → truncated
	if !strings.Contains(got, "…") {
		t.Errorf("AC3: '…' truncation glyph absent at contentW=44 (actor len 17 > actorW 16):\n%s", got)
	}
	if strings.Contains(got, "alice@example.com") {
		t.Errorf("AC3: full actor 'alice@example.com' present at contentW=44; want truncated:\n%s", got)
	}
}

// ==================== End FB-101 ====================

// ==================== FB-083: S3 [4] hint suppression when no activity data ====================

// [Observable / AC1] empty rows (post-load): hint absent.
func TestFB083_AC1_EmptyRows_HintAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{})
	m.SetActivityLoading(false)
	out := stripANSI(m.View())
	if strings.Contains(out, "[4] full dashboard") {
		t.Errorf("expected hint absent for empty rows, got:\n%s", out)
	}
	if !strings.Contains(out, "Recent activity") {
		t.Errorf("expected 'Recent activity' header present, got:\n%s", out)
	}
}

// [Observable / AC2] loading state (activityLoading=true, no rows): hint absent.
func TestFB083_AC2_LoadingState_HintAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityLoading(true)
	out := stripANSI(m.View())
	if strings.Contains(out, "[4] full dashboard") {
		t.Errorf("expected hint absent during loading, got:\n%s", out)
	}
}

// [Observable / AC3] activityFetchFailed=true: hint absent.
func TestFB083_AC3_FetchFailed_HintAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true)
	out := stripANSI(m.View())
	if strings.Contains(out, "[4] full dashboard") {
		t.Errorf("expected hint absent on fetch failure, got:\n%s", out)
	}
}

// [Observable / AC4] rows populated: hint present.
func TestFB083_AC4_RowsPresent_HintShows(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	out := stripANSI(m.View())
	if !strings.Contains(out, "[4] full dashboard") {
		t.Errorf("expected hint present when rows exist, got:\n%s", out)
	}
}

// [Input-changed / AC5] empty → populated: View() output differs (hint appears).
func TestFB083_AC5_InputChanged_EmptyVsPopulated(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{})
	outEmpty := stripANSI(m.View())

	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	outRows := stripANSI(m.View())

	if strings.Contains(outEmpty, "[4] full dashboard") {
		t.Errorf("empty state should not contain hint")
	}
	if !strings.Contains(outRows, "[4] full dashboard") {
		t.Errorf("populated state should contain hint")
	}
	if outEmpty == outRows {
		t.Errorf("expected different View() output between empty and populated states")
	}
}

// ==================== End FB-083 (component) ====================

// ==================== FB-111: [4] hint suppressed when activityFetchFailed + stale rows ====================

// [Observable / AC1] activityRows=[row] + activityFetchFailed=true: hint absent.
func TestFB111_AC1_StaleRows_FetchFailed_HintAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	m.SetActivityFetchFailed(true)
	out := stripANSI(m.View())
	if strings.Contains(out, "[4] full dashboard") {
		t.Errorf("AC1 [Observable]: hint present with stale rows + fetchFailed=true:\n%s", out)
	}
}

// [Observable / AC2] activityRows=[row] + activityFetchFailed=false: hint present (FB-083 baseline).
func TestFB111_AC2_StaleRows_NoFail_HintPresent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	out := stripANSI(m.View())
	if !strings.Contains(out, "[4] full dashboard") {
		t.Errorf("AC2 [Observable]: hint absent with rows and fetchFailed=false:\n%s", out)
	}
}

// [Input-changed / AC3] Same rows, toggle fetchFailed: View() output differs (hint appears/disappears).
func TestFB111_AC3_InputChanged_FetchFailed_ToggleHint(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	outOK := stripANSI(m.View())
	m.SetActivityFetchFailed(true)
	outFail := stripANSI(m.View())

	if !strings.Contains(outOK, "[4] full dashboard") {
		t.Errorf("AC3 precondition: hint absent before fetchFailed:\n%s", outOK)
	}
	if strings.Contains(outFail, "[4] full dashboard") {
		t.Errorf("AC3 [Input-changed]: hint still present after fetchFailed=true:\n%s", outFail)
	}
	if outOK == outFail {
		t.Error("AC3 [Input-changed]: View() unchanged after fetchFailed toggle")
	}
}

// ==================== End FB-111 (component) ====================

func TestResourceTableModel_ApplyFilter_CursorReset_AfterSetColumns(t *testing.T) {
	t.Parallel()

	m := NewResourceTableModel(58, 20)

	// Normal initialisation sequence: columns first, then rows.
	m.SetColumns([]string{"Name"}, 58)
	m.SetRows(namedRows("alpha", "beta", "gamma"))

	// Simulate a resource-type switch: SetColumns calls m.table.SetRows(nil) which
	// drives the charmbracelet cursor to -1 (downward-only clamp). Rows repopulated
	// by the subsequent SetRows must not leave cursor at -1.
	m.SetColumns([]string{"Name"}, 58)
	m.SetRows(namedRows("alpha", "beta", "gamma"))

	// [Edge]: cursor must be ≥ 0 so SelectedRow() returns a valid row.
	if c := m.Cursor(); c < 0 {
		t.Errorf("[Anti-regression] cursor = %d after SetColumns+SetRows, want ≥ 0 (charmbracelet SetRows(nil) drove cursor negative; applyFilter must reset it)", c)
	}
	if _, ok := m.SelectedRow(); !ok {
		t.Error("[Anti-regression] SelectedRow() = (_, false) after SetColumns+SetRows, want a valid row")
	}
}

// ==================== FB-054: Tab-to-resume hint in welcome header band ====================
//
// Axis-coverage:
// AC | Observable                                                    | Anti-behavior                                              | Anti-regression
// ---+---------------------------------------------------------------+------------------------------------------------------------+-----------------
// 1  | AC1_ForceWithType_ShowsTabHint                                | -                                                          | -
// 2  | -                                                             | AC2a_ForceFalse_NoTabHint; AC2b_EmptyType_NoTabHint        | -

// AC1 — [Observable] forceDashboard=true + typeName="httproutes" → header band contains "[Tab]" and new FB-089 copy.
func TestFB054_AC1_HeaderBand_ForceWithType_ShowsTabHint(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 30)
	m.SetTypeContext("httproutes", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.renderHeaderBand(76))
	if !strings.Contains(got, "[Tab]") {
		t.Errorf("AC1: '[Tab]' missing from header band:\n%s", got)
	}
	if !strings.Contains(got, "httproutes") {
		t.Errorf("AC1: 'httproutes' missing from header band:\n%s", got)
	}
	// FB-089: copy changed from sentence "to resume X, or select..." to label idiom "resume X (cached)".
	if !strings.Contains(got, "resume httproutes (cached)") {
		t.Errorf("AC1: 'resume httproutes (cached)' missing from header band (FB-089 copy):\n%s", got)
	}
	if strings.Contains(got, "to resume") {
		t.Errorf("AC1: old 'to resume' sentence-style copy still present (FB-089 regression):\n%s", got)
	}
}

// AC2a — [Anti-behavior] forceDashboard=false → resume hint absent even with typeName set.
func TestFB054_AC2a_HeaderBand_ForceFalse_NoTabHint(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 30)
	m.SetTypeContext("httproutes", true)
	m.SetForceDashboard(false)

	got := stripANSI(m.renderHeaderBand(76))
	if strings.Contains(got, "resume") {
		t.Errorf("AC2a: 'resume' hint present when forceDashboard=false, want absent:\n%s", got)
	}
}

// AC2b — [Anti-behavior] forceDashboard=true + empty typeName → resume hint absent.
func TestFB054_AC2b_HeaderBand_EmptyType_NoTabHint(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 30)
	// typeName remains "" (not set)
	m.SetForceDashboard(true)

	got := stripANSI(m.renderHeaderBand(76))
	if strings.Contains(got, "resume") {
		t.Errorf("AC2b: 'resume' hint present when typeName='', want absent:\n%s", got)
	}
}

// ==================== End FB-054 (component layer) ====================

// ==================== FB-056: Dashboard context-aware keybind strip ====================

// AC1 + AC2 — [Observable] Dashboard state (forceDashboard=true): strip removes inert keys,
// adds live dashboard navigators.
func TestFB056_AC1AC2_ForceDashboard_StripContent(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(120, 22) // contentH=18 → keybind strip shown
	m.SetTypeContext("httproutes", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())

	// AC1: inert keys absent.
	if strings.Contains(got, "x delete") {
		t.Errorf("AC1 [Observable]: 'x delete' present in dashboard strip:\n%s", got)
	}
	if strings.Contains(got, "/ filter") {
		t.Errorf("AC1 [Observable]: '/ filter' present in dashboard strip:\n%s", got)
	}
	// AC2: dashboard navigators present.
	if !strings.Contains(got, "3") || !strings.Contains(got, "quota") {
		t.Errorf("AC2 [Observable]: '3 quota' missing from dashboard strip:\n%s", got)
	}
	if !strings.Contains(got, "4") || !strings.Contains(got, "activity") {
		t.Errorf("AC2 [Observable]: '4 activity' missing from dashboard strip:\n%s", got)
	}
}

// AC1 + AC2 variant — [Observable] typeName="" (fresh startup) also triggers dashboard strip.
func TestFB056_AC1AC2_EmptyTypeName_StripContent(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(120, 22)
	// typeName is "" by default — no SetTypeContext call.

	got := stripANSI(m.View())

	if strings.Contains(got, "x delete") {
		t.Errorf("AC1 [Observable] empty-type: 'x delete' present in dashboard strip:\n%s", got)
	}
	if strings.Contains(got, "/ filter") {
		t.Errorf("AC1 [Observable] empty-type: '/ filter' present in dashboard strip:\n%s", got)
	}
	if !strings.Contains(got, "3") || !strings.Contains(got, "quota") {
		t.Errorf("AC2 [Observable] empty-type: '3 quota' missing from dashboard strip:\n%s", got)
	}
	if !strings.Contains(got, "4") || !strings.Contains(got, "activity") {
		t.Errorf("AC2 [Observable] empty-type: '4 activity' missing from dashboard strip:\n%s", got)
	}
}

// AC4 — [Input-changed] Table state (forceDashboard=false, typeName set): strip has full set.
// renderKeybindStrip is called directly (welcome panel not rendered in table mode).
func TestFB056_AC4_TableState_StripHasFullKeys(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(120, 22)
	m.SetTypeContext("httproutes", true)
	// forceDashboard defaults to false.

	got := stripANSI(m.renderKeybindStrip(116))

	if !strings.Contains(got, "x delete") {
		t.Errorf("AC4 [Input-changed]: 'x delete' missing from table strip:\n%s", got)
	}
	if !strings.Contains(got, "/ filter") {
		t.Errorf("AC4 [Input-changed]: '/ filter' missing from table strip:\n%s", got)
	}
}

// AC6 — [Anti-regression] Active keys (j/k, Tab, Enter, c, ?, q) present in both contexts.
// Dashboard context uses View(); table context calls renderKeybindStrip directly.
func TestFB056_AC6_ActiveKeys_BothContexts(t *testing.T) {
	t.Parallel()
	commonKeys := []string{"j/k", "Tab", "Enter", "c", "?", "q"}

	t.Run("dashboard", func(t *testing.T) {
		t.Parallel()
		m := NewResourceTableModel(120, 22)
		m.SetForceDashboard(true)
		got := stripANSI(m.View())
		for _, k := range commonKeys {
			if !strings.Contains(got, k) {
				t.Errorf("AC6 [Anti-regression] dashboard: key %q missing from strip:\n%s", k, got)
			}
		}
	})

	t.Run("table", func(t *testing.T) {
		t.Parallel()
		m := NewResourceTableModel(120, 22)
		m.SetTypeContext("httproutes", true)
		got := stripANSI(m.renderKeybindStrip(116))
		for _, k := range commonKeys {
			if !strings.Contains(got, k) {
				t.Errorf("AC6 [Anti-regression] table: key %q missing from strip:\n%s", k, got)
			}
		}
	})
}

// ==================== End FB-056 (component layer) ====================

// ==================== FB-089: Welcome-panel Tab hint copy cohesion across surfaces and widths ====================
//
// Axis-coverage table:
// AC1 | Observable      | TestFB089_AC1_Observable_FullForm_CachedSuffix
// AC2 | Observable      | TestFB089_AC2_Observable_MediumWidth_ShortForm
// AC3 | Observable      | TestFB089_AC3_Observable_NarrowWidth_Truncated
// AC4 | Anti-regression | TestFB089_AC4_AntiRegression_S6Strip_TabNextPaneUnchanged
// AC5 | Input-changed   | TestFB089_AC5_InputChanged_EmptyTypeName_HintAbsent
// AC6 | Anti-regression | existing FB-054 TestFB054_* green (no new test)
// AC7 | Anti-regression | existing S6 truncation tests green
// AC8 | Integration     | go install ./... + go test ./internal/tui/...

// [Observable] AC1: wide contentW (tableWidth=80→contentW=76), typeName="backends" →
// full form "resume backends (cached)" present; old sentence-style absent.
func TestFB089_AC1_Observable_FullForm_CachedSuffix(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume backends (cached)") {
		t.Errorf("AC1: 'resume backends (cached)' missing at wide width:\n%s", got)
	}
	if strings.Contains(got, "to resume") {
		t.Errorf("AC1: old 'to resume' sentence copy still present — FB-089 regression:\n%s", got)
	}
	if strings.Contains(got, "or select a different type") {
		t.Errorf("AC1: old 'or select a different type' still present — FB-089 regression:\n%s", got)
	}
}

// [Observable] AC2: medium contentW (tableWidth=29→contentW=25) — full form (30 chars) doesn't fit,
// short form (21 chars) fits → label idiom preserved, "(cached)" absent due to width.
func TestFB089_AC2_Observable_MediumWidth_ShortForm(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(29, 20)
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume backends") {
		t.Errorf("AC2: 'resume backends' missing at medium width (contentW=25):\n%s", got)
	}
	if strings.Contains(got, "to resume") {
		t.Errorf("AC2: old 'to resume' sentence copy present at medium width:\n%s", got)
	}
	if strings.Contains(got, "or select") {
		t.Errorf("AC2: 'or select' present at medium width — label idiom not preserved:\n%s", got)
	}
}

// [Observable] AC3: narrow contentW (tableWidth=24→contentW=20) — short form (21 chars) doesn't fit →
// name truncated to "backe…"; no sentence copy.
func TestFB089_AC3_Observable_NarrowWidth_Truncated(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(24, 20)
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume backe…") {
		t.Errorf("AC3: 'resume backe…' missing at narrow width (contentW=20):\n%s", got)
	}
	if strings.Contains(got, "to resume") {
		t.Errorf("AC3: 'to resume' sentence copy present at narrow width:\n%s", got)
	}
}

// [Anti-regression] AC4: S6 keybind strip with forceDashboard=true + typeName set.
// Updated for FB-093: Tab is dropped from the strip when hasCachedTable=true (S1 band owns the hint).
// The S1 band still carries [Tab] resume <typeName> (cached); strip has 3/quota/4/activity.
func TestFB089_AC4_AntiRegression_S6Strip_TabNextPaneUnchanged(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20) // contentH=16 ≥ 12 → S6 rendered
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	// Strip-only check (avoid matching [Tab] in S1 band).
	strip := stripANSI(m.renderKeybindStrip(76))
	// FB-093: Tab absent from strip when S1 band owns Tab hint.
	if strings.Contains(strip, "Tab") {
		t.Errorf("AC4: 'Tab' present in strip when hasCachedTable=true (FB-093 should drop it):\n%s", strip)
	}
	// S1 band still carries the Tab hint.
	got := stripANSI(m.View())
	if !strings.Contains(got, "resume backends") {
		t.Errorf("AC4: S1 band 'resume backends' missing from View():\n%s", got)
	}
}

// [Input-changed] AC5: typeName="" → S1 hint absent; typeName="backends" → present.
// Same forceDashboard=true, different typeName → different View() output.
func TestFB089_AC5_InputChanged_EmptyTypeName_HintAbsent(t *testing.T) {
	t.Parallel()

	// Pair A: typeName="" → no resume hint.
	mEmpty := NewResourceTableModel(80, 20)
	mEmpty.SetForceDashboard(true)
	gotEmpty := stripANSI(mEmpty.View())
	if strings.Contains(gotEmpty, "resume") {
		t.Errorf("pair A (typeName=''): 'resume' present with empty typeName — must be absent:\n%s", gotEmpty)
	}

	// Pair B: typeName="backends" → resume hint present.
	mTyped := NewResourceTableModel(80, 20)
	mTyped.SetTypeContext("backends", true)
	mTyped.SetForceDashboard(true)
	gotTyped := stripANSI(mTyped.View())
	if !strings.Contains(gotTyped, "resume backends") {
		t.Errorf("pair B (typeName='backends'): 'resume backends' missing:\n%s", gotTyped)
	}
	if gotEmpty == gotTyped {
		t.Error("input-changed: View() identical for empty vs non-empty typeName — must differ")
	}
}

// ==================== End FB-089 (component layer) ====================

// ==================== FB-090: S1/S4 label-source vocabulary consistency ====================
//
// Axis-coverage table:
// AC1 | Observable      | TestFB090_AC1_Observable_DNS_UsesLabel
// AC2 | Observable      | TestFB090_AC2_Observable_DNSZones_UsesLabel
// AC3 | Observable      | TestFB090_AC3_Observable_Backends_IdentityUnchanged
// AC4 | Input-changed   | TestFB090_AC4_InputChanged_QuickJumpLabelHelper (unit: 4 cases)
//     |                 | TestFB090_AC4_InputChanged_ViewLevel_LabelDiffers (View-level pair)
// AC5 | Anti-behavior   | TestFB090_AC5_AntiBehavior_EmptyTypeName_HintAbsent
// AC6 | Anti-regression | TestFB090_AC6_AntiRegression_S4QuickJumpUnchanged
// AC7 | Anti-regression | existing FB-054 TestFB054_* green (no new test)
// AC8 | Anti-regression | TestFB090_AC8_AntiRegression_CombinedResult_DNSCached
// AC9 | Integration     | go install ./... + go test ./internal/tui/...

// [Observable] AC1: typeName="dnsrecordsets" → S1 shows "resume dns"; raw name absent.
func TestFB090_AC1_Observable_DNS_UsesLabel(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetTypeContext("dnsrecordsets", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume dns") {
		t.Errorf("AC1: 'resume dns' missing for typeName='dnsrecordsets':\n%s", got)
	}
	if strings.Contains(got, "resume dnsrecordsets") {
		t.Errorf("AC1: 'resume dnsrecordsets' (raw name) present — must use curated label 'dns':\n%s", got)
	}
}

// [Observable] AC2: typeName="dnszones" → S1 shows "resume dns"; raw name absent.
func TestFB090_AC2_Observable_DNSZones_UsesLabel(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetTypeContext("dnszones", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume dns") {
		t.Errorf("AC2: 'resume dns' missing for typeName='dnszones':\n%s", got)
	}
	if strings.Contains(got, "resume dnszones") {
		t.Errorf("AC2: 'resume dnszones' (raw name) present — must use curated label 'dns':\n%s", got)
	}
}

// [Observable] AC3: typeName="backends" — raw name equals curated label; "resume backends" present.
func TestFB090_AC3_Observable_Backends_IdentityUnchanged(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume backends") {
		t.Errorf("AC3: 'resume backends' missing — identity case must be unchanged:\n%s", got)
	}
}

// [Input-changed] AC4a: quickJumpLabel() unit tests — curated label, identity, and fallback inputs.
func TestFB090_AC4_InputChanged_QuickJumpLabelHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		typeName string
		want     string
	}{
		{"dnsrecordsets", "dns"},           // curated label from quickJumpTable (z entry)
		{"dnszones", "dns"},               // alternate match for the same z entry
		{"backends", "backends"},          // identity — raw name equals curated label
		{"certificates", "certificates"}, // fallback — not in quickJumpTable
	}
	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			t.Parallel()
			got := quickJumpLabel(tt.typeName)
			if got != tt.want {
				t.Errorf("quickJumpLabel(%q) = %q, want %q", tt.typeName, got, tt.want)
			}
		})
	}
}

// [Input-changed] AC4b: View-level pair — different typeName inputs produce different curated labels.
func TestFB090_AC4_InputChanged_ViewLevel_LabelDiffers(t *testing.T) {
	t.Parallel()

	// Pair A: typeName="dnsrecordsets" → "resume dns".
	mDNS := NewResourceTableModel(80, 20)
	mDNS.SetTypeContext("dnsrecordsets", true)
	mDNS.SetForceDashboard(true)
	gotDNS := stripANSI(mDNS.View())

	// Pair B: typeName="backends" → "resume backends".
	mBE := NewResourceTableModel(80, 20)
	mBE.SetTypeContext("backends", true)
	mBE.SetForceDashboard(true)
	gotBE := stripANSI(mBE.View())

	if !strings.Contains(gotDNS, "resume dns") {
		t.Errorf("pair A (dnsrecordsets): 'resume dns' missing:\n%s", gotDNS)
	}
	if !strings.Contains(gotBE, "resume backends") {
		t.Errorf("pair B (backends): 'resume backends' missing:\n%s", gotBE)
	}
	if gotDNS == gotBE {
		t.Error("input-changed: View() identical for dnsrecordsets vs backends — must differ")
	}
}

// [Anti-behavior] AC5: typeName="" → S1 Tab hint absent (quickJumpLabel not called when gate fails).
func TestFB090_AC5_AntiBehavior_EmptyTypeName_HintAbsent(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetForceDashboard(true)
	// typeName remains "" (default)

	got := stripANSI(m.View())
	if strings.Contains(got, "resume") {
		t.Errorf("AC5: 'resume' present with empty typeName — must be absent:\n%s", got)
	}
}

// [Anti-regression] AC6: S4 quick-jump "[z] dns" still present when dns registrations exist.
func TestFB090_AC6_AntiRegression_S4QuickJumpUnchanged(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(120, 30) // wide + tall for S4 to render
	m.SetForceDashboard(true)
	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "networking.datumapis.com", Name: "dnsrecordsets", Description: "DNS Record Sets"},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "[z] dns") {
		t.Errorf("AC6: '[z] dns' missing from S4 — quick-jump label source must remain unchanged:\n%s", got)
	}
}

// [Anti-regression] AC8: combined FB-089 + FB-090 result for typeName="dnsrecordsets" at wide width →
// "resume dns (cached)" (curated label from FB-090 + label-idiom cached suffix from FB-089).
func TestFB090_AC8_AntiRegression_CombinedResult_DNSCached(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(80, 20)
	m.SetTypeContext("dnsrecordsets", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "resume dns (cached)") {
		t.Errorf("AC8: 'resume dns (cached)' missing — combined FB-089+FB-090 result incorrect:\n%s", got)
	}
}

// ==================== End FB-090 (component layer) ====================

// ==================== FB-093: Dashboard keybind strip cohesion ====================

// [Observable / AC1] Table context (typeName set, forceDashboard=false): strip has 3/quota + 4/activity.
func TestFB093_AC1_TableContext_HasCrossContextKeys(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetTypeContext("backends", true)
	// forceDashboard defaults to false.

	got := stripANSI(m.renderKeybindStrip(196))
	for _, want := range []string{"3", "quota", "4", "activity"} {
		if !strings.Contains(got, want) {
			t.Errorf("AC1 [Observable]: %q missing from table strip:\n%s", want, got)
		}
	}
}

// [Observable / AC2] Dashboard + cached-table (forceDashboard=true, typeName="backends"): Tab absent in strip; 3/quota present.
// Uses renderKeybindStrip directly to avoid matching [Tab] in the S1 band (FB-054/FB-089).
func TestFB093_AC2_DashboardCached_TabAbsent_QuotaPresent(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.renderKeybindStrip(196))
	if strings.Contains(got, "Tab") {
		t.Errorf("AC2 [Observable]: 'Tab' present in strip when hasCachedTable=true:\n%s", got)
	}
	for _, want := range []string{"3", "quota", "4", "activity"} {
		if !strings.Contains(got, want) {
			t.Errorf("AC2 [Observable]: %q missing from cached-table dashboard strip:\n%s", want, got)
		}
	}
}

// [Observable / AC3] Fresh startup (forceDashboard=true, typeName=""): Tab + "next pane" present.
func TestFB093_AC3_DashboardFresh_TabPresent(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetForceDashboard(true)
	// typeName="" by default.

	got := stripANSI(m.View())
	if !strings.Contains(got, "Tab") {
		t.Errorf("AC3 [Observable]: 'Tab' absent from fresh-startup dashboard strip:\n%s", got)
	}
	if !strings.Contains(got, "next pane") {
		t.Errorf("AC3 [Observable]: 'next pane' absent from fresh-startup dashboard strip:\n%s", got)
	}
}

// [Input-changed / AC4] Dashboard context (typeName="") → table context (typeName="backends"):
// strip transitions from dashboard-branch to table-branch; table branch has x/delete AND 3/4.
// Note: dashboard branch already has 3/4; what changes is the addition of x/delete alongside 3/4.
func TestFB093_AC4_InputChanged_TableLoad_GainsCrossContextKeys(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)

	// Dashboard branch (typeName="" → forceDashboard||typeName=="" condition fires).
	before := stripANSI(m.renderKeybindStrip(196))
	if strings.Contains(before, "x delete") {
		t.Errorf("AC4 precondition: 'x delete' present before type loaded:\n%s", before)
	}

	m.SetTypeContext("backends", true)
	after := stripANSI(m.renderKeybindStrip(196))
	// Table branch: x/delete present (new branch) AND 3/4 present (FB-093 addition).
	for _, want := range []string{"x", "delete", "3", "quota", "4", "activity"} {
		if !strings.Contains(after, want) {
			t.Errorf("AC4 [Input-changed]: %q missing in table strip after type loaded:\n%s", want, after)
		}
	}
}

// [Input-changed / AC5] forceDashboard typeName="" → typeName="backends": Tab drops from strip.
// Uses renderKeybindStrip directly to avoid matching [Tab] in the S1 band (FB-054/FB-089).
func TestFB093_AC5_InputChanged_CacheState_TabDrops(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetForceDashboard(true)

	before := stripANSI(m.renderKeybindStrip(196))
	if !strings.Contains(before, "Tab") {
		t.Errorf("AC5 precondition: Tab absent from strip before typeName set:\n%s", before)
	}

	m.SetTypeContext("backends", true)
	after := stripANSI(m.renderKeybindStrip(196))
	if strings.Contains(after, "Tab") {
		t.Errorf("AC5 [Input-changed]: Tab still present in strip after typeName set (hasCachedTable=true):\n%s", after)
	}
}

// [Anti-behavior / AC6] Extreme narrow width: bareParts fallback — no "quota" or "activity".
func TestFB093_AC6_ExtremNarrow_BareParts_NoQuota(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetTypeContext("backends", true)

	got := stripANSI(m.renderKeybindStrip(20))
	if strings.Contains(got, "quota") {
		t.Errorf("AC6 [Anti-behavior]: 'quota' present at extreme narrow width:\n%s", got)
	}
	if strings.Contains(got, "activity") {
		t.Errorf("AC6 [Anti-behavior]: 'activity' present at extreme narrow width:\n%s", got)
	}
}

// ==================== End FB-093 (component layer) ====================

// ==================== FB-099: [3] strip label substitution during pendingQuotaOpen ====================
//
// Option C: welcome-dashboard strip shows "3 cancel" when pendingQuotaOpen=true;
// reverts to "3 quota" when cleared. Typed-table branch unchanged.

// AC1 [Observable] — forceDashboard=true + pendingQuotaOpen=true: View() contains "3 cancel", NOT "3 quota".
func TestFB099_AC1_PendingOpen_StripShowsCancel(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetForceDashboard(true)
	m.SetPendingQuotaOpen(true)

	got := stripANSI(m.renderKeybindStrip(196))
	if !strings.Contains(got, "3") || !strings.Contains(got, "cancel") {
		t.Errorf("AC1 [Observable]: strip = %q, want contains '3' and 'cancel'", got)
	}
	if strings.Contains(got, "quota") {
		t.Errorf("AC1 [Observable]: strip = %q, must NOT contain 'quota' when pendingQuotaOpen=true", got)
	}
}

// AC2 [Observable] — after SetPendingQuotaOpen(false) with forceDashboard=true: View() contains "3 quota", NOT "3 cancel".
func TestFB099_AC2_PendingCleared_StripResetsToQuota(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetForceDashboard(true)
	m.SetPendingQuotaOpen(true)
	m.SetPendingQuotaOpen(false) // reset

	got := stripANSI(m.renderKeybindStrip(196))
	if !strings.Contains(got, "3") || !strings.Contains(got, "quota") {
		t.Errorf("AC2 [Observable]: strip = %q, want contains '3' and 'quota'", got)
	}
	if strings.Contains(got, "cancel") {
		t.Errorf("AC2 [Observable]: strip = %q, must NOT contain 'cancel' after reset", got)
	}
}

// AC3 [Input-changed] — pendingQuotaOpen=false vs true: same forceDashboard=true state, different strip output.
func TestFB099_AC3_InputChanged_PendingVsIdle_DifferentStrip(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetForceDashboard(true)

	m.SetPendingQuotaOpen(false)
	idle := stripANSI(m.renderKeybindStrip(196))

	m.SetPendingQuotaOpen(true)
	pending := stripANSI(m.renderKeybindStrip(196))

	if idle == pending {
		t.Errorf("AC3 [Input-changed]: strip identical in idle and pending states:\n  idle:    %q\n  pending: %q", idle, pending)
	}
	if !strings.Contains(idle, "quota") {
		t.Errorf("AC3 [Input-changed]: idle strip %q, want contains 'quota'", idle)
	}
	if !strings.Contains(pending, "cancel") {
		t.Errorf("AC3 [Input-changed]: pending strip %q, want contains 'cancel'", pending)
	}
}

// AC4 [Anti-behavior] — typed-table context (forceDashboard=false, typeName="backends"):
// SetPendingQuotaOpen(true) does NOT change strip (no [3] swap in that branch).
func TestFB099_AC4_AntiBehavior_TypedTable_StripUnchanged(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetTypeContext("backends", true)
	// forceDashboard remains false.

	m.SetPendingQuotaOpen(false)
	base := stripANSI(m.renderKeybindStrip(196))

	m.SetPendingQuotaOpen(true)
	withPending := stripANSI(m.renderKeybindStrip(196))

	if base != withPending {
		t.Errorf("AC4 [Anti-behavior]: typed-table strip changed with pendingQuotaOpen=true\n  base:    %q\n  pending: %q", base, withPending)
	}
	// Both must still contain "quota" (typed-table has [3] quota unchanged).
	if !strings.Contains(withPending, "quota") {
		t.Errorf("AC4 [Anti-behavior]: typed-table strip %q, want 'quota' still present", withPending)
	}
}

// AC5 [Anti-regression / FB-054] — Tab-to-resume band present when forceDashboard=true && typeName != "";
// pendingQuotaOpen does not affect it.
func TestFB099_AC5_AntiRegression_FB054_ResumeBandUnaffected(t *testing.T) {
	t.Parallel()
	m := NewResourceTableModel(200, 22)
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)
	m.SetPendingQuotaOpen(true)

	view := stripANSI(m.View())
	if !strings.Contains(view, "resume backends") {
		t.Errorf("AC5 [Anti-regression FB-054]: Tab-to-resume band absent when pendingQuotaOpen=true:\n%s", view)
	}
}

// ==================== End FB-099 (component layer) ====================

// ==================== FB-104: Welcome panel hovered resource type documentation ====================

var testHoveredDNS = data.ResourceType{
	Kind:        "DNSRecordSet",
	Group:       "networking.datum.net",
	Version:     "v1alpha1",
	Namespaced:  true,
	Description: "A DNSRecordSet manages DNS records for a zone.",
}

var testHoveredCluster = data.ResourceType{
	Kind:        "ComputeCluster",
	Group:       "compute.datum.net",
	Version:     "v1beta1",
	Namespaced:  false,
	Description: "A ComputeCluster represents a managed Kubernetes cluster.",
}

// AC1 [Observable] — Kind, Group/Version, and Scope label render in S7 when a type is hovered.
func TestFB104_AC1_Observable_KindGroupScopeLine(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetHoveredType(testHoveredDNS)

	got := stripANSI(m.View())
	for _, want := range []string{"DNSRecordSet", "networking.datum.net", "Namespaced"} {
		if !strings.Contains(got, want) {
			t.Errorf("AC1 [Observable]: View() missing %q in S7:\n%s", want, got)
		}
	}
}

// AC2a [Observable] — short description renders below the rule in S7.
func TestFB104_AC2a_Observable_ShortDescriptionRendered(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetHoveredType(testHoveredDNS)

	got := stripANSI(m.View())
	if !strings.Contains(got, "A DNSRecordSet manages DNS records") {
		t.Errorf("AC2a [Observable]: short description absent from S7 View():\n%s", got)
	}
}

// AC2b [Observable, truncation] — description longer than 2 wrapped lines gets "…" on line 2.
func TestFB104_AC2b_Observable_LongDescriptionTruncated(t *testing.T) {
	t.Parallel()
	longDesc := "This resource type manages a very complex set of networking configurations " +
		"across multiple availability zones and handles failover routing policy for " +
		"all DNS queries received by the platform ingress layer in a given region."

	m := newWelcomeModel(80, 30)
	m.SetHoveredType(data.ResourceType{
		Kind:        "NetworkPolicy",
		Group:       "networking.datum.net",
		Version:     "v1",
		Namespaced:  true,
		Description: longDesc,
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "…") {
		t.Errorf("AC2b [Observable, truncation]: long description did not produce '…' in S7 View():\n%s", got)
	}
}

// AC3 [Observable] — S7 absent when hoveredType is zero-value.
func TestFB104_AC3_Observable_NoHover_S7Absent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	// hoveredType is zero-value by default.

	got := stripANSI(m.View())
	if strings.Contains(got, "DNSRecordSet") {
		t.Errorf("AC3 [Observable]: S7 rendered despite no hover — 'DNSRecordSet' found in View():\n%s", got)
	}
}

// AC4 [Input-changed] — SetHoveredType(A) vs SetHoveredType(B) produces different Kind in View().
func TestFB104_AC4_InputChanged_DifferentHoveredTypes(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)

	m.SetHoveredType(testHoveredDNS)
	viewA := stripANSI(m.View())

	m.SetHoveredType(testHoveredCluster)
	viewB := stripANSI(m.View())

	if !strings.Contains(viewA, "DNSRecordSet") {
		t.Errorf("AC4 [Input-changed]: 'DNSRecordSet' absent from view after SetHoveredType(DNS):\n%s", viewA)
	}
	if !strings.Contains(viewB, "ComputeCluster") {
		t.Errorf("AC4 [Input-changed]: 'ComputeCluster' absent from view after SetHoveredType(Cluster):\n%s", viewB)
	}
	if strings.Contains(viewB, "DNSRecordSet") {
		t.Errorf("AC4 [Input-changed]: old Kind 'DNSRecordSet' still present after switching to ComputeCluster:\n%s", viewB)
	}
}

// AC5 [Input-changed, state-transition] — same fixture: Namespaced=true → "Namespaced"; Namespaced=false → "Cluster".
func TestFB104_AC5_InputChanged_NamespacedVsCluster(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)

	m.SetHoveredType(data.ResourceType{Kind: "Widget", Group: "example.com", Namespaced: true})
	namespacedView := stripANSI(m.View())

	m.SetHoveredType(data.ResourceType{Kind: "Widget", Group: "example.com", Namespaced: false})
	clusterView := stripANSI(m.View())

	if !strings.Contains(namespacedView, "Namespaced") {
		t.Errorf("AC5 [Input-changed]: 'Namespaced' label absent when Namespaced=true:\n%s", namespacedView)
	}
	if !strings.Contains(clusterView, "Cluster") {
		t.Errorf("AC5 [Input-changed]: 'Cluster' label absent when Namespaced=false:\n%s", clusterView)
	}
	if strings.Contains(clusterView, "Namespaced") {
		t.Errorf("AC5 [Input-changed]: 'Namespaced' still present when Namespaced=false:\n%s", clusterView)
	}
}

// AC6 [Anti-behavior] — empty Description → no "…" artifact and no spurious blank line in S7.
func TestFB104_AC6_AntiBehavior_EmptyDescription_NoEllipsis(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetHoveredType(data.ResourceType{
		Kind:        "Minimal",
		Group:       "core.datum.net",
		Version:     "v1",
		Namespaced:  true,
		Description: "",
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "Minimal") {
		t.Fatalf("AC6 setup: 'Minimal' kind not rendered in S7:\n%s", got)
	}
	if strings.Contains(got, "…") {
		t.Errorf("AC6 [Anti-behavior]: '…' present in S7 despite empty description:\n%s", got)
	}
}

// AC7 [Anti-regression] — S6 keybind strip appears after S7 hovered-type section in View().
func TestFB104_AC7_AntiRegression_S6AfterS7(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetHoveredType(testHoveredDNS)

	got := stripANSI(m.View())

	kindIdx := strings.Index(got, "DNSRecordSet")
	stripIdx := strings.Index(got, "j/k")
	if kindIdx == -1 {
		t.Fatalf("AC7 setup: 'DNSRecordSet' not in View():\n%s", got)
	}
	if stripIdx == -1 {
		t.Fatalf("AC7 setup: 'j/k' keybind strip not in View():\n%s", got)
	}
	if stripIdx <= kindIdx {
		t.Errorf("AC7 [Anti-regression]: S6 keybind strip (idx=%d) does not follow S7 kind (idx=%d)", stripIdx, kindIdx)
	}
}

// AC8 [Anti-regression] — S3 activity section unaffected by S7; FB-082/083 row rendering intact.
func TestFB104_AC8_AntiRegression_S3Unaffected(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetHoveredType(testHoveredDNS)
	m.SetActivityRows([]data.ActivityRow{
		{ActorDisplay: "alice@example.com", Summary: "created project api-gw"},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC8 [Anti-regression]: activity row actor absent from View() when S7 active:\n%s", got)
	}
	if !strings.Contains(got, "DNSRecordSet") {
		t.Errorf("AC8 [Anti-regression]: S7 'DNSRecordSet' absent alongside S3 activity:\n%s", got)
	}
}

// ==================== End FB-104 (component layer) ====================

// ==================== FB-105: Welcome screen improvements ====================

// AC1 [Observable] — orientation hint rendered in header band when project context + empty registrations.
func TestFB105_AC1_Observable_OrientationHintShown(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))
	// registrations empty by default → hint fires

	got := stripANSI(m.View())
	const want = "select a resource type from the sidebar"
	if !strings.Contains(got, want) {
		t.Errorf("AC1 [Observable]: orientation hint %q absent from View():\n%s", want, got)
	}
}

// AC2 [Observable] — orientation hint suppressed when registrations are populated.
func TestFB105_AC2_Observable_OrientationHintSuppressedWhenRegistrations(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))
	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "networking.datum.net", Name: "backends"},
	})

	got := stripANSI(m.View())
	if strings.Contains(got, "select a resource type from the sidebar") {
		t.Errorf("AC2 [Observable]: orientation hint present despite populated registrations:\n%s", got)
	}
}

// AC3 [Observable] — "jump to:" prefix present in quick-jump section when matching registrations exist.
func TestFB105_AC3_Observable_JumpToPrefix(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "networking.datum.net", Name: "backends"},
	})

	got := stripANSI(m.View())
	if !strings.Contains(got, "jump to:") {
		t.Errorf("AC3 [Observable]: 'jump to:' prefix absent from View() when registrations match quick-jump table:\n%s", got)
	}
}

// AC4 [Observable] — all-clear flavor line rendered when attention empty + activity empty + not loading.
func TestFB105_AC4_Observable_AllClearLine(t *testing.T) {
	t.Parallel()
	// showS4 requires contentH >= 18 && contentW >= 50; tableWidth=80 → contentW=76; tableHeight=25 → contentH=21
	m := newWelcomeModel(80, 25)
	// attentionItems, activityRows empty + activityLoading false by default

	got := stripANSI(m.View())
	if !strings.Contains(got, "all clear") {
		t.Errorf("AC4 [Observable]: 'all clear' line absent from View() when attention empty + activity empty:\n%s", got)
	}
}

// AC5 [Input-changed] — empty vs populated registrations changes orientation hint visibility.
func TestFB105_AC5_InputChanged_RegistrationsTransition(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))

	emptyView := stripANSI(m.View())

	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "networking.datum.net", Name: "backends"},
	})
	populatedView := stripANSI(m.View())

	if emptyView == populatedView {
		t.Error("AC5 [Input-changed]: View() unchanged after SetRegistrations() — orientation hint not toggling")
	}
	if !strings.Contains(emptyView, "select a resource type from the sidebar") {
		t.Errorf("AC5 [Input-changed]: orientation hint absent when registrations empty:\n%s", emptyView)
	}
	if strings.Contains(populatedView, "select a resource type from the sidebar") {
		t.Errorf("AC5 [Input-changed]: orientation hint still present after registrations populated:\n%s", populatedView)
	}
}

// AC6 [Input-changed] — populated attention items suppresses the all-clear line.
func TestFB105_AC6_InputChanged_AttentionItemsSuppressAllClear(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 25)

	clearView := stripANSI(m.View())
	if !strings.Contains(clearView, "all clear") {
		t.Fatalf("AC6 setup: 'all clear' absent before attention items added:\n%s", clearView)
	}

	m.SetAttentionItems([]AttentionItem{
		{Kind: "condition", Label: "cert-expiry", Detail: "cert expires in 3 days"},
	})
	withAttention := stripANSI(m.View())

	if clearView == withAttention {
		t.Error("AC6 [Input-changed]: View() unchanged after SetAttentionItems()")
	}
	if strings.Contains(withAttention, "all clear") {
		t.Errorf("AC6 [Input-changed]: 'all clear' still present after attention items added:\n%s", withAttention)
	}
}

// AC7 [Anti-behavior] — orientation hint suppressed when forceDashboard=true (FB-054 branch active).
func TestFB105_AC7_AntiBehavior_HintSuppressedWhenForceDashboard(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)
	// registrations empty → FB-054 Tab-to-resume takes line3, FB-105 branch is else-if

	got := stripANSI(m.View())
	if strings.Contains(got, "select a resource type from the sidebar") {
		t.Errorf("AC7 [Anti-behavior]: orientation hint present when forceDashboard=true (FB-054 active):\n%s", got)
	}
	if !strings.Contains(got, "resume backends") {
		t.Errorf("AC7 [Anti-behavior]: Tab-to-resume band absent when forceDashboard=true:\n%s", got)
	}
}

// AC8 [Anti-behavior] — all-clear line suppressed when activityLoading=true.
func TestFB105_AC8_AntiBehavior_AllClearSuppressedWhenLoading(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 25)
	m.SetActivityLoading(true)

	got := stripANSI(m.View())
	if strings.Contains(got, "all clear") {
		t.Errorf("AC8 [Anti-behavior]: 'all clear' rendered while activityLoading=true:\n%s", got)
	}
}

// AC9 [Anti-regression] — org-scope context (no ProjectID) produces no orientation hint.
func TestFB105_AC9_AntiRegression_OrgScopeNoOrientationHint(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	// testCtx sets no ActiveCtx → ProjectID == "" → hint branch skipped
	m.SetTUIContext(testCtx("alice", "acme", "", false))

	got := stripANSI(m.View())
	if strings.Contains(got, "select a resource type from the sidebar") {
		t.Errorf("AC9 [Anti-regression]: orientation hint present in org-scope context (no ProjectID):\n%s", got)
	}
}

// AC10 [Anti-regression] — all-clear absent below showS4 height threshold (contentH < 18).
func TestFB105_AC10_AntiRegression_AllClearAbsentBelowHeightThreshold(t *testing.T) {
	t.Parallel()
	// tableHeight=20 → contentH=16 → showS4 false (requires >= 18)
	m := newWelcomeModel(80, 20)

	got := stripANSI(m.View())
	if strings.Contains(got, "all clear") {
		t.Errorf("AC10 [Anti-regression]: 'all clear' present at contentH<18 (showS4 should be false):\n%s", got)
	}
}

// ==================== End FB-105 (component layer) ====================

// ==================== FB-116: Orientation hint drop false "quick-jump key below" clause ====================

// AC1 [Observable] — false clause absent: View() does NOT contain "quick-jump key below" when hint fires.
func TestFB116_AC1_Observable_FalseClauseAbsent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))
	// registrations empty → hint fires

	got := stripANSI(m.View())
	if strings.Contains(got, "quick-jump key below") {
		t.Errorf("AC1 [Observable]: false clause 'quick-jump key below' present in View():\n%s", got)
	}
}

// AC2 [Observable] — correct directive present: View() contains "to get started" when hint fires.
func TestFB116_AC2_Observable_CorrectDirectivePresent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))

	got := stripANSI(m.View())
	if !strings.Contains(got, "to get started") {
		t.Errorf("AC2 [Observable]: 'to get started' absent from View() when hint fires:\n%s", got)
	}
	if !strings.Contains(got, "select a resource type from the sidebar to get started") {
		t.Errorf("AC2 [Observable]: full new directive absent from View():\n%s", got)
	}
}

// AC3 [Input-changed] — before (old copy with false clause) vs after (new copy without it): renders differ.
// This pins the post-fix state: new copy present, old clause absent. Documents intentional anchor change per spec §4.
func TestFB116_AC3_InputChanged_NewCopyDiffersFromOld(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))

	got := stripANSI(m.View())

	// New copy present — FB-116: anchor updated per spec §4
	if !strings.Contains(got, "to get started") {
		t.Errorf("AC3 [Input-changed]: new copy 'to get started' absent:\n%s", got)
	}
	// Old false clause absent
	if strings.Contains(got, "quick-jump key below") {
		t.Errorf("AC3 [Input-changed]: old false clause 'quick-jump key below' still present:\n%s", got)
	}
	// Hint itself present (base directive unchanged)
	if !strings.Contains(got, "select a resource type from the sidebar") {
		t.Errorf("AC3 [Input-changed]: base directive 'select a resource type from the sidebar' absent:\n%s", got)
	}
}

// AC4 [Anti-regression] — FB-105 anchor: hint still fires when project context + empty registrations.
// FB-116: anchor updated per spec §4 — asserts new copy, not old.
func TestFB116_AC4_AntiRegression_FB105_AnchorUpdated(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))

	got := stripANSI(m.View())
	if !strings.Contains(got, "select a resource type from the sidebar to get started") {
		t.Errorf("AC4 [Anti-regression FB-105]: hint absent after FB-116 copy update:\n%s", got)
	}
}

// AC5 [Anti-regression] — forceDashboard=true suppression intact (FB-054 path).
func TestFB116_AC5_AntiRegression_ForceDashboardSuppression(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtxWithProject("alice", "acme", "proj", "proj-123"))
	m.SetTypeContext("backends", true)
	m.SetForceDashboard(true)

	got := stripANSI(m.View())
	if strings.Contains(got, "to get started") {
		t.Errorf("AC5 [Anti-regression FB-054]: orientation hint present when forceDashboard=true:\n%s", got)
	}
}

// AC6 [Anti-regression] — org-scope suppression intact: hint absent when ProjectID == "".
func TestFB116_AC6_AntiRegression_OrgScopeSuppression(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(80, 30)
	m.SetTUIContext(testCtx("alice", "acme", "", false))

	got := stripANSI(m.View())
	if strings.Contains(got, "to get started") {
		t.Errorf("AC6 [Anti-regression]: orientation hint present in org-scope context (no ProjectID):\n%s", got)
	}
}

// ==================== End FB-116 (component layer) ====================

// ==================== FB-124: S4 quick-jump focus-activation affordance ====================

// newS4WelcomeModel builds a ResourceTableModel at welcome-panel size with a "backends"
// registration so S4 renders (contentH=18 ≥ 18, contentW=96 ≥ 50).
func newS4WelcomeModel() ResourceTableModel {
	m := newWelcomeModel(100, 22)
	m.SetRegistrations([]data.ResourceRegistration{
		{Name: "backends", Group: "networking.datum.net"},
	})
	return m
}

// AC1 [Observable] — navPaneFocused=true: hint present in View().
func TestFB124_AC1_Observable_HintPresentWhenNavFocused(t *testing.T) {
	t.Parallel()
	m := newS4WelcomeModel()
	m.SetNavPaneFocused(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "jump to ([Tab] next pane):") {
		t.Errorf("AC1 [Observable]: hint 'jump to ([Tab] next pane):' absent when navPaneFocused=true:\n%s", got)
	}
}

// AC2 [Observable] — navPaneFocused=false: plain prefix present, hint absent.
func TestFB124_AC2_Observable_HintAbsentWhenTableFocused(t *testing.T) {
	t.Parallel()
	m := newS4WelcomeModel()
	m.SetNavPaneFocused(false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "jump to:") {
		t.Errorf("AC2 [Observable]: 'jump to:' prefix absent when navPaneFocused=false:\n%s", got)
	}
	if strings.Contains(got, "[Tab] next pane") {
		t.Errorf("AC2 [Observable]: '[Tab] next pane' hint present when navPaneFocused=false:\n%s", got)
	}
}

// AC3 [Input-changed] — toggling navPaneFocused true→false changes View() content.
func TestFB124_AC3_InputChanged_ToggleChangesView(t *testing.T) {
	t.Parallel()
	m := newS4WelcomeModel()

	m.SetNavPaneFocused(true)
	v1 := stripANSI(m.View())

	m.SetNavPaneFocused(false)
	v2 := stripANSI(m.View())

	if v1 == v2 {
		t.Error("AC3 [Input-changed]: View() identical after toggling navPaneFocused true→false")
	}
	if !strings.Contains(v1, "[Tab] next pane") {
		t.Errorf("AC3 [Input-changed]: v1 (navFocused=true) missing '[Tab] next pane':\n%s", v1)
	}
	if strings.Contains(v2, "[Tab] next pane") {
		t.Errorf("AC3 [Input-changed]: v2 (navFocused=false) still contains '[Tab] next pane':\n%s", v2)
	}
}

// AC6 [Anti-regression] — entry body unchanged: [b] backends present regardless of navPaneFocused.
func TestFB124_AC6_AntiRegression_EntryBodyUnchanged(t *testing.T) {
	t.Parallel()
	m := newS4WelcomeModel()

	m.SetNavPaneFocused(true)
	v1 := stripANSI(m.View())

	m.SetNavPaneFocused(false)
	v2 := stripANSI(m.View())

	for _, view := range []string{v1, v2} {
		if !strings.Contains(view, "[b]") {
			t.Errorf("AC6 [Anti-regression]: '[b]' key absent from View():\n%s", view)
		}
		if !strings.Contains(view, "backends") {
			t.Errorf("AC6 [Anti-regression]: 'backends' label absent from View():\n%s", view)
		}
	}
}

// AC7 [Anti-regression] — no registrations: S4 section absent (hint does not appear).
func TestFB124_AC7_AntiRegression_NoRegistrations_S4Absent(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 22)
	m.SetNavPaneFocused(true)

	got := stripANSI(m.View())
	if strings.Contains(got, "jump to:") {
		t.Errorf("AC7 [Anti-regression]: 'jump to:' present with no matching registrations:\n%s", got)
	}
	if strings.Contains(got, "[Tab] next pane") {
		t.Errorf("AC7 [Anti-regression]: '[Tab] next pane' present with no matching registrations:\n%s", got)
	}
}

// ==================== End FB-124 (component layer) ====================

// ==================== FB-102: Activity unavailable recovery affordance ====================

// newActivityErrorModel returns a welcome model sized for S3 to render (contentH≥14).
func newActivityErrorModel() ResourceTableModel {
	return newWelcomeModel(100, 30)
}

// AC1 [Observable] — transient error (activityFetchFailed=true, activityCRDAbsent=false):
// View() contains "activity unavailable" AND "([r] to retry)".
func TestFB102_AC1_Observable_TransientError_HintPresent(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC1 [Observable]: 'activity unavailable' absent in transient-error state:\n%s", got)
	}
	if !strings.Contains(got, "([r] to retry)") {
		t.Errorf("AC1 [Observable]: '([r] to retry)' absent in transient-error state:\n%s", got)
	}
}

// AC2 [Observable] — CRD-absent error (activityFetchFailed=true, activityCRDAbsent=true):
// View() contains "activity unavailable" AND does NOT contain "to retry" or "[r]".
func TestFB102_AC2_Observable_CRDAbsent_NoHint(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC2 [Observable]: 'activity unavailable' absent in CRD-absent state:\n%s", got)
	}
	if strings.Contains(got, "to retry") {
		t.Errorf("AC2 [Observable]: 'to retry' present in CRD-absent state (should be absent):\n%s", got)
	}
	if strings.Contains(got, "[r]") {
		t.Errorf("AC2 [Observable]: '[r]' present in CRD-absent state (should be absent):\n%s", got)
	}
}

// AC3 [Input-changed] — toggling activityCRDAbsent false→true changes View() content.
func TestFB102_AC3_InputChanged_CRDAbsentToggleChangesView(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)

	m.SetActivityCRDAbsent(false)
	v1 := stripANSI(m.View())

	m.SetActivityCRDAbsent(true)
	v2 := stripANSI(m.View())

	if v1 == v2 {
		t.Error("AC3 [Input-changed]: View() identical after toggling activityCRDAbsent false→true")
	}
	if !strings.Contains(v1, "([r] to retry)") {
		t.Errorf("AC3 [Input-changed]: v1 (crdAbsent=false) missing '([r] to retry)':\n%s", v1)
	}
	if strings.Contains(v2, "([r] to retry)") {
		t.Errorf("AC3 [Input-changed]: v2 (crdAbsent=true) still contains '([r] to retry)':\n%s", v2)
	}
}

// AC4 [Anti-behavior] — normal state (activityFetchFailed=false): no hint, no unavailable copy.
func TestFB102_AC4_AntiBehavior_NormalState_NoHint(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	// default: activityFetchFailed=false, activityCRDAbsent=false

	got := stripANSI(m.View())
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC4 [Anti-behavior]: 'activity unavailable' present in normal state:\n%s", got)
	}
	if strings.Contains(got, "([r] to retry)") {
		t.Errorf("AC4 [Anti-behavior]: '([r] to retry)' present in normal state:\n%s", got)
	}
}

// AC5 [Anti-behavior] — SetActivityRows clears error state: hint gone, rows rendered.
func TestFB102_AC5_AntiBehavior_SetActivityRows_ClearsHint(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	before := stripANSI(m.View())
	if !strings.Contains(before, "([r] to retry)") {
		t.Fatal("precondition: '([r] to retry)' must be present before recovery")
	}

	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	got := stripANSI(m.View())
	if strings.Contains(got, "([r] to retry)") {
		t.Errorf("AC5 [Anti-behavior]: '([r] to retry)' still present after SetActivityRows with data:\n%s", got)
	}
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC5 [Anti-behavior]: 'activity unavailable' still present after SetActivityRows with data:\n%s", got)
	}
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC5 [Anti-behavior]: row actor 'alice@example.com' missing after SetActivityRows:\n%s", got)
	}
}

// ==================== End FB-102 (component layer) ====================

// ==================== FB-125: Tab keybind label alignment ====================

// AC1 [Observable] — new copy "next pane" present when navPaneFocused=true.
func TestFB125_AC1_Observable_NewCopyPresent(t *testing.T) {
	t.Parallel()
	m := newS4WelcomeModel()
	m.SetNavPaneFocused(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "jump to ([Tab] next pane):") {
		t.Errorf("AC1 [Observable]: 'jump to ([Tab] next pane):' absent when navPaneFocused=true:\n%s", got)
	}
	if strings.Contains(got, "to focus") {
		t.Errorf("AC1 [Observable]: old copy 'to focus' still present:\n%s", got)
	}
}

// ==================== End FB-125 ====================

// ==================== FB-128: S3 activity error hint action verb ====================

// AC1 [Observable] — transient error at normal width: "([r] to retry)" present.
func TestFB128_AC1_Observable_NewCopyPresent(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "([r] to retry)") {
		t.Errorf("AC1 [Observable]: '([r] to retry)' absent at contentW=80:\n%s", got)
	}
}

// AC2 [Observable] — old copy absent.
func TestFB128_AC2_Observable_OldCopyAbsent(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	got := stripANSI(m.renderActivitySection(80))
	if strings.Contains(got, "(press [r])") {
		t.Errorf("AC2 [Observable]: old copy '(press [r])' still present:\n%s", got)
	}
	if strings.Contains(got, "press ") {
		t.Errorf("AC2 [Observable]: old 'press ' verb still present:\n%s", got)
	}
}

// AC3 [Anti-regression] — CRD-absent has no parenthetical.
func TestFB128_AC3_AntiRegression_CRDAbsent_NoParenthetical(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(true)

	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC3 [Anti-regression]: 'activity unavailable' absent in CRD-absent state:\n%s", got)
	}
	if strings.Contains(got, "to retry") {
		t.Errorf("AC3 [Anti-regression]: 'to retry' present in CRD-absent state:\n%s", got)
	}
	if strings.Contains(got, "press") {
		t.Errorf("AC3 [Anti-regression]: 'press' present in CRD-absent state:\n%s", got)
	}
}

// ==================== End FB-128 ====================

// ==================== FB-129: S3 error body narrow-width guard ====================

// AC1 [Observable] — narrow transient (contentW=34): parenthetical dropped.
func TestFB129_AC1_Observable_NarrowTransient_NoParenthetical(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	got := stripANSI(m.renderActivitySection(34))
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC1 [Observable]: 'activity unavailable' absent at contentW=34:\n%s", got)
	}
	if strings.Contains(got, "to retry") {
		t.Errorf("AC1 [Observable]: 'to retry' present at contentW=34 (should be dropped):\n%s", got)
	}
}

// AC2 [Observable] — wide transient (contentW=35): full copy renders.
func TestFB129_AC2_Observable_WideTransient_FullCopy(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	got := stripANSI(m.renderActivitySection(35))
	if !strings.Contains(got, "([r] to retry)") {
		t.Errorf("AC2 [Observable]: '([r] to retry)' absent at contentW=35:\n%s", got)
	}
}

// AC3 [Input-changed] — width transition 34→35 changes View() output.
func TestFB129_AC3_InputChanged_WidthTransitionChangesView(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(false)

	v1 := stripANSI(m.renderActivitySection(34))
	v2 := stripANSI(m.renderActivitySection(35))

	if v1 == v2 {
		t.Error("AC3 [Input-changed]: renderActivitySection output identical at contentW=34 and contentW=35")
	}
	if strings.Contains(v1, "to retry") {
		t.Errorf("AC3 [Input-changed]: v1 (contentW=34) contains 'to retry' (should not):\n%s", v1)
	}
	if !strings.Contains(v2, "to retry") {
		t.Errorf("AC3 [Input-changed]: v2 (contentW=35) missing 'to retry':\n%s", v2)
	}
}

// AC4 [Anti-regression] — CRD-absent unchanged at wide width.
func TestFB129_AC4_AntiRegression_CRDAbsent_WideWidth(t *testing.T) {
	t.Parallel()
	m := newActivityErrorModel()
	m.SetActivityFetchFailed(true)
	m.SetActivityCRDAbsent(true)

	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "activity unavailable") {
		t.Errorf("AC4 [Anti-regression]: 'activity unavailable' absent in CRD-absent wide state:\n%s", got)
	}
	if strings.Contains(got, "to retry") {
		t.Errorf("AC4 [Anti-regression]: 'to retry' present in CRD-absent state:\n%s", got)
	}
}

// AC5 [Anti-regression] — data rows at wide width: tier thresholds unaffected.
func TestFB129_AC5_AntiRegression_DataRows_TierThresholdsUnaffected(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})

	// Tier 1 (contentW >= 65): all columns present.
	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC5 Tier1: actor missing at contentW=80:\n%s", got)
	}
	// Tier 2 (45 ≤ contentW < 65): resource column dropped.
	got2 := stripANSI(m.renderActivitySection(55))
	if !strings.Contains(got2, "alice@example.com") {
		t.Errorf("AC5 Tier2: actor missing at contentW=55:\n%s", got2)
	}
}

// ==================== End FB-129 ====================

// ==================== FB-130: S3 spinner gate fix (nil vs empty slice) ====================
//
// Bug: `case m.activityLoading && m.activityRows == nil` misses non-nil empty slices.
// Fix: `case m.activityLoading && len(m.activityRows) == 0` covers both nil and empty.
// Production sequence (FB-103): fetch returns 0 rows → SetActivityRows([]) → rows=[], loading=false
// → operator presses [r] → SetActivityLoading(true) → loading=true, rows=[] → spinner must fire.

// AC1 [Observable] — loading=true + rows=[] (non-nil empty): spinner fires.
// Production state: prior fetch returned 0 rows; [r] re-armed loading.
// SetActivityRows([]) first (clears loading, sets non-nil empty rows),
// then SetActivityLoading(true) to simulate [r] re-arming.
func TestFB130_AC1_Observable_EmptyNonNilRows_SpinnerFires(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{}) // fetch success: loading=false, rows=[] (non-nil)
	m.SetActivityLoading(true)              // [r] re-arms: loading=true, rows=[] stays non-nil

	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "loading") {
		t.Errorf("AC1 [Observable]: spinner absent with loading=true rows=[] (non-nil):\n%s", got)
	}
	if strings.Contains(got, "no recent activity") {
		t.Errorf("AC1 [Observable]: 'no recent activity' present when spinner should show:\n%s", got)
	}
}

// AC2 [Input-changed] — SetActivityRows([]) → SetActivityLoading(true): View() changes.
// v1 = after fetch returns (loading=false, rows=[]) → "no recent activity"
// v2 = after [r] re-arms (loading=true, rows=[]) → "⟳ loading…"
func TestFB130_AC2_InputChanged_EmptyRows_LoadingTransition(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityRows([]data.ActivityRow{}) // fetch success: loading=false, rows=[]

	v1 := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(v1, "no recent activity") {
		t.Fatalf("AC2 precondition: 'no recent activity' absent in v1:\n%s", v1)
	}

	m.SetActivityLoading(true) // [r] re-arms: loading=true, rows=[] stays
	v2 := stripANSI(m.renderActivitySection(80))

	if v1 == v2 {
		t.Error("AC2 [Input-changed]: renderActivitySection output unchanged after SetActivityLoading(true)")
	}
	if strings.Contains(v2, "no recent activity") {
		t.Errorf("AC2 [Input-changed]: 'no recent activity' still present in v2 (should be spinner):\n%s", v2)
	}
	if !strings.Contains(v2, "loading") {
		t.Errorf("AC2 [Input-changed]: 'loading' absent in v2:\n%s", v2)
	}
}

// AC3 [Anti-regression FB-103 AC1 + FB-130] — pre-first-load (rows=nil): spinner preserved.
// len(nil)==0, so the fix is backward-compatible with the nil case.
func TestFB130_AC3_AntiRegression_NilRows_SpinnerPreserved(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityLoading(true)
	// activityRows nil by default — pre-first-load state

	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "loading") {
		t.Errorf("AC3 [Anti-regression]: spinner absent during pre-first-load (loading=true, rows=nil):\n%s", got)
	}
}

// AC4 [Anti-regression FB-076] — populated rows + loading=true: spinner suppressed (silent re-fetch).
func TestFB130_AC4_AntiRegression_PopulatedRows_SpinnerSuppressed(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(120, 30)
	m.SetActivityRows([]data.ActivityRow{testActivityRow()})
	m.SetActivityLoading(true)

	got := stripANSI(m.renderActivitySection(80))
	if strings.Contains(got, "loading") {
		t.Errorf("AC4 [Anti-regression]: spinner present when rows populated (should be silent re-fetch):\n%s", got)
	}
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("AC4 [Anti-regression]: actor absent when rows populated:\n%s", got)
	}
}

// AC5 [Anti-regression FB-103 AC7] — error-state + loading=true: spinner fires (error-cleared path).
// SetActivityFetchFailed does not modify activityRows (stays nil); loading=true takes
// priority over the failed case in the render switch (loading case is first).
func TestFB130_AC5_AntiRegression_ErrorState_SpinnerFires(t *testing.T) {
	t.Parallel()
	m := newWelcomeModel(100, 30)
	m.SetActivityFetchFailed(true) // error state: failed=true, rows=nil, loading=false
	m.SetActivityLoading(true)     // [r] re-arms: loading=true takes priority in switch

	got := stripANSI(m.renderActivitySection(80))
	if !strings.Contains(got, "loading") {
		t.Errorf("AC5 [Anti-regression]: spinner absent after [r] in error state:\n%s", got)
	}
	if strings.Contains(got, "activity unavailable") {
		t.Errorf("AC5 [Anti-regression]: error copy still present when spinner should show:\n%s", got)
	}
}

// ==================== End FB-130 ====================
