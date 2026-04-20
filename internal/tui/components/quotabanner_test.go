package components

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/tui/data"
)

// --- QuotaBarStyling threshold boundaries ---

func TestQuotaBarStyling_Thresholds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		pct        int
		wantSuffix string
		noSuffix   bool
	}{
		{"69pct — no suffix", 69, "", true},
		{"70pct — no suffix", 70, "", true},
		{"89pct — no suffix", 89, "", true},
		{"90pct — ⚠ near", 90, "⚠ near", false},
		{"99pct — ⚠ near", 99, "⚠ near", false},
		{"100pct — ⛔ full", 100, "⛔ full", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, _, suffix := QuotaBarStyling(tt.pct)
			if tt.noSuffix && suffix != "" {
				t.Errorf("QuotaBarStyling(%d): suffix = %q, want empty", tt.pct, suffix)
			}
			// Suffix may have a leading space; use Contains for robustness.
			if !tt.noSuffix && !strings.Contains(suffix, tt.wantSuffix) {
				t.Errorf("QuotaBarStyling(%d): suffix = %q, want to contain %q", tt.pct, suffix, tt.wantSuffix)
			}
		})
	}
}

// --- bucketResourceLabel ---

func TestBucketResourceLabel_WithGroup(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{ResourceType: "compute.example.io/cpus"}
	got := bucketResourceLabel(b, nil)
	if got != "cpus" {
		t.Errorf("bucketResourceLabel with group = %q, want %q", got, "cpus")
	}
}

func TestBucketResourceLabel_CoreGroup(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{ResourceType: "pods"}
	got := bucketResourceLabel(b, nil)
	if got != "pods" {
		t.Errorf("bucketResourceLabel core group = %q, want %q", got, "pods")
	}
}

func TestBucketResourceLabel_MultipleSlashes(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{ResourceType: "a.b.c/d/resource"}
	got := bucketResourceLabel(b, nil)
	if got != "resource" {
		t.Errorf("bucketResourceLabel multi-slash = %q, want %q", got, "resource")
	}
}


// --- QuotaBannerModel height / presence ---

func TestQuotaBannerModel_Empty_Height(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(80)
	if m.Height() != 0 {
		t.Errorf("Height() = %d, want 0 for empty banner", m.Height())
	}
	if m.HasBuckets() {
		t.Error("HasBuckets() = true, want false for empty banner")
	}
}

func TestQuotaBannerModel_SetBuckets_OneRow(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(80)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "compute.example.io/cpus", Allocated: 50, Limit: 100},
	})
	if m.Height() != 1 {
		t.Errorf("Height() = %d, want 1", m.Height())
	}
	if !m.HasBuckets() {
		t.Error("HasBuckets() = false, want true after SetBuckets")
	}
}

func TestQuotaBannerModel_SetBuckets_TwoRows(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(80)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 90, Limit: 100},
		{ResourceType: "a/mem", Allocated: 50, Limit: 100},
	})
	if m.Height() != 2 {
		t.Errorf("Height() = %d, want 2", m.Height())
	}
}

func TestQuotaBannerModel_SetBuckets_Nil_ClearsRows(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(80)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 1, Limit: 100},
	})
	m.SetBuckets(nil)
	if m.Height() != 0 {
		t.Errorf("Height() = %d after SetBuckets(nil), want 0", m.Height())
	}
}

func TestQuotaBannerModel_View_Empty_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(80)
	if m.View() != "" {
		t.Errorf("View() on empty banner = %q, want empty string", m.View())
	}
}

// --- View full form ---

func TestQuotaBannerModel_View_FullForm_ContainsResourceLabel(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200) // wide enough for full form
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "compute.example.io/cpus", Allocated: 50, Limit: 100, ConsumerKind: "project", ConsumerName: "proj-1"},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "cpus") {
		t.Errorf("View full: want resource label 'cpus' in %q", got)
	}
	if !strings.Contains(got, "Quota") {
		t.Errorf("View full: want 'Quota' in %q", got)
	}
	if strings.Contains(got, "consumer:") {
		t.Errorf("View full: must not contain 'consumer:' field in %q", got)
	}
}

func TestQuotaBannerModel_View_FullForm_Suffix90(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 90, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "⚠ near") {
		t.Errorf("View full 90pct: want '⚠ near' suffix in %q", got)
	}
}

func TestQuotaBannerModel_View_FullForm_Suffix100(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 100, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "⛔ full") {
		t.Errorf("View full 100pct: want '⛔ full' suffix in %q", got)
	}
}

// --- View compact form ---

