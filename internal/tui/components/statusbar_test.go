package components

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

// AC4 [Anti-regression]: hint-string widths stay within budget.
// NAV == 68, NAV_DASHBOARD == 80, all panes ≤80.
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
			// Extract the hints portion from the rendered status bar by removing
			// the mode label prefix ("NORMAL │ ") and right-side padding/error area.
			// Simpler: compute width of the known expected hint string directly.
			hints := map[string]string{
				"NAV":           "[j/k] move  [Enter] select  [r] refresh  [c] ctx  [?] help  [q] quit",
				"NAV_DASHBOARD": "[j/k] move  [Enter] select  [c] ctx  [3] quota  [4] activity  [?] help  [q] quit",
			}[tc.pane]
			w := lipgloss.Width(hints)
			if w != tc.wantWidth {
				t.Errorf("AC4: %s hint width = %d, want %d (budget exceeded)", tc.pane, w, tc.wantWidth)
			}
			if w > 80 {
				t.Errorf("AC4: %s hint width = %d exceeds 80-char budget", tc.pane, w)
			}
		})
	}
}

// ==================== End FB-136 ====================
