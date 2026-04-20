package components

import (
	"strings"
	"testing"
)

// ==================== FB-114: Help overlay [d] describe deduplication ====================

// AC1 [Observable] — rendered help overlay contains "[d]" exactly once (VIEW column only).
func TestFB114_AC1_Observable_DOnlyInView(t *testing.T) {
	t.Parallel()
	m := NewHelpOverlayModel()
	m.Width = 120
	m.Height = 40

	got := stripANSI(m.View())
	count := strings.Count(got, "[d]")
	if count != 1 {
		t.Errorf("AC1 [Observable]: '[d]' appears %d times in View(), want exactly 1:\n%s", count, got)
	}
}

// AC2 [Input-changed] — count of "[d]" is 1 after fix (was 2 before fix).
// This test pins the post-fix state: exactly 1 occurrence, not 2.
func TestFB114_AC2_InputChanged_DeduplicatedCount(t *testing.T) {
	t.Parallel()
	m := NewHelpOverlayModel()
	m.Width = 120
	m.Height = 40

	got := stripANSI(m.View())
	count := strings.Count(got, "[d]")
	if count == 2 {
		t.Errorf("AC2 [Input-changed]: '[d]' appears 2 times (pre-fix state) — deduplication not applied:\n%s", got)
	}
	if count != 1 {
		t.Errorf("AC2 [Input-changed]: '[d]' count = %d, want 1 (post-fix state):\n%s", count, got)
	}
}

// AC3 [Anti-regression] — "[d]" is present in VIEW section, not absent entirely.
func TestFB114_AC3_AntiRegression_DescribeStillInView(t *testing.T) {
	t.Parallel()
	m := NewHelpOverlayModel()
	m.Width = 120
	m.Height = 40

	got := stripANSI(m.View())
	if !strings.Contains(got, "[d]") {
		t.Errorf("AC3 [Anti-regression]: '[d]' absent from View() entirely — should appear once in VIEW:\n%s", got)
	}
	if !strings.Contains(got, "describe") {
		t.Errorf("AC3 [Anti-regression]: 'describe' absent from View():\n%s", got)
	}
}

// AC4 [Anti-regression] — conditional lines ([C] conditions, [E] events, [x]) still appear/suppress correctly.
func TestFB114_AC4_AntiRegression_ConditionalLinesUnaffected(t *testing.T) {
	t.Parallel()

	// Default: no conditional flags set → none of the conditional lines appear.
	base := NewHelpOverlayModel()
	base.Width = 120
	base.Height = 40
	baseView := stripANSI(base.View())
	if strings.Contains(baseView, "[C]  conditions") {
		t.Errorf("AC4 [Anti-regression]: '[C]  conditions' present when ShowConditionsHint=false:\n%s", baseView)
	}
	if strings.Contains(baseView, "[E]  events") {
		t.Errorf("AC4 [Anti-regression]: '[E]  events' present when ShowEventsHint=false:\n%s", baseView)
	}
	if strings.Contains(baseView, "[x]") {
		t.Errorf("AC4 [Anti-regression]: '[x]' present when ShowDeleteHint=false:\n%s", baseView)
	}

	// With all flags: all conditional lines appear.
	full := NewHelpOverlayModel()
	full.Width = 120
	full.Height = 40
	full.ShowConditionsHint = true
	full.ShowEventsHint = true
	full.ShowDeleteHint = true
	fullView := stripANSI(full.View())
	for _, want := range []string{"[C]  conditions", "[E]  events", "[x]"} {
		if !strings.Contains(fullView, want) {
			t.Errorf("AC4 [Anti-regression]: %q absent from View() when flag set:\n%s", want, fullView)
		}
	}
}

// ==================== End FB-114 (component layer) ====================

// ==================== FB-119: HelpOverlay static content unchanged ====================

// TestFB119_AC5_AntiRegression_HelpOverlay_StaticContent verifies that the
// HelpOverlay VIEW section is unaffected by the describeAvailable gate.
// The overlay is a static reference document (FB-026 D3) — its [d] describe entry
// and [C] conditions entry (when ShowConditionsHint=true) are always present.
func TestFB119_AC5_AntiRegression_HelpOverlay_StaticContent(t *testing.T) {
	t.Parallel()
	m := NewHelpOverlayModel()
	m.Width = 120
	m.Height = 40
	m.ShowConditionsHint = true

	got := stripANSI(m.View())
	if !strings.Contains(got, "[d]") {
		t.Errorf("AC5 [Anti-regression FB-119]: '[d]' absent from HelpOverlay VIEW section; static content must not be affected:\n%s", got)
	}
	if !strings.Contains(got, "[C]  conditions") {
		t.Errorf("AC5 [Anti-regression FB-119]: '[C]  conditions' absent when ShowConditionsHint=true; static content must not be affected:\n%s", got)
	}
}

// ==================== End FB-119 (HelpOverlay layer) ====================
