package components

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

var ansiEscapeStatusBar = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSIStatusBar(s string) string {
	return ansiEscapeStatusBar.ReplaceAllString(s, "")
}

func newStatusBarModel(pane string) StatusBarModel {
	return StatusBarModel{Width: 120, Pane: pane, Mode: ModeNormal}
}

// ==================== FB-136: NavPane [r] refresh hint ====================

// AC1 [Observable]: NAV pane status bar contains "[r] refresh".
func TestFB136_AC1_Observable_NAV_ContainsRefreshHint(t *testing.T) {
	t.Parallel()
	m := newStatusBarModel("NAV")
	got := stripANSIStatusBar(m.View())
	if !strings.Contains(got, "[r] refresh") {
		t.Errorf("AC1: NAV status bar does not contain '[r] refresh':\n%s", got)
	}
}

// AC2 [Observable]: NAV_DASHBOARD pane status bar does NOT contain "[r] refresh".
func TestFB136_AC2_Observable_NAV_DASHBOARD_NoRefreshHint(t *testing.T) {
	t.Parallel()
	m := newStatusBarModel("NAV_DASHBOARD")
	got := stripANSIStatusBar(m.View())
	if strings.Contains(got, "[r] refresh") {
		t.Errorf("AC2: NAV_DASHBOARD status bar contains '[r] refresh' but Option B scopes it out:\n%s", got)
	}
}

// AC3 [Anti-regression]: TABLE and QUOTA hint lines still contain "[r] refresh".
func TestFB136_AC3_AntiRegression_TABLE_QUOTA_HaveRefreshHint(t *testing.T) {
	t.Parallel()
	for _, pane := range []string{"TABLE", "QUOTA"} {
		pane := pane
		t.Run(pane, func(t *testing.T) {
			t.Parallel()
			m := newStatusBarModel(pane)
			got := stripANSIStatusBar(m.View())
			if !strings.Contains(got, "[r] refresh") {
				t.Errorf("AC3: %s status bar does not contain '[r] refresh' (anti-regression):\n%s", pane, got)
			}
		})
	}
}

// extractHintSegment returns the hint portion of a stripped status-bar View() string.
// The status bar renders "MODELABEL │ <hints><padding><optional-right>"; this helper
// splits on the first " │ " separator and trims trailing whitespace so the caller gets
// only the hint text, suitable for width measurement.
func extractHintSegment(stripped string) string {
	const sep = " │ "
	idx := strings.Index(stripped, sep)
	if idx < 0 {
		return strings.TrimRight(stripped, " \t\r\n")
	}
	return strings.TrimRight(stripped[idx+len(sep):], " \t\r\n")
}

// AC4 [Anti-regression]: rendered hint-string widths stay within 80-char budget.
// NAV == 68, NAV_DASHBOARD == 80.
// Measures the live View() output — not a hardcoded literal — so future hint-string
// changes that overflow the budget are caught automatically.
func TestFB136_AC4_AntiRegression_HintStringWidths(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pane      string
		wantWidth int
	}{
		{"NAV", 68},
		{"NAV_DASHBOARD", 80},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.pane, func(t *testing.T) {
			t.Parallel()
			m := newStatusBarModel(tc.pane)
			rendered := stripANSIStatusBar(m.View())
			hint := extractHintSegment(rendered)
			w := lipgloss.Width(hint)
			if w != tc.wantWidth {
				t.Errorf("AC4: %s rendered hint width = %d, want %d\nhint segment: %q", tc.pane, w, tc.wantWidth, hint)
			}
			if w > 80 {
				t.Errorf("AC4: %s rendered hint width = %d exceeds 80-char budget\nhint segment: %q", tc.pane, w, hint)
			}
		})
	}
}

// ==================== End FB-136 ====================