func TestQuotaBannerModel_View_CompactForm_NarrowWidth(t *testing.T) {
	t.Parallel()
	// Narrow width forces compact form (no 'Quota', no 'consumer:').
	m := NewQuotaBannerModel(40)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "compute.example.io/cpus", Allocated: 50, Limit: 100, ConsumerKind: "project", ConsumerName: "x"},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "cpus") {
		t.Errorf("View compact: want resource label 'cpus' in %q", got)
	}
	if strings.Contains(got, "consumer:") {
		t.Errorf("View compact: must not contain 'consumer:' at narrow width, got %q", got)
	}
}

func TestQuotaBannerModel_View_TwoBuckets_TwoLines(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 90, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
		{ResourceType: "a/mem", Allocated: 50, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
	})
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 2 {
		t.Errorf("View two buckets: got %d lines, want 2; view = %q", len(lines), view)
	}
}


// --- Tree-mode Height (AC-3, AC-5, AC-11) ---

func TestQuotaBannerModel_TreeHeight_ParentChildSiblings_IsThree(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		{Name: "sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other", Allocated: 5, Limit: 100},
	})
	if m.Height() != 3 {
		t.Errorf("Height() = %d, want 3 (parent+sibling-consume+child)", m.Height())
	}
}

func TestQuotaBannerModel_TreeHeight_ParentChildNoSiblings_IsTwo(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
	})
	if m.Height() != 2 {
		t.Errorf("Height() = %d, want 2 (parent+child, no siblings)", m.Height())
	}
}

func TestQuotaBannerModel_TreeHeight_SiblingsRestricted_IsTwo(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetSiblingRestricted(true)
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		{Name: "sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other", Allocated: 5, Limit: 100},
	})
	if m.Height() != 2 {
		t.Errorf("Height() = %d, want 2 when siblingRestricted=true (sibling row suppressed)", m.Height())
	}
}

// --- Tree-mode View ---

func TestQuotaBannerModel_TreeView_ContainsConnectors(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "├─") {
		t.Errorf("tree banner: want '├─' parent connector in %q", got)
	}
	if !strings.Contains(got, "└─") {
		t.Errorf("tree banner: want '└─' child connector in %q", got)
	}
}

func TestQuotaBannerModel_TreeView_WithSiblings_HasSibRow(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		{Name: "sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other", Allocated: 5, Limit: 100},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "sibling") {
		t.Errorf("tree banner with siblings: want sibling row in %q", got)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("tree banner with siblings: want 3 lines, got %d: %q", len(lines), got)
	}
}

func TestQuotaBannerModel_TreeView_SiblingsRestricted_NoSibRow(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetSiblingRestricted(true)
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100},
		{Name: "sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other", Allocated: 5, Limit: 100},
	})
	got := stripANSI(m.View())
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("tree banner restricted: want 2 lines, got %d: %q", len(lines), got)
	}
}

// --- FB-014: bucketResourceLabel with registrations ---

func TestBucketResourceLabel_WithRegistrations_HitDescription(t *testing.T) {
	t.Parallel()
	regs := []data.ResourceRegistration{
		{Group: "compute.example.io", Name: "cpus", Description: "CPU cores"},
	}
	b := data.AllowanceBucket{ResourceType: "compute.example.io/cpus"}
	got := bucketResourceLabel(b, regs)
	if got != "CPU cores" {
		t.Errorf("bucketResourceLabel with matching reg = %q, want %q", got, "CPU cores")
	}
}

func TestBucketResourceLabel_WithRegistrations_EmptyDescription_FallsBackToShortName(t *testing.T) {
	t.Parallel()
	// Entry present but Description is "" — must fall back.
	regs := []data.ResourceRegistration{
		{Group: "compute.example.io", Name: "cpus", Description: ""},
	}
	b := data.AllowanceBucket{ResourceType: "compute.example.io/cpus"}
	got := bucketResourceLabel(b, regs)
	if got != "cpus" {
		t.Errorf("bucketResourceLabel empty description = %q, want short name %q", got, "cpus")
	}
}

func TestBucketResourceLabel_WithRegistrations_NoMatch_FallsBackToShortName(t *testing.T) {
	t.Parallel()
	regs := []data.ResourceRegistration{
		{Group: "other.io", Name: "memory", Description: "RAM"},
	}
	b := data.AllowanceBucket{ResourceType: "compute.example.io/cpus"}
	got := bucketResourceLabel(b, regs)
	if got != "cpus" {
		t.Errorf("bucketResourceLabel no-match reg = %q, want short name %q", got, "cpus")
	}
}

