package components

import (
	"strings"
	"testing"
	"time"

	"go.datum.net/datumctl/internal/console/data"
)

func makeHistoryRow(rev int, verb, user string) data.HistoryRow {
	return data.HistoryRow{
		Rev:       rev,
		Verb:      verb,
		User:      user,
		Timestamp: time.Date(2026, 4, 18, 9, 12, 3, 0, time.UTC),
		Status:    200,
		Parseable: true,
	}
}

// --- DiffViewModel: normal diff view ---

func TestDiffViewModel_View_NormalDiff_BannerContainsRevLabels(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(3, "update", "alice@example.com")
	prev := makeHistoryRow(2, "update", "alice@example.com")
	m.SetRevision(rev, &prev, "+spec.nodeName: node-2\n-spec.nodeName: node-1\n", false, false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Rev 3") {
		t.Errorf("normal diff banner: want 'Rev 3', got %q", got)
	}
	if !strings.Contains(got, "Rev 2") {
		t.Errorf("normal diff banner: want 'Rev 2' (predecessor), got %q", got)
	}
}

func TestDiffViewModel_View_NormalDiff_ContentVisible(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(3, "update", "bob@example.com")
	prev := makeHistoryRow(2, "update", "bob@example.com")
	diffBody := "+new line\n-old line\n"
	m.SetRevision(rev, &prev, diffBody, false, false)

	got := m.View()
	if !strings.Contains(got, "new line") || !strings.Contains(got, "old line") {
		t.Errorf("DiffViewModel: diff body not visible in View(), got %q", got)
	}
}

// --- DiffViewModel: creation view (rev 1) ---

func TestDiffViewModel_View_CreationView_BannerSaysCreation(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(1, "create", "alice@example.com")
	m.SetRevision(rev, nil, `{"spec":{"nodeName":"node-1"}}`, true, false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "Rev 1") {
		t.Errorf("creation banner: want 'Rev 1', got %q", got)
	}
	if !strings.Contains(got, "creation") {
		t.Errorf("creation banner: want 'creation' label, got %q", got)
	}
}

func TestDiffViewModel_View_CreationView_ShowsCreatedResourceLabel(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(1, "create", "alice@example.com")
	m.SetRevision(rev, nil, `{}`, true, false)

	got := stripANSI(m.View())
	// Creation body has the "Created resource" label.
	if !strings.Contains(got, "Created resource") {
		t.Errorf("creation view: want 'Created resource' label, got %q", got)
	}
}

// Anti-behavior: creation view must NOT show diff +/- content (no tinting of the manifest).
func TestDiffViewModel_View_CreationView_NoPredecessorLabel(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(1, "create", "alice@example.com")
	m.SetRevision(rev, nil, `{"spec":"v1"}`, true, false)

	got := stripANSI(m.View())
	// Banner must NOT say "← Rev" (no predecessor for creation).
	if strings.Contains(got, "← Rev") {
		t.Errorf("creation view: banner should not show '← Rev', got %q", got)
	}
}

// --- DiffViewModel: predecessor missing view (§2f) ---

func TestDiffViewModel_View_PredecessorMissing_BannerSaysNotLoaded(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(5, "update", "alice@example.com")
	m.SetRevision(rev, nil, `{"spec":"v5"}`, false, true)

	got := stripANSI(m.View())
	if !strings.Contains(got, "not loaded") {
		t.Errorf("predMissing banner: want '(not loaded)', got %q", got)
	}
	if !strings.Contains(got, "Initial state") {
		t.Errorf("predMissing body: want 'Initial state' label, got %q", got)
	}
}

// --- DiffViewModel: empty diff (metadata-only) ---

func TestDiffViewModel_View_EmptyBody_ShowsMetadataOnlyNotice(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(2, "update", "alice@example.com")
	prev := makeHistoryRow(1, "create", "alice@example.com")
	m.SetRevision(rev, &prev, "", false, false)

	got := stripANSI(m.View())
	if !strings.Contains(got, "No visible changes") {
		t.Errorf("empty diff: want 'No visible changes' notice, got %q", got)
	}
}

// --- DiffViewModel: Reset ---

func TestDiffViewModel_Reset_ClearsState(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(3, "update", "bob@example.com")
	prev := makeHistoryRow(2, "update", "bob@example.com")
	m.SetRevision(rev, &prev, "+new\n", false, false)
	m.Reset()

	if m.rev.Rev != 0 {
		t.Errorf("Reset: rev.Rev = %d, want 0", m.rev.Rev)
	}
	if m.prev != nil {
		t.Errorf("Reset: prev = %v, want nil", m.prev)
	}
	if m.content != "" {
		t.Errorf("Reset: content = %q, want empty", m.content)
	}
	if m.isCreation || m.predecessorMissing {
		t.Error("Reset: flags not cleared")
	}
}

// --- DiffViewModel: HTTP status tinting ---

func TestDiffViewModel_View_Status200_Visible(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(3, "update", "alice@example.com")
	prev := makeHistoryRow(2, "update", "alice@example.com")
	m.SetRevision(rev, &prev, "+x", false, false)

	got := m.View()
	if !strings.Contains(got, "200") {
		t.Errorf("status 200: want '[200]' in View(), got %q", got)
	}
}

// --- DiffViewModel: SetRevision updates content ---

// TestDiffViewModel_SetRevision_UpdatesContent verifies that calling SetRevision
// twice with different content results in different View() output (observable outcome).
func TestDiffViewModel_SetRevision_UpdatesContent(t *testing.T) {
	t.Parallel()
	m := NewDiffViewModel(80, 20)
	rev := makeHistoryRow(3, "update", "alice@example.com")
	prev := makeHistoryRow(2, "update", "alice@example.com")

	m.SetRevision(rev, &prev, "+first change\n", false, false)
	view1 := m.View()

	rev2 := makeHistoryRow(4, "update", "alice@example.com")
	m.SetRevision(rev2, &rev, "+second change\n", false, false)
	view2 := m.View()

	// Different revisions and content should produce different views.
	if view1 == view2 {
		t.Error("SetRevision: View() identical for different revisions, want distinct output")
	}
	if !strings.Contains(view2, "second change") {
		t.Errorf("SetRevision: updated content not reflected in View(), got %q", view2)
	}
}
