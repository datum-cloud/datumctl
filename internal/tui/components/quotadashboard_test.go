package components

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/tui/data"
)

// --- helpers ---

func projBucket(name, rt string, allocated, limit int64) data.AllowanceBucket {
	return data.AllowanceBucket{
		Name: name, ConsumerKind: "project",
		ResourceType: rt, Allocated: allocated, Limit: limit,
	}
}

func orgBucket(name, rt string, allocated, limit int64) data.AllowanceBucket {
	return data.AllowanceBucket{
		Name: name, ConsumerKind: "org",
		ResourceType: rt, Allocated: allocated, Limit: limit,
	}
}

func newDashboard(w, h int) QuotaDashboardModel {
	return NewQuotaDashboardModel(w, h, "test-project (proj)")
}

// --- SelectedBucket ---

func TestQuotaDashboardModel_Empty_SelectedBucketReturnsFalse(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	_, ok := m.SelectedBucket()
	if ok {
		t.Error("SelectedBucket() = (_, true), want (_, false) on empty dashboard")
	}
}

func TestQuotaDashboardModel_SetBuckets_SelectedBucketHighestPctFirst(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("low", "cpus", 10, 100),   // 10%
		projBucket("high", "memory", 80, 100), // 80%
	})
	// Grouped default: project-sorted-by-%, so "high" (80%) comes first.
	b, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false, want a bucket")
	}
	if b.Name != "high" {
		t.Errorf("SelectedBucket().Name = %q, want %q (highest %% should be first)", b.Name, "high")
	}
}

// --- cursor navigation ---

func TestQuotaDashboardModel_CursorNavigation(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("first", "cpus", 80, 100),   // 80% → sorted index 0
		projBucket("second", "memory", 20, 100), // 20% → sorted index 1
	})

	b0, _ := m.SelectedBucket()
	if b0.Name != "first" {
		t.Fatalf("initial cursor: SelectedBucket = %q, want %q", b0.Name, "first")
	}

	m.moveCursor(1)
	b1, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false after move down")
	}
	if b1.Name != "second" {
		t.Errorf("after j: SelectedBucket = %q, want %q", b1.Name, "second")
	}

	// clamp at end
	m.moveCursor(1)
	bEnd, _ := m.SelectedBucket()
	if bEnd.Name != "second" {
		t.Errorf("cursor clamped at end: SelectedBucket = %q, want %q", bEnd.Name, "second")
	}

	// move back
	m.moveCursor(-1)
	bBack, _ := m.SelectedBucket()
	if bBack.Name != "first" {
		t.Errorf("after k: SelectedBucket = %q, want %q", bBack.Name, "first")
	}

	// clamp at start
	m.moveCursor(-1)
	bStart, _ := m.SelectedBucket()
	if bStart.Name != "first" {
		t.Errorf("cursor clamped at start: SelectedBucket = %q, want %q", bStart.Name, "first")
	}
}

// --- grouped ordering ---

func TestQuotaDashboardModel_GroupedOrdering(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	// FB-013: grouped mode is now tree-aware per-resource-type. Different resource
	// types = standalone single-bucket groups sorted by max-child-pct then parent-pct.
	// No project/org divider in tree-aware mode (spec §4a: divider is now unused).
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("p-low", "cpus", 10, 100),   // proj 10%
		projBucket("p-high", "memory", 80, 100), // proj 80%
		orgBucket("o-high", "storage", 90, 100), // org  90%
		orgBucket("o-low", "network", 5, 100),   // org   5%
	})

	items := m.orderedItems()

	var names []string
	for _, item := range items {
		if item.isDivider || item.isGroupHeader || item.isSiblingConsume {
			continue
		}
		names = append(names, item.bucket.Name)
	}

	if len(names) != 4 {
		t.Fatalf("grouped: got %d buckets, want 4", len(names))
	}
	// Groups sorted by max-child-pct desc, then parent-pct desc.
	// p-high: child pct 80% → first; p-low: child pct 10% → second;
	// o-high: no child, parent 90% → third; o-low: no child, parent 5% → fourth.
	if names[0] != "p-high" {
		t.Errorf("grouped[0] = %q, want p-high", names[0])
	}
	if names[1] != "p-low" {
		t.Errorf("grouped[1] = %q, want p-low", names[1])
	}
	if names[2] != "o-high" {
		t.Errorf("grouped[2] = %q, want o-high", names[2])
	}
	if names[3] != "o-low" {
		t.Errorf("grouped[3] = %q, want o-low", names[3])
	}
}

