package components

import (
	"github.com/charmbracelet/lipgloss"
	"go.datum.net/datumctl/internal/tui/data"
	"go.datum.net/datumctl/internal/tui/styles"
)

type StatusMode int

const (
	ModeNormal StatusMode = iota
	ModeFilter
	ModeDetail
	ModeOverlay
)

type StatusBarModel struct {
	Width       int
	Err         error
	ErrSeverity data.ErrorSeverity // drives glyph + color; defaults to ErrorSeverityWarning
	Hint        string        // transient amber hint; auto-clears after 3s or next keypress
	hintToken   int           // bumped on each postHint and early-clear; matched in HintClearMsg
	Mode        StatusMode
	Pane        string
}

// PostHint sets a new transient hint and bumps the hint token. Returns the new
// token so the caller can schedule a HintClearCmd with the same token.
func (m *StatusBarModel) PostHint(text string) int {
	m.hintToken++
	m.Hint = text
	return m.hintToken
}

// BumpHintToken invalidates any in-flight HintClearMsg without posting a new hint.
// Used for early-clear on keypress.
func (m *StatusBarModel) BumpHintToken() {
	m.hintToken++
}

// ClearHintIfToken clears the hint only when token matches the current hint token.
// Mismatched tokens (stale ticks) are silently ignored.
func (m *StatusBarModel) ClearHintIfToken(token int) {
	if token == m.hintToken {
		m.Hint = ""
	}
}

// HintToken returns the current hint token (read-only; for tests).
func (m StatusBarModel) HintToken() int { return m.hintToken }

func (m StatusBarModel) View() string {
	var modeLabel, hints string
	switch m.Mode {
	case ModeFilter:
		modeLabel = "FILTER"
		hints = "[Enter] apply  [Esc] cancel"
	case ModeDetail:
		modeLabel = "DETAIL"
		switch m.Pane {
		case "HISTORY":
			hints = "[↑/↓] scroll  [Enter] diff  [c] human only  [H] describe  [Esc] back  [r] refresh"
		case "DIFF":
			hints = "[↑/↓] scroll  [[]prev  []]next  [H] describe  [Esc] back  [r] refresh"
		default:
			hints = "[j/k] scroll  [Esc] back"
		}
	case ModeOverlay:
		modeLabel = "OVERLAY"
		hints = "[j/k] move  [Enter] select  [Esc] close"
	default:
		modeLabel = "NORMAL"
		switch m.Pane {
		case "NAV":
			hints = "[j/k] move  [Enter] select  [r] refresh  [c] ctx  [?] help  [q] quit"
		case "NAV_DASHBOARD":
			hints = "[j/k] move  [Enter] select  [c] ctx  [3] quota  [4] activity  [?] help  [q] quit"
		case "TABLE":
			hints = "[j/k] move  [Enter] open  [/] filter  [d] describe  [r] refresh  [c] ctx"
		case "QUOTA":
			hints = "[↑↓] move  [t] table  [s] group  [r] refresh  [3] back  [?] help  [q] quit"
		default:
			hints = "[j/k] move  [Enter] select  [/] filter  [d] describe  [c] ctx  [r] refresh  [?] help  [q] quit"
		}
	}

	label := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary).Render(modeLabel)
	left := label + " │ " + hints

	var right string
	switch {
	case m.Err != nil:
		g := errorGlyph(m.ErrSeverity)
		color := errorColor(m.ErrSeverity)
		right = lipgloss.NewStyle().Foreground(color).Render(g + " " + SanitizeErrMsg(m.Err))
	case m.Hint != "":
		right = lipgloss.NewStyle().Foreground(styles.Warning).Render("⚡ " + m.Hint)
	}

	content := lipgloss.NewStyle().Width(m.Width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, left,
			lipgloss.NewStyle().
				Width(m.Width-lipgloss.Width(left)).
				Align(lipgloss.Right).
				Render(right),
		),
	)
	return styles.FooterStyle.Width(m.Width).Render(content)
}