// TestBucketResourceLabel_WithLongDescription_TruncatesWithEllipsis verifies that
// truncateLabelToWidth clips a long description to the budget and appends "…"
// without exceeding the cell budget (AC#10 — label truncation).
func TestBucketResourceLabel_WithLongDescription_TruncatesWithEllipsis(t *testing.T) {
	t.Parallel()
	longDesc := "Projects created within Organizations" // 37 chars
	budget := 20

	result := truncateLabelToWidth(longDesc, budget)

	if !strings.HasSuffix(result, "…") {
		t.Errorf("truncateLabelToWidth: result = %q, want suffix '…'", result)
	}
	if lipgloss.Width(result) > budget {
		t.Errorf("truncateLabelToWidth: width %d > budget %d, result = %q",
			lipgloss.Width(result), budget, result)
	}
	if !strings.HasPrefix(result, "P") {
		t.Errorf("truncateLabelToWidth: result = %q, want non-empty prefix of description", result)
	}
}

func TestQuotaBannerModel_SetRegistrations_ViewShowsDescription(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "compute.example.io/cpus", Allocated: 5, Limit: 100, ConsumerKind: "project", ConsumerName: "p"},
	})

	// Before registrations: label should be the short name.
	before := stripANSI(m.View())
	if !strings.Contains(before, "cpus") {
		t.Errorf("before SetRegistrations: want 'cpus' short name in %q", before)
	}

	// After SetRegistrations with a matching description, label should change.
	m.SetRegistrations([]data.ResourceRegistration{
		{Group: "compute.example.io", Name: "cpus", Description: "CPU cores"},
	})
	after := stripANSI(m.View())
	if !strings.Contains(after, "CPU cores") {
		t.Errorf("after SetRegistrations: want 'CPU cores' description in %q", after)
	}
	if strings.Contains(after, "cpus") && !strings.Contains(after, "CPU cores") {
		t.Errorf("after SetRegistrations: short name 'cpus' still shown without description in %q", after)
	}
}

// --- E2E-15: Six percentage threshold fixtures in banner compact form ---

func TestQuotaBannerModel_View_ThresholdFixtures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		allocated  int64
		limit      int64
		wantSuffix string
		noSuffix   bool
	}{
		{"69pct — no suffix", 69, 100, "", true},
		{"70pct — no suffix", 70, 100, "", true},
		{"89pct — no suffix", 89, 100, "", true},
		{"90pct — ⚠ near", 90, 100, "⚠ near", false},
		{"99pct — ⚠ near", 99, 100, "⚠ near", false},
		{"100pct — ⛔ full", 100, 100, "⛔ full", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewQuotaBannerModel(40) // compact form
			m.SetBuckets([]data.AllowanceBucket{
				{ResourceType: "a/r", Allocated: tt.allocated, Limit: tt.limit},
			})
			got := stripANSI(m.View())
			if tt.noSuffix && (strings.Contains(got, "⚠") || strings.Contains(got, "⛔")) {
				t.Errorf("banner compact %d%%: unexpected suffix in %q", tt.allocated, got)
			}
			if !tt.noSuffix && !strings.Contains(got, tt.wantSuffix) {
				t.Errorf("banner compact %d%%: want %q in %q", tt.allocated, tt.wantSuffix, got)
			}
		})
	}
}

// ==================== FB-036: Remove recon age and claim count ====================

// TestFB036_BannerFull_NoReconSubstring — AC#1+2: full-form banner (w=200) contains
// no "recon" or "stale" substring, even when LastReconciliation is set or stale (>15m).
func TestFB036_BannerFull_NoReconSubstring(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		recon time.Time
	}{
		{"recent_recon", time.Now().Add(-2 * time.Minute)},
		{"stale_recon_20m", time.Now().Add(-20 * time.Minute)},
		{"zero_recon", time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewQuotaBannerModel(200)
			m.SetBuckets([]data.AllowanceBucket{
				{ResourceType: "a/cpus", Allocated: 50, Limit: 100, LastReconciliation: tt.recon},
			})
			got := stripANSI(m.View())
			if strings.Contains(got, "recon") {
				t.Errorf("AC#1 %s: 'recon' found in banner view %q", tt.name, got)
			}
			if strings.Contains(got, "stale") {
				t.Errorf("AC#2 %s: 'stale' found in banner view %q", tt.name, got)
			}
		})
	}
}

// TestFB036_BannerTree_NoReconSubstring — AC#3: tree-form banner (w=200) contains no "recon".
func TestFB036_BannerTree_NoReconSubstring(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetActiveConsumer("Project", "my-proj")
	m.SetBuckets([]data.AllowanceBucket{
		{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org",
			Allocated: 30, Limit: 100, LastReconciliation: time.Now().Add(-20 * time.Minute)},
		{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj",
			Allocated: 10, Limit: 100, LastReconciliation: time.Now()},
	})
	got := stripANSI(m.View())
	if strings.Contains(got, "recon") {
		t.Errorf("AC#3: 'recon' found in tree banner:\n%s", got)
	}
}

