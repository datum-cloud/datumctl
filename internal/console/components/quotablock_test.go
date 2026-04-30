package components

import (
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/console/data"
)


func TestRenderBarLine_Thresholds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		allocated  int64
		limit      int64
		wantSuffix string // non-empty: expected suffix fragment
		noSuffix   bool   // true: must not contain ⚠ or ⛔
	}{
		{"0pct success no suffix", 0, 100, "", true},
		{"50pct success no suffix", 50, 100, "", true},
		{"69pct warning boundary no suffix", 69, 100, "", true},
		{"70pct warning no suffix", 70, 100, "", true},
		{"89pct warning no suffix", 89, 100, "", true},
		{"90pct error near suffix", 90, 100, "⚠ near", false},
		{"99pct error near suffix", 99, 100, "⚠ near", false},
		{"100pct error full suffix", 100, 100, "⛔ full", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := data.AllowanceBucket{Allocated: tt.allocated, Limit: tt.limit}
			got := stripANSI(renderBarLine(b, 80))
			if tt.wantSuffix != "" && !strings.Contains(got, tt.wantSuffix) {
				t.Errorf("renderBarLine(%d/%d): want suffix %q in %q", tt.allocated, tt.limit, tt.wantSuffix, got)
			}
			if tt.noSuffix && (strings.Contains(got, "⚠") || strings.Contains(got, "⛔")) {
				t.Errorf("renderBarLine(%d/%d): unexpected suffix in %q", tt.allocated, tt.limit, got)
			}
		})
	}
}

func TestRenderBarLine_Unlimited(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{Allocated: 5, Limit: 0}
	got := stripANSI(renderBarLine(b, 80))
	if !strings.Contains(got, "░") {
		t.Errorf("unlimited bar: want ░ (empty fill) in %q", got)
	}
	if !strings.Contains(got, "∞") {
		t.Errorf("unlimited bar: want ∞ in %q", got)
	}
	if !strings.Contains(got, "—") {
		t.Errorf("unlimited bar: want — (no pct) in %q", got)
	}
}

func TestRenderBarLine_FilledProportion(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{Allocated: 50, Limit: 100}
	got := stripANSI(renderBarLine(b, 80))
	filled := strings.Count(got, "█")
	empty := strings.Count(got, "░")
	if filled == 0 || empty == 0 {
		t.Fatalf("50pct bar: expected both █ and ░, got %q", got)
	}
	// Filled and empty should be roughly equal (within 5 chars).
	diff := filled - empty
	if diff < 0 {
		diff = -diff
	}
	if diff > 5 {
		t.Errorf("50pct bar: filled=%d empty=%d, expected roughly equal", filled, empty)
	}
}

func TestRenderBarLine_FullBar(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{Allocated: 100, Limit: 100}
	got := stripANSI(renderBarLine(b, 80))
	if strings.Contains(got, "░") {
		t.Errorf("100pct bar: unexpected empty fill ░ in %q", got)
	}
	if strings.Count(got, "█") == 0 {
		t.Errorf("100pct bar: expected filled chars █ in %q", got)
	}
}

// --- consumerLabel ---

func TestConsumerLabel_ShortString_PaddedTo20(t *testing.T) {
	t.Parallel()
	got := consumerLabel("Project", "p")
	if len([]rune(got)) != 20 {
		t.Errorf("consumerLabel short len = %d, want 20; got %q", len([]rune(got)), got)
	}
}

func TestConsumerLabel_LongString_TruncatedTo20(t *testing.T) {
	t.Parallel()
	got := consumerLabel("Organization", "very-long-name-here-exceeds")
	if len([]rune(got)) != 20 {
		t.Errorf("consumerLabel long len = %d, want 20; got %q", len([]rune(got)), got)
	}
}


// --- RenderQuotaTree ---

func TestRenderQuotaTree_NoTree_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	tree := data.TreeBuckets{HasTree: false}
	got := RenderQuotaTree(tree, 120, false)
	if got != "" {
		t.Errorf("RenderQuotaTree HasTree=false = %q, want empty string", got)
	}
}