func TestQuotaDashboardModel_GroupedOrdering_NoDividerWhenOnlyProject(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("a", "cpus", 10, 100),
		projBucket("b", "memory", 50, 100),
	})
	for _, item := range m.orderedItems() {
		if item.isDivider {
			t.Error("grouped with only project buckets: unexpected divider")
		}
	}
}

// --- flat ordering ---

func TestQuotaDashboardModel_FlatOrdering(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("p-low", "cpus", 10, 100),    // 10%
		projBucket("p-high", "memory", 80, 100),  // 80%
		orgBucket("o-top", "storage", 90, 100),   // 90%
		orgBucket("o-low", "network", 5, 100),    // 5%
	})
	m.grouping = GroupingFlat

	items := m.orderedItems()
	// Flat sorted by % desc: o-top (90%), p-high (80%), p-low (10%), o-low (5%)
	want := []string{"o-top", "p-high", "p-low", "o-low"}
	if len(items) != len(want) {
		t.Fatalf("flat: got %d items, want %d", len(items), len(want))
	}
	for i, item := range items {
		if item.isDivider {
			t.Errorf("flat: unexpected divider at index %d", i)
			continue
		}
		if item.bucket.Name != want[i] {
			t.Errorf("flat[%d] = %q, want %q", i, item.bucket.Name, want[i])
		}
	}
}

// --- toggle grouping ---

func TestQuotaDashboardModel_ToggleGrouping_ChangesModeAndPreservesCursor(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	// Grouped order: p-high (80%), p-low (10%), o-top (0% child/90% parent), o-low (5%)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("p-low", "cpus", 10, 100),
		projBucket("p-high", "memory", 80, 100),
		orgBucket("o-top", "storage", 90, 100),
		orgBucket("o-low", "network", 5, 100),
	})
	// Navigate cursor to p-low (grouped index 1).
	m.moveCursor(1)
	b, _ := m.SelectedBucket()
	if b.Name != "p-low" {
		t.Fatalf("pre-toggle: SelectedBucket = %q, want p-low", b.Name)
	}

	m.ToggleGrouping()
	if m.grouping != GroupingFlat {
		t.Error("ToggleGrouping: expected GroupingFlat")
	}

	// Flat order: o-top (90%), p-high (80%), p-low (10%), o-low (5%)
	// Cursor should follow "p-low" to flat index 2.
	bAfter, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false after toggle")
	}
	if bAfter.Name != "p-low" {
		t.Errorf("after toggle to flat: SelectedBucket = %q, want p-low (cursor follows name)", bAfter.Name)
	}

	// Toggle back to grouped — cursor should follow p-low back to grouped index 1.
	m.ToggleGrouping()
	if m.grouping != GroupingGrouped {
		t.Error("ToggleGrouping: expected GroupingGrouped after second toggle")
	}
	bGrouped, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false after toggle back to grouped")
	}
	if bGrouped.Name != "p-low" {
		t.Errorf("after toggle back to grouped: SelectedBucket = %q, want p-low", bGrouped.Name)
	}
}

// --- filter ---

func TestQuotaDashboardModel_SetFilter_MatchesCaseInsensitive(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "compute/cpus", 10, 100),
		projBucket("b2", "storage/bytes", 50, 100),
		projBucket("b3", "COMPUTE/memory", 80, 100),
	})

	m.SetFilter("compute")
	filtered := m.filteredBuckets()
	if len(filtered) != 2 {
		t.Fatalf("filter 'compute': got %d buckets, want 2", len(filtered))
	}
	names := map[string]bool{}
	for _, b := range filtered {
		names[b.Name] = true
	}
	if !names["b1"] || !names["b3"] {
		t.Errorf("filter 'compute': expected b1 and b3, got %v", names)
	}
}

func TestQuotaDashboardModel_SetFilter_EmptyClearsFilter(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "cpus", 10, 100),
		projBucket("b2", "memory", 50, 100),
	})
	m.SetFilter("cpus")
	if len(m.filteredBuckets()) != 1 {
		t.Fatalf("filter 'cpus': want 1 bucket")
	}
	m.SetFilter("")
	if len(m.filteredBuckets()) != 2 {
		t.Errorf("clear filter: got %d buckets, want 2", len(m.filteredBuckets()))
	}
}