// TestFB036_BannerFull_BarAndCountsPresent — AC#6/AC#7: bar glyphs and counts
// still render after recon overhead was removed.
func TestFB036_BannerFull_BarAndCountsPresent(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 50, Limit: 100},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "/ ") {
		t.Errorf("AC#6: '/ ' counts separator missing in banner: %q", got)
	}
	if !strings.Contains(got, "█") && !strings.Contains(got, "░") {
		t.Errorf("AC#7: bar glyphs (█ or ░) missing in banner: %q", got)
	}
}

// TestFB036_BannerCompact_W40_Unchanged — AC#7: compact form (w=40) still renders counts.
func TestFB036_BannerCompact_W40_Unchanged(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(40)
	m.SetBuckets([]data.AllowanceBucket{
		{ResourceType: "a/cpus", Allocated: 50, Limit: 100},
	})
	got := stripANSI(m.View())
	if strings.Contains(got, "Quota") {
		t.Error("AC#7: compact mode should not contain 'Quota' keyword (full-form leaked)")
	}
	if !strings.Contains(got, "50") {
		t.Errorf("AC#7: allocated count '50' missing in compact banner: %q", got)
	}
}

// TestFB036_BannerFull_W60_ShortLabel_StaysFullMode — AC#7a: at w=60 with a short label,
// post-FB-036 overhead reduction keeps the bar in full mode (barWidth ≥ 10).
func TestFB036_BannerFull_W60_ShortLabel_StaysFullMode(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(60)
	m.SetBuckets([]data.AllowanceBucket{
		// "cpus" = 4 chars → overhead = 2+4+9+2+18+8 = 43, barWidth = 60-43 = 17 ≥ 10 → full
		{ResourceType: "a/cpus", Allocated: 50, Limit: 100},
	})
	got := stripANSI(m.View())
	if !strings.Contains(got, "Quota") {
		t.Errorf("AC#7a: w=60 short label should render full mode with 'Quota' keyword, got: %q", got)
	}
}

// TestFB036_BannerFull_ZeroLastRecon_NoEmDash — AC#1a: zero LastReconciliation
// produces no "recon —" em-dash (recon cell removed entirely).
func TestFB036_BannerFull_ZeroLastRecon_NoEmDash(t *testing.T) {
	t.Parallel()
	m := NewQuotaBannerModel(200)
	m.SetBuckets([]data.AllowanceBucket{
		// LastReconciliation is zero value; Limit>0 so no em-dash from unlimited path.
		{ResourceType: "a/cpus", Allocated: 50, Limit: 100},
	})
	got := stripANSI(m.View())
	if strings.Contains(got, "recon") {
		t.Errorf("AC#1a: 'recon' found even with zero LastReconciliation: %q", got)
	}
}

// TestFB036_CompactFormTree_NoOutOfSync — AC#5a (compact path): renderBannerTreeRowCompact
// must not emit "(out of sync)" across recent recon, stale recon, and zero-time recon states.
// Mirrors TestFB036_FullFormTree_NoOutOfSync but drives the compact-form code path (w≤59).
func TestFB036_CompactFormTree_NoOutOfSync(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		parentLastRecon    time.Time
		childLastRecon     time.Time
	}{
		{"recent_recon", time.Now().Add(-1 * time.Minute), time.Now()},
		{"stale_recon_20m", time.Now().Add(-20 * time.Minute), time.Now().Add(-25 * time.Minute)},
		{"zero_recon", time.Time{}, time.Time{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := NewQuotaBannerModel(40) // w=40 forces compact-form path
			m.SetActiveConsumer("Project", "my-proj")
			m.SetBuckets([]data.AllowanceBucket{
				{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org",
					Allocated: 30, Limit: 100, LastReconciliation: tt.parentLastRecon},
				{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj",
					Allocated: 10, Limit: 100, LastReconciliation: tt.childLastRecon},
			})
			got := stripANSI(m.View())
			if strings.Contains(got, "out of sync") {
				t.Errorf("AC#5a compact %s: '(out of sync)' found in compact tree banner:\n%s", tt.name, got)
			}
			if strings.Contains(got, "recon") {
				t.Errorf("AC#5a compact %s: 'recon' found in compact tree banner:\n%s", tt.name, got)
			}
		})
	}
}

// ==================== End FB-036 (banner) ====================
