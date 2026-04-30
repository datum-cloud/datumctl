package components

import (
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/console/data"
)

// ==================== FB-060: Failed quota refresh signal ====================
//
// Axis-coverage table:
// AC1 | Observable                   | TestFB060_AC1_Observable_RefreshFailedInTitleBar
// AC2 | Input-changed (fail→succeed) | TestFB060_AC2_InputChanged_SetBuckets_ClearsFailedSignal
// AC3 | Anti-regression (fetchedAt)  | TestFB060_AC3_AntiRegression_FetchedAtUnchangedOnFailure
// AC4 | Anti-regression (preemption) | TestFB060_AC4_AntiRegression_RefreshingPreemptsFailedSignal
// AC5 | Anti-regression (existing)   | go test ./internal/console/...
// AC6 | Integration                  | go install ./... + go test ./internal/console/...

// newBucketedWideDashboard returns a wide (w=100, h=20) dashboard with two
// pre-loaded buckets and a recent fetchedAt so the title-bar chrome is active.
func newBucketedWideDashboard() QuotaDashboardModel {
	m := NewQuotaDashboardModel(100, 20, "")
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("dns-zones", "networking/dnszones", 120, 200),
		projBucket("backends", "networking/backends", 40, 100),
	})
	m.SetBucketFetchedAt(time.Now().Add(-5 * time.Second))
	return m
}

// AC1 [Observable] — after SetRefreshFailed(true) with refreshing=false, wide
// View() contains "refresh failed".
func TestFB060_AC1_Observable_RefreshFailedInTitleBar(t *testing.T) {
	t.Parallel()
	m := newBucketedWideDashboard() // width=100 ≥ 80 (wide mode), height=20 ≥ 6 (chrome)
	m.SetRefreshFailed(true)
	// refreshing defaults to false — only refreshFailed is set

	got := stripANSI(m.View())
	if !strings.Contains(got, "refresh failed") {
		t.Errorf("AC1: 'refresh failed' missing from View() after SetRefreshFailed(true) at width=100:\n%s", got)
	}
}

// AC2 [Input-changed — fail→succeed cycle]
// State A: fail signal present in title bar.
// State B: after SetBuckets(), fail signal absent.
// Both states must produce View() output that differs at the title-bar substring.
func TestFB060_AC2_InputChanged_SetBuckets_ClearsFailedSignal(t *testing.T) {
	t.Parallel()
	m := newBucketedWideDashboard() // width=100, height=20
	m.SetRefreshFailed(true)

	// State A — fail signal present.
	stateA := stripANSI(m.View())
	if !strings.Contains(stateA, "refresh failed") {
		t.Fatalf("AC2 state A precondition: 'refresh failed' missing before SetBuckets; got:\n%s", stateA)
	}

	// State B — SetBuckets clears the failed indicator.
	m.SetBuckets([]data.AllowanceBucket{
		projBucket("dns-zones", "networking/dnszones", 130, 200),
	})
	stateB := stripANSI(m.View())
	if strings.Contains(stateB, "refresh failed") {
		t.Errorf("AC2 state B: 'refresh failed' still present after SetBuckets(); want absent:\n%s", stateB)
	}

	// The two states must differ at the title-bar region.
	if stateA == stateB {
		t.Errorf("AC2 [Input-changed]: View() output identical between fail and success states; want distinct content")
	}
}

// AC3 [Anti-regression — fetchedAt unchanged on error]
// SetRefreshFailed(true) must not modify m.fetchedAt.
func TestFB060_AC3_AntiRegression_FetchedAtUnchangedOnFailure(t *testing.T) {
	t.Parallel()
	m := newBucketedWideDashboard()
	before := m.fetchedAt

	m.SetRefreshFailed(true)

	if !m.fetchedAt.Equal(before) {
		t.Errorf("AC3: fetchedAt changed after SetRefreshFailed(true); got %v, want %v", m.fetchedAt, before)
	}
}

// AC4 [Anti-regression — refreshing preempts failure]
// With refreshing=true AND refreshFailed=true simultaneously, View() must contain
// "refreshing" and must NOT contain "refresh failed".
func TestFB060_AC4_AntiRegression_RefreshingPreemptsFailedSignal(t *testing.T) {
	t.Parallel()
	m := newBucketedWideDashboard() // width=100, height=20
	m.SetRefreshing(true)
	m.SetRefreshFailed(true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "refreshing") {
		t.Errorf("AC4: 'refreshing' absent when both refreshing=true and refreshFailed=true; refreshing must preempt:\n%s", got)
	}
	if strings.Contains(got, "refresh failed") {
		t.Errorf("AC4: 'refresh failed' present alongside 'refreshing'; must be preempted by refreshing state:\n%s", got)
	}
}

// ==================== End FB-060 ====================