func TestQuotaDashboardModel_SetFilter_ResetsCursor(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "cpus", 80, 100),
		projBucket("b2", "memory", 50, 100),
		projBucket("b3", "storage", 20, 100),
	})
	m.moveCursor(2)

	m.SetFilter("memory")
	if m.cursor != 0 {
		t.Errorf("SetFilter: cursor = %d, want 0 after filter reset", m.cursor)
	}
	b, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false after filter")
	}
	if b.Name != "b2" {
		t.Errorf("after filter 'memory': SelectedBucket = %q, want b2", b.Name)
	}
}

// --- cursor snapping ---

func TestQuotaDashboardModel_SetBuckets_SnapCursorToName(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b-high", "cpus", 80, 100),   // 80% → index 0
		projBucket("b-low", "memory", 20, 100),  // 20% → index 1
	})
	m.moveCursor(1)
	b, _ := m.SelectedBucket()
	if b.Name != "b-low" {
		t.Fatalf("pre-refresh: SelectedBucket = %q, want b-low", b.Name)
	}

	// Refresh: b-low now has higher % and moves to index 0.
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b-high", "cpus", 10, 100),  // now 10% → index 1
		projBucket("b-low", "memory", 90, 100), // now 90% → index 0
	})
	bAfter, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false after refresh")
	}
	if bAfter.Name != "b-low" {
		t.Errorf("after refresh: SelectedBucket = %q, want b-low (cursor should follow name)", bAfter.Name)
	}
}

func TestQuotaDashboardModel_SetBuckets_CursorClampsWhenNameGone(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "cpus", 80, 100),    // index 0
		projBucket("b2", "memory", 50, 100),  // index 1
		projBucket("b3", "storage", 10, 100), // index 2
	})
	m.moveCursor(2)
	b, _ := m.SelectedBucket()
	if b.Name != "b3" {
		t.Fatalf("pre-remove: SelectedBucket = %q, want b3", b.Name)
	}

	// Remove b3 — cursor should clamp to last (b2 at index 1).
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "cpus", 80, 100),
		projBucket("b2", "memory", 50, 100),
	})
	bAfter, ok := m.SelectedBucket()
	if !ok {
		t.Fatal("SelectedBucket() returned false after bucket removed")
	}
	if bAfter.Name != "b2" {
		t.Errorf("after remove: SelectedBucket = %q, want b2 (clamp to last)", bAfter.Name)
	}
}

// --- view states ---

func TestQuotaDashboardModel_BuildMainContent_Loading(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetLoading(true)
	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "Loading quota data") {
		t.Errorf("loading state: want 'Loading quota data' in content, got %q", got)
	}
}

func TestQuotaDashboardModel_BuildMainContent_Empty(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "No allowance buckets configured") {
		t.Errorf("empty state: want 'No allowance buckets configured' in content, got %q", got)
	}
}

func TestQuotaDashboardModel_BuildMainContent_Error(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetLoadErr(errors.New("connection refused"))
	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "Could not load allowance buckets") {
		t.Errorf("error state: want 'Could not load allowance buckets' in content, got %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("error state: want error text in content, got %q", got)
	}
}

func TestQuotaDashboardModel_BuildMainContent_ShowsBuckets(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("my-bucket", "compute/cpus", 40, 100),
	})
	got := stripANSI(m.buildMainContent())
	// ResolveDescription with nil registrations falls back to the short name (last "/" segment).
	// Short name "cpus" appears in the group header.
	if !strings.Contains(got, "cpus") {
		t.Errorf("normal state: want resource type short name in content, got %q", got)
	}
}

// --- FB-014: dashboard SetRegistrations description labels (AC#5) ---

// TestQuotaDashboardModel_BuildMainContent_WithRegistrations_ShowsDescription verifies
// that after SetRegistrations is called with a matching entry, buildMainContent()
// renders the description string instead of the short name (AC#5 — S2 Dashboard).
func TestQuotaDashboardModel_BuildMainContent_WithRegistrations_ShowsDescription(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "resourcemanager.miloapis.com/projects", 4, 10),
	})
	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "resourcemanager.miloapis.com", Name: "projects", Description: "Projects created within Organizations"},
	})

	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "Projects created within Organizations") {
		t.Errorf("buildMainContent with regs: want description in output, got %q", got)
	}
	// The fully-qualified name must not appear in place of the description.
	if strings.Contains(got, "resourcemanager.miloapis.com/projects") {
		t.Errorf("buildMainContent with regs: fully-qualified name leaked into output %q", got)
	}
}