func newTestTree() data.TreeBuckets {
	org := data.AllowanceBucket{Name: "org", ResourceType: "a/r", ConsumerKind: "Organization", ConsumerName: "my-org", Allocated: 30, Limit: 100}
	proj := data.AllowanceBucket{Name: "proj", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "my-proj", Allocated: 10, Limit: 100}
	return data.TreeBuckets{
		Parent:      &org,
		ActiveChild: &proj,
		HasTree:     true,
	}
}

func TestRenderQuotaTree_BasicTree_ContainsConnectors(t *testing.T) {
	t.Parallel()
	tree := newTestTree()
	got := stripANSI(RenderQuotaTree(tree, 120, false))
	if !strings.Contains(got, "├─") {
		t.Errorf("RenderQuotaTree: want '├─' parent connector in %q", got)
	}
	if !strings.Contains(got, "└─") {
		t.Errorf("RenderQuotaTree: want '└─' child connector in %q", got)
	}
}

func TestRenderQuotaTree_WithSiblings_HasSiblingRow(t *testing.T) {
	t.Parallel()
	tree := newTestTree()
	sib := data.AllowanceBucket{Name: "sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other-proj", Allocated: 5, Limit: 100}
	tree.Siblings = []data.AllowanceBucket{sib}
	got := stripANSI(RenderQuotaTree(tree, 120, false))
	if !strings.Contains(got, "sibling-consume") {
		t.Errorf("RenderQuotaTree with siblings: want 'sibling-consume' row in %q", got)
	}
	// Should be 3 lines: parent, sibling-consume, child.
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("RenderQuotaTree with siblings: want 3 lines, got %d: %q", len(lines), got)
	}
}

func TestRenderQuotaTree_SiblingsRestricted_NoSiblingRow_HasNote(t *testing.T) {
	t.Parallel()
	tree := newTestTree()
	sib := data.AllowanceBucket{Name: "sib", ResourceType: "a/r", ConsumerKind: "Project", ConsumerName: "other"}
	tree.Siblings = []data.AllowanceBucket{sib}
	got := stripANSI(RenderQuotaTree(tree, 120, true)) // siblingsRestricted = true
	if strings.Contains(got, "sibling-consume") {
		t.Errorf("RenderQuotaTree restricted: must not show sibling-consume row, got %q", got)
	}
	if !strings.Contains(got, "other projects' usage hidden") {
		t.Errorf("RenderQuotaTree restricted: want '(other projects' usage hidden)' in parent row %q", got)
	}
	// Should be 2 lines: parent (with note), child
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("RenderQuotaTree restricted: want 2 lines, got %d: %q", len(lines), got)
	}
}

func TestRenderQuotaTree_NoSiblings_TwoLines(t *testing.T) {
	t.Parallel()
	tree := newTestTree() // no siblings
	got := stripANSI(RenderQuotaTree(tree, 120, false))
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("RenderQuotaTree no siblings: want 2 lines, got %d: %q", len(lines), got)
	}
}


// --- FB-014: RenderQuotaBlock with registrations ---

// TestRenderQuotaBlock_WithRegistrations_ShowsDescription verifies that when
// a matching ResourceRegistration with a non-empty Description exists, the
// rendered block header shows the description and NOT the fully-qualified type
// name (AC#4 — S3 describe block).
func TestRenderQuotaBlock_WithRegistrations_ShowsDescription(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{
		ResourceType: "resourcemanager.miloapis.com/projects",
		Allocated:    4,
		Limit:        10,
	}
	regs := []data.ResourceRegistration{
		{Group: "resourcemanager.miloapis.com", Name: "projects", Description: "Projects created within Organizations"},
	}
	got := stripANSI(RenderQuotaBlock(b, 80, regs))
	if !strings.Contains(got, "Projects created within Organizations") {
		t.Errorf("RenderQuotaBlock with regs: want description in output, got %q", got)
	}
	// The fully-qualified type should be replaced, not appended.
	if strings.Contains(got, "resourcemanager.miloapis.com/projects") {
		t.Errorf("RenderQuotaBlock with regs: fully-qualified name leaked into output %q", got)
	}
}