// --- view chrome (height >= 6) ---

func TestQuotaDashboardModel_View_ShowsTitleBar(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20) // height >= 6 triggers full chrome
	got := stripANSI(m.View())
	if !strings.Contains(got, "quota usage") {
		t.Errorf("view chrome: want 'quota usage' title in view, got %q", got)
	}
}

func TestQuotaDashboardModel_View_SmallHeightSuppressesChrome(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 5) // height < 6 → no title/rules chrome
	got := stripANSI(m.View())
	if strings.Contains(got, "quota usage") {
		t.Errorf("height<6: want no 'quota usage' title, got %q", got)
	}
}

// ==================== FB-036: Remove recon age and claim count (dashboard) ====================

// TestFB036_DashboardTreeRow_NoReconSubstring — AC#4/AC#6: QuotaDashboard
// buildMainContent() must not contain "recon" after the recon cell is removed.
func TestFB036_DashboardTreeRow_NoReconSubstring(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("my-bucket", "compute/cpus", 40, 100),
		orgBucket("org-bucket", "compute/cpus", 80, 200),
	})
	got := stripANSI(m.buildMainContent())
	if strings.Contains(got, "recon") {
		t.Errorf("AC#4/AC#6: 'recon' found in dashboard content: %q", got)
	}
	if strings.Contains(got, "claims:") {
		t.Errorf("AC#4/AC#6: 'claims:' found in dashboard content: %q", got)
	}
}

// ==================== End FB-036 (dashboard) ====================

// ==================== FB-043: Consumer-legible data freshness signal ====================

// AC#1 — [Observable] TitleBar shows freshness when fetchedAt is set and width is wide.
func TestFB043_TitleBar_FreshnessAfterFetch(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetBucketFetchedAt(time.Now().Add(-5 * time.Second))

	got := stripANSI(m.titleBar())
	if !strings.Contains(got, "updated") {
		t.Errorf("AC#1: titleBar() missing 'updated' freshness signal (wide model, fetchedAt set):\n%s", got)
	}
	if !strings.Contains(got, "s ago") {
		t.Errorf("AC#1: titleBar() missing 'Xs ago' freshness value:\n%s", got)
	}
}

// [Observable] fetchedAt = time.Now() → "just now" appears.
func TestFB043_TitleBar_JustNow(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetBucketFetchedAt(time.Now())

	got := stripANSI(m.titleBar())
	if !strings.Contains(got, "just now") {
		t.Errorf("titleBar() missing 'just now' immediately after fetch:\n%s", got)
	}
}

// [Anti-behavior] Zero fetchedAt → no freshness signal in titleBar.
func TestFB043_TitleBar_ZeroFetchedAt_NoFreshness(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	// fetchedAt is zero by default — no SetBucketFetchedAt call.

	got := stripANSI(m.titleBar())
	if strings.Contains(got, "updated") {
		t.Errorf("titleBar() contains 'updated' with zero fetchedAt; want absent:\n%s", got)
	}
}

// [Anti-behavior] Gap guard: width too narrow to fit freshness → freshness absent.
func TestFB043_TitleBar_NarrowWidth_FreshnessAbsent(t *testing.T) {
	t.Parallel()
	m := newDashboard(60, 20) // hint alone is ~50 chars; no room for freshness at width=60
	m.SetBucketFetchedAt(time.Now().Add(-5 * time.Second))

	got := stripANSI(m.titleBar())
	if strings.Contains(got, "updated") {
		t.Errorf("titleBar() contains 'updated' at narrow width=60; want absent (gap guard):\n%s", got)
	}
}

// [Input-changed] Freshness string differs between a recent and an older fetchedAt.
func TestFB043_TitleBar_FreshnessChanges_BetweenFetches(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)

	m.SetBucketFetchedAt(time.Now().Add(-30 * time.Second))
	got1 := stripANSI(m.titleBar())
	if !strings.Contains(got1, "30s ago") {
		t.Errorf("fetchedAt=T-30s: want '30s ago' in titleBar(), got:\n%s", got1)
	}

	m.SetBucketFetchedAt(time.Now())
	got2 := stripANSI(m.titleBar())
	if !strings.Contains(got2, "just now") {
		t.Errorf("fetchedAt=now: want 'just now' in titleBar(), got:\n%s", got2)
	}
	if got1 == got2 {
		t.Error("titleBar() output identical before/after fetchedAt change; want content to differ")
	}
}

// ==================== End FB-043 (component) ====================

// ==================== FB-058: Sibling-data note reword ====================

// newTreeDashboard returns a QuotaDashboardModel with an org parent, a project
// child (active consumer), and one sibling project — ready for tree rendering.
func newTreeDashboard(w, h int, sibRestricted bool) QuotaDashboardModel {
	m := newDashboard(w, h)
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org-parent", ResourceType: "compute/cpus", ConsumerKind: "Organization", ConsumerName: "my-org", Limit: 200, Allocated: 50},
		{Name: "proj-child", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "my-project", Limit: 60, Allocated: 10},
		{Name: "proj-sibling", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "other-project", Limit: 60, Allocated: 30},
	})
	m.SetActiveConsumer("Project", "my-project")
	m.SetSiblingRestricted(sibRestricted)
	return m
}

// [Observable] sibRestricted tree in QuotaDashboard → new sibling note present.
func TestFB058_Dashboard_SibRestricted_HasNewSibNote(t *testing.T) {
	t.Parallel()
	m := newTreeDashboard(200, 20, true)
	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "other projects' usage hidden") {
		t.Errorf("'(other projects' usage hidden)' missing with sibRestricted=true:\n%s", got)
	}
}

// [Anti-behavior] old sibling note "(sibling data unavailable)" is absent.
func TestFB058_Dashboard_SibRestricted_OldSibNoteAbsent(t *testing.T) {
	t.Parallel()
	m := newTreeDashboard(200, 20, true)
	got := stripANSI(m.buildMainContent())
	if strings.Contains(got, "sibling data unavailable") {
		t.Errorf("stale 'sibling data unavailable' present with sibRestricted=true; want absent:\n%s", got)
	}
}

// [Observable] QuotaBannerModel with sibRestricted=true → new sibling note, old absent.
func TestFB058_QuotaBanner_SibRestricted_HasNewSibNote(t *testing.T) {
	t.Parallel()
	bm := NewQuotaBannerModel(200)
	bm.SetBuckets([]data.AllowanceBucket{
		{Name: "org-parent", ResourceType: "compute/cpus", ConsumerKind: "Organization", ConsumerName: "my-org", Limit: 200, Allocated: 50},
		{Name: "proj-child", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "my-project", Limit: 60, Allocated: 10},
		{Name: "proj-sibling", ResourceType: "compute/cpus", ConsumerKind: "Project", ConsumerName: "other-project", Limit: 60, Allocated: 30},
	})
	bm.SetActiveConsumer("Project", "my-project")
	bm.SetSiblingRestricted(true)
	got := stripANSI(bm.View())
	if !strings.Contains(got, "other projects' usage hidden") {
		t.Errorf("'(other projects' usage hidden)' missing from QuotaBannerModel.View() with sibRestricted=true:\n%s", got)
	}
	if strings.Contains(got, "sibling data unavailable") {
		t.Errorf("stale 'sibling data unavailable' present in QuotaBannerModel.View(); want absent:\n%s", got)
	}
}

// ==================== End FB-058 (component) ====================

// ==================== FB-063: r refresh on QuotaDashboard must not flash/blank previously-rendered data ====================
//
// Axis-coverage table:
// AC1 | Observable + Anti-behavior | TestFB063_AC1_Observable_BucketLabelsRetainedDuringRefresh
//     |                            | TestFB063_AC1_AntiBlank_LoadingTextAbsentDuringRefresh
// AC2 | Observable                 | TestFB063_AC2_Observable_RefreshingIndicatorInTitleBar
// AC3 | Input-changed              | TestFB063_AC3_InputChanged_RefreshingClearsAfterLoad
// AC4 | Anti-regression            | TestFB063_AC4_AntiRegression_InitialLoad_ShowsSpinner
// AC5 | Anti-regression            | TestFB063_AC5_AntiRegression_CursorPreservedAcrossSetRefreshing
// AC6 | Anti-regression            | TestFB063_AC6_AntiRegression_SetLoading_ZeroState_BlanksPaneStill
// AC7 | Integration                | go install ./... + go test ./internal/tui/...