// TestRenderQuotaBlock_WithRegistrations_EmptyDescription_FallsBackToShortName verifies
// that when a matching registration exists but Description is empty, the block falls
// back to the short name (last "/" segment) rather than showing an empty label (AC#4 / AC#9).
func TestRenderQuotaBlock_WithRegistrations_EmptyDescription_FallsBackToShortName(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{
		ResourceType: "resourcemanager.miloapis.com/projects",
		Allocated:    4,
		Limit:        10,
	}
	regs := []data.ResourceRegistration{
		{Group: "resourcemanager.miloapis.com", Name: "projects", Description: ""},
	}
	got := stripANSI(RenderQuotaBlock(b, 80, regs))
	if !strings.Contains(got, "projects") {
		t.Errorf("RenderQuotaBlock empty description: want short name 'projects' in output, got %q", got)
	}
}

func TestRenderQuotaBlock_Structure(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{
		ResourceType: "compute.example.io/cpus",
		Allocated:    40,
		Limit:        100,
		ClaimCount:   7,
	}
	got := stripANSI(RenderQuotaBlock(b, 80, nil))
	// ResolveDescription with nil registrations falls back to the short name (last "/" segment).
	if !strings.Contains(got, "cpus") {
		t.Errorf("RenderQuotaBlock: resource short name 'cpus' not found in %q", got)
	}
	if !strings.Contains(got, "40") {
		t.Errorf("RenderQuotaBlock: allocated count not found in %q", got)
	}
	if !strings.Contains(got, "100") {
		t.Errorf("RenderQuotaBlock: limit not found in %q", got)
	}
	// Should be exactly 2 lines (header, bar) — stats line removed in FB-036.
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Errorf("RenderQuotaBlock: want 2 lines, got %d:\n%s", len(lines), got)
	}
}

// ==================== FB-036: Remove recon age and claim count (block / tree) ====================

// TestFB036_QuotaBlock_NoReconOrClaims — AC#3/4/5: RenderQuotaBlock output must
// not contain "claims:", "reconciled", or "recon" even when ClaimCount and
// LastReconciliation are set.
func TestFB036_QuotaBlock_NoReconOrClaims(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{
		ResourceType:       "compute.example.io/cpus",
		Allocated:          40,
		Limit:              100,
		ClaimCount:         7,
		LastReconciliation: time.Now().Add(-20 * time.Minute),
	}
	got := stripANSI(RenderQuotaBlock(b, 80, nil))
	for _, forbidden := range []string{"claims:", "reconciled", "recon"} {
		if strings.Contains(got, forbidden) {
			t.Errorf("AC#3/4/5: %q found in RenderQuotaBlock output: %q", forbidden, got)
		}
	}
}

// TestFB036_QuotaBlock_HeightIs2Lines — AC#4a: block is exactly 2 lines (header + bar).
func TestFB036_QuotaBlock_HeightIs2Lines(t *testing.T) {
	t.Parallel()
	b := data.AllowanceBucket{ResourceType: "a/cpus", Allocated: 5, Limit: 100}
	got := stripANSI(RenderQuotaBlock(b, 80, nil))
	newlines := strings.Count(got, "\n")
	if newlines != 1 {
		t.Errorf("AC#4a: RenderQuotaBlock has %d newlines (want 1 → 2 lines):\n%s", newlines, got)
	}
}

// TestFB036_FullFormTree_NoOutOfSync — AC#5a: RenderQuotaTree at w=200 with
// out-of-sync timestamps does not render "(out of sync)" in full-form output.
func TestFB036_FullFormTree_NoOutOfSync(t *testing.T) {
	t.Parallel()
	now := time.Now()
	org := data.AllowanceBucket{
		Name: "org", ResourceType: "a/r",
		ConsumerKind: "Organization", ConsumerName: "my-org",
		Allocated: 30, Limit: 100,
		LastReconciliation: now.Add(-20 * time.Minute),
	}
	proj := data.AllowanceBucket{
		Name: "proj", ResourceType: "a/r",
		ConsumerKind: "Project", ConsumerName: "my-proj",
		Allocated: 10, Limit: 100,
		LastReconciliation: now,
	}
	tree := data.TreeBuckets{
		Parent:      &org,
		ActiveChild: &proj,
		HasTree:     true,
	}
	got := stripANSI(RenderQuotaTree(tree, 200, false))
	if strings.Contains(got, "out of sync") {
		t.Errorf("AC#5a: '(out of sync)' found in full-form tree output: %q", got)
	}
}

// ==================== End FB-036 (block/tree) ====================