// newBucketedDashboard returns a wide quota dashboard with two pre-loaded project buckets.
func newBucketedDashboard() QuotaDashboardModel {
	m := newDashboard(200, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("dns-zones", "networking/dnszones", 120, 200),
		projBucket("backends", "networking/backends", 40, 100),
	})
	return m
}

// [Observable] AC1a: mid-refresh, previously-loaded bucket labels still appear in View().
func TestFB063_AC1_Observable_BucketLabelsRetainedDuringRefresh(t *testing.T) {
	t.Parallel()
	m := newBucketedDashboard()
	m.SetRefreshing(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "dnszones") {
		t.Errorf("AC1: 'dnszones' missing from View() during refresh — bucket data must remain:\n%s", got)
	}
	if !strings.Contains(got, "backends") {
		t.Errorf("AC1: 'backends' missing from View() during refresh — bucket data must remain:\n%s", got)
	}
}

// [Anti-behavior] AC1b: mid-refresh, "Loading quota data" must NOT appear (pane must not blank).
func TestFB063_AC1_AntiBlank_LoadingTextAbsentDuringRefresh(t *testing.T) {
	t.Parallel()
	m := newBucketedDashboard()
	m.SetRefreshing(true)

	got := stripANSI(m.buildMainContent())
	if strings.Contains(got, "Loading quota data") {
		t.Errorf("AC1: 'Loading quota data' present during refresh — pane blanked when data already loaded:\n%s", got)
	}
}

// [Observable] AC2: mid-refresh title bar shows "⟳ refreshing…" at wide widths (≥90).
func TestFB063_AC2_Observable_RefreshingIndicatorInTitleBar(t *testing.T) {
	t.Parallel()
	m := newBucketedDashboard() // width=200
	m.SetRefreshing(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "⟳ refreshing…") {
		t.Errorf("AC2: '⟳ refreshing…' missing from View() at width=200 during refresh:\n%s", got)
	}
	// Freshness ("updated ...") must not appear simultaneously with the refreshing indicator.
	if strings.Contains(got, "updated ") {
		t.Errorf("AC2: 'updated' freshness and '⟳ refreshing…' both present — must be mutually exclusive:\n%s", got)
	}
}

// [Input-changed] AC3: after BucketsLoadedMsg sequence View() loses "⟳ refreshing…" and gains "updated".
// Pair A (before): refreshing=true → "⟳ refreshing…" present, "updated" absent.
// Pair B (after):  SetLoading(false)+SetBuckets+SetBucketFetchedAt → "updated" present, "refreshing" absent.
func TestFB063_AC3_InputChanged_RefreshingClearsAfterLoad(t *testing.T) {
	t.Parallel()
	m := newBucketedDashboard()
	m.SetRefreshing(true)

	// Pair A: before state — refreshing indicator present.
	before := stripANSI(m.View())
	if !strings.Contains(before, "⟳ refreshing…") {
		t.Fatal("AC3 precondition (pair A): '⟳ refreshing…' missing before BucketsLoadedMsg sequence")
	}
	if strings.Contains(before, "updated ") {
		t.Fatal("AC3 precondition (pair A): 'updated' present before load completes — unexpected")
	}

	// Simulate BucketsLoadedMsg: SetLoading(false) clears refreshing; SetBuckets+SetBucketFetchedAt complete update.
	m.SetLoading(false)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("dns-zones", "networking/dnszones", 150, 200),
	})
	m.SetBucketFetchedAt(time.Now())

	// Pair B: after state — freshness present, refreshing indicator gone.
	after := stripANSI(m.View())
	if strings.Contains(after, "refreshing") {
		t.Errorf("AC3 (pair B): 'refreshing' still present after load completes:\n%s", after)
	}
	if !strings.Contains(after, "updated") {
		t.Errorf("AC3 (pair B): 'updated' freshness signal missing after load completes:\n%s", after)
	}
}

// [Anti-regression] AC4: initial-load (zero-state, no prior buckets) still shows spinner in buildMainContent().
// Uses buildMainContent() because height≥6 View() uses the viewport path (spinner text is not in the viewport).
func TestFB063_AC4_AntiRegression_InitialLoad_ShowsSpinner(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20) // zero state — no SetBuckets call
	m.SetLoading(true)

	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "Loading quota data") {
		t.Errorf("AC4: 'Loading quota data' missing from buildMainContent() in zero-state initial load:\n%s", got)
	}
}

// [Anti-regression] AC5: cursor position is preserved when SetRefreshing(true) is called.
func TestFB063_AC5_AntiRegression_CursorPreservedAcrossSetRefreshing(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "cpus", 80, 100),    // sorted index 0 (highest %)
		projBucket("b2", "memory", 50, 100),  // sorted index 1
		projBucket("b3", "storage", 20, 100), // sorted index 2
	})
	m.moveCursor(2)
	if m.cursor != 2 {
		t.Fatalf("AC5 precondition: cursor = %d, want 2 after moveCursor(2)", m.cursor)
	}

	m.SetRefreshing(true)

	if m.cursor != 2 {
		t.Errorf("AC5: cursor = %d after SetRefreshing(true), want 2 (must be preserved)", m.cursor)
	}
}

// [Anti-regression] AC6: SetLoading(true) on zero-state model still blanks the pane (FB-035 guard).
// Ensures the buildMainContent loading branch was not accidentally removed by the FB-063 change.
func TestFB063_AC6_AntiRegression_SetLoading_ZeroState_BlanksPaneStill(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20) // zero state
	m.SetLoading(true)

	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "Loading quota data") {
		t.Errorf("AC6: 'Loading quota data' missing — zero-state+SetLoading(true) must still show spinner:\n%s", got)
	}
	// Refreshing indicator must NOT appear here — this is initial load, not a refresh.
	if strings.Contains(got, "refreshing") {
		t.Errorf("AC6: 'refreshing' present in initial-load state — unexpected:\n%s", got)
	}
}

// ==================== End FB-063 ====================

// ==================== FB-088: QuotaDashboard origin affordance ====================

// AC1 [Observable] — TablePane origin → titleBar contains "[3] back to resource list".
func TestFB088_QuotaDashboard_AC1_TablePaneOrigin(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetOriginLabel("resource list")

	got := stripANSI(m.titleBar())
	if !strings.Contains(got, "[3] back to resource list") {
		t.Errorf("AC1: titleBar() missing '[3] back to resource list', got:\n%s", got)
	}
}

// AC2 [Observable] — NavPane (welcome) origin → titleBar contains "[3] back to welcome panel".
func TestFB088_QuotaDashboard_AC2_WelcomePanelOrigin(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetOriginLabel("welcome panel")

	got := stripANSI(m.titleBar())
	if !strings.Contains(got, "[3] back to welcome panel") {
		t.Errorf("AC2: titleBar() missing '[3] back to welcome panel', got:\n%s", got)
	}
}

// AC3 [Observable] — ActivityDashboard chain origin → titleBar contains "[3] back to activity dashboard".
func TestFB088_QuotaDashboard_AC3_ActivityDashOrigin(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetOriginLabel("activity dashboard")

	got := stripANSI(m.titleBar())
	if !strings.Contains(got, "[3] back to activity dashboard") {
		t.Errorf("AC3: titleBar() missing '[3] back to activity dashboard', got:\n%s", got)
	}
}

// AC4 [Input-changed] — different origin labels produce different titleBar content.
func TestFB088_QuotaDashboard_AC4_InputChanged(t *testing.T) {
	t.Parallel()
	m1 := newDashboard(200, 20)
	m1.SetOriginLabel("resource list")
	got1 := stripANSI(m1.titleBar())

	m2 := newDashboard(200, 20)
	m2.SetOriginLabel("detail view")
	got2 := stripANSI(m2.titleBar())

	if !strings.Contains(got1, "resource list") {
		t.Errorf("AC4: got1 missing 'resource list': %s", got1)
	}
	if !strings.Contains(got2, "detail view") {
		t.Errorf("AC4: got2 missing 'detail view': %s", got2)
	}
	if got1 == got2 {
		t.Error("AC4: titleBar() output identical for different origin labels; want distinct content")
	}
}

// AC5 [Anti-behavior] — empty originLabel (fresh startup) → titleBar does NOT contain "back to".
func TestFB088_QuotaDashboard_AC5_FreshStartupNoHint(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	// No SetOriginLabel call — zero state.

	got := stripANSI(m.titleBar())
	if strings.Contains(got, "back to") {
		t.Errorf("AC5: titleBar() contains 'back to' with empty originLabel; want absent:\n%s", got)
	}
}

// AC6 [Anti-behavior] — after SetOriginLabel("") the hint is suppressed.
func TestFB088_QuotaDashboard_AC6_ClearOriginSuppressesHint(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetOriginLabel("resource list")
	m.SetOriginLabel("") // simulate Esc clear

	got := stripANSI(m.titleBar())
	if strings.Contains(got, "back to") {
		t.Errorf("AC6: titleBar() contains 'back to' after SetOriginLabel(\"\"); want absent:\n%s", got)
	}
}

// AC8 [Anti-regression] — existing FB-043 titleBar layout (freshness) unaffected when originLabel is empty.
func TestFB088_QuotaDashboard_AC8_TitleBarLayoutRegression(t *testing.T) {
	t.Parallel()
	m := newDashboard(200, 20)
	m.SetBucketFetchedAt(time.Now().Add(-10 * time.Second))
	// originLabel is empty — should behave identically to pre-FB-088.

	got := stripANSI(m.titleBar())
	if !strings.Contains(got, "updated") {
		t.Errorf("AC8: freshness signal 'updated' missing when originLabel is empty:\n%s", got)
	}
	if strings.Contains(got, "back to") {
		t.Errorf("AC8: 'back to' spuriously present when originLabel is empty:\n%s", got)
	}
}

// ==================== End FB-088 (QuotaDashboard) ====================

// ==================== FB-113: Empty-bucket viewport origin-label hint ====================

// AC1 [Observable] — empty-state with originLabel "resource list" shows "[Esc] back to resource list".
func TestFB113_AC1_ResourceListOrigin_EmptyStateHint(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetOriginLabel("resource list")

	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "[Esc] back to resource list") {
		t.Errorf("AC1: '[Esc] back to resource list' missing from empty-state:\n%s", got)
	}
}

// AC2 [Observable] — empty-state with originLabel "welcome panel" shows "[Esc] back to welcome panel".
func TestFB113_AC2_WelcomePanelOrigin_EmptyStateHint(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetOriginLabel("welcome panel")

	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "[Esc] back to welcome panel") {
		t.Errorf("AC2: '[Esc] back to welcome panel' missing from empty-state:\n%s", got)
	}
}

// AC3 [Input-changed] — same empty fixture, different originLabel → rendered content differs.
func TestFB113_AC3_InputChanged_DifferentOrigins_DifferentContent(t *testing.T) {
	t.Parallel()
	m1 := newDashboard(80, 20)
	m1.SetOriginLabel("resource list")

	m2 := newDashboard(80, 20)
	m2.SetOriginLabel("welcome panel")

	got1 := stripANSI(m1.buildMainContent())
	got2 := stripANSI(m2.buildMainContent())

	if got1 == got2 {
		t.Errorf("AC3 [Input-changed]: buildMainContent() identical for different originLabels:\n  resource list: %q\n  welcome panel: %q", got1, got2)
	}
	if !strings.Contains(got1, "resource list") {
		t.Errorf("AC3: 'resource list' missing from first render:\n%s", got1)
	}
	if !strings.Contains(got2, "welcome panel") {
		t.Errorf("AC3: 'welcome panel' missing from second render:\n%s", got2)
	}
}

// AC4 [Anti-regression] — empty originLabel falls back to "navigation"; no crash.
func TestFB113_AC4_EmptyOriginLabel_FallbackToNavigation(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	// originLabel is "" (default)

	got := stripANSI(m.buildMainContent())
	if !strings.Contains(got, "[Esc] back to navigation") {
		t.Errorf("AC4: '[Esc] back to navigation' fallback missing when originLabel is empty:\n%s", got)
	}
}

// AC5 [Anti-regression] — populated-bucket state does not render the empty-state block.
func TestFB113_AC5_PopulatedBuckets_NoEmptyStateHint(t *testing.T) {
	t.Parallel()
	m := newDashboard(80, 20)
	m.SetOriginLabel("resource list")
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("b1", "compute/cpus", 10, 100),
	})

	got := stripANSI(m.buildMainContent())
	if strings.Contains(got, "No allowance buckets configured") {
		t.Errorf("AC5: empty-state text present with populated buckets:\n%s", got)
	}
	if strings.Contains(got, "[Esc] back to") {
		t.Errorf("AC5: '[Esc] back to' hint present with populated buckets; must only appear in empty-state:\n%s", got)
	}
}

// ==================== End FB-113 ====================
